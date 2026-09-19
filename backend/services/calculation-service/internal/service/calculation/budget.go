package calculation

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"order-fill/backend/services/calculation-service/internal/domain"
)

var (
	budgetNorms = map[string]float64{"C": 2, "B": 2.5, "A": 3, "A+": 3.5}
	budgetCats  = []string{"C", "B", "A", "A+"}
)

// DiscountValue matches origin/main budgetPlanner.discountValue.
func DiscountValue(value string) (float64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, errors.New("введите скидку числом")
	}
	trimmed = strings.TrimSuffix(trimmed, "%")
	trimmed = strings.ReplaceAll(trimmed, ",", ".")
	n, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n >= 100 {
		return 0, errors.New("скидка должна быть от 0 до 99,99%")
	}
	return n, nil
}

// Coverage matches origin/main budgetPlanner.coverage. ok is false when demand <= 0 (JS null).
func Coverage(row domain.BudgetRow, quantity float64) (float64, bool) {
	if row.Demand <= 0 {
		return 0, false
	}
	return (row.Stock + row.Transit - row.Outbound + quantity*row.Unit) / row.Demand, true
}

func nextQuantity(row domain.BudgetRow, q, direction float64) float64 {
	if direction > 0 {
		return max(row.Minimum, (math.Floor(q/row.Step+1e-8)+1)*row.Step)
	}
	next := (math.Ceil(q/row.Step-1e-8) - 1) * row.Step
	if next < row.Minimum {
		return 0
	}
	return max(0, next)
}

func budgetCents(n float64) int64 {
	return int64(math.Round(n * 100))
}

func finiteNumber(n float64) bool {
	return !math.IsNaN(n) && !math.IsInf(n, 0)
}

type budgetCand struct {
	idx                   int
	q, months, nextMonths float64
	cost                  int64
}

func eligibleBudget(r domain.BudgetRow) bool {
	_, known := budgetNorms[r.Category]
	return !r.Locked && !r.Excluded && !r.Unsafe && r.Price > 0 && r.Demand > 0 && known
}

func collectBudgetCands(rows []domain.BudgetRow, up, fast bool) []budgetCand {
	dir := 1.0
	if !up {
		dir = -1
	}
	lines := proffLineMembers(rows)
	var out []budgetCand
	for i, r := range rows {
		if !eligibleBudget(r) {
			continue
		}
		q := nextQuantity(r, r.Quantity, dir)
		if q == r.Quantity {
			continue
		}
		months, _ := Coverage(r, r.Quantity)
		nextMonths, _ := Coverage(r, q)
		out = append(out, budgetCand{
			idx: i, q: q, months: months, nextMonths: nextMonths,
			cost: changeCost(rows, i, q, fast, lines),
		})
	}
	return out
}

// NormalizeChristinaProffMode maps empty/UNSPECIFIED/unknown to standard.
func NormalizeChristinaProffMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case domain.ChristinaProffFast:
		return domain.ChristinaProffFast
	case domain.ChristinaProffCompare:
		return domain.ChristinaProffCompare
	default:
		return domain.ChristinaProffStandard
	}
}

func useFastOracle(opts domain.BudgetOptions) bool {
	return NormalizeChristinaProffMode(opts.ChristinaProffMode) == domain.ChristinaProffFast
}

func proffLineMembers(rows []domain.BudgetRow) map[string][]int {
	out := make(map[string][]int)
	for i, r := range rows {
		if r.Group == "proff" && r.Line != nil {
			out[r.Line.ID] = append(out[r.Line.ID], i)
		}
	}
	return out
}

func lineMembers(rows []domain.BudgetRow, id string, index map[string][]int) []int {
	if index != nil {
		return index[id]
	}
	var out []int
	for i, r := range rows {
		if r.Group == "proff" && r.Line != nil && r.Line.ID == id {
			out = append(out, i)
		}
	}
	return out
}

