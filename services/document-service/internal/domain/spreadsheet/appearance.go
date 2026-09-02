package spreadsheet

// Styled is implemented by workbook adapters that can describe how a sheet looks.
// The fill engine does not use it; preview capture type-asserts when present.
type Styled interface {
	Styles() []Style
	StyleIndex(row int, column int) int
	ColumnWidths() []float64
	DefaultRowHeight() float64
	CustomRowHeights() map[int]float64
	Merges() []Merge
}

// Style is the resolved appearance of a cell. Empty fields mean Excel defaults.
type Style struct {
	Fill    string
	Color   string
	Bold    bool
	Italic  bool
	Size    int
	Align   string
	Valign  string
	Wrap    bool
	BorderT bool
	BorderR bool
	BorderB bool
	BorderL bool
}

// Merge is a 1-based rectangular span. Height and Width are at least 1.
type Merge struct {
	Row    int
	Column int
	Height int
	Width  int
}
