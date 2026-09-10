package xlsx

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/beevik/etree"

	"order-fill/backend/services/document-service/internal/domain/preview"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

const (
	priceSheetName = "Прайс"
	orderSheetName = "Заказ"
	sheet1Path     = "xl/worksheets/sheet1.xml"
)

func TestLoadReadsCellValues(t *testing.T) {
	book := mustLoad(t, buildFixture(t))

	cases := []struct {
		name   string
		sheet  string
		row    int
		column int
		want   string
	}{
		{name: "shared string", sheet: priceSheetName, row: 1, column: 1, want: "Артикул"},
		{name: "shared string with runs", sheet: priceSheetName, row: 1, column: 2, want: "Партия"},
		{name: "inline string", sheet: priceSheetName, row: 1, column: 4, want: "Заказ"},
		{name: "inline string in data row", sheet: priceSheetName, row: 2, column: 1, want: "АРТ-1"},
		{name: "number keeps stored text", sheet: priceSheetName, row: 2, column: 2, want: "12.5"},
		{name: "formula result", sheet: priceSheetName, row: 2, column: 3, want: "25"},
		{name: "boolean", sheet: priceSheetName, row: 4, column: 1, want: "1"},
		{name: "typed string", sheet: priceSheetName, row: 4, column: 3, want: "Ок"},
		{name: "empty cell", sheet: priceSheetName, row: 3, column: 3, want: ""},
		{name: "styled cell without value", sheet: priceSheetName, row: 4, column: 4, want: ""},
		{name: "second sheet", sheet: orderSheetName, row: 1, column: 1, want: "Итог"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			sheet := mustSheet(t, book, testCase.sheet)
			if got := sheet.Value(testCase.row, testCase.column); got != testCase.want {
				t.Fatalf("Value(%d, %d) = %q, want %q", testCase.row, testCase.column, got, testCase.want)
			}
		})
	}
}

func TestLoadExposesSheetAppearance(t *testing.T) {
	book := mustLoad(t, buildFixture(t))
	sheet := mustSheet(t, book, priceSheetName)
	styled, ok := sheet.(spreadsheet.Styled)
	if !ok {
		t.Fatal("xlsx sheet must expose appearance for preview")
	}
	widths := styled.ColumnWidths()
	if len(widths) < 4 || widths[0] != 135 || widths[1] != 91 {
		t.Fatalf("column widths %v", widths)
	}
	if styled.DefaultRowHeight() != 20 {
		t.Fatalf("default row height %v", styled.DefaultRowHeight())
	}
	if styled.CustomRowHeights()[1] != 40 {
		t.Fatalf("row 1 height %#v", styled.CustomRowHeights())
	}
	merges := styled.Merges()
	if len(merges) != 1 || merges[0] != (spreadsheet.Merge{Row: 1, Column: 1, Height: 1, Width: 2}) {
		t.Fatalf("merges %#v", merges)
	}
	index := styled.StyleIndex(1, 1)
	if index == 0 {
		t.Fatal("A1 should carry a non-default style")
	}
	style := styled.Styles()[index]
	if style.Fill != "#1f4e79" || !style.Bold {
		t.Fatalf("A1 style %#v", style)
	}
}

func TestLoadHidesExcelHiddenColumns(t *testing.T) {
	book := mustLoad(t, buildFixture(t))
	sheet := mustSheet(t, book, priceSheetName)
	styled, ok := sheet.(spreadsheet.Styled)
	if !ok {
		t.Fatal("xlsx sheet must expose appearance for preview")
	}
	widths := styled.ColumnWidths()
	if len(widths) < 4 {
		t.Fatalf("column widths %v", widths)
	}
	if widths[2] != 0 {
		t.Fatalf("hidden column C should have width 0, got %v", widths)
	}
	if widths[1] == 0 || widths[3] == 0 {
		t.Fatalf("visible neighbors should keep width, got %v", widths)
	}
}

func TestLoadExposesFormulas(t *testing.T) {
	book := mustLoad(t, buildFixture(t))
	sheet := mustSheet(t, book, priceSheetName)
	formulated, ok := sheet.(spreadsheet.Formulated)
	if !ok {
		t.Fatal("xlsx sheet must expose formulas for live preview totals")
	}
	formulas := formulated.Formulas()
	if len(formulas) != 1 || formulas[0].Row != 2 || formulas[0].Column != 3 || formulas[0].Text != "B2*2" {
		t.Fatalf("formulas %#v", formulas)
	}
}

