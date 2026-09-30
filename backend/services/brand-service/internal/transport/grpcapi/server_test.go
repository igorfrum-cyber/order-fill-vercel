package grpcapi

import (
	"math"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	"order-fill/backend/services/brand-service/internal/domain"
	"order-fill/backend/services/brand-service/internal/service/brands"
	"order-fill/backend/services/brand-service/internal/storage/memory"
)

func TestGetBrandPolicyExposesMatchingAndBlankRules(t *testing.T) {
	t.Parallel()
	s := NewServer(brands.New())
	resp, err := s.GetBrandPolicy(t.Context(), &brandv1.GetBrandPolicyRequest{Brand: "levissime"})
	if err != nil {
		t.Fatalf("GetBrandPolicy: %v", err)
	}
	p := resp.GetPolicy()
	if p.GetLabel() != "LeviSsime" {
		t.Fatalf("label = %q", p.GetLabel())
	}
	if p.GetAdjustment() != "box" || p.GetBlankQuantityHeader() != "order" || p.GetBlankBoxHeader() != "packageQuantity" {
		t.Fatalf("levissime policy %+v", p)
	}
	if len(p.GetPrefixAliases()) != 1 || p.GetPrefixAliases()[0] != "MT" {
		t.Fatalf("prefix aliases %v", p.GetPrefixAliases())
	}
}

func TestUpdateBrandPolicyRequiresPlatformAdmin(t *testing.T) {
	t.Parallel()
	s := NewServer(brands.New(memory.New()))
	req := &brandv1.UpdateBrandPolicyRequest{Policy: &brandv1.BrandPolicy{Brand: "klapp", Label: "KLAPP", Adjustment: "nearestMultiple", QuantityMultiple: 3}}
	if _, err := s.UpdateBrandPolicy(t.Context(), req); status.Code(err) != codes.PermissionDenied {
		t.Fatalf("code = %v", status.Code(err))
	}
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("x-actor-role", "platform_admin"))
	if _, err := s.UpdateBrandPolicy(ctx, req); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("missing actor code = %v", status.Code(err))
	}
}

func TestUpdateBrandPolicySavesForPlatformAdmin(t *testing.T) {
	t.Parallel()
	store := memory.New()
	s := NewServer(brands.New(store))
	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("x-actor-role", "platform_admin"))
	req := &brandv1.UpdateBrandPolicyRequest{
		Meta:   &commonv1.RequestMeta{ActorUserId: "admin"},
		Policy: &brandv1.BrandPolicy{Brand: "klapp", Label: "KLAPP", Adjustment: "nearestMultiple", QuantityMultiple: 6},
	}
	resp, err := s.UpdateBrandPolicy(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetPolicy().GetQuantityMultiple() != 6 {
		t.Fatalf("rpc policy %+v", resp.GetPolicy())
	}
	got, ok, err := store.Get(t.Context(), "klapp")
	if err != nil || !ok || got.Multiple != 6 {
		t.Fatalf("stored %+v ok=%v err=%v", got, ok, err)
	}
}

func TestGetBrandPolicyPreservesSothysHyphen(t *testing.T) {
	t.Parallel()
	s := NewServer(brands.New())
	resp, err := s.GetBrandPolicy(t.Context(), &brandv1.GetBrandPolicyRequest{Brand: "sothys"})
	if err != nil {
		t.Fatalf("GetBrandPolicy: %v", err)
	}
	if !resp.GetPolicy().GetPreserveHyphen() || resp.GetPolicy().GetBlankLayout() != "splitVariants" {
		t.Fatalf("%+v", resp.GetPolicy())
	}
}

func TestDetectBrandMapsNomenclatureGroup(t *testing.T) {
	t.Parallel()
	s := NewServer(brands.New())
	resp, err := s.DetectBrand(t.Context(), &brandv1.DetectBrandRequest{NomenclatureGroup: "LeviSsime"})
	if err != nil {
		t.Fatalf("DetectBrand: %v", err)
	}
	if resp.GetBrand() != "levissime" {
		t.Fatalf("brand = %q", resp.GetBrand())
	}
}

func TestListBrandsReturnsCatalog(t *testing.T) {
	t.Parallel()

	resp, err := NewServer(brands.New()).ListBrands(t.Context(), &brandv1.ListBrandsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"angiopharm", "christina", "klapp", "levissime", "novacutan", "skin_synergy", "sothys"}
	if len(resp.GetBrands()) != len(want) {
		t.Fatalf("brands = %v", resp.GetBrands())
	}
	for index, brand := range want {
		if resp.GetBrands()[index] != brand {
			t.Fatalf("brand[%d] = %q, want %q", index, resp.GetBrands()[index], brand)
		}
	}
}

func TestDetectBrandReturnsEmptyForUnknownInputs(t *testing.T) {
	t.Parallel()

	resp, err := NewServer(brands.New()).DetectBrand(t.Context(), &brandv1.DetectBrandRequest{NomenclatureGroup: "unknown", FileName: "unknown.xlsx"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.GetBrand() != "" || resp.GetVariant() != "" {
		t.Fatalf("unknown brand detection = %+v", resp)
	}
}

func TestProtoPolicyClampsIntegerFieldsAndPreservesOptionalUnitFlag(t *testing.T) {
	t.Parallel()

	policy := protoPolicy(domain.Policy{Key: "brand", Multiple: math.MaxInt32 + 1, MinQuantity: math.MinInt32 - 1})
	if policy.GetQuantityMultiple() != math.MaxInt32 || policy.GetMinQuantity() != math.MinInt32 || policy.RequireUnit != nil {
		t.Fatalf("clamped policy = %+v", policy)
	}
	requireUnit := false
	policy = protoPolicy(domain.Policy{RequireUnit: &requireUnit})
	if policy.RequireUnit == nil || *policy.RequireUnit {
		t.Fatalf("optional require-unit value was lost: %+v", policy)
	}
}

func TestNewRegistersConfiguredHandler(t *testing.T) {
	t.Parallel()

	server := New(&Server{svc: brands.New()})
	defer server.Stop()
	if len(server.GetServiceInfo()) != 1 {
		t.Fatalf("registered services = %v", server.GetServiceInfo())
	}
}

func TestUpdateBrandPolicyRejectsInvalidPolicy(t *testing.T) {
	t.Parallel()

	ctx := metadata.NewIncomingContext(t.Context(), metadata.Pairs("x-actor-role", "platform_admin"))
	request := &brandv1.UpdateBrandPolicyRequest{
		Meta:   &commonv1.RequestMeta{ActorUserId: "admin"},
		Policy: &brandv1.BrandPolicy{Brand: "klapp", Label: "KLAPP", Adjustment: "invalid"},
	}
	if _, err := NewServer(brands.New(memory.New())).UpdateBrandPolicy(ctx, request); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("invalid policy code = %v, error = %v", status.Code(err), err)
	}
}
