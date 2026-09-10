package calculation

import (
	"context"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

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
	AdjustQuantity(ctx context.Context, recommended, orderedFact float64, hasFact bool, boxSize string, rule brand.RuleConfig) (brand.AdjustedQuantity, error)
	Recommend(ctx context.Context, brand, cityRule string, weeks float64, rows []orderfill.RecommendRow) ([]orderfill.RecommendRow, error)
	NorthPlan(ctx context.Context, brand string, needs []NorthNeed, stock []TyumenStock) ([]NorthRow, error)
}
