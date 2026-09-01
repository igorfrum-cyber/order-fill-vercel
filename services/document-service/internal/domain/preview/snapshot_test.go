package preview

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"strings"
	"testing"

	"order-fill/services/document-service/internal/domain/spreadsheet"
)

type fakeSheet struct {
	name  string
	cells map[[2]int]string
}

func (s *fakeSheet) Name() string { return s.name }

func (s *fakeSheet) Bounds() spreadsheet.Bounds {
	bounds := spreadsheet.Bounds{}
	for ref := range s.cells {
		if ref[0] > bounds.MaxRow {
			bounds.MaxRow = ref[0]
		}
		if ref[1] > bounds.MaxColumn {
			bounds.MaxColumn = ref[1]
		}
	}
	return bounds
}

func (s *fakeSheet) Value(row int, column int) string { return s.cells[[2]int{row, column}] }

func (s *fakeSheet) SetNumber(int, int, float64) {}
func (s *fakeSheet) ClearValue(int, int)         {}
func (s *fakeSheet) SetText(int, int, string)    {}
func (s *fakeSheet) DeleteRows([]int)            {}

type fakeWorkbook struct {
	sheets []spreadsheet.Sheet
}

func (w *fakeWorkbook) Sheets() []spreadsheet.Sheet { return w.sheets }

func (w *fakeWorkbook) Sheet(name string) (spreadsheet.Sheet, bool) {
	for _, sheet := range w.sheets {
		if sheet.Name() == name {
			return sheet, true
		}
	}
	return nil, false
}

func (w *fakeWorkbook) Save() ([]byte, error) { return nil, nil }

func sourceSheet() *fakeSheet {
	return &fakeSheet{
		name: "Тюмень",
		cells: map[[2]int]string{
			{13, 4}:  "Артикул",
			{13, 5}:  "Товар",
			{14, 32}: "Артикул",
			{14, 33}: "Товар",
			{14, 34}: "Заказано по факту",
			{14, 35}: "Комментарий",
			{15, 32}: "RG01",
			{15, 33}: "Крем",
			{15, 34}: "12",
			{15, 35}: "коробка",
			{16, 32}: "CT02",
			{16, 33}: "Сыворотка",
			{16, 34}: "3",
		},
	}
}

func TestCaptureIndexesTheOrderTableArticleColumn(t *testing.T) {
	snapshot := Capture(&fakeWorkbook{sheets: []spreadsheet.Sheet{sourceSheet()}})
	if len(snapshot.Meta.Sheets) != 1 {
		t.Fatalf("expected one sheet, got %d", len(snapshot.Meta.Sheets))
	}
	meta := snapshot.Meta.Sheets[0]
	if meta.Name != "Тюмень" {
		t.Fatalf("sheet name %q", meta.Name)
	}
	if meta.HeaderRow != 14 || meta.ArticleColumn != 32 {
		t.Fatalf("wanted header row 14 col 32, got row %d col %d", meta.HeaderRow, meta.ArticleColumn)
	}
	if meta.MaxRow != 16 || meta.MaxColumn != 35 {
		t.Fatalf("bounds row=%d col=%d", meta.MaxRow, meta.MaxColumn)
	}
	if meta.Articles["RG01"] != 15 || meta.Articles["CT02"] != 16 {
		t.Fatalf("article index %#v", meta.Articles)
	}
}

