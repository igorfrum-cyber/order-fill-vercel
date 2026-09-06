package orderfill

import (
	"context"

	"order-fill/backend/services/document-service/internal/domain/brand"
)

const (
	CategoryToOrder           = "to_order"
	CategoryNeedsDecision     = "needs_decision"
	CategoryNotInSource       = "not_in_source"
	CategoryCheckNameOrVolume = "check_name_or_volume"
	CategoryNotInBlank        = "not_in_blank"
	CategoryOrderNotNeeded    = "order_not_needed"
)

type MatchItem struct {
	ID          string
	Article     string
	Name        string
	Volume      string
	Form        string
	ChestnyZnak bool
	Rounded     int
}

type MatchReasons struct {
	Article    string `json:"article,omitempty"`
	Name       string `json:"name,omitempty"`
	Volume     string `json:"volume,omitempty"`
	Form       string `json:"form,omitempty"`
	Duplicates string `json:"duplicates,omitempty"`
	Source     string `json:"source,omitempty"`
}

type MatchResult struct {
	BlankID      string
	SourceID     string
	Category     string
	Score        float64
	CandidateIDs []string
	Reasons      MatchReasons
}

type MatchOptions struct {
	Mode           string
	PrefixAliases  []string
	PreserveHyphen bool
}

type Matcher interface {
	Match(ctx context.Context, blank, source []MatchItem, opts MatchOptions) ([]MatchResult, error)
}

type ChzMerge struct {
	TargetID      string
	CloneIDs      []string
	NeedsDecision bool
}

type ChzMerger interface {
	MergeChz(ctx context.Context, items []MatchItem, opts MatchOptions) ([]ChzMerge, error)
}

type RecommendRow struct {
	ID                                string
	Category                          string
	Revenue, Stock, InTransit         float64
	MonthlySales                      []float64
	Recommended, TargetStock          float64
	RevenuePercent, CumulativePercent float64
	AverageMonthly, TotalQuantity     float64
}

type SourceRecommender interface {
	Recommend(ctx context.Context, brand, cityRule string, weeks float64, rows []RecommendRow) ([]RecommendRow, error)
}

type QuantityAdjuster interface {
	AdjustQuantity(ctx context.Context, recommended, orderedFact float64, hasFact bool, boxSize string, rule brand.RuleConfig) (brand.AdjustedQuantity, error)
}
