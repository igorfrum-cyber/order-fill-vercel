package calculation

import (
	"cmp"
	"errors"
	"math"
	"slices"

	"order-fill/backend/services/calculation-service/internal/domain"
)

// CHRISTINA PROFF procurement pricing, independent of the workbook/export price.
// BudgetRow.Price is the unit price AFTER the main discount, in rubles; the set
// discount below is the extra 5% on complete sets of three. Only "proff" rows
// with line membership qualify — HOME never does. Mirrors origin/main
// christinaLines.js 1:1.

const (
	// christinaSetSize is the supplier set/step: discounts exist only in
	// multiples of three, and manual non-multiple quantities still count whole
	// sets (18,18,18,3 → 3 sets).
	christinaSetSize = 3
	// christinaSetDiscount is the extra fraction knocked off each unit of a
	// complete set, on top of the main discount already baked into price.
	christinaSetDiscount = 0.05
)

// christinaLineCoverage caps months of coverage per ABC category when topping a
// PROFF line up to the next set. Matches origin/main LINE_COVERAGE.
var christinaLineCoverage = map[string]float64{"C": 2, "B": 2.5, "A": 3, "A+": 3.5}

// LineGroup is one CHRISTINA PROFF line with its computed set discount. rowIdx
// points back into the rows slice passed to ChristinaLineGroups.
type LineGroup struct {
	ID          string
	Name        string
	Required    []string
	Valid       bool
	Sets        int
	BaseCents   int64
	SavingCents int64
	NetCents    int64
	rowIdx      []int
}

// ChristinaLineGroups groups PROFF rows by line, validates membership, and
// computes the per-line set discount. Incomplete or duplicated metadata never
// earns a discount. Mirrors origin/main christinaLineGroups.
func ChristinaLineGroups(rows []domain.BudgetRow) []LineGroup {
	index := make(map[string]int)
	var groups []*LineGroup
	for i, row := range rows {
		if row.Group != "proff" || row.Line == nil {
			continue
		}
		id := row.Line.ID
		gi, ok := index[id]
		if !ok {
			gi = len(groups)
			index[id] = gi
			groups = append(groups, &LineGroup{ID: id, Name: row.Line.Name, Required: row.Line.Required, Valid: true})
		}
		g := groups[gi]
		g.rowIdx = append(g.rowIdx, i)
		if !sameArticleSet(row.Line.Required, g.Required) {
			g.Valid = false
		}
	}
	out := make([]LineGroup, len(groups))
	for gi, g := range groups {
		distinct := distinctCount(g.Required)
		articles := make([]string, len(g.rowIdx))
		for k, idx := range g.rowIdx {
			articles[k] = rows[idx].Line.Article
		}
		g.Valid = g.Valid &&
			distinct > 0 && distinct == len(g.Required) &&
			len(articles) == distinct && distinctCount(articles) == len(articles) &&
			everyIn(articles, g.Required) &&
			allRowsSafe(rows, g.rowIdx)
		if g.Valid {
			g.Sets = int(math.Floor(minQuantity(rows, g.rowIdx)))
		}
		if g.Sets < christinaSetSize {
			g.Sets = 0
		}
		for _, idx := range g.rowIdx {
			r := rows[idx]
			g.BaseCents += budgetCents(r.Price * r.Quantity)
			if g.Sets > 0 {
				g.SavingCents += budgetCents(r.Price * float64(g.Sets) * christinaSetDiscount)
			}
		}
		g.NetCents = g.BaseCents - g.SavingCents
		out[gi] = *g
	}
	return out
}

// ProcurementTotalCents is the whole-order cost in kopecks: gross rounded line
// values minus every line's set discount. Never rounds each set separately.
// Mirrors origin/main procurementTotalCents.
func ProcurementTotalCents(rows []domain.BudgetRow) int64 {
	var sum int64
	for _, r := range rows {
		sum += budgetCents(r.Price * r.Quantity)
	}
	for _, g := range ChristinaLineGroups(rows) {
		sum -= g.SavingCents
	}
	return sum
}

