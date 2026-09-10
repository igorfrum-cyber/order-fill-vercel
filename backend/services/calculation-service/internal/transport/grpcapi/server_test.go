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
		Rows: []*calculationv1.OrderRow{{Id: "a", Revenue: 80, MonthlySales: []float64{2, 2, 2, 2, 2, 2}}},
	})
	if err != nil || len(abc.GetRows()) != 1 || abc.GetRows()[0].GetRevenuePercent() != 100 {
		t.Fatalf("abc %+v err=%v", abc, err)
	}
	urengoy, err := s.CalculateOrderRecommendations(t.Context(), &calculationv1.CalculateOrderRecommendationsRequest{
		Brand: "angiopharm", CityRule: "urengoy", DeliveryWeeks: 4,
		Rows: []*calculationv1.OrderRow{{Id: "r1", AbcCategory: "C", MonthlySales: []float64{10, 4}}},
	})
	if err != nil || urengoy.GetRows()[0].GetRecommendedQty() != 30 {
		t.Fatalf("urengoy %+v err=%v", urengoy, err)
	}
}
