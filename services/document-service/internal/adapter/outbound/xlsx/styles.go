package xlsx

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/beevik/etree"

	"order-fill/services/document-service/internal/domain/spreadsheet"
)

const (
	defaultColWidthChars = 8.43
	defaultRowHeightPt   = 15.0
)

// styleBook maps each Excel xf onto a compact appearance catalog.
type styleBook struct {
	catalog []spreadsheet.Style
	xfMap   []int
}

func (b *styleBook) catalogIndex(xf int) int {
	if b == nil || xf < 0 || xf >= len(b.xfMap) {
		return 0
	}
	return b.xfMap[xf]
}

func parseStyleBook(stylesXML []byte, themeXML []byte) *styleBook {
	theme := parseThemeColors(themeXML)
	if len(stylesXML) == 0 {
		return &styleBook{catalog: []spreadsheet.Style{{}}, xfMap: []int{0}}
	}
	document, err := parseDocument(stylesXML)
	if err != nil {
		return &styleBook{catalog: []spreadsheet.Style{{}}, xfMap: []int{0}}
	}
	root := document.Root()
	fonts := parseFonts(child(root, "fonts"), theme)
	fills := parseFills(child(root, "fills"), theme)
	borders := parseBorders(child(root, "borders"), theme)
	xfs := children(child(root, "cellXfs"), "xf")
	book := &styleBook{catalog: []spreadsheet.Style{{}}, xfMap: make([]int, len(xfs))}
	index := map[spreadsheet.Style]int{{}: 0}
	for i, xf := range xfs {
		style := resolveXf(xf, fonts, fills, borders)
		if id, ok := index[style]; ok {
			book.xfMap[i] = id
			continue
		}
		id := len(book.catalog)
		index[style] = id
		book.catalog = append(book.catalog, style)
		book.xfMap[i] = id
	}
	if len(book.xfMap) == 0 {
		book.xfMap = []int{0}
	}
	return book
}

func resolveXf(xf *etree.Element, fonts []spreadsheet.Style, fills []spreadsheet.Style, borders []spreadsheet.Style) spreadsheet.Style {
	style := spreadsheet.Style{}
	if id := attrInt(xf, "fontId"); id >= 0 && id < len(fonts) {
		font := fonts[id]
		style.Bold = font.Bold
		style.Italic = font.Italic
		style.Size = font.Size
		style.Color = font.Color
	}
	if id := attrInt(xf, "fillId"); id >= 0 && id < len(fills) {
		style.Fill = fills[id].Fill
	}
	if id := attrInt(xf, "borderId"); id >= 0 && id < len(borders) {
		border := borders[id]
		style.BorderT = border.BorderT
		style.BorderR = border.BorderR
		style.BorderB = border.BorderB
		style.BorderL = border.BorderL
	}
	if alignment := child(xf, "alignment"); alignment != nil {
		switch strings.ToLower(alignment.SelectAttrValue("horizontal", "")) {
		case "center", "centerContinuous", "distributed":
			style.Align = "center"
		case "right":
			style.Align = "right"
		case "left":
			style.Align = "left"
		}
		switch strings.ToLower(alignment.SelectAttrValue("vertical", "")) {
		case "top":
			style.Valign = "top"
		case "bottom":
			style.Valign = "bottom"
		case "center":
			style.Valign = "middle"
		}
		if alignment.SelectAttrValue("wrapText", "") == "1" {
			style.Wrap = true
		}
	}
	if style.Color == "#000000" {
		style.Color = ""
	}
	if style.Fill == "#ffffff" {
		style.Fill = ""
	}
	return style
}

func parseFonts(node *etree.Element, theme []string) []spreadsheet.Style {
	items := children(node, "font")
	out := make([]spreadsheet.Style, 0, len(items))
	for _, item := range items {
		style := spreadsheet.Style{}
		if child(item, "b") != nil {
			style.Bold = true
		}
		if child(item, "i") != nil {
			style.Italic = true
		}
		if sz := child(item, "sz"); sz != nil {
			if size, err := strconv.ParseFloat(sz.SelectAttrValue("val", ""), 64); err == nil && size > 0 && size != 11 {
				style.Size = int(math.Round(size))
			}
		}
		style.Color = resolveColor(child(item, "color"), theme, false)
		out = append(out, style)
	}
	return out
}

