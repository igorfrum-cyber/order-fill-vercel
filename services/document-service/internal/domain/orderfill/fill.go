package orderfill

import (
	"fmt"
	"sync/atomic"

	"order-fill/services/document-service/internal/domain/brand"
	"order-fill/services/document-service/internal/domain/matching"
	"order-fill/services/document-service/internal/domain/normalize"
	"order-fill/services/document-service/internal/domain/spreadsheet"
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

// nameMatchThreshold is the similarity below which an article match is flagged
// for review.
const nameMatchThreshold = 0.32

// FillCommand is the input of the order-fill engine.
type FillCommand struct {
	Source     spreadsheet.Workbook
	Blank      spreadsheet.Workbook
	OrderMonth string
	Brand      string
	BlankID    string
	BlankLabel string
	OnProgress func(fraction float64, message string)
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
	duplicate           bool
	duplicateCandidates []DuplicateCandidate
}

// Fill matches the supplier blank against the 1C export and writes the ordered
// quantities into the blank workbook.
func Fill(command FillCommand) (Result, error) {
	rule := brand.Rule(command.Brand)
	if rule.BlankLayout != "" {
		return Result{}, fmt.Errorf("%w: раскладка бланка %q для бренда %s пока не поддерживается сервисом", ErrInvalidInput, rule.BlankLayout, rule.Label)
	}

	command.report(0.02, "Читаю таблицу заказа")
	source, err := readSource(command.Source, command.OrderMonth, rule, func(fraction float64, message string) {
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
	matches := matchPositions(positions, context, command, rule)
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
		rows = append(rows, match.row)
	}

	command.report(0.88, "Собираю позиции вне бланка")
	missing, notInBlank := missingFromBlankRows(source, rows, command, rule)
	summary.NotInBlank = notInBlank
	rows = append(rows, missing...)
	rows = append(rows, sourceDuplicateRows(context, rows, command, rule)...)
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

func matchPositions(positions []blankPosition, context SourceContext, command FillCommand, rule brand.RuleConfig) []positionMatch {
	matches := make([]positionMatch, len(positions))
	if len(positions) == 0 {
		return matches
	}
	var done atomic.Int64
	total := len(positions)
	runWorkers(len(positions), func(index int) {
		matches[index] = resolvePosition(positions[index], context, command, rule)
		current := done.Add(1)
		if current == int64(total) || current%8 == 0 {
			command.report(0.35+0.50*float64(current)/float64(total), "Подбираю позиции бланка")
		}
	})
	return matches
}

func resolvePosition(position blankPosition, context SourceContext, command FillCommand, rule brand.RuleConfig) positionMatch {
	candidates := context.CandidatesFor(position.article, rule)
	if len(candidates) == 0 {
		fallback, ok := matching.ChooseNameFallback(toMatchingItems(context.NoArticleItems), position.name, position.unit)
		if !ok {
			return positionMatch{
				position:  position,
				row:       unmatchedRow(position, command, rule),
				clear:     true,
				unmatched: 1,
			}
		}
		selected := findItem(context.NoArticleItems, fallback.Item)
		if selected.Rounded > 0 {
			order := orderForItem(selected, rule, position.boxSize)
			order.Inserted = nil
			order.AutoComment = ""
			return positionMatch{
				position:   position,
				row:        matchedRow(StatusWarningNameOnly, position, selected, fallback.Score, order, command, rule),
				clear:      true,
				suspicious: 1,
			}
		}
		order := orderForItem(selected, rule, position.boxSize)
		status := StatusMatchedByName
		match := positionMatch{position: position, row: matchedRow(status, position, selected, fallback.Score, order, command, rule)}
		if order.Inserted == nil {
			match.clear = true
			match.leftBlank = 1
			match.row.Status = StatusLeftBlank
		} else {
			match.inserted = order.Inserted
			match.filled = 1
		}
		return match
	}

	if len(candidates) > 1 {
		position.duplicate = true
		position.duplicateCandidates = DuplicateCandidatesFor(candidates)
	}
	candidate, _ := matching.ChooseCandidate(toMatchingItems(candidates), position.name, position.unit)
	selected := findItem(candidates, candidate.Item)
	status := StatusMatched
	match := positionMatch{position: position, duplicates: 0}
	if len(candidates) > 1 {
		match.duplicates = 1
	}
	if candidate.Score < nameMatchThreshold {
		status = StatusWarningNameDiffers
		match.suspicious = 1
	}
	order := orderForItem(selected, rule, position.boxSize)
	if order.Inserted == nil {
		match.clear = true
		match.leftBlank = 1
		status = StatusLeftBlank
	} else {
		match.inserted = order.Inserted
		match.filled = 1
	}
	match.row = matchedRow(status, position, selected, candidate.Score, order, command, rule)
	return match
}

func blankPositions(blank Detection, blankID string, rule brand.RuleConfig) []blankPosition {
	bounds := blank.Sheet.Bounds()
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
		})
	}
	return positions
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
func orderForItem(item SourceItem, rule brand.RuleConfig, boxSize string) brand.AdjustedQuantity {
	if !item.HasOrderedFact {
		return brand.CalculateAdjustedQuantity(item.Recommended, rule, boxSize)
	}
	order := brand.CalculateAdjustedQuantity(item.OrderedFact, rule, boxSize)
	order.AutoComment = ""
	return order
}

func toMatchingItems(items []SourceItem) []matching.Item {
	converted := make([]matching.Item, 0, len(items))
	for index, item := range items {
		converted = append(converted, matching.Item{Ref: index, Article: item.Article, Name: item.Name, Rounded: item.Rounded})
	}
	return converted
}

func findItem(items []SourceItem, target matching.Item) SourceItem {
	if target.Ref < 0 || target.Ref >= len(items) {
		return SourceItem{}
	}
	return items[target.Ref]
}
