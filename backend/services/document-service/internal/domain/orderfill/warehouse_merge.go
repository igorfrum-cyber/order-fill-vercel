package orderfill

import (
	"fmt"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// MergeTyumenSources combines the office and delivery-warehouse 1C exports in
// the office workbook. The merged workbook then follows the ordinary fill
// pipeline, so recommendation math remains owned by calculation-service.
func MergeTyumenSources(office, warehouse spreadsheet.Workbook, orderMonth string, rule brand.RuleConfig) (spreadsheet.Workbook, error) {
	if office == nil || warehouse == nil {
		return nil, fmt.Errorf("%w: загрузите обе таблицы Тюмени: офис и склад доставки", ErrInvalidInput)
	}
	if _, err := ValidateSourcePeriods(office, orderMonth); err != nil {
		return nil, fmt.Errorf("таблица офиса: %w", err)
	}
	if _, err := ValidateSourcePeriods(warehouse, orderMonth); err != nil {
		return nil, fmt.Errorf("таблица склада доставки: %w", err)
	}
	if officeWeeks, warehouseWeeks := max(1, detectDeliveryWeeks(office)), max(1, detectDeliveryWeeks(warehouse)); officeWeeks != warehouseWeeks {
		return nil, fmt.Errorf("%w: срок поставки в таблицах офиса и склада различается", ErrInvalidInput)
	}

	officeDetection, err := DetectSourceColumns(office)
	if err != nil {
		return nil, fmt.Errorf("таблица офиса: %w", err)
	}
	warehouseDetection, err := DetectSourceColumns(warehouse)
	if err != nil {
		return nil, fmt.Errorf("таблица склада доставки: %w", err)
	}
	officeCalculation := detectCalculationColumns(officeDetection)
	if officeCalculation == nil {
		return nil, fmt.Errorf("%w: для двух складов нужна полная таблица офиса с помесячными продажами и ABC-анализом", ErrInvalidInput)
	}
	warehouseCalculation := detectCalculationColumns(warehouseDetection)
	if warehouseCalculation != nil && !sameSalesMonths(officeDetection, *officeCalculation, warehouseDetection, *warehouseCalculation) {
		return nil, fmt.Errorf("%w: месяцы продаж в таблицах офиса и склада не совпадают", ErrInvalidInput)
	}

	officeRows, err := locationRows(officeDetection, rule)
	if err != nil {
		return nil, fmt.Errorf("таблица офиса: %w", err)
	}
	warehouseRows, err := locationRows(warehouseDetection, rule)
	if err != nil {
		return nil, fmt.Errorf("таблица склада доставки: %w", err)
	}

	nextRow := officeDetection.Sheet.Bounds().MaxRow + 1
	matchedOfficeRows := make(map[int]bool)
	for _, key := range warehouseRows.order {
		warehouseRow := warehouseRows.byKey[key]
		officeRow, exists := officeRows.byKey[key]
		if !exists {
			officeRow, exists, err = locationNameMatch(officeRows, warehouseRow, matchedOfficeRows)
			if err != nil {
				return nil, err
			}
		}
		if exists {
			if !compatibleLocationNames(officeRow.name, warehouseRow.name) {
				return nil, fmt.Errorf("%w: у позиции %q разные названия в таблицах офиса и склада", ErrInvalidInput, warehouseRow.article)
			}
			matchedOfficeRows[officeRow.row] = true
			if officeRow.article == "" && warehouseRow.article != "" {
				officeDetection.Sheet.SetText(officeRow.row, officeDetection.Columns[ColumnArticle], warehouseDetection.Sheet.Value(warehouseRow.row, warehouseDetection.Columns[ColumnArticle]))
			}
			mergeLocationRow(officeDetection, *officeCalculation, officeRow.row, warehouseDetection, warehouseCalculation, warehouseRow.row)
			continue
		}
		copyLocationRow(officeDetection, *officeCalculation, nextRow, warehouseDetection, warehouseCalculation, warehouseRow.row)
		nextRow++
	}
	return office, nil
}

func locationNameMatch(office locationIndex, warehouse locationRow, matched map[int]bool) (locationRow, bool, error) {
	var candidates []locationRow
	for _, candidate := range office.byKey {
		if candidate.article != "" && warehouse.article != "" {
			continue
		}
		if compatibleLocationNames(candidate.name, warehouse.name) {
			candidates = append(candidates, candidate)
		}
	}
	if len(candidates) > 1 || len(candidates) == 1 && matched[candidates[0].row] {
		return locationRow{}, false, fmt.Errorf("%w: неоднозначное соответствие позиции %q между офисом и складом", ErrInvalidInput, warehouse.name)
	}
	if len(candidates) == 0 {
		return locationRow{}, false, nil
	}
	return candidates[0], true, nil
}

type locationRow struct {
	row     int
	article string
	name    string
}

type locationIndex struct {
	byKey map[string]locationRow
	order []string
}

func locationRows(detection Detection, rule brand.RuleConfig) (locationIndex, error) {
	rows := locationIndex{byKey: make(map[string]locationRow)}
	options := brand.ArticleNormalizeOptions(rule)
	for row := detection.HeaderRow + 1; row <= detection.Sheet.Bounds().MaxRow; row++ {
		if isSourceTotalRow(detection, row, detection.Sheet.Bounds().MaxColumn) {
			continue
		}
		article := normalize.NormalizeArticle(detection.Sheet.Value(row, detection.Columns[ColumnArticle]), options)
		name := normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnName]))
		key := "article:" + article
		if article == "" {
			key = "name:" + cleanLocationName(name)
		}
		if key == "name:" {
			continue
		}
		if _, duplicate := rows.byKey[key]; duplicate {
			return locationIndex{}, fmt.Errorf("%w: позиция %q встречается несколько раз; объедините дубли перед загрузкой", ErrInvalidInput, firstNonEmpty(article, name))
		}
		rows.byKey[key] = locationRow{row: row, article: article, name: name}
		rows.order = append(rows.order, key)
	}
	return rows, nil
}

