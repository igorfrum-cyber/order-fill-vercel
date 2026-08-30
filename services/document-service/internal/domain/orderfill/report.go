package orderfill

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"order-fill/services/document-service/internal/domain/brand"
)

// Summary carries the aggregate numbers shown above the review table.
type Summary struct {
	Brand                  string
	OrderMonthLabel        string
	AdjustmentLabel        string
	ActualMainPeriod       string
	ActualPreviousPeriod   string
	SourceCity             string
	CityRule               string
	DeliveryWeeks          float64
	Filled                 int
	LeftBlank              int
	Suspicious             int
	Unmatched              int
	Duplicates             int
	NotInBlank             int
	BlankDuplicateArticles int
	SourceItems            int
	SourceArticles         int
	SourceSheet            string
	SourceHeaderRow        int
	BlankSheet             string
	BlankHeaderRow         int
}

// ReportRow is one reviewable line of the result.
type ReportRow struct {
	Key                 string
	Status              string
	BlankID             string
	BlankLabel          string
	BlankRow            int
	BlankQuantityColumn int
	BlankArticle        string
	BlankName           string
	BlankUnit           string
	BlankBoxSize        string
	SourceRow           *int
	SourceArticle       string
	SourceName          string
	HasOrderedFact      bool
	OrderedFact         *float64
	SourceComment       string
	Stock               string
	InTransit           string
	Recommended         *float64
	Rounded             *int
	BaseRounded         *int
	Inserted            *float64
	AutoComment         string
	AdjustmentLabel     string
	BoxAdjusted         bool
	Duplicate           bool
	DuplicateCandidates []DuplicateCandidate
	Editable            bool
	Similarity          float64
}

func unmatchedRow(position blankPosition, command FillCommand, rule brand.RuleConfig) ReportRow {
	return ReportRow{
		Key:                 position.key,
		Status:              StatusNotInSource,
		BlankID:             command.BlankID,
		BlankLabel:          command.BlankLabel,
		BlankRow:            position.blankRow,
		BlankQuantityColumn: position.blankQuantityColumn,
		BlankArticle:        position.articleRaw,
		BlankName:           position.name,
		BlankUnit:           position.unit,
		BlankBoxSize:        position.boxSize,
		AdjustmentLabel:     rule.AdjustmentLabel,
		Duplicate:           position.duplicate,
		DuplicateCandidates: emptyCandidates(position.duplicateCandidates),
		Editable:            false,
	}
}

func matchedRow(
	status string,
	position blankPosition,
	selected SourceItem,
	score float64,
	order brand.AdjustedQuantity,
	command FillCommand,
	rule brand.RuleConfig,
) ReportRow {
	sourceRow := selected.RowIndex
	recommended := selected.Recommended
	rounded := selected.Rounded
	baseRounded := order.Rounded
	row := ReportRow{
		Key:                 position.key,
		Status:              status,
		BlankID:             command.BlankID,
		BlankLabel:          command.BlankLabel,
		BlankRow:            position.blankRow,
		BlankQuantityColumn: position.blankQuantityColumn,
		BlankArticle:        position.articleRaw,
		BlankName:           position.name,
		BlankUnit:           position.unit,
		BlankBoxSize:        position.boxSize,
		SourceRow:           &sourceRow,
		SourceArticle:       selected.ArticleRaw,
		SourceName:          selected.Name,
		HasOrderedFact:      selected.HasOrderedFact,
		SourceComment:       selected.SourceComment,
		Stock:               selected.Stock,
		InTransit:           selected.InTransit,
		Recommended:         &recommended,
		Rounded:             &rounded,
		BaseRounded:         &baseRounded,
		Inserted:            order.Inserted,
		AutoComment:         order.AutoComment,
		AdjustmentLabel:     rule.AdjustmentLabel,
		BoxAdjusted:         order.BoxAdjusted,
		Duplicate:           position.duplicate,
		DuplicateCandidates: emptyCandidates(position.duplicateCandidates),
		Editable:            true,
		Similarity:          math.Round(score*10000) / 10000,
	}
	if selected.HasOrderedFact {
		fact := selected.OrderedFact
		row.OrderedFact = &fact
	}
	return row
}

