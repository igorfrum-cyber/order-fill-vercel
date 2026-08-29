package orderfill

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"order-fill/services/document-service/internal/domain/brand"
	"order-fill/services/document-service/internal/domain/normalize"
	"order-fill/services/document-service/internal/domain/spreadsheet"
)

// ManualEdit is a reviewer correction for a single report row.
type ManualEdit struct {
	Key     string
	Value   string
	Comment string
}

// FinalizeCommand applies reviewer corrections to both workbooks.
type FinalizeCommand struct {
	Source spreadsheet.Workbook
	Blank  spreadsheet.Workbook
	Rows   []ReportRow
	Edits  []ManualEdit
	Brand  string
}

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
	rule := brand.Rule(command.Brand)
	if rule.BlankLayout != "" {
		return fmt.Errorf("%w: раскладка бланка %q для бренда %s пока не поддерживается сервисом", ErrInvalidInput, rule.BlankLayout, rule.Label)
	}
	blank, err := DetectBlankColumns(command.Blank, rule)
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

		if !sameQuantity(quantity, baseline) && comment == "" {
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
	return nil
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
