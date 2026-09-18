package north

import (
	"testing"

	"order-fill/backend/services/document-service/internal/domain/orderfill"
)

func TestBuildReportCarriesChristinaLine(t *testing.T) {
	t.Parallel()
	line := &orderfill.ChristinaLine{ID: "MUSE", Name: "MUSE", Article: "CHR001", Required: []string{"CHR001", "CHR002"}}
	needs := []Need{
		{City: "surgut", Article: "proff:CHR001", Name: "Товар A", Variant: "proff", Qty: 3, Price: 100, Line: line},
		{City: "surgut", Article: "proff:CHR002", Name: "Товар B", Variant: "proff", Qty: 3, Price: 100},
	}
	report := BuildReport("christina", needs, nil, nil, nil)
	byKey := map[string]PlanRow{}
	for _, row := range report.PlanRows {
		byKey[row.Key] = row
	}
	if got := byKey["proff:CHR001"].ChristinaLine; got == nil || got.ID != "MUSE" {
		t.Fatalf("proff:CHR001 line = %+v, want MUSE", got)
	}
	if byKey["proff:CHR002"].ChristinaLine != nil {
		t.Fatalf("proff:CHR002 had no line in needs; must stay nil: %+v", byKey["proff:CHR002"].ChristinaLine)
	}
}

func TestBuildReport(t *testing.T) {
	t.Parallel()
	needs := []Need{
		{City: "surgut", Article: "A1", Name: "Cream", Qty: 10},
		{City: "urengoy", Article: "A1", Name: "Cream", Qty: 5},
	}
	stock := []Stock{{Article: "A1", Name: "Cream", Stock: 20, Target: 5}}
	planned := []Planned{{
		Article: "A1", Name: "Cream", TransferQty: 15, SupplierQty: 0, Comment: "from tyumen",
	}}
	groups := []ConfirmationGroup{{
		City: CityQty{Key: "surgut", Label: "Сургут"}, Variants: []string{"HOME", "PROFF"},
	}}

	report := BuildReport("klapp", needs, stock, planned, groups)

	if !report.HasTyumenSource {
		t.Fatal("tyumen stock must mark has_tyumen_source")
	}
	if len(report.UploadedCities) != 2 {
		t.Fatalf("uploaded cities=%v", report.UploadedCities)
	}
	if report.Summary.Kind != "klapp" {
		t.Fatalf("summary=%+v", report.Summary)
	}
	if len(report.PlanRows) != 1 {
		t.Fatalf("rows=%d", len(report.PlanRows))
	}
	row := report.PlanRows[0]
	if row.Key != "A1" || row.Name != "Cream" || row.NorthNeed != 15 || row.FromTyumen != 15 || row.ActualSupplierOrder != 0 {
		t.Fatalf("row=%+v", row)
	}
	if !row.HasTyumenSource || row.TyumenStock != 20 {
		t.Fatalf("stock not applied: %+v", row)
	}
	if len(row.Cities) != 2 {
		t.Fatalf("cities=%v", row.Cities)
	}
	if len(report.Transfers) != 1 || report.Transfers[0].Qty != 15 {
		t.Fatalf("transfers=%v", report.Transfers)
	}
	if len(report.ConfirmationGroups) != 1 || report.ConfirmationGroups[0].Variants[0] != "HOME" {
		t.Fatalf("groups=%v", report.ConfirmationGroups)
	}
}

func TestBuildReportWithoutTyumenStock(t *testing.T) {
	t.Parallel()
	report := BuildReport("angiopharm", []Need{{City: "surgut", Article: "B2", Qty: 3}}, nil, []Planned{{
		Article: "B2", SupplierQty: 3,
	}}, nil)
	if report.HasTyumenSource {
		t.Fatal("no tyumen file")
	}
	if report.PlanRows[0].HasTyumenSource {
		t.Fatal("article was not in tyumen table")
	}
}

func TestApplyEditsSetsActualSupplierOrder(t *testing.T) {
	t.Parallel()
	report := BuildReport("angiopharm", []Need{{City: "surgut", Article: "A1", Qty: 3}}, nil, []Planned{{
		Article: "A1", SupplierQty: 3,
	}}, nil)
	if err := ApplyEdits(&report, []Edit{{Key: "A1", Value: "12"}}); err != nil {
		t.Fatal(err)
	}
	if report.PlanRows[0].ActualSupplierOrder != 12 {
		t.Fatalf("actual=%v", report.PlanRows[0].ActualSupplierOrder)
	}
}

func TestApplyEditsValidatesAndKeepsCityQuantities(t *testing.T) {
	t.Parallel()
	report := BuildReport("angiopharm", []Need{{City: "surgut", Article: "A1", Qty: 3}}, nil, []Planned{{Article: "A1", SupplierQty: 3}}, nil)
	err := ApplyEdits(&report, []Edit{{Key: "A1", Value: "8", Comment: `{"cities":{"surgut":5,"urengoy":2},"discount":30}`}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.PlanRows[0].Cities) != 2 || report.PlanRows[0].BudgetDiscount != 30 {
		t.Fatalf("row=%+v", report.PlanRows[0])
	}
	if err := ApplyEdits(&report, []Edit{{Key: "A1", Value: "8", Comment: `{"cities":{"surgut":-1}}`}}); err == nil {
		t.Fatal("negative city quantity must fail")
	}
}

func TestPopulateAllocationIncludesTyumenOrderAndWarehouseCap(t *testing.T) {
	t.Parallel()
	row := PlanRow{
		Cities: []CityQty{
			{Key: "tyumen", Quantity: 8},
			{Key: "nizhnevartovsk", Quantity: 6},
			{Key: "surgut", Quantity: 6},
		},
		TyumenStock: 20, TyumenTarget: 5,
		WarehouseStock: 4, HasWarehouseStock: true,
	}
	PopulateAllocation(&row)
	if row.TyumenFree != 12 || len(row.TyumenParts) != 2 || len(row.SupplierParts) != 1 || row.SupplierParts[0].Key != "tyumen" {
		t.Fatalf("allocation=%+v", row)
	}
}

func TestPopulateAllocationUsesTyumenSourcePlanWhenNoTyumenBlank(t *testing.T) {
	t.Parallel()
	row := PlanRow{
		Cities:             []CityQty{{Key: "surgut", Quantity: 6}},
		TyumenStock:        2,
		TyumenTarget:       5,
		TyumenPlannedOrder: 8,
	}
	PopulateAllocation(&row)
	if row.TyumenFree != 5 || len(row.TyumenParts) != 1 || len(row.SupplierParts) != 2 || row.SupplierParts[0].Key != "tyumen" || row.SupplierParts[1].Key != "surgut" {
		t.Fatalf("allocation=%+v", row)
	}
}