// changeCost is the whole-order kopeck delta of setting rows[i] to q. A PROFF
// row's quantity changes the set discount on its siblings. Non-PROFF is O(1)
// per-row cents. Standard PROFF re-evaluates the full order; fast PROFF is O(s)
// on the touched line and must match the full oracle as int64.
//
// ponytail: standard is O(n) per call via ProcurementTotalCents → O(n²) per
// planning iteration. Fast is O(s) after one O(n) member index per iteration.
func changeCost(rows []domain.BudgetRow, i int, q float64, fast bool, lines map[string][]int) int64 {
	r := rows[i]
	if r.Group != "proff" || r.Line == nil {
		return budgetCents(r.Price*q) - budgetCents(r.Price*r.Quantity)
	}
	if fast {
		return changeCostFast(rows, i, q, lineMembers(rows, r.Line.ID, lines))
	}
	before := rows[i].Quantity
	oldTotal := ProcurementTotalCents(rows)
	rows[i].Quantity = q
	cost := ProcurementTotalCents(rows) - oldTotal
	rows[i].Quantity = before
	return cost
}

// changeCostFast is the incremental PROFF oracle: Δgross on the touched row
// minus Δsaving over the line. Invalid/incomplete lines have Sets=0, so cost
// is Δgross only. Does not mutate rows.
func changeCostFast(rows []domain.BudgetRow, i int, q float64, members []int) int64 {
	r := rows[i]
	dGross := budgetCents(r.Price*q) - budgetCents(r.Price*r.Quantity)
	oldValid, oldSets := proffLineSets(rows, members, -1, 0)
	newValid, newSets := proffLineSets(rows, members, i, q)
	if !oldValid {
		oldSets = 0
	}
	if !newValid {
		newSets = 0
	}
	var dSaving int64
	for _, j := range members {
		p := rows[j].Price
		dSaving += budgetCents(p*float64(newSets)*christinaSetDiscount) - budgetCents(p*float64(oldSets)*christinaSetDiscount)
	}
	return dGross - dSaving
}

func proffLineSets(rows []domain.BudgetRow, members []int, overrideIdx int, overrideQ float64) (valid bool, sets int) {
	if len(members) == 0 {
		return false, 0
	}
	first := rows[members[0]]
	if first.Line == nil {
		return false, 0
	}
	required := first.Line.Required
	for _, idx := range members {
		row := rows[idx]
		if row.Line == nil || !sameArticleSet(row.Line.Required, required) {
			return false, 0
		}
	}
	distinct := distinctCount(required)
	articles := make([]string, len(members))
	for k, idx := range members {
		articles[k] = rows[idx].Line.Article
	}
	if !(distinct > 0 && distinct == len(required) &&
		len(articles) == distinct && distinctCount(articles) == len(articles) &&
		everyIn(articles, required) &&
		allRowsSafeAt(rows, members, overrideIdx, overrideQ)) {
		return false, 0
	}
	m := math.Inf(1)
	for _, idx := range members {
		qty := rows[idx].Quantity
		if idx == overrideIdx {
			qty = overrideQ
		}
		m = min(m, qty)
	}
	sets = int(math.Floor(m))
	if sets < christinaSetSize {
		sets = 0
	}
	return true, sets
}

func allRowsSafeAt(rows []domain.BudgetRow, idx []int, overrideIdx int, overrideQ float64) bool {
	for _, i := range idx {
		r := rows[i]
		q := r.Quantity
		if i == overrideIdx {
			q = overrideQ
		}
		if r.Unsafe || !(r.Price > 0) || !finiteNumber(q) || q < 0 {
			return false
		}
	}
	return true
}

func pickUpStage(rows []domain.BudgetRow, candidates []budgetCand) string {
	for _, cat := range budgetCats {
		norm := budgetNorms[cat]
		for _, c := range candidates {
			if rows[c.idx].Category == cat && c.months < norm+rows[c.idx].Delivery-1e-8 {
				return cat
			}
		}
	}
	return ""
}

