package grpcapi

import (
	"testing"

	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
	"order-fill/backend/services/brand-service/internal/service/brands"
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
