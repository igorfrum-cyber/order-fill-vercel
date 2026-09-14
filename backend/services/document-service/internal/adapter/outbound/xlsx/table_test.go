package xlsx

import (
	"testing"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/north"
)

func TestNewTableRoundTrip(t *testing.T) {
	t.Parallel()
	workbook, err := (codec{}).NewTable("Перемещение", []string{"Артикул", "Количество"}, [][]any{{"A-1", 3.0}})
	if err != nil {
		t.Fatal(err)
	}
	sheet, ok := workbook.Sheet("Перемещение")
	if !ok || sheet.Value(1, 1) != "Артикул" || sheet.Value(2, 1) != "A-1" || sheet.Value(2, 2) != "3" {
		t.Fatalf("sheet=%v ok=%v", sheet, ok)
	}
	if _, err := workbook.Save(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteSupplierQuantitiesAppendsMissingPosition(t *testing.T) {
	t.Parallel()
	workbook, err := (codec{}).NewTable("Заказ", []string{"Артикул", "Наименование", "Ед.", "Кол-во"}, [][]any{{"A-1", "Первый", "шт", 1.0}})
	if err != nil {
		t.Fatal(err)
	}
	lines := []north.SupplierLine{
		{Key: "A1", Article: "A-1", Name: "Первый", Unit: "шт", Quantity: 2},
		{Key: "B2", Article: "B-2", Name: "Второй", Unit: "шт", Quantity: 3},
	}
	if err := north.WriteSupplierQuantities(workbook, brand.Rule("angiopharm"), "", lines); err != nil {
		t.Fatal(err)
	}
	sheet, _ := workbook.Sheet("Заказ")
	if sheet.Value(2, 4) != "2" || sheet.Value(3, 1) != "B-2" || sheet.Value(3, 2) != "Второй" || sheet.Value(3, 4) != "3" {
		t.Fatalf("updated=%q appended=%q/%q/%q", sheet.Value(2, 4), sheet.Value(3, 1), sheet.Value(3, 2), sheet.Value(3, 4))
	}
}
