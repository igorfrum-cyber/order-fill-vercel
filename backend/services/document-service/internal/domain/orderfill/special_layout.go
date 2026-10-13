package orderfill

import (
	"fmt"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// SothysBlock identifies one side-by-side supplier size variant.
type SothysBlock struct{ Article, Volume, Unit, Quantity int }

// DetectOrderBlank recognises the brand's actual supplier layout.
func DetectOrderBlank(workbook spreadsheet.Workbook, rule brand.RuleConfig) (Detection, error) {
	switch rule.BlankLayout {
	case "splitVariants":
		return detectSothysBlank(workbook)
	case "novacutan":
		return detectNovacutanBlank(workbook)
	}
	if rule.BlankLayout != "" {
		return Detection{}, fmt.Errorf("%w: неизвестная раскладка бланка %q", ErrInvalidInput, rule.BlankLayout)
	}
	return DetectBlankColumns(workbook, rule)
}

func detectSothysBlank(workbook spreadsheet.Workbook) (Detection, error) {
	best := Detection{}
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 220); row++ {
			blocks := []SothysBlock{}
			for col := 1; col <= bounds.MaxColumn-4; col++ {
				article := normalize.NormalizeHeader(sheet.Value(row, col))
				volume := normalize.NormalizeHeader(sheet.Value(row, col+1))
				qty := normalize.NormalizeHeader(sheet.Value(row, col+4))
				if (strings.Contains(article, "арт") || strings.Contains(article, "артикул")) && (strings.Contains(volume, "объем") || strings.Contains(volume, "объм") || strings.Contains(volume, "обьм")) && qty == "заказ" {
					blocks = append(blocks, SothysBlock{Article: col, Volume: col + 1, Unit: col + 2, Quantity: col + 4})
				}
			}
			if len(blocks) > len(best.LayoutBlocks) {
				best = Detection{Sheet: sheet, SheetName: sheet.Name(), HeaderRow: row, Columns: map[string]int{ColumnName: 3, ColumnQuantity: blocks[0].Quantity}, LayoutBlocks: blocks}
			}
			if len(blocks) >= 2 {
				return best, nil
			}
		}
	}
	if len(best.LayoutBlocks) > 0 {
		return best, nil
	}
	return Detection{}, fmt.Errorf("%w: не удалось найти блоки SOTHYS: Артикул, Объем, Заказ", ErrInvalidInput)
}

func detectNovacutanBlank(workbook spreadsheet.Workbook) (Detection, error) {
	signal := false
	for _, sheet := range workbook.Sheets() {
		if strings.Contains(normalize.NormalizeHeader(sheet.Name()), "novacutan") {
			signal = true
		}
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 80); row++ {
			for col := 1; col <= bounds.MaxColumn; col++ {
				if strings.Contains(normalize.NormalizeHeader(sheet.Value(row, col)), "novacutan") {
					signal = true
				}
			}
		}
	}
	if signal {
		for _, sheet := range workbook.Sheets() {
			bounds := sheet.Bounds()
			for row := 1; row <= min(bounds.MaxRow, 120); row++ {
				name, price, minimum, quantity := 0, 0, 0, 0
				for col := 1; col <= bounds.MaxColumn; col++ {
					header := normalize.NormalizeHeader(sheet.Value(row, col))
					switch {
					case strings.Contains(header, "описание") || strings.Contains(header, "товар") || strings.Contains(header, "название"):
						name = col
					case strings.Contains(header, "цена"):
						price = col
					case strings.Contains(header, "заказ") && strings.Contains(header, "от"):
						minimum = col
					case header == "заказ" || strings.Contains(header, "кол во") || strings.Contains(header, "количество") || strings.Contains(header, "qty"):
						quantity = col
					}
				}
				if name > 0 && (quantity > 0 || minimum > 0) {
					created := quantity == 0
					if created {
						quantity = max(name, minimum) + 1
					}
					return Detection{Sheet: sheet, SheetName: sheet.Name(), HeaderRow: row, Columns: map[string]int{ColumnName: name, ColumnQuantity: quantity, "price": price, "minimum": minimum}, QuantityColumnCreated: created}, nil
				}
			}
		}
	}
	return Detection{}, fmt.Errorf("%w: не удалось распознать бланк NOVACUTAN: не нашел описание товара и колонку «кол-во» или «Заказ от»", ErrInvalidInput)
}