func filterStage(rows []domain.BudgetRow, candidates []budgetCand, stage string) []budgetCand {
	if stage == "" {
		return candidates
	}
	norm := budgetNorms[stage]
	var out []budgetCand
	for _, c := range candidates {
		if rows[c.idx].Category == stage && c.months < norm+rows[c.idx].Delivery-1e-8 {
			out = append(out, c)
		}
	}
	return out
}

func firstPresentCategory(rows []domain.BudgetRow, candidates []budgetCand) string {
	for _, k := range budgetCats {
		for _, c := range candidates {
			if rows[c.idx].Category == k {
				return k
			}
		}
	}
	return ""
}

func filterCategory(rows []domain.BudgetRow, candidates []budgetCand, cat string) []budgetCand {
	var out []budgetCand
	for _, c := range candidates {
		if rows[c.idx].Category == cat {
			out = append(out, c)
		}
	}
	return out
}

func sortBudgetUp(rows []domain.BudgetRow, candidates []budgetCand, goal, amount int64) {
	slices.SortFunc(candidates, func(a, b budgetCand) int {
		na := budgetNorms[rows[a.idx].Category] + rows[a.idx].Delivery
		nb := budgetNorms[rows[b.idx].Category] + rows[b.idx].Delivery
		if d := cmp.Compare(a.months/na, b.months/nb); d != 0 {
			return d
		}
		return cmp.Compare(math.Abs(float64(goal-amount-a.cost)), math.Abs(float64(goal-amount-b.cost)))
	})
}

func sortBudgetDown(candidates []budgetCand, goal, amount int64) {
	slices.SortFunc(candidates, func(a, b budgetCand) int {
		if d := cmp.Compare(b.months, a.months); d != 0 {
			return d
		}
		return cmp.Compare(math.Abs(float64(goal-amount-a.cost)), math.Abs(float64(goal-amount-b.cost)))
	})
}

