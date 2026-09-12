package xlsx

import "testing"

func TestXMLOpenTagsOKRejectsOverBudget(t *testing.T) {
	t.Parallel()
	if xmlOpenTagsOK([]byte("<r><c/><c/></r>"), 3) {
		t.Fatal("expected over-budget xml to be rejected")
	}
	if !xmlOpenTagsOK([]byte("<r><c/></r>"), 3) {
		t.Fatal("expected xml within budget")
	}
}

func TestParseCellRefRejectsOverflowRow(t *testing.T) {
	t.Parallel()
	if _, ok := parseCellRef("A1048577"); ok {
		t.Fatal("expected overflow row to be rejected")
	}
}
