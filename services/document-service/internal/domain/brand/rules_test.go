package brand

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

func TestKeyFromNomenclatureGroupMaps1CFilterNames(t *testing.T) {
	cases := map[string]string{
		"Ангиофарм ":   "angiopharm",
		"ANGIOPHARM":   "angiopharm",
		"Кристина":     "christina",
		"CHRISTINA":    "christina",
		"KLAPP":        "klapp",
		"SKIN SYNERGY": "skin_synergy",
		"Skin Synergy": "skin_synergy",
		"LeviSsime":    "levissime",
		"SOTHYS":       "sothys",
		"NOVACUTAN":    "novacutan",
	}
	for group, want := range cases {
		got, ok := KeyFromNomenclatureGroup(group)
		if !ok || got != want {
			t.Fatalf("group %q: got (%q, %v), want %q", group, got, ok, want)
		}
	}
	if _, ok := KeyFromNomenclatureGroup("Неизвестный бренд"); ok {
		t.Fatal("unknown group must not map to a brand")
	}
}
