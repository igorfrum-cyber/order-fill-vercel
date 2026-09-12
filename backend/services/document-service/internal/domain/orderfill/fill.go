package orderfill

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// Report row statuses. They are part of the API contract with the frontend.
const (
	StatusMatched            = "matched"
	StatusMatchedByName      = "matched_by_name"
	StatusWarningNameDiffers = "warning_name_differs"
	StatusWarningNameOnly    = "warning_name_only"
	StatusLeftBlank          = "left_blank_nonpositive"
	StatusNotInSource        = "not_in_source"
	StatusNotInBlank         = "not_in_blank"
	StatusSourceDuplicate    = "source_duplicate"
)

// FillCommand is the input of the order-fill engine.
type FillCommand struct {
	Source       spreadsheet.Workbook
	Blank        spreadsheet.Workbook
	OrderMonth   string
	Brand        string
	BlankID      string
	BlankLabel   string
	MatchingMode string
	Matcher      Matcher
	Chz          ChzMerger
	Adjuster     QuantityAdjuster
	Recommender  SourceRecommender
	Rule         brand.RuleConfig
	Context      context.Context
	OnProgress   func(fraction float64, message string)
}

func (c FillCommand) ctx() context.Context {
	if c.Context != nil {
		return c.Context
	}
	return context.TODO()
}

