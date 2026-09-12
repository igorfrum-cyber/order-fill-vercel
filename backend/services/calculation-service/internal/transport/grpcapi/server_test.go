package grpcapi

import (
	"testing"

	calculationv1 "order-fill/backend/proto/gen/go/orderfill/calculation/v1"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

func TestCalculateAdjustedQuantityUsesBoxSize(t *testing.T) {
	t.Parallel()
	s := NewServer(calculation.New())
	resp, err := s.CalculateAdjustedQuantity(t.Context(), &calculationv1.CalculateAdjustedQuantityRequest{
		Brand: "angiopharm", RecommendedQty: 10, BoxSize: "3", Adjustment: "box", AdjustmentComment: "до коробки",
	})
	if err != nil {
		t.Fatalf("Adjust: %v", err)
	}
	if !resp.GetInserted() || resp.GetQty() != 10 {
		t.Fatalf("%+v", resp)
	}
}

func TestCalculateAdjustedQuantityLeavesSmallOrderEmpty(t *testing.T) {
	t.Parallel()
	s := NewServer(calculation.New())
	resp, err := s.CalculateAdjustedQuantity(t.Context(), &calculationv1.CalculateAdjustedQuantityRequest{
		Brand: "angiopharm", RecommendedQty: 1.2, BoxSize: "3", Adjustment: "box",
	})
	if err != nil {
		t.Fatalf("Adjust: %v", err)
	}
	if resp.GetInserted() {
		t.Fatalf("small order must not insert, got %+v", resp)
	}
}

func TestCalculateAdjustedQuantityUsesPolicyNotBrandTable(t *testing.T) {
	t.Parallel()
	s := NewServer(calculation.New())
	resp, err := s.CalculateAdjustedQuantity(t.Context(), &calculationv1.CalculateAdjustedQuantityRequest{
		Brand: "angiopharm", RecommendedQty: 5, Adjustment: "none",
	})
	if err != nil {
		t.Fatalf("Adjust: %v", err)
	}
	if !resp.GetInserted() || resp.GetQty() != 5 || resp.GetBoxAdjusted() {
		t.Fatalf("policy none should insert 5, got %+v", resp)
	}
}

func TestCalculateOrderRecommendationsHonoursDeliveryWeeksAndUrengoy(t *testing.T) {
	t.Parallel()
	s := NewServer(calculation.New())
	abc, err := s.CalculateOrderRecommendations(t.Context(), &calculationv1.CalculateOrderRecommendationsRequest{
		Brand: "angiopharm", DeliveryWeeks: 4,
		Rows: []*calculationv1.OrderRow{{
			Id: "a", Revenue: 80, MonthlySales: []float64{2, 2, 2, 2, 2, 2},
			WarehouseStock: 7, WarehouseTransit: 2, HasWarehouseStock: true,
			BoxSize: 6, HasBoxSize: true,
		}},
	})
	if err != nil || len(abc.GetRows()) != 1 || abc.GetRows()[0].GetRevenuePercent() != 100 {
		t.Fatalf("abc %+v err=%v", abc, err)
	}
	row := abc.GetRows()[0]
	if row.GetWarehouseStock() != 7 || row.GetWarehouseTransit() != 2 || !row.GetHasWarehouseStock() || row.GetBoxSize() != 6 || !row.GetHasBoxSize() {
		t.Fatalf("warehouse fields were lost: %+v", row)
	}
	urengoy, err := s.CalculateOrderRecommendations(t.Context(), &calculationv1.CalculateOrderRecommendationsRequest{
		Brand: "angiopharm", CityRule: "urengoy", DeliveryWeeks: 4,
		Rows: []*calculationv1.OrderRow{{Id: "r1", AbcCategory: "C", MonthlySales: []float64{10, 4}}},
	})
	if err != nil || urengoy.GetRows()[0].GetRecommendedQty() != 30 {
		t.Fatalf("urengoy %+v err=%v", urengoy, err)
	}
}

func TestRecalculateNorthRowPreservesPlanningFields(t *testing.T) {
	t.Parallel()
	s := NewServer(calculation.New())
	resp, err := s.RecalculateNorthRow(t.Context(), &calculationv1.RecalculateNorthRowRequest{
		Brand: "angiopharm",
		Row: &calculationv1.NorthPlanRow{
			Article: "A1", Name: "Товар", Variant: "retail",
			TyumenStock: 20, TyumenTarget: 5, UnitSize: 1,
			BoxSize: 6, HasBoxSize: true,
			WarehouseStock: 10, WarehouseTransit: 2, HasWarehouseStock: true,
		},
		EditedQty: 20,
	})
	if err != nil {
		t.Fatal(err)
	}
	row := resp.GetRow()
	if row.GetArticle() != "A1" || row.GetVariant() != "retail" || row.GetBoxSize() != 6 || !row.GetHasBoxSize() {
		t.Fatalf("supplier fields were lost: %+v", row)
	}
	if row.GetWarehouseStock() != 10 || row.GetWarehouseTransit() != 2 || !row.GetHasWarehouseStock() {
		t.Fatalf("warehouse fields were lost: %+v", row)
	}
	if row.GetTransferQty() != 12 || row.GetSupplierQty() != 12 {
		t.Fatalf("unexpected recalculation: %+v", row)
	}
}

func TestPlanBudgetAndWarehouseTransferRPC(t *testing.T) {
	t.Parallel()
	s := NewServer(calculation.New())
	budget, err := s.PlanBudget(t.Context(), &calculationv1.PlanBudgetRequest{
		Target: 2200,
		Rows: []*calculationv1.BudgetRow{
			{Key: "a", Name: "a", Category: "A", Quantity: 10, Price: 100, Demand: 10, Delivery: 0.25, Unit: 1, Step: 1, Minimum: 1},
			{Key: "c", Name: "c", Category: "C", Quantity: 10, Price: 100, Demand: 10, Delivery: 0.25, Unit: 1, Step: 1, Minimum: 1},
		},
	})
	if err != nil || budget.GetRows()[0].GetQuantity() != 10 || budget.GetRows()[1].GetQuantity() != 12 {
		t.Fatalf("%+v err=%v", budget, err)
	}
	_, err = s.PlanBudget(t.Context(), &calculationv1.PlanBudgetRequest{Target: -1})
	if err == nil {
		t.Fatal("expected invalid target")
	}
	xfer, err := s.CalculateWarehouseTransfer(t.Context(), &calculationv1.CalculateWarehouseTransferRequest{
		OfficeStock: 10, WarehouseStock: 90,
	})
	if err != nil || xfer.GetQuantity() != 15 || xfer.GetTarget() != 25 {
		t.Fatalf("%+v err=%v", xfer, err)
	}
}
