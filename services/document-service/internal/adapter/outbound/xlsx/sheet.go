package xlsx

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/beevik/etree"

	"order-fill/services/document-service/internal/domain/spreadsheet"
)

// sheetDataFollowers are the worksheet elements that the schema places after
// sheetData; they position a sheetData element created for a worksheet that
// carries none.
var sheetDataFollowers = []string{
	"sheetCalcPr", "sheetProtection", "protectedRanges", "scenarios", "autoFilter",
	"sortState", "dataConsolidate", "customSheetViews", "mergeCells", "phoneticPr",
	"conditionalFormatting", "dataValidations", "hyperlinks", "printOptions",
	"pageMargins", "pageSetup", "headerFooter", "rowBreaks", "colBreaks",
	"customProperties", "cellWatches", "ignoredErrors", "smartTags", "drawing",
	"legacyDrawing", "legacyDrawingHF", "drawingHF", "picture", "oleObjects",
	"controls", "webPublishItems", "tableParts", "extLst",
}

type sheet struct {
	name      string
	entry     *part
	document  *etree.Document
	sheetData *etree.Element
	rows      map[int]*etree.Element
	cells     map[cellKey]*cell
	maxRow    int
	maxColumn int
	modified  bool
}

type cell struct {
	element *etree.Element
	value   string
}

func newSheet(name string, entry *part, data []byte, shared []string, report func(float64)) (*sheet, error) {
	document, rows, cells, err := parseSheetDocument(data, shared, report)
	if err != nil {
		return nil, err
	}
	root := document.Root()
	if root == nil {
		return nil, fmt.Errorf("xlsx: worksheet %q has no root element", name)
	}
	loaded := &sheet{
		name:      name,
		entry:     entry,
		document:  document,
		sheetData: ensureChild(root, "sheetData", sheetDataFollowers...),
		rows:      rows,
		cells:     cells,
	}
	for key := range cells {
		if key.row > loaded.maxRow {
			loaded.maxRow = key.row
		}
		if key.column > loaded.maxColumn {
			loaded.maxColumn = key.column
		}
	}
	return loaded, nil
}

func (s *sheet) Name() string {
	return s.name
}

func (s *sheet) Bounds() spreadsheet.Bounds {
	return spreadsheet.Bounds{MaxRow: s.maxRow, MaxColumn: s.maxColumn}
}

func (s *sheet) noteBounds(row int, column int) {
	if row > s.maxRow {
		s.maxRow = row
	}
	if column > s.maxColumn {
		s.maxColumn = column
	}
}

func (s *sheet) Value(row int, column int) string {
	if found, ok := s.cells[cellKey{row: row, column: column}]; ok {
		return found.value
	}
	return ""
}

func (s *sheet) SetNumber(row int, column int, value float64) {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		s.ClearValue(row, column)
		return
	}
	target := s.findOrCreateCell(row, column)
	text := strconv.FormatFloat(value, 'f', -1, 64)
	if !overwriteNumber(target.element, text) {
		clearCell(target.element)
		newChild(target.element, "v", nil).SetText(text)
	}
	target.value = text
	s.modified = true
}

func (s *sheet) ClearValue(row int, column int) {
	target := s.findOrCreateCell(row, column)
	clearCell(target.element)
	target.value = ""
	s.modified = true
}

func (s *sheet) SetText(row int, column int, value string) {
	if value == "" {
		s.ClearValue(row, column)
		return
	}
	target := s.findOrCreateCell(row, column)
	clearCell(target.element)
	target.element.CreateAttr("t", "inlineStr")
	inline := newChild(target.element, "is", nil)
	newChild(inline, "t", nil).SetText(value)
	target.value = value
	s.modified = true
}

