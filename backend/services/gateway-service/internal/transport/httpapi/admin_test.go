package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/gateway-service/internal/clients"
)

type enableUserClient struct {
	identityv1.IdentityServiceClient
	userID string
	actor  string
	err    error
}

func (c *enableUserClient) EnableUser(_ context.Context, req *identityv1.EnableUserRequest, _ ...grpc.CallOption) (*identityv1.EnableUserResponse, error) {
	c.userID = req.GetUserId()
	c.actor = req.GetMeta().GetActorUserId()
	if c.err != nil {
		return nil, c.err
	}
	return &identityv1.EnableUserResponse{}, nil
}

func TestCompanyMatchingMode(t *testing.T) {
	t.Parallel()
	if got := companyMatchingMode(nil); got != "standard" {
		t.Fatalf("nil got %q", got)
	}
	if got := companyMatchingMode(&identityv1.Company{}); got != "standard" {
		t.Fatalf("zero got %q", got)
	}
	if got := companyMatchingMode(&identityv1.Company{MatchingMode: commonv1.MatchingMode_MATCHING_MODE_SMART}); got != "smart" {
		t.Fatalf("smart got %q", got)
	}
}

func TestProtoMatchingMode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want commonv1.MatchingMode
	}{
		{"smart", commonv1.MatchingMode_MATCHING_MODE_SMART},
		{"SMART", commonv1.MatchingMode_MATCHING_MODE_SMART},
		{"standard", commonv1.MatchingMode_MATCHING_MODE_STANDARD},
		{"", commonv1.MatchingMode_MATCHING_MODE_UNSPECIFIED},
		{"nope", commonv1.MatchingMode_MATCHING_MODE_UNSPECIFIED},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			if got := protoMatchingMode(tc.in); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestPresentCompanyForHidesMatchingMode(t *testing.T) {
	t.Parallel()
	api := &API{}
	company := &identityv1.Company{Id: "c1", MatchingMode: commonv1.MatchingMode_MATCHING_MODE_SMART}
	admin := api.presentCompanyFor(t.Context(), User{Role: "platform_admin"}, company)
	if admin["matching_mode"] != "smart" {
		t.Fatalf("admin got %#v", admin["matching_mode"])
	}
	purchaser := api.presentCompanyFor(t.Context(), User{Role: "purchaser"}, company)
	if _, ok := purchaser["matching_mode"]; ok {
		t.Fatalf("purchaser saw matching_mode: %#v", purchaser)
	}
}

func TestCompanyOrderProfileTransportPreservesExplicitZeroDiscount(t *testing.T) {
	t.Parallel()
	payload := companyOrderProfilePayload{
		LegalName:  "ООО Тест",
		BrandTerms: []companyBrandTermsPayload{{Brand: "klapp", DiscountSet: true}},
	}
	profile := payload.proto()
	if profile.GetLegalName() != "ООО Тест" || len(profile.GetBrandTerms()) != 1 || !profile.GetBrandTerms()[0].GetDiscountSet() {
		t.Fatalf("proto=%+v", profile)
	}
	presented := presentOrderProfile(profile)
	terms, ok := presented["brand_terms"].([]map[string]any)
	if !ok || len(terms) != 1 || terms[0]["discount_set"] != true || terms[0]["discount_basis_points"] != int32(0) {
		t.Fatalf("presented=%#v", presented)
	}
}

func TestGetCompanyOrderProfileRejectsPurchaserBeforeCallingIdentity(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/companies/co-1/order-profile", nil)
	req.SetPathValue("company_id", "co-1")
	req = req.WithContext(withUser(req.Context(), User{Role: "purchaser", CompanyID: "co-1"}))
	rec := httptest.NewRecorder()
	api.getCompanyOrderProfile(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestListAuditRequiresPlatformAdmin(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
	req = req.WithContext(withUser(req.Context(), User{Role: "purchaser", CompanyID: "co-1"}))
	rec := httptest.NewRecorder()
	api.listAudit(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestListCompaniesRequiresPlatformAdmin(t *testing.T) {
	t.Parallel()
	api := &API{}
	for _, role := range []string{"purchaser", "company_admin", "company_owner"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/companies", nil)
		req = req.WithContext(withUser(req.Context(), User{Role: role, CompanyID: "co-1"}))
		rec := httptest.NewRecorder()
		api.listCompanies(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("%s status=%d", role, rec.Code)
		}
	}
}

func TestListStatusRequiresPlatformAdmin(t *testing.T) {
	t.Parallel()
	api := &API{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/status", nil)
	req = req.WithContext(withUser(req.Context(), User{Role: "company_owner", CompanyID: "co-1"}))
	rec := httptest.NewRecorder()
	api.listStatus(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestEnableUserForwardsActorAndUserID(t *testing.T) {
	t.Parallel()
	identity := &enableUserClient{}
	api := &API{Clients: clients.Clients{Identity: identity}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/buyer/enable", nil)
	req.SetPathValue("user_id", "buyer")
	req = req.WithContext(withUser(t.Context(), User{ID: "owner", Role: "company_owner", CompanyID: "c1"}))
	rec := httptest.NewRecorder()
	api.enableUser(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if identity.userID != "buyer" || identity.actor != "owner" {
		t.Fatalf("user=%q actor=%q", identity.userID, identity.actor)
	}
}

func TestEnableUserMapsIdentityNotFound(t *testing.T) {
	t.Parallel()
	identity := &enableUserClient{err: status.Error(codes.NotFound, "not found")}
	api := &API{Clients: clients.Clients{Identity: identity}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/root/enable", nil)
	req.SetPathValue("user_id", "root")
	req = req.WithContext(withUser(t.Context(), User{ID: "ops", Role: "platform_admin"}))
	rec := httptest.NewRecorder()
	api.enableUser(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestPlatformAdminManagementRequiresPlatformRoleAndPrimaryForCreate(t *testing.T) {
	t.Parallel()
	api := &API{}

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/platform-admins", nil)
	listReq = listReq.WithContext(withUser(listReq.Context(), User{Role: "company_owner", CompanyID: "co-1"}))
	listRec := httptest.NewRecorder()
	api.listPlatformAdmins(listRec, listReq)
	if listRec.Code != http.StatusNotFound {
		t.Fatalf("company owner list status=%d", listRec.Code)
	}

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/platform-admins", nil)
	createReq = createReq.WithContext(withUser(createReq.Context(), User{Role: "platform_admin"}))
	createRec := httptest.NewRecorder()
	api.createPlatformAdmin(createRec, createReq)
	if createRec.Code != http.StatusNotFound {
		t.Fatalf("secondary admin create status=%d", createRec.Code)
	}
}
