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
		}
		planned := recalculateNorthRow(row, cities, brand)
		out = append(out, planned)
	}
	return out
}

func (s *Service) RecalculateNorthRow(brand string, row domain.PlanRow, editedQty float64) domain.PlanRow {
	cities := map[string]float64{"surgut": editedQty}
	return recalculateNorthRow(row, cities, brand)
}

func recalculateNorthRow(row domain.PlanRow, cities map[string]float64, brand string) domain.PlanRow {
	tyumenPlanned := cities["tyumen"]
	freeLeft := math.Max(0, row.TyumenStock+row.TyumenTransit+tyumenPlanned-row.TyumenTarget)
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
		fromTyumenPart := math.Min(qty, freeLeft)
		fromSupplierPart := qty - fromTyumenPart
		freeLeft -= fromTyumenPart
		fromTyumen += fromTyumenPart
		supplierNorth += fromSupplierPart
		if fromTyumenPart > 0 {
			commentParts = append(commentParts, fmt.Sprintf("Отправить в %s: %s", northLabels[city], formatQty(fromTyumenPart)))
		}
	}
	row.TransferQty = roundTo2(fromTyumen)
	demand := roundTo2(math.Max(0, tyumenPlanned) + supplierNorth)
	unit := row.UnitSize
	if unit <= 1 {
		row.SupplierQty = defaultNorthActual(brand, demand, row.NovacutanMin)
	} else {
		row.SupplierQty = defaultNorthActual(brand, math.Ceil(demand/unit), row.NovacutanMin)
	}
	row.TyumenQty = roundTo2(math.Max(0, tyumenPlanned))
	if row.SupplierQty > 0 {
		commentParts = append([]string{fmt.Sprintf("Заказать у поставщика: %s", formatQty(row.SupplierQty))}, commentParts...)
	}
	if northNeed > 0 && row.SupplierQty == 0 && fromTyumen > 0 {
		commentParts = append(commentParts, "Закрывается остатком Тюмени")
	}
	row.Comment = strings.Join(commentParts, "\n")
	_ = northNeed
	return row
}

func defaultNorthActual(brand string, supplierNeed, novacutanMin float64) float64 {
	if supplierNeed <= 0 {
		return 0
	}
	if brand == "klapp" {
		v, ok := nearestMultipleValue(supplierNeed, 3)
		if ok {
			return v
		}
	}
	if brand == "novacutan" {
		minimum := novacutanMin
		if minimum <= 0 {
			minimum = 100
		}
		return math.Round(math.Max(supplierNeed, minimum)/10) * 10
	}
	return roundTo2(supplierNeed)
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
