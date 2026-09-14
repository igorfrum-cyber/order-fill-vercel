package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"

	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
	"order-fill/backend/services/gateway-service/internal/clients"
)

type brandRulesClient struct{ brandv1.BrandServiceClient }

func (brandRulesClient) ListBrands(context.Context, *brandv1.ListBrandsRequest, ...grpc.CallOption) (*brandv1.ListBrandsResponse, error) {
	return &brandv1.ListBrandsResponse{Brands: []string{"klapp"}}, nil
}

func (brandRulesClient) GetBrandPolicy(context.Context, *brandv1.GetBrandPolicyRequest, ...grpc.CallOption) (*brandv1.GetBrandPolicyResponse, error) {
	return &brandv1.GetBrandPolicyResponse{Policy: &brandv1.BrandPolicy{Brand: "klapp", Label: "KLAPP", Adjustment: "nearestMultiple", QuantityMultiple: 3}}, nil
}

func TestListBrandRulesRequiresPlatformAdmin(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/brand-rules", nil)
	req = req.WithContext(withUser(t.Context(), User{Role: "company_owner"}))
	recorder := httptest.NewRecorder()
	api.listBrandRules(recorder, req)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d", recorder.Code)
	}
}

func TestListBrandRulesReturnsPolicies(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Brand: brandRulesClient{}}}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/brand-rules", nil)
	req = req.WithContext(withUser(t.Context(), User{ID: "admin", Role: "platform_admin"}))
	recorder := httptest.NewRecorder()
	api.listBrandRules(recorder, req)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"quantity_multiple":3`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
