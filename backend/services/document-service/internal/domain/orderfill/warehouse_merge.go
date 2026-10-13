package orderfill

import (
	"context"
	"fmt"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// TyumenMergeOptions preserves actual orders for standalone recalculation and
// supplies the matching owner for merging ЧЗ before combining locations.
type TyumenMergeOptions struct {
	PreserveFacts bool
	Chz           ChzMerger
	MatchingMode  string
	Context       context.Context
}

var tyumenLocationLabels = []string{"Офис: остаток", "Склад: остаток", "Офис: в пути", "Склад: в пути"}

// MergeTyumenSources combines raw office and delivery-warehouse history. The
// warehouse template is retained when it has sales; otherwise the office sales
// timeline is used. Conflicting identities remain separate for review.
func MergeTyumenSources(office, warehouse spreadsheet.Workbook, orderMonth string, rule brand.RuleConfig, options ...TyumenMergeOptions) (spreadsheet.Workbook, error) {
	if office == nil || warehouse == nil {
		return nil, fmt.Errorf("%w: загрузите обе таблицы Тюмени: офис и склад доставки", ErrInvalidInput)
	}
	option := TyumenMergeOptions{}
	if len(options) > 0 {
		option = options[0]
	}
	if _, err := ValidateSourcePeriods(office, orderMonth); err != nil {
		return nil, fmt.Errorf("таблица офиса: %w", err)
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
	warehouseHasSales := warehouseCalculation != nil
	if warehouseHasSales {
		if _, err := ValidateSourcePeriods(warehouse, orderMonth); err != nil {
			return nil, fmt.Errorf("таблица склада доставки: %w", err)
		}
		if max(1, detectDeliveryWeeks(office)) != max(1, detectDeliveryWeeks(warehouse)) {
			return nil, fmt.Errorf("%w: срок поставки в таблицах офиса и склада различается", ErrInvalidInput)
		}
		if !sameSalesMonths(officeDetection, *officeCalculation, warehouseDetection, *warehouseCalculation) {
			return nil, fmt.Errorf("%w: месяцы продаж в таблицах офиса и склада не совпадают", ErrInvalidInput)
		}
	}
	for _, detection := range []Detection{officeDetection, warehouseDetection} {
		for column := 1; column <= detection.Sheet.Bounds().MaxColumn; column++ {
			for _, label := range tyumenLocationLabels {
				if normalize.NormalizeHeader(detection.Sheet.Value(detection.HeaderRow, column)) == normalize.NormalizeHeader(label) {
					return nil, fmt.Errorf("%w: загружена уже объединённая таблица; нужны отдельные исходные таблицы офиса и склада", ErrInvalidInput)
				}
			}
		}
	}
	command := FillCommand{Chz: option.Chz, MatchingMode: option.MatchingMode, Context: option.Context}
	if _, err := applyChestnyZnak(command, officeDetection, rule, *officeCalculation, nil); err != nil {
		return nil, err
	}
	warehouseLoose := locationCalculationColumns(warehouseDetection)
	if warehouseCalculation != nil {
		warehouseLoose = *warehouseCalculation
	}
	if _, err := applyChestnyZnak(command, warehouseDetection, rule, warehouseLoose, nil); err != nil {
		return nil, err
	}
	officeRows := locationRows(officeDetection, rule)
	warehouseRows := locationRows(warehouseDetection, rule)
	if !warehouseHasSales {
		if len(warehouseLoose.salesColumns) > 0 {
			return nil, fmt.Errorf("%w: в таблице склада неполный ABC-анализ", ErrInvalidInput)
		}
		for _, item := range warehouseRows.byKey {
			if item.number(warehouseLoose.totalQuantity) != 0 || item.number(warehouseLoose.revenue) != 0 {
				return nil, fmt.Errorf("%w: в отчёте склада нет месячных колонок, но у «%s» указаны продажи или выручка; нужен отчёт с разбивкой по месяцам", ErrInvalidInput, item.name)
			}
		}
	}
	if err := alignLocationKeys(&officeRows, &warehouseRows, rule); err != nil {
		return nil, err
	}
	output, target, calculation := office, officeDetection, *officeCalculation
	if warehouseHasSales {
		output, target, calculation = warehouse, warehouseDetection, *warehouseCalculation
	}
	nextRow := target.Sheet.Bounds().MaxRow
	firstExtra := target.Sheet.Bounds().MaxColumn + 1
	for i, label := range tyumenLocationLabels {
		target.Sheet.SetText(target.HeaderRow, firstExtra+i, label)
	}
	keys := append([]string{}, warehouseRows.order...)
	seen := map[string]bool{}
	for _, key := range keys {
		seen[key] = true
	}
	for _, key := range officeRows.order {
		if !seen[key] {
			keys = append(keys, key)
		}
	}
	productRows := make([]int, 0, len(keys))
	for _, key := range keys {
		a, b := officeRows.byKey[key], warehouseRows.byKey[key]
		row := 0
		if warehouseHasSales && b != nil {
			row = b.row
		} else if !warehouseHasSales && a != nil {
			row = a.row
		}
		if row == 0 {
			nextRow++
			row = nextRow
		}
		productRows = append(productRows, row)
		article, name := "", ""
		if b != nil {
			article, name = b.articleRaw, b.name
		}
		if a != nil {
			if article == "" {
				article = a.articleRaw
			}
			if name == "" {
				name = a.name
			}
		}
		if a != nil && b != nil && (hasChzPrefix(a.name) || hasChzPrefix(b.name)) {
			name = "ЧЗ + " + stripLocationChz(name)
		}
		target.Sheet.SetText(row, target.Columns[ColumnArticle], article)
		target.Sheet.SetText(row, target.Columns[ColumnName], name)
		for i, col := range calculation.salesColumns {
			target.Sheet.SetNumber(row, col, roundTo2(a.number(officeCalculation.salesColumns[i])+b.number(columnAt(warehouseLoose.salesColumns, i))))
		}
		for _, cols := range [][3]int{{calculation.revenue, officeCalculation.revenue, warehouseLoose.revenue}, {calculation.previousQuantity, officeCalculation.previousQuantity, warehouseLoose.previousQuantity}, {calculation.totalQuantity, officeCalculation.totalQuantity, warehouseLoose.totalQuantity}} {
			if cols[0] > 0 {
				target.Sheet.SetNumber(row, cols[0], roundTo2(a.number(cols[1])+b.number(cols[2])))
			}
		}
		officeStock, warehouseStock := a.number(officeDetection.Columns[ColumnStock]), b.number(warehouseDetection.Columns[ColumnStock])
		officeTransit, warehouseTransit := a.number(officeDetection.Columns[ColumnInTransit]), b.number(warehouseDetection.Columns[ColumnInTransit])
		target.Sheet.SetNumber(row, target.Columns[ColumnStock], roundTo2(officeStock+warehouseStock))
		target.Sheet.SetNumber(row, target.Columns[ColumnInTransit], roundTo2(officeTransit+warehouseTransit))
		target.Sheet.ClearValue(row, target.Columns[ColumnOrderedFact])
		comments := []string{}
		if option.PreserveFacts {
			fact, hasFact := 0.0, false
			for _, part := range []struct {
				item      *locationRow
				detection Detection
			}{{a, officeDetection}, {b, warehouseDetection}} {
				if part.item == nil {
					continue
				}
				value, ok := normalize.ParseNumber(part.item.values[part.detection.Columns[ColumnOrderedFact]])
				if ok {
					fact += value
					hasFact = true
				}
				comment := part.item.values[part.detection.Columns[ColumnComment]]
				if comment != "" && (len(comments) == 0 || comments[0] != comment) {
					comments = append(comments, comment)
				}
			}
			if hasFact {
				target.Sheet.SetNumber(row, target.Columns[ColumnOrderedFact], roundTo2(fact))
			}
		}
		if a != nil && a.ambiguous || b != nil && b.ambiguous {
			comments = append(comments, "Проверить: неоднозначная позиция. Сохранена отдельно, данные складов по ней не объединены.")
		}
		target.Sheet.SetText(row, target.Columns[ColumnComment], strings.Join(comments, "; "))
		for i, value := range []float64{officeStock, warehouseStock, officeTransit, warehouseTransit} {
			target.Sheet.SetNumber(row, firstExtra+i, value)
		}
	}
	additive := append([]int{}, calculation.salesColumns...)
	additive = append(additive, calculation.totalQuantity, calculation.revenue, calculation.previousQuantity, target.Columns[ColumnStock], target.Columns[ColumnInTransit], firstExtra, firstExtra+1, firstExtra+2, firstExtra+3)
	for row := target.HeaderRow + 1; row <= target.Sheet.Bounds().MaxRow; row++ {
		if isSourceTotalRow(target, row, target.Sheet.Bounds().MaxColumn) {
			sumInto(target.Sheet, row, productRows, additive)
		}
	}
	return output, nil
}

func columnAt(columns []int, index int) int {
	if index < len(columns) {
		return columns[index]
	}
	return 0
}

type locationRow struct {
	row                       int
	article, articleRaw, name string
	ambiguous                 bool
	values                    map[int]string
}

func (row *locationRow) number(column int) float64 {
	if row == nil || column == 0 {
		return 0
	}
	value, _ := normalize.ParseNumber(row.values[column])
	return value
}

type locationIndex struct {
	byKey map[string]*locationRow
	order []string
}

func locationRows(detection Detection, rule brand.RuleConfig) locationIndex {
	rows := locationIndex{byKey: make(map[string]*locationRow)}
	bounds := detection.Sheet.Bounds()
	for row := detection.HeaderRow + 1; row <= bounds.MaxRow; row++ {
		if isSourceTotalRow(detection, row, bounds.MaxColumn) {
			continue
		}
		item := &locationRow{row: row, articleRaw: normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnArticle])), name: normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnName])), values: map[int]string{}}
		item.article = normalize.NormalizeArticle(item.articleRaw, brand.ArticleNormalizeOptions(rule))
		if item.article == "" && item.name == "" {
			continue
		}
		for col := 1; col <= bounds.MaxColumn; col++ {
			item.values[col] = detection.Sheet.Value(row, col)
		}
		key := locationProductKey(item, rule)
		if previous := rows.byKey[key]; previous != nil {
			previous.ambiguous = true
			item.ambiguous = true
			key = fmt.Sprintf("%s:duplicate:%d", key, row)
		}
		rows.byKey[key] = item
		rows.order = append(rows.order, key)
	}
	return rows
}

