// Package xlsx adapts xlsx and xlsm archives to the spreadsheet port. It edits
// the uploaded document in place: only the parts the domain changes are
// re-serialized and every other zip entry is copied as it was read, so styles,
// number formats, merged cells, images, defined names, charts, data validation
// and macros survive a load and save round trip.
package xlsx

import (
	"fmt"
	"strings"

	"order-fill/services/document-service/internal/domain/spreadsheet"
)

const (
	workbookPath      = "xl/workbook.xml"
	workbookRelsPath  = "xl/_rels/workbook.xml.rels"
	sharedStringsPath = "xl/sharedStrings.xml"
	contentTypesPath  = "[Content_Types].xml"
	calcChainPath     = "xl/calcChain.xml"
)

type codec struct{}

func NewCodec() spreadsheet.Codec {
	return codec{}
}

func (codec) Load(content []byte) (spreadsheet.Workbook, error) {
	parts, err := readArchive(content)
	if err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	workbookPart, ok := parts.get(workbookPath)
	if !ok {
		return nil, fmt.Errorf("xlsx: %s is missing", workbookPath)
	}
	workbookData, err := workbookPart.bytes()
	if err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	workbookDocument, err := parseDocument(workbookData)
	if err != nil {
		return nil, fmt.Errorf("xlsx: parse %s: %w", workbookPath, err)
	}
	targets, err := readRelationshipTargets(parts)
	if err != nil {
		return nil, err
	}
	shared, err := readSharedStrings(parts)
	if err != nil {
		return nil, err
	}

	book := &workbook{parts: parts, document: workbookDocument, byName: map[string]*sheet{}}
	sheets := workbookDocument.Root().SelectElement("sheets")
	if sheets == nil {
		return nil, fmt.Errorf("xlsx: %s has no sheets element", workbookPath)
	}
	for _, node := range sheets.SelectElements("sheet") {
		name := node.SelectAttrValue("name", "")
		target, ok := targets[relationshipID(node)]
		if !ok || target == "" {
			return nil, fmt.Errorf("xlsx: sheet %q has no worksheet relationship", name)
		}
		sheetPath := normalizeWorkbookTarget(target)
		sheetPart, ok := parts.get(sheetPath)
		if !ok {
			return nil, fmt.Errorf("xlsx: worksheet part %s is missing", sheetPath)
		}
		sheetData, err := sheetPart.bytes()
		if err != nil {
			return nil, fmt.Errorf("xlsx: %w", err)
		}
		sheetDocument, err := parseDocument(sheetData)
		if err != nil {
			return nil, fmt.Errorf("xlsx: parse %s: %w", sheetPath, err)
		}
		loaded := newSheet(name, sheetPart, sheetDocument, shared)
		book.sheets = append(book.sheets, loaded)
		if _, exists := book.byName[name]; !exists {
			book.byName[name] = loaded
		}
	}
	return book, nil
}

func readRelationshipTargets(parts *archive) (map[string]string, error) {
	entry, ok := parts.get(workbookRelsPath)
	if !ok {
		return nil, fmt.Errorf("xlsx: %s is missing", workbookRelsPath)
	}
	data, err := entry.bytes()
	if err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	document, err := parseDocument(data)
	if err != nil {
		return nil, fmt.Errorf("xlsx: parse %s: %w", workbookRelsPath, err)
	}
	targets := map[string]string{}
	for _, node := range document.Root().SelectElements("Relationship") {
		targets[node.SelectAttrValue("Id", "")] = node.SelectAttrValue("Target", "")
	}
	return targets, nil
}

func readSharedStrings(parts *archive) ([]string, error) {
	entry, ok := parts.get(sharedStringsPath)
	if !ok {
		return nil, nil
	}
	data, err := entry.bytes()
	if err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	document, err := parseDocument(data)
	if err != nil {
		return nil, fmt.Errorf("xlsx: parse %s: %w", sharedStringsPath, err)
	}
	items := document.Root().SelectElements("si")
	values := make([]string, 0, len(items))
	for _, item := range items {
		var builder strings.Builder
		for _, node := range descendants(item, "t") {
			builder.WriteString(elementText(node))
		}
		values = append(values, builder.String())
	}
	return values, nil
}

func normalizeWorkbookTarget(target string) string {
	clean := strings.TrimLeft(target, "/")
	if strings.HasPrefix(clean, "xl/") {
		return clean
	}
	return "xl/" + clean
}
