package preview

import (
	"strings"
	"unicode"

	"order-fill/services/document-service/internal/domain/spreadsheet"
)

// DefaultChunkRows is small enough that one or two chunks fit the api-service
// memory budget even for a 35-column 1C sheet, and large enough that a typical
// viewport (~80 rows) is one request.
const DefaultChunkRows = 256

const headerScanRows = 50

// Snapshot is an in-memory grid ready to encode into sidecar objects.
type Snapshot struct {
	Meta   Meta
	Chunks [][]Chunk
}

// Meta describes every sheet so the browser can size the grid before any cells
// are fetched.
type Meta struct {
	ChunkRows int         `json:"chunk_rows"`
	Sheets    []SheetMeta `json:"sheets"`
}

// SheetMeta is the used range plus an article→row index for jump-to-SKU.
type SheetMeta struct {
	Name          string         `json:"name"`
	Index         int            `json:"index"`
	MaxRow        int            `json:"max_row"`
	MaxColumn     int            `json:"max_column"`
	HeaderRow     int            `json:"header_row,omitempty"`
	ArticleColumn int            `json:"article_column,omitempty"`
	Articles      map[string]int `json:"articles,omitempty"`
}

// Chunk is a contiguous row window. Rows[i] is sheet row StartRow+i; the inner
// slice is 0-based columns (index 0 = column A) trimmed of trailing empties.
type Chunk struct {
	StartRow int        `json:"start_row"`
	EndRow   int        `json:"end_row"`
	Rows     [][]string `json:"rows"`
}

// Window is a slice of the grid returned to the browser.
type Window struct {
	FromRow int        `json:"from_row"`
	ToRow   int        `json:"to_row"`
	Rows    [][]string `json:"rows"`
}

// Hit is a cell matched by article search.
type Hit struct {
	Row    int
	Column int
}

// Capture builds a snapshot with the default chunk size.
func Capture(workbook spreadsheet.Workbook) Snapshot {
	return CaptureWith(workbook, DefaultChunkRows)
}

// CaptureWith splits every sheet into chunks of the given height.
func CaptureWith(workbook spreadsheet.Workbook, chunkRows int) Snapshot {
	if chunkRows < 1 {
		chunkRows = DefaultChunkRows
	}
	sheets := workbook.Sheets()
	snapshot := Snapshot{
		Meta:   Meta{ChunkRows: chunkRows, Sheets: make([]SheetMeta, 0, len(sheets))},
		Chunks: make([][]Chunk, 0, len(sheets)),
	}
	for index, sheet := range sheets {
		meta, chunks := captureSheet(sheet, index, chunkRows)
		snapshot.Meta.Sheets = append(snapshot.Meta.Sheets, meta)
		snapshot.Chunks = append(snapshot.Chunks, chunks)
	}
	return snapshot
}

func captureSheet(sheet spreadsheet.Sheet, index int, chunkRows int) (SheetMeta, []Chunk) {
	bounds := sheet.Bounds()
	headerRow, articleCol := detectHeader(sheet, bounds)
	meta := SheetMeta{
		Name:          sheet.Name(),
		Index:         index,
		MaxRow:        bounds.MaxRow,
		MaxColumn:     bounds.MaxColumn,
		HeaderRow:     headerRow,
		ArticleColumn: articleCol,
	}
	if articleCol > 0 && headerRow > 0 && bounds.MaxRow > headerRow {
		meta.Articles = indexArticles(sheet, headerRow, articleCol, bounds.MaxRow)
	}
	if bounds.MaxRow == 0 {
		return meta, nil
	}
	chunkCount := (bounds.MaxRow + chunkRows - 1) / chunkRows
	chunks := make([]Chunk, 0, chunkCount)
	for start := 1; start <= bounds.MaxRow; start += chunkRows {
		end := start + chunkRows - 1
		if end > bounds.MaxRow {
			end = bounds.MaxRow
		}
		rows := make([][]string, 0, end-start+1)
		for row := start; row <= end; row++ {
			rows = append(rows, captureRow(sheet, row, bounds.MaxColumn))
		}
		chunks = append(chunks, Chunk{StartRow: start, EndRow: end, Rows: rows})
	}
	return meta, chunks
}

func captureRow(sheet spreadsheet.Sheet, row int, maxColumn int) []string {
	cells := make([]string, maxColumn)
	last := -1
	for column := 1; column <= maxColumn; column++ {
		value := sheet.Value(row, column)
		cells[column-1] = value
		if value != "" {
			last = column - 1
		}
	}
	if last < 0 {
		return []string{}
	}
	return cells[:last+1]
}