func orderBlankPositions(blank Detection, id string, rule brand.RuleConfig) []blankPosition {
	switch rule.BlankLayout {
	case "splitVariants":
		return sothysBlankPositions(blank, id, rule)
	case "novacutan":
		return novacutanBlankPositions(blank, id)
	}
	return blankPositions(blank, id, rule)
}

func sothysBlankPositions(blank Detection, id string, rule brand.RuleConfig) []blankPosition {
	positions := []blankPosition{}
	lastEnglish, lastRussian := "", ""
	for row := blank.HeaderRow + 1; row <= blank.Sheet.Bounds().MaxRow; row++ {
		hasHeader, hasArticle := false, false
		for _, block := range blank.LayoutBlocks {
			if strings.Contains(normalize.NormalizeHeader(blank.Sheet.Value(row, block.Article)), "арт") {
				hasHeader = true
			}
			if normalize.AsText(blank.Sheet.Value(row, block.Article)) != "" {
				hasArticle = true
			}
		}
		if hasHeader {
			continue
		}
		english, russian := normalize.AsText(blank.Sheet.Value(row, 2)), normalize.AsText(blank.Sheet.Value(row, 3))
		if hasArticle && (english != "" || russian != "") {
			if english != "" {
				lastEnglish = english
			}
			if russian != "" {
				lastRussian = russian
			}
		}
		if english == "" {
			english = lastEnglish
		}
		if russian == "" {
			russian = lastRussian
		}
		nameParts := []string{}
		for _, name := range []string{english, russian} {
			if name != "" {
				nameParts = append(nameParts, name)
			}
		}
		for _, block := range blank.LayoutBlocks {
			raw := normalize.AsText(blank.Sheet.Value(row, block.Article))
			article := normalize.NormalizeArticle(raw, brand.ArticleNormalizeOptions(rule))
			if article == "" {
				continue
			}
			unitParts := []string{}
			for _, col := range []int{block.Volume, block.Unit} {
				if value := normalize.AsText(blank.Sheet.Value(row, col)); value != "" {
					unitParts = append(unitParts, value)
				}
			}
			priceCol := block.Quantity - 1
			price := budgetPriceAt(blank.Sheet, row, priceCol)
			prices := []BudgetPrice{{Column: priceCol, Label: blank.Sheet.Value(blank.HeaderRow, priceCol) + " (по блоку объема)", Price: price}}
			positions = append(positions, blankPosition{key: fmt.Sprintf("%s:%d:%d", id, row, block.Quantity), blankRow: row, blankQuantityColumn: block.Quantity, articleRaw: raw, article: article, name: strings.Join(nameParts, " / "), unit: strings.Join(unitParts, " "), budgetPrice: price, budgetPrices: prices})
		}
	}
	return positions
}