func TestPreviewCaptureReadsFixtureAppearance(t *testing.T) {
	snapshot := preview.Capture(mustLoad(t, buildFixture(t)))
	meta := snapshot.Meta.Sheets[0]
	if len(meta.Columns) < 4 || meta.Columns[0] != 135 || meta.RowHeight != 20 || meta.RowHeights[1] != 40 {
		t.Fatalf("layout %#v", meta)
	}
	if len(meta.Merges) != 1 || meta.Merges[0].Width != 2 {
		t.Fatalf("merges %#v", meta.Merges)
	}
	if len(meta.Styles) < 2 || meta.Styles[1].Fill != "#1f4e79" {
		t.Fatalf("catalog %#v", meta.Styles)
	}
	if snapshot.Chunks[0][0].Styles[0][0] == 0 {
		t.Fatalf("A1 should keep a style index, %#v", snapshot.Chunks[0][0].Styles)
	}
	if len(meta.Formulas) == 0 {
		t.Fatal("preview meta must carry formulas so the browser can refresh totals")
	}
	found := false
	for _, item := range meta.Formulas {
		if item.Row == 2 && item.Column == 3 && strings.Contains(item.Text, "B2") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected B2 formula on C2, got %#v", meta.Formulas)
	}
	if meta.FormulaValues["2:2"] != "12.5" {
		t.Fatalf("formula values %#v", meta.FormulaValues)
	}
}

func TestSheetsFollowWorkbookOrder(t *testing.T) {
	book := mustLoad(t, buildFixture(t))

	sheets := book.Sheets()
	want := []string{priceSheetName, orderSheetName}
	if len(sheets) != len(want) {
		t.Fatalf("Sheets() returned %d sheets, want %d", len(sheets), len(want))
	}
	for index, name := range want {
		if got := sheets[index].Name(); got != name {
			t.Fatalf("sheet %d is %q, want %q", index, got, name)
		}
	}
	if _, ok := book.Sheet("отсутствует"); ok {
		t.Fatal("Sheet() found a sheet that does not exist")
	}
}

func TestLoadWithProgressReportsFractions(t *testing.T) {
	loader, ok := NewCodec().(spreadsheet.ProgressCodec)
	if !ok {
		t.Fatal("xlsx codec must report load progress for large workbooks")
	}

	var last float64
	count := 0
	book, err := loader.LoadWithProgress(buildFixture(t), func(fraction float64) {
		count++
		if fraction+1e-9 < last {
			t.Fatalf("progress went backwards: %v -> %v", last, fraction)
		}
		last = fraction
	})
	if err != nil {
		t.Fatalf("LoadWithProgress failed: %v", err)
	}
	if mustSheet(t, book, priceSheetName).Value(2, 1) != "АРТ-1" {
		t.Fatal("progress-aware load must still read cell values")
	}
	if count == 0 || last < 0.99 {
		t.Fatalf("expected progress to reach the end, got %d updates ending at %v", count, last)
	}
}

func TestChunkRowsByWorkersKeepsEveryRow(t *testing.T) {
	rows := make([][]byte, 100)
	for index := range rows {
		rows[index] = bytes.Repeat([]byte("x"), 10+index)
	}
	chunks := chunkRowsByWorkers(rows)
	if len(chunks) == 0 {
		t.Fatal("expected at least one chunk")
	}
	total := 0
	for _, chunk := range chunks {
		total += len(chunk)
	}
	if total != len(rows) {
		t.Fatalf("split %d rows into %d, want %d", total, len(chunks), len(rows))
	}
}

