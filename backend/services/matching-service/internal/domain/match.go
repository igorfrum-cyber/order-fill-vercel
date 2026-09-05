package domain

type Reasons struct {
	Article    string
	Name       string
	Volume     string
	Form       string
	Duplicates string
	Source     string
}

type Category string

const (
	CategoryUnspecified       Category = ""
	CategoryNeedsDecision     Category = "needs_decision"
	CategoryNotInSource       Category = "not_in_source"
	CategoryCheckNameOrVolume Category = "check_name_or_volume"
	CategoryNotInBlank        Category = "not_in_blank"
	CategoryToOrder           Category = "to_order"
	CategoryOrderNotNeeded    Category = "order_not_needed"
)

type Result struct {
	BlankItemID  string
	SourceItemID string
	Category     Category
	Reasons      Reasons
	Score        float64
	CandidateIDs []string
}

type Mode string

const (
	ModeStandard Mode = "standard"
	ModeSmart    Mode = "smart"
)
