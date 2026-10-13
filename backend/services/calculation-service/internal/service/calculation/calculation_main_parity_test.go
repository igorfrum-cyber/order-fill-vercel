package calculation_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

func TestCalculationMatchesPinnedMainOracle(t *testing.T) {
	t.Parallel()
	data, err := os.ReadFile("testdata/calculation_main_1e2baa1.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Recommendations []struct {
			Name, Brand    string
			Weeks          float64
			Rows, Expected []domain.OrderRow
		}
		Adjustments []struct {
			Brand, Box string
			Quantity   float64
			Expected   calculation.AdjustedQuantity
		}
		North []struct {
			Name, Brand string
			Cities      map[string]float64
			Stock       domain.OrderRow
			Expected    domain.PlanRow
		}
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if len(corpus.Recommendations) != 56 || len(corpus.Adjustments) != 308 || len(corpus.North) != 73 {
		t.Fatal("incomplete calculation oracle corpus")
	}
	for _, tc := range corpus.Recommendations {
		t.Run("recommendations/"+tc.Name, func(t *testing.T) {
			t.Parallel()
			got := calculation.New().RecommendWithWeeks(tc.Brand, tc.Rows, tc.Weeks)
			for i, row := range got {
				want := tc.Expected[i]
				if row.Recommended != want.Recommended || row.TargetStock != want.TargetStock || row.ABCCategory != want.ABCCategory || row.RevenuePercent != want.RevenuePercent || row.CumulativePercent != want.CumulativePercent || row.AverageMonthly != want.AverageMonthly || row.TotalQuantity != want.TotalQuantity {
					t.Fatalf("row %d: got %+v; main %+v", i, row, want)
				}
			}
		})
	}
	for i, tc := range corpus.Adjustments {
		t.Run(fmt.Sprintf("adjustments/%s-%d", tc.Brand, i), func(t *testing.T) {
			t.Parallel()
			got := calculation.AdjustQuantity(tc.Quantity, tc.Brand, 0, false, tc.Box)
			want := tc.Expected
			sameInserted := got.Inserted == nil && want.Inserted == nil || got.Inserted != nil && want.Inserted != nil && *got.Inserted == *want.Inserted
			if got.Rounded != want.Rounded || !sameInserted || got.AutoComment != want.AutoComment || got.BoxAdjusted != want.BoxAdjusted {
				t.Fatalf("quantity=%v box=%q: got %+v; main %+v", tc.Quantity, tc.Box, got, want)
			}
		})
	}
	for _, tc := range corpus.North {
		t.Run("north/"+tc.Name, func(t *testing.T) {
			t.Parallel()
			needs := make([]domain.CityNeed, 0, len(tc.Cities))
			for city, qty := range tc.Cities {
				needs = append(needs, domain.CityNeed{City: city, Article: "X", Qty: qty})
			}
			got := calculation.New().NorthPlan(tc.Brand, needs, []domain.OrderRow{tc.Stock})
			if len(got) != 1 || got[0].TransferQty != tc.Expected.TransferQty || got[0].SupplierQty != tc.Expected.SupplierQty || got[0].TyumenQty != tc.Expected.TyumenQty {
				t.Fatalf("got %+v; main %+v; cities=%v stock=%+v", got, tc.Expected, tc.Cities, tc.Stock)
			}
		})
	}
}