// planBudgetCore is the pure coverage planner: no PROFF line awareness beyond
// the set discount folded into ProcurementTotalCents and changeCost. It copies
// input and never mutates it. Mirrors origin/main planBudgetCore.
func planBudgetCore(input []domain.BudgetRow, target float64, opts domain.BudgetOptions) (domain.BudgetPlan, error) {
	if !finiteNumber(target) || target < 0 {
		return domain.BudgetPlan{}, errors.New("введите неотрицательную сумму")
	}
	rows := make([]domain.BudgetRow, len(input))
	for i, r := range input {
		r.Before = r.Quantity
		r.HasBefore = true
		rows[i] = r
		if !(r.Price > 0) && r.Quantity > 0 {
			return domain.BudgetPlan{}, fmt.Errorf("не найдена закупочная цена: %s", r.Name)
		}
		if !finiteNumber(r.Quantity) || !finiteNumber(r.Stock) || !finiteNumber(r.Transit) ||
			!finiteNumber(r.Unit) || !finiteNumber(r.Step) || !finiteNumber(r.Minimum) ||
			r.Quantity < 0 || r.Unit <= 0 || r.Step <= 0 {
			return domain.BudgetPlan{}, fmt.Errorf("некорректные данные: %s", r.Name)
		}
	}
	fast := useFastOracle(opts)
	total := func() int64 { return ProcurementTotalCents(rows) }
	start := total()
	goal := budgetCents(target)
	up := goal > start
	amount := start
	reason := ""
	for iterations := 0; (up && amount < goal) || (!up && amount > goal); {
		iterations++
		if iterations > 250000 {
			reason = "Достигнут предел вычислений. Уточните сумму."
			break
		}
		candidates := collectBudgetCands(rows, up, fast)
		if up {
			var belowCap []budgetCand
			for _, c := range candidates {
				if c.nextMonths <= 6+1e-8 {
					belowCap = append(belowCap, c)
				}
			}
			if len(belowCap) > 0 {
				candidates = belowCap
			} else if !opts.AllowOverSix && len(candidates) > 0 {
				reason = "overSix"
				break
			}
			candidates = filterStage(rows, candidates, pickUpStage(rows, candidates))
			cap := int64(math.Floor(float64(goal)*1.05 + 1e-8))
			var within []budgetCand
			for _, c := range candidates {
				if amount+c.cost <= cap {
					within = append(within, c)
				}
			}
			candidates = within
			sortBudgetUp(rows, candidates, goal, amount)
		} else {
			var protectedRows []budgetCand
			for _, c := range candidates {
				if c.nextMonths >= 1+rows[c.idx].Delivery-1e-8 {
					protectedRows = append(protectedRows, c)
				}
			}
			if len(protectedRows) > 0 {
				candidates = protectedRows
			} else if !opts.AllowBelowOne && len(candidates) > 0 {
				reason = "belowOne"
				break
			} else {
				candidates = filterCategory(rows, candidates, firstPresentCategory(rows, candidates))
			}
			sortBudgetDown(candidates, goal, amount)
		}
		if len(candidates) == 0 {
			reason = "Нет допустимых изменений для достижения суммы с учетом цен, закреплений и партий."
			break
		}
		chosen := candidates[0]
		rows[chosen.idx].Quantity = chosen.q
		amount += chosen.cost
	}
	if !up && amount < goal && reason == "" {
		idx := make([]int, len(rows))
		for i := range rows {
			idx[i] = i
		}
		slices.SortFunc(idx, func(a, b int) int {
			return cmp.Compare(slices.Index(budgetCats, rows[b].Category), slices.Index(budgetCats, rows[a].Category))
		})
		lines := proffLineMembers(rows)
		for _, i := range idx {
			r := &rows[i]
			for r.Quantity < r.Before {
				q := nextQuantity(*r, r.Quantity, 1)
				cost := changeCost(rows, i, q, fast, lines)
				if q > r.Before || amount+cost > goal {
					break
				}
				r.Quantity = q
				amount += cost
			}
		}
	}
	complete := reason == "" && ((up && amount >= goal && float64(amount) <= float64(goal)*1.05) || (!up && amount <= goal))
	return domain.BudgetPlan{
		Rows: rows, Before: float64(start) / 100, Total: float64(amount) / 100,
		Target: target, Reason: reason, Complete: complete,
	}, nil
}

// PlanBudget matches origin/main planBudget: an atomic CHRISTINA PROFF
// set-completion pre-pass, then the normal coverage top-up, then a final
// first-set completion that can still lower the total. Copies input and never
// mutates it, so callers can preview safely.
//
// ChristinaProffMode "compare" runs standard and fast independently. Applied
// Rows/Total/Complete/LineSteps/Reason are the standard plan; Fast* holds the
// fast plan for download/diff. Greedy decisions never mix the two oracles.
func PlanBudget(input []domain.BudgetRow, target float64, opts domain.BudgetOptions) (domain.BudgetPlan, error) {
	mode := NormalizeChristinaProffMode(opts.ChristinaProffMode)
	opts.ChristinaProffMode = mode
	if mode == domain.ChristinaProffCompare {
		return planBudgetCompare(input, target, opts)
	}
	plan, err := planBudgetRun(input, target, opts)
	if err != nil {
		return domain.BudgetPlan{}, err
	}
	plan.ChristinaProffMode = mode
	return plan, nil
}

func planBudgetCompare(input []domain.BudgetRow, target float64, opts domain.BudgetOptions) (domain.BudgetPlan, error) {
	stdOpts := opts
	stdOpts.ChristinaProffMode = domain.ChristinaProffStandard
	t0 := time.Now()
	std, err := planBudgetRun(input, target, stdOpts)
	stdMs := time.Since(t0).Milliseconds()
	if err != nil {
		return domain.BudgetPlan{}, err
	}
	fastOpts := opts
	fastOpts.ChristinaProffMode = domain.ChristinaProffFast
	t1 := time.Now()
	fast, err := planBudgetRun(input, target, fastOpts)
	fastMs := time.Since(t1).Milliseconds()
	if err != nil {
		return domain.BudgetPlan{}, err
	}
	return attachCompareResult(std, fast, stdMs, fastMs), nil
}

