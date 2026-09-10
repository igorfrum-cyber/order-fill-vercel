package north

import (
	"context"

	"order-fill/backend/services/document-service/internal/clients/calculation"
)

type Processor struct {
	calc calculation.Client
}

func New(calc calculation.Client) *Processor { return &Processor{calc: calc} }

func (p *Processor) Plan(ctx context.Context, brand string, needs []calculation.NorthNeed, stock []calculation.TyumenStock) ([]calculation.NorthRow, error) {
	return p.calc.NorthPlan(ctx, brand, needs, stock)
}
