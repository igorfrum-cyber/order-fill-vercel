package matching

import (
	"context"
	"math"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	matchingv1 "order-fill/backend/proto/gen/go/orderfill/matching/v1"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

type Client interface {
	Match(ctx context.Context, blank, source []orderfill.MatchItem, opts orderfill.MatchOptions) ([]orderfill.MatchResult, error)
	MergeChz(ctx context.Context, items []orderfill.MatchItem, opts orderfill.MatchOptions) ([]orderfill.ChzMerge, error)
}

type GRPC struct {
	client matchingv1.MatchingServiceClient
}

func Dial(ctx context.Context, addr string) (*GRPC, error) {
	conn, err := grpcutil.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &GRPC{client: matchingv1.NewMatchingServiceClient(conn)}, nil
}

func (c *GRPC) Match(ctx context.Context, blank, source []orderfill.MatchItem, opts orderfill.MatchOptions) ([]orderfill.MatchResult, error) {
	req := &matchingv1.MatchRowsRequest{
		MatchingMode:   matchingMode(opts.Mode),
		PrefixAliases:  opts.PrefixAliases,
		PreserveHyphen: opts.PreserveHyphen,
	}
	for _, item := range blank {
		req.BlankItems = append(req.BlankItems, protoItem(item))
	}
	for _, item := range source {
		req.SourceItems = append(req.SourceItems, protoItem(item))
	}
	resp, err := c.client.MatchRows(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]orderfill.MatchResult, 0, len(resp.GetResults()))
	for _, result := range resp.GetResults() {
		reasons := result.GetReasons()
		out = append(out, orderfill.MatchResult{
			BlankID:      result.GetBlankItemId(),
			SourceID:     result.GetSourceItemId(),
			Category:     categoryName(result.GetCategory()),
			Score:        result.GetScore(),
			CandidateIDs: result.GetCandidateIds(),
			Reasons: orderfill.MatchReasons{
				Article:    reasons.GetArticle(),
				Name:       reasons.GetName(),
				Volume:     reasons.GetVolume(),
				Form:       reasons.GetForm(),
				Duplicates: reasons.GetDuplicates(),
				Source:     reasons.GetSource(),
			},
		})
	}
	return out, nil
}

func (c *GRPC) MergeChz(ctx context.Context, items []orderfill.MatchItem, opts orderfill.MatchOptions) ([]orderfill.ChzMerge, error) {
	req := &matchingv1.MergeChestnyZnakRequest{
		MatchingMode:   matchingMode(opts.Mode),
		PrefixAliases:  opts.PrefixAliases,
		PreserveHyphen: opts.PreserveHyphen,
	}
	for _, item := range items {
		req.Items = append(req.Items, protoItem(item))
	}
	resp, err := c.client.MergeChestnyZnak(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]orderfill.ChzMerge, 0, len(resp.GetMerges()))
	for _, merge := range resp.GetMerges() {
		out = append(out, orderfill.ChzMerge{
			TargetID: merge.GetTargetId(), CloneIDs: merge.GetCloneIds(), NeedsDecision: merge.GetNeedsDecision(),
		})
	}
	return out, nil
}

func protoItem(item orderfill.MatchItem) *matchingv1.Item {
	return &matchingv1.Item{
		Id: item.ID, Article: item.Article, Name: item.Name, Volume: item.Volume,
		Form: item.Form, ChestnyZnak: item.ChestnyZnak, Rounded: int32Clamp(item.Rounded),
	}
}

func int32Clamp(n int) int32 {
	if n > math.MaxInt32 {
		return math.MaxInt32
	}
	if n < math.MinInt32 {
		return math.MinInt32
	}
	return int32(n)
}

func matchingMode(mode string) commonv1.MatchingMode {
	if mode == "smart" {
		return commonv1.MatchingMode_MATCHING_MODE_SMART
	}
	return commonv1.MatchingMode_MATCHING_MODE_STANDARD
}

func categoryName(c commonv1.ReportCategory) string {
	switch c {
	case commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION:
		return orderfill.CategoryNeedsDecision
	case commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_SOURCE:
		return orderfill.CategoryNotInSource
	case commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME:
		return orderfill.CategoryCheckNameOrVolume
	case commonv1.ReportCategory_REPORT_CATEGORY_TO_ORDER:
		return orderfill.CategoryToOrder
	case commonv1.ReportCategory_REPORT_CATEGORY_ORDER_NOT_NEEDED:
		return orderfill.CategoryOrderNotNeeded
	case commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK:
		return orderfill.CategoryNotInBlank
	default:
		return ""
	}
}
