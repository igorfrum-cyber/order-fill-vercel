package domain

type CityNeed struct {
	City    string
	Article string
	Qty     float64
}

type PlanRow struct {
	Article       string
	Name          string
	TyumenQty     float64
	TransferQty   float64
	SupplierQty   float64
	Comment       string
	TyumenStock   float64
	TyumenTransit float64
	TyumenTarget  float64
	UnitSize      float64
	NovacutanMin  float64
}
