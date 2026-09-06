package orderfill_test

import (
	"context"
	"testing"

	"order-fill/backend/services/document-service/internal/clients/calculation"
	"order-fill/backend/services/document-service/internal/domain/brand"
	filldom "order-fill/backend/services/document-service/internal/domain/orderfill"
	"order-fill/backend/services/document-service/internal/service/north"
	"order-fill/backend/services/document-service/internal/service/orderfill"
	"order-fill/backend/services/document-service/internal/service/worker"
)

type fakeBrand struct{ key string }

func (f fakeBrand) Detect(context.Context, string, string) (string, string, error) {
	return f.key, "PROFF", nil
}
func (f fakeBrand) Policy(_ context.Context, key, _ string) (brand.RuleConfig, error) {
	return brand.RuleConfig{Key: key, Multiple: 3}, nil
}

type fakeMatch struct{}

func (fakeMatch) Match(_ context.Context, blank, _ []filldom.MatchItem, _ filldom.MatchOptions) ([]filldom.MatchResult, error) {
	return []filldom.MatchResult{{BlankID: blank[0].ID, SourceID: "s1", Category: "to_order", Score: 1}}, nil
}
func (fakeMatch) MergeChz(context.Context, []filldom.MatchItem, filldom.MatchOptions) ([]filldom.ChzMerge, error) {
	return nil, nil
}

type fakeCalc struct{ qty float64 }

func (f fakeCalc) Adjust(context.Context, string, float64) (float64, error) { return f.qty, nil }
func (f fakeCalc) AdjustQuantity(context.Context, float64, float64, bool, string, brand.RuleConfig) (brand.AdjustedQuantity, error) {
	qty := f.qty
	return brand.AdjustedQuantity{Inserted: &qty}, nil
}
func (f fakeCalc) Recommend(_ context.Context, _ string, _ string, _ float64, rows []filldom.RecommendRow) ([]filldom.RecommendRow, error) {
	return rows, nil
}
func (f fakeCalc) NorthPlan(_ context.Context, _ string, _ []calculation.NorthNeed, _ []calculation.TyumenStock) ([]calculation.NorthRow, error) {
	return []calculation.NorthRow{{Article: "A1", SupplierQty: 9, Comment: "KLAPP"}}, nil
}

func TestFillOrchestratesBrandMatchingCalculation(t *testing.T) {
	t.Parallel()
	p := orderfill.New(fakeBrand{key: "christina"}, fakeMatch{}, fakeCalc{qty: 12})
	brand, cells, err := p.Fill(t.Context(), orderfill.FillRequest{
		NomenclatureGroup: "CHRISTINA",
		BlankFileName:     "blank.xlsx",
		Blank:             []filldom.MatchItem{{ID: "b1", Article: "A1"}},
		Source:            []filldom.MatchItem{{ID: "s1", Article: "A1"}},
		Recommended:       map[string]float64{"s1": 11},
	})
	if err != nil || brand != "christina" || len(cells) != 1 || cells[0].Qty != 12 {
		t.Fatalf("brand=%s cells=%v err=%v", brand, cells, err)
	}
}

func TestPlanOneBlank(t *testing.T) {
	t.Parallel()
	if err := orderfill.PlanOneBlank("christina", []string{"a.xlsx", "b.xlsx"}); err == nil {
		t.Fatal("expected one blank")
	}
}

func TestNorthProcessor(t *testing.T) {
	t.Parallel()
	rows, err := north.New(fakeCalc{}).Plan(t.Context(), "klapp", []calculation.NorthNeed{{City: "surgut", Article: "A1", Qty: 10}}, nil)
	if err != nil || rows[0].SupplierQty != 9 {
		t.Fatalf("%v err=%v", rows, err)
	}
}

func TestDispatch(t *testing.T) {
	t.Parallel()
	got, err := worker.Dispatch("north_merge")
	if err != nil || got != "north" {
		t.Fatalf("%s %v", got, err)
	}
	if _, err := worker.Dispatch("other"); err == nil {
		t.Fatal("unknown")
	}
}
