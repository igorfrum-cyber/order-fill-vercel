package excel

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
)

type Workbook struct {
	sheets []*Sheet
	byName map[string]*Sheet
}

type Sheet struct {
	name  string
	cells map[string]Cell
}

type Cell struct {
	ref   string
	value string
	kind  cellKind
}

type cellKind string

const (
	cellKindNumber cellKind = "number"
	cellKindText   cellKind = "text"
)

func NewWorkbook() *Workbook {
	return &Workbook{byName: make(map[string]*Sheet)}
}

func (w *Workbook) AddSheet(name string) *Sheet {
	sheet := &Sheet{name: name, cells: make(map[string]Cell)}
	w.sheets = append(w.sheets, sheet)
	w.byName[name] = sheet
	return sheet
}

func (w *Workbook) Sheet(name string) (*Sheet, bool) {
	sheet, ok := w.byName[name]
	return sheet, ok
}

func (s *Sheet) Value(ref string) (string, bool) {
	cell, ok := s.cells[strings.ToUpper(ref)]
	return cell.value, ok
}

func (s *Sheet) SetText(ref string, value string) {
	normalizedRef := strings.ToUpper(ref)
	s.cells[normalizedRef] = Cell{ref: normalizedRef, value: value, kind: cellKindText}
}

func (s *Sheet) SetNumber(ref string, value float64) {
	normalizedRef := strings.ToUpper(ref)
	s.cells[normalizedRef] = Cell{
		ref:   normalizedRef,
		value: strconv.FormatFloat(value, 'f', -1, 64),
		kind:  cellKindNumber,
	}
}

func Load(reader io.ReaderAt, size int64) (*Workbook, error) {
	archive, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, err
	}
	files := map[string]*zip.File{}
	for _, file := range archive.File {
		files[file.Name] = file
	}

	workbookXML, err := readZipFile(files["xl/workbook.xml"])
	if err != nil {
		return nil, err
	}
	relsXML, err := readZipFile(files["xl/_rels/workbook.xml.rels"])
	if err != nil {
		return nil, err
	}

	var workbookDoc workbookDocument
	if err := xml.Unmarshal(workbookXML, &workbookDoc); err != nil {
		return nil, err
	}
	var rels relationshipsDocument
	if err := xml.Unmarshal(relsXML, &rels); err != nil {
		return nil, err
	}

	targetByID := map[string]string{}
	for _, rel := range rels.Relationships {
		targetByID[rel.ID] = rel.Target
	}

	workbook := NewWorkbook()
	for _, sourceSheet := range workbookDoc.Sheets {
		sheet := workbook.AddSheet(sourceSheet.Name)
		target := targetByID[sourceSheet.RelationshipID]
		if target == "" {
			return nil, fmt.Errorf("missing relationship for sheet %s", sourceSheet.Name)
		}
		sheetPath := path.Clean("xl/" + target)
		content, err := readZipFile(files[sheetPath])
		if err != nil {
			return nil, err
		}
		if err := loadSheet(content, sheet); err != nil {
			return nil, err
		}
	}
	return workbook, nil
}

func (w *Workbook) Save(writer io.Writer) error {
	archive := zip.NewWriter(writer)
	if err := writeZipFile(archive, "[Content_Types].xml", []byte(contentTypesXML)); err != nil {
		return err
	}
	if err := writeZipFile(archive, "_rels/.rels", []byte(rootRelsXML)); err != nil {
		return err
	}
	if err := writeZipFile(archive, "xl/workbook.xml", []byte(w.workbookXML())); err != nil {
		return err
	}
	if err := writeZipFile(archive, "xl/_rels/workbook.xml.rels", []byte(w.workbookRelsXML())); err != nil {
		return err
	}
	for index, sheet := range w.sheets {
		if err := writeZipFile(archive, fmt.Sprintf("xl/worksheets/sheet%d.xml", index+1), []byte(sheet.xml())); err != nil {
			return err
		}
	}
	return archive.Close()
}

func loadSheet(content []byte, sheet *Sheet) error {
	var document worksheetDocument
	if err := xml.Unmarshal(content, &document); err != nil {
		return err
	}
	for _, row := range document.SheetData.Rows {
		for _, sourceCell := range row.Cells {
			ref := strings.ToUpper(sourceCell.Ref)
			if sourceCell.Type == "inlineStr" {
				sheet.cells[ref] = Cell{ref: ref, value: sourceCell.InlineString.Text, kind: cellKindText}
				continue
			}
			sheet.cells[ref] = Cell{ref: ref, value: sourceCell.Value, kind: cellKindNumber}
		}
	}
	return nil
}

