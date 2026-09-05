package domain

type ReportCategory string

const (
	CategoryNeedsDecision     ReportCategory = "needs_decision"
	CategoryNotInSource       ReportCategory = "not_in_source"
	CategoryCheckNameOrVolume ReportCategory = "check_name_or_volume"
	CategoryNotInBlank        ReportCategory = "not_in_blank"
	CategoryToOrder           ReportCategory = "to_order"
	CategoryOrderNotNeeded    ReportCategory = "order_not_needed"
)

type ReportSummary struct {
	NeedsDecision     int
	NotInSource       int
	CheckNameOrVolume int
	NotInBlank        int
	ToOrder           int
	OrderNotNeeded    int
}

type ReportRow struct {
	ID       string
	Category ReportCategory
	Article  string
	Name     string
	Reasons  []string
}

type Report struct {
	Summary ReportSummary
	Rows    []ReportRow
}

type Edit struct {
	RowKey  string
	Field   string
	Value   string
	Comment string
}
