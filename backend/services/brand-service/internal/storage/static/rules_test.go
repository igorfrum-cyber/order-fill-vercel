package static

import (
	"reflect"
	"testing"

	"order-fill/backend/services/brand-service/internal/domain"
)

func TestListReturnsSupportedBrandKeys(t *testing.T) {
	t.Parallel()

	want := []string{"angiopharm", "christina", "klapp", "levissime", "novacutan", "skin_synergy", "sothys"}
	if got := List(); !reflect.DeepEqual(got, want) {
		t.Fatalf("List() = %v, want %v", got, want)
	}
}

func TestPolicyReturnsBrandSettingsAndFallback(t *testing.T) {
	t.Parallel()

	if got := Policy("unknown"); got.Key != "angiopharm" || got.Adjustment != domain.AdjustmentBox {
		t.Fatalf("unknown brand fallback = %+v", got)
	}
	if got := Policy("christina"); got.Key != "christina" || got.Adjustment != domain.AdjustmentMultiple || got.Multiple != 3 {
		t.Fatalf("Christina policy = %+v", got)
	}
	if got := Policy("levissime"); got.Key != "levissime" || len(got.ArticlePrefixAliases) != 1 || got.ArticlePrefixAliases[0] != "MT" || got.BlankQuantityHeader != "order" || got.BlankBoxHeader != "packageQuantity" {
		t.Fatalf("LeviSsime policy = %+v", got)
	}
	if got := Policy("sothys"); got.Key != "sothys" || !got.PreserveArticleHyphen || got.BlankLayout != "splitVariants" {
		t.Fatalf("SOTHYS policy = %+v", got)
	}
	if got := Policy("novacutan"); got.Key != "novacutan" || got.BlankLayout != "novacutan" {
		t.Fatalf("Novacutan policy = %+v", got)
	}
	if got := Policy("skin_synergy"); got.RequireUnit == nil || *got.RequireUnit || got.BlankQuantityHeader != "exactQuantity" {
		t.Fatalf("Skin Synergy policy = %+v", got)
	}
	if got := Policy("klapp"); got.Key != "klapp" || got.Adjustment != domain.AdjustmentNearestMultiple || got.BlankQuantityHeader != "order" {
		t.Fatalf("KLAPP policy = %+v", got)
	}
}
