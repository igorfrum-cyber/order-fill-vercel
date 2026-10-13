package north

import (
	"fmt"
	"math"
	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

func specializedNeeds(detection orderfill.Detection, rule brand.RuleConfig, city, variant string) ([]Need, error) {
	out := []Need{}
	for _, position := range orderfill.SpecializedBlankPositions(detection, rule) {
		raw := detection.Sheet.Value(position.Row, position.QuantityColumn)
		qty, ok := normalize.ParseNumber(raw)
		if raw == "" {
			qty, ok = 0, true
		}
		if !ok || qty < 0 || math.IsNaN(qty) || math.IsInf(qty, 0) {
			return nil, fmt.Errorf("%w: в строке %d указано неверное количество", orderfill.ErrInvalidInput, position.Row)
		}
		if rule.Key == "novacutan" {
			qty *= max(1, position.SupplierUnitSize)
		}
		key := position.Key
		if variant != "" {
			key = variant + ":" + key
		}
		out = append(out, Need{Row: position.Row, QuantityColumn: position.QuantityColumn, PriceColumn: position.PriceColumn, Prices: []orderfill.BudgetPrice{{Column: position.PriceColumn, Label: detection.Sheet.Value(detection.HeaderRow, position.PriceColumn), Price: position.Price}}, City: city, Article: key, BaseArticle: position.Key, ArticleRaw: position.ArticleRaw, Name: position.Name, Variant: variant, Qty: qty, Price: position.Price, Minimum: position.Minimum, SupplierUnitSize: max(1, position.SupplierUnitSize), BoxSize: position.BoxSize, HasBoxSize: position.HasBoxSize})
	}
	return out, nil
}

func writeSpecializedQuantities(detection orderfill.Detection, rule brand.RuleConfig, variant string, lines []SupplierLine) error {
	quantities := map[string]float64{}
	for _, line := range lines {
		if line.Quantity > 0 {
			quantities[line.Key] = line.Quantity
		}
	}
	for _, position := range orderfill.SpecializedBlankPositions(detection, rule) {
		key := position.Key
		if variant != "" {
			key = variant + ":" + key
		}
		if value := quantities[key]; value > 0 {
			detection.Sheet.SetNumber(position.Row, position.QuantityColumn, value)
		} else {
			detection.Sheet.ClearValue(position.Row, position.QuantityColumn)
		}
	}
	return nil
}

func discountSpecializedPrices(detection orderfill.Detection, rule brand.RuleConfig, discount float64) error {
	rows := []orderfill.ReportRow{}
	entries := []orderfill.BudgetPricing{}
	for _, position := range orderfill.SpecializedBlankPositions(detection, rule) {
		base, ok := orderfill.BudgetCellPrice(detection.Sheet, position.Row, position.PriceColumn)
		if !ok || base <= 0 {
			continue
		}
		qty, _ := normalize.ParseNumber(detection.Sheet.Value(position.Row, position.QuantityColumn))
		key := fmt.Sprintf("north:%d:%d", position.Row, position.QuantityColumn)
		rows = append(rows, orderfill.ReportRow{Key: key, BlankRow: position.Row, BlankQuantityColumn: position.QuantityColumn, Inserted: new(qty), BlankName: position.Name})
		price := math.Floor(base*(1-discount/100)*100+0.5) / 100
		entries = append(entries, orderfill.BudgetPricing{RowKey: key, PriceColumn: position.PriceColumn, BasePrice: base, Discount: discount, Price: price})
	}
	return orderfill.ApplyBudgetPricing(detection.Sheet, detection.HeaderRow, rule.Key, rows, entries)
}
