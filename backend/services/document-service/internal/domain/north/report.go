package north

import (
	"cmp"
	"slices"
)

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
	Article, Name, Comment              string
	TyumenQty, TransferQty, SupplierQty float64
}

type Transfer struct {
	Article string  `json:"article"`
	Qty     float64 `json:"qty"`
}

type PlanRow struct {
	Key                 string    `json:"key"`
	Article             string    `json:"article"`
	Name                string    `json:"name"`
	Cities              []CityQty `json:"cities"`
	TyumenStock         float64   `json:"tyumenStock"`
	TyumenInTransit     float64   `json:"tyumenInTransit"`
	TyumenTarget        float64   `json:"tyumenTarget"`
	TyumenFree          float64   `json:"tyumenFree"`
	FromTyumen          float64   `json:"fromTyumen"`
	SupplierNeed        float64   `json:"supplierNeed"`
	ActualSupplierOrder float64   `json:"actualSupplierOrder"`
	NorthNeed           float64   `json:"northNeed"`
	Comment             string    `json:"comment"`
	HasTyumenSource     bool      `json:"hasTyumenSource"`
}

type Summary struct {
	Kind string `json:"kind"`
}

type Report struct {
	HasTyumenSource    bool                `json:"has_tyumen_source"`
	UploadedCities     []string            `json:"uploaded_cities"`
	PlanRows           []PlanRow           `json:"plan_rows"`
	Transfers          []Transfer          `json:"transfers"`
	ConfirmationGroups []ConfirmationGroup `json:"confirmation_groups"`
	Summary            Summary             `json:"summary"`
}

func BuildReport(brand string, needs []Need, stock []Stock, planned []Planned, groups []ConfirmationGroup) Report {
	stockByArticle := map[string]Stock{}
	for _, item := range stock {
		stockByArticle[item.Article] = item
	}
	qtyByArticle := map[string]map[string]float64{}
	names := map[string]string{}
	uploaded := make([]string, 0)
	for _, need := range needs {
		if qtyByArticle[need.Article] == nil {
			qtyByArticle[need.Article] = map[string]float64{}
		}
		qtyByArticle[need.Article][need.City] += need.Qty
		if names[need.Article] == "" {
			names[need.Article] = need.Name
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
		src, inStock := stockByArticle[article]
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
			Name:                name,
			Cities:              cities,
			TyumenStock:         src.Stock,
			TyumenInTransit:     src.InTransit,
			TyumenTarget:        src.Target,
			TyumenFree:          max(0, src.Stock+src.InTransit-src.Target),
			FromTyumen:          plan.TransferQty,
			SupplierNeed:        plan.SupplierQty,
			ActualSupplierOrder: plan.SupplierQty,
			NorthNeed:           northNeed,
			Comment:             plan.Comment,
			HasTyumenSource:     inStock,
		}
		rows = append(rows, row)
		if plan.TransferQty > 0 {
			transfers = append(transfers, Transfer{Article: article, Qty: plan.TransferQty})
		}
	}
	if groups == nil {
		groups = []ConfirmationGroup{}
	}
	return Report{
		HasTyumenSource:    len(stock) > 0,
		UploadedCities:     uploaded,
		PlanRows:           rows,
		Transfers:          transfers,
		ConfirmationGroups: groups,
		Summary:            Summary{Kind: brand},
	}
}
