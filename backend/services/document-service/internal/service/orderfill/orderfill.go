package orderfill

import (
	"context"
	"fmt"

	"order-fill/backend/services/document-service/internal/clients/brand"
	"order-fill/backend/services/document-service/internal/clients/calculation"
	"order-fill/backend/services/document-service/internal/clients/matching"
)

type FillRequest struct {
	NomenclatureGroup string
	BlankFileName     string
	MatchingMode      string
	Blank             []matching.Item
	Source            []matching.Item
	Recommended       map[string]float64
}

type FillCell struct {
	BlankItemID string
	Qty         float64
	Category    string
}

type Processor struct {
	brands brand.Client
	match  matching.Client
	calc   calculation.Client
}

func New(brands brand.Client, match matching.Client, calc calculation.Client) *Processor {
	return &Processor{brands: brands, match: match, calc: calc}
}

func (p *Processor) Fill(ctx context.Context, req FillRequest) (brandKey string, cells []FillCell, err error) {
	brandKey, _, err = p.brands.Detect(ctx, req.NomenclatureGroup, req.BlankFileName)
	if err != nil {
		return "", nil, err
	}
	if err := PlanOneBlank(brandKey, []string{req.BlankFileName}); err != nil {
		return "", nil, err
	}
	results, err := p.match.Match(ctx, req.MatchingMode, req.Blank, req.Source)
	if err != nil {
		return "", nil, err
	}
	cells = make([]FillCell, 0, len(results))
	for _, result := range results {
		qty := 0.0
		if rec, ok := req.Recommended[result.SourceID]; ok {
			qty, err = p.calc.Adjust(ctx, brandKey, rec)
			if err != nil {
				return "", nil, err
			}
		}
		cells = append(cells, FillCell{BlankItemID: result.BlankID, Qty: qty, Category: result.Category})
	}
	return brandKey, cells, nil
}

func PlanOneBlank(brandKey string, names []string) error {
	if len(names) != 1 {
		return fmt.Errorf("для %s нужен один бланк поставщика", brandKey)
	}
	return nil
}
