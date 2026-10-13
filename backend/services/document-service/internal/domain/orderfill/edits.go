package orderfill

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// ManualEdit is a reviewer correction for a single report row.
type ManualEdit struct {
	Key     string
	Value   string
	Comment string
}

// FinalizeCommand applies reviewer corrections to both workbooks.
type FinalizeCommand struct {
	Source             spreadsheet.Workbook
	Blank              spreadsheet.Workbook
	Rows               []ReportRow
	Edits              []ManualEdit
	Brand              string
	Rule               brand.RuleConfig
	IncludeBlankStocks bool
}

var strictStock = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)
var fileNamePattern = regexp.MustCompile(`[^\p{L}\p{N}_ .-]+`)

// ParseEditValue reads a quantity typed by the reviewer. An empty value clears
// the cell, which is why it returns a nil pointer instead of zero.
func ParseEditValue(value string) (*float64, error) {
	text := normalize.AsText(value)
	if text == "" {
		return nil, nil
	}
	number, err := strconv.ParseFloat(strings.ReplaceAll(text, ",", "."), 64)
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) || number < 0 || number != math.Trunc(number) {
		return nil, fmt.Errorf("%w: количество должно быть целым неотрицательным числом", ErrInvalidInput)
	}
	if number == 0 {
		return nil, nil
	}
	return &number, nil
}

// ApplyFinalEdits writes reviewer quantities into the blank and mirrors them
// into the source workbook as "Заказано по факту" with the justification.
func ApplyFinalEdits(command FinalizeCommand) error {
	rule := command.Rule
	if rule.Key == "" {
		rule = brand.Rule(command.Brand)
	}

	blank, err := DetectOrderBlank(command.Blank, rule)
	if err != nil {
		return err
	}
	source, err := DetectSourceColumns(command.Source)
	if err != nil {
		return err
	}

	editsByKey := make(map[string]ManualEdit, len(command.Edits))
	for _, edit := range command.Edits {
		editsByKey[edit.Key] = edit
	}

	for _, row := range command.Rows {
		edit, ok := editsByKey[row.Key]
		if !ok {
			continue
		}
		quantity, err := ParseEditValue(edit.Value)
		if err != nil {
			return err
		}
		comment := normalize.AsText(edit.Comment)
		baseline := baselineQuantity(row, rule)
		autoComment := normalize.AsText(row.AutoComment)

		stillAuto := autoComment != "" && strings.EqualFold(comment, autoComment)
		autoAllowed := stillAuto && sameQuantity(quantity, row.Inserted)
		if !sameQuantity(quantity, baseline) && (comment == "" || stillAuto && !autoAllowed) {
			return fmt.Errorf("%w: если значение в колонке «Вставлено» изменено, нужно заполнить комментарий (строка %d)", ErrInvalidInput, row.BlankRow)
		}

		quantityColumn := row.BlankQuantityColumn
		if quantityColumn == 0 {
			quantityColumn = blank.Columns[ColumnQuantity]
		}
		if row.BlankRow > blank.HeaderRow && quantityColumn > 0 {
			if quantity == nil {
				blank.Sheet.ClearValue(row.BlankRow, quantityColumn)
			} else {
				blank.Sheet.SetNumber(row.BlankRow, quantityColumn, *quantity)
			}
		}

		if row.SourceRow == nil || *row.SourceRow <= source.HeaderRow {
			continue
		}
		recordFact := !sameQuantity(quantity, baseline)
		if recordFact {
			value := 0.0
			if quantity != nil {
				value = *quantity
			}
			source.Sheet.SetNumber(*row.SourceRow, source.Columns[ColumnOrderedFact], value)
			source.Sheet.SetText(*row.SourceRow, source.Columns[ColumnComment], comment)
			continue
		}
		source.Sheet.ClearValue(*row.SourceRow, source.Columns[ColumnOrderedFact])
		source.Sheet.SetText(*row.SourceRow, source.Columns[ColumnComment], "")
	}
	if command.IncludeBlankStocks {
		addOrderBlankStocks(blank, command.Rows, rule, command.Brand)
	}
	return nil
}

