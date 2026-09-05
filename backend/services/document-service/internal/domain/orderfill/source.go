package orderfill

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// SourceItem is one product row of the 1C order calculation table.
type SourceItem struct {
	RowIndex       int
	ArticleRaw     string
	Article        string
	Name           string
	Recommended    float64
	Rounded        int
	HasOrderedFact bool
	OrderedFact    float64
	SourceComment  string
	Stock          string
	InTransit      string
}

// Source is the parsed 1C export together with the metadata the report shows.
type Source struct {
	Workbook   spreadsheet.Workbook
	Detection  Detection
	Items      []SourceItem
	PeriodInfo PeriodInfo
	SourceCity string
	CityRule   string
	Delivery   float64
}

// SourceContext indexes source items for matching.
type SourceContext struct {
	byArticle       map[string][]SourceItem
	NoArticleItems  []SourceItem
	DuplicateGroups []DuplicateGroup
	ArticleCount    int
}

// DuplicateGroup is one article present in several source rows.
type DuplicateGroup struct {
	Article    string
	Candidates []DuplicateCandidate
}

// DuplicateCandidate is a competing source row shown to the reviewer.
type DuplicateCandidate struct {
	SourceRow     int     `json:"source_row"`
	SourceArticle string  `json:"source_article"`
	SourceName    string  `json:"source_name"`
	Recommended   float64 `json:"recommended"`
	Rounded       int     `json:"rounded"`
	Stock         string  `json:"stock"`
	InTransit     string  `json:"in_transit"`
}

var digitsPattern = regexp.MustCompile(`^\d+$`)

// ReadSource validates the export period and extracts every product row.
func ReadSource(workbook spreadsheet.Workbook, orderMonth string, rule brand.RuleConfig) (Source, error) {
	return readSource(workbook, orderMonth, rule, nil)
}

func readSource(workbook spreadsheet.Workbook, orderMonth string, rule brand.RuleConfig, report func(float64, string)) (Source, error) {
	reportProgress(report, 0, "Читаю таблицу заказа")
	periodInfo, err := ValidateSourcePeriods(workbook, orderMonth)
	if err != nil {
		return Source{}, err
	}
	detection, err := DetectSourceColumns(workbook)
	if err != nil {
		return Source{}, err
	}

	deliveryWeeks := math.Max(1, detectDeliveryWeeks(workbook))
	calculationColumns := detectCalculationColumns(detection)
	if calculationColumns != nil {
		reportProgress(report, 0.08, "Объединяю строки ЧЗ")
		rebuildSourceWithChz(detection, deliveryWeeks, rule, *calculationColumns, func(fraction float64) {
			reportProgress(report, 0.08+0.45*fraction, "Объединяю строки ЧЗ")
		})
	}

	urengoy := (*urengoyInfo)(nil)
	cityRule := ""
	if calculationColumns == nil && rule.Label != "ANGIOPHARM" && isUrengoySource(workbook) {
		detected, err := detectUrengoyColumns(detection)
		if err != nil {
			return Source{}, err
		}
		detected.rule = rule
		detected.deliveryCoefficient = 1 + 0.25*deliveryWeeks
		urengoy = detected
		cityRule = "Новый Уренгой"
	}

	bounds := detection.Sheet.Bounds()
	items, err := collectSourceItems(detection, rule, urengoy, bounds, report)
	if err != nil {
		return Source{}, err
	}
	reportProgress(report, 1, "Читаю таблицу заказа")

	return Source{
		Workbook:   workbook,
		Detection:  detection,
		Items:      items,
		PeriodInfo: periodInfo,
		CityRule:   cityRule,
		Delivery:   deliveryWeeks,
	}, nil
}

type scannedSourceRow struct {
	item SourceItem
	keep bool
	err  error
}

