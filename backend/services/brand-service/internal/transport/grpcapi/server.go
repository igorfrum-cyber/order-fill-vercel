package grpcapi

import (
	"context"

	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	brandv1 "order-fill/backend/proto/gen/go/orderfill/brand/v1"
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
	p := s.svc.GetPolicy(ctx, req.GetBrand(), req.GetVariant())
	return &brandv1.GetBrandPolicyResponse{Policy: &brandv1.BrandPolicy{
		Brand: p.Key, Variant: p.Variant, QuantityMultiple: int32(p.Multiple), MinQuantity: int32(p.MinQuantity),
	}}, nil
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
