package xlsx

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"

	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

func (c codec) NewTable(sheetName string, headers []string, rows [][]any) (spreadsheet.Workbook, error) {
	var sheet bytes.Buffer
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetViews><sheetView workbookViewId="0"><pane ySplit="1" topLeftCell="A2" state="frozen"/></sheetView></sheetViews><sheetData>`)
	writeTableRow(&sheet, 1, stringValues(headers))
	for index, row := range rows {
		writeTableRow(&sheet, index+2, row)
	}
	sheet.WriteString(`</sheetData><autoFilter ref="A1:` + spreadsheet.ColumnName(len(headers)) + strconv.Itoa(len(rows)+1) + `"/><pageMargins left="0.7" right="0.7" top="0.75" bottom="0.75" header="0.3" footer="0.3"/></worksheet>`)

	var content bytes.Buffer
	archive := zip.NewWriter(&content)
	parts := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/><Override PartName="/xl/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.styles+xml"/></Types>`,
		"_rels/.rels":                `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="` + escapeXML(sheetName) + `" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/></Relationships>`,
		"xl/styles.xml":              `<?xml version="1.0" encoding="UTF-8"?><styleSheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><fonts count="1"><font><sz val="11"/><name val="Calibri"/></font></fonts><fills count="1"><fill><patternFill patternType="none"/></fill></fills><borders count="1"><border/></borders><cellStyleXfs count="1"><xf/></cellStyleXfs><cellXfs count="1"><xf xfId="0"/></cellXfs></styleSheet>`,
		"xl/worksheets/sheet1.xml":   sheet.String(),
	}
	for name, body := range parts {
		writer, err := archive.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := writer.Write([]byte(body)); err != nil {
			return nil, err
		}
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return c.Load(content.Bytes())
}

func writeTableRow(out *bytes.Buffer, rowNumber int, values []any) {
	fmt.Fprintf(out, `<row r="%d">`, rowNumber)
	for index, value := range values {
		ref := spreadsheet.ColumnName(index+1) + strconv.Itoa(rowNumber)
		switch typed := value.(type) {
		case int:
			fmt.Fprintf(out, `<c r="%s"><v>%d</v></c>`, ref, typed)
		case float64:
			fmt.Fprintf(out, `<c r="%s"><v>%s</v></c>`, ref, strconv.FormatFloat(typed, 'f', -1, 64))
		default:
			fmt.Fprintf(out, `<c r="%s" t="inlineStr"><is><t>%s</t></is></c>`, ref, escapeXML(fmt.Sprint(value)))
		}
	}
	out.WriteString(`</row>`)
}

func stringValues(values []string) []any {
	out := make([]any, len(values))
	for index, value := range values {
		out[index] = value
	}
	return out
}

func escapeXML(value string) string {
	var out bytes.Buffer
	_ = xml.EscapeText(&out, []byte(value))
	return out.String()
}
