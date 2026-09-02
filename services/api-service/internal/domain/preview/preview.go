package preview

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"
)

// MaxWindowRows is the largest range api-service will inflate in one request.
// Two 256-row chunks already cover a viewport; 200 is a hard cap against
// accidental full-sheet fetches that would blow the 256 MiB process limit.
const MaxWindowRows = 200

// DefaultChunkRows matches document-service/internal/domain/preview.
const DefaultChunkRows = 256

// Meta is the gzipped sidecar written next to a generated workbook.
type Meta struct {
	ChunkRows int         `json:"chunk_rows"`
	Sheets    []SheetMeta `json:"sheets"`
}

// SheetMeta is the used range plus an article→row index.
type SheetMeta struct {
	Name          string          `json:"name"`
	Index         int             `json:"index"`
	MaxRow        int             `json:"max_row"`
	MaxColumn     int             `json:"max_column"`
	HeaderRow     int             `json:"header_row,omitempty"`
	ArticleColumn int             `json:"article_column,omitempty"`
	Articles      map[string]int  `json:"articles,omitempty"`
	Styles        []CellStyle     `json:"styles,omitempty"`
	Columns       []float64       `json:"columns,omitempty"`
	RowHeight     float64         `json:"row_height,omitempty"`
	RowHeights    map[int]float64 `json:"row_heights,omitempty"`
	Merges        []Merge         `json:"merges,omitempty"`
}

// CellStyle is one interned appearance. Empty fields mean Excel defaults.
type CellStyle struct {
	Fill    string `json:"fill,omitempty"`
	Color   string `json:"color,omitempty"`
	Bold    bool   `json:"bold,omitempty"`
	Italic  bool   `json:"italic,omitempty"`
	Size    int    `json:"size,omitempty"`
	Align   string `json:"align,omitempty"`
	Valign  string `json:"valign,omitempty"`
	Wrap    bool   `json:"wrap,omitempty"`
	BorderT bool   `json:"border_t,omitempty"`
	BorderR bool   `json:"border_r,omitempty"`
	BorderB bool   `json:"border_b,omitempty"`
	BorderL bool   `json:"border_l,omitempty"`
}

// Merge is a 1-based rectangular span.
type Merge struct {
	Row    int `json:"row"`
	Column int `json:"column"`
	Height int `json:"height"`
	Width  int `json:"width"`
}

// Chunk is one gzipped row window. Rows are trimmed of trailing empty cells.
type Chunk struct {
	StartRow int        `json:"start_row"`
	EndRow   int        `json:"end_row"`
	Rows     [][]string `json:"rows"`
	Styles   [][]int    `json:"styles,omitempty"`
}

// Window is a dense row slice for the browser grid.
type Window struct {
	FromRow int        `json:"from_row"`
	ToRow   int        `json:"to_row"`
	Rows    [][]string `json:"rows"`
	Styles  [][]int    `json:"styles,omitempty"`
}

// Hit is a cell matched by article search.
type Hit struct {
	Found  bool `json:"found"`
	Row    int  `json:"row,omitempty"`
	Column int  `json:"column,omitempty"`
}

// MetaKey is the object-store path of the gzipped sheet index.
func MetaKey(jobID string, fileID string) string {
	return fmt.Sprintf("jobs/%s/preview/%s/meta.json.gz", jobID, fileID)
}

// ChunkKey is the object-store path of one gzipped row window.
func ChunkKey(jobID string, fileID string, sheetIndex int, chunkIndex int) string {
	return fmt.Sprintf("jobs/%s/preview/%s/s%d/c%d.json.gz", jobID, fileID, sheetIndex, chunkIndex)
}

// DecodeMeta inflates a gzipped meta object.
func DecodeMeta(raw []byte) (Meta, error) {
	var meta Meta
	if err := gunzipJSON(raw, &meta); err != nil {
		return Meta{}, err
	}
	if meta.ChunkRows < 1 {
		meta.ChunkRows = DefaultChunkRows
	}
	return meta, nil
}

// DecodeChunk inflates one gzipped row window.
func DecodeChunk(raw []byte) (Chunk, error) {
	var chunk Chunk
	if err := gunzipJSON(raw, &chunk); err != nil {
		return Chunk{}, err
	}
	return chunk, nil
}

// ChunkIndex is the 0-based chunk that contains a 1-based sheet row.
func ChunkIndex(row int, chunkRows int) int {
	if row < 1 {
		row = 1
	}
	if chunkRows < 1 {
		chunkRows = DefaultChunkRows
	}
	return (row - 1) / chunkRows
}

// AssembleWindow pads trimmed chunk rows into a dense from_row..to_row slice.
func AssembleWindow(sheet SheetMeta, chunkRows int, chunks map[int]Chunk, fromRow int, toRow int) Window {
	if fromRow < 1 {
		fromRow = 1
	}
	if sheet.MaxRow > 0 && toRow > sheet.MaxRow {
		toRow = sheet.MaxRow
	}
	if toRow < fromRow {
		return Window{FromRow: fromRow, ToRow: toRow, Rows: [][]string{}}
	}
	width := sheet.MaxColumn
	rows := make([][]string, 0, toRow-fromRow+1)
	withStyles := len(sheet.Styles) > 0
	var styles [][]int
	if withStyles {
		styles = make([][]int, 0, toRow-fromRow+1)
	}
	for row := fromRow; row <= toRow; row++ {
		rows = append(rows, denseRow(cellsAt(chunks, chunkRows, row), width))
		if withStyles {
			styles = append(styles, denseInts(stylesAt(chunks, chunkRows, row), width))
		}
	}
	return Window{FromRow: fromRow, ToRow: toRow, Rows: rows, Styles: styles}
}

func cellsAt(chunks map[int]Chunk, chunkRows int, row int) []string {
	chunk, ok := chunks[ChunkIndex(row, chunkRows)]
	if !ok {
		return nil
	}
	offset := row - chunk.StartRow
	if offset < 0 || offset >= len(chunk.Rows) {
		return nil
	}
	return chunk.Rows[offset]
}

func stylesAt(chunks map[int]Chunk, chunkRows int, row int) []int {
	chunk, ok := chunks[ChunkIndex(row, chunkRows)]
	if !ok {
		return nil
	}
	offset := row - chunk.StartRow
	if offset < 0 || offset >= len(chunk.Styles) {
		return nil
	}
	return chunk.Styles[offset]
}

func denseRow(cells []string, width int) []string {
	if width < 1 {
		return []string{}
	}
	out := make([]string, width)
	copy(out, cells)
	return out
}

func denseInts(cells []int, width int) []int {
	if width < 1 {
		return []int{}
	}
	out := make([]int, width)
	copy(out, cells)
	return out
}

// FindArticle returns the first data row for a SKU.
func (m SheetMeta) FindArticle(query string) Hit {
	needle := strings.TrimSpace(query)
	if needle == "" || m.ArticleColumn < 1 || len(m.Articles) == 0 {
		return Hit{}
	}
	if row, ok := m.Articles[needle]; ok {
		return Hit{Found: true, Row: row, Column: m.ArticleColumn}
	}
	folded := foldArticle(needle)
	for article, row := range m.Articles {
		if foldArticle(article) == folded {
			return Hit{Found: true, Row: row, Column: m.ArticleColumn}
		}
	}
	return Hit{}
}

func foldArticle(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, value)
}

func gunzipJSON(raw []byte, dest any) error {
	reader, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	decoded, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	return json.Unmarshal(decoded, dest)
}