// BudgetBaselines captures each line's net cost at the start of a planning run.
// The 130% growth ceiling is measured against these fixed baselines and never
// reset between steps. Mirrors origin/main's default baselines map.
func BudgetBaselines(rows []domain.BudgetRow) map[string]int64 {
	base := make(map[string]int64)
	for _, g := range ChristinaLineGroups(rows) {
		base[g.ID] = g.NetCents
	}
	return base
}

// LineStep is one atomic next-set proposal. On rejection nothing is added; the
// reason explains why the UI skipped the line. Mirrors origin/main
// proposeLineStep's return shape.
type LineStep struct {
	Accepted    bool
	Reason      string
	Rows        []domain.BudgetRow
	ID          string
	Name        string
	Sets        int
	AddedCents  int64
	SavedCents  int64
	TotalCents  int64
	FirstSingle bool
	Rank        [3]int // counts of A+, A, B rows in the line
}

// ProposeLineStep evaluates bumping line id to its next full set. baselines are
// the fixed net costs from BudgetBaselines. Mirrors origin/main proposeLineStep.
func ProposeLineStep(rows []domain.BudgetRow, id string, baselines map[string]int64, target float64) LineStep {
	groups := ChristinaLineGroups(rows)
	li := slices.IndexFunc(groups, func(g LineGroup) bool { return g.ID == id })
	if li < 0 || !groups[li].Valid {
		return LineStep{Reason: "incompleteMetadata"}
	}
	line := groups[li]
	nextSets := max(christinaSetSize, (line.Sets/christinaSetSize+1)*christinaSetSize)

	var changes []int
	for _, idx := range line.rowIdx {
		if rows[idx].Quantity < float64(nextSets) {
			changes = append(changes, idx)
		}
	}
	// Exception: the first three sets may complete one missing article even
	// without sales or within stock limits. It cannot repeat for later sets.
	firstSingle := line.Sets == 0 && len(changes) == 1 && rows[changes[0]].Quantity == 0

	if ProcurementTotalCents(rows) >= budgetCents(target) && !firstSingle {
		return LineStep{Reason: "targetReached"}
	}
	for _, idx := range changes {
		if rows[idx].Locked || rows[idx].Excluded {
			return LineStep{Reason: "locked"}
		}
	}
	if !firstSingle {
		for _, idx := range changes {
			r := rows[idx]
			cover, ok := Coverage(r, float64(nextSets))
			norm, hasNorm := christinaLineCoverage[r.Category]
			if !ok || !finiteNumber(r.Delivery) || !hasNorm ||
				cover > norm+r.Delivery+1e-8 || cover > 6+1e-8 {
				return LineStep{Reason: "coverage"}
			}
		}
	}

	proposed := slices.Clone(rows)
	for _, idx := range changes {
		proposed[idx].Quantity = float64(nextSets)
	}
	after := ChristinaLineGroups(proposed)
	ai := slices.IndexFunc(after, func(g LineGroup) bool { return g.ID == id })

	var addedCents int64
	for _, idx := range changes {
		r := rows[idx]
		addedCents += budgetCents(r.Price*float64(nextSets)) - budgetCents(r.Price*r.Quantity)
	}
	savedCents := after[ai].SavingCents - line.SavingCents

	baseline, ok := baselines[id]
	if !ok || baseline <= 0 || after[ai].NetCents > int64(math.Floor(float64(baseline)*1.3+1e-8)) {
		return LineStep{Reason: "lineGrowth"}
	}
	if addedCents <= 0 || savedCents*100 < addedCents*15 {
		return LineStep{Reason: "saving"}
	}
	totalCents := ProcurementTotalCents(proposed)
	if totalCents > int64(math.Floor(float64(budgetCents(target))*1.05+1e-8)) {
		return LineStep{Reason: "targetLimit"}
	}
	return LineStep{
		Accepted: true, Rows: proposed, ID: id, Name: line.Name, Sets: after[ai].Sets,
		AddedCents: addedCents, SavedCents: savedCents, TotalCents: totalCents, FirstSingle: firstSingle,
		Rank: [3]int{
			countCategory(rows, line.rowIdx, "A+"),
			countCategory(rows, line.rowIdx, "A"),
			countCategory(rows, line.rowIdx, "B"),
		},
	}
}

