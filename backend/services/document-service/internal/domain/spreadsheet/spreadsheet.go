// Package spreadsheet defines the workbook port used by the domain layer.
// It carries no knowledge of xlsx, zip or XML: adapters implement it.
package spreadsheet

// Workbook is a mutable spreadsheet loaded from a supplier or 1C file.
// Implementations must preserve every part of the original document that the
// domain does not explicitly change, because users download the same file back.
type Workbook interface {
	Sheets() []Sheet
	Sheet(name string) (Sheet, bool)
	Save() ([]byte, error)
}

// Sheet exposes a single worksheet as a 1-based cell grid.
type Sheet interface {
	Name() string
	Bounds() Bounds
	Value(row int, column int) string
	SetNumber(row int, column int, value float64)
	ClearValue(row int, column int)
	SetText(row int, column int, value string)
	// DeleteRows removes the given 1-based rows and moves every row below them
	// up, so that row numbers stay contiguous. The Ч3 merge relies on it.
	DeleteRows(rows []int)
}

// Bounds is the used range of a sheet. Zero values mean an empty sheet.
type Bounds struct {
	MaxRow    int
	MaxColumn int
}

// Codec loads workbooks from raw bytes.
type Codec interface {
	Load(content []byte) (Workbook, error)
}

// LoadProgress reports a 0..1 fraction of workbook loading.
type LoadProgress func(fraction float64)

// ProgressCodec is implemented by the xlsx adapter so the worker can show
// honest progress while a large sheet is inflated and parsed.
type ProgressCodec interface {
	LoadWithProgress(content []byte, report LoadProgress) (Workbook, error)
}

// ColumnName converts a 1-based column index into its spreadsheet letters.
func ColumnName(column int) string {
	name := ""
	for current := column; current > 0; {
		remainder := (current - 1) % 26
		name = string(rune('A'+remainder)) + name
		current = (current - 1) / 26
	}
	return name
}

// ParseColumnName converts spreadsheet letters into a 1-based column index.
func ParseColumnName(name string) int {
	column := 0
	for _, char := range name {
		if char >= 'a' && char <= 'z' {
			char -= 'a' - 'A'
		}
		if char < 'A' || char > 'Z' {
			return 0
		}
		column = column*26 + int(char-'A'+1)
	}
	return column
}
