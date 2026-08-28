package brands

import "testing"

func TestCategoryCoefficientUsesBrandSpecificNovacutanRules(t *testing.T) {
	if got := CategoryCoefficient("C", Rule("novacutan")); got != 1.5 {
		t.Fatalf("expected novacutan C coefficient 1.5, got %v", got)
	}
	if got := CategoryCoefficient("C", Rule("angiopharm")); got != 1 {
		t.Fatalf("expected default C coefficient 1, got %v", got)
	}
}

func TestAdjustQuantityForBrandAppliesBrandRounding(t *testing.T) {
	christina := AdjustQuantityForBrand(11, "christina", "")
	if christina.Inserted == nil || *christina.Inserted != 12 || !christina.BoxAdjusted {
		t.Fatalf("expected Christina 11 to adjust to 12, got %#v", christina)
	}

	klapp := AdjustQuantityForBrand(10, "klapp", "")
	if klapp.Inserted == nil || *klapp.Inserted != 9 || !klapp.BoxAdjusted {
		t.Fatalf("expected KLAPP 10 to nearest multiple 9, got %#v", klapp)
	}
}
