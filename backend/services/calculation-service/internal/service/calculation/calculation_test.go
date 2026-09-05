package calculation_test

import (
	"math"
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

func TestABCAndCoefficients(t *testing.T) {
	t.Parallel()
	if calculation.CategoryFromCumulative(10) != "A+" || calculation.CategoryFromCumulative(60) != "A" {
		t.Fatal("abc")
	}
	if calculation.CategoryFromCumulative(90) != "B" || calculation.CategoryFromCumulative(99) != "C" {
		t.Fatal("abc tail")
	}
	if calculation.CategoryCoefficient("C", "novacutan") != 1.5 || calculation.CategoryCoefficient("C", "angiopharm") != 1 {
		t.Fatal("coeff")
	}
	if calculation.DeliveryCoefficient(4) != 1 {
		t.Fatal("delivery")
	}
}

func TestRecommendNonNegativeAndStockSubtract(t *testing.T) {
	t.Parallel()
	svc := calculation.New()
	rows := svc.Recommend("angiopharm", []domain.OrderRow{{
		ID: "r1", Revenue: 100, Stock: 1000, InTransit: 1000,
		MonthlySales: []float64{1, 1, 1, 1},
	}})
	if rows[0].Recommended < 0 {
		t.Fatal(rows[0].Recommended)
	}
	if rows[0].Recommended != 0 {
		t.Fatalf("stock should cover: %v", rows[0].Recommended)
	}
}

func TestNoveltyAndStableAverage(t *testing.T) {
	t.Parallel()
	svc := calculation.New()
	novel := svc.RecommendWithWeeks("angiopharm", []domain.OrderRow{{
		ID: "n1", Revenue: 10, MonthlySales: []float64{0, 0, 0, 10},
	}}, 4)
	if novel[0].ABCCategory != "A+/New" && novel[0].ABCCategory != "C/New" && novel[0].TargetStock <= 0 {
		t.Fatalf("%+v", novel[0])
	}
	stable := svc.RecommendWithWeeks("angiopharm", []domain.OrderRow{{
		ID: "s1", Revenue: 10, MonthlySales: []float64{10, 10, 10, 10, 10, 10},
	}}, 4)
	if stable[0].TargetStock <= 0 {
		t.Fatalf("stable %+v", stable[0])
	}
}

func TestAdjustQuantityRules(t *testing.T) {
	t.Parallel()
	none := calculation.AdjustQuantity(5, "sothys", 0, false, "")
	if none.Inserted == nil || *none.Inserted != 5 || none.BoxAdjusted {
		t.Fatalf("none %#v", none)
	}
	christina := calculation.AdjustQuantity(11, "christina", 0, false, "")
	if christina.Inserted == nil || *christina.Inserted != 12 || !christina.BoxAdjusted {
		t.Fatalf("christina %#v", christina)
	}
	klapp := calculation.AdjustQuantity(10, "klapp", 0, false, "")
	if klapp.Inserted == nil || *klapp.Inserted != 9 || !klapp.BoxAdjusted {
		t.Fatalf("klapp %#v", klapp)
	}
	box := calculation.AdjustQuantity(11, "angiopharm", 0, false, "10")
	if box.Inserted == nil || *box.Inserted != 10 && *box.Inserted != 11 {
		t.Fatalf("box %#v", box)
	}
	small := calculation.AdjustQuantity(1.2, "angiopharm", 0, false, "6")
	if small.Inserted != nil {
		t.Fatalf("small %#v", small)
	}
	fact := calculation.AdjustQuantity(11, "christina", 3, true, "")
	if fact.Inserted == nil || *fact.Inserted != 3 || fact.AutoComment != "" {
		t.Fatalf("fact %#v", fact)
	}
}

func TestNorthPlanTyumenThenSupplier(t *testing.T) {
	t.Parallel()
	svc := calculation.New()
	rows := svc.NorthPlan("angiopharm",
		[]domain.CityNeed{{City: "surgut", Article: "A1", Qty: 10}, {City: "urengoy", Article: "A1", Qty: 10}},
		[]domain.OrderRow{{Article: "A1", Stock: 20, TargetStock: 5}},
	)
	if len(rows) != 1 {
		t.Fatalf("%+v", rows)
	}
	if math.Abs(rows[0].TransferQty-15) > 0.01 {
		t.Fatalf("from tyumen %v", rows[0].TransferQty)
	}
	if rows[0].SupplierQty != 5 {
		t.Fatalf("supplier %v", rows[0].SupplierQty)
	}
}

func TestNorthKLAPPAndNovacutan(t *testing.T) {
	t.Parallel()
	svc := calculation.New()
	klapp := svc.NorthPlan("klapp", []domain.CityNeed{{City: "surgut", Article: "A1", Qty: 10}}, nil)
	if klapp[0].SupplierQty != 9 {
		t.Fatalf("klapp %v", klapp[0].SupplierQty)
	}
	nova := svc.NorthPlan("novacutan", []domain.CityNeed{{City: "surgut", Article: "A1", Qty: 12}}, nil)
	if nova[0].SupplierQty != 100 {
		t.Fatalf("novacutan %v", nova[0].SupplierQty)
	}
}

func TestValidateManualEditsRequireComment(t *testing.T) {
	t.Parallel()
	svc := calculation.New()
	ok, blocking := svc.ValidateManualEdits(
		[]calculation.ManualEdit{{RowID: "r1", Qty: 3}},
		[]domain.OrderRow{{ID: "r1"}},
	)
	if ok || len(blocking) != 1 {
		t.Fatalf("ok=%v blocking=%v", ok, blocking)
	}
}
