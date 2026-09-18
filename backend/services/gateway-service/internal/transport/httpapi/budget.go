package httpapi

import (
	"net/http"

	calculationv1 "order-fill/backend/proto/gen/go/orderfill/calculation/v1"
)

// budgetPlanLimit caps the request body; a blank rarely exceeds a few hundred
// rows, so 1 MiB is generous while bounding untrusted input.
const budgetPlanLimit = 1 << 20

type budgetLineJSON struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Article  string   `json:"article"`
	Required []string `json:"required"`
}

type budgetPlanRowJSON struct {
	Key       string          `json:"key"`
	Name      string          `json:"name"`
	Category  string          `json:"category"`
	Quantity  float64         `json:"quantity"`
	BasePrice float64         `json:"base_price"`
	Demand    float64         `json:"demand"`
	Stock     float64         `json:"stock"`
	Transit   float64         `json:"transit"`
	Outbound  float64         `json:"outbound"`
	BoxSize   float64         `json:"box_size"`
	Group     string          `json:"group"`
	Locked    bool            `json:"locked"`
	Excluded  bool            `json:"excluded"`
	Unsafe    bool            `json:"unsafe"`
	Line      *budgetLineJSON `json:"line"`
}

type budgetPlanRequestJSON struct {
	Brand         string              `json:"brand"`
	Target        float64             `json:"target"`
	Discount      float64             `json:"discount"`
	DeliveryWeeks float64             `json:"delivery_weeks"`
	AllowOverSix  bool                `json:"allow_over_six"`
	AllowBelowOne bool                `json:"allow_below_one"`
	Rows          []budgetPlanRowJSON `json:"rows"`
}

type plannedBudgetRowJSON struct {
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	Category string  `json:"category"`
	Before   float64 `json:"before"`
	Quantity float64 `json:"quantity"`
	Comment  string  `json:"comment"`
}

type budgetLineStepJSON struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Sets  int32   `json:"sets"`
	Added float64 `json:"added"`
	Saved float64 `json:"saved"`
}

type budgetLineGroupJSON struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Valid  bool    `json:"valid"`
	Sets   int32   `json:"sets"`
	Saving float64 `json:"saving"`
	Net    float64 `json:"net"`
}

type budgetPlanResponseJSON struct {
	Before     float64                `json:"before"`
	Total      float64                `json:"total"`
	Target     float64                `json:"target"`
	Reason     string                 `json:"reason"`
	Complete   bool                   `json:"complete"`
	Rows       []plannedBudgetRowJSON `json:"rows"`
	LineSteps  []budgetLineStepJSON   `json:"line_steps"`
	LineGroups []budgetLineGroupJSON  `json:"line_groups"`
}

// planOrderBudget forwards a budget preview to calculation-service. Gateway stays
// thin: it validates the discount range at the trust boundary, maps raw report
// rows to the RPC, and returns the planned order; all pricing and the CHRISTINA
// PROFF math live in calculation-service.
func (a *API) planOrderBudget(w http.ResponseWriter, r *http.Request) {
	user, _ := userFrom(r)
	var payload budgetPlanRequestJSON
	if !decodeJSON(w, r, &payload, budgetPlanLimit) {
		return
	}
	if payload.Discount < 0 || payload.Discount >= 100 {
		writeError(w, http.StatusBadRequest, "bad_request", "скидка должна быть от 0 до 99,99%")
		return
	}
	rows := make([]*calculationv1.BudgetRow, 0, len(payload.Rows))
	for _, row := range payload.Rows {
		rows = append(rows, &calculationv1.BudgetRow{
			Key: row.Key, Name: row.Name, Category: row.Category, Quantity: row.Quantity,
			BasePrice: row.BasePrice, Demand: row.Demand, Stock: row.Stock, Transit: row.Transit,
			Outbound: row.Outbound, BoxSize: row.BoxSize, Group: row.Group,
			Locked: row.Locked, Excluded: row.Excluded, Unsafe: row.Unsafe,
			Line: budgetLineProto(row.Line),
		})
	}
	resp, err := a.Clients.Calculation.PlanBudget(a.jobCtx(r, user), &calculationv1.PlanBudgetRequest{
		Brand: payload.Brand, Target: payload.Target, Discount: payload.Discount,
		DeliveryWeeks: payload.DeliveryWeeks, AllowOverSix: payload.AllowOverSix,
		AllowBelowOne: payload.AllowBelowOne, Rows: rows,
	})
	if err != nil {
		writeGRPCError(w, "budget_plan_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentBudgetPlan(resp))
}

func budgetLineProto(line *budgetLineJSON) *calculationv1.ChristinaLine {
	if line == nil {
		return nil
	}
	return &calculationv1.ChristinaLine{
		Id: line.ID, Name: line.Name, Article: line.Article, Required: line.Required,
	}
}

func presentBudgetPlan(resp *calculationv1.PlanBudgetResponse) budgetPlanResponseJSON {
	out := budgetPlanResponseJSON{
		Before: resp.GetBefore(), Total: resp.GetTotal(), Target: resp.GetTarget(),
		Reason: resp.GetReason(), Complete: resp.GetComplete(),
		Rows:       make([]plannedBudgetRowJSON, 0, len(resp.GetRows())),
		LineSteps:  make([]budgetLineStepJSON, 0, len(resp.GetLineSteps())),
		LineGroups: make([]budgetLineGroupJSON, 0, len(resp.GetLineGroups())),
	}
	for _, row := range resp.GetRows() {
		out.Rows = append(out.Rows, plannedBudgetRowJSON{
			Key: row.GetKey(), Name: row.GetName(), Category: row.GetCategory(),
			Before: row.GetBefore(), Quantity: row.GetQuantity(), Comment: row.GetComment(),
		})
	}
	for _, step := range resp.GetLineSteps() {
		out.LineSteps = append(out.LineSteps, budgetLineStepJSON{
			ID: step.GetId(), Name: step.GetName(), Sets: step.GetSets(),
			Added: step.GetAdded(), Saved: step.GetSaved(),
		})
	}
	for _, group := range resp.GetLineGroups() {
		out.LineGroups = append(out.LineGroups, budgetLineGroupJSON{
			ID: group.GetId(), Name: group.GetName(), Valid: group.GetValid(),
			Sets: group.GetSets(), Saving: group.GetSaving(), Net: group.GetNet(),
		})
	}
	return out
}