func TestWrapRowChunkParsesWithThinEnvelope(t *testing.T) {
	data := []byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
	<dimension ref="A1:A2"/>
	<sheetData>
		<row r="1"><c r="A1"><v>1</v></c></row>
		<row r="2"><c r="A2"><v>2</v></c></row>
	</sheetData>
</worksheet>`)
	open, closeTag, ok := worksheetEnvelope(data)
	if !ok {
		t.Fatal("expected a worksheet envelope")
	}
	split, ok := splitWorksheet(data)
	if !ok || len(split.rows) != 2 {
		t.Fatalf("split rows = %d, want 2", len(split.rows))
	}
	document, err := parseDocument(wrapRowChunk(open, split.rows, closeTag))
	if err != nil {
		t.Fatalf("thin envelope did not parse: %v", err)
	}
	sheetData := findDirectSheetData(document.Root())
	if sheetData == nil || len(sheetData.ChildElements()) != 2 {
		t.Fatal("thin envelope lost row elements")
	}
}

func TestLoadIndexesManyRowsInParallel(t *testing.T) {
	var rows bytes.Buffer
	for index := 1; index <= 40; index++ {
		fmt.Fprintf(&rows, `<row r="%d"><c r="A%d" t="inlineStr"><is><t>R%d</t></is></c></row>`, index, index, index)
	}
	sheetXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>` + rows.String() + `</sheetData></worksheet>`
	content := buildArchive(t, []fixtureEntry{
		{name: contentTypesPath, data: []byte(contentTypesFixtureXML)},
		{name: "_rels/.rels", data: []byte(rootRelsFixtureXML)},
		{name: workbookPath, data: []byte(workbookXML)},
		{name: workbookRelsPath, data: []byte(workbookRelsXML)},
		{name: sharedStringsPath, data: []byte(sharedStringsXML)},
		{name: "xl/styles.xml", data: []byte(stylesXML)},
		{name: sheet1Path, data: []byte(sheetXML)},
		{name: "xl/worksheets/sheet2.xml", data: []byte(sheet2XML)},
	})
	book := mustLoad(t, content)
	sheet := mustSheet(t, book, priceSheetName)
	if got := sheet.Value(1, 1); got != "R1" {
		t.Fatalf("first row = %q, want R1", got)
	}
	if got := sheet.Value(40, 1); got != "R40" {
		t.Fatalf("last row = %q, want R40", got)
	}
	if got := sheet.Bounds(); got.MaxRow != 40 {
		t.Fatalf("bounds max row = %d, want 40", got.MaxRow)
	}
}

func TestBounds(t *testing.T) {
	book := mustLoad(t, buildFixture(t))

	cases := []struct {
		name  string
		sheet string
		want  spreadsheet.Bounds
	}{
		{name: "price sheet", sheet: priceSheetName, want: spreadsheet.Bounds{MaxRow: 4, MaxColumn: 4}},
		{name: "order sheet", sheet: orderSheetName, want: spreadsheet.Bounds{MaxRow: 1, MaxColumn: 1}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := mustSheet(t, book, testCase.sheet).Bounds(); got != testCase.want {
				t.Fatalf("Bounds() = %+v, want %+v", got, testCase.want)
			}
		})
	}
}

func TestSetNumberRoundTrip(t *testing.T) {
	cases := []struct {
		name      string
		row       int
		column    int
		value     float64
		want      string
		wantStyle string
	}{
		{name: "existing numeric cell", row: 2, column: 2, value: 99.5, want: "99.5", wantStyle: "4"},
		{name: "integer drops trailing zeros", row: 2, column: 2, value: 12, want: "12", wantStyle: "4"},
		{name: "large value avoids exponent", row: 2, column: 2, value: 1234567890123, want: "1234567890123", wantStyle: "4"},
		{name: "over shared string", row: 1, column: 1, value: 5, want: "5", wantStyle: "1"},
		{name: "over formula", row: 2, column: 3, value: 42, want: "42", wantStyle: "5"},
		{name: "new cell in existing row", row: 2, column: 4, value: 8, want: "8"},
		{name: "new cell in new row between rows", row: 3, column: 1, value: 3.25, want: "3.25"},
		{name: "new cell in new row after last row", row: 7, column: 2, value: 12, want: "12"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			book := mustLoad(t, buildFixture(t))
			mustSheet(t, book, priceSheetName).SetNumber(testCase.row, testCase.column, testCase.value)

			saved := mustSave(t, book)
			reloaded := mustSheet(t, mustLoad(t, saved), priceSheetName)
			if got := reloaded.Value(testCase.row, testCase.column); got != testCase.want {
				t.Fatalf("value after round trip = %q, want %q", got, testCase.want)
			}

			element := savedCell(t, saved, formatCellRef(testCase.row, testCase.column))
			if got := element.SelectAttrValue("s", ""); got != testCase.wantStyle {
				t.Fatalf("style attribute = %q, want %q", got, testCase.wantStyle)
			}
			if got := element.SelectAttrValue("t", ""); got != "" {
				t.Fatalf("type attribute = %q, want it removed", got)
			}
			if element.SelectElement("f") != nil {
				t.Fatal("formula survived a numeric write")
			}
			if got := elementText(element.SelectElement("v").NotNil()); got != testCase.want {
				t.Fatalf("stored value = %q, want %q", got, testCase.want)
			}
		})
	}
}

