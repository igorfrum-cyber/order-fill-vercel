package orderfill

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"order-fill/services/document-service/internal/domain/brand"
	"order-fill/services/document-service/internal/domain/matching"
	"order-fill/services/document-service/internal/domain/normalize"
	"order-fill/services/document-service/internal/domain/spreadsheet"
)

// calculationColumns are the ABC-analysis columns of a full 1C order table.
// They are optional: exports that only carry a recommended order are used as is.
type calculationColumns struct {
	salesColumns      []int
	totalQuantity     int
	revenue           int
	revenuePercent    int
	cumulativePercent int
	category          int
	averageMonthly    int
	previousQuantity  int
	targetStock       int
}

type urengoyInfo struct {
	rule                brand.RuleConfig
	deliveryCoefficient float64
	categoryColumn      int
	salesColumns        []int
}

type sourceRowRef struct {
	row     int
	article string
	name    string
	isChz   bool
}

var (
	chzCombinedPattern = regexp.MustCompile(`^\s*чз\s*\+`)
	chzPrefixPattern   = regexp.MustCompile(`^чз\s+`)
	chzWordPattern     = regexp.MustCompile(`\bчз\b`)
	brandNoisePattern  = regexp.MustCompile(`\b(ан|angiopharm)\b`)
	spacesPattern      = regexp.MustCompile(`\s+`)
)

func detectCalculationColumns(detection Detection) *calculationColumns {
	bounds := detection.Sheet.Bounds()
	found := calculationColumns{}
	recommended := detection.Columns[ColumnRecommended]

	for column := 1; column <= bounds.MaxColumn; column++ {
		header := normalize.NormalizeHeader(detection.Sheet.Value(detection.HeaderRow, column))
		if header == "" {
			continue
		}
		upper := detection.Sheet.Value(detection.HeaderRow-1, column)
		upperHeader := normalize.NormalizeHeader(upper)

		switch {
		case column < recommended && header == "количество" && isMonthHeader(upper):
			found.salesColumns = append(found.salesColumns, column)
		case found.totalQuantity == 0 && column < recommended && header == "количество" && strings.Contains(upperHeader, "итого"):
			found.totalQuantity = column
		case found.revenue == 0 && strings.Contains(header, "сумма") && strings.Contains(header, "выруч"):
			found.revenue = column
		case found.revenuePercent == 0 && strings.Contains(header, "%") && strings.Contains(header, "выруч"):
			found.revenuePercent = column
		case found.cumulativePercent == 0 && strings.Contains(header, "кумулятив"):
			found.cumulativePercent = column
		case found.category == 0 && header == "категория":
			found.category = column
		case found.averageMonthly == 0 && strings.Contains(header, "среднее") && strings.Contains(header, "месяц"):
			found.averageMonthly = column
		case found.previousQuantity == 0 && strings.Contains(header, "количество") && strings.Contains(header, "прошлый"):
			found.previousQuantity = column
		case found.targetStock == 0 && strings.Contains(header, "целевой") && strings.Contains(header, "запас"):
			found.targetStock = column
		}
	}

	complete := len(found.salesColumns) > 0 &&
		found.totalQuantity > 0 && found.revenue > 0 && found.revenuePercent > 0 &&
		found.cumulativePercent > 0 && found.category > 0 && found.averageMonthly > 0 &&
		found.previousQuantity > 0 && found.targetStock > 0
	if !complete {
		return nil
	}
	return &found
}

func detectUrengoyColumns(detection Detection) (*urengoyInfo, error) {
	bounds := detection.Sheet.Bounds()
	info := urengoyInfo{}
	recommended := detection.Columns[ColumnRecommended]
	for column := 1; column <= bounds.MaxColumn; column++ {
		header := normalize.NormalizeHeader(detection.Sheet.Value(detection.HeaderRow, column))
		upper := detection.Sheet.Value(detection.HeaderRow-1, column)
		if info.categoryColumn == 0 && header == "категория" {
			info.categoryColumn = column
		}
		if column < recommended && header == "количество" && isMonthHeader(upper) {
			info.salesColumns = append(info.salesColumns, column)
		}
	}
	if info.categoryColumn == 0 {
		return nil, fmt.Errorf("%w: для Уренгоя не нашел колонку «Категория» в таблице заказа", ErrInvalidInput)
	}
	if len(info.salesColumns) == 0 {
		return nil, fmt.Errorf("%w: для Уренгоя не нашел месячные колонки продаж с заголовком «Количество»", ErrInvalidInput)
	}
	return &info, nil
}