func detectHeader(sheet spreadsheet.Sheet, bounds spreadsheet.Bounds) (int, int) {
	limit := bounds.MaxRow
	if limit > headerScanRows {
		limit = headerScanRows
	}
	type hit struct{ row, col int }
	var hits []hit
	for row := 1; row <= limit; row++ {
		for col := 1; col <= bounds.MaxColumn; col++ {
			if isArticleHeader(sheet.Value(row, col)) {
				hits = append(hits, hit{row, col})
			}
		}
	}
	if len(hits) == 0 {
		return 0, 0
	}
	for i := len(hits) - 1; i >= 0; i-- {
		if rowLooksLikeOrderHeader(sheet, hits[i].row, bounds.MaxColumn) {
			return hits[i].row, hits[i].col
		}
	}
	last := hits[len(hits)-1]
	return last.row, last.col
}

func rowLooksLikeOrderHeader(sheet spreadsheet.Sheet, row int, maxColumn int) bool {
	for col := 1; col <= maxColumn; col++ {
		folded := foldHeader(sheet.Value(row, col))
		if strings.Contains(folded, "заказано") ||
			strings.Contains(folded, "комментарий") ||
			strings.Contains(folded, "наименование") ||
			folded == "товар" ||
			strings.HasPrefix(folded, "товар ") {
			return true
		}
	}
	return false
}

func isArticleHeader(value string) bool {
	return foldHeader(value) == "артикул"
}

func foldHeader(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func indexArticles(sheet spreadsheet.Sheet, headerRow int, articleCol int, maxRow int) map[string]int {
	articles := make(map[string]int)
	for row := headerRow + 1; row <= maxRow; row++ {
		article := strings.TrimSpace(sheet.Value(row, articleCol))
		if article == "" {
			continue
		}
		if _, exists := articles[article]; exists {
			continue
		}
		articles[article] = row
	}
	return articles
}

// Window returns the inclusive row range of one sheet. Missing cells are empty
// strings so the browser can address columns by index without a sparse map.
func (s Snapshot) Window(sheetIndex int, fromRow int, toRow int) (Window, error) {
	if sheetIndex < 0 || sheetIndex >= len(s.Meta.Sheets) {
		return Window{}, errUnknownSheet
	}
	meta := s.Meta.Sheets[sheetIndex]
	if fromRow < 1 {
		fromRow = 1
	}
	if toRow < fromRow {
		toRow = fromRow
	}
	if meta.MaxRow > 0 && toRow > meta.MaxRow {
		toRow = meta.MaxRow
	}
	if fromRow > toRow {
		return Window{FromRow: fromRow, ToRow: toRow, Rows: [][]string{}}, nil
	}
	width := meta.MaxColumn
	rows := make([][]string, 0, toRow-fromRow+1)
	for row := fromRow; row <= toRow; row++ {
		rows = append(rows, denseRow(s.rowCells(sheetIndex, row), width))
	}
	return Window{FromRow: fromRow, ToRow: toRow, Rows: rows}, nil
}

func (s Snapshot) rowCells(sheetIndex int, row int) []string {
	if sheetIndex < 0 || sheetIndex >= len(s.Chunks) {
		return nil
	}
	chunkRows := s.Meta.ChunkRows
	if chunkRows < 1 {
		chunkRows = DefaultChunkRows
	}
	index := (row - 1) / chunkRows
	chunks := s.Chunks[sheetIndex]
	if index < 0 || index >= len(chunks) {
		return nil
	}
	chunk := chunks[index]
	offset := row - chunk.StartRow
	if offset < 0 || offset >= len(chunk.Rows) {
		return nil
	}
	return chunk.Rows[offset]
}

func denseRow(cells []string, width int) []string {
	if width < 1 {
		return []string{}
	}
	out := make([]string, width)
	copy(out, cells)
	return out
}

// FindArticle looks up a SKU in the sheet's article index.
func (s Snapshot) FindArticle(sheetIndex int, query string) (Hit, bool) {
	if sheetIndex < 0 || sheetIndex >= len(s.Meta.Sheets) {
		return Hit{}, false
	}
	return s.Meta.Sheets[sheetIndex].FindArticle(query)
}

// FindArticle returns the first data row for a SKU.
func (m SheetMeta) FindArticle(query string) (Hit, bool) {
	needle := strings.TrimSpace(query)
	if needle == "" || m.ArticleColumn < 1 || len(m.Articles) == 0 {
		return Hit{}, false
	}
	if row, ok := m.Articles[needle]; ok {
		return Hit{Row: row, Column: m.ArticleColumn}, true
	}
	folded := foldArticle(needle)
	for article, row := range m.Articles {
		if foldArticle(article) == folded {
			return Hit{Row: row, Column: m.ArticleColumn}, true
		}
	}
	return Hit{}, false
}

func foldArticle(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, value)
}
