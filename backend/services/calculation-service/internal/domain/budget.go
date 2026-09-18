package domain

// ChristinaLine is the PROFF product line a budget row belongs to. Required is
// the full article list from the blank (not just matched rows), so the planner
// can tell whether a line is complete. Matches origin/main christinaLines.js.
type ChristinaLine struct {
	ID       string
	Name     string
	Article  string
	Required []string
}

// BudgetRow is one priced supplier line for PlanBudget. Quantities are supplier
// units; demand, stock and outbound are pieces. Price is the unit price AFTER
// the main discount — pricing happens upstream, this service only plans
// quantities. Matches origin/main budgetPlanner.js.
type BudgetRow struct {
	Key       string
	Name      string
	Category  string
	Quantity  float64
	Before    float64
	HasBefore bool
	Price     float64
	Demand    float64
	Delivery  float64
	Stock     float64
	Transit   float64
	Outbound  float64
	Unit      float64
	Step      float64
	Minimum   float64
	Locked    bool
	Excluded  bool
	Unsafe    bool
	// Group is the blank type; only "proff" rows earn the CHRISTINA set discount.
	Group string
	// Line is the PROFF line membership; nil for HOME and other brands.
	Line *ChristinaLine
	// LineCompletion carries the line name into the change comment when a PROFF
	// set was completed for this row. Set by PlanBudget, empty otherwise.
	LineCompletion string
}

// BudgetOptions are explicit per-preview opt-ins, never inferred from the target.
type BudgetOptions struct {
	AllowOverSix  bool
	AllowBelowOne bool
}

// BudgetInputRow is a raw report row before pricing and brand order rules. The
// service turns it into a BudgetRow so the browser and gateway pass only raw
// numbers and never run the pricing/quantity business logic.
type BudgetInputRow struct {
	Key       string
	Name      string
	Category  string
	Quantity  float64
	BasePrice float64 // price before the main discount
	Demand    float64
	Stock     float64
	Transit   float64
	Outbound  float64
	BoxSize   float64 // blank box/multiple hint for order rules
	Group     string
	Line      *ChristinaLine
	Locked    bool
	Excluded  bool
	Unsafe    bool
}

// BudgetRequest is the report-facing budget input: raw rows plus the brand,
// discount and delivery that the service needs to price and apply order rules.
type BudgetRequest struct {
	Brand         string
	Discount      float64 // main discount percent (0..99.99)
	DeliveryWeeks float64
	Target        float64
	Options       BudgetOptions
	Rows          []BudgetInputRow
}

// CompletionStep records one accepted CHRISTINA PROFF set-completion step so the
// UI can explain the added sets and discount. Matches origin/main lineSteps.
type CompletionStep struct {
	ID         string
	Name       string
	Sets       int
	AddedCents int64
	SavedCents int64
}

// BudgetPlan is a pure preview: it never mutates the input.
type BudgetPlan struct {
	Rows      []BudgetRow
	Before    float64
	Total     float64
	Target    float64
	Reason    string
	Complete  bool
	LineSteps []CompletionStep
}