func (u *urengoyInfo) recommendedFor(sheet spreadsheet.Sheet, row int) float64 {
	maxSales := maxMonthlySales(sheet, row, u.salesColumns)
	category := sheet.Value(row, u.categoryColumn)
	categoryPart := maxSales * brand.CategoryCoefficient(category, u.rule)
	deliveryPart := maxSales * u.deliveryCoefficient
	return roundTo2(categoryPart + deliveryPart)
}

func maxMonthlySales(sheet spreadsheet.Sheet, row int, columns []int) float64 {
	highest := 0.0
	for _, column := range columns {
		if value, ok := normalize.ParseNumber(sheet.Value(row, column)); ok {
			highest = math.Max(highest, value)
		}
	}
	return highest
}

// rebuildSourceWithChz merges "ЧЗ" clone rows into their base article and then
// recalculates the whole table, mirroring the 1C workflow.
func rebuildSourceWithChz(detection Detection, deliveryWeeks float64, rule brand.RuleConfig, columns calculationColumns) {
	rows := readSourceRowRefs(detection, rule)
	byArticle := map[string][]sourceRowRef{}
	articles := make([]string, 0)
	for _, row := range rows {
		if row.article == "" {
			continue
		}
		if _, seen := byArticle[row.article]; !seen {
			articles = append(articles, row.article)
		}
		byArticle[row.article] = append(byArticle[row.article], row)
	}

	rowsToDelete := make([]int, 0)
	for _, article := range articles {
		group := byArticle[article]
		normalRows := make([]sourceRowRef, 0, len(group))
		chzRows := make([]sourceRowRef, 0, len(group))
		for _, row := range group {
			if row.isChz {
				chzRows = append(chzRows, row)
				continue
			}
			normalRows = append(normalRows, row)
		}
		if len(normalRows) == 0 || len(chzRows) == 0 {
			continue
		}

		target := normalRows[0]
		matched := make([]int, 0, len(chzRows))
		for _, row := range chzRows {
			if matching.Similarity(comparableChzName(target.name), comparableChzName(row.name)) >= 0.9 {
				matched = append(matched, row.row)
			}
		}
		if len(matched) == 0 {
			continue
		}

		merged := append([]int{target.row}, matched...)
		mergedName := "ЧЗ + " + target.name
		detection.Sheet.SetText(target.row, detection.Columns[ColumnName], mergedName)
		if detection.Columns[ColumnName] != 1 {
			detection.Sheet.SetText(target.row, 1, mergedName)
		}
		sumInto(detection.Sheet, target.row, merged, append(append([]int{}, columns.salesColumns...),
			columns.totalQuantity, columns.revenue, columns.previousQuantity,
			detection.Columns[ColumnStock], detection.Columns[ColumnInTransit]))

		fact, comment := mergeFactAndComment(detection, merged)
		if fact != nil {
			detection.Sheet.SetNumber(target.row, detection.Columns[ColumnOrderedFact], *fact)
		} else {
			detection.Sheet.ClearValue(target.row, detection.Columns[ColumnOrderedFact])
		}
		detection.Sheet.SetText(target.row, detection.Columns[ColumnComment], comment)
		rowsToDelete = append(rowsToDelete, matched...)
	}

	if len(rowsToDelete) > 0 {
		sort.Ints(rowsToDelete)
		detection.Sheet.DeleteRows(rowsToDelete)
	}
	recalculateSourceTable(detection, deliveryWeeks, rule, columns)
}