func (c FillCommand) report(fraction float64, message string) {
	if c.OnProgress == nil {
		return
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	c.OnProgress(fraction, message)
}

// Result is the outcome of filling one supplier blank.
type Result struct {
	BlankID    string
	BlankLabel string
	Source     spreadsheet.Workbook
	Blank      spreadsheet.Workbook
	Rows       []ReportRow
	Summary    Summary
}

// blankPosition is one orderable line of the supplier blank.
type blankPosition struct {
	key                 string
	blankRow            int
	blankQuantityColumn int
	articleRaw          string
	article             string
	name                string
	unit                string
	boxSize             string
	budgetPrice         float64
	duplicate           bool
	duplicateCandidates []DuplicateCandidate
}

// Fill matches the supplier blank against the 1C export and writes the ordered
// quantities into the blank workbook.
func Fill(command FillCommand) (Result, error) {
	rule := command.Rule
	if rule.Key == "" {
		rule = brand.Rule(command.Brand)
	}
	if command.Matcher == nil {
		return Result{}, fmt.Errorf("matching-service is required")
	}
	if rule.BlankLayout != "" {
		return Result{}, fmt.Errorf("%w: раскладка бланка %q для бренда %s пока не поддерживается сервисом", ErrInvalidInput, rule.BlankLayout, rule.Label)
	}
	if command.Brand == "christina" {
		command.BlankLabel = LabelChristinaBlank(command.Blank, command.BlankLabel)
	}

	command.report(0.02, "Читаю таблицу заказа")
	source, err := readSource(command, rule, func(fraction float64, message string) {
		command.report(0.02+0.30*fraction, message)
	})
	if err != nil {
		return Result{}, err
	}
	context := BuildSourceContext(source, rule)

	blank, err := DetectBlankColumns(command.Blank, rule)
	if err != nil {
		return Result{}, err
	}
	positions := blankPositions(blank, command.BlankID, rule)

	summary := Summary{
		Brand:                  rule.Label,
		AdjustmentLabel:        rule.AdjustmentLabel,
		OrderMonthLabel:        source.PeriodInfo.OrderMonthLabel,
		ActualMainPeriod:       source.PeriodInfo.ActualMainPeriod,
		ActualPreviousPeriod:   source.PeriodInfo.ActualPreviousPeriod,
		SourceCity:             source.SourceCity,
		CityRule:               source.CityRule,
		DeliveryWeeks:          source.Delivery,
		SourceItems:            len(source.Items),
		SourceArticles:         context.ArticleCount,
		SourceSheet:            source.Detection.SheetName,
		SourceHeaderRow:        source.Detection.HeaderRow,
		BlankSheet:             blank.SheetName,
		BlankHeaderRow:         blank.HeaderRow,
		BlankDuplicateArticles: countDuplicateArticles(positions),
	}

	command.report(0.35, "Подбираю позиции бланка")
	matches, err := matchPositions(positions, source, command, rule)
	if err != nil {
		return Result{}, err
	}
	rows := make([]ReportRow, 0, len(matches))
	for _, match := range matches {
		if match.clear {
			blank.Sheet.ClearValue(match.position.blankRow, match.position.blankQuantityColumn)
		} else if match.inserted != nil {
			blank.Sheet.SetNumber(match.position.blankRow, match.position.blankQuantityColumn, *match.inserted)
		}
		summary.Unmatched += match.unmatched
		summary.Suspicious += match.suspicious
		summary.LeftBlank += match.leftBlank
		summary.Filled += match.filled
		summary.Duplicates += match.duplicates
		addCategory(&summary, match.row.Category)
		rows = append(rows, match.row)
	}

	command.report(0.88, "Собираю позиции вне бланка")
	missing, notInBlank := missingFromBlankRows(source, rows, command, rule)
	for _, row := range missing {
		addCategory(&summary, row.Category)
	}
	summary.NotInBlank = notInBlank
	rows = append(rows, missing...)
	dups := sourceDuplicateRows(context, rows, command, rule)
	for _, row := range dups {
		addCategory(&summary, row.Category)
	}
	rows = append(rows, dups...)
	chzRows := chzDecisionRows(source, command, rule)
	for _, row := range chzRows {
		addCategory(&summary, row.Category)
	}
	rows = append(rows, chzRows...)
	command.report(1, "Сверяю итог")

	return Result{
		BlankID:    command.BlankID,
		BlankLabel: command.BlankLabel,
		Source:     source.Workbook,
		Blank:      command.Blank,
		Rows:       rows,
		Summary:    summary,
	}, nil
}

type positionMatch struct {
	position   blankPosition
	row        ReportRow
	clear      bool
	inserted   *float64
	unmatched  int
	suspicious int
	leftBlank  int
	filled     int
	duplicates int
}

func matchPositions(positions []blankPosition, source Source, command FillCommand, rule brand.RuleConfig) ([]positionMatch, error) {
	matches := make([]positionMatch, len(positions))
	if len(positions) == 0 {
		return matches, nil
	}
	blankItems := make([]MatchItem, len(positions))
	for i, position := range positions {
		blankItems[i] = MatchItem{ID: position.key, Article: position.article, Name: position.name, Volume: position.unit}
	}
	sourceItems := make([]MatchItem, len(source.Items))
	byID := make(map[string]SourceItem, len(source.Items))
	for i, item := range source.Items {
		id := strconv.Itoa(item.RowIndex)
		sourceItems[i] = MatchItem{ID: id, Article: item.Article, Name: item.Name, Rounded: item.Rounded}
		byID[id] = item
	}
	results, err := command.Matcher.Match(command.ctx(), blankItems, sourceItems, MatchOptions{
		Mode:           command.MatchingMode,
		PrefixAliases:  rule.ArticlePrefixAliases,
		PreserveHyphen: rule.PreserveArticleHyphen,
	})
	if err != nil {
		return nil, err
	}
	byBlank := make(map[string]MatchResult, len(results))
	for _, result := range results {
		byBlank[result.BlankID] = result
	}
	for i, position := range positions {
		match, err := applyMatch(position, byBlank[position.key], byID, command, rule)
		if err != nil {
			return nil, err
		}
		matches[i] = match
		if i == len(positions)-1 || (i+1)%8 == 0 {
			command.report(0.35+0.50*float64(i+1)/float64(len(positions)), "Подбираю позиции бланка")
		}
	}
	return matches, nil
}

func applyMatch(position blankPosition, result MatchResult, byID map[string]SourceItem, command FillCommand, rule brand.RuleConfig) (positionMatch, error) {
	if len(result.CandidateIDs) > 1 {
		position.duplicate = true
		items := make([]SourceItem, 0, len(result.CandidateIDs))
		for _, id := range result.CandidateIDs {
			if item, ok := byID[id]; ok {
				items = append(items, item)
			}
		}
		position.duplicateCandidates = DuplicateCandidatesFor(items)
	}
	selected := byID[result.SourceID]
	if result.Category == CategoryNotInSource || result.SourceID == "" && result.Category != CategoryNeedsDecision {
		return positionMatch{position: position, row: unmatchedRow(position, command, rule), clear: true, unmatched: 1}, nil
	}
	if result.Category == CategoryNeedsDecision && result.Reasons.Source == "name" {
		order, err := orderForItem(selected, rule, position.boxSize, command)
		if err != nil {
			return positionMatch{}, err
		}
		order.Inserted = nil
		order.AutoComment = ""
		return positionMatch{
			position:   position,
			row:        matchedRow(StatusWarningNameOnly, position, selected, result.Score, order, command, rule, CategoryNeedsDecision, result.Reasons),
			clear:      true,
			suspicious: 1,
		}, nil
	}
	if result.Category == CategoryNeedsDecision {
		row := matchedRow(StatusSourceDuplicate, position, selected, result.Score, brand.AdjustedQuantity{}, command, rule, CategoryNeedsDecision, result.Reasons)
		row.Duplicate = true
		row.Editable = true
		return positionMatch{position: position, row: row, clear: true, duplicates: 1}, nil
	}

	status := StatusMatched
	match := positionMatch{position: position}
	if len(result.CandidateIDs) > 1 {
		match.duplicates = 1
	}
	if result.Category == CategoryCheckNameOrVolume {
		status = StatusWarningNameDiffers
		match.suspicious = 1
	}
	if result.Reasons.Source == "name" {
		status = StatusMatchedByName
	}
	order, err := orderForItem(selected, rule, position.boxSize, command)
	if err != nil {
		return positionMatch{}, err
	}
	if result.Category == CategoryOrderNotNeeded || order.Inserted == nil {
		match.clear = true
		match.leftBlank = 1
		status = StatusLeftBlank
	} else {
		match.inserted = order.Inserted
		match.filled = 1
	}
	match.row = matchedRow(status, position, selected, result.Score, order, command, rule, reportCategory(result, order), result.Reasons)
	return match, nil
}

func blankPositions(blank Detection, blankID string, rule brand.RuleConfig) []blankPosition {
	bounds := blank.Sheet.Bounds()
	priceColumn := budgetPriceColumn(blank)
	positions := make([]blankPosition, 0)
	for row := blank.HeaderRow + 1; row <= bounds.MaxRow; row++ {
		articleRaw := normalize.AsText(blank.Sheet.Value(row, blank.Columns[ColumnArticle]))
		article := normalize.NormalizeArticle(articleRaw, brand.ArticleNormalizeOptions(rule))
		if article == "" {
			continue
		}
		boxSize := ""
		if rule.Adjustment == brand.AdjustmentBox {
			boxSize = normalize.AsText(blank.Sheet.Value(row, blank.Columns[ColumnBoxSize]))
		} else if rule.Multiple > 0 {
			boxSize = fmt.Sprint(rule.Multiple)
		}
		positions = append(positions, blankPosition{
			key:                 fmt.Sprintf("%s:%d", blankID, row),
			blankRow:            row,
			blankQuantityColumn: blank.Columns[ColumnQuantity],
			articleRaw:          articleRaw,
			article:             article,
			name:                normalize.AsText(blank.Sheet.Value(row, blank.Columns[ColumnName])),
			unit:                normalize.AsText(blank.Sheet.Value(row, blank.Columns[ColumnUnit])),
			boxSize:             boxSize,
			budgetPrice:         budgetPriceAt(blank.Sheet, row, priceColumn),
		})
	}
	return positions
}

func budgetPriceColumn(blank Detection) int {
	preferred := 0
	fallback := 0
	for column := 1; column <= blank.Sheet.Bounds().MaxColumn; column++ {
		header := normalize.NormalizeHeader(blank.Sheet.Value(blank.HeaderRow, column))
		if !strings.Contains(header, "цена") || strings.Contains(header, "сумма") {
			continue
		}
		if header == "закупочная цена" || header == "цена закупки" {
			if preferred != 0 {
				return 0
			}
			preferred = column
			continue
		}
		if fallback != 0 {
			fallback = -1
		} else {
			fallback = column
		}
	}
	if preferred != 0 {
		return preferred
	}
	if fallback > 0 {
		return fallback
	}
	return 0
}

func budgetPriceAt(sheet spreadsheet.Sheet, row, column int) float64 {
	if column <= 0 {
		return 0
	}
	value := sheet.Value(row, column)
	number, _ := normalize.ParseNumber(value)
	return number
}

func countDuplicateArticles(positions []blankPosition) int {
	counts := map[string]int{}
	for _, position := range positions {
		counts[position.article]++
	}
	duplicates := 0
	for _, count := range counts {
		if count > 1 {
			duplicates++
		}
	}
	return duplicates
}

// orderForItem applies the brand rounding rule, honouring a quantity the buyer
// already recorded in "Заказано по факту".
func orderForItem(item SourceItem, rule brand.RuleConfig, boxSize string, command FillCommand) (brand.AdjustedQuantity, error) {
	if command.Adjuster != nil {
		return command.Adjuster.AdjustQuantity(command.ctx(), item.Recommended, item.OrderedFact, item.HasOrderedFact, boxSize, rule)
	}
	if !item.HasOrderedFact {
		return brand.CalculateAdjustedQuantity(item.Recommended, rule, boxSize), nil
	}
	order := brand.CalculateAdjustedQuantity(item.OrderedFact, rule, boxSize)
	order.AutoComment = ""
	return order, nil
}
