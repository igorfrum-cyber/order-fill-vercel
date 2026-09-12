package grpcapi

import (
	"context"
	"math"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

func toProtoRow(row domain.OrderRow) *calculationv1.OrderRow {
	return &calculationv1.OrderRow{
		Id: row.ID, Article: row.Article, Name: row.Name, Revenue: row.Revenue,
		Stock: row.Stock, InTransit: row.InTransit, MonthlySales: row.MonthlySales,
		RecommendedQty: row.Recommended, AbcCategory: row.ABCCategory,
		TargetStock: row.TargetStock, RevenuePercent: row.RevenuePercent,
		CumulativePercent: row.CumulativePercent, AverageMonthly: row.AverageMonthly,
		TotalQuantity:  row.TotalQuantity,
		WarehouseStock: row.WarehouseStock, WarehouseTransit: row.WarehouseTransit,
		HasWarehouseStock: row.HasWarehouseStock, BoxSize: row.BoxSize, HasBoxSize: row.HasBoxSize,
	}
}

func protoRow(row *calculationv1.OrderRow) domain.OrderRow {
	if row == nil {
		return domain.OrderRow{}
	}
	return domain.OrderRow{
		ID: row.GetId(), Article: row.GetArticle(), Name: row.GetName(),
		Revenue: row.GetRevenue(), Stock: row.GetStock(), InTransit: row.GetInTransit(),
		MonthlySales: row.GetMonthlySales(), Recommended: row.GetRecommendedQty(), ABCCategory: row.GetAbcCategory(),
		TargetStock: row.GetTargetStock(), RevenuePercent: row.GetRevenuePercent(),
		CumulativePercent: row.GetCumulativePercent(), AverageMonthly: row.GetAverageMonthly(),
		TotalQuantity:  row.GetTotalQuantity(),
		WarehouseStock: row.GetWarehouseStock(), WarehouseTransit: row.GetWarehouseTransit(),
		HasWarehouseStock: row.GetHasWarehouseStock(), BoxSize: row.GetBoxSize(), HasBoxSize: row.GetHasBoxSize(),
	}
}

func toProtoPlan(row domain.PlanRow) *calculationv1.NorthPlanRow {
	return &calculationv1.NorthPlanRow{
		Article: row.Article, Name: row.Name, TyumenQty: row.TyumenQty,
		TransferQty: row.TransferQty, SupplierQty: row.SupplierQty, Comment: row.Comment,
		Variant: row.Variant, TyumenStock: row.TyumenStock, TyumenTransit: row.TyumenTransit,
		TyumenTarget: row.TyumenTarget, UnitSize: row.UnitSize, NovacutanMin: row.NovacutanMin,
		BoxSize: row.BoxSize, HasBoxSize: row.HasBoxSize,
		WarehouseStock: row.WarehouseStock, WarehouseTransit: row.WarehouseTransit,
		HasWarehouseStock: row.HasWarehouseStock,
	}
}

func protoPlan(row *calculationv1.NorthPlanRow) domain.PlanRow {
	if row == nil {
		return domain.PlanRow{}
	}
	return domain.PlanRow{
		Article: row.GetArticle(), Name: row.GetName(), TyumenQty: row.GetTyumenQty(),
		TransferQty: row.GetTransferQty(), SupplierQty: row.GetSupplierQty(), Comment: row.GetComment(),
		Variant: row.GetVariant(), TyumenStock: row.GetTyumenStock(), TyumenTransit: row.GetTyumenTransit(),
		TyumenTarget: row.GetTyumenTarget(), UnitSize: row.GetUnitSize(), NovacutanMin: row.GetNovacutanMin(),
		BoxSize: row.GetBoxSize(), HasBoxSize: row.GetHasBoxSize(),
		WarehouseStock: row.GetWarehouseStock(), WarehouseTransit: row.GetWarehouseTransit(),
		HasWarehouseStock: row.GetHasWarehouseStock(),
	}
}

func (s *Server) CalculateOrderRecommendations(_ context.Context, req *calculationv1.CalculateOrderRecommendationsRequest) (*calculationv1.CalculateOrderRecommendationsResponse, error) {
	in := make([]domain.OrderRow, 0, len(req.GetRows()))
	for _, row := range req.GetRows() {
		in = append(in, protoRow(row))
	}
	var out []domain.OrderRow
	switch req.GetCityRule() {
	case "urengoy":
		out = s.svc.RecommendUrengoy(req.GetBrand(), in, req.GetDeliveryWeeks())
	default:
		if req.GetDeliveryWeeks() > 0 {
			out = s.svc.RecommendWithWeeks(req.GetBrand(), in, req.GetDeliveryWeeks())
		} else {
			out = s.svc.Recommend(req.GetBrand(), in)
		}
	}
	rows := make([]*calculationv1.OrderRow, 0, len(out))
	for _, row := range out {
		rows = append(rows, toProtoRow(row))
	}
	return &calculationv1.CalculateOrderRecommendationsResponse{Rows: rows}, nil
}

func (s *Server) CalculateAdjustedQuantity(_ context.Context, req *calculationv1.CalculateAdjustedQuantityRequest) (*calculationv1.CalculateAdjustedQuantityResponse, error) {
	rule := calculation.RuleFromPolicy(req.GetAdjustment(), int(req.GetQuantityMultiple()), req.GetAdjustmentComment(), req.GetAllowSmallPositiveOrder())
	var adj calculation.AdjustedQuantity
	if req.GetAdjustment() != "" {
		adj = calculation.AdjustQuantityWithRule(req.GetRecommendedQty(), rule, req.GetOrderedFact(), req.GetHasOrderedFact(), req.GetBoxSize())
	} else {
		adj = calculation.AdjustQuantity(req.GetRecommendedQty(), req.GetBrand(), req.GetOrderedFact(), req.GetHasOrderedFact(), req.GetBoxSize())
	}
	resp := &calculationv1.CalculateAdjustedQuantityResponse{Rounded: int32Clamp(adj.Rounded), AutoComment: adj.AutoComment, BoxAdjusted: adj.BoxAdjusted}
	if adj.Inserted != nil {
		resp.Inserted = true
		resp.Qty = *adj.Inserted
	}
	return resp, nil
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
		out = append(out, toProtoPlan(row))
	}
	return &calculationv1.CalculateNorthPlanResponse{Rows: out}, nil
}

