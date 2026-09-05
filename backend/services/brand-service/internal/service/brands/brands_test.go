package brands_test

import (
	"slices"
	"testing"

	"order-fill/backend/services/brand-service/internal/domain"
	"order-fill/backend/services/brand-service/internal/service/brands"
	"order-fill/backend/services/brand-service/internal/storage/static"
)

func TestAllCurrentBrandsHavePolicies(t *testing.T) {
	t.Parallel()
	want := []string{"angiopharm", "christina", "klapp", "levissime", "novacutan", "skin_synergy", "sothys"}
	got := static.List()
	if !slices.Equal(got, want) {
		t.Fatalf("got %v", got)
	}
	for _, key := range want {
		if static.Policy(key).Key != key {
			t.Fatalf("missing policy %s", key)
		}
	}
}

func TestChristinaMultipleAndKlappNearest(t *testing.T) {
	t.Parallel()
	if static.Policy("christina").Multiple != 3 || static.Policy("christina").Adjustment != domain.AdjustmentMultiple {
		t.Fatal("christina")
	}
	if static.Policy("klapp").Adjustment != domain.AdjustmentNearestMultiple {
		t.Fatal("klapp")
	}
}

func TestKeyFromNomenclatureGroup(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"Ангиофарм ": "angiopharm", "ANGIOPHARM": "angiopharm",
		"Кристина": "christina", "CHRISTINA": "christina",
		"KLAPP": "klapp", "SKIN SYNERGY": "skin_synergy",
		"LeviSsime": "levissime", "SOTHYS": "sothys", "NOVACUTAN": "novacutan",
	}
	for group, want := range cases {
		got, ok := brands.KeyFromNomenclatureGroup(group)
		if !ok || got != want {
			t.Fatalf("%q: got %q ok=%v", group, got, ok)
		}
	}
	if _, ok := brands.KeyFromNomenclatureGroup("Неизвестный бренд"); ok {
		t.Fatal("unknown")
	}
}

func TestDetectChristinaVariantFromFileName(t *testing.T) {
	t.Parallel()
	brand, variant, ok := brands.Detect("CHRISTINA", "Актуальный_бланк PROFF.xlsx")
	if !ok || brand != "christina" || variant != "PROFF" {
		t.Fatalf("got %s %s ok=%v", brand, variant, ok)
	}
	_, variant, ok = brands.Detect("Кристина", "бланк HOME.xlsx")
	if !ok || variant != "HOME" {
		t.Fatalf("home variant=%q", variant)
	}
}