func (w *Workbook) workbookXML() string {
	var builder strings.Builder
	builder.WriteString(xml.Header)
	builder.WriteString(`<workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets>`)
	for index, sheet := range w.sheets {
		builder.WriteString(fmt.Sprintf(`<sheet name="%s" sheetId="%d" r:id="rId%d"/>`, xmlEscape(sheet.name), index+1, index+1))
	}
	builder.WriteString(`</sheets></workbook>`)
	return builder.String()
}

func (w *Workbook) workbookRelsXML() string {
	var builder strings.Builder
	builder.WriteString(xml.Header)
	builder.WriteString(`<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	for index := range w.sheets {
		builder.WriteString(fmt.Sprintf(`<Relationship Id="rId%d" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet%d.xml"/>`, index+1, index+1))
	}
	builder.WriteString(`</Relationships>`)
	return builder.String()
}

func (s *Sheet) xml() string {
	refs := make([]string, 0, len(s.cells))
	for ref := range s.cells {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		left := parseRef(refs[i])
		right := parseRef(refs[j])
		if left.row == right.row {
			return left.column < right.column
		}
		return left.row < right.row
	})

	rows := map[int][]Cell{}
	rowNumbers := make([]int, 0)
	seenRows := map[int]bool{}
	for _, ref := range refs {
		parsed := parseRef(ref)
		rows[parsed.row] = append(rows[parsed.row], s.cells[ref])
		if !seenRows[parsed.row] {
			seenRows[parsed.row] = true
			rowNumbers = append(rowNumbers, parsed.row)
		}
	}
	sort.Ints(rowNumbers)

	var builder strings.Builder
	builder.WriteString(xml.Header)
	builder.WriteString(`<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for _, rowNumber := range rowNumbers {
		builder.WriteString(fmt.Sprintf(`<row r="%d">`, rowNumber))
		for _, cell := range rows[rowNumber] {
			if cell.kind == cellKindText {
				builder.WriteString(fmt.Sprintf(`<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, cell.ref, xmlEscape(cell.value)))
				continue
			}
			builder.WriteString(fmt.Sprintf(`<c r="%s"><v>%s</v></c>`, cell.ref, xmlEscape(cell.value)))
		}
		builder.WriteString(`</row>`)
	}
	builder.WriteString(`</sheetData></worksheet>`)
	return builder.String()
}

type parsedRef struct {
	column int
	row    int
}

func parseRef(ref string) parsedRef {
	var column int
	var index int
	for index < len(ref) {
		char := ref[index]
		if char < 'A' || char > 'Z' {
			break
		}
		column = column*26 + int(char-'A'+1)
		index++
	}
	row, _ := strconv.Atoi(ref[index:])
	return parsedRef{column: column, row: row}
}

func readZipFile(file *zip.File) ([]byte, error) {
	if file == nil {
		return nil, fmt.Errorf("xlsx part was not found")
	}
	reader, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

func writeZipFile(archive *zip.Writer, name string, content []byte) error {
	writer, err := archive.Create(name)
	if err != nil {
		return err
	}
	_, err = writer.Write(content)
	return err
}

func xmlEscape(value string) string {
	var builder strings.Builder
	if err := xml.EscapeText(&builder, []byte(value)); err != nil {
		return value
	}
	return builder.String()
}

type workbookDocument struct {
	Sheets []workbookSheet `xml:"sheets>sheet"`
}

type workbookSheet struct {
	Name           string `xml:"name,attr"`
	RelationshipID string `xml:"http://schemas.openxmlformats.org/officeDocument/2006/relationships id,attr"`
}

type relationshipsDocument struct {
	Relationships []relationship `xml:"Relationship"`
}

type relationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type worksheetDocument struct {
	SheetData sheetDataDocument `xml:"sheetData"`
}

type sheetDataDocument struct {
	Rows []rowDocument `xml:"row"`
}

type rowDocument struct {
	Cells []cellDocument `xml:"c"`
}

type cellDocument struct {
	Ref          string               `xml:"r,attr"`
	Type         string               `xml:"t,attr"`
	Value        string               `xml:"v"`
	InlineString inlineStringDocument `xml:"is"`
}

type inlineStringDocument struct {
	Text string `xml:"t"`
}

const contentTypesXML = `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/>
</Types>`

const rootRelsXML = `<?xml version="1.0" encoding="UTF-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/>
</Relationships>`
