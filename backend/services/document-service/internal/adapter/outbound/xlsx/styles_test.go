package xlsx

import "testing"

func TestNormalizeHexStripsAlphaAndHash(t *testing.T) {
	if got := normalizeHex("FF1F4E79"); got != "#1f4e79" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeHex("#AABBCC"); got != "#aabbcc" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyTintDarkensAndLightens(t *testing.T) {
	darker := applyTint("#808080", -0.5)
	if darker != "#404040" {
		t.Fatalf("darker %q", darker)
	}
	lighter := applyTint("#808080", 0.5)
	if lighter != "#c0c0c0" {
		t.Fatalf("lighter %q", lighter)
	}
}

func TestParseStyleBookInternsSolidFillAndBold(t *testing.T) {
	styles := []byte(`<?xml version="1.0"?>
<styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <fonts count="2">
    <font><sz val="11"/><name val="Calibri"/></font>
    <font><b/><sz val="14"/><color rgb="FFFFFFFF"/><name val="Calibri"/></font>
  </fonts>
  <fills count="3">
    <fill><patternFill patternType="none"/></fill>
    <fill><patternFill patternType="gray125"/></fill>
    <fill><patternFill patternType="solid"><fgColor rgb="FF1F4E79"/></patternFill></fill>
  </fills>
  <borders count="2">
    <border/>
    <border><left style="thin"/><right style="thin"/><top style="thin"/><bottom style="thin"/></border>
  </borders>
  <cellXfs count="3">
    <xf fontId="0" fillId="0" borderId="0"/>
    <xf fontId="1" fillId="2" borderId="1"><alignment horizontal="center" wrapText="1"/></xf>
    <xf fontId="1" fillId="2" borderId="1"><alignment horizontal="center" wrapText="1"/></xf>
  </cellXfs>
</styleSheet>`)
	book := parseStyleBook(styles, nil)
	if len(book.catalog) != 2 {
		t.Fatalf("expected default + one interned style, got %d %#v", len(book.catalog), book.catalog)
	}
	if book.xfMap[1] != book.xfMap[2] || book.xfMap[1] == 0 {
		t.Fatalf("duplicate xfs should intern, map=%v", book.xfMap)
	}
	style := book.catalog[book.xfMap[1]]
	if style.Fill != "#1f4e79" || style.Color != "#ffffff" || !style.Bold || style.Size != 14 || style.Align != "center" || !style.Wrap {
		t.Fatalf("style %#v", style)
	}
	if !style.BorderT || !style.BorderL {
		t.Fatalf("borders %#v", style)
	}
}

func TestThemeIndex1IsDarkText(t *testing.T) {
	theme := parseThemeColors(nil)
	if got := themeBySpreadsheetIndex(theme, 1); got != "#000000" {
		t.Fatalf("theme 1 = %q, want dark text", got)
	}
	if got := themeBySpreadsheetIndex(theme, 0); got != "#ffffff" {
		t.Fatalf("theme 0 = %q, want light fill", got)
	}
}

func TestColAndRowSizeConversion(t *testing.T) {
	if got := colWidthToPx(18.5); got != 135 {
		t.Fatalf("col px %v", got)
	}
	if got := rowHeightToPx(15); got != 20 {
		t.Fatalf("row px %v", got)
	}
	if got := rowHeightToPx(30); got != 40 {
		t.Fatalf("custom row px %v", got)
	}
}