func sameSalesMonths(left Detection, leftColumns calculationColumns, right Detection, rightColumns calculationColumns) bool {
	if len(leftColumns.salesColumns) != len(rightColumns.salesColumns) {
		return false
	}
	for index, column := range leftColumns.salesColumns {
		leftMonth := normalize.NormalizeHeader(left.Sheet.Value(left.HeaderRow-1, column))
		rightMonth := normalize.NormalizeHeader(right.Sheet.Value(right.HeaderRow-1, rightColumns.salesColumns[index]))
		if leftMonth != rightMonth {
			return false
		}
	}
	return true
}

func mergeLocationRow(target Detection, targetColumns calculationColumns, targetRow int, source Detection, sourceColumns *calculationColumns, sourceRow int) {
	sumLocationNumber(target.Sheet, targetRow, target.Columns[ColumnStock], source.Sheet, sourceRow, source.Columns[ColumnStock])
	sumLocationNumber(target.Sheet, targetRow, target.Columns[ColumnInTransit], source.Sheet, sourceRow, source.Columns[ColumnInTransit])
	mergeOrderedFact(target.Sheet, targetRow, target.Columns[ColumnOrderedFact], source.Sheet, sourceRow, source.Columns[ColumnOrderedFact])
	mergeLocationComment(target.Sheet, targetRow, target.Columns[ColumnComment], source.Sheet, sourceRow, source.Columns[ColumnComment])
	if sourceColumns == nil {
		return
	}
	for index, column := range targetColumns.salesColumns {
		sumLocationNumber(target.Sheet, targetRow, column, source.Sheet, sourceRow, sourceColumns.salesColumns[index])
	}
	for _, columns := range [][2]int{
		{targetColumns.totalQuantity, sourceColumns.totalQuantity},
		{targetColumns.revenue, sourceColumns.revenue},
		{targetColumns.previousQuantity, sourceColumns.previousQuantity},
	} {
		sumLocationNumber(target.Sheet, targetRow, columns[0], source.Sheet, sourceRow, columns[1])
	}
}