func parseFills(node *etree.Element, theme []string) []spreadsheet.Style {
	items := children(node, "fill")
	out := make([]spreadsheet.Style, 0, len(items))
	for _, item := range items {
		pattern := child(item, "patternFill")
		style := spreadsheet.Style{}
		if pattern != nil {
			kind := strings.ToLower(pattern.SelectAttrValue("patternType", ""))
			if kind == "solid" {
				style.Fill = resolveColor(child(pattern, "fgColor"), theme, true)
			}
		}
		out = append(out, style)
	}
	return out
}

func parseBorders(node *etree.Element, theme []string) []spreadsheet.Style {
	items := children(node, "border")
	out := make([]spreadsheet.Style, 0, len(items))
	for _, item := range items {
		style := spreadsheet.Style{
			BorderT: hasBorder(child(item, "top")),
			BorderR: hasBorder(child(item, "right")),
			BorderB: hasBorder(child(item, "bottom")),
			BorderL: hasBorder(child(item, "left")),
		}
		out = append(out, style)
	}
	return out
}

func hasBorder(node *etree.Element) bool {
	if node == nil {
		return false
	}
	style := strings.ToLower(strings.TrimSpace(node.SelectAttrValue("style", "")))
	return style != "" && style != "none"
}

func resolveColor(node *etree.Element, theme []string, fill bool) string {
	if node == nil {
		return ""
	}
	if rgb := strings.TrimSpace(node.SelectAttrValue("rgb", "")); rgb != "" {
		return normalizeHex(rgb)
	}
	if indexed := strings.TrimSpace(node.SelectAttrValue("indexed", "")); indexed != "" {
		if index, err := strconv.Atoi(indexed); err == nil {
			if hex, ok := indexedColor(index, fill); ok {
				return hex
			}
		}
	}
	if themeIndex := strings.TrimSpace(node.SelectAttrValue("theme", "")); themeIndex != "" {
		if index, err := strconv.Atoi(themeIndex); err == nil {
			if hex := themeBySpreadsheetIndex(theme, index); hex != "" {
				if tint, err := strconv.ParseFloat(node.SelectAttrValue("tint", "0"), 64); err == nil && tint != 0 {
					hex = applyTint(hex, tint)
				}
				return hex
			}
		}
	}
	return ""
}

// SpreadsheetML swaps lt/dk pairs relative to the theme file order.
var spreadsheetThemeOrder = []int{1, 0, 3, 2, 4, 5, 6, 7, 8, 9, 10, 11}

func themeBySpreadsheetIndex(theme []string, index int) string {
	if index < 0 || index >= len(spreadsheetThemeOrder) {
		return ""
	}
	mapped := spreadsheetThemeOrder[index]
	if mapped < 0 || mapped >= len(theme) {
		return ""
	}
	return theme[mapped]
}

func parseThemeColors(data []byte) []string {
	fallback := []string{"#000000", "#ffffff", "#1f497d", "#eeece1", "#4f81bd", "#c0504d", "#9bbb59", "#8064a2", "#4bacc6", "#f79646", "#0000ff", "#800080"}
	if len(data) == 0 {
		return fallback
	}
	document, err := parseDocument(data)
	if err != nil {
		return fallback
	}
	scheme := findNamed(document.Root(), "clrScheme")
	if scheme == nil {
		return fallback
	}
	names := []string{"dk1", "lt1", "dk2", "lt2", "accent1", "accent2", "accent3", "accent4", "accent5", "accent6", "hlink", "folHlink"}
	out := make([]string, len(names))
	copy(out, fallback)
	for index, name := range names {
		node := child(scheme, name)
		if node == nil {
			continue
		}
		if hex := themeColorValue(node); hex != "" {
			out[index] = hex
		}
	}
	return out
}

