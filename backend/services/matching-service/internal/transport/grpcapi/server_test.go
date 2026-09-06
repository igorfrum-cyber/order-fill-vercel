package grpcapi

import (
	"testing"

	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	matchingv1 "order-fill/backend/proto/gen/go/orderfill/matching/v1"
	"order-fill/backend/services/matching-service/internal/service/matching"
)

func TestMatchRowsPassesPrefixAliases(t *testing.T) {
	t.Parallel()
	s := NewServer(matching.New())
	resp, err := s.MatchRows(t.Context(), &matchingv1.MatchRowsRequest{
		PrefixAliases: []string{"MT"},
		BlankItems:    []*matchingv1.Item{{Id: "b1", Article: "123", Name: "Cream 50 ml"}},
		SourceItems:   []*matchingv1.Item{{Id: "s1", Article: "MT123", Name: "Cream 50 ml"}},
	})
	if err != nil {
		t.Fatalf("MatchRows: %v", err)
	}
	if len(resp.GetResults()) != 1 {
		t.Fatalf("results = %d, want 1", len(resp.GetResults()))
	}
	got := resp.GetResults()[0]
	if got.GetSourceItemId() != "s1" {
		t.Fatalf("source = %q, want s1", got.GetSourceItemId())
	}
	if got.GetCategory() != commonv1.ReportCategory_REPORT_CATEGORY_TO_ORDER {
		t.Fatalf("category = %v", got.GetCategory())
	}
	if got.GetReasons().GetArticle() != "alias" {
		t.Fatalf("article reason = %q, want alias", got.GetReasons().GetArticle())
	}
}

func TestMatchRowsPassesSmartMode(t *testing.T) {
	t.Parallel()
	s := NewServer(matching.New())
	resp, err := s.MatchRows(t.Context(), &matchingv1.MatchRowsRequest{
		MatchingMode: commonv1.MatchingMode_MATCHING_MODE_SMART,
		BlankItems:   []*matchingv1.Item{{Id: "b1", Article: "A1", Name: "Cream"}},
		SourceItems: []*matchingv1.Item{
			{Id: "s1", Article: "A1", Name: "Cream"},
			{Id: "s2", Article: "A1", Name: "Cream"},
		},
	})
	if err != nil {
		t.Fatalf("MatchRows: %v", err)
	}
	if got := resp.GetResults()[0].GetCategory(); got != commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION {
		t.Fatalf("category = %v, want needs_decision", got)
	}
}

func TestNormalizeArticlePreservesHyphen(t *testing.T) {
	t.Parallel()
	s := NewServer(matching.New())
	resp, err := s.NormalizeArticle(t.Context(), &matchingv1.NormalizeArticleRequest{
		Article:        "AB-12",
		PreserveHyphen: true,
	})
	if err != nil {
		t.Fatalf("NormalizeArticle: %v", err)
	}
	if got := resp.GetNormalized(); got != "AB-12" {
		t.Fatalf("normalized = %q, want AB-12", got)
	}
}

func TestMergeChestnyZnakJoinsClone(t *testing.T) {
	t.Parallel()
	s := NewServer(matching.New())
	resp, err := s.MergeChestnyZnak(t.Context(), &matchingv1.MergeChestnyZnakRequest{
		Items: []*matchingv1.Item{
			{Id: "5", Article: "AA04", Name: "АН Сыворотка 30 мл"},
			{Id: "6", Article: "AA04", Name: "ЧЗ АН Сыворотка 30 мл"},
		},
	})
	if err != nil {
		t.Fatalf("MergeChestnyZnak: %v", err)
	}
	if len(resp.GetMerges()) != 1 || resp.GetMerges()[0].GetTargetId() != "5" || resp.GetMerges()[0].GetCloneIds()[0] != "6" {
		t.Fatalf("%+v", resp.GetMerges())
	}
}