func (s *Server) RecalculateNorthRow(_ context.Context, req *calculationv1.RecalculateNorthRowRequest) (*calculationv1.RecalculateNorthRowResponse, error) {
	got := s.svc.RecalculateNorthRow(req.GetBrand(), protoPlan(req.GetRow()), req.GetEditedQty())
	return &calculationv1.RecalculateNorthRowResponse{Row: toProtoPlan(got)}, nil
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

func protoBudgetRow(row *calculationv1.BudgetRow) domain.BudgetRow {
	if row == nil {
		return domain.BudgetRow{}
	}
	return domain.BudgetRow{
		Key: row.GetKey(), Name: row.GetName(), Category: row.GetCategory(),
		Quantity: row.GetQuantity(), Price: row.GetPrice(), Demand: row.GetDemand(),
		Delivery: row.GetDelivery(), Stock: row.GetStock(), Transit: row.GetTransit(),
		Outbound: row.GetOutbound(), Unit: row.GetUnit(), Step: row.GetStep(), Minimum: row.GetMinimum(),
		Locked: row.GetLocked(), Excluded: row.GetExcluded(), Unsafe: row.GetUnsafe(),
	}
}

func (s *Server) PlanBudget(_ context.Context, req *calculationv1.PlanBudgetRequest) (*calculationv1.PlanBudgetResponse, error) {
	in := make([]domain.BudgetRow, 0, len(req.GetRows()))
	for _, row := range req.GetRows() {
		in = append(in, protoBudgetRow(row))
	}
	plan, err := calculation.PlanBudget(in, req.GetTarget(), domain.BudgetOptions{
		AllowOverSix: req.GetAllowOverSix(), AllowBelowOne: req.GetAllowBelowOne(),
	})
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	out := make([]*calculationv1.PlannedBudgetRow, 0, len(plan.Rows))
	for _, row := range plan.Rows {
		out = append(out, &calculationv1.PlannedBudgetRow{
			Key: row.Key, Name: row.Name, Before: row.Before, Quantity: row.Quantity,
			Comment: calculation.BudgetChangeComment(row),
		})
	}
	return &calculationv1.PlanBudgetResponse{
		Rows: out, Before: plan.Before, Total: plan.Total, Target: plan.Target,
		Reason: plan.Reason, Complete: plan.Complete,
	}, nil
}

func (s *Server) CalculateWarehouseTransfer(_ context.Context, req *calculationv1.CalculateWarehouseTransferRequest) (*calculationv1.CalculateWarehouseTransferResponse, error) {
	return &calculationv1.CalculateWarehouseTransferResponse{
		Quantity: calculation.WarehouseTransferQuantity(req.GetOfficeStock(), req.GetWarehouseStock()),
		Target:   calculation.WarehouseOfficeTarget(req.GetOfficeStock(), req.GetWarehouseStock()),
	}, nil
}
