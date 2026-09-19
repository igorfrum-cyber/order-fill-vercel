package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	calculationv1 "order-fill/backend/proto/gen/go/orderfill/calculation/v1"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
	"order-fill/backend/services/gateway-service/internal/clients"
)

type fakeCalcClient struct {
	calculationv1.CalculationServiceClient
	req  *calculationv1.PlanBudgetRequest
	resp *calculationv1.PlanBudgetResponse
	err  error
}

func (c *fakeCalcClient) PlanBudget(_ context.Context, req *calculationv1.PlanBudgetRequest, _ ...grpc.CallOption) (*calculationv1.PlanBudgetResponse, error) {
	c.req = req
	if c.err != nil {
		return nil, c.err
	}
	return c.resp, nil
}

func TestPlanOrderBudgetMapsRequestAndResponse(t *testing.T) {
	t.Parallel()
	calc := &fakeCalcClient{resp: &calculationv1.PlanBudgetResponse{
		Before: 5640, Total: 5880, Target: 6100, Complete: true,
		Rows: []*calculationv1.PlannedBudgetRow{
			{Key: "MUSE:3", Name: "MUSE 3", Category: "A+", Before: 3, Quantity: 6, Comment: "Добавилось 3 шт. Для закупа до суммы."},
		},
		LineSteps:  []*calculationv1.PlannedLineStep{{Id: "MUSE", Name: "MUSE", Sets: 6, Added: 300, Saved: 60}},
		LineGroups: []*calculationv1.PlannedLineGroup{{Id: "MUSE", Name: "MUSE", Valid: true, Sets: 6, Saving: 120, Net: 5880}},
	}}
	api := &API{Clients: clients.Clients{Calculation: calc}}

	body := `{"brand":"christina","target":6100,"discount":0,"delivery_weeks":1,"rows":[
		{"key":"MUSE:3","name":"MUSE 3","category":"A+","quantity":3,"base_price":100,"demand":10,"group":"proff",
		 "line":{"id":"MUSE","name":"MUSE","article":"3","required":["0","1","2","3"]}}]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/order/budget-plan", strings.NewReader(body))
	rec := httptest.NewRecorder()
	api.planOrderBudget(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if calc.req.GetBrand() != "christina" || calc.req.GetDeliveryWeeks() != 1 {
		t.Fatalf("request not forwarded: %+v", calc.req)
	}
	sent := calc.req.GetRows()[0]
	if sent.GetBasePrice() != 100 || sent.GetGroup() != "proff" || sent.GetLine().GetId() != "MUSE" || len(sent.GetLine().GetRequired()) != 4 {
		t.Fatalf("row not mapped: %+v", sent)
	}
	var got budgetPlanResponseJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !got.Complete || got.Total != 5880 || len(got.Rows) != 1 || got.Rows[0].Category != "A+" {
		t.Fatalf("response not mapped: %+v", got)
	}
	if len(got.LineSteps) != 1 || got.LineSteps[0].Saved != 60 || len(got.LineGroups) != 1 || got.LineGroups[0].Net != 5880 {
		t.Fatalf("line detail not mapped: %+v", got)
	}
	if got.ChristinaProffMode != "standard" || got.Compare != nil {
		t.Fatalf("default mode not mapped: %+v", got)
	}
	if calc.req.GetChristinaProffMode() != commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_STANDARD {
		t.Fatalf("nil identity should send standard: %v", calc.req.GetChristinaProffMode())
	}
}

func TestPlanOrderBudgetRejectsBadDiscount(t *testing.T) {
	t.Parallel()
	api := &API{Clients: clients.Clients{Calculation: &fakeCalcClient{}}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/order/budget-plan", strings.NewReader(`{"discount":100,"target":10}`))
	rec := httptest.NewRecorder()
	api.planOrderBudget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d", rec.Code)
	}
}

func TestPlanOrderBudgetForwardsCalcError(t *testing.T) {
	t.Parallel()
	calc := &fakeCalcClient{err: status.Error(codes.InvalidArgument, "введите неотрицательную сумму")}
	api := &API{Clients: clients.Clients{Calculation: calc}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/order/budget-plan", strings.NewReader(`{"target":-1}`))
	rec := httptest.NewRecorder()
	api.planOrderBudget(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

type fakeIdentityListClient struct {
	identityv1.IdentityServiceClient
	companies []*identityv1.Company
}

func (c *fakeIdentityListClient) ListCompanies(_ context.Context, _ *identityv1.ListCompaniesRequest, _ ...grpc.CallOption) (*identityv1.ListCompaniesResponse, error) {
	return &identityv1.ListCompaniesResponse{Companies: c.companies}, nil
}

func TestPlanOrderBudgetInjectsCompanyModeAndIgnoresClientMode(t *testing.T) {
	t.Parallel()
	calc := &fakeCalcClient{resp: &calculationv1.PlanBudgetResponse{
		Before: 100, Total: 120, Target: 150, Complete: true,
		ChristinaProffMode: commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_COMPARE,
		FastTotal:          120, FastComplete: true,
		FastRows: []*calculationv1.PlannedBudgetRow{{Key: "MUSE:3", Quantity: 6}},
		Compare: &calculationv1.BudgetOracleCompare{
			Match: false, StandardMs: 40, FastMs: 8,
			Mismatches: []*calculationv1.BudgetOracleMismatch{
				{Where: "changeCost", Key: "MUSE:3", Field: "quantity", Want: "6", Got: "9"},
			},
		},
	}}
	ident := &fakeIdentityListClient{companies: []*identityv1.Company{{
		Id: "co-1", ChristinaProffMode: commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_COMPARE,
	}}}
	api := &API{Clients: clients.Clients{Calculation: calc, Identity: ident}}

	body := `{"brand":"christina","target":150,"discount":0,"rows":[{"key":"MUSE:3"}],"christina_proff_mode":"fast"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/order/budget-plan", strings.NewReader(body))
	req = req.WithContext(withUser(req.Context(), User{ID: "u1", Role: "purchaser", CompanyID: "co-1"}))
	rec := httptest.NewRecorder()
	api.planOrderBudget(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if calc.req.GetChristinaProffMode() != commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_COMPARE {
		t.Fatalf("injected mode=%v", calc.req.GetChristinaProffMode())
	}
	var got budgetPlanResponseJSON
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.ChristinaProffMode != "compare" || got.Compare == nil || got.Compare.Match || len(got.Compare.Mismatches) != 1 {
		t.Fatalf("compare not mapped: %+v", got)
	}
	if got.FastTotal != 120 || len(got.FastRows) != 1 || got.Compare.Mismatches[0].Want != "6" {
		t.Fatalf("fast/mismatch not passed through: %+v", got)
	}
}

func TestPlanOrderBudgetMissingCompanyUsesStandard(t *testing.T) {
	t.Parallel()
	calc := &fakeCalcClient{resp: &calculationv1.PlanBudgetResponse{Total: 10, Target: 10}}
	ident := &fakeIdentityListClient{companies: []*identityv1.Company{{
		Id: "other", ChristinaProffMode: commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_FAST,
	}}}
	api := &API{Clients: clients.Clients{Calculation: calc, Identity: ident}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/order/budget-plan", strings.NewReader(`{"target":10,"rows":[]}`))
	req = req.WithContext(withUser(req.Context(), User{ID: "u1", Role: "purchaser", CompanyID: "co-1"}))
	rec := httptest.NewRecorder()
	api.planOrderBudget(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if calc.req.GetChristinaProffMode() != commonv1.ChristinaProffMode_CHRISTINA_PROFF_MODE_STANDARD {
		t.Fatalf("mode=%v", calc.req.GetChristinaProffMode())
	}
}