func copyLocationRow(target Detection, targetColumns calculationColumns, targetRow int, source Detection, sourceColumns *calculationColumns, sourceRow int) {
	target.Sheet.SetText(targetRow, target.Columns[ColumnArticle], source.Sheet.Value(sourceRow, source.Columns[ColumnArticle]))
	target.Sheet.SetText(targetRow, target.Columns[ColumnName], source.Sheet.Value(sourceRow, source.Columns[ColumnName]))
	copyLocationNumber(target.Sheet, targetRow, target.Columns[ColumnStock], source.Sheet, sourceRow, source.Columns[ColumnStock])
	copyLocationNumber(target.Sheet, targetRow, target.Columns[ColumnInTransit], source.Sheet, sourceRow, source.Columns[ColumnInTransit])
	copyLocationCell(target.Sheet, targetRow, target.Columns[ColumnOrderedFact], source.Sheet, sourceRow, source.Columns[ColumnOrderedFact])
	copyLocationCell(target.Sheet, targetRow, target.Columns[ColumnComment], source.Sheet, sourceRow, source.Columns[ColumnComment])
	if sourceColumns == nil {
		return
	}
	for index, column := range targetColumns.salesColumns {
		copyLocationNumber(target.Sheet, targetRow, column, source.Sheet, sourceRow, sourceColumns.salesColumns[index])
	}
	for _, columns := range [][2]int{
		{targetColumns.totalQuantity, sourceColumns.totalQuantity},
		{targetColumns.revenue, sourceColumns.revenue},
		{targetColumns.previousQuantity, sourceColumns.previousQuantity},
	} {
		copyLocationNumber(target.Sheet, targetRow, columns[0], source.Sheet, sourceRow, columns[1])
	}
}

func mergeOrderedFact(target spreadsheet.Sheet, targetRow, targetColumn int, source spreadsheet.Sheet, sourceRow, sourceColumn int) {
	leftRaw := normalize.AsText(target.Value(targetRow, targetColumn))
	rightRaw := normalize.AsText(source.Value(sourceRow, sourceColumn))
	if leftRaw == "" && rightRaw == "" {
		target.ClearValue(targetRow, targetColumn)
		return
	}
	left, leftOK := normalize.ParseNumber(leftRaw)
	right, rightOK := normalize.ParseNumber(rightRaw)
	if leftRaw != "" && !leftOK {
		target.SetText(targetRow, targetColumn, leftRaw)
		return
	}
	if rightRaw != "" && !rightOK {
		target.SetText(targetRow, targetColumn, rightRaw)
		return
	}
	target.SetNumber(targetRow, targetColumn, roundTo2(left+right))
}

func mergeLocationComment(target spreadsheet.Sheet, targetRow, targetColumn int, source spreadsheet.Sheet, sourceRow, sourceColumn int) {
	left := normalize.AsText(target.Value(targetRow, targetColumn))
	right := normalize.AsText(source.Value(sourceRow, sourceColumn))
	switch {
	case left == "":
		target.SetText(targetRow, targetColumn, right)
	case right == "" || right == left:
		target.SetText(targetRow, targetColumn, left)
	default:
		target.SetText(targetRow, targetColumn, left+"; "+right)
	}
}

func copyLocationCell(target spreadsheet.Sheet, targetRow, targetColumn int, source spreadsheet.Sheet, sourceRow, sourceColumn int) {
	target.SetText(targetRow, targetColumn, source.Value(sourceRow, sourceColumn))
}

func sumLocationNumber(target spreadsheet.Sheet, targetRow, targetColumn int, source spreadsheet.Sheet, sourceRow, sourceColumn int) {
	left, _ := normalize.ParseNumber(target.Value(targetRow, targetColumn))
	right, _ := normalize.ParseNumber(source.Value(sourceRow, sourceColumn))
	target.SetNumber(targetRow, targetColumn, roundTo2(left+right))
}

func copyLocationNumber(target spreadsheet.Sheet, targetRow, targetColumn int, source spreadsheet.Sheet, sourceRow, sourceColumn int) {
	value, _ := normalize.ParseNumber(source.Value(sourceRow, sourceColumn))
	target.SetNumber(targetRow, targetColumn, roundTo2(value))
}

func cleanLocationName(value string) string {
	return strings.TrimSpace(strings.TrimPrefix(normalize.NormalizeName(value), "чз "))
}

func compatibleLocationNames(left, right string) bool {
	return cleanLocationName(left) == cleanLocationName(right)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "без артикула"
}
