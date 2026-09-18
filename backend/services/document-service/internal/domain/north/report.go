package north

import (
	"cmp"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

type Edit struct {
	Key, Value, Comment string
}

type CityQty struct {
	Key      string  `json:"key"`
	Label    string  `json:"label"`
	Quantity float64 `json:"quantity,omitzero"`
}

type ConfirmationGroup struct {
	City     CityQty  `json:"city"`
	Variants []string `json:"variants"`
}

type Planned struct {
	Article, Name, Variant, Comment          string
	TyumenQty, TransferQty, SupplierQty      float64
	TyumenStock, TyumenTransit, TyumenTarget float64
	UnitSize, NovacutanMin, BoxSize          float64
	HasBoxSize                               bool
	WarehouseStock, WarehouseTransit         float64
	HasWarehouseStock                        bool
}

type Transfer struct {
	Article string  `json:"article"`
	Qty     float64 `json:"qty"`
}

type PlanRow struct {
	Key                 string                   `json:"key"`
	Article             string                   `json:"article"`
	ArticleRaw          string                   `json:"articleRaw,omitempty"`
	Name                string                   `json:"name"`
	Cities              []CityQty                `json:"cities"`
	SupplierParts       []CityQty                `json:"supplierParts"`
	TyumenParts         []CityQty                `json:"tyumenParts"`
	TyumenStock         float64                  `json:"tyumenStock"`
	TyumenInTransit     float64                  `json:"tyumenInTransit"`
	TyumenTarget        float64                  `json:"tyumenTarget"`
	TyumenPlannedOrder  float64                  `json:"tyumenPlannedOrder"`
	TyumenFree          float64                  `json:"tyumenFree"`
	FromTyumen          float64                  `json:"fromTyumen"`
	SupplierNeed        float64                  `json:"supplierNeed"`
	SupplierUnitSize    float64                  `json:"supplierUnitSize"`
	NovacutanMinimum    float64                  `json:"novacutanMinimum,omitzero"`
	BlankBoxSize        float64                  `json:"blankBoxSize,omitzero"`
	HasBoxSize          bool                     `json:"hasBoxSize,omitzero"`
	Variant             string                   `json:"variant,omitempty"`
	ChristinaLine       *orderfill.ChristinaLine `json:"christinaLine,omitempty"`
	HasBudgetData       bool                     `json:"hasBudgetData,omitzero"`
	BudgetCategory      string                   `json:"budgetCategory,omitempty"`
	BudgetDemand        float64                  `json:"budgetDemand,omitzero"`
	BudgetPrice         float64                  `json:"budgetPrice,omitzero"`
	BudgetDiscount      float64                  `json:"budgetDiscount,omitzero"`
	BudgetComment       string                   `json:"budgetComment,omitempty"`
	ActualSupplierOrder float64                  `json:"actualSupplierOrder"`
	NorthNeed           float64                  `json:"northNeed"`
	Comment             string                   `json:"comment"`
	HasTyumenSource     bool                     `json:"hasTyumenSource"`
	WarehouseStock      float64                  `json:"warehouseStock"`
	WarehouseTransit    float64                  `json:"warehouseTransit"`
	HasWarehouseStock   bool                     `json:"hasWarehouseStock"`
}

type Summary struct {
	Kind          string  `json:"kind"`
	DeliveryWeeks float64 `json:"deliveryWeeks"`
}

type Report struct {
	HasTyumenSource    bool                `json:"has_tyumen_source"`
	UploadedCities     []string            `json:"uploaded_cities"`
	PlanRows           []PlanRow           `json:"plan_rows"`
	Transfers          []Transfer          `json:"transfers"`
	ConfirmationGroups []ConfirmationGroup `json:"confirmation_groups"`
	Summary            Summary             `json:"summary"`
}

func BuildReport(brand string, needs []Need, stock []Stock, planned []Planned, groups []ConfirmationGroup, deliveryWeeks ...float64) Report {
	stockByArticle := map[string]Stock{}
	for _, item := range stock {
		stockByArticle[item.Article] = item
	}
	qtyByArticle := map[string]map[string]float64{}
	names := map[string]string{}
	rawArticles := map[string]string{}
	prices := map[string]float64{}
	lines := map[string]*orderfill.ChristinaLine{}
	uploaded := make([]string, 0)
	for _, need := range needs {
		if qtyByArticle[need.Article] == nil {
			qtyByArticle[need.Article] = map[string]float64{}
		}
		qtyByArticle[need.Article][need.City] += need.Qty
		if names[need.Article] == "" {
			names[need.Article] = need.Name
		}
		if lines[need.Article] == nil && need.Line != nil {
			lines[need.Article] = need.Line
		}
		if rawArticles[need.Article] == "" {
			rawArticles[need.Article] = need.ArticleRaw
		}
		if prices[need.Article] == 0 && need.Price > 0 {
			prices[need.Article] = need.Price
		}
		label := Label(need.City)
		if !slices.Contains(uploaded, label) {
			uploaded = append(uploaded, label)
		}
	}
	planByArticle := map[string]Planned{}
	for _, row := range planned {
		planByArticle[row.Article] = row
		if names[row.Article] == "" {
			names[row.Article] = row.Name
		}
	}
	articles := make([]string, 0, len(qtyByArticle))
	for article := range qtyByArticle {
		articles = append(articles, article)
	}
	slices.Sort(articles)
	rows := make([]PlanRow, 0, len(articles))
	transfers := make([]Transfer, 0)
	for _, article := range articles {
		baseArticle := article
		if strings.HasPrefix(article, "home:") || strings.HasPrefix(article, "proff:") {
			_, baseArticle, _ = strings.Cut(article, ":")
		}
		src, inStock := stockByArticle[baseArticle]
		plan := planByArticle[article]
		name := cmp.Or(plan.Name, names[article], src.Name, article)
		cities := make([]CityQty, 0)
		northNeed := 0.0
		for _, city := range cityLabels {
			qty := qtyByArticle[article][city.key]
			if qty <= 0 {
				continue
			}
			northNeed += qty
			cities = append(cities, CityQty{Key: city.key, Label: city.label, Quantity: qty})
		}
		row := PlanRow{
			Key:                 article,
			Article:             article,
			ArticleRaw:          rawArticles[article],
			Name:                name,
			Cities:              cities,
			TyumenStock:         src.Stock,
			TyumenInTransit:     src.InTransit,
			TyumenTarget:        src.Target,
			TyumenPlannedOrder:  plan.TyumenQty,
			TyumenFree:          max(0, src.Stock+src.InTransit-src.Target),
			FromTyumen:          plan.TransferQty,
			SupplierNeed:        plan.SupplierQty,
			SupplierUnitSize:    cmp.Or(plan.UnitSize, 1),
			NovacutanMinimum:    plan.NovacutanMin,
			BlankBoxSize:        plan.BoxSize,
			HasBoxSize:          plan.HasBoxSize,
			Variant:             plan.Variant,
			ChristinaLine:       lines[article],
			HasBudgetData:       prices[article] > 0 && src.MonthlyDemand > 0 && slices.Contains([]string{"A+", "A", "B", "C"}, src.Category),
			BudgetCategory:      src.Category,
			BudgetDemand:        src.MonthlyDemand,
			BudgetPrice:         prices[article],
			ActualSupplierOrder: plan.SupplierQty,
			NorthNeed:           northNeed,
			Comment:             plan.Comment,
			HasTyumenSource:     inStock,
			WarehouseStock:      plan.WarehouseStock,
			WarehouseTransit:    plan.WarehouseTransit,
			HasWarehouseStock:   plan.HasWarehouseStock,
		}
		PopulateAllocation(&row)
		rows = append(rows, row)
		if plan.TransferQty > 0 {
			transfers = append(transfers, Transfer{Article: article, Qty: plan.TransferQty})
		}
	}
	if groups == nil {
		groups = []ConfirmationGroup{}
	}
	weeks := 1.0
	if len(deliveryWeeks) > 0 && deliveryWeeks[0] > 0 {
		weeks = deliveryWeeks[0]
	}
	return Report{
		HasTyumenSource:    len(stock) > 0,
		UploadedCities:     uploaded,
		PlanRows:           rows,
		Transfers:          transfers,
		ConfirmationGroups: groups,
		Summary:            Summary{Kind: brand, DeliveryWeeks: weeks},
	}
}

func ApplyEdits(report *Report, edits []Edit) error {
	if report == nil || len(edits) == 0 {
		return nil
	}
	byKey := make(map[string]Edit, len(edits))
	for _, edit := range edits {
		byKey[edit.Key] = edit
	}
	for i := range report.PlanRows {
		edit, ok := byKey[report.PlanRows[i].Key]
		if !ok {
			continue
		}
		if strings.TrimSpace(edit.Comment) != "" {
			var payload struct {
				Cities        map[string]float64 `json:"cities"`
				Discount      float64            `json:"discount"`
				BudgetComment string             `json:"budgetComment"`
			}
			if err := json.Unmarshal([]byte(edit.Comment), &payload); err != nil {
				return fmt.Errorf("%w: неверный формат количеств городов", orderfill.ErrInvalidInput)
			}
			cities := make([]CityQty, 0, len(payload.Cities))
			for _, city := range cityLabels {
				quantity, exists := payload.Cities[city.key]
				if !exists || quantity == 0 {
					continue
				}
				if quantity < 0 || math.IsNaN(quantity) || math.IsInf(quantity, 0) {
					return fmt.Errorf("%w: количество города должно быть неотрицательным числом", orderfill.ErrInvalidInput)
				}
				cities = append(cities, CityQty{Key: city.key, Label: city.label, Quantity: quantity})
			}
			report.PlanRows[i].Cities = cities
			if payload.Discount < 0 || payload.Discount >= 100 || math.IsNaN(payload.Discount) || math.IsInf(payload.Discount, 0) {
				return fmt.Errorf("%w: скидка должна быть от 0 до 99,99%%", orderfill.ErrInvalidInput)
			}
			report.PlanRows[i].BudgetDiscount = payload.Discount
			if len([]rune(payload.BudgetComment)) > 500 {
				return fmt.Errorf("%w: комментарий бюджетного расчёта слишком длинный", orderfill.ErrInvalidInput)
			}
			report.PlanRows[i].BudgetComment = strings.TrimSpace(payload.BudgetComment)
		}
		raw := edit.Value
		text := strings.TrimSpace(strings.ReplaceAll(raw, ",", "."))
		if text == "" {
			continue
		}
		n, err := strconv.ParseFloat(text, 64)
		if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 {
			return fmt.Errorf("%w: количество должно быть неотрицательным числом", orderfill.ErrInvalidInput)
		}
		report.PlanRows[i].ActualSupplierOrder = n
	}
	return nil
}

// PopulateAllocation reconstructs the same ordered allocation parts used by
// calculation-service and origin/main for comments and the review table.
func PopulateAllocation(row *PlanRow) {
	if row == nil {
		return
	}
	quantities := make(map[string]float64, len(row.Cities))
	for _, city := range row.Cities {
		quantities[city.Key] = city.Quantity
	}
	plannedTyumen := max(0, row.TyumenPlannedOrder)
	if quantity, exists := quantities["tyumen"]; exists {
		plannedTyumen = max(0, quantity)
	}
	free := max(0, row.TyumenStock+row.TyumenInTransit+plannedTyumen-row.TyumenTarget)
	if row.HasWarehouseStock {
		free = min(free, max(0, row.WarehouseStock+row.WarehouseTransit+plannedTyumen))
	}
	row.TyumenFree = free
	row.SupplierParts = row.SupplierParts[:0]
	row.TyumenParts = row.TyumenParts[:0]
	if plannedTyumen > 0 {
		row.SupplierParts = append(row.SupplierParts, CityQty{Key: "tyumen", Label: Label("tyumen"), Quantity: plannedTyumen})
	}
	for _, key := range []string{"nizhnevartovsk", "urengoy", "surgut"} {
		quantity := quantities[key]
		if quantity <= 0 {
			continue
		}
		fromTyumen := min(quantity, free)
		fromSupplier := quantity - fromTyumen
		free -= fromTyumen
		if fromTyumen > 0 {
			row.TyumenParts = append(row.TyumenParts, CityQty{Key: key, Label: Label(key), Quantity: fromTyumen})
		}
		if fromSupplier > 0 {
			row.SupplierParts = append(row.SupplierParts, CityQty{Key: key, Label: Label(key), Quantity: fromSupplier})
		}
	}
}