// recalculateSourceTable rebuilds ABC metrics, target stock and the recommended
// order for every product row.
func recalculateSourceTable(detection Detection, deliveryWeeks float64, rule brand.RuleConfig, columns calculationColumns) {
	rows := readSourceRowRefs(detection, rule)
	safeWeeks := math.Max(1, deliveryWeeks)
	deliveryCoefficient := 0.25 * safeWeeks

	type calculatedRow struct {
		row           int
		values        []float64
		totalQuantity float64
		revenue       float64
	}

	totalRevenue := 0.0
	calculated := make([]calculatedRow, 0, len(rows))
	for _, row := range rows {
		values := monthlyValues(detection.Sheet, row.row, columns.salesColumns)
		total := 0.0
		for _, value := range values {
			total += value
		}
		revenue, _ := normalize.ParseNumber(detection.Sheet.Value(row.row, columns.revenue))
		totalRevenue += revenue
		calculated = append(calculated, calculatedRow{row: row.row, values: values, totalQuantity: total, revenue: revenue})
	}

	ranked := make([]calculatedRow, len(calculated))
	copy(ranked, calculated)
	sort.SliceStable(ranked, func(i, j int) bool { return ranked[i].revenue > ranked[j].revenue })

	type metrics struct {
		revenuePercent    float64
		cumulativePercent float64
		category          string
	}
	rankMetrics := make(map[int]metrics, len(ranked))
	cumulative := 0.0
	for _, item := range ranked {
		cumulative += item.revenue
		revenuePercent := 0.0
		cumulativePercent := 100.0
		if totalRevenue > 0 {
			revenuePercent = item.revenue / totalRevenue * 100
			cumulativePercent = cumulative / totalRevenue * 100
		}
		rankMetrics[item.row] = metrics{
			revenuePercent:    revenuePercent,
			cumulativePercent: cumulativePercent,
			category:          categoryFromCumulative(cumulativePercent),
		}
	}

	for _, item := range calculated {
		metric, ok := rankMetrics[item.row]
		if !ok {
			metric = metrics{cumulativePercent: 100, category: "C"}
		}
		novelty := newProductCalculation(item.values, safeWeeks)
		monthlyNeed := calculateTargetNew(item.values)
		category := metric.category
		targetStock := 0.0
		if novelty != nil {
			category = metric.category + "/New"
			targetStock = novelty.targetStock
		} else {
			targetStock = monthlyNeed*brand.CategoryCoefficient(metric.category, rule) + monthlyNeed*deliveryCoefficient
		}

		stock, _ := normalize.ParseNumber(detection.Sheet.Value(item.row, detection.Columns[ColumnStock]))
		inTransit, _ := normalize.ParseNumber(detection.Sheet.Value(item.row, detection.Columns[ColumnInTransit]))
		recommended := math.Max(0, targetStock-stock-inTransit)

		detection.Sheet.SetNumber(item.row, columns.totalQuantity, roundTo2(item.totalQuantity))
		detection.Sheet.SetNumber(item.row, columns.revenuePercent, roundTo2(metric.revenuePercent))
		detection.Sheet.SetNumber(item.row, columns.cumulativePercent, roundTo2(metric.cumulativePercent))
		detection.Sheet.SetText(item.row, columns.category, category)
		detection.Sheet.SetNumber(item.row, columns.averageMonthly, roundTo2(item.totalQuantity/float64(len(columns.salesColumns))))
		detection.Sheet.SetNumber(item.row, columns.targetStock, roundTo2(targetStock))
		detection.Sheet.SetNumber(item.row, detection.Columns[ColumnRecommended], roundTo2(recommended))
	}
}

type noveltyCalculation struct {
	monthlyNeed float64
	targetStock float64
}

// newProductCalculation recognises a product whose sales start in the last one
// to three months and sizes the first orders from its peak month.
func newProductCalculation(values []float64, deliveryWeeks float64) *noveltyCalculation {
	suffix := 0
	for index := len(values) - 1; index >= 0; index-- {
		if values[index] <= 0 {
			break
		}
		suffix++
	}
	if suffix < 1 || suffix > 3 {
		return nil
	}
	firstNovelty := len(values) - suffix
	for _, value := range values[:firstNovelty] {
		if value > 0 {
			return nil
		}
	}
	maxMonth := 0.0
	for _, value := range values[firstNovelty:] {
		maxMonth = math.Max(maxMonth, value)
	}
	if maxMonth <= 0 {
		return nil
	}
	deliveryCoefficient := 0.25 * math.Max(1, deliveryWeeks)
	return &noveltyCalculation{
		monthlyNeed: maxMonth * 1.5,
		targetStock: maxMonth*1.5 + maxMonth*deliveryCoefficient,
	}
}

