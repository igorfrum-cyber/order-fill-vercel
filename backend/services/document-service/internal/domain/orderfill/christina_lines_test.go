package orderfill

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// grid helper: column 1 = article, column 2 = name.
func proffGrid(rows []struct{ article, name string }) [][]string {
	grid := make([][]string, len(rows))
	for i, r := range rows {
		grid[i] = []string{r.article, r.name}
	}
	return grid
}

// TestChristinaProffLinesByRow_NoStyles uses a plain fakeSheet (no Styled),
// so the fallback path (no colour check) is exercised.
func TestChristinaProffLinesByRow_NoStyles(t *testing.T) {
	// Layout:
	//   row 1: "" / "MUSE" header  → new line
	//   row 2: CHR001 / "Product A"
	//   row 3: CHR002 / "Product B"
	//   row 4: "" / "ОБЩАЯ ЛИНИЯ"  → end current line
	//   row 5: CHR003 / "Product C" → no current group, skipped
	grid := proffGrid([]struct{ article, name string }{
		{"", "MUSE"},
		{"CHR001", "Product A"},
		{"CHR002", "Product B"},
		{"", "ОБЩАЯ ЛИНИЯ"},
		{"CHR003", "Product C"},
	})
	sheet := newFakeWorkbook("Sheet1", grid).sheets[0]
	result := ChristinaProffLinesByRow(sheet)

	if len(result) != 2 {
		t.Fatalf("want 2 mapped rows, got %d", len(result))
	}
	line2, ok := result[2]
	if !ok {
		t.Fatal("row 2 should be mapped")
	}
	if line2.ID != "MUSE" {
		t.Errorf("want ID MUSE, got %q", line2.ID)
	}
	if line2.Article != "CHR001" {
		t.Errorf("want Article CHR001, got %q", line2.Article)
	}
	if len(line2.Required) != 2 {
		t.Errorf("want Required [CHR001 CHR002], got %v", line2.Required)
	}

	line3, ok := result[3]
	if !ok {
		t.Fatal("row 3 should be mapped")
	}
	if line3.Article != "CHR002" {
		t.Errorf("want Article CHR002, got %q", line3.Article)
	}
	// Both rows share the same Required list.
	if len(line3.Required) != 2 {
		t.Errorf("row 3: want Required len 2, got %v", line3.Required)
	}

	if _, present := result[5]; present {
		t.Error("row 5 is after ОБЩАЯ ЛИНИЯ reset, should not be mapped")
	}
}

// TestChristinaProffLinesByRow_MultipleLines checks two distinct line groups.
func TestChristinaProffLinesByRow_MultipleLines(t *testing.T) {
	grid := proffGrid([]struct{ article, name string }{
		{"", "MUSE"},
		{"CHR001", "Product A"},
		{"", "NANO"},
		{"CHR010", "Product X"},
		{"CHR011", "Product Y"},
	})
	sheet := newFakeWorkbook("Sheet1", grid).sheets[0]
	result := ChristinaProffLinesByRow(sheet)

	if len(result) != 3 {
		t.Fatalf("want 3 mapped rows, got %d", len(result))
	}

	muse := result[2]
	if muse == nil || muse.ID != "MUSE" {
		t.Errorf("row 2: want MUSE, got %v", muse)
	}
	if len(muse.Required) != 1 {
		t.Errorf("MUSE should have 1 required article, got %v", muse.Required)
	}

	nano4 := result[4]
	if nano4 == nil || nano4.ID != "NANO" {
		t.Errorf("row 4: want NANO, got %v", nano4)
	}
	nano5 := result[5]
	if nano5 == nil || nano5.ID != "NANO" {
		t.Errorf("row 5: want NANO, got %v", nano5)
	}
	if len(nano4.Required) != 2 {
		t.Errorf("NANO should have 2 required articles, got %v", nano4.Required)
	}
}

// TestChristinaProffLinesByRow_NuanceNoHeader verifies that a Nuance product
// found without an explicit line header creates its own group, matching the
// September PROFF blank quirk documented in origin/main.
func TestChristinaProffLinesByRow_NuanceNoHeader(t *testing.T) {
	grid := proffGrid([]struct{ article, name string }{
		{"", "MUSE"},
		{"CHR001", "Product A"},
		{"CHR002", "Nuance Cream"},
		{"CHR003", "Nuance Serum"},
	})
	sheet := newFakeWorkbook("Sheet1", grid).sheets[0]
	result := ChristinaProffLinesByRow(sheet)

	// CHR001 → MUSE; CHR002 & CHR003 → NUANCE (auto-created).
	if len(result) != 3 {
		t.Fatalf("want 3 mapped rows, got %d", len(result))
	}
	if result[2].ID != "MUSE" {
		t.Errorf("row 2: want MUSE, got %q", result[2].ID)
	}
	if result[3].ID != "NUANCE" {
		t.Errorf("row 3: want NUANCE, got %q", result[3].ID)
	}
	if result[4].ID != "NUANCE" {
		t.Errorf("row 4: want NUANCE, got %q", result[4].ID)
	}
}

// TestChristinaProffLinesByRow_EmptySheet should return an empty map.
func TestChristinaProffLinesByRow_EmptySheet(t *testing.T) {
	sheet := newFakeWorkbook("Sheet1", [][]string{}).sheets[0]
	result := ChristinaProffLinesByRow(sheet)
	if len(result) != 0 {
		t.Errorf("want empty map, got %d entries", len(result))
	}
}

// TestChristinaProffLinesByRow_LineHeaderLabel checks trailing-separator strip.
func TestLineHeaderLabel(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"MUSE", "MUSE"},
		{"MUSE — 4 линии", "MUSE"},
		{"NANO - набор", "NANO"},
		{"SERENITY–5", "SERENITY"},
	}
	for _, tc := range cases {
		got := lineHeaderLabel(tc.input)
		if got != tc.want {
			t.Errorf("lineHeaderLabel(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