// MaxNotInBlankReportRows keeps the review payload bounded when the 1C export
// lists tens of thousands of SKUs that are not on the supplier blank.
const MaxNotInBlankReportRows = 500

// missingFromBlankRows lists source positions that need an order but were not
// represented by any matched blank row. The returned slice is capped; total is
// the true number of missing positions.
func missingFromBlankRows(source Source, rows []ReportRow, command FillCommand, rule brand.RuleConfig) ([]ReportRow, int) {
	matched := map[int]bool{}
	for _, row := range rows {
		if row.SourceRow != nil {
			matched[*row.SourceRow] = true
		}
	}
	missing := make([]ReportRow, 0)
	total := 0
	for _, item := range ItemsMissingFromBlank(source) {
		if matched[item.RowIndex] {
			continue
		}
		total++
		if len(missing) >= MaxNotInBlankReportRows {
			continue
		}
		sourceRow := item.RowIndex
		recommended := item.Recommended
		rounded := item.Rounded
		row := ReportRow{
			Key:                 keyForSourceRow(command.BlankID, "missing", item.RowIndex),
			Status:              StatusNotInBlank,
			BlankID:             command.BlankID,
			BlankLabel:          command.BlankLabel,
			SourceRow:           &sourceRow,
			SourceArticle:       item.ArticleRaw,
			SourceName:          item.Name,
			HasOrderedFact:      item.HasOrderedFact,
			SourceComment:       item.SourceComment,
			Stock:               item.Stock,
			InTransit:           item.InTransit,
			Recommended:         &recommended,
			Rounded:             &rounded,
			AdjustmentLabel:     rule.AdjustmentLabel,
			DuplicateCandidates: []DuplicateCandidate{},
			Editable:            false,
		}
		if item.HasOrderedFact {
			fact := item.OrderedFact
			row.OrderedFact = &fact
		}
		missing = append(missing, row)
	}
	return missing, total
}

// sourceDuplicateRows surfaces articles the 1C export repeats so the buyer can
// clean up the source data. Groups already attached to a blank row are skipped:
// the original UI counted those once, as "Пусто / Дубль" on the matched line.
func sourceDuplicateRows(context SourceContext, existing []ReportRow, command FillCommand, rule brand.RuleConfig) []ReportRow {
	represented := map[string]bool{}
	for _, row := range existing {
		if row.Duplicate {
			represented[duplicateSignature(row.DuplicateCandidates)] = true
		}
	}

	rows := make([]ReportRow, 0)
	for _, group := range context.DuplicateGroups {
		signature := duplicateSignature(group.Candidates)
		if signature == "" || represented[signature] {
			continue
		}
		rows = append(rows, ReportRow{
			Key:                 command.BlankID + ":duplicate:" + group.Article,
			Status:              StatusSourceDuplicate,
			BlankID:             command.BlankID,
			BlankLabel:          command.BlankLabel,
			BlankArticle:        group.Article,
			AdjustmentLabel:     rule.AdjustmentLabel,
			Duplicate:           true,
			DuplicateCandidates: group.Candidates,
			Editable:            false,
		})
	}
	return rows
}

func duplicateSignature(candidates []DuplicateCandidate) string {
	rows := make([]int, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.SourceRow > 0 {
			rows = append(rows, candidate.SourceRow)
		}
	}
	sort.Ints(rows)
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, strconv.Itoa(row))
	}
	return strings.Join(parts, ":")
}

func keyForSourceRow(blankID string, kind string, row int) string {
	return blankID + ":" + kind + ":" + strconv.Itoa(row)
}

func emptyCandidates(candidates []DuplicateCandidate) []DuplicateCandidate {
	if candidates == nil {
		return []DuplicateCandidate{}
	}
	return candidates
}