func attachCompareResult(std, fast domain.BudgetPlan, stdMs, fastMs int64) domain.BudgetPlan {
	std.ChristinaProffMode = domain.ChristinaProffCompare
	std.FastRows = fast.Rows
	std.FastTotal = fast.Total
	std.FastComplete = fast.Complete
	std.FastReason = fast.Reason
	std.FastLineSteps = fast.LineSteps
	std.Mismatches = diffBudgetPlans(std, fast)
	std.CompareMatch = len(std.Mismatches) == 0
	std.CompareStandardMs = stdMs
	std.CompareFastMs = fastMs
	if len(std.Mismatches) > 0 {
		m := std.Mismatches[0]
		std.CompareMismatchWhere = m.Where
		std.CompareMismatchKey = m.Key
		std.CompareMismatchWant = m.Want
		std.CompareMismatchGot = m.Got
	}
	return std
}

func boolInt(v bool) int64 {
	if v {
		return 1
	}
	return 0
}

func diffBudgetPlans(std, fast domain.BudgetPlan) []domain.BudgetOracleMismatch {
	type view struct {
		qty     float64
		comment string
	}
	stdRows := make(map[string]view, len(std.Rows))
	for _, r := range std.Rows {
		stdRows[r.Key] = view{r.Quantity, BudgetChangeComment(r)}
	}
	fastRows := make(map[string]view, len(fast.Rows))
	for _, r := range fast.Rows {
		fastRows[r.Key] = view{r.Quantity, BudgetChangeComment(r)}
	}
	keys := make(map[string]struct{}, len(stdRows)+len(fastRows))
	for k := range stdRows {
		keys[k] = struct{}{}
	}
	for k := range fastRows {
		keys[k] = struct{}{}
	}
	var out []domain.BudgetOracleMismatch
	for _, key := range slices.Sorted(maps.Keys(keys)) {
		s, sok := stdRows[key]
		f, fok := fastRows[key]
		if !sok || !fok || s.qty != f.qty {
			out = append(out, domain.BudgetOracleMismatch{
				Where: "quantity", Key: key,
				Want: budgetCents(s.qty), Got: budgetCents(f.qty),
			})
		}
		if !sok || !fok || s.comment != f.comment {
			out = append(out, domain.BudgetOracleMismatch{Where: "comment", Key: key})
		}
	}
	if budgetCents(std.Total) != budgetCents(fast.Total) {
		out = append(out, domain.BudgetOracleMismatch{
			Where: "total", Want: budgetCents(std.Total), Got: budgetCents(fast.Total),
		})
	}
	if std.Complete != fast.Complete {
		out = append(out, domain.BudgetOracleMismatch{
			Where: "complete", Want: boolInt(std.Complete), Got: boolInt(fast.Complete),
		})
	}
	n := max(len(std.LineSteps), len(fast.LineSteps))
	for i := range n {
		var s, f domain.CompletionStep
		if i < len(std.LineSteps) {
			s = std.LineSteps[i]
		}
		if i < len(fast.LineSteps) {
			f = fast.LineSteps[i]
		}
		if s.ID != f.ID || s.Name != f.Name || s.Sets != f.Sets || s.AddedCents != f.AddedCents || s.SavedCents != f.SavedCents {
			key := s.ID
			if key == "" {
				key = f.ID
			}
			out = append(out, domain.BudgetOracleMismatch{
				Where: "line_steps", Key: key,
				Want: s.AddedCents, Got: f.AddedCents,
			})
		}
	}
	return out
}