func TestSetNumberKeepsRowsAndCellsOrdered(t *testing.T) {
	book := mustLoad(t, buildFixture(t))
	sheet := mustSheet(t, book, priceSheetName)
	sheet.SetNumber(7, 1, 1)
	sheet.SetNumber(3, 1, 2)
	sheet.SetNumber(2, 4, 3)

	saved := mustSave(t, book)
	root := mustParse(t, readPart(t, saved, sheet1Path)).Root()
	sheetData := root.SelectElement("sheetData")

	var rows []string
	for _, row := range sheetData.SelectElements("row") {
		rows = append(rows, row.SelectAttrValue("r", ""))
	}
	assertEqualStrings(t, "row order", rows, []string{"1", "2", "3", "4", "7"})

	var refs []string
	for _, element := range sheetData.SelectElements("row")[1].SelectElements("c") {
		refs = append(refs, element.SelectAttrValue("r", ""))
	}
	assertEqualStrings(t, "cell order", refs, []string{"A2", "B2", "C2", "D2"})
}

// The fixture holds rows 1, 2 and 4, so deleting row 2 also proves that a gap
// in the numbering does not shift the surviving rows twice.
func TestDeleteRowsRenumbersEverythingBelow(t *testing.T) {
	book := mustLoad(t, buildFixture(t))
	sheet := mustSheet(t, book, priceSheetName)
	sheet.DeleteRows([]int{2, 2})

	saved := mustSave(t, book)
	for _, reloaded := range []spreadsheet.Sheet{sheet, mustSheet(t, mustLoad(t, saved), priceSheetName)} {
		if got := reloaded.Value(1, 1); got != "Артикул" {
			t.Fatalf("row above the deleted one = %q, want it untouched", got)
		}
		if got := reloaded.Value(2, 1); got != "" {
			t.Fatalf("value at row 2 = %q, want the empty row that moved up into it", got)
		}
		if got := reloaded.Value(3, 1); got != "1" {
			t.Fatalf("value at row 3 = %q, want the content of former row 4", got)
		}
		if got := reloaded.Value(3, 3); got != "Ок" {
			t.Fatalf("value at C3 = %q, want the content of former C4", got)
		}
		if got := reloaded.Bounds(); got != (spreadsheet.Bounds{MaxRow: 3, MaxColumn: 4}) {
			t.Fatalf("Bounds() = %+v, want the sheet to end at row 3", got)
		}
	}

	sheetData := mustParse(t, readPart(t, saved, sheet1Path)).Root().SelectElement("sheetData")
	var rows []string
	for _, row := range sheetData.SelectElements("row") {
		rows = append(rows, row.SelectAttrValue("r", ""))
	}
	assertEqualStrings(t, "row numbers", rows, []string{"1", "3"})

	var refs []string
	for _, element := range sheetData.SelectElements("row")[1].SelectElements("c") {
		refs = append(refs, element.SelectAttrValue("r", ""))
	}
	assertEqualStrings(t, "cell references", refs, []string{"A3", "B3", "C3", "D3"})
}

func TestDeleteRowsKeepsLaterWritesAddressable(t *testing.T) {
	book := mustLoad(t, buildFixture(t))
	sheet := mustSheet(t, book, priceSheetName)
	sheet.DeleteRows([]int{2})
	sheet.SetNumber(3, 2, 42)

	if got := sheet.Value(3, 2); got != "42" {
		t.Fatalf("value after write = %q, want 42", got)
	}
	reloaded := mustSheet(t, mustLoad(t, mustSave(t, book)), priceSheetName)
	if got := reloaded.Value(3, 2); got != "42" {
		t.Fatalf("value after round trip = %q, want 42", got)
	}
}

