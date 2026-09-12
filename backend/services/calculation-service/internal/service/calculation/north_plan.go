package calculation

import (
	"fmt"
	"math"
	"strings"

	"order-fill/backend/services/calculation-service/internal/domain"
)

var northAllocation = []string{"nizhnevartovsk", "urengoy", "surgut"}

var northLabels = map[string]string{
	"tyumen": "Тюмень", "surgut": "Сургут", "nizhnevartovsk": "Вартовск", "urengoy": "Уренгой",
}

func (s *Service) NorthPlan(brand string, needs []domain.CityNeed, tyumen []domain.OrderRow) []domain.PlanRow {
	stock := map[string]domain.OrderRow{}
	for _, row := range tyumen {
		stock[row.Article] = row
	}
	byArticle := map[string]map[string]float64{}
	names := map[string]string{}
	for _, need := range needs {
		if byArticle[need.Article] == nil {
			byArticle[need.Article] = map[string]float64{}
		}
		byArticle[need.Article][need.City] += need.Qty
		if names[need.Article] == "" {
			names[need.Article] = need.Article
		}
	}
	out := make([]domain.PlanRow, 0, len(byArticle))
	for article, cities := range byArticle {
		row := domain.PlanRow{Article: article, Name: names[article], UnitSize: 1, NovacutanMin: 100}
		if src, ok := stock[article]; ok {
			row.TyumenStock = src.Stock
			row.TyumenTransit = src.InTransit
			row.TyumenTarget = src.TargetStock
			row.Name = src.Name
			row.BoxSize = src.BoxSize
			row.HasBoxSize = src.HasBoxSize
			row.WarehouseStock = src.WarehouseStock
			row.WarehouseTransit = src.WarehouseTransit
			row.HasWarehouseStock = src.HasWarehouseStock
		}
		if brand == "novacutan" {
			row.UnitSize = NovacutanSupplierUnitSize(row.Name)
			row.NovacutanMin = NovacutanMinimumQuantity(row.Name)
		}
		out = append(out, recalculateNorthRow(row, cities, brand))
	}
	return out
}

func (s *Service) RecalculateNorthRow(brand string, row domain.PlanRow, editedQty float64) domain.PlanRow {
	cities := map[string]float64{"surgut": editedQty}
	return recalculateNorthRow(row, cities, brand)
}

func recalculateNorthRow(row domain.PlanRow, cities map[string]float64, brand string) domain.PlanRow {
	tyumenPlanned := cities["tyumen"]
	freeLeft := NorthTyumenFreeStock(row.TyumenStock, row.TyumenTransit, tyumenPlanned, row.TyumenTarget, row.WarehouseStock, row.WarehouseTransit, row.HasWarehouseStock)
	fromTyumen := 0.0
	supplierNorth := 0.0
	northNeed := 0.0
	var commentParts []string
	for _, city := range northAllocation {
		qty := cities[city]
		if qty <= 0 {
			continue
		}
		northNeed += qty
		fromTyumenPart := min(qty, freeLeft)
		fromSupplierPart := qty - fromTyumenPart
		freeLeft -= fromTyumenPart
		fromTyumen += fromTyumenPart
		supplierNorth += fromSupplierPart
		if fromTyumenPart > 0 {
			commentParts = append(commentParts, fmt.Sprintf("Отправить в %s: %s", northLabels[city], formatQty(fromTyumenPart)))
		}
	}
	row.TransferQty = roundTo2(fromTyumen)
	demand := roundTo2(max(0, tyumenPlanned) + supplierNorth)
	unit := row.UnitSize
	if unit == 0 || math.IsNaN(unit) {
		unit = 1
	}
	need := demand
	if unit > 1 {
		need = math.Ceil(demand / unit)
	}
	row.SupplierQty = DefaultNorthActualSupplierOrder(brand, row.Variant, domain.NorthSupplierPosition{
		BlankBoxSize:     row.BoxSize,
		HasBoxSize:       row.HasBoxSize,
		SupplierUnitSize: unit,
		NovacutanMinimum: row.NovacutanMin,
	}, need)
	row.TyumenQty = roundTo2(max(0, tyumenPlanned))
	if row.SupplierQty > 0 {
		commentParts = append([]string{fmt.Sprintf("Заказать у поставщика: %s", formatQty(row.SupplierQty))}, commentParts...)
	}
	if northNeed > 0 && row.SupplierQty == 0 && fromTyumen > 0 {
		commentParts = append(commentParts, "Закрывается остатком Тюмени")
	}
	row.Comment = strings.Join(commentParts, "\n")
	return row
}

func formatQty(value float64) string {
	if value <= 0 {
		return ""
	}
	if math.Trunc(value) == value {
		return strconvI(int(value))
	}
	return fmt.Sprintf("%.2f", value)
}

func strconvI(v int) string {
	return fmt.Sprintf("%d", v)
}