func themeColorValue(node *etree.Element) string {
	if srgb := findNamed(node, "srgbClr"); srgb != nil {
		return normalizeHex(srgb.SelectAttrValue("val", ""))
	}
	if sys := findNamed(node, "sysClr"); sys != nil {
		if hex := normalizeHex(sys.SelectAttrValue("lastClr", "")); hex != "" {
			return hex
		}
	}
	return ""
}

func normalizeHex(value string) string {
	hex := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "#")
	if len(hex) == 8 {
		hex = hex[2:]
	}
	if len(hex) != 6 {
		return ""
	}
	for _, char := range hex {
		if char < '0' || char > '9' && (char < 'a' || char > 'f') {
			return ""
		}
	}
	return "#" + hex
}

func applyTint(hex string, tint float64) string {
	rgb, ok := parseHex(hex)
	if !ok {
		return hex
	}
	r := tintChannel(rgb[0], tint)
	g := tintChannel(rgb[1], tint)
	b := tintChannel(rgb[2], tint)
	return fmt.Sprintf("#%02x%02x%02x", int(math.Round(r*255)), int(math.Round(g*255)), int(math.Round(b*255)))
}

func tintChannel(value float64, tint float64) float64 {
	if tint < 0 {
		value *= 1 + tint
	} else {
		value = value*(1-tint) + tint
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func parseHex(hex string) ([3]float64, bool) {
	value := strings.TrimPrefix(hex, "#")
	if len(value) != 6 {
		return [3]float64{}, false
	}
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return [3]float64{}, false
	}
	return [3]float64{float64(n>>16) / 255, float64((n>>8)&0xff) / 255, float64(n&0xff) / 255}, true
}

func indexedColor(index int, fill bool) (string, bool) {
	if fill && (index == 64 || index == 65) {
		return "", false
	}
	if index < 0 || index >= len(indexedPalette) {
		return "", false
	}
	return indexedPalette[index], true
}

type sheetAppearance struct {
	columns    []float64
	rowHeight  float64
	rowHeights map[int]float64
	merges     []spreadsheet.Merge
}

func parseSheetAppearance(root *etree.Element, rows map[int]*etree.Element, maxColumn int) sheetAppearance {
	look := sheetAppearance{rowHeight: rowHeightToPx(defaultRowHeightPt)}
	format := child(root, "sheetFormatPr")
	defaultWidth := defaultColWidthChars
	if format != nil {
		if raw := strings.TrimSpace(format.SelectAttrValue("defaultRowHeight", "")); raw != "" {
			if height, err := strconv.ParseFloat(raw, 64); err == nil && height > 0 {
				look.rowHeight = rowHeightToPx(height)
			}
		}
		if raw := strings.TrimSpace(format.SelectAttrValue("defaultColWidth", "")); raw != "" {
			if width, err := strconv.ParseFloat(raw, 64); err == nil && width > 0 {
				defaultWidth = width
			}
		}
	}
	if maxColumn > 0 {
		look.columns = make([]float64, maxColumn)
		defaultPx := colWidthToPx(defaultWidth)
		for index := range look.columns {
			look.columns[index] = defaultPx
		}
		for _, col := range children(child(root, "cols"), "col") {
			min := attrInt(col, "min")
			max := attrInt(col, "max")
			if min < 1 {
				continue
			}
			if max < min {
				max = min
			}
			if attrInt(col, "hidden") != 0 {
				for column := min; column <= max && column <= maxColumn; column++ {
					look.columns[column-1] = 0
				}
				continue
			}
			width, err := strconv.ParseFloat(strings.TrimSpace(col.SelectAttrValue("width", "")), 64)
			if err != nil || width <= 0 {
				continue
			}
			px := colWidthToPx(width)
			for column := min; column <= max && column <= maxColumn; column++ {
				look.columns[column-1] = px
			}
		}
	}
	look.rowHeights = customRowHeights(rows, look.rowHeight)
	for _, node := range children(child(root, "mergeCells"), "mergeCell") {
		merge, ok := parseMergeRef(node.SelectAttrValue("ref", ""))
		if !ok || (merge.Width < 2 && merge.Height < 2) {
			continue
		}
		look.merges = append(look.merges, merge)
	}
	return look
}

func customRowHeights(rows map[int]*etree.Element, defaultPx float64) map[int]float64 {
	var heights map[int]float64
	for number, row := range rows {
		raw := strings.TrimSpace(row.SelectAttrValue("ht", ""))
		if raw == "" {
			continue
		}
		height, err := strconv.ParseFloat(raw, 64)
		if err != nil || height <= 0 {
			continue
		}
		px := rowHeightToPx(height)
		if px == defaultPx {
			continue
		}
		if heights == nil {
			heights = map[int]float64{}
		}
		heights[number] = px
	}
	return heights
}

func parseMergeRef(reference string) (spreadsheet.Merge, bool) {
	parts := strings.Split(strings.TrimSpace(reference), ":")
	if len(parts) == 0 || parts[0] == "" {
		return spreadsheet.Merge{}, false
	}
	start, ok := parseCellRef(parts[0])
	if !ok {
		return spreadsheet.Merge{}, false
	}
	end := start
	if len(parts) > 1 {
		if parsed, parsedOK := parseCellRef(parts[1]); parsedOK {
			end = parsed
		}
	}
	row, lastRow := start.row, end.row
	column, lastColumn := start.column, end.column
	if lastRow < row {
		row, lastRow = lastRow, row
	}
	if lastColumn < column {
		column, lastColumn = lastColumn, column
	}
	return spreadsheet.Merge{
		Row:    row,
		Column: column,
		Height: lastRow - row + 1,
		Width:  lastColumn - column + 1,
	}, true
}

func colWidthToPx(width float64) float64 {
	if width <= 0 {
		width = defaultColWidthChars
	}
	return math.Round(width*7 + 5)
}

func rowHeightToPx(points float64) float64 {
	if points <= 0 {
		points = defaultRowHeightPt
	}
	return math.Round(points * 96 / 72)
}

func attrInt(node *etree.Element, name string) int {
	if node == nil {
		return 0
	}
	raw := strings.TrimSpace(node.SelectAttrValue(name, "0"))
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return value
}

func child(node *etree.Element, name string) *etree.Element {
	if node == nil {
		return nil
	}
	for _, item := range node.ChildElements() {
		if item.Tag == name || hasLocalName(item.Tag, name) {
			return item
		}
	}
	return nil
}

func children(node *etree.Element, name string) []*etree.Element {
	if node == nil {
		return nil
	}
	out := make([]*etree.Element, 0)
	for _, item := range node.ChildElements() {
		if item.Tag == name || hasLocalName(item.Tag, name) {
			out = append(out, item)
		}
	}
	return out
}

func findNamed(node *etree.Element, name string) *etree.Element {
	if node == nil {
		return nil
	}
	if node.Tag == name || hasLocalName(node.Tag, name) {
		return node
	}
	for _, item := range node.ChildElements() {
		if found := findNamed(item, name); found != nil {
			return found
		}
	}
	return nil
}

var indexedPalette = []string{
	"#000000", "#ffffff", "#ff0000", "#00ff00", "#0000ff", "#ffff00", "#ff00ff", "#00ffff",
	"#000000", "#ffffff", "#ff0000", "#00ff00", "#0000ff", "#ffff00", "#ff00ff", "#00ffff",
	"#800000", "#008000", "#000080", "#808000", "#800080", "#008080", "#c0c0c0", "#808080",
	"#9999ff", "#993366", "#ffffcc", "#ccffff", "#660066", "#ff8080", "#0066cc", "#ccccff",
	"#000080", "#ff00ff", "#ffff00", "#00ffff", "#800080", "#800000", "#008080", "#0000ff",
	"#00ccff", "#ccffff", "#ccffcc", "#ffff99", "#99ccff", "#ff99cc", "#cc99ff", "#ffcc99",
	"#3366ff", "#33cccc", "#99cc00", "#ffcc00", "#ff9900", "#ff6600", "#666699", "#969696",
	"#003366", "#339966", "#003300", "#333300", "#993300", "#993366", "#333399", "#333333",
}