func alignLocationKeys(office, warehouse *locationIndex, rule brand.RuleConfig) error {
	conflicts := map[string]bool{}
	for _, side := range []*locationIndex{office, warehouse} {
		for _, item := range side.byKey {
			if item.ambiguous {
				conflicts[locationProductKey(item, rule)] = true
			}
		}
	}
	for key, a := range office.byKey {
		if b := warehouse.byKey[key]; b != nil && !compatibleLocationProducts(a, b, rule) {
			conflicts[locationProductKey(a, rule)] = true
		}
	}
	for i, side := range []*locationIndex{office, warehouse} {
		for j, key := range side.order {
			item := side.byKey[key]
			if conflicts[locationProductKey(item, rule)] {
				item.ambiguous = true
				newKey := fmt.Sprintf("review:%d:%d", i, item.row)
				delete(side.byKey, key)
				side.byKey[newKey] = item
				side.order[j] = newKey
			}
		}
	}
	for j, key := range office.order {
		item := office.byKey[key]
		if item.ambiguous || warehouse.byKey[key] != nil {
			continue
		}
		candidates := []string{}
		for _, otherKey := range warehouse.order {
			other := warehouse.byKey[otherKey]
			if !other.ambiguous && (item.article == "" || other.article == "") && compatibleLocationProducts(item, other, rule) {
				candidates = append(candidates, otherKey)
			}
		}
		if len(candidates) > 1 {
			return fmt.Errorf("%w: неоднозначное соответствие по названию «%s» между офисом и складом", ErrInvalidInput, item.name)
		}
		if len(candidates) == 1 {
			targetKey := candidates[0]
			if office.byKey[targetKey] != nil {
				return fmt.Errorf("%w: несколько позиций офиса соответствуют одной позиции склада «%s»", ErrInvalidInput, item.name)
			}
			delete(office.byKey, key)
			office.byKey[targetKey] = item
			office.order[j] = targetKey
		}
	}
	return nil
}

