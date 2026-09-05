package orderfill

import (
	"fmt"
	"slices"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// Detection is a recognised table: the sheet, its header row and the column
// index of every field the rules need.
type Detection struct {
	Sheet     spreadsheet.Sheet
	SheetName string
	HeaderRow int
	Columns   map[string]int
}

// Column keys shared by the source and blank layouts.
const (
	ColumnArticle     = "article"
	ColumnName        = "name"
	ColumnRecommended = "recommended"
	ColumnStock       = "stock"
	ColumnInTransit   = "inTransit"
	ColumnOrderedFact = "orderedFact"
	ColumnComment     = "comment"
	ColumnUnit        = "unit"
	ColumnQuantity    = "quantity"
	ColumnBoxSize     = "boxSize"
)

type headerMatcher func(header string) bool

// scanRowLimit mirrors the browser engine: headers never sit deeper than this.
const scanRowLimit = 120

func sourceMatchers() ([]string, map[string]headerMatcher) {
	order := []string{ColumnArticle, ColumnName, ColumnRecommended, ColumnStock, ColumnInTransit, ColumnOrderedFact, ColumnComment}
	return order, map[string]headerMatcher{
		ColumnArticle: func(h string) bool {
			return contains(h, "арт") || contains(h, "артикул") || contains(h, "код")
		},
		ColumnName:        isProductNameHeader,
		ColumnRecommended: func(h string) bool { return contains(h, "рекоменд") && contains(h, "заказ") },
		ColumnStock:       func(h string) bool { return contains(h, "остаток") },
		ColumnInTransit:   func(h string) bool { return contains(h, "в пути") },
		ColumnOrderedFact: func(h string) bool { return contains(h, "заказано") && contains(h, "факт") },
		ColumnComment:     func(h string) bool { return contains(h, "комментар") },
	}
}

func blankMatchers(rule brand.RuleConfig) ([]string, map[string]headerMatcher) {
	matchers := map[string]headerMatcher{
		ColumnArticle: func(h string) bool {
			return contains(h, "арт") || contains(h, "артикул") || contains(h, "код")
		},
		ColumnName:     func(h string) bool { return isProductNameHeader(h) || contains(h, "продукт") },
		ColumnQuantity: quantityMatcher(rule.BlankQuantityHeader),
	}
	order := []string{ColumnArticle, ColumnName}

	if rule.RequireUnit == nil || *rule.RequireUnit {
		matchers[ColumnUnit] = func(h string) bool {
			return contains(h, "объем") || contains(h, "обьем") || contains(h, "форма выпуска") || (contains(h, "мл") && contains(h, "гр"))
		}
		order = append(order, ColumnUnit)
	}
	order = append(order, ColumnQuantity)
	if rule.Adjustment == brand.AdjustmentBox {
		matchers[ColumnBoxSize] = boxMatcher(rule.BlankBoxHeader)
		order = append(order, ColumnBoxSize)
	}
	return order, matchers
}

func quantityMatcher(header string) headerMatcher {
	isQuantity := func(h string) bool {
		return contains(h, "кол во") || contains(h, "количество") || contains(h, "кол-во") || contains(h, "к во") || contains(h, "qty")
	}
	isPackage := func(h string) bool {
		return contains(h, "короб") || contains(h, "упак") || slices.Contains(strings.Fields(h), "уп")
	}
	switch header {
	case "order":
		return func(h string) bool { return h == "заказ" || contains(h, "коробка заказ") }
	case "exactQuantity":
		return func(h string) bool { return h == "количество" }
	case "anyOrder":
		return func(h string) bool {
			return h == "заказ" || contains(h, "коробка заказ") || (isQuantity(h) && !isPackage(h))
		}
	default:
		return isQuantity
	}
}

func boxMatcher(header string) headerMatcher {
	if header == "packageQuantity" {
		return func(h string) bool {
			return contains(h, "кол во в уп") || h == "кол во" || h == "количество"
		}
	}
	return func(h string) bool {
		return contains(h, "короб") || (contains(h, "шт") && contains(h, "упак"))
	}
}

func isProductNameHeader(header string) bool {
	return contains(header, "товар") || contains(header, "номенклатура") || contains(header, "наименование") || contains(header, "название")
}

// DetectSourceColumns recognises the 1C order calculation table.
func DetectSourceColumns(workbook spreadsheet.Workbook) (Detection, error) {
	order, matchers := sourceMatchers()
	return detectColumns(workbook, order, matchers, "этот файл не похож на таблицу продаж из 1С. Загрузите выгрузку с отбором номенклатуры, а не бланк поставщика")
}

// DetectBlankColumns recognises the supplier blank for a brand.
func DetectBlankColumns(workbook spreadsheet.Workbook, rule brand.RuleConfig) (Detection, error) {
	order, matchers := blankMatchers(rule)
	return detectColumns(workbook, order, matchers, "этот файл не похож на бланк поставщика. Проверьте, что загрузили бланк заказа, а не таблицу из 1С")
}

// detectColumns picks the header row where every required field is present and
// the matched columns sit closest together, which disambiguates workbooks that
// repeat similar captions in several blocks.
func detectColumns(workbook spreadsheet.Workbook, required []string, matchers map[string]headerMatcher, missing string) (Detection, error) {
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, scanRowLimit); row++ {
			candidates := make(map[string][]int, len(required))
			for column := 1; column <= bounds.MaxColumn; column++ {
				header := normalize.NormalizeHeader(sheet.Value(row, column))
				if header == "" {
					continue
				}
				for _, key := range required {
					if matchers[key](header) {
						candidates[key] = append(candidates[key], column)
					}
				}
			}
			found := 0
			for _, key := range required {
				if len(candidates[key]) > 0 {
					found++
				}
			}
			if found != len(required) {
				continue
			}
			if columns, ok := tightestCombination(required, candidates); ok {
				return Detection{Sheet: sheet, SheetName: sheet.Name(), HeaderRow: row, Columns: columns}, nil
			}
		}
	}
	return Detection{}, fmt.Errorf("%w: %s", ErrInvalidInput, missing)
}

// tightestCombination picks distinct columns for every field minimising the
// span between the leftmost and rightmost match.
func tightestCombination(required []string, candidates map[string][]int) (map[string]int, bool) {
	best := map[string]int{}
	bestSpan := -1
	current := make([]int, len(required))

	var walk func(index int)
	walk = func(index int) {
		if index == len(required) {
			used := make(map[int]bool, len(current))
			low, high := current[0], current[0]
			for _, column := range current {
				if used[column] {
					return
				}
				used[column] = true
				low = min(low, column)
				high = max(high, column)
			}
			span := high - low + 1
			if bestSpan >= 0 && span >= bestSpan {
				return
			}
			bestSpan = span
			for position, key := range required {
				best[key] = current[position]
			}
			return
		}
		for _, column := range candidates[required[index]] {
			current[index] = column
			walk(index + 1)
		}
	}
	walk(0)
	return best, bestSpan >= 0
}

func contains(haystack string, needle string) bool {
	return strings.Contains(haystack, needle)
}