func addOrderBlankStocks(blank Detection, rows []ReportRow, rule brand.RuleConfig, brandKey string) {
	existingColumn := 0
	for candidate := 1; candidate <= blank.Sheet.Bounds().MaxColumn; candidate++ {
		if normalize.NormalizeHeader(blank.Sheet.Value(blank.HeaderRow, candidate)) == "остаток" {
			existingColumn = candidate
			break
		}
	}
	angioColumn := 0
	if brandKey == "angiopharm" {
		header := normalize.NormalizeHeader(blank.Sheet.Value(blank.HeaderRow, 10))
		available := header == "" || header == "остаток"
		for _, row := range rows {
			if !available || row.BlankRow <= blank.HeaderRow {
				continue
			}
			cell := blank.Sheet.Value(row.BlankRow, 10)
			if cell != "" {
				_, available = normalize.ParseNumber(cell)
			}
		}
		if available {
			angioColumn = 10
		}
	}
	column := angioColumn
	if column == 0 {
		column = existingColumn
	}
	if column == 0 {
		column = blank.Sheet.Bounds().MaxColumn + 1
	}
	blanksByArticle := make(map[string]int, len(rows))
	for _, row := range rows {
		article := normalize.NormalizeArticle(row.BlankArticle, brand.ArticleNormalizeOptions(rule))
		if article != "" {
			blanksByArticle[article]++
		}
	}
	if copier, ok := blank.Sheet.(spreadsheet.CellStyleCopier); ok {
		copier.CopyCellStyle(blank.HeaderRow, blank.Columns[ColumnQuantity], column)
	}
	blank.Sheet.SetText(blank.HeaderRow, column, "Остаток")
	for _, row := range rows {
		if row.BlankRow <= blank.HeaderRow {
			continue
		}
		blank.Sheet.ClearValue(row.BlankRow, column)
		article := normalize.NormalizeArticle(row.BlankArticle, brand.ArticleNormalizeOptions(rule))
		sourceArticle := normalize.NormalizeArticle(row.SourceArticle, brand.ArticleNormalizeOptions(rule))
		sameItem := article != "" && article == sourceArticle && blanksByArticle[article] == 1
		if article == "" {
			sameItem = row.BlankName != "" && normalize.NormalizeName(row.BlankName) == normalize.NormalizeName(row.SourceName)
		}
		rawStock := strings.Replace(strings.Join(strings.Fields(row.Stock), ""), ",", ".", 1)
		stock, parseErr := strconv.ParseFloat(rawStock, 64)
		hasStock := strictStock.MatchString(rawStock) && parseErr == nil && !math.IsNaN(stock) && !math.IsInf(stock, 0)
		reliable := row.SourceRow != nil && !row.Duplicate && !strings.HasPrefix(row.Status, "warning") && sameItem && hasStock
		if reliable {
			blank.Sheet.SetNumber(row.BlankRow, column, stock)
		}
	}
}

// baselineQuantity is the quantity the engine would insert without any manual
// correction. Matching it means the reviewer confirmed the calculation.
func baselineQuantity(row ReportRow, rule brand.RuleConfig) *float64 {
	if row.Recommended == nil || row.Rounded == nil {
		return nil
	}
	if !rule.AllowSmallPositiveOrder && *row.Recommended < 1.5 {
		return nil
	}
	if *row.Rounded <= 0 {
		return nil
	}
	value := float64(*row.Rounded)
	return &value
}

func sameQuantity(left *float64, right *float64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

// BlankOutputFileName is the name of the filled supplier blank.
func BlankOutputFileName(originalName string, cityName string) string {
	stem := fileNameStem(originalName, "blank")
	city := strings.TrimSpace(fileNamePattern.ReplaceAllString(normalize.AsText(cityName), ""))
	if city != "" && !strings.Contains(normalize.NormalizeHeader(stem), normalize.NormalizeHeader(city)) {
		stem = stem + " " + city
	}
	return stem + " заполненный." + outputExtension(originalName)
}

// SourceOutputFileName is the name of the filled 1C order table.
func SourceOutputFileName(originalName string) string {
	return fileNameStem(originalName, "order") + " заполненная таблица." + outputExtension(originalName)
}

func fileNameStem(originalName string, fallback string) string {
	text := normalize.AsText(originalName)
	trimmed := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(text, ".xlsx"), ".xlsm"), ".xls")
	if lower := strings.ToLower(text); strings.HasSuffix(lower, ".xlsx") || strings.HasSuffix(lower, ".xlsm") || strings.HasSuffix(lower, ".xls") {
		trimmed = text[:strings.LastIndex(text, ".")]
	}
	stem := strings.TrimSpace(fileNamePattern.ReplaceAllString(trimmed, ""))
	if stem == "" {
		return fallback
	}
	return stem
}

func outputExtension(fileName string) string {
	if strings.HasSuffix(strings.ToLower(normalize.AsText(fileName)), ".xlsm") {
		return "xlsm"
	}
	return "xlsx"
}

// AddOrderBlankStocks adds reliable stock values to the generated export copy.
func AddOrderBlankStocks(workbook spreadsheet.Workbook, rows []ReportRow, rule brand.RuleConfig, key string) error {
	detection, err := DetectOrderBlank(workbook, rule)
	if err != nil {
		return err
	}
	addOrderBlankStocks(detection, rows, rule, key)
	return nil
}