// Rejection records why a line could not take its next set this pass.
type Rejection struct {
	ID     string
	Reason string
}

// CompleteResult is the outcome of a CHRISTINA completion pre-pass.
type CompleteResult struct {
	Rows     []domain.BudgetRow
	Steps    []domain.CompletionStep
	Rejected []Rejection
}

// CompleteChristinaLines greedily completes eligible PROFF sets first: pick more
// A+, then A, then B, then the best saving-to-cost ratio, breaking ties by id.
// The caller then runs the normal coverage top-up. Mirrors origin/main
// completeChristinaLines.
func CompleteChristinaLines(input []domain.BudgetRow, target float64, baselines map[string]int64) (CompleteResult, error) {
	if !finiteNumber(target) || target < 0 {
		return CompleteResult{}, errors.New("введите неотрицательную сумму")
	}
	rows := slices.Clone(input)
	var steps []domain.CompletionStep
	for range 250000 {
		groups := ChristinaLineGroups(rows)
		proposals := make([]LineStep, len(groups))
		for i, g := range groups {
			proposals[i] = ProposeLineStep(rows, g.ID, baselines, target)
			proposals[i].ID = g.ID
		}
		var candidates []LineStep
		for _, p := range proposals {
			if p.Accepted {
				candidates = append(candidates, p)
			}
		}
		if len(candidates) == 0 {
			var rejected []Rejection
			for _, p := range proposals {
				if !p.Accepted {
					rejected = append(rejected, Rejection{ID: p.ID, Reason: p.Reason})
				}
			}
			return CompleteResult{Rows: rows, Steps: steps, Rejected: rejected}, nil
		}
		slices.SortFunc(candidates, func(a, b LineStep) int {
			return cmp.Or(
				cmp.Compare(b.Rank[0], a.Rank[0]),
				cmp.Compare(b.Rank[1], a.Rank[1]),
				cmp.Compare(b.Rank[2], a.Rank[2]),
				cmp.Compare(savingRatio(b), savingRatio(a)),
				cmp.Compare(a.ID, b.ID),
			)
		})
		chosen := candidates[0]
		rows = chosen.Rows
		steps = append(steps, domain.CompletionStep{
			ID: chosen.ID, Name: chosen.Name, Sets: chosen.Sets,
			AddedCents: chosen.AddedCents, SavedCents: chosen.SavedCents,
		})
	}
	return CompleteResult{}, errors.New("достигнут предел расчета комплектов, уточните сумму")
}

func savingRatio(s LineStep) float64 {
	if s.AddedCents == 0 {
		return 0
	}
	return float64(s.SavedCents) / float64(s.AddedCents)
}

// sameArticleSet reports whether two required lists hold the same articles.
func sameArticleSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	sa, sb := slices.Clone(a), slices.Clone(b)
	slices.Sort(sa)
	slices.Sort(sb)
	return slices.Equal(sa, sb)
}

func distinctCount(vals []string) int {
	seen := make(map[string]struct{}, len(vals))
	for _, v := range vals {
		seen[v] = struct{}{}
	}
	return len(seen)
}

func everyIn(vals, set []string) bool {
	member := make(map[string]struct{}, len(set))
	for _, v := range set {
		member[v] = struct{}{}
	}
	for _, v := range vals {
		if _, ok := member[v]; !ok {
			return false
		}
	}
	return true
}

func allRowsSafe(rows []domain.BudgetRow, idx []int) bool {
	for _, i := range idx {
		r := rows[i]
		if r.Unsafe || !(r.Price > 0) || !finiteNumber(r.Quantity) || r.Quantity < 0 {
			return false
		}
	}
	return true
}

func minQuantity(rows []domain.BudgetRow, idx []int) float64 {
	m := math.Inf(1)
	for _, i := range idx {
		m = min(m, rows[i].Quantity)
	}
	return m
}

func countCategory(rows []domain.BudgetRow, idx []int, category string) int {
	n := 0
	for _, i := range idx {
		if rows[i].Category == category {
			n++
		}
	}
	return n
}
