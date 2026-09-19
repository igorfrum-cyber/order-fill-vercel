package calculation_test

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

// proffFixture mirrors the `fixture` helper in origin/main
// scripts/test-christina-lines.mjs: an equal PROFF line whose required list is
// the full article set, priced after the main discount.
func proffFixture(quantities []float64, id string) []domain.BudgetRow {
	required := make([]string, len(quantities))
	for j := range quantities {
		required[j] = strconv.Itoa(j)
	}
	rows := make([]domain.BudgetRow, len(quantities))
	for i, q := range quantities {
		rows[i] = domain.BudgetRow{
			Key: fmt.Sprintf("%s:%d", id, i), Name: fmt.Sprintf("%s %d", id, i),
			Group: "proff", Category: "A+", Price: 100, Quantity: q,
			Demand: 10, Stock: 0, Transit: 0, Unit: 1, Step: 3, Minimum: 3, Delivery: 0.25,
			Line: &domain.ChristinaLine{ID: id, Name: id, Article: strconv.Itoa(i), Required: slices.Clone(required)},
		}
	}
	return rows
}

func muse(quantities ...float64) []domain.BudgetRow { return proffFixture(quantities, "MUSE") }

func propose(t *testing.T, rows []domain.BudgetRow, target float64) calculation.LineStep {
	t.Helper()
	return calculation.ProposeLineStep(rows, "MUSE", calculation.BudgetBaselines(rows), target)
}

func quantities(rows []domain.BudgetRow) []float64 {
	out := make([]float64, len(rows))
	for i, r := range rows {
		out[i] = r.Quantity
	}
	return out
}

func firstGroup(t *testing.T, rows []domain.BudgetRow) calculation.LineGroup {
	t.Helper()
	groups := calculation.ChristinaLineGroups(rows)
	if len(groups) == 0 {
		t.Fatal("expected at least one line group")
	}
	return groups[0]
}

func TestChristinaLineGroupsAndTotal(t *testing.T) {
	t.Parallel()
	rows := muse(18, 18, 18, 3)

	if got := calculation.ProcurementTotalCents(rows); got != 564000 {
		t.Fatalf("procurementTotalCents = %d, want 564000", got)
	}
	if got := firstGroup(t, rows).Sets; got != 3 {
		t.Fatalf("sets = %d, want 3", got)
	}

	home := slices.Clone(rows)
	for i := range home {
		home[i].Group = "home"
	}
	if got := calculation.ProcurementTotalCents(home); got != 570000 {
		t.Fatalf("home total = %d, want 570000 (no set discount)", got)
	}

	if got := firstGroup(t, rows[1:]).Sets; got != 0 {
		t.Fatalf("missing membership sets = %d, want 0", got)
	}

	dup := append(slices.Clone(rows), rows[0])
	if got := firstGroup(t, dup).Sets; got != 0 {
		t.Fatalf("duplicate membership sets = %d, want 0", got)
	}

	unsafe := slices.Clone(rows)
	unsafe[0].Unsafe = true
	if got := firstGroup(t, unsafe).Sets; got != 0 {
		t.Fatalf("unsafe sets = %d, want 0", got)
	}
}

func TestProposeLineStepAccepts(t *testing.T) {
	t.Parallel()
	rows := muse(18, 18, 18, 3)
	step := propose(t, rows, 10000)
	if !step.Accepted {
		t.Fatalf("step rejected: %s", step.Reason)
	}
	if got := quantities(step.Rows); !slices.Equal(got, []float64{18, 18, 18, 6}) {
		t.Fatalf("proposed quantities = %v, want [18 18 18 6]", got)
	}
	if step.AddedCents != 30000 || step.SavedCents != 6000 || step.TotalCents != 588000 {
		t.Fatalf("added/saved/total = %d/%d/%d, want 30000/6000/588000", step.AddedCents, step.SavedCents, step.TotalCents)
	}
	if got := quantities(rows); !slices.Equal(got, []float64{18, 18, 18, 3}) {
		t.Fatalf("input mutated: %v (proposal must be a pure preview)", got)
	}
}

func TestProposeLineStepRejections(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		rows   []domain.BudgetRow
		target float64
		reason string
	}{
		{"saving", muse(18, 18, 3, 3), 10000, "saving"},
		{"targetReached", muse(18, 18, 18, 3), 5500, "targetReached"},
		{"targetLimit", proffFixture([]float64{3, 3, 3, 0}, "MUSE"), 700, "targetLimit"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := propose(t, tc.rows, tc.target).Reason; got != tc.reason {
				t.Fatalf("reason = %q, want %q", got, tc.reason)
			}
		})
	}

	locked := muse(18, 18, 18, 3)
	for i := range locked {
		locked[i].Locked = true
	}
	if got := propose(t, locked, 10000).Reason; got != "locked" {
		t.Fatalf("locked reason = %q, want locked", got)
	}

	stocked := muse(18, 18, 18, 3)
	for i := range stocked {
		stocked[i].Stock = 100
	}
	if got := propose(t, stocked, 10000).Reason; got != "coverage" {
		t.Fatalf("coverage reason = %q, want coverage", got)
	}
}

