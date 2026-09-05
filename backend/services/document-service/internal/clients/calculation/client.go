package calculation

import "context"

type NorthNeed struct {
	City, Article, Name string
	Qty                 float64
}

type TyumenStock struct {
	Article, Name            string
	Stock, InTransit, Target float64
}

type NorthRow struct {
	Article, Name, Comment              string
	TyumenQty, TransferQty, SupplierQty float64
}

type Client interface {
	Adjust(ctx context.Context, brand string, recommended float64) (float64, error)
	NorthPlan(ctx context.Context, brand string, needs []NorthNeed, stock []TyumenStock) ([]NorthRow, error)
}