func TestDeleteRowsIgnoresNothingToDo(t *testing.T) {
	fixture := buildFixture(t)
	book := mustLoad(t, fixture)
	sheet := mustSheet(t, book, priceSheetName)
	sheet.DeleteRows(nil)
	sheet.DeleteRows([]int{0, -3})

	if got, want := readPart(t, mustSave(t, book), sheet1Path), readPart(t, fixture, sheet1Path); !bytes.Equal(got, want) {
		t.Fatalf("worksheet changed without a row to delete: got %q, want %q", got, want)
	}
}

func TestClearValueKeepsStyle(t *testing.T) {
	cases := []struct {
		name      string
		row       int
		column    int
		wantStyle string
	}{
		{name: "numeric cell", row: 2, column: 2, wantStyle: "4"},
		{name: "shared string cell", row: 1, column: 1, wantStyle: "1"},
		{name: "inline string cell", row: 1, column: 4, wantStyle: "2"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			book := mustLoad(t, buildFixture(t))
			sheet := mustSheet(t, book, priceSheetName)
			sheet.ClearValue(testCase.row, testCase.column)
			if got := sheet.Value(testCase.row, testCase.column); got != "" {
				t.Fatalf("value after clear = %q, want empty", got)
			}

			saved := mustSave(t, book)
			element := savedCell(t, saved, formatCellRef(testCase.row, testCase.column))
			if got := element.SelectAttrValue("s", ""); got != testCase.wantStyle {
				t.Fatalf("style attribute = %q, want %q", got, testCase.wantStyle)
			}
			if got := element.SelectAttrValue("t", ""); got != "" {
				t.Fatalf("type attribute = %q, want it removed", got)
			}
			if len(element.Child) != 0 {
				t.Fatalf("cleared cell kept %d child tokens", len(element.Child))
			}
			if got := mustSheet(t, mustLoad(t, saved), priceSheetName).Value(testCase.row, testCase.column); got != "" {
				t.Fatalf("value after round trip = %q, want empty", got)
			}
		})
	}
}

func TestSetText(t *testing.T) {
	cases := []struct {
		name      string
		row       int
		column    int
		value     string
		wantStyle string
		wantType  string
	}{
		{name: "over number", row: 2, column: 2, value: "нет в наличии", wantStyle: "4", wantType: "inlineStr"},
		{name: "new cell", row: 5, column: 3, value: "Комментарий & <тег>", wantType: "inlineStr"},
		{name: "empty text clears the cell", row: 2, column: 2, value: "", wantStyle: "4"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			book := mustLoad(t, buildFixture(t))
			sheet := mustSheet(t, book, priceSheetName)
			sheet.SetText(testCase.row, testCase.column, testCase.value)
			if got := sheet.Value(testCase.row, testCase.column); got != testCase.value {
				t.Fatalf("value after write = %q, want %q", got, testCase.value)
			}

			saved := mustSave(t, book)
			if got := mustSheet(t, mustLoad(t, saved), priceSheetName).Value(testCase.row, testCase.column); got != testCase.value {
				t.Fatalf("value after round trip = %q, want %q", got, testCase.value)
			}
			element := savedCell(t, saved, formatCellRef(testCase.row, testCase.column))
			if got := element.SelectAttrValue("t", ""); got != testCase.wantType {
				t.Fatalf("type attribute = %q, want %q", got, testCase.wantType)
			}
			if got := element.SelectAttrValue("s", ""); got != testCase.wantStyle {
				t.Fatalf("style attribute = %q, want %q", got, testCase.wantStyle)
			}
			if testCase.value == "" {
				return
			}
			inline := element.SelectElement("is").NotNil().SelectElement("t").NotNil()
			if got := elementText(inline); got != testCase.value {
				t.Fatalf("inline text = %q, want %q", got, testCase.value)
			}
		})
	}
}

