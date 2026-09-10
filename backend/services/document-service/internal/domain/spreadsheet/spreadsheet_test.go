package spreadsheet

import "testing"

func TestColumnNameRoundTrip(t *testing.T) {
	t.Parallel()
	tests := []struct {
		column int
		name   string
	}{
		{column: 1, name: "A"},
		{column: 26, name: "Z"},
		{column: 27, name: "AA"},
		{column: 702, name: "ZZ"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ColumnName(tt.column); got != tt.name {
				t.Fatalf("ColumnName(%d) = %q, want %q", tt.column, got, tt.name)
			}
			if got := ParseColumnName(tt.name); got != tt.column {
				t.Fatalf("ParseColumnName(%q) = %d, want %d", tt.name, got, tt.column)
			}
		})
	}
	if got := ParseColumnName("a"); got != 1 {
		t.Fatalf("ParseColumnName(a) = %d, want 1", got)
	}
	if got := ParseColumnName("A1"); got != 0 {
		t.Fatalf("invalid name = %d, want 0", got)
	}
}