func planBudgetRun(input []domain.BudgetRow, target float64, opts domain.BudgetOptions) (domain.BudgetPlan, error) {
	original := make([]domain.BudgetRow, len(input))
	for i, r := range input {
		r.Quantity = numberOrZero(r.Quantity)
		original[i] = r
	}
	hasProff := slices.ContainsFunc(original, func(r domain.BudgetRow) bool {
		return r.Group == "proff" && r.Line != nil
	})
	before := float64(ProcurementTotalCents(original)) / 100
	if !hasProff || target < before {
		return planBudgetCore(input, target, opts)
	}
	// Validate the same row contract even when the line pass reaches the target.
	if _, err := planBudgetCore(original, before, opts); err != nil {
		return domain.BudgetPlan{}, err
	}
	baselines := BudgetBaselines(original)
	starting := make(map[string]float64, len(original))
	for _, r := range original {
		starting[r.Key] = r.Quantity
	}
	rows := original
	var result domain.BudgetPlan
	var steps []domain.CompletionStep
	// A final first-set completion can reduce the total, so resume the top-up if
	// needed; a completed line cannot receive that exception a second time.
	fast := useFastOracle(opts)
	for pass := 0; pass <= len(baselines)+1; pass++ {
		lines, err := completeChristinaLines(rows, target, baselines, fast)
		if err != nil {
			return domain.BudgetPlan{}, err
		}
		rows = lines.Rows
		steps = append(steps, lines.Steps...)
		if float64(ProcurementTotalCents(rows))/100 >= target {
			result = domain.BudgetPlan{Rows: rows, Total: float64(ProcurementTotalCents(rows)) / 100, Target: target,
				Complete: float64(ProcurementTotalCents(rows))/100 <= target*1.05}
			break
		}
		result, err = planBudgetCore(rows, target, opts)
		if err != nil {
			return domain.BudgetPlan{}, err
		}
		rows = result.Rows
		if !result.Complete {
			break
		}
		finish, err := completeChristinaLines(rows, target, baselines, fast)
		if err != nil {
			return domain.BudgetPlan{}, err
		}
		rows = finish.Rows
		steps = append(steps, finish.Steps...)
		result.Rows = rows
		result.Total = float64(ProcurementTotalCents(rows)) / 100
		if result.Total >= target {
			break
		}
	}

	completed := make(map[string]bool, len(steps))
	for _, s := range steps {
		completed[s.ID] = true
	}
	for i := range rows {
		key := rows[i].Key
		rows[i].Before = starting[key]
		rows[i].HasBefore = true
		rows[i].LineCompletion = ""
		if rows[i].Line != nil && completed[rows[i].Line.ID] && rows[i].Quantity > starting[key] {
			rows[i].LineCompletion = rows[i].Line.Name
		}
	}
	totalCents := ProcurementTotalCents(rows)
	result.Rows = rows
	result.Before = before
	result.Complete = result.Complete &&
		totalCents >= int64(math.Round(target*100)) &&
		totalCents <= int64(math.Floor(target*105+1e-8))
	result.LineSteps = steps
	return result, nil
}

// PlanReportBudget prices raw report rows (main discount + brand order rules +
// delivery months) and runs PlanBudget. It is the owner-side entry point so the
// browser and gateway pass only raw numbers and never compute pricing, brand
// multiples or the CHRISTINA set discount. Mirrors origin/main
// budgetRowsFromReport + planBudget.
func PlanReportBudget(req domain.BudgetRequest) (domain.BudgetPlan, error) {
	factor := 1 - req.Discount/100
	rows := make([]domain.BudgetRow, len(req.Rows))
	for i, r := range req.Rows {
		unit, step, minimum := BudgetOrderRules(req.Brand, r.Name, r.BoxSize)
		rows[i] = domain.BudgetRow{
			Key: r.Key, Name: r.Name, Category: r.Category, Quantity: r.Quantity,
			Price:    math.Round(r.BasePrice*factor*100) / 100,
			Demand:   r.Demand,
			Delivery: req.DeliveryWeeks * 0.25,
			Stock:    r.Stock, Transit: r.Transit, Outbound: r.Outbound,
			Unit: unit, Step: step, Minimum: minimum,
			Locked: r.Locked, Excluded: r.Excluded, Unsafe: r.Unsafe,
			Group: r.Group, Line: r.Line,
		}
	}
	return PlanBudget(rows, req.Target, req.Options)
}

