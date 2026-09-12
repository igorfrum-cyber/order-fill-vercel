package calculation

import (
	"cmp"
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

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

func collectBudgetCands(rows []domain.BudgetRow, up bool) []budgetCand {
	dir := 1.0
	if !up {
		dir = -1
	}
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
			cost: budgetCents(r.Price*q) - budgetCents(r.Price*r.Quantity),
		})
	}
	return out
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

// PlanBudget matches origin/main planBudget. It copies input and never mutates it.
func PlanBudget(input []domain.BudgetRow, target float64, opts domain.BudgetOptions) (domain.BudgetPlan, error) {
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
	total := func() int64 {
		var sum int64
		for _, r := range rows {
			sum += budgetCents(r.Price * r.Quantity)
		}
		return sum
	}
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
		candidates := collectBudgetCands(rows, up)
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
		for _, i := range idx {
			r := &rows[i]
			for r.Quantity < r.Before {
				q := nextQuantity(*r, r.Quantity, 1)
				cost := budgetCents(r.Price*q) - budgetCents(r.Price*r.Quantity)
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

// BudgetChangeComment matches origin/main budgetChangeComment.
func BudgetChangeComment(row domain.BudgetRow) string {
	if !row.HasBefore {
		return ""
	}
	delta := row.Quantity - row.Before
	if delta > 0 {
		return additionComment(delta, row.Unit)
	}
	if delta < 0 {
		return "Уменьшено на " + ruMoney(-delta) + " " + unitLabel(row.Unit) + " Для снижения заказа до указанной суммы."
	}
	return ""
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
