package orderfill

import (
	"fmt"

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

	source, err := ReadSource(command.Source, command.OrderMonth, rule)
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

	rows := make([]ReportRow, 0, len(positions))
	for index := range positions {
		position := &positions[index]
		candidates := context.CandidatesFor(position.article, rule)

		if len(candidates) == 0 {
			fallback, ok := matching.ChooseNameFallback(toMatchingItems(context.NoArticleItems), position.name, position.unit)
			if !ok {
				summary.Unmatched++
				blank.Sheet.ClearValue(position.blankRow, position.blankQuantityColumn)
				rows = append(rows, unmatchedRow(*position, command, rule))
				continue
			}
			selected := findItem(context.NoArticleItems, fallback.Item)
			if selected.Rounded > 0 {
				summary.Suspicious++
				order := orderForItem(selected, rule, position.boxSize)
				order.Inserted = nil
				order.AutoComment = ""
				blank.Sheet.ClearValue(position.blankRow, position.blankQuantityColumn)
				rows = append(rows, matchedRow(StatusWarningNameOnly, *position, selected, fallback.Score, order, command, rule))
				continue
			}
			order := orderForItem(selected, rule, position.boxSize)
			status := StatusMatchedByName
			if order.Inserted == nil {
				blank.Sheet.ClearValue(position.blankRow, position.blankQuantityColumn)
				summary.LeftBlank++
				status = StatusLeftBlank
			} else {
				blank.Sheet.SetNumber(position.blankRow, position.blankQuantityColumn, *order.Inserted)
				summary.Filled++
			}
			rows = append(rows, matchedRow(status, *position, selected, fallback.Score, order, command, rule))
			continue
		}

		if len(candidates) > 1 {
			summary.Duplicates++
			position.duplicate = true
			position.duplicateCandidates = DuplicateCandidatesFor(candidates)
		}
		candidate, _ := matching.ChooseCandidate(toMatchingItems(candidates), position.name, position.unit)
		selected := findItem(candidates, candidate.Item)
		status := StatusMatched
		if candidate.Score < nameMatchThreshold {
			status = StatusWarningNameDiffers
			summary.Suspicious++
		}

		order := orderForItem(selected, rule, position.boxSize)
		if order.Inserted == nil {
			blank.Sheet.ClearValue(position.blankRow, position.blankQuantityColumn)
			summary.LeftBlank++
			status = StatusLeftBlank
		} else {
			blank.Sheet.SetNumber(position.blankRow, position.blankQuantityColumn, *order.Inserted)
			summary.Filled++
		}
		rows = append(rows, matchedRow(status, *position, selected, candidate.Score, order, command, rule))
	}

	rows = append(rows, missingFromBlankRows(source, rows, command, rule)...)
	rows = append(rows, sourceDuplicateRows(context, rows, command, rule)...)

	return Result{
		BlankID:    command.BlankID,
		BlankLabel: command.BlankLabel,
		Source:     source.Workbook,
		Blank:      command.Blank,
		Rows:       rows,
		Summary:    summary,
	}, nil
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