func TestProposeLineStepFirstSetException(t *testing.T) {
	t.Parallel()
	// One missing article with no sales may still complete the first three sets.
	exception := proffFixture([]float64{18, 18, 18, 0}, "MUSE")
	exception[3].Demand = 0
	exception[3].Stock = 1000

	accepted := propose(t, exception, 5400)
	if !accepted.Accepted {
		t.Fatalf("first-set exception rejected: %s", accepted.Reason)
	}
	if got := propose(t, accepted.Rows, 10000).Reason; got != "coverage" {
		t.Fatalf("exception must not repeat: reason = %q, want coverage", got)
	}

	excluded := proffFixture([]float64{18, 18, 18, 0}, "MUSE")
	excluded[3].Demand = 0
	excluded[3].Stock = 1000
	excluded[3].Excluded = true
	if got := propose(t, excluded, 5400).Reason; got != "locked" {
		t.Fatalf("excluded reason = %q, want locked", got)
	}
}

func TestCompleteChristinaLines(t *testing.T) {
	t.Parallel()
	rows := muse(18, 18, 18, 3)
	baselines := calculation.BudgetBaselines(rows)
	res, err := calculation.CompleteChristinaLines(rows, 10000, baselines)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Steps) == 0 {
		t.Fatal("expected completion steps")
	}
	if got, ceiling := firstGroup(t, res.Rows).NetCents, int64(math.Floor(float64(baselines["MUSE"])*1.3+1e-8)); got > ceiling {
		t.Fatalf("net %d exceeds 130%% baseline ceiling %d", got, ceiling)
	}
	if !slices.ContainsFunc(res.Rejected, func(r calculation.Rejection) bool { return r.Reason == "lineGrowth" }) {
		t.Fatal("expected a lineGrowth rejection (baseline is not reset per step)")
	}
	if got := quantities(rows); !slices.Equal(got, []float64{18, 18, 18, 3}) {
		t.Fatalf("input mutated: %v", got)
	}
}

// TestProcurementTotalWithMainDiscount mirrors the priceOrderRows assertion:
// club price 100 at 30% main discount → 70, then the 5% set discount on three
// of each yields 394800 kopecks.
func TestProcurementTotalWithMainDiscount(t *testing.T) {
	t.Parallel()
	priced := muse(18, 18, 18, 3)
	for i := range priced {
		priced[i].Price = 70 // 100 club * (1 - 30/100)
	}
	if got := calculation.ProcurementTotalCents(priced); got != 394800 {
		t.Fatalf("procurementTotalCents = %d, want 394800", got)
	}
}