func collectSourceItems(detection Detection, rule brand.RuleConfig, urengoy *urengoyInfo, bounds spreadsheet.Bounds, report func(float64, string)) ([]SourceItem, error) {
	start := detection.HeaderRow + 1
	if bounds.MaxRow < start {
		return nil, nil
	}
	count := bounds.MaxRow - start + 1
	scanned := make([]scannedSourceRow, count)
	options := brand.ArticleNormalizeOptions(rule)
	total := count
	if total < 1 {
		total = 1
	}

	scan := func(offset int) {
		row := start + offset
		if isSourceTotalRow(detection, row, bounds.MaxColumn) {
			return
		}
		articleRaw := normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnArticle]))
		name := normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnName]))
		recommendedRaw := detection.Sheet.Value(row, detection.Columns[ColumnRecommended])

		if urengoy != nil && (articleRaw != "" || name != "" || normalize.AsText(recommendedRaw) != "") {
			detection.Sheet.SetNumber(row, detection.Columns[ColumnRecommended], urengoy.recommendedFor(detection.Sheet, row))
		}

		recommendedValue, hasRecommended := normalize.ParseNumber(detection.Sheet.Value(row, detection.Columns[ColumnRecommended]))
		orderedFactRaw := detection.Sheet.Value(row, detection.Columns[ColumnOrderedFact])
		orderedFact, hasParsedFact := normalize.ParseNumber(orderedFactRaw)
		hasOrderedFact := normalize.AsText(orderedFactRaw) != ""

		if articleRaw == "" && name == "" && !hasRecommended {
			return
		}
		if hasOrderedFact && !hasParsedFact {
			scanned[offset] = scannedSourceRow{err: fmt.Errorf("%w: в строке %d таблицы заказа некорректно заполнено «Заказано по факту»", ErrInvalidInput, row)}
			return
		}
		scanned[offset] = scannedSourceRow{
			keep: true,
			item: SourceItem{
				RowIndex:       row,
				ArticleRaw:     articleRaw,
				Article:        normalize.NormalizeArticle(articleRaw, options),
				Name:           name,
				Recommended:    recommendedValue,
				Rounded:        normalize.RoundHalfUp(recommendedValue),
				HasOrderedFact: hasOrderedFact,
				OrderedFact:    orderedFact,
				SourceComment:  normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnComment])),
				Stock:          normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnStock])),
				InTransit:      normalize.AsText(detection.Sheet.Value(row, detection.Columns[ColumnInTransit])),
			},
		}
		if offset%512 == 0 {
			reportProgress(report, 0.55+0.45*float64(offset)/float64(total), "Читаю таблицу заказа")
		}
	}

	if urengoy != nil || count < 32 {
		for offset := 0; offset < count; offset++ {
			scan(offset)
		}
	} else {
		runWorkers(count, scan)
	}

	items := make([]SourceItem, 0, count)
	for _, row := range scanned {
		if row.err != nil {
			return nil, row.err
		}
		if row.keep {
			items = append(items, row.item)
		}
	}
	return items, nil
}

// BuildSourceContext indexes source items by every article alias of the brand.
func BuildSourceContext(source Source, rule brand.RuleConfig) SourceContext {
	byArticle := map[string][]SourceItem{}
	noArticle := make([]SourceItem, 0)
	distinct := map[string]bool{}
	for _, item := range source.Items {
		if item.Article == "" {
			noArticle = append(noArticle, item)
			continue
		}
		distinct[item.Article] = true
		for _, key := range articleKeys(item.Article, rule) {
			byArticle[key] = append(byArticle[key], item)
		}
	}
	return SourceContext{
		byArticle:       byArticle,
		NoArticleItems:  noArticle,
		DuplicateGroups: duplicateGroups(byArticle),
		ArticleCount:    len(distinct),
	}
}

// CandidatesFor returns every source row registered under an article.
func (c SourceContext) CandidatesFor(article string, rule brand.RuleConfig) []SourceItem {
	found := make([]SourceItem, 0)
	for _, key := range articleKeys(article, rule) {
		found = append(found, c.byArticle[key]...)
	}
	return uniqueBySourceRow(found)
}

