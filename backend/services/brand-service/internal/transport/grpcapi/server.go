package grpcapi

import (
	"context"
	"errors"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"order-fill/backend/pkg/grpcutil"
	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
	"order-fill/backend/services/brand-service/internal/domain"
	"order-fill/backend/services/brand-service/internal/service/brands"
)

type Server struct {
	brandv1.UnimplementedBrandServiceServer
	svc *brands.Service
}

func NewServer(svc *brands.Service) *Server { return &Server{svc: svc} }

func New(handler brandv1.BrandServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		brandv1.RegisterBrandServiceServer(s, handler)
	}
	return s
}

func (s *Server) GetBrandPolicy(ctx context.Context, req *brandv1.GetBrandPolicyRequest) (*brandv1.GetBrandPolicyResponse, error) {
	p, err := s.svc.GetPolicy(ctx, req.GetBrand(), req.GetVariant())
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to read brand policy")
	}
	return &brandv1.GetBrandPolicyResponse{Policy: protoPolicy(p)}, nil
}

func protoPolicy(p domain.Policy) *brandv1.BrandPolicy {
	policy := &brandv1.BrandPolicy{
		Brand:                   p.Key,
		Variant:                 p.Variant,
		QuantityMultiple:        int32Clamp(p.Multiple),
		MinQuantity:             int32Clamp(p.MinQuantity),
		Label:                   p.Label,
		Adjustment:              string(p.Adjustment),
		AdjustmentLabel:         p.AdjustmentLabel,
		AdjustmentComment:       p.AdjustmentComment,
		PreserveHyphen:          p.PreserveArticleHyphen,
		PrefixAliases:           p.ArticlePrefixAliases,
		BlankQuantityHeader:     p.BlankQuantityHeader,
		BlankBoxHeader:          p.BlankBoxHeader,
		BlankLayout:             p.BlankLayout,
		AllowSmallPositiveOrder: p.AllowSmallPositiveOrder,
	}
	if p.RequireUnit != nil {
		policy.RequireUnit = p.RequireUnit
	}
	return policy
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

func (s *Server) ListBrands(ctx context.Context, _ *brandv1.ListBrandsRequest) (*brandv1.ListBrandsResponse, error) {
	return &brandv1.ListBrandsResponse{Brands: s.svc.List(ctx)}, nil
}

func (s *Server) DetectBrand(_ context.Context, req *brandv1.DetectBrandRequest) (*brandv1.DetectBrandResponse, error) {
	brand, variant, ok := brands.Detect(req.GetNomenclatureGroup(), req.GetFileName())
	if !ok {
		return &brandv1.DetectBrandResponse{}, nil
	}
	return &brandv1.DetectBrandResponse{Brand: brand, Variant: variant}, nil
}

func (s *Server) UpdateBrandPolicy(ctx context.Context, req *brandv1.UpdateBrandPolicyRequest) (*brandv1.UpdateBrandPolicyResponse, error) {
	if grpcutil.ActorRole(ctx) != "platform_admin" {
		return nil, status.Error(codes.PermissionDenied, "brand policy update is not allowed")
	}
	if req.GetPolicy() == nil || req.GetMeta().GetActorUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "policy and actor are required")
	}
	updated, err := s.svc.UpdatePolicy(ctx, domainPolicy(req.GetPolicy()), req.GetMeta().GetActorUserId())
	if errors.Is(err, domain.ErrInvalidPolicy) {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to save brand policy")
	}
	return &brandv1.UpdateBrandPolicyResponse{Policy: protoPolicy(updated)}, nil
}

func domainPolicy(p *brandv1.BrandPolicy) domain.Policy {
	return domain.Policy{
		Key: p.GetBrand(), Label: p.GetLabel(), Variant: p.GetVariant(), Adjustment: domain.Adjustment(p.GetAdjustment()),
		Multiple: int(p.GetQuantityMultiple()), MinQuantity: int(p.GetMinQuantity()), AdjustmentLabel: p.GetAdjustmentLabel(),
		AdjustmentComment: p.GetAdjustmentComment(), PreserveArticleHyphen: p.GetPreserveHyphen(), ArticlePrefixAliases: p.GetPrefixAliases(),
		BlankQuantityHeader: p.GetBlankQuantityHeader(), BlankBoxHeader: p.GetBlankBoxHeader(), BlankLayout: p.GetBlankLayout(),
		AllowSmallPositiveOrder: p.GetAllowSmallPositiveOrder(), RequireUnit: p.RequireUnit,
	}
}
