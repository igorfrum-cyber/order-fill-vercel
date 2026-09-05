package calculation

import (
	"context"

	"order-fill/backend/pkg/grpcutil"
	calculationv1 "order-fill/backend/proto/gen/go/orderfill/calculation/v1"
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

func (c *GRPC) Adjust(ctx context.Context, brand string, recommended float64) (float64, error) {
	resp, err := c.client.CalculateAdjustedQuantity(ctx, &calculationv1.CalculateAdjustedQuantityRequest{
		Brand: brand, RecommendedQty: recommended,
	})
	if err != nil {
		return 0, err
	}
	return resp.GetQty(), nil
}

func (c *GRPC) NorthPlan(ctx context.Context, brand string, needs []NorthNeed, stock []TyumenStock) ([]NorthRow, error) {
	req := &calculationv1.CalculateNorthPlanRequest{Brand: brand}
	for _, n := range needs {
		req.Needs = append(req.Needs, &calculationv1.NorthCityNeed{City: n.City, Article: n.Article, Qty: n.Qty})
	}
	for _, row := range stock {
		req.TyumenStock = append(req.TyumenStock, &calculationv1.OrderRow{
			Article: row.Article, Name: row.Name, Stock: row.Stock, InTransit: row.InTransit,
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
