package grpcapi

import (
	"context"

	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	commonv1 "order-fill/backend/proto/gen/go/orderfill/common/v1"
	matchingv1 "order-fill/backend/proto/gen/go/orderfill/matching/v1"
	"order-fill/backend/services/matching-service/internal/domain"
	"order-fill/backend/services/matching-service/internal/service/matching"
)

type Server struct {
	matchingv1.UnimplementedMatchingServiceServer
	svc *matching.Service
}

func NewServer(svc *matching.Service) *Server { return &Server{svc: svc} }

func New(handler matchingv1.MatchingServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		matchingv1.RegisterMatchingServiceServer(s, handler)
	}
	return s
}

func protoItem(item *matchingv1.Item) domain.Item {
	if item == nil {
		return domain.Item{}
	}
	return domain.Item{
		ID: item.GetId(), Article: item.GetArticle(), Name: item.GetName(),
		Volume: item.GetVolume(), Form: item.GetForm(), ChestnyZnak: item.GetChestnyZnak(),
	}
}

func protoCategory(c domain.Category) commonv1.ReportCategory {
	switch c {
	case domain.CategoryNeedsDecision:
		return commonv1.ReportCategory_REPORT_CATEGORY_NEEDS_DECISION
	case domain.CategoryNotInSource:
		return commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_SOURCE
	case domain.CategoryCheckNameOrVolume:
		return commonv1.ReportCategory_REPORT_CATEGORY_CHECK_NAME_OR_VOLUME
	case domain.CategoryNotInBlank:
		return commonv1.ReportCategory_REPORT_CATEGORY_NOT_IN_BLANK
	case domain.CategoryToOrder:
		return commonv1.ReportCategory_REPORT_CATEGORY_TO_ORDER
	case domain.CategoryOrderNotNeeded:
		return commonv1.ReportCategory_REPORT_CATEGORY_ORDER_NOT_NEEDED
	default:
		return commonv1.ReportCategory_REPORT_CATEGORY_UNSPECIFIED
	}
}

func (s *Server) MatchRows(_ context.Context, req *matchingv1.MatchRowsRequest) (*matchingv1.MatchRowsResponse, error) {
	blank := make([]domain.Item, 0, len(req.GetBlankItems()))
	for _, item := range req.GetBlankItems() {
		blank = append(blank, protoItem(item))
	}
	source := make([]domain.Item, 0, len(req.GetSourceItems()))
	for _, item := range req.GetSourceItems() {
		source = append(source, protoItem(item))
	}
	mode := domain.ModeStandard
	if req.GetMatchingMode() == commonv1.MatchingMode_MATCHING_MODE_SMART {
		mode = domain.ModeSmart
	}
	results := s.svc.Match(blank, source, matching.Options{Mode: mode})
	out := make([]*matchingv1.MatchResult, 0, len(results))
	for _, r := range results {
		out = append(out, &matchingv1.MatchResult{
			BlankItemId: r.BlankItemID, SourceItemId: r.SourceItemID,
			Category: protoCategory(r.Category), Score: r.Score, CandidateIds: r.CandidateIDs,
			Reasons: &matchingv1.MatchReasons{
				Article: r.Reasons.Article, Name: r.Reasons.Name, Volume: r.Reasons.Volume,
				Form: r.Reasons.Form, Duplicates: r.Reasons.Duplicates, Source: r.Reasons.Source,
			},
		})
	}
	return &matchingv1.MatchRowsResponse{Results: out}, nil
}

func (s *Server) NormalizeArticle(_ context.Context, req *matchingv1.NormalizeArticleRequest) (*matchingv1.NormalizeArticleResponse, error) {
	return &matchingv1.NormalizeArticleResponse{Normalized: s.svc.NormalizeArticle(req.GetArticle(), false)}, nil
}

func (s *Server) NormalizeName(_ context.Context, req *matchingv1.NormalizeNameRequest) (*matchingv1.NormalizeNameResponse, error) {
	return &matchingv1.NormalizeNameResponse{Normalized: s.svc.NormalizeName(req.GetName())}, nil
}
