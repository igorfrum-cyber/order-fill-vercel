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