func articleKeys(article string, rule brand.RuleConfig) []string {
	if article == "" {
		return nil
	}
	keys := []string{article}
	for _, prefix := range rule.ArticlePrefixAliases {
		switch {
		case strings.HasPrefix(article, prefix) && digitsPattern.MatchString(article[len(prefix):]):
			keys = append(keys, article[len(prefix):])
		case digitsPattern.MatchString(article):
			keys = append(keys, prefix+article)
		}
	}
	return keys
}

func uniqueBySourceRow(items []SourceItem) []SourceItem {
	seen := map[int]bool{}
	unique := make([]SourceItem, 0, len(items))
	for _, item := range items {
		if seen[item.RowIndex] {
			continue
		}
		seen[item.RowIndex] = true
		unique = append(unique, item)
	}
	return unique
}

func duplicateGroups(byArticle map[string][]SourceItem) []DuplicateGroup {
	articles := make([]string, 0, len(byArticle))
	for article := range byArticle {
		articles = append(articles, article)
	}
	sort.Strings(articles)

	seen := map[string]bool{}
	groups := make([]DuplicateGroup, 0)
	for _, article := range articles {
		items := uniqueBySourceRow(byArticle[article])
		if len(items) < 2 {
			continue
		}
		rows := make([]string, 0, len(items))
		for _, item := range items {
			rows = append(rows, fmt.Sprint(item.RowIndex))
		}
		sort.Strings(rows)
		signature := strings.Join(rows, ":")
		if seen[signature] {
			continue
		}
		seen[signature] = true
		groups = append(groups, DuplicateGroup{Article: article, Candidates: DuplicateCandidatesFor(items)})
	}
	return groups
}

// DuplicateCandidatesFor projects source rows into the reviewer-facing shape.
func DuplicateCandidatesFor(items []SourceItem) []DuplicateCandidate {
	candidates := make([]DuplicateCandidate, 0, len(items))
	for _, item := range items {
		candidates = append(candidates, DuplicateCandidate{
			SourceRow:     item.RowIndex,
			SourceArticle: item.ArticleRaw,
			SourceName:    item.Name,
			Recommended:   item.Recommended,
			Rounded:       item.Rounded,
			Stock:         item.Stock,
			InTransit:     item.InTransit,
		})
	}
	return candidates
}

// ItemsMissingFromBlank lists source positions that need an order but were not
// matched by any blank row.
func ItemsMissingFromBlank(source Source) []SourceItem {
	missing := make([]SourceItem, 0)
	for _, item := range source.Items {
		if item.Recommended > 0 {
			missing = append(missing, item)
		}
	}
	return missing
}

func isSourceTotalRow(detection Detection, row int, maxColumn int) bool {
	end := min(maxColumn, max(1, detection.Columns[ColumnName]-1))
	for column := 1; column <= end; column++ {
		text := normalize.NormalizeHeader(detection.Sheet.Value(row, column))
		if text == "итого" || text == "total" {
			return true
		}
	}
	return false
}

func detectDeliveryWeeks(workbook spreadsheet.Workbook) float64 {
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 40); row++ {
			for column := 1; column <= bounds.MaxColumn; column++ {
				text := normalize.AsText(sheet.Value(row, column))
				header := normalize.NormalizeHeader(text)
				if !strings.Contains(header, "срок") || !strings.Contains(header, "постав") {
					continue
				}
				if weeks, ok := normalize.ParseNumber(text); ok && weeks > 0 {
					return weeks
				}
			}
		}
	}
	return 1
}

func isUrengoySource(workbook spreadsheet.Workbook) bool {
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 40); row++ {
			for column := 1; column <= bounds.MaxColumn; column++ {
				if strings.Contains(normalize.NormalizeHeader(sheet.Value(row, column)), "уренгой") {
					return true
				}
			}
		}
	}
	return false
}

func isMonthHeader(value string) bool {
	header := normalize.NormalizeHeader(value)
	if !regexp.MustCompile(`\b20\d{2}\b`).MatchString(header) {
		return false
	}
	for _, month := range monthsRU[1:] {
		if strings.Contains(header, month) {
			return true
		}
	}
	return false
}

func reportProgress(report func(float64, string), fraction float64, message string) {
	if report == nil {
		return
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	report(fraction, message)
}