func TestSavePreservesUntouchedParts(t *testing.T) {
	fixture := buildFixture(t)
	book := mustLoad(t, fixture)
	mustSheet(t, book, priceSheetName).SetNumber(2, 2, 4)
	saved := mustSave(t, book)

	preserved := []string{
		"_rels/.rels",
		"xl/styles.xml",
		"xl/sharedStrings.xml",
		"xl/media/image1.png",
		"xl/worksheets/sheet2.xml",
		"xl/vbaProject.bin",
	}
	for _, name := range preserved {
		t.Run(name, func(t *testing.T) {
			if got, want := readPart(t, saved, name), readPart(t, fixture, name); !bytes.Equal(got, want) {
				t.Fatalf("part %s changed: got %q, want %q", name, got, want)
			}
		})
	}

	t.Run("worksheet keeps unrelated elements", func(t *testing.T) {
		root := mustParse(t, readPart(t, saved, sheet1Path)).Root()
		for _, tag := range []string{"cols", "mergeCells", "dataValidations", "pageMargins", "drawing"} {
			if root.SelectElement(tag) == nil {
				t.Fatalf("worksheet lost the %s element", tag)
			}
		}
	})

	t.Run("entry order is stable", func(t *testing.T) {
		var want []string
		for _, name := range partNames(t, fixture) {
			if name != calcChainPath {
				want = append(want, name)
			}
		}
		assertEqualStrings(t, "entry names", partNames(t, saved), want)
	})
}

func TestSaveForcesRecalculation(t *testing.T) {
	book := mustLoad(t, buildFixture(t))
	mustSheet(t, book, priceSheetName).SetNumber(2, 2, 4)
	saved := mustSave(t, book)

	for _, name := range partNames(t, saved) {
		if name == calcChainPath {
			t.Fatalf("%s survived the save", calcChainPath)
		}
	}

	root := mustParse(t, readPart(t, saved, workbookPath)).Root()
	calcPr := root.SelectElement("calcPr")
	if calcPr == nil {
		t.Fatal("workbook has no calcPr element")
	}
	for attribute, want := range map[string]string{"calcMode": "auto", "fullCalcOnLoad": "1", "forceFullCalc": "1"} {
		if got := calcPr.SelectAttrValue(attribute, ""); got != want {
			t.Fatalf("calcPr %s = %q, want %q", attribute, got, want)
		}
	}
	if extension := root.SelectElement("extLst"); extension == nil || calcPr.Index() > extension.Index() {
		t.Fatal("calcPr must be placed before extLst to keep the workbook schema valid")
	}

	rels := mustParse(t, readPart(t, saved, workbookRelsPath)).Root()
	for _, relationship := range rels.SelectElements("Relationship") {
		if got := relationship.SelectAttrValue("Type", ""); bytes.Contains([]byte(got), []byte("calcChain")) {
			t.Fatalf("relationship %q survived the save", got)
		}
	}

	types := mustParse(t, readPart(t, saved, contentTypesPath)).Root()
	for _, override := range types.SelectElements("Override") {
		if got := override.SelectAttrValue("PartName", ""); got == "/"+calcChainPath {
			t.Fatalf("content type override %q survived the save", got)
		}
	}
}

func TestLoadRejectsBrokenDocuments(t *testing.T) {
	cases := []struct {
		name    string
		content []byte
	}{
		{name: "not a zip", content: []byte("не архив")},
		{name: "zip without workbook", content: buildArchive(t, []fixtureEntry{{name: "hello.txt", data: []byte("hi")}})},
		{name: "workbook without relationships", content: buildArchive(t, []fixtureEntry{{name: workbookPath, data: []byte(workbookXML)}})},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if _, err := NewCodec().Load(testCase.content); err == nil {
				t.Fatal("Load() accepted a broken document")
			}
		})
	}
}

func TestNormalizeWorkbookTarget(t *testing.T) {
	cases := []struct {
		target string
		want   string
	}{
		{target: "worksheets/sheet1.xml", want: "xl/worksheets/sheet1.xml"},
		{target: "/xl/worksheets/sheet2.xml", want: "xl/worksheets/sheet2.xml"},
		{target: "xl/worksheets/sheet3.xml", want: "xl/worksheets/sheet3.xml"},
	}
	for _, testCase := range cases {
		t.Run(testCase.target, func(t *testing.T) {
			if got := normalizeWorkbookTarget(testCase.target); got != testCase.want {
				t.Fatalf("normalizeWorkbookTarget(%q) = %q, want %q", testCase.target, got, testCase.want)
			}
		})
	}
}

