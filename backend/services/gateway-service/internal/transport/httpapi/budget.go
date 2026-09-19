package httpapi

import (
	"cmp"
	"net/http"

	calculationv1 "order-fill/backend/proto/gen/go/orderfill/calculation/v1"
	identityv1 "order-fill/backend/proto/gen/go/orderfill/identity/v1"
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
	Before             float64                `json:"before"`
	Total              float64                `json:"total"`
	Target             float64                `json:"target"`
	Reason             string                 `json:"reason"`
	Complete           bool                   `json:"complete"`
	Rows               []plannedBudgetRowJSON `json:"rows"`
	LineSteps          []budgetLineStepJSON   `json:"line_steps"`
	LineGroups         []budgetLineGroupJSON  `json:"line_groups"`
	ChristinaProffMode string                 `json:"christina_proff_mode"`
	FastTotal          float64                `json:"fast_total,omitzero"`
	FastComplete       bool                   `json:"fast_complete,omitzero"`
	FastReason         string                 `json:"fast_reason,omitempty"`
	FastRows           []plannedBudgetRowJSON `json:"fast_rows,omitempty"`
	FastLineSteps      []budgetLineStepJSON   `json:"fast_line_steps,omitempty"`
	Compare            *budgetCompareJSON     `json:"compare,omitempty"`
}

type budgetCompareJSON struct {
	Match      bool                 `json:"match"`
	StandardMs int64                `json:"standard_ms"`
	FastMs     int64                `json:"fast_ms"`
	Mismatches []budgetMismatchJSON `json:"mismatches"`
}

type budgetMismatchJSON struct {
	Where string `json:"where"`
	Key   string `json:"key"`
	Field string `json:"field,omitempty"`
	Want  string `json:"want"`
	Got   string `json:"got"`
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
	mode := a.actorChristinaProffMode(r, user)
	resp, err := a.Clients.Calculation.PlanBudget(a.jobCtx(r, user), &calculationv1.PlanBudgetRequest{
		Brand: payload.Brand, Target: payload.Target, Discount: payload.Discount,
		DeliveryWeeks: payload.DeliveryWeeks, AllowOverSix: payload.AllowOverSix,
		AllowBelowOne: payload.AllowBelowOne, Rows: rows,
		ChristinaProffMode: protoChristinaProffMode(mode),
	})
	if err != nil {
		writeGRPCError(w, "budget_plan_failed", err)
		return
	}
	writeJSON(w, http.StatusOK, presentBudgetPlan(resp, mode))
}

// actorChristinaProffMode reads the session company's oracle from identity.
// Missing company, identity errors, and client JSON are ignored → standard.
func (a *API) actorChristinaProffMode(r *http.Request, user User) string {
	if a.Clients.Identity == nil || user.CompanyID == "" {
		return "standard"
	}
	resp, err := a.Clients.Identity.ListCompanies(r.Context(), &identityv1.ListCompaniesRequest{Meta: a.meta(user)})
	if err != nil {
		return "standard"
	}
	for _, company := range resp.GetCompanies() {
		if company.GetId() == user.CompanyID {
			return companyChristinaProffMode(company)
		}
	}
	return "standard"
}

func budgetLineProto(line *budgetLineJSON) *calculationv1.ChristinaLine {
	if line == nil {
		return nil
	}
	return &calculationv1.ChristinaLine{
		Id: line.ID, Name: line.Name, Article: line.Article, Required: line.Required,
	}
}

func presentBudgetPlan(resp *calculationv1.PlanBudgetResponse, mode string) budgetPlanResponseJSON {
	used := jsonChristinaProffMode(resp.GetChristinaProffMode())
	if resp.GetChristinaProffMode() == 0 {
		used = cmp.Or(mode, "standard")
	}
	out := budgetPlanResponseJSON{
		Before: resp.GetBefore(), Total: resp.GetTotal(), Target: resp.GetTarget(),
		Reason: resp.GetReason(), Complete: resp.GetComplete(),
		ChristinaProffMode: used,
		Rows:               plannedRowsJSON(resp.GetRows()),
		LineSteps:          plannedStepsJSON(resp.GetLineSteps()),
		LineGroups:         make([]budgetLineGroupJSON, 0, len(resp.GetLineGroups())),
	}
	for _, group := range resp.GetLineGroups() {
		out.LineGroups = append(out.LineGroups, budgetLineGroupJSON{
			ID: group.GetId(), Name: group.GetName(), Valid: group.GetValid(),
			Sets: group.GetSets(), Saving: group.GetSaving(), Net: group.GetNet(),
		})
	}
	if cmpRes := resp.GetCompare(); cmpRes != nil {
		out.Compare = presentBudgetCompare(cmpRes)
		out.FastTotal = resp.GetFastTotal()
		out.FastComplete = resp.GetFastComplete()
		out.FastReason = resp.GetFastReason()
		out.FastRows = plannedRowsJSON(resp.GetFastRows())
		out.FastLineSteps = plannedStepsJSON(resp.GetFastLineSteps())
	}
	return out
}

func plannedRowsJSON(rows []*calculationv1.PlannedBudgetRow) []plannedBudgetRowJSON {
	out := make([]plannedBudgetRowJSON, 0, len(rows))
	for _, row := range rows {
		out = append(out, plannedBudgetRowJSON{
			Key: row.GetKey(), Name: row.GetName(), Category: row.GetCategory(),
			Before: row.GetBefore(), Quantity: row.GetQuantity(), Comment: row.GetComment(),
		})
	}
	return out
}

func plannedStepsJSON(steps []*calculationv1.PlannedLineStep) []budgetLineStepJSON {
	out := make([]budgetLineStepJSON, 0, len(steps))
	for _, step := range steps {
		out = append(out, budgetLineStepJSON{
			ID: step.GetId(), Name: step.GetName(), Sets: step.GetSets(),
			Added: step.GetAdded(), Saved: step.GetSaved(),
		})
	}
	return out
}

func presentBudgetCompare(cmpRes *calculationv1.BudgetOracleCompare) *budgetCompareJSON {
	out := &budgetCompareJSON{
		Match: cmpRes.GetMatch(), StandardMs: cmpRes.GetStandardMs(), FastMs: cmpRes.GetFastMs(),
		Mismatches: make([]budgetMismatchJSON, 0, len(cmpRes.GetMismatches())),
	}
	for _, item := range cmpRes.GetMismatches() {
		out.Mismatches = append(out.Mismatches, budgetMismatchJSON{
			Where: item.GetWhere(), Key: item.GetKey(), Field: item.GetField(),
			Want: item.GetWant(), Got: item.GetGot(),
		})
	}
	return out
}
