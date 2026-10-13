package calculation_test

import (
	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
	"testing"
)

func TestNorthPlanUsesSelectedBlankMinimum(t *testing.T) {
	t.Parallel()
	got := calculation.New().NorthPlan("novacutan", []domain.CityNeed{{City: "surgut", Article: "novacutan:sbio", Name: "SBIO NOVACUTAN", Qty: 1, Minimum: 25}}, nil)
	if len(got) != 1 || got[0].SupplierQty != 30 || got[0].NovacutanMin != 25 {
		t.Fatalf("got %+v", got)
	}
}
