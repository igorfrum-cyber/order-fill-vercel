package xlsx

import (
	"testing"

	"github.com/beevik/etree"
)

func TestShiftFormulaMovesRelativeRefs(t *testing.T) {
	got := shiftFormula("B2*2", cellKey{row: 2, column: 3}, cellKey{row: 3, column: 3})
	if got != "B3*2" {
		t.Fatalf("got %q", got)
	}
	got = shiftFormula("SUM(E2:E10)", cellKey{row: 11, column: 5}, cellKey{row: 21, column: 5})
	if got != "SUM(E12:E20)" {
		t.Fatalf("got %q", got)
	}
	got = shiftFormula("$B$2+B2", cellKey{row: 2, column: 3}, cellKey{row: 4, column: 3})
	if got != "$B$2+B4" {
		t.Fatalf("got %q", got)
	}
}

func TestExpandSharedFormulasFillsSiblings(t *testing.T) {
	master := etree.NewElement("c")
	formula := master.CreateElement("f")
	formula.CreateAttr("t", "shared")
	formula.CreateAttr("si", "0")
	formula.SetText("B2*2")
	sibling := etree.NewElement("c")
	shared := sibling.CreateElement("f")
	shared.CreateAttr("t", "shared")
	shared.CreateAttr("si", "0")
	cells := map[cellKey]*cell{
		{row: 2, column: 3}: {element: master, formula: "B2*2"},
		{row: 3, column: 3}: {element: sibling},
	}
	expandSharedFormulas(cells)
	if got := cells[cellKey{row: 3, column: 3}].formula; got != "B3*2" {
		t.Fatalf("sibling formula %q", got)
	}
}