func novacutanBlankPositions(blank Detection, id string) []blankPosition {
	if blank.QuantityColumnCreated {
		copyQuantityStyle(blank, blank.HeaderRow)
		blank.Sheet.SetText(blank.HeaderRow, blank.Columns[ColumnQuantity], "Заказ")
	}
	positions := []blankPosition{}
	for row := blank.HeaderRow + 1; row <= blank.Sheet.Bounds().MaxRow; row++ {
		name := normalize.AsText(blank.Sheet.Value(row, blank.Columns[ColumnName]))
		header := normalize.NormalizeHeader(name)
		if name == "" || header == "описание" || header == "novacutan" {
			continue
		}
		minimum, hasMinimum := normalize.ParseNumber(blank.Sheet.Value(row, blank.Columns["minimum"]))
		if !hasMinimum {
			minimum = novacutanLayoutMinimum(name)
		}
		size := 1.0
		key := locationNovacutanKey(name)
		if key == "mask-eye" || key == "mask-face" {
			size = 5
		}
		copyQuantityStyle(blank, row)
		price := budgetPriceAt(blank.Sheet, row, blank.Columns["price"])
		positions = append(positions, blankPosition{key: fmt.Sprintf("%s:%d", id, row), blankRow: row, blankQuantityColumn: blank.Columns[ColumnQuantity], name: name, unit: "шт", boxSize: strconv.FormatFloat(minimum, 'f', -1, 64), supplierUnitSize: size, budgetPrice: price, budgetPrices: BudgetPrices(blank.Sheet, blank.HeaderRow, row, blank.Columns[ColumnQuantity])})
	}
	return positions
}

func novacutanLayoutMinimum(name string) float64 {
	text := normalize.NormalizeHeader(name)
	if (strings.Contains(text, "mask") || strings.Contains(text, "маск")) && (strings.Contains(text, "filler") || strings.Contains(text, "филлер")) {
		return 10
	}
	if strings.Contains(text, "fbio") || strings.Contains(text, "bright") || strings.Contains(text, "брайт") || strings.Contains(text, "gentle") || strings.Contains(text, "джентл") || strings.Contains(text, "джентел") {
		return 50
	}
	return 100
}
func copyQuantityStyle(blank Detection, row int) {
	if !blank.QuantityColumnCreated {
		return
	}
	source := blank.Columns["minimum"]
	if source == 0 {
		source = blank.Columns[ColumnName]
	}
	if copier, ok := blank.Sheet.(spreadsheet.CellStyleCopier); ok {
		copier.CopyCellStyle(row, source, blank.Columns[ColumnQuantity])
	}
}

// OrderBlankPosition exposes parsed supplier geometry to the North document flow.
type OrderBlankPosition struct {
	Row, QuantityColumn, PriceColumn          int
	Key, ArticleRaw, Name, Unit               string
	Price, SupplierUnitSize, Minimum, BoxSize float64
	HasBoxSize                                bool
}

func SpecializedBlankPositions(blank Detection, rule brand.RuleConfig) []OrderBlankPosition {
	positions := orderBlankPositions(blank, "north", rule)
	out := make([]OrderBlankPosition, 0, len(positions))
	boxColumn := BlankBoxColumn(blank, rule)
	for _, position := range positions {
		key := position.article
		if rule.Key == "novacutan" {
			key = NovacutanProductKey(position.name)
		}
		priceCol := 0
		if len(position.budgetPrices) > 0 {
			priceCol = position.budgetPrices[0].Column
		}
		box, hasBox := normalize.ParseNumber(blank.Sheet.Value(position.blankRow, boxColumn))
		out = append(out, OrderBlankPosition{Row: position.blankRow, QuantityColumn: position.blankQuantityColumn, PriceColumn: priceCol, Key: key, ArticleRaw: position.articleRaw, Name: position.name, Unit: position.unit, Price: position.budgetPrice, SupplierUnitSize: position.supplierUnitSize, Minimum: parsedLayoutMinimum(position.boxSize), BoxSize: box, HasBoxSize: hasBox})
	}
	return out
}

func NovacutanProductKey(name string) string { return "novacutan:" + locationNovacutanKey(name) }

func parsedLayoutMinimum(value string) float64 {
	number, _ := normalize.ParseNumber(value)
	return number
}

// BlankBoxColumn mirrors the prototype's first matching supplier pack header.
func BlankBoxColumn(blank Detection, rule brand.RuleConfig) int {
	matches := boxMatcher(rule.BlankBoxHeader)
	for col := 1; col <= blank.Sheet.Bounds().MaxColumn; col++ {
		if matches(normalize.NormalizeHeader(blank.Sheet.Value(blank.HeaderRow, col))) {
			return col
		}
	}
	return 0
}
