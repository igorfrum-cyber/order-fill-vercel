package domain

// BudgetRow is one priced supplier line for planBudget. Quantities are supplier
// units; demand, stock and outbound are pieces. Matches origin/main budgetPlanner.js.
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
}

// BudgetOptions are explicit per-preview opt-ins, never inferred from the target.
type BudgetOptions struct {
	AllowOverSix  bool
	AllowBelowOne bool
}

// BudgetPlan is a pure preview: it never mutates the input.
type BudgetPlan struct {
	Rows     []BudgetRow
	Before   float64
	Total    float64
	Target   float64
	Reason   string
	Complete bool
}
