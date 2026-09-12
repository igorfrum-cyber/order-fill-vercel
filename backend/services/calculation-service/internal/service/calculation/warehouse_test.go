package calculation_test

import (
	"testing"

	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

func TestWarehouseTransferQuantity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name                string
		office, warehouse   float64
		wantQty, wantTarget float64
	}{
		{"top up 15", 10, 90, 15, 25},
		{"already at 25%", 35, 65, 0, 25},
		{"empty warehouse", 10, 0, 0, 3},
		{"empty office", 0, 40, 10, 10},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := calculation.WarehouseOfficeTarget(tc.office, tc.warehouse); got != tc.wantTarget {
				t.Fatalf("target %v want %v", got, tc.wantTarget)
			}
			if got := calculation.WarehouseTransferQuantity(tc.office, tc.warehouse); got != tc.wantQty {
				t.Fatalf("qty %v want %v", got, tc.wantQty)
			}
		})
	}
}

func TestNorthTyumenFreeStock(t *testing.T) {
	t.Parallel()
	legacy := calculation.NorthTyumenFreeStock(20, 0, 0, 5, 0, 0, false)
	if legacy != 15 {
		t.Fatalf("legacy %v", legacy)
	}
	capped := calculation.NorthTyumenFreeStock(20, 0, 0, 5, 10, 0, true)
	if capped != 10 {
		t.Fatalf("warehouse cap %v", capped)
	}
	planned := calculation.NorthTyumenFreeStock(20, 0, 8, 5, 10, 0, true)
	if planned != 18 {
		t.Fatalf("planned adds to both sides %v", planned)
	}
}

func TestNorthPlanWarehouseCap(t *testing.T) {
	t.Parallel()
	svc := calculation.New()
	rows := svc.NorthPlan("angiopharm",
		[]domain.CityNeed{{City: "surgut", Article: "A1", Qty: 10}, {City: "urengoy", Article: "A1", Qty: 10}},
		[]domain.OrderRow{{Article: "A1", Stock: 20, TargetStock: 5, WarehouseStock: 10, HasWarehouseStock: true}},
	)
	if len(rows) != 1 {
		t.Fatalf("%+v", rows)
	}
	if rows[0].TransferQty != 10 {
		t.Fatalf("transfer %v", rows[0].TransferQty)
	}
	if rows[0].SupplierQty != 10 {
		t.Fatalf("supplier %v", rows[0].SupplierQty)
	}
}
