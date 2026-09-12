package calculation_test

import (
	"testing"

	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

func TestNovacutanNameRules(t *testing.T) {
	t.Parallel()
	if calculation.NovacutanSupplierUnitSize("EYE FILLER MASK NOVACUTAN") != 5 {
		t.Fatal("mask unit")
	}
	if calculation.NovacutanMinimumQuantity("EYE FILLER MASK NOVACUTAN") != 10 {
		t.Fatal("mask min")
	}
	if calculation.NovacutanMinimumQuantity("Novacutan SBIO, 2 мл") != 100 {
		t.Fatal("sbio min")
	}
	if calculation.NovacutanMinimumQuantity("NOVACUTAN FBIO Light") != 50 {
		t.Fatal("fbio min")
	}
	if calculation.NovacutanMatchKey("EYE FILLER MASK NOVACUTAN") != "mask-eye" {
		t.Fatal("mask key")
	}
	if calculation.NovacutanPositionKey("Novacutan SBIO, 2 мл") != "novacutan:sbio" {
		t.Fatalf("%s", calculation.NovacutanPositionKey("Novacutan SBIO, 2 мл"))
	}
}
