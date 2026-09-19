package calculation

import (
	"fmt"
	"slices"
	"strconv"
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
)

func oracleMuse(quantities ...float64) []domain.BudgetRow {
	required := make([]string, len(quantities))
	for j := range quantities {
		required[j] = strconv.Itoa(j)
	}
	rows := make([]domain.BudgetRow, len(quantities))
	for i, q := range quantities {
		rows[i] = domain.BudgetRow{
			Key: fmt.Sprintf("MUSE:%d", i), Name: fmt.Sprintf("MUSE %d", i),
			Group: "proff", Category: "A+", Price: 100, Quantity: q,
			Demand: 10, Stock: 0, Transit: 0, Unit: 1, Step: 3, Minimum: 3, Delivery: 0.25,
			Line: &domain.ChristinaLine{ID: "MUSE", Name: "MUSE", Article: strconv.Itoa(i), Required: slices.Clone(required)},
		}
	}
	return rows
}

func TestChangeCostFastMatchesFull(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rows []domain.BudgetRow
		i    int
		q    float64
	}{
		{"first-set 0to3", oracleMuse(18, 18, 18, 0), 3, 3},
		{"min 3to6", oracleMuse(18, 18, 18, 3), 3, 6},
		{"sibling 18to21", oracleMuse(18, 18, 18, 3), 0, 21},
		{"down 18to15", oracleMuse(18, 18, 18, 3), 0, 15},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			orig := slices.Clone(tc.rows)
			want := changeCost(tc.rows, tc.i, tc.q, false, nil)
			got := changeCost(tc.rows, tc.i, tc.q, true, nil)
			if want != got {
				t.Fatalf("fast=%d full=%d", got, want)
			}
			if !slices.EqualFunc(orig, tc.rows, func(a, b domain.BudgetRow) bool { return a.Quantity == b.Quantity }) {
				t.Fatal("changeCost mutated input")
			}
		})
	}

	rows := oracleMuse(18, 18, 18, 3)
	for i := range rows {
		for _, dir := range []float64{1, -1} {
			q := nextQuantity(rows[i], rows[i].Quantity, dir)
			if q == rows[i].Quantity {
				continue
			}
			want := changeCost(rows, i, q, false, nil)
			got := changeCost(rows, i, q, true, nil)
			if want != got {
				t.Fatalf("row %d q %v: fast=%d full=%d", i, q, got, want)
			}
		}
	}

	incomplete := oracleMuse(18, 18, 18, 3)[1:]
	want := changeCost(incomplete, 0, 21, false, nil)
	got := changeCost(incomplete, 0, 21, true, nil)
	if want != got {
		t.Fatalf("incomplete fast=%d full=%d", got, want)
	}

	home := oracleMuse(18, 18, 18, 3)
	for i := range home {
		home[i].Group = "home"
	}
	want = changeCost(home, 3, 6, false, nil)
	got = changeCost(home, 3, 6, true, nil)
	if want != got {
		t.Fatalf("home fast=%d full=%d", got, want)
	}
}

func TestProposeLineStepFastTotalsMatchFull(t *testing.T) {
	t.Parallel()
	rows := oracleMuse(18, 18, 18, 3)
	full := ProposeLineStep(rows, "MUSE", BudgetBaselines(rows), 10000)
	fast := proposeLineStep(rows, "MUSE", BudgetBaselines(rows), 10000, true, ProcurementTotalCents(rows), true)
	if full.AddedCents != fast.AddedCents || full.SavedCents != fast.SavedCents || full.TotalCents != fast.TotalCents {
		t.Fatalf("full %d/%d/%d fast %d/%d/%d", full.AddedCents, full.SavedCents, full.TotalCents, fast.AddedCents, fast.SavedCents, fast.TotalCents)
	}

	first := oracleMuse(18, 18, 18, 0)
	full = ProposeLineStep(first, "MUSE", BudgetBaselines(first), 10000)
	fast = proposeLineStep(first, "MUSE", BudgetBaselines(first), 10000, true, ProcurementTotalCents(first), true)
	if full.AddedCents != fast.AddedCents || full.SavedCents != fast.SavedCents || full.TotalCents != fast.TotalCents {
		t.Fatalf("first-set full %d/%d/%d fast %d/%d/%d", full.AddedCents, full.SavedCents, full.TotalCents, fast.AddedCents, fast.SavedCents, fast.TotalCents)
	}
}

func TestAttachCompareKeepsStandardWhenFastDiverges(t *testing.T) {
	t.Parallel()
	std, err := PlanBudget(oracleMuse(18, 18, 18, 3), 6100, domain.BudgetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	fast := std
	fast.Rows = slices.Clone(std.Rows)
	fast.Rows[3].Quantity += 3
	fast.Total += 24
	fast.Complete = false
	out := attachCompareResult(std, fast, 4, 1)
	if out.ChristinaProffMode != domain.ChristinaProffCompare {
		t.Fatalf("mode=%q", out.ChristinaProffMode)
	}
	if out.CompareMatch || len(out.Mismatches) == 0 {
		t.Fatalf("expected mismatches, got %+v", out.Mismatches)
	}
	if out.Rows[3].Quantity != std.Rows[3].Quantity {
		t.Fatalf("applied qty=%v, want standard %v", out.Rows[3].Quantity, std.Rows[3].Quantity)
	}
	if out.FastRows[3].Quantity != fast.Rows[3].Quantity {
		t.Fatalf("fast qty=%v", out.FastRows[3].Quantity)
	}
	if out.CompareStandardMs != 4 || out.CompareFastMs != 1 {
		t.Fatalf("timings %d/%d", out.CompareStandardMs, out.CompareFastMs)
	}
}
