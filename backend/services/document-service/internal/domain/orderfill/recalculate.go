package orderfill

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
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
	categoryColumn int
	salesColumns   []int
}

type sourceRowRef struct {
	row     int
	article string
	name    string
}

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

func calculationColumnsFromUrengoy(info urengoyInfo) calculationColumns {
	return calculationColumns{salesColumns: info.salesColumns, category: info.categoryColumn}
}

func applyChestnyZnak(command FillCommand, detection Detection, rule brand.RuleConfig, columns calculationColumns, onProgress func(float64)) error {
	if command.Chz == nil {
		return nil
	}
	report := func(fraction float64) {
		if onProgress == nil {
			return
		}
		onProgress(min(1, max(0, fraction)))
	}
	refs := readSourceRowRefs(detection, rule)
	items := make([]MatchItem, 0, len(refs))
	for _, ref := range refs {
		items = append(items, MatchItem{ID: strconv.Itoa(ref.row), Article: ref.article, Name: ref.name})
	}
	report(0.05)
	merges, err := command.Chz.MergeChz(command.ctx(), items, MatchOptions{
		Mode:           command.MatchingMode,
		PrefixAliases:  rule.ArticlePrefixAliases,
		PreserveHyphen: rule.PreserveArticleHyphen,
	})
	if err != nil {
		return err
	}
	report(0.45)
	rowsToDelete := make([]int, 0)
	for _, merge := range merges {
		if merge.NeedsDecision || merge.TargetID == "" || len(merge.CloneIDs) == 0 {
			continue
		}
		targetRow, err := strconv.Atoi(merge.TargetID)
		if err != nil {
			continue
		}
		matched := make([]int, 0, len(merge.CloneIDs))
		for _, id := range merge.CloneIDs {
			row, err := strconv.Atoi(id)
			if err != nil {
				continue
			}
			matched = append(matched, row)
		}
		if len(matched) == 0 {
			continue
		}
		targetName := normalize.AsText(detection.Sheet.Value(targetRow, detection.Columns[ColumnName]))
		mergedName := "ЧЗ + " + targetName
		detection.Sheet.SetText(targetRow, detection.Columns[ColumnName], mergedName)
		if detection.Columns[ColumnName] != 1 {
			detection.Sheet.SetText(targetRow, 1, mergedName)
		}
		sumInto(detection.Sheet, targetRow, append([]int{targetRow}, matched...), append(append([]int{}, columns.salesColumns...),
			columns.totalQuantity, columns.revenue, columns.previousQuantity,
			detection.Columns[ColumnStock], detection.Columns[ColumnInTransit]))
		fact, comment := mergeFactAndComment(detection, append([]int{targetRow}, matched...))
		if fact != nil {
			detection.Sheet.SetNumber(targetRow, detection.Columns[ColumnOrderedFact], *fact)
		} else {
			detection.Sheet.ClearValue(targetRow, detection.Columns[ColumnOrderedFact])
		}
		detection.Sheet.SetText(targetRow, detection.Columns[ColumnComment], comment)
		rowsToDelete = append(rowsToDelete, matched...)
	}
	if len(rowsToDelete) > 0 {
		sort.Ints(rowsToDelete)
		detection.Sheet.DeleteRows(rowsToDelete)
	}
	report(1)
	return nil
}

func applyRecommendations(command FillCommand, detection Detection, rule brand.RuleConfig, columns calculationColumns, weeks float64, cityRule string) error {
	refs := readSourceRowRefs(detection, rule)
	in := make([]RecommendRow, 0, len(refs))
	for _, ref := range refs {
		revenue, _ := normalize.ParseNumber(detection.Sheet.Value(ref.row, columns.revenue))
		stock, _ := normalize.ParseNumber(detection.Sheet.Value(ref.row, detection.Columns[ColumnStock]))
		inTransit, _ := normalize.ParseNumber(detection.Sheet.Value(ref.row, detection.Columns[ColumnInTransit]))
		in = append(in, RecommendRow{
			ID:           strconv.Itoa(ref.row),
			Category:     normalize.AsText(detection.Sheet.Value(ref.row, columns.category)),
			Revenue:      revenue,
			Stock:        stock,
			InTransit:    inTransit,
			MonthlySales: monthlyValues(detection.Sheet, ref.row, columns.salesColumns),
		})
	}
	brandKey := command.Brand
	if brandKey == "" {
		brandKey = rule.Key
	}
	out, err := command.Recommender.Recommend(command.ctx(), brandKey, cityRule, weeks, in)
	if err != nil {
		return err
	}
	for _, row := range out {
		target, err := strconv.Atoi(row.ID)
		if err != nil {
			continue
		}
		detection.Sheet.SetNumber(target, detection.Columns[ColumnRecommended], roundTo2(row.Recommended))
		if cityRule == "urengoy" {
			continue
		}
		if columns.totalQuantity > 0 {
			detection.Sheet.SetNumber(target, columns.totalQuantity, roundTo2(row.TotalQuantity))
		}
		if columns.revenuePercent > 0 {
			detection.Sheet.SetNumber(target, columns.revenuePercent, roundTo2(row.RevenuePercent))
		}
		if columns.cumulativePercent > 0 {
			detection.Sheet.SetNumber(target, columns.cumulativePercent, roundTo2(row.CumulativePercent))
		}
		if columns.category > 0 && row.Category != "" {
			detection.Sheet.SetText(target, columns.category, row.Category)
		}
		if columns.averageMonthly > 0 {
			detection.Sheet.SetNumber(target, columns.averageMonthly, roundTo2(row.AverageMonthly))
		}
		if columns.targetStock > 0 {
			detection.Sheet.SetNumber(target, columns.targetStock, roundTo2(row.TargetStock))
		}
	}
	return nil
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
		rows = append(rows, sourceRowRef{row: row, article: article, name: name})
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

func roundTo2(value float64) float64 {
	return math.Round(value*100) / 100
}