func mustLoad(t *testing.T, content []byte) spreadsheet.Workbook {
	t.Helper()
	book, err := NewCodec().Load(content)
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	return book
}

func mustSave(t *testing.T, book spreadsheet.Workbook) []byte {
	t.Helper()
	saved, err := book.Save()
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}
	return saved
}

func mustSheet(t *testing.T, book spreadsheet.Workbook, name string) spreadsheet.Sheet {
	t.Helper()
	sheet, ok := book.Sheet(name)
	if !ok {
		t.Fatalf("sheet %q was not found", name)
	}
	return sheet
}

func mustParse(t *testing.T, data []byte) *etree.Document {
	t.Helper()
	document, err := parseDocument(data)
	if err != nil {
		t.Fatalf("parse document failed: %v", err)
	}
	return document
}

func savedCell(t *testing.T, saved []byte, reference string) *etree.Element {
	t.Helper()
	root := mustParse(t, readPart(t, saved, sheet1Path)).Root()
	for _, element := range descendants(root, "c") {
		if element.SelectAttrValue("r", "") == reference {
			return element
		}
	}
	t.Fatalf("cell %s was not found in the saved worksheet", reference)
	return nil
}

func readPart(t *testing.T, content []byte, name string) []byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("open archive failed: %v", err)
	}
	for _, file := range reader.File {
		if file.Name != name {
			continue
		}
		stream, err := file.Open()
		if err != nil {
			t.Fatalf("open entry %s failed: %v", name, err)
		}
		defer func() { _ = stream.Close() }()
		data, err := io.ReadAll(stream)
		if err != nil {
			t.Fatalf("read entry %s failed: %v", name, err)
		}
		return data
	}
	t.Fatalf("entry %s was not found", name)
	return nil
}

func partNames(t *testing.T, content []byte) []string {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("open archive failed: %v", err)
	}
	names := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		names = append(names, file.Name)
	}
	return names
}

func assertEqualStrings(t *testing.T, subject string, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", subject, got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("%s = %v, want %v", subject, got, want)
		}
	}
}

type fixtureEntry struct {
	name  string
	data  []byte
	store bool
}

func buildFixture(t *testing.T) []byte {
	t.Helper()
	return buildArchive(t, []fixtureEntry{
		{name: contentTypesPath, data: []byte(contentTypesFixtureXML)},
		{name: "_rels/.rels", data: []byte(rootRelsFixtureXML)},
		{name: workbookPath, data: []byte(workbookXML)},
		{name: workbookRelsPath, data: []byte(workbookRelsXML)},
		{name: sharedStringsPath, data: []byte(sharedStringsXML)},
		{name: "xl/styles.xml", data: []byte(stylesXML)},
		{name: sheet1Path, data: []byte(sheet1XML)},
		{name: "xl/worksheets/sheet2.xml", data: []byte(sheet2XML)},
		{name: calcChainPath, data: []byte(calcChainXML)},
		{name: "xl/media/image1.png", data: pngFixture, store: true},
		{name: "xl/vbaProject.bin", data: []byte{0xd0, 0xcf, 0x11, 0xe0, 0x00, 0x01}, store: true},
	})
}

func buildArchive(t *testing.T, entries []fixtureEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		method := uint16(zip.Deflate)
		if entry.store {
			method = zip.Store
		}
		target, err := writer.CreateHeader(&zip.FileHeader{Name: entry.name, Method: method})
		if err != nil {
			t.Fatalf("create entry %s failed: %v", entry.name, err)
		}
		if _, err := target.Write(entry.data); err != nil {
			t.Fatalf("write entry %s failed: %v", entry.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close archive failed: %v", err)
	}
	return buffer.Bytes()
}

var pngFixture = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x11, 0x22}

const contentTypesFixtureXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Default Extension="png" ContentType="image/png"/><Default Extension="bin" ContentType="application/vnd.ms-office.vbaProject"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.ms-excel.sheet.macroEnabled.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/worksheets/sheet2.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/sharedStrings.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sharedStrings+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/><Override PartName="/xl/calcChain.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.calcChain+xml"/></Types>`

const rootRelsFixtureXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`

const workbookXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><fileVersion appName="xl" lastEdited="7"/><workbookPr codeName="ЭтаКнига"/><bookViews><workbookView xWindow="0" yWindow="0" windowWidth="20000" windowHeight="12000"/></bookViews><sheets><sheet name="Прайс" sheetId="1" r:id="rId1"/><sheet name="Заказ" sheetId="2" r:id="rId2"/></sheets><definedNames><definedName name="Артикулы">Прайс!$A$2:$A$4</definedName></definedNames><extLst><ext uri="{7523E5D3-25F3-A5E0-1632-64F254C22452}"/></extLst></workbook>`

const workbookRelsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="/xl/worksheets/sheet2.xml"/><Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/sharedStrings" Target="sharedStrings.xml"/><Relationship Id="rId4" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/><Relationship Id="rId5" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/calcChain" Target="calcChain.xml"/><Relationship Id="rId6" Type="http://schemas.microsoft.com/office/2006/relationships/vbaProject" Target="vbaProject.bin"/></Relationships>`

const sharedStringsXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="3" uniqueCount="3"><si><t>Артикул</t></si><si><r><rPr><b/></rPr><t>Пар</t></r><r><t>тия</t></r></si><si><t>Остаток</t></si></sst>`

const stylesXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><numFmts count="1"><numFmt numFmtId="164" formatCode="#,##0.00\ &quot;шт&quot;"/></numFmts><fonts count="2"><font><sz val="11"/><name val="Calibri"/></font><font><b/><sz val="11"/><name val="Calibri"/></font></fonts><fills count="2"><fill><patternFill patternType="none"/></fill><fill><patternFill patternType="solid"><fgColor rgb="FF1F4E79"/></patternFill></fill></fills><borders count="1"><border/></borders><cellXfs count="8"><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/><xf numFmtId="0" fontId="1" fillId="1" borderId="0"/><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/><xf numFmtId="49" fontId="0" fillId="0" borderId="0"/><xf numFmtId="164" fontId="0" fillId="0" borderId="0"/><xf numFmtId="2" fontId="0" fillId="0" borderId="0"/><xf numFmtId="0" fontId="0" fillId="0" borderId="0"/><xf numFmtId="1" fontId="0" fillId="0" borderId="0"/></cellXfs></styleSheet>`

const sheet1XML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><dimension ref="A1:D4"/><sheetViews><sheetView tabSelected="1" workbookViewId="0"><pane ySplit="1" topLeftCell="A2" state="frozen"/></sheetView></sheetViews><sheetFormatPr defaultRowHeight="15"/><cols><col min="1" max="1" width="18.5" customWidth="1"/><col min="2" max="2" width="12.25" customWidth="1"/><col min="3" max="3" width="11.66" hidden="1" customWidth="1"/><col min="4" max="4" width="12.25" customWidth="1"/></cols><sheetData><row r="1" spans="1:4" ht="30" customHeight="1"><c r="A1" s="1" t="s"><v>0</v></c><c r="B1" s="1" t="s"><v>1</v></c><c r="D1" s="2" t="inlineStr"><is><t>Заказ</t></is></c></row><row r="2" spans="1:4"><c r="A2" s="3" t="inlineStr"><is><t>АРТ-1</t></is></c><c r="B2" s="4"><v>12.5</v></c><c r="C2" s="5"><f>B2*2</f><v>25</v></c></row><row r="4" spans="1:4"><c r="A4" s="3" t="b"><v>1</v></c><c r="B4" s="7"><v>7</v></c><c r="C4" s="6" t="str"><v>Ок</v></c><c r="D4" s="6"/></row></sheetData><mergeCells count="1"><mergeCell ref="A1:B1"/></mergeCells><dataValidations count="1"><dataValidation type="whole" sqref="B2:B4" allowBlank="1"><formula1>0</formula1></dataValidation></dataValidations><pageMargins left="0.7" right="0.7" top="0.75" bottom="0.75" header="0.3" footer="0.3"/><drawing r:id="rId1"/></worksheet>`

const sheet2XML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><dimension ref="A1"/><sheetData><row r="1"><c r="A1" s="1" t="inlineStr"><is><t>Итог</t></is></c></row></sheetData><pageMargins left="0.7" right="0.7" top="0.75" bottom="0.75" header="0.3" footer="0.3"/></worksheet>`

const calcChainXML = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<calcChain xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="1"><c r="C2" i="1"/></calcChain>`