// numberOrZero mirrors JS Number(x || 0): NaN and 0 collapse to 0.
func numberOrZero(v float64) float64 {
	if v == 0 || math.IsNaN(v) {
		return 0
	}
	return v
}

// BudgetOrderRules matches origin/main budgetOrderRules.
func BudgetOrderRules(brand, name string, box float64) (unit, step, minimum float64) {
	if brand == "novacutan" {
		unit = NovacutanSupplierUnitSize(name)
		minimum = NovacutanMinimumQuantity(name)
		if box > 0 {
			minimum = box
		}
		step = 10
		if unit > 1 {
			step = 1
		}
		return unit, step, math.Ceil(minimum/step) * step
	}
	rule := brandAdjustment(brand)
	switch rule.kind {
	case AdjustmentMultiple, AdjustmentNearestMultiple:
		step = float64(rule.multiple)
	case AdjustmentBox:
		n := box
		if n == 0 || math.IsNaN(n) {
			n = 1
		}
		step = max(1, n)
	default:
		step = 1
	}
	minimum = step
	if rule.kind == AdjustmentMinimum {
		n := box
		if n == 0 || math.IsNaN(n) {
			n = 1
		}
		minimum = max(1, n)
	}
	return 1, step, minimum
}

// BudgetChangeComment matches origin/main budgetChangeComment. A positive change
// on a completed PROFF line also names which line was filled.
func BudgetChangeComment(row domain.BudgetRow) string {
	if !row.HasBefore {
		return ""
	}
	delta := row.Quantity - row.Before
	switch {
	case delta > 0:
		note := additionComment(delta, row.Unit)
		if row.LineCompletion != "" {
			note += " Дополнение комплектов " + row.LineCompletion + "."
		}
		return note
	case delta < 0:
		return "Уменьшено на " + ruMoney(-delta) + " " + unitLabel(row.Unit) + " Для снижения заказа до указанной суммы."
	default:
		return ""
	}
}

func additionComment(quantity, unit float64) string {
	return "Добавилось " + ruMoney(quantity) + " " + unitLabel(unit) + " Для закупа до суммы."
}

func unitLabel(unit float64) string {
	if unit > 1 {
		return "уп."
	}
	return "шт."
}

// AppendBudgetComment joins notes the way origin/main app.js budgetComment does.
func AppendBudgetComment(previous, note string) string {
	switch {
	case previous == "":
		return note
	case note == "":
		return previous
	default:
		return previous + "; " + note
	}
}

func ruMoney(n float64) string {
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	cents := int64(math.Round(n * 100))
	intPart := cents / 100
	frac := cents % 100
	s := groupRuInt(intPart)
	if frac == 0 {
		return sign + s
	}
	if frac%10 == 0 {
		return sign + s + "," + strconv.FormatInt(frac/10, 10)
	}
	return sign + s + "," + fmt.Sprintf("%02d", frac)
}

func groupRuInt(n int64) string {
	if n < 0 {
		n = -n
	}
	raw := strconv.FormatInt(n, 10)
	if len(raw) <= 3 {
		return raw
	}
	var b strings.Builder
	lead := len(raw) % 3
	if lead == 0 {
		lead = 3
	}
	b.WriteString(raw[:lead])
	for i := lead; i < len(raw); i += 3 {
		b.WriteRune('\u00a0')
		b.WriteString(raw[i : i+3])
	}
	return b.String()
}