// DeleteRows drops the given rows from sheetData and renumbers the rows and
// cell references below them, the way a spreadsheet application would. Row
// numbers referenced by other worksheet parts, such as merged ranges, are left
// alone: the 1C export the merge runs on carries none below the header.
func (s *sheet) DeleteRows(rows []int) {
	removed := make(map[int]bool, len(rows))
	ordered := make([]int, 0, len(rows))
	for _, row := range rows {
		if row <= 0 || removed[row] {
			continue
		}
		removed[row] = true
		ordered = append(ordered, row)
	}
	if len(ordered) == 0 {
		return
	}
	sort.Ints(ordered)
	// A surviving row moves up by the number of deleted rows above it.
	shift := func(row int) int { return row - sort.SearchInts(ordered, row) }

	for _, rowElement := range s.sheetData.SelectElements("row") {
		number, err := strconv.Atoi(rowElement.SelectAttrValue("r", ""))
		if err == nil && removed[number] {
			s.sheetData.RemoveChild(rowElement)
		}
	}

	movedRows := make(map[int]*etree.Element, len(s.rows))
	for _, rowElement := range s.sheetData.SelectElements("row") {
		number, err := strconv.Atoi(rowElement.SelectAttrValue("r", ""))
		if err != nil || number <= 0 {
			continue
		}
		moved := shift(number)
		rowElement.CreateAttr("r", strconv.Itoa(moved))
		if _, exists := movedRows[moved]; !exists {
			movedRows[moved] = rowElement
		}
	}
	s.rows = movedRows

	movedCells := make(map[cellKey]*cell, len(s.cells))
	for key, current := range s.cells {
		if removed[key.row] {
			continue
		}
		moved := cellKey{row: shift(key.row), column: key.column}
		current.element.CreateAttr("r", formatCellRef(moved.row, moved.column))
		movedCells[moved] = current
	}
	s.cells = movedCells
	s.modified = true
	s.maxRow = 0
	s.maxColumn = 0
	for key := range s.cells {
		if key.row > s.maxRow {
			s.maxRow = key.row
		}
		if key.column > s.maxColumn {
			s.maxColumn = key.column
		}
	}
}

func (s *sheet) serialize() error {
	if !s.modified {
		return nil
	}
	data, err := s.document.WriteToBytes()
	if err != nil {
		return err
	}
	s.entry.replace(data)
	return nil
}

func (s *sheet) findOrCreateCell(row int, column int) *cell {
	key := cellKey{row: row, column: column}
	if existing, ok := s.cells[key]; ok {
		return existing
	}
	rowElement := s.findOrCreateRow(row)
	var before *etree.Element
	for _, candidate := range rowElement.SelectElements("c") {
		reference, ok := parseCellRef(candidate.SelectAttrValue("r", ""))
		if ok && reference.column > column {
			before = candidate
			break
		}
	}
	element := newChild(rowElement, "c", before)
	element.CreateAttr("r", formatCellRef(row, column))
	created := &cell{element: element}
	s.cells[key] = created
	s.noteBounds(row, column)
	return created
}

func (s *sheet) findOrCreateRow(number int) *etree.Element {
	if existing, ok := s.rows[number]; ok {
		return existing
	}
	var before *etree.Element
	for _, candidate := range s.sheetData.SelectElements("row") {
		current, err := strconv.Atoi(candidate.SelectAttrValue("r", ""))
		if err == nil && current > number {
			before = candidate
			break
		}
	}
	element := newChild(s.sheetData, "row", before)
	element.CreateAttr("r", strconv.Itoa(number))
	s.rows[number] = element
	return element
}

// clearCell drops the value and the type of a cell but keeps the cell element
// and its remaining attributes, notably the style index, because the user
// downloads the uploaded document back and it must look unchanged.
func clearCell(element *etree.Element) {
	for len(element.Child) > 0 {
		element.RemoveChildAt(0)
	}
	element.RemoveAttr("t")
}

func readCellValue(element *etree.Element, shared []string) string {
	kind := element.SelectAttrValue("t", "")
	if kind == "inlineStr" {
		if inline := element.SelectElement("is"); inline != nil {
			if text := inline.SelectElement("t"); text != nil {
				return elementText(text)
			}
		}
		var builder strings.Builder
		for _, node := range descendants(element, "t") {
			builder.WriteString(elementText(node))
		}
		return builder.String()
	}
	value := element.SelectElement("v")
	if value == nil {
		return ""
	}
	raw := elementText(value)
	switch kind {
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil || index < 0 || index >= len(shared) {
			return ""
		}
		return shared[index]
	case "b":
		if raw == "1" {
			return "1"
		}
		return "0"
	default:
		return raw
	}
}

func overwriteNumber(element *etree.Element, text string) bool {
	if element.SelectAttrValue("t", "") != "" {
		return false
	}
	children := element.ChildElements()
	if len(children) != 1 || (children[0].Tag != "v" && !hasLocalName(children[0].Tag, "v")) {
		return false
	}
	children[0].SetText(text)
	return true
}
