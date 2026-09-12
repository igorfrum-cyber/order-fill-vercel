package domain

type CityNeed struct {
	City    string
	Article string
	Qty     float64
}

type PlanRow struct {
	Article           string
	Name              string
	Variant           string
	TyumenQty         float64
	TransferQty       float64
	SupplierQty       float64
	Comment           string
	TyumenStock       float64
	TyumenTransit     float64
	TyumenTarget      float64
	UnitSize          float64
	NovacutanMin      float64
	BoxSize           float64
	HasBoxSize        bool
	WarehouseStock    float64
	WarehouseTransit  float64
	HasWarehouseStock bool
}

// NorthSupplierPosition is the blank/source fields defaultNorthActualSupplierOrder reads.
type NorthSupplierPosition struct {
	BlankBoxSize     float64
	HasBoxSize       bool
	SupplierUnitSize float64
	NovacutanMinimum float64
}
