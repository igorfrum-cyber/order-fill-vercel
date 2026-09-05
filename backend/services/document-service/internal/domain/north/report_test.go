package north

import "testing"

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
