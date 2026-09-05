package grpcapi

import (
	"context"

	"google.golang.org/grpc"

	"order-fill/backend/pkg/grpcutil"
	calculationv1 "order-fill/backend/proto/gen/go/orderfill/calculation/v1"
	"order-fill/backend/services/calculation-service/internal/domain"
	"order-fill/backend/services/calculation-service/internal/service/calculation"
)

type Server struct {
	calculationv1.UnimplementedCalculationServiceServer
	svc *calculation.Service
}

func NewServer(svc *calculation.Service) *Server { return &Server{svc: svc} }

func New(handler calculationv1.CalculationServiceServer) *grpc.Server {
	s := grpcutil.NewServer()
	if handler != nil {
		calculationv1.RegisterCalculationServiceServer(s, handler)
	}
	return s
}

func protoRow(row *calculationv1.OrderRow) domain.OrderRow {
	if row == nil {
		return domain.OrderRow{}
	}
	return domain.OrderRow{
		ID: row.GetId(), Article: row.GetArticle(), Name: row.GetName(),
		Revenue: row.GetRevenue(), Stock: row.GetStock(), InTransit: row.GetInTransit(),
		MonthlySales: row.GetMonthlySales(), Recommended: row.GetRecommendedQty(), ABCCategory: row.GetAbcCategory(),
	}
}

func toProtoRow(row domain.OrderRow) *calculationv1.OrderRow {
	return &calculationv1.OrderRow{
		Id: row.ID, Article: row.Article, Name: row.Name, Revenue: row.Revenue,
		Stock: row.Stock, InTransit: row.InTransit, MonthlySales: row.MonthlySales,
		RecommendedQty: row.Recommended, AbcCategory: row.ABCCategory,
	}
}

func (s *Server) CalculateOrderRecommendations(_ context.Context, req *calculationv1.CalculateOrderRecommendationsRequest) (*calculationv1.CalculateOrderRecommendationsResponse, error) {
	in := make([]domain.OrderRow, 0, len(req.GetRows()))
	for _, row := range req.GetRows() {
		in = append(in, protoRow(row))
	}
	out := s.svc.Recommend(req.GetBrand(), in)
	rows := make([]*calculationv1.OrderRow, 0, len(out))
	for _, row := range out {
		rows = append(rows, toProtoRow(row))
	}
	return &calculationv1.CalculateOrderRecommendationsResponse{Rows: rows}, nil
}

func (s *Server) CalculateAdjustedQuantity(_ context.Context, req *calculationv1.CalculateAdjustedQuantityRequest) (*calculationv1.CalculateAdjustedQuantityResponse, error) {
	hasFact := req.GetOrderedFact() != 0
	adj := calculation.AdjustQuantity(req.GetRecommendedQty(), req.GetBrand(), req.GetOrderedFact(), hasFact, "")
	qty := float64(adj.Rounded)
	if adj.Inserted != nil {
		qty = *adj.Inserted
	}
	return &calculationv1.CalculateAdjustedQuantityResponse{Qty: qty}, nil
}

func (s *Server) CalculateNorthPlan(_ context.Context, req *calculationv1.CalculateNorthPlanRequest) (*calculationv1.CalculateNorthPlanResponse, error) {
	needs := make([]domain.CityNeed, 0, len(req.GetNeeds()))
	for _, n := range req.GetNeeds() {
		needs = append(needs, domain.CityNeed{City: n.GetCity(), Article: n.GetArticle(), Qty: n.GetQty()})
	}
	stock := make([]domain.OrderRow, 0, len(req.GetTyumenStock()))
	for _, row := range req.GetTyumenStock() {
		stock = append(stock, protoRow(row))
	}
	plan := s.svc.NorthPlan(req.GetBrand(), needs, stock)
	out := make([]*calculationv1.NorthPlanRow, 0, len(plan))
	for _, row := range plan {
		out = append(out, &calculationv1.NorthPlanRow{
			Article: row.Article, Name: row.Name, TyumenQty: row.TyumenQty,
			TransferQty: row.TransferQty, SupplierQty: row.SupplierQty, Comment: row.Comment,
		})
	}
	return &calculationv1.CalculateNorthPlanResponse{Rows: out}, nil
}

func (s *Server) RecalculateNorthRow(_ context.Context, req *calculationv1.RecalculateNorthRowRequest) (*calculationv1.RecalculateNorthRowResponse, error) {
	in := req.GetRow()
	row := domain.PlanRow{}
	if in != nil {
		row = domain.PlanRow{
			Article: in.GetArticle(), Name: in.GetName(), TyumenQty: in.GetTyumenQty(),
			TransferQty: in.GetTransferQty(), SupplierQty: in.GetSupplierQty(), Comment: in.GetComment(),
		}
	}
	got := s.svc.RecalculateNorthRow(req.GetBrand(), row, req.GetEditedQty())
	return &calculationv1.RecalculateNorthRowResponse{Row: &calculationv1.NorthPlanRow{
		Article: got.Article, Name: got.Name, TyumenQty: got.TyumenQty,
		TransferQty: got.TransferQty, SupplierQty: got.SupplierQty, Comment: got.Comment,
	}}, nil
}

func (s *Server) ValidateManualEdits(_ context.Context, req *calculationv1.ValidateManualEditsRequest) (*calculationv1.ValidateManualEditsResponse, error) {
	edits := make([]calculation.ManualEdit, 0, len(req.GetEdits()))
	for _, e := range req.GetEdits() {
		edits = append(edits, calculation.ManualEdit{RowID: e.GetRowId(), Qty: e.GetQty(), Comment: e.GetComment()})
	}
	rows := make([]domain.OrderRow, 0, len(req.GetRows()))
	for _, row := range req.GetRows() {
		rows = append(rows, protoRow(row))
	}
	ok, blocking := s.svc.ValidateManualEdits(edits, rows)
	return &calculationv1.ValidateManualEditsResponse{Ok: ok, BlockingRowIds: blocking}, nil
}