// calculateTargetNew averages monthly sales, favouring a stable sales history.
func calculateTargetNew(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	total := 0.0
	for _, value := range values {
		total += value
	}
	threshold := total / 24
	stable := true
	lowest := math.Inf(1)
	for _, value := range values {
		if value <= 0 {
			stable = false
		}
		lowest = math.Min(lowest, value)
	}
	if stable && lowest > threshold {
		average := total / float64(len(values))
		secondMonth := 0.0
		if len(values) > 1 {
			secondMonth = values[1]
		}
		firstThree := 0.0
		for index := 0; index < 3 && index < len(values); index++ {
			firstThree += values[index]
		}
		return (average + secondMonth + firstThree/3) / 3
	}

	filteredTotal := 0.0
	filteredCount := 0
	for _, value := range values {
		if value > 0 && value > threshold {
			filteredTotal += value
			filteredCount++
		}
	}
	if filteredCount == 0 {
		return 0
	}
	return filteredTotal / float64(filteredCount)
}

func categoryFromCumulative(percent float64) string {
	switch {
	case percent <= 50:
		return "A+"
	case percent <= 80:
		return "A"
	case percent <= 95:
		return "B"
	default:
		return "C"
	}
}

func readSourceRowRefs(detection Detection, rule brand.RuleConfig) []sourceRowRef {
	bounds := detection.Sheet.Bounds()
	rows := make([]sourceRowRef, 0)
	for row := detection.HeaderRow + 1; row <= bounds.MaxRow; row++ {
		if isSourceTotalRow(detection, row, bounds.MaxColumn) {
			continue
		}
		articleRaw := normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnArticle]))
		name := normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnName]))
		article := normalize.NormalizeArticle(articleRaw, brand.ArticleNormalizeOptions(rule))
		if article == "" && name == "" {
			continue
		}
		rows = append(rows, sourceRowRef{row: row, article: article, name: name, isChz: isChzCloneName(name)})
	}
	return rows
}

func monthlyValues(sheet spreadsheet.Sheet, row int, columns []int) []float64 {
	values := make([]float64, 0, len(columns))
	for _, column := range columns {
		value, _ := normalize.ParseNumber(sheet.Value(row, column))
		values = append(values, value)
	}
	return values
}

func sumInto(sheet spreadsheet.Sheet, targetRow int, rows []int, columns []int) {
	for _, column := range columns {
		if column <= 0 {
			continue
		}
		total := 0.0
		for _, row := range rows {
			value, _ := normalize.ParseNumber(sheet.Value(row, column))
			total += value
		}
		sheet.SetNumber(targetRow, column, roundTo2(total))
	}
}

func mergeFactAndComment(detection Detection, rows []int) (*float64, string) {
	total := 0.0
	hasFact := false
	comments := make([]string, 0, len(rows))
	seen := map[string]bool{}
	for _, row := range rows {
		if value, ok := normalize.ParseNumber(detection.Sheet.Value(row, detection.Columns[ColumnOrderedFact])); ok {
			total += value
			hasFact = true
		}
		comment := normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnComment]))
		if comment == "" || seen[comment] {
			continue
		}
		seen[comment] = true
		comments = append(comments, comment)
	}
	if !hasFact {
		return nil, strings.Join(comments, "; ")
	}
	rounded := roundTo2(total)
	return &rounded, strings.Join(comments, "; ")
}

func isChzCloneName(value string) bool {
	header := normalize.NormalizeHeader(value)
	return strings.HasPrefix(header, "чз ") && !chzCombinedPattern.MatchString(strings.ToLower(normalize.AsText(value)))
}

func comparableChzName(value string) string {
	text := normalize.NormalizeHeader(value)
	text = chzPrefixPattern.ReplaceAllString(text, "")
	text = chzWordPattern.ReplaceAllString(text, " ")
	text = brandNoisePattern.ReplaceAllString(text, " ")
	return strings.TrimSpace(spacesPattern.ReplaceAllString(text, " "))
}

func roundTo2(value float64) float64 {
	return math.Round(value*100) / 100
}
