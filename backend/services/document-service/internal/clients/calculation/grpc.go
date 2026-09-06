package calculation

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	calculationv1 "order-fill/backend/proto/gen/go/orderfill/calculation/v1"
	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

type GRPC struct {
	client calculationv1.CalculationServiceClient
}

func Dial(ctx context.Context, addr string) (*GRPC, error) {
	conn, err := grpcutil.Dial(ctx, addr)
	if err != nil {
		return nil, err
	}
	return &GRPC{client: calculationv1.NewCalculationServiceClient(conn)}, nil
}

func (c *GRPC) Adjust(ctx context.Context, brandKey string, recommended float64) (float64, error) {
	got, err := c.AdjustQuantity(ctx, recommended, 0, false, "", brand.RuleConfig{Key: brandKey})
	if err != nil {
		return 0, err
	}
	if got.Inserted == nil {
		return 0, nil
	}
	return *got.Inserted, nil
}

func (c *GRPC) AdjustQuantity(ctx context.Context, recommended, orderedFact float64, hasFact bool, boxSize string, rule brand.RuleConfig) (brand.AdjustedQuantity, error) {
	resp, err := c.client.CalculateAdjustedQuantity(ctx, &calculationv1.CalculateAdjustedQuantityRequest{
		Brand:                   rule.Key,
		RecommendedQty:          recommended,
		OrderedFact:             orderedFact,
		HasOrderedFact:          hasFact,
		BoxSize:                 boxSize,
		Adjustment:              string(rule.Adjustment),
		QuantityMultiple:        int32(rule.Multiple),
		AdjustmentComment:       rule.AdjustmentComment,
		AllowSmallPositiveOrder: rule.AllowSmallPositiveOrder,
	})
	if err != nil {
		return brand.AdjustedQuantity{}, err
	}
	out := brand.AdjustedQuantity{Rounded: int(resp.GetRounded()), AutoComment: resp.GetAutoComment(), BoxAdjusted: resp.GetBoxAdjusted()}
	if resp.GetInserted() {
		qty := resp.GetQty()
		out.Inserted = &qty
	}
	return out, nil
}

func (c *GRPC) Recommend(ctx context.Context, brandKey, cityRule string, weeks float64, rows []orderfill.RecommendRow) ([]orderfill.RecommendRow, error) {
	req := &calculationv1.CalculateOrderRecommendationsRequest{
		Brand: brandKey, DeliveryWeeks: weeks, CityRule: cityRule,
	}
	for _, row := range rows {
		req.Rows = append(req.Rows, &calculationv1.OrderRow{
			Id: row.ID, Revenue: row.Revenue, Stock: row.Stock, InTransit: row.InTransit,
			MonthlySales: row.MonthlySales, AbcCategory: row.Category,
		})
	}
	resp, err := c.client.CalculateOrderRecommendations(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]orderfill.RecommendRow, 0, len(resp.GetRows()))
	for _, row := range resp.GetRows() {
		out = append(out, orderfill.RecommendRow{
			ID: row.GetId(), Category: row.GetAbcCategory(),
			Recommended: row.GetRecommendedQty(), TargetStock: row.GetTargetStock(),
			RevenuePercent: row.GetRevenuePercent(), CumulativePercent: row.GetCumulativePercent(),
			AverageMonthly: row.GetAverageMonthly(), TotalQuantity: row.GetTotalQuantity(),
		})
	}
	return out, nil
}

func (c *GRPC) NorthPlan(ctx context.Context, brand string, needs []NorthNeed, stock []TyumenStock) ([]NorthRow, error) {
	req := &calculationv1.CalculateNorthPlanRequest{Brand: brand}
	for _, n := range needs {
		req.Needs = append(req.Needs, &calculationv1.NorthCityNeed{City: n.City, Article: n.Article, Qty: n.Qty})
	}
	for _, row := range stock {
		req.TyumenStock = append(req.TyumenStock, &calculationv1.OrderRow{
			Article: row.Article, Name: row.Name, Stock: row.Stock, InTransit: row.InTransit, TargetStock: row.Target,
		})
	}
	resp, err := c.client.CalculateNorthPlan(ctx, req)
	if err != nil {
		return nil, err
	}
	out := make([]NorthRow, 0, len(resp.GetRows()))
	for _, row := range resp.GetRows() {
		out = append(out, NorthRow{
			Article: row.GetArticle(), Name: row.GetName(), Comment: row.GetComment(),
			TyumenQty: row.GetTyumenQty(), TransferQty: row.GetTransferQty(), SupplierQty: row.GetSupplierQty(),
		})
	}
	return out, nil
}
