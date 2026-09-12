package domain

type OrderRow struct {
	ID                string
	Article           string
	Name              string
	Revenue           float64
	Stock             float64
	InTransit         float64
	MonthlySales      []float64
	Recommended       float64
	ABCCategory       string
	TargetStock       float64
	HasOrderedFact    bool
	OrderedFact       float64
	RevenuePercent    float64
	CumulativePercent float64
	AverageMonthly    float64
	TotalQuantity     float64
	BoxSize           float64
	HasBoxSize        bool
	WarehouseStock    float64
	WarehouseTransit  float64
	HasWarehouseStock bool
}