func locationCalculationColumns(detection Detection) calculationColumns {
	found := calculationColumns{}
	for col := 1; col <= detection.Sheet.Bounds().MaxColumn; col++ {
		header := normalize.NormalizeHeader(detection.Sheet.Value(detection.HeaderRow, col))
		upper := detection.Sheet.Value(detection.HeaderRow-1, col)
		switch {
		case col < detection.Columns[ColumnRecommended] && header == "количество" && isMonthHeader(upper):
			found.salesColumns = append(found.salesColumns, col)
		case col < detection.Columns[ColumnRecommended] && header == "количество" && strings.Contains(normalize.NormalizeHeader(upper), "итого"):
			found.totalQuantity = col
		case strings.Contains(header, "сумма") && strings.Contains(header, "выруч"):
			found.revenue = col
		case strings.Contains(header, "количество") && strings.Contains(header, "прошлый"):
			found.previousQuantity = col
		}
	}
	return found
}

func sameSalesMonths(left Detection, leftColumns calculationColumns, right Detection, rightColumns calculationColumns) bool {
	if len(leftColumns.salesColumns) != len(rightColumns.salesColumns) {
		return false
	}
	for i, col := range leftColumns.salesColumns {
		if normalize.NormalizeHeader(left.Sheet.Value(left.HeaderRow-1, col)) != normalize.NormalizeHeader(right.Sheet.Value(right.HeaderRow-1, rightColumns.salesColumns[i])) {
			return false
		}
	}
	return true
}
func cleanLocationName(value string) string { return normalize.NormalizeName(stripLocationChz(value)) }
func compatibleLocationNames(left, right string) bool {
	return cleanLocationName(left) == cleanLocationName(right)
}

func refreshLocationTotals(detection Detection, columns calculationColumns, refs []sourceRowRef) {
	extras := []int{}
	for col := 1; col <= detection.Sheet.Bounds().MaxColumn; col++ {
		for _, label := range tyumenLocationLabels {
			if normalize.NormalizeHeader(detection.Sheet.Value(detection.HeaderRow, col)) == normalize.NormalizeHeader(label) {
				extras = append(extras, col)
			}
		}
	}
	if len(extras) == 0 {
		return
	}
	productRows := make([]int, 0, len(refs))
	for _, ref := range refs {
		productRows = append(productRows, ref.row)
	}
	additive := append([]int{}, columns.salesColumns...)
	additive = append(additive, columns.totalQuantity, columns.revenue, columns.previousQuantity, columns.targetStock, detection.Columns[ColumnStock], detection.Columns[ColumnInTransit], detection.Columns[ColumnRecommended])
	additive = append(additive, extras...)
	for row := detection.HeaderRow + 1; row <= detection.Sheet.Bounds().MaxRow; row++ {
		if isSourceTotalRow(detection, row, detection.Sheet.Bounds().MaxColumn) {
			sumInto(detection.Sheet, row, productRows, additive)
		}
	}
}
