package preview

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"reflect"
	"testing"
)

func TestObjectKeys(t *testing.T) {
	t.Parallel()

	if got, want := MetaKey("job-1", "file-2"), "jobs/job-1/preview/file-2/meta.json.gz"; got != want {
		t.Fatalf("MetaKey() = %q, want %q", got, want)
	}
	if got, want := ChunkKey("job-1", "file-2", 3, 4), "jobs/job-1/preview/file-2/s3/c4.json.gz"; got != want {
		t.Fatalf("ChunkKey() = %q, want %q", got, want)
	}
}

func TestDecodeMetaAndChunk(t *testing.T) {
	t.Parallel()

	metaBytes := gzipJSONForTest(t, Meta{Sheets: []SheetMeta{{Name: "Sheet1", Index: 0}}})
	meta, err := DecodeMeta(metaBytes)
	if err != nil {
		t.Fatal(err)
	}
	if meta.ChunkRows != DefaultChunkRows || len(meta.Sheets) != 1 {
		t.Fatalf("DecodeMeta() = %+v", meta)
	}

	chunkBytes := gzipJSONForTest(t, Chunk{StartRow: 1, EndRow: 1, Rows: [][]string{{"A"}}})
	chunk, err := DecodeChunk(chunkBytes)
	if err != nil {
		t.Fatal(err)
	}
	if chunk.StartRow != 1 || !reflect.DeepEqual(chunk.Rows, [][]string{{"A"}}) {
		t.Fatalf("DecodeChunk() = %+v", chunk)
	}
}

func TestDecodeErrors(t *testing.T) {
	t.Parallel()

	if _, err := DecodeMeta([]byte("not gzip")); err == nil {
		t.Fatal("DecodeMeta() accepted invalid gzip")
	}
	if _, err := DecodeChunk(gzipJSONForTest(t, "not an object")); err == nil {
		t.Fatal("DecodeChunk() accepted invalid JSON")
	}
	corrupt := gzipJSONForTest(t, map[string]string{"valid": "json"})
	corrupt[len(corrupt)-8] ^= 0xff
	if _, err := DecodeMeta(corrupt); err == nil {
		t.Fatal("DecodeMeta() accepted a gzip checksum error")
	}
}

func TestChunkIndex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		row       int
		chunkRows int
		want      int
	}{
		{name: "first row", row: 1, chunkRows: 10, want: 0},
		{name: "last row in chunk", row: 10, chunkRows: 10, want: 0},
		{name: "first row in next chunk", row: 11, chunkRows: 10, want: 1},
		{name: "row clamps to one", row: 0, chunkRows: 10, want: 0},
		{name: "chunk size defaults", row: 257, chunkRows: 0, want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ChunkIndex(tt.row, tt.chunkRows); got != tt.want {
				t.Fatalf("ChunkIndex(%d, %d) = %d, want %d", tt.row, tt.chunkRows, got, tt.want)
			}
		})
	}
}

func TestAssembleWindow(t *testing.T) {
	t.Parallel()

	sheet := SheetMeta{MaxRow: 4, MaxColumn: 3, Styles: []CellStyle{{Bold: true}}}
	chunks := map[int]Chunk{
		0: {StartRow: 1, Rows: [][]string{{"A"}, {"B", "C"}}, Styles: [][]int{{1}, {2, 3}}},
		1: {StartRow: 3, Rows: [][]string{{"D", "E", "F"}}, Styles: [][]int{{4}}},
	}
	window := AssembleWindow(sheet, 2, chunks, 0, 99)
	if window.FromRow != 1 || window.ToRow != 4 {
		t.Fatalf("window bounds = %d..%d", window.FromRow, window.ToRow)
	}
	wantRows := [][]string{{"A", "", ""}, {"B", "C", ""}, {"D", "E", "F"}, {"", "", ""}}
	wantStyles := [][]int{{1, 0, 0}, {2, 3, 0}, {4, 0, 0}, {0, 0, 0}}
	if !reflect.DeepEqual(window.Rows, wantRows) || !reflect.DeepEqual(window.Styles, wantStyles) {
		t.Fatalf("window rows/styles = %#v / %#v", window.Rows, window.Styles)
	}

	empty := AssembleWindow(SheetMeta{MaxColumn: 0}, 0, nil, 4, 2)
	if empty.FromRow != 4 || empty.ToRow != 2 || len(empty.Rows) != 0 {
		t.Fatalf("invalid window bounds = %+v", empty)
	}
	withoutStyles := AssembleWindow(SheetMeta{MaxRow: 1, MaxColumn: 0}, 1, nil, 1, 1)
	if len(withoutStyles.Rows) != 1 || withoutStyles.Rows[0] == nil || withoutStyles.Styles != nil {
		t.Fatalf("zero-width window = %+v", withoutStyles)
	}
	missingCellsAndStyles := AssembleWindow(
		SheetMeta{MaxRow: 1, MaxColumn: 0, Styles: []CellStyle{{Bold: true}}},
		1,
		nil,
		1,
		1,
	)
	if !reflect.DeepEqual(missingCellsAndStyles.Styles, [][]int{{}}) {
		t.Fatalf("zero-width style window = %+v", missingCellsAndStyles)
	}
}

func TestFindArticle(t *testing.T) {
	t.Parallel()

	meta := SheetMeta{ArticleColumn: 2, Articles: map[string]int{"AB 12": 7}}
	if got := meta.FindArticle("AB 12"); got != (Hit{Found: true, Row: 7, Column: 2}) {
		t.Fatalf("exact match = %+v", got)
	}
	if got := meta.FindArticle(" ab12 "); got != (Hit{Found: true, Row: 7, Column: 2}) {
		t.Fatalf("folded match = %+v", got)
	}
	for _, query := range []string{"", "missing"} {
		if got := meta.FindArticle(query); got.Found {
			t.Fatalf("FindArticle(%q) = %+v, want no hit", query, got)
		}
	}
	if got := (SheetMeta{Articles: meta.Articles}).FindArticle("AB 12"); got.Found {
		t.Fatalf("match without article column = %+v", got)
	}
	if got := (SheetMeta{ArticleColumn: 1}).FindArticle("AB 12"); got.Found {
		t.Fatalf("match without article index = %+v", got)
	}
}

func gzipJSONForTest(t *testing.T, value any) []byte {
	t.Helper()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if err := json.NewEncoder(writer).Encode(value); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return compressed.Bytes()
}
