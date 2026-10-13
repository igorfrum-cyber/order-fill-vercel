package xlsx

import (
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
	"testing"
)

func TestCopyCellStyleRetainsStyleWhenOrderColumnIsCreated(t *testing.T) {
	book := mustLoad(t, buildFixture(t))
	original, _ := book.Sheet(priceSheetName)
	source := original.(*sheet).findOrCreateCell(2, 2)
	source.xf = 7
	source.element.CreateAttr("s", "7")
	copier, ok := original.(spreadsheet.CellStyleCopier)
	if !ok {
		t.Fatal("xlsx does not support style copying")
	}
	copier.CopyCellStyle(2, 2, 9)
	original.SetNumber(2, 9, 3)
	saved, err := book.Save()
	if err != nil {
		t.Fatal(err)
	}
	reloaded := mustLoad(t, saved)
	result, _ := reloaded.Sheet(priceSheetName)
	cell := result.(*sheet).cells[cellKey{row: 2, column: 9}]
	if cell.xf != 7 || cell.value != "3" {
		t.Fatalf("copied cell = style %d value %q", cell.xf, cell.value)
	}
}
