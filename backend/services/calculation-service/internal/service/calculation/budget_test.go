package calculation_test

import (
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

func budgetRow(key, category string) domain.BudgetRow {
	return domain.BudgetRow{
		Key: key, Name: key, Category: category, Quantity: 10, Demand: 10,
		Delivery: 0.25, Price: 100, Unit: 1, Step: 1, Minimum: 1,
	}
}

func TestDiscountValue(t *testing.T) {
	t.Parallel()
	n, err := calculation.DiscountValue("30%")
	if err != nil || n != 30 {
		t.Fatalf("%v %v", n, err)
	}
	n, err = calculation.DiscountValue("30")
	if err != nil || n != 30 {
		t.Fatalf("%v %v", n, err)
	}
	_, err = calculation.DiscountValue("100%")
	if err == nil {
		t.Fatal("expected range error")
	}
}

func TestBudgetChangeComment(t *testing.T) {
	t.Parallel()
	before := func(b, q, u float64) domain.BudgetRow {
		return domain.BudgetRow{Before: b, HasBefore: true, Quantity: q, Unit: u}
	}
	cases := []struct {
		name string
		row  domain.BudgetRow
		want string
	}{
		{"down pieces", before(13, 4, 1), "Уменьшено на 9 шт. Для снижения заказа до указанной суммы."},
		{"down packs", before(3, 0, 5), "Уменьшено на 3 уп. Для снижения заказа до указанной суммы."},
		{"zero", before(0, 0, 1), ""},
		{"no before", domain.BudgetRow{Quantity: 4, Unit: 1}, ""},
		{"up", before(4, 13, 1), "Добавилось 9 шт. Для закупа до суммы."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := calculation.BudgetChangeComment(tc.row); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestAppendBudgetComment(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, previous, note, want string
	}{
		{"both", "Ручная правка", "Уменьшено", "Ручная правка; Уменьшено"},
		{"previous only", "Ручная правка", "", "Ручная правка"},
		{"note only", "", "Уменьшено", "Уменьшено"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := calculation.AppendBudgetComment(tc.previous, tc.note); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestPlanBudgetStagesAndPermissions(t *testing.T) {
	t.Parallel()
	p, err := calculation.PlanBudget([]domain.BudgetRow{budgetRow("a", "A"), budgetRow("c", "C")}, 2200, domain.BudgetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if p.Rows[0].Quantity != 10 || p.Rows[1].Quantity != 12 {
		t.Fatalf("stage C %+v", p.Rows)
	}

	manual := budgetRow("manual", "A")
	manual.Quantity = 15
	p, err = calculation.PlanBudget([]domain.BudgetRow{manual}, 1700, domain.BudgetOptions{})
	if err != nil || p.Rows[0].Before != 15 || p.Rows[0].Quantity != 17 {
		t.Fatalf("manual %+v err=%v", p.Rows, err)
	}

	locked := budgetRow("locked", "C")
	locked.Locked = true
	p, err = calculation.PlanBudget([]domain.BudgetRow{locked, budgetRow("free", "A")}, 2200, domain.BudgetOptions{})
	if err != nil || p.Rows[0].Quantity != 10 || p.Rows[1].Quantity != 12 {
		t.Fatalf("locked %+v err=%v", p.Rows, err)
	}

	none := budgetRow("none", "C")
	none.Demand = 0
	p, err = calculation.PlanBudget([]domain.BudgetRow{none, budgetRow("free", "A")}, 2200, domain.BudgetOptions{})
	if err != nil || p.Rows[0].Quantity != 10 {
		t.Fatalf("no demand %+v err=%v", p.Rows, err)
	}

	zero := budgetRow("zero", "C")
	zero.Quantity = 0
	excluded := budgetRow("manual-zero", "C")
	excluded.Quantity = 0
	excluded.Excluded = true
	p, err = calculation.PlanBudget([]domain.BudgetRow{zero, excluded}, 200, domain.BudgetOptions{})
	if err != nil || p.Rows[0].Quantity != 2 || p.Rows[1].Quantity != 0 {
		t.Fatalf("zero %+v err=%v", p.Rows, err)
	}

	cap := budgetRow("cap", "A")
	cap.Quantity = 60
	p, err = calculation.PlanBudget([]domain.BudgetRow{cap}, 6200, domain.BudgetOptions{})
	if err != nil || p.Reason != "overSix" {
		t.Fatalf("overSix %+v err=%v", p, err)
	}
	p, err = calculation.PlanBudget([]domain.BudgetRow{cap}, 6200, domain.BudgetOptions{AllowOverSix: true})
	if err != nil || p.Total != 6200 {
		t.Fatalf("allow overSix %+v err=%v", p, err)
	}

	floor := budgetRow("floor", "A")
	floor.Quantity = 15
	p, err = calculation.PlanBudget([]domain.BudgetRow{floor}, 1200, domain.BudgetOptions{})
	if err != nil || p.Reason != "belowOne" || p.Total != 1300 {
		t.Fatalf("belowOne %+v err=%v", p, err)
	}
	p, err = calculation.PlanBudget([]domain.BudgetRow{floor}, 1200, domain.BudgetOptions{AllowBelowOne: true})
	if err != nil || p.Total != 1200 {
		t.Fatalf("allow belowOne %+v err=%v", p, err)
	}

	c := budgetRow("c", "C")
	a := budgetRow("a", "A")
	p, err = calculation.PlanBudget([]domain.BudgetRow{c, a}, 1800, domain.BudgetOptions{AllowBelowOne: true})
	if err != nil || p.Rows[0].Quantity != 8 || p.Rows[1].Quantity != 10 {
		t.Fatalf("cut C first %+v err=%v", p.Rows, err)
	}
}

func TestPlanBudgetUnitsAndInputIsolation(t *testing.T) {
	t.Parallel()
	u, step, min := calculation.BudgetOrderRules("novacutan", "Novacutan SBIO, 2 мл", 0)
	if u != 1 || step != 10 || min != 100 {
		t.Fatalf("nova rules %v %v %v", u, step, min)
	}
	nova := budgetRow("nova", "A")
	nova.Unit, nova.Step, nova.Minimum = u, step, min
	nova.Quantity = 0
	nova.Demand = 100
	p, err := calculation.PlanBudget([]domain.BudgetRow{nova}, 9800, domain.BudgetOptions{})
	if err != nil || p.Total != 10000 || !p.Complete {
		t.Fatalf("nova 9800 %+v err=%v", p, err)
	}
	p, err = calculation.PlanBudget([]domain.BudgetRow{nova}, 9000, domain.BudgetOptions{})
	if err != nil || p.Complete {
		t.Fatalf("nova 9000 complete=%v err=%v", p.Complete, err)
	}

	mask := budgetRow("mask", "A")
	mask.Unit, mask.Step, mask.Minimum = calculation.BudgetOrderRules("novacutan", "EYE FILLER MASK NOVACUTAN", 0)
	if mask.Unit != 5 {
		t.Fatalf("mask unit %v", mask.Unit)
	}
	mask.Demand = 10
	cov, ok := calculation.Coverage(mask, 12)
	if !ok || cov != 6 {
		t.Fatalf("coverage %v %v", cov, ok)
	}

	for _, tc := range []struct {
		brand string
		box   float64
		step  float64
	}{
		{"christina", 0, 3},
		{"klapp", 0, 3},
		{"angiopharm", 6, 6},
		{"levissime", 4, 4},
		{"skin_synergy", 0, 1},
		{"sothys", 0, 1},
	} {
		t.Run(tc.brand, func(t *testing.T) {
			t.Parallel()
			_, step, min := calculation.BudgetOrderRules(tc.brand, "Товар", tc.box)
			if step != tc.step {
				t.Fatalf("step %v want %v", step, tc.step)
			}
			r := budgetRow(tc.brand, "A")
			r.Unit, r.Step, r.Minimum = 1, step, min
			r.Quantity = 0
			r.Demand = 100
			planned, err := calculation.PlanBudget([]domain.BudgetRow{r}, tc.step*100, domain.BudgetOptions{})
			if err != nil || planned.Rows[0].Quantity != tc.step || !planned.Complete {
				t.Fatalf("%+v err=%v", planned, err)
			}
		})
	}

	for i := 1; i < 80; i++ {
		source := []domain.BudgetRow{
			func() domain.BudgetRow {
				r := budgetRow("a", "A")
				r.Quantity = float64(i)
				r.Price = 37
				return r
			}(),
			func() domain.BudgetRow {
				r := budgetRow("b", "B")
				r.Quantity = 20
				r.Price = 59
				return r
			}(),
		}
		orig := source[0].Quantity
		result, err := calculation.PlanBudget(source, 3000, domain.BudgetOptions{AllowOverSix: true, AllowBelowOne: true})
		if err != nil {
			t.Fatal(err)
		}
		if result.Complete {
			okRange := result.Total >= 3000 && result.Total <= 3150 || result.Before >= 3000 && result.Total <= 3000
			if !okRange {
				t.Fatalf("i=%d total=%v before=%v", i, result.Total, result.Before)
			}
		}
		if source[0].Quantity != orig {
			t.Fatalf("mutated input at i=%d", i)
		}
	}
}

func TestPlanBudgetRejectsBadTarget(t *testing.T) {
	t.Parallel()
	_, err := calculation.PlanBudget(nil, -1, domain.BudgetOptions{})
	if err == nil || err.Error() != "введите неотрицательную сумму" {
		t.Fatalf("%v", err)
	}
}