func TestCaptureKeepsSheetAppearance(t *testing.T) {
	sheet := &styledFakeSheet{
		fakeSheet: fakeSheet{
			name: "Бланк",
			cells: map[[2]int]string{
				{1, 1}: "Артикул",
				{1, 2}: "Наименование",
				{2, 1}: "A100",
				{2, 3}: "",
			},
		},
		styles: []spreadsheet.Style{{}, {Fill: "#1f4e79", Bold: true, Color: "#ffffff"}},
		xf: map[[2]int]int{
			{1, 1}: 1,
			{1, 2}: 1,
			{2, 3}: 1,
		},
		columns:    []float64{135, 91, 91},
		rowHeight:  20,
		rowHeights: map[int]float64{1: 40},
		merges:     []spreadsheet.Merge{{Row: 1, Column: 1, Height: 1, Width: 2}},
	}
	snapshot := Capture(&fakeWorkbook{sheets: []spreadsheet.Sheet{sheet}})
	meta := snapshot.Meta.Sheets[0]
	if len(meta.Columns) != 3 || meta.Columns[0] != 135 || meta.RowHeight != 20 || meta.RowHeights[1] != 40 {
		t.Fatalf("layout %#v", meta)
	}
	if len(meta.Merges) != 1 || meta.Merges[0].Width != 2 {
		t.Fatalf("merges %#v", meta.Merges)
	}
	if len(meta.Styles) < 2 || meta.Styles[1].Fill != "#1f4e79" || !meta.Styles[1].Bold {
		t.Fatalf("catalog %#v", meta.Styles)
	}
	header := snapshot.Chunks[0][0].Rows[0]
	headerStyles := snapshot.Chunks[0][0].Styles[0]
	if header[0] != "Артикул" || headerStyles[0] != 1 || headerStyles[1] != 1 {
		t.Fatalf("header row=%#v styles=%#v", header, headerStyles)
	}
	dataStyles := snapshot.Chunks[0][0].Styles[1]
	if len(dataStyles) < 3 || dataStyles[2] != 1 {
		t.Fatalf("empty styled C2 must be kept, styles=%#v", dataStyles)
	}
	window, err := snapshot.Window(0, 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(window.Styles) != 2 || window.Styles[0][0] != 1 || window.Styles[1][2] != 1 {
		t.Fatalf("window styles %#v", window.Styles)
	}
}

type styledFakeSheet struct {
	fakeSheet
	styles     []spreadsheet.Style
	xf         map[[2]int]int
	columns    []float64
	rowHeight  float64
	rowHeights map[int]float64
	merges     []spreadsheet.Merge
}

func (s *styledFakeSheet) Styles() []spreadsheet.Style        { return s.styles }
func (s *styledFakeSheet) StyleIndex(row int, column int) int { return s.xf[[2]int{row, column}] }
func (s *styledFakeSheet) ColumnWidths() []float64            { return s.columns }
func (s *styledFakeSheet) DefaultRowHeight() float64          { return s.rowHeight }
func (s *styledFakeSheet) CustomRowHeights() map[int]float64  { return s.rowHeights }
func (s *styledFakeSheet) Merges() []spreadsheet.Merge        { return s.merges }

func TestCaptureSplitsRowsIntoChunks(t *testing.T) {
	sheet := &fakeSheet{name: "Лист1", cells: map[[2]int]string{
		{1, 1}: "Артикул",
		{2, 1}: "A1",
		{3, 1}: "A2",
		{4, 1}: "A3",
		{5, 1}: "A4",
	}}
	snapshot := CaptureWith(&fakeWorkbook{sheets: []spreadsheet.Sheet{sheet}}, 2)
	if snapshot.Meta.ChunkRows != 2 {
		t.Fatalf("chunk rows %d", snapshot.Meta.ChunkRows)
	}
	if len(snapshot.Chunks[0]) != 3 {
		t.Fatalf("expected 3 chunks for 5 rows, got %d", len(snapshot.Chunks[0]))
	}
	if snapshot.Chunks[0][0].StartRow != 1 || snapshot.Chunks[0][0].EndRow != 2 {
		t.Fatalf("first chunk %#v", snapshot.Chunks[0][0])
	}
	if got := snapshot.Chunks[0][0].Rows[1][0]; got != "A1" {
		t.Fatalf("row 2 col A = %q", got)
	}
}

func TestWindowSlicesAcrossChunkBoundaries(t *testing.T) {
	sheet := &fakeSheet{name: "Лист1", cells: map[[2]int]string{
		{1, 1}: "Артикул",
		{2, 1}: "A1",
		{3, 1}: "A2",
		{4, 1}: "A3",
	}}
	snapshot := CaptureWith(&fakeWorkbook{sheets: []spreadsheet.Sheet{sheet}}, 2)
	window, err := snapshot.Window(0, 2, 4)
	if err != nil {
		t.Fatal(err)
	}
	if window.FromRow != 2 || window.ToRow != 4 {
		t.Fatalf("window bounds %#v", window)
	}
	if len(window.Rows) != 3 || window.Rows[0][0] != "A1" || window.Rows[2][0] != "A3" {
		t.Fatalf("rows %#v", window.Rows)
	}
}

func TestFindArticleIsCaseInsensitive(t *testing.T) {
	snapshot := Capture(&fakeWorkbook{sheets: []spreadsheet.Sheet{sourceSheet()}})
	hit, ok := snapshot.FindArticle(0, "rg01")
	if !ok || hit.Row != 15 || hit.Column != 32 {
		t.Fatalf("find %#v ok=%v", hit, ok)
	}
	if _, ok := snapshot.FindArticle(0, "missing"); ok {
		t.Fatal("missing article should not match")
	}
}

func TestEncodeRoundTripGzipJSON(t *testing.T) {
	snapshot := Capture(&fakeWorkbook{sheets: []spreadsheet.Sheet{sourceSheet()}})
	objects, err := Encode(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if len(objects) < 2 {
		t.Fatalf("expected meta + chunks, got %d objects", len(objects))
	}
	if objects[0].Name != "meta.json.gz" {
		t.Fatalf("first object %q", objects[0].Name)
	}
	if !bytes.HasPrefix(objects[0].Content, []byte{0x1f, 0x8b}) {
		t.Fatal("preview objects must be gzip-compressed so api-service stays small")
	}
	restored, err := Decode(objects)
	if err != nil {
		t.Fatal(err)
	}
	if restored.Meta.Sheets[0].Articles["RG01"] != 15 {
		t.Fatalf("decoded articles %#v", restored.Meta.Sheets[0].Articles)
	}
	window, err := restored.Window(0, 15, 15)
	if err != nil {
		t.Fatal(err)
	}
	if window.Rows[0][31] != "RG01" || window.Rows[0][33] != "12" {
		t.Fatalf("decoded row %#v", window.Rows[0])
	}
}

func TestObjectKeysStayUnderTheOutputFile(t *testing.T) {
	if got := MetaKey("job-1", "output-2"); got != "jobs/job-1/preview/output-2/meta.json.gz" {
		t.Fatalf("meta key %q", got)
	}
	if got := ChunkKey("job-1", "output-2", 0, 3); got != "jobs/job-1/preview/output-2/s0/c3.json.gz" {
		t.Fatalf("chunk key %q", got)
	}
}

func TestBlankHeaderUsesTheArticleColumnOnTheNameRow(t *testing.T) {
	sheet := &fakeSheet{
		name: "Бланк",
		cells: map[[2]int]string{
			{1, 1}: "Артикул",
			{1, 2}: "Наименование",
			{1, 4}: "Кол-во",
			{2, 1}: "A100",
			{2, 2}: "Крем",
		},
	}
	snapshot := Capture(&fakeWorkbook{sheets: []spreadsheet.Sheet{sheet}})
	meta := snapshot.Meta.Sheets[0]
	if meta.HeaderRow != 1 || meta.ArticleColumn != 1 {
		t.Fatalf("blank header row=%d col=%d", meta.HeaderRow, meta.ArticleColumn)
	}
	if meta.Articles["A100"] != 2 {
		t.Fatalf("articles %#v", meta.Articles)
	}
}

func decodeGzipJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = reader.Close() }()
	var payload map[string]any
	if err := json.NewDecoder(reader).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func TestEncodedChunkOmitsTrailingEmptyCells(t *testing.T) {
	sheet := &fakeSheet{name: "Лист1", cells: map[[2]int]string{
		{1, 1}: "Артикул",
		{1, 8}: "хвост",
		{2, 1}: "A1",
	}}
	snapshot := Capture(&fakeWorkbook{sheets: []spreadsheet.Sheet{sheet}})
	objects, err := Encode(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	payload := decodeGzipJSON(t, objects[1].Content)
	rows, ok := payload["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatalf("rows %#v", payload["rows"])
	}
	first, _ := rows[0].([]any)
	if len(first) != 8 {
		t.Fatalf("row should trim to last filled cell, got %d cells: %#v", len(first), first)
	}
	encoded, _ := json.Marshal(payload)
	if strings.Count(string(encoded), `""`) > 20 {
		t.Fatalf("chunk still looks dense: %s", encoded)
	}
}
