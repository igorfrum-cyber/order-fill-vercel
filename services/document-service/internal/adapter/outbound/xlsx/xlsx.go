package xlsx

import (
	"fmt"
	"strings"
	"sync"

	"order-fill/services/document-service/internal/domain/spreadsheet"
)

const (
	workbookPath      = "xl/workbook.xml"
	workbookRelsPath  = "xl/_rels/workbook.xml.rels"
	sharedStringsPath = "xl/sharedStrings.xml"
	stylesPath        = "xl/styles.xml"
	contentTypesPath  = "[Content_Types].xml"
	calcChainPath     = "xl/calcChain.xml"
)

type codec struct{}

func NewCodec() spreadsheet.Codec {
	return codec{}
}

func (c codec) Load(content []byte) (spreadsheet.Workbook, error) {
	return c.LoadWithProgress(content, nil)
}

func (c codec) LoadWithProgress(content []byte, report spreadsheet.LoadProgress) (spreadsheet.Workbook, error) {
	safe := monotonicReporter(report)
	safe(0)

	parts, err := readArchive(content)
	if err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	safe(0.06)

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
	styles := parseStyleBook(partBytes(parts, stylesPath), readTheme(parts))
	safe(0.12)

	book := &workbook{parts: parts, document: workbookDocument, byName: map[string]*sheet{}}
	sheets := workbookDocument.Root().SelectElement("sheets")
	if sheets == nil {
		return nil, fmt.Errorf("xlsx: %s has no sheets element", workbookPath)
	}

	type sheetSpec struct {
		name string
		part *part
		path string
	}
	specs := make([]sheetSpec, 0)
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
		specs = append(specs, sheetSpec{name: name, part: sheetPart, path: sheetPath})
	}

	weights := make([]float64, len(specs))
	var weightSum float64
	for index, spec := range specs {
		weight := float64(spec.part.header.UncompressedSize64)
		if weight <= 0 {
			weight = 1
		}
		weights[index] = weight
		weightSum += weight
	}

	loaded := make([]*sheet, len(specs))
	frac := make([]float64, len(specs))
	var fracMu sync.Mutex
	var firstErr error
	var errOnce sync.Once

	publish := func() {
		fracMu.Lock()
		var sum float64
		for index := range frac {
			sum += frac[index] * weights[index]
		}
		fracMu.Unlock()
		if weightSum > 0 {
			safe(0.12 + 0.88*(sum/weightSum))
		}
	}
	setSheet := func(index int, value float64) {
		if value < 0 {
			value = 0
		}
		if value > 1 {
			value = 1
		}
		fracMu.Lock()
		if value > frac[index] {
			frac[index] = value
		}
		fracMu.Unlock()
		publish()
	}

	runWorkers(len(specs), func(index int) {
		spec := specs[index]
		data, err := spec.part.bytesWithProgress(func(fraction float64) {
			setSheet(index, 0.2*fraction)
		})
		if err != nil {
			errOnce.Do(func() { firstErr = fmt.Errorf("xlsx: %w", err) })
			return
		}
		setSheet(index, 0.2)
		item, err := newSheet(spec.name, spec.part, data, shared, styles, func(fraction float64) {
			setSheet(index, 0.2+0.8*fraction)
		})
		if err != nil {
			errOnce.Do(func() { firstErr = fmt.Errorf("xlsx: parse %s: %w", spec.path, err) })
			return
		}
		loaded[index] = item
		setSheet(index, 1)
	})
	if firstErr != nil {
		return nil, firstErr
	}

	for index, spec := range specs {
		item := loaded[index]
		book.sheets = append(book.sheets, item)
		if _, exists := book.byName[spec.name]; !exists {
			book.byName[spec.name] = item
		}
	}
	safe(1)
	return book, nil
}

func monotonicReporter(report spreadsheet.LoadProgress) func(float64) {
	var mu sync.Mutex
	last := -1.0
	return func(fraction float64) {
		if report == nil {
			return
		}
		if fraction < 0 {
			fraction = 0
		}
		if fraction > 1 {
			fraction = 1
		}
		mu.Lock()
		defer mu.Unlock()
		if fraction+1e-9 < last {
			return
		}
		last = fraction
		report(fraction)
	}
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

func partBytes(parts *archive, path string) []byte {
	entry, ok := parts.get(path)
	if !ok {
		return nil
	}
	data, err := entry.bytes()
	if err != nil {
		return nil
	}
	return data
}

func readTheme(parts *archive) []byte {
	for _, name := range []string{"xl/theme/theme1.xml", "xl/theme/theme.xml"} {
		if data := partBytes(parts, name); len(data) > 0 {
			return data
		}
	}
	for _, entry := range parts.parts {
		name := entry.header.Name
		if entry.removed || !strings.Contains(name, "/theme/") || !strings.HasSuffix(name, ".xml") {
			continue
		}
		data, err := entry.bytes()
		if err == nil && len(data) > 0 {
			return data
		}
	}
	return nil
}

func normalizeWorkbookTarget(target string) string {
	clean := strings.TrimLeft(target, "/")
	if strings.HasPrefix(clean, "xl/") {
		return clean
	}
	return "xl/" + clean
}
