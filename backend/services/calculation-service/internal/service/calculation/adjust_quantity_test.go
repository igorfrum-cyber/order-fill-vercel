package calculation

import "testing"

func TestSupplierUnitConversionMatchesMain(t *testing.T) {
	t.Parallel()
	got := ConvertSupplierUnits(AdjustedQuantity{Rounded: 11, Inserted: new(11.0)}, 5)
	if got.Rounded != 3 || got.Inserted == nil || *got.Inserted != 3 || got.AutoComment != "коробки по 5 шт." {
		t.Fatalf("got %+v", got)
	}
}