func TestPlanBudgetProffCompletesUpward(t *testing.T) {
	t.Parallel()
	rows := muse(18, 18, 18, 3)
	plan, err := calculation.PlanBudget(rows, 6100, domain.BudgetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Complete {
		t.Fatalf("plan not complete: reason=%q total=%v", plan.Reason, plan.Total)
	}
	if plan.Total < 6100 || plan.Total > 6405 {
		t.Fatalf("total = %v, want within [6100, 6405]", plan.Total)
	}
	if got := beforeQuantities(plan.Rows); !slices.Equal(got, []float64{18, 18, 18, 3}) {
		t.Fatalf("before = %v, want [18 18 18 3]", got)
	}
	if cents := calculation.ProcurementTotalCents(plan.Rows); int64(math.Round(plan.Total*100)) != cents {
		t.Fatalf("total*100 = %d, want %d", int64(math.Round(plan.Total*100)), cents)
	}
	if got := quantities(rows); !slices.Equal(got, []float64{18, 18, 18, 3}) {
		t.Fatalf("input mutated: %v", got)
	}
}

func TestPlanBudgetProffReduction(t *testing.T) {
	t.Parallel()
	down, err := calculation.PlanBudget(muse(18, 18, 18, 18), 5000, domain.BudgetOptions{AllowBelowOne: true})
	if err != nil {
		t.Fatal(err)
	}
	if !down.Complete {
		t.Fatalf("reduction not complete: reason=%q", down.Reason)
	}
	if down.Total > 5000 {
		t.Fatalf("total = %v, want <= 5000", down.Total)
	}
	// Reduction revalues every complete set, so the reported total is the
	// discounted procurement total.
	if cents := calculation.ProcurementTotalCents(down.Rows); int64(math.Round(down.Total*100)) != cents {
		t.Fatalf("round(total*100) = %d, want %d", int64(math.Round(down.Total*100)), cents)
	}
}

func TestPlanBudgetProffLockedStaysPut(t *testing.T) {
	t.Parallel()
	locked := muse(18, 18, 18, 3)
	for i := range locked {
		locked[i].Locked = true
	}
	plan, err := calculation.PlanBudget(locked, 1000, domain.BudgetOptions{AllowBelowOne: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Complete {
		t.Fatal("expected incomplete plan for fully locked line")
	}
	if got := quantities(plan.Rows); !slices.Equal(got, []float64{18, 18, 18, 3}) {
		t.Fatalf("locked rows changed: %v", got)
	}
}

func TestPlanBudgetCompareModeMuseMatch(t *testing.T) {
	t.Parallel()
	rows := muse(18, 18, 18, 3)
	std, err := calculation.PlanBudget(rows, 6100, domain.BudgetOptions{ChristinaProffMode: domain.ChristinaProffStandard})
	if err != nil {
		t.Fatal(err)
	}
	cmpPlan, err := calculation.PlanBudget(rows, 6100, domain.BudgetOptions{ChristinaProffMode: domain.ChristinaProffCompare})
	if err != nil {
		t.Fatal(err)
	}
	if cmpPlan.ChristinaProffMode != domain.ChristinaProffCompare {
		t.Fatalf("mode=%q", cmpPlan.ChristinaProffMode)
	}
	if !cmpPlan.CompareMatch || len(cmpPlan.Mismatches) != 0 {
		t.Fatalf("match=%v mismatches=%+v", cmpPlan.CompareMatch, cmpPlan.Mismatches)
	}
	if got, want := quantities(cmpPlan.Rows), quantities(std.Rows); !slices.Equal(got, want) {
		t.Fatalf("applied qty=%v, want standard %v", got, want)
	}
	if got, want := quantities(cmpPlan.FastRows), quantities(cmpPlan.Rows); !slices.Equal(got, want) {
		t.Fatalf("fast qty=%v, want applied %v", got, want)
	}
	if cmpPlan.FastTotal != cmpPlan.Total || cmpPlan.FastComplete != cmpPlan.Complete {
		t.Fatalf("fast total/complete=%v/%v applied=%v/%v", cmpPlan.FastTotal, cmpPlan.FastComplete, cmpPlan.Total, cmpPlan.Complete)
	}
	if cmpPlan.CompareStandardMs < 0 || cmpPlan.CompareFastMs < 0 {
		t.Fatalf("timings %d/%d", cmpPlan.CompareStandardMs, cmpPlan.CompareFastMs)
	}
	if got := quantities(rows); !slices.Equal(got, []float64{18, 18, 18, 3}) {
		t.Fatalf("input mutated: %v", got)
	}
}

func TestPlanBudgetCompareModeHomeMatch(t *testing.T) {
	t.Parallel()
	home := muse(18, 18, 18, 3)
	for i := range home {
		home[i].Group = "home"
	}
	plan, err := calculation.PlanBudget(home, 6100, domain.BudgetOptions{ChristinaProffMode: domain.ChristinaProffCompare})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.CompareMatch || len(plan.Mismatches) != 0 {
		t.Fatalf("home compare mismatches=%+v", plan.Mismatches)
	}
	if got, want := quantities(plan.FastRows), quantities(plan.Rows); !slices.Equal(got, want) {
		t.Fatalf("home fast qty=%v applied=%v", got, want)
	}
}

func TestPlanBudgetUnknownModeIsStandard(t *testing.T) {
	t.Parallel()
	rows := muse(18, 18, 18, 3)
	std, err := calculation.PlanBudget(rows, 6100, domain.BudgetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := calculation.PlanBudget(rows, 6100, domain.BudgetOptions{ChristinaProffMode: "UNSPECIFIED"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ChristinaProffMode != domain.ChristinaProffStandard {
		t.Fatalf("mode=%q", got.ChristinaProffMode)
	}
	if !slices.Equal(quantities(got.Rows), quantities(std.Rows)) {
		t.Fatalf("unknown mode qty=%v want %v", quantities(got.Rows), quantities(std.Rows))
	}
}

// TestPlanReportBudgetNorthProffSetDiscount mirrors the North flow: raw rows with
// group "proff" and line metadata must earn the CHRISTINA set discount, so the
// reported "before" reflects the discounted procurement total, not the gross.
func TestPlanReportBudgetNorthProffSetDiscount(t *testing.T) {
	t.Parallel()
	rows := make([]domain.BudgetInputRow, 4)
	for i := range rows {
		rows[i] = domain.BudgetInputRow{
			Key: fmt.Sprintf("proff:%d", i), Name: fmt.Sprintf("MUSE %d", i), Category: "A+",
			Quantity: 18, BasePrice: 100, Demand: 10, Outbound: 5, Group: "proff",
			Line: &domain.ChristinaLine{ID: "MUSE", Name: "MUSE", Article: strconv.Itoa(i), Required: []string{"0", "1", "2", "3"}},
		}
	}
	// gross = 100*72 = 7200; set discount at 18 sets = 4*(100*18*0.05) = 360 → 6840.
	plan, err := calculation.PlanReportBudget(domain.BudgetRequest{Brand: "christina", DeliveryWeeks: 1, Target: 6840, Rows: rows})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Before != 6840 {
		t.Fatalf("before = %v, want 6840 (set discount applied on North PROFF)", plan.Before)
	}
}

func beforeQuantities(rows []domain.BudgetRow) []float64 {
	out := make([]float64, len(rows))
	for i, r := range rows {
		out[i] = r.Before
	}
	return out
}
