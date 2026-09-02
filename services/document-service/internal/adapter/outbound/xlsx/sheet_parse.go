package xlsx

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/beevik/etree"
)

const parallelRowThreshold = 24

type worksheetSplit struct {
	prologue []byte
	rows     [][]byte
	epilogue []byte
}

func parseSheetDocument(data []byte, shared []string, report func(float64)) (*etree.Document, map[int]*etree.Element, map[cellKey]*cell, error) {
	split, ok := splitWorksheet(data)
	if !ok || len(split.rows) < parallelRowThreshold {
		document, err := parseDocument(data)
		if err != nil {
			return nil, nil, nil, err
		}
		rows, cells := indexSheet(document, shared, report)
		return document, rows, cells, nil
	}

	skeleton, err := parseDocument(concatBytes(split.prologue, split.epilogue))
	if err != nil {
		document, fallbackErr := parseDocument(data)
		if fallbackErr != nil {
			return nil, nil, nil, err
		}
		rows, cells := indexSheet(document, shared, report)
		return document, rows, cells, nil
	}
	sheetData := findDirectSheetData(skeleton.Root())
	if sheetData == nil {
		document, err := parseDocument(data)
		if err != nil {
			return nil, nil, nil, err
		}
		rows, cells := indexSheet(document, shared, report)
		return document, rows, cells, nil
	}

	open, closeTag, ok := worksheetEnvelope(data)
	if !ok {
		document, err := parseDocument(data)
		if err != nil {
			return nil, nil, nil, err
		}
		rows, cells := indexSheet(document, shared, report)
		return document, rows, cells, nil
	}

	chunks := chunkRowsByWorkers(split.rows)
	type chunkResult struct {
		rows  []*etree.Element
		index map[int]*etree.Element
		cells map[cellKey]*cell
		err   error
	}
	results := make([]chunkResult, len(chunks))
	var done atomic.Int64
	total := len(split.rows)
	reportProgress(report, 0)

	runWorkers(len(chunks), func(i int) {
		chunkXML := wrapRowChunk(open, chunks[i], closeTag)
		document, err := parseDocument(chunkXML)
		if err != nil {
			results[i].err = err
			return
		}
		dataElement := findDirectSheetData(document.Root())
		if dataElement == nil {
			results[i].err = fmt.Errorf("chunk %d has no sheetData", i)
			return
		}
		index := make(map[int]*etree.Element, len(chunks[i]))
		cells := make(map[cellKey]*cell, len(chunks[i])*24)
		rows := make([]*etree.Element, 0, len(chunks[i]))
		for _, rowElement := range dataElement.ChildElements() {
			if rowElement.Tag != "row" && !hasLocalName(rowElement.Tag, "row") {
				continue
			}
			dataElement.RemoveChild(rowElement)
			indexRow(rowElement, shared, index, cells)
			rows = append(rows, rowElement)
			current := done.Add(1)
			if current == int64(total) || current%512 == 0 {
				reportProgress(report, 0.05+0.9*float64(current)/float64(total))
			}
		}
		results[i] = chunkResult{rows: rows, index: index, cells: cells}
	})

	for _, result := range results {
		if result.err != nil {
			document, err := parseDocument(data)
			if err != nil {
				return nil, nil, nil, result.err
			}
			rows, cells := indexSheet(document, shared, report)
			return document, rows, cells, nil
		}
	}

	rows := make(map[int]*etree.Element, total)
	cells := make(map[cellKey]*cell, total*24)
	for _, result := range results {
		for _, rowElement := range result.rows {
			sheetData.AddChild(rowElement)
		}
		for number, rowElement := range result.index {
			if _, exists := rows[number]; !exists {
				rows[number] = rowElement
			}
		}
		for key, value := range result.cells {
			cells[key] = value
		}
	}
	reportProgress(report, 1)
	return skeleton, rows, cells, nil
}

func indexSheet(document *etree.Document, shared []string, report func(float64)) (map[int]*etree.Element, map[cellKey]*cell) {
	root := document.Root()
	if root == nil {
		return map[int]*etree.Element{}, map[cellKey]*cell{}
	}
	sheetData := findDirectSheetData(root)
	if sheetData == nil {
		return map[int]*etree.Element{}, map[cellKey]*cell{}
	}
	rowElements := sheetData.SelectElements("row")
	total := len(rowElements)
	loadedRows := make(map[int]*etree.Element, total)
	cells := make(map[cellKey]*cell, total*24)
	if total == 0 {
		reportProgress(report, 1)
		return loadedRows, cells
	}
	if total < parallelRowThreshold {
		for index, rowElement := range rowElements {
			indexRow(rowElement, shared, loadedRows, cells)
			if index == total-1 || index%64 == 0 {
				reportProgress(report, float64(index+1)/float64(total))
			}
		}
		reportProgress(report, 1)
		return loadedRows, cells
	}

	type partial struct {
		rows  map[int]*etree.Element
		cells map[cellKey]*cell
	}
	parts := make([]partial, workerCount(total))
	var done atomic.Int64
	var wg sync.WaitGroup
	wg.Add(len(parts))
	stride := (total + len(parts) - 1) / len(parts)
	for w := 0; w < len(parts); w++ {
		start := w * stride
		end := start + stride
		if end > total {
			end = total
		}
		go func(slot int, from int, to int) {
			defer wg.Done()
			rows := make(map[int]*etree.Element, to-from)
			cells := make(map[cellKey]*cell, (to-from)*24)
			for index := from; index < to; index++ {
				indexRow(rowElements[index], shared, rows, cells)
				current := done.Add(1)
				if current == int64(total) || current%64 == 0 {
					reportProgress(report, float64(current)/float64(total))
				}
			}
			parts[slot] = partial{rows: rows, cells: cells}
		}(w, start, end)
	}
	wg.Wait()
	for _, part := range parts {
		for number, rowElement := range part.rows {
			if _, exists := loadedRows[number]; !exists {
				loadedRows[number] = rowElement
			}
		}
		for key, value := range part.cells {
			cells[key] = value
		}
	}
	reportProgress(report, 1)
	return loadedRows, cells
}

func findDirectSheetData(root *etree.Element) *etree.Element {
	if root == nil {
		return nil
	}
	if found := root.SelectElement("sheetData"); found != nil {
		return found
	}
	for _, child := range root.ChildElements() {
		if child.Tag == "sheetData" || hasLocalName(child.Tag, "sheetData") {
			return child
		}
	}
	return nil
}

func indexRow(rowElement *etree.Element, shared []string, rows map[int]*etree.Element, cells map[cellKey]*cell) {
	number, err := parsePositiveInt(rowElement.SelectAttrValue("r", ""))
	if err == nil && number > 0 {
		if _, exists := rows[number]; !exists {
			rows[number] = rowElement
		}
	}
	for _, cellElement := range rowElement.ChildElements() {
		if cellElement.Tag != "c" && !hasLocalName(cellElement.Tag, "c") {
			continue
		}
		key, ok := parseCellRef(cellElement.SelectAttrValue("r", ""))
		if !ok {
			continue
		}
		cells[key] = &cell{element: cellElement, value: readCellValue(cellElement, shared), xf: attrInt(cellElement, "s")}
	}
}

func parsePositiveInt(value string) (int, error) {
	n := 0
	if value == "" {
		return 0, fmt.Errorf("empty")
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(char-'0')
	}
	return n, nil
}

func chunkRowsByWorkers(rows [][]byte) [][][]byte {
	if len(rows) == 0 {
		return nil
	}
	workers := workerCount(len(rows))
	if workers < 2 {
		return [][][]byte{rows}
	}
	total := 0
	for _, row := range rows {
		total += len(row)
	}
	if total == 0 {
		return [][][]byte{rows}
	}
	target := total / workers
	if target < 1 {
		target = 1
	}
	chunks := make([][][]byte, 0, workers)
	start := 0
	acc := 0
	for index, row := range rows {
		acc += len(row)
		last := index == len(rows)-1
		filled := acc >= target && len(chunks) < workers-1
		if filled || last {
			chunks = append(chunks, rows[start:index+1])
			start = index + 1
			acc = 0
		}
	}
	return chunks
}

func worksheetEnvelope(data []byte) (open []byte, closeTag []byte, ok bool) {
	_, inner, prefix, found := findOpenTag(data, "worksheet")
	if !found || inner <= 0 {
		return nil, nil, false
	}
	open = data[:inner]
	if len(prefix) > 0 {
		closeTag = concatBytes([]byte("</"), prefix, []byte(":worksheet>"))
	} else {
		closeTag = []byte("</worksheet>")
	}
	return open, closeTag, true
}

func wrapRowChunk(open []byte, rows [][]byte, closeTag []byte) []byte {
	n := len(open) + len(closeTag) + len("<sheetData></sheetData>")
	for _, row := range rows {
		n += len(row)
	}
	out := make([]byte, 0, n)
	out = append(out, open...)
	out = append(out, "<sheetData>"...)
	for _, row := range rows {
		out = append(out, row...)
	}
	out = append(out, "</sheetData>"...)
	out = append(out, closeTag...)
	return out
}

func concatBytes(parts ...[]byte) []byte {
	n := 0
	for _, part := range parts {
		n += len(part)
	}
	out := make([]byte, 0, n)
	for _, part := range parts {
		out = append(out, part...)
	}
	return out
}

func reportProgress(report func(float64), fraction float64) {
	if report == nil {
		return
	}
	if fraction < 0 {
		fraction = 0
	}
	if fraction > 1 {
		fraction = 1
	}
	report(fraction)
}

func hasLocalName(tag string, local string) bool {
	if tag == local {
		return true
	}
	if i := bytes.LastIndexByte([]byte(tag), ':'); i >= 0 {
		return tag[i+1:] == local
	}
	return false
}

func splitWorksheet(data []byte) (worksheetSplit, bool) {
	openStart, inner, prefix, ok := findOpenTag(data, "sheetData")
	if !ok || openStart < 0 {
		return worksheetSplit{}, false
	}
	closeTag := []byte("</sheetData>")
	if len(prefix) > 0 {
		closeTag = concatBytes([]byte("</"), prefix, []byte(":sheetData>"))
	}
	closeStart := bytes.Index(data[inner:], closeTag)
	if closeStart < 0 {
		return worksheetSplit{}, false
	}
	closeStart += inner
	rows, ok := splitTopLevelElements(data[inner:closeStart], "row")
	if !ok {
		return worksheetSplit{}, false
	}
	return worksheetSplit{
		prologue: data[:inner],
		rows:     rows,
		epilogue: data[closeStart:],
	}, true
}

func findOpenTag(data []byte, local string) (start int, inner int, prefix []byte, ok bool) {
	needle := []byte(local)
	offset := 0
	for offset < len(data) {
		at := bytes.Index(data[offset:], needle)
		if at < 0 {
			return 0, 0, nil, false
		}
		abs := offset + at
		nameStart, foundPrefix, valid := tagNameAround(data, abs)
		if !valid {
			offset = abs + len(needle)
			continue
		}
		after := abs + len(needle)
		if after < len(data) && !isTagNameEnd(data[after]) {
			offset = abs + len(needle)
			continue
		}
		tagEnd := indexTagEnd(data, nameStart)
		if tagEnd < 0 {
			return 0, 0, nil, false
		}
		if tagEnd > 0 && data[tagEnd-1] == '/' {
			return nameStart, tagEnd + 1, foundPrefix, true
		}
		return nameStart, tagEnd + 1, foundPrefix, true
	}
	return 0, 0, nil, false
}

func tagNameAround(data []byte, localStart int) (start int, prefix []byte, ok bool) {
	if localStart > 0 && data[localStart-1] == '<' {
		return localStart - 1, nil, true
	}
	if localStart > 1 && data[localStart-1] == ':' {
		i := localStart - 2
		for i >= 0 && isNameChar(data[i]) {
			i--
		}
		if i >= 0 && data[i] == '<' {
			return i, data[i+1 : localStart-1], true
		}
	}
	return 0, nil, false
}

func isTagNameEnd(char byte) bool {
	return char == ' ' || char == '\t' || char == '\n' || char == '\r' || char == '>' || char == '/'
}

func isNameChar(char byte) bool {
	return char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '_'
}

func splitTopLevelElements(inner []byte, local string) ([][]byte, bool) {
	rows := make([][]byte, 0)
	i := 0
	for i < len(inner) {
		for i < len(inner) && inner[i] != '<' {
			i++
		}
		if i >= len(inner) {
			break
		}
		if i+1 < len(inner) && (inner[i+1] == '!' || inner[i+1] == '?' || inner[i+1] == '/') {
			tagEnd := indexTagEnd(inner, i)
			if tagEnd < 0 {
				return nil, false
			}
			i = tagEnd + 1
			continue
		}
		end, ok := elementEnd(inner, i)
		if !ok {
			return nil, false
		}
		if elementLocalName(inner[i:end]) == local {
			rows = append(rows, inner[i:end])
		}
		i = end
	}
	return rows, true
}

func elementLocalName(element []byte) string {
	if len(element) < 2 || element[0] != '<' {
		return ""
	}
	start := 1
	i := start
	for i < len(element) && !isTagNameEnd(element[i]) && element[i] != ':' {
		i++
	}
	if i < len(element) && element[i] == ':' {
		start = i + 1
		i = start
		for i < len(element) && !isTagNameEnd(element[i]) {
			i++
		}
	}
	return string(element[start:i])
}

func elementEnd(data []byte, start int) (int, bool) {
	tagEnd := indexTagEnd(data, start)
	if tagEnd < 0 {
		return 0, false
	}
	if tagEnd > start && data[tagEnd-1] == '/' {
		return tagEnd + 1, true
	}
	depth := 1
	i := tagEnd + 1
	for depth > 0 {
		next := bytes.IndexByte(data[i:], '<')
		if next < 0 {
			return 0, false
		}
		i += next
		if i+3 < len(data) && bytes.HasPrefix(data[i:], []byte("<!--")) {
			closeAt := bytes.Index(data[i+4:], []byte("-->"))
			if closeAt < 0 {
				return 0, false
			}
			i += 4 + closeAt + 3
			continue
		}
		if i+1 < len(data) && data[i+1] == '/' {
			depth--
			closeEnd := indexTagEnd(data, i)
			if closeEnd < 0 {
				return 0, false
			}
			i = closeEnd + 1
			continue
		}
		if i+1 < len(data) && (data[i+1] == '!' || data[i+1] == '?') {
			skipEnd := indexTagEnd(data, i)
			if skipEnd < 0 {
				return 0, false
			}
			i = skipEnd + 1
			continue
		}
		openEnd := indexTagEnd(data, i)
		if openEnd < 0 {
			return 0, false
		}
		if openEnd > i && data[openEnd-1] != '/' {
			depth++
		}
		i = openEnd + 1
	}
	return i, true
}

func indexTagEnd(data []byte, start int) int {
	inQuote := byte(0)
	for i := start; i < len(data); i++ {
		char := data[i]
		if inQuote != 0 {
			if char == inQuote {
				inQuote = 0
			}
			continue
		}
		if char == '"' || char == '\'' {
			inQuote = char
			continue
		}
		if char == '>' {
			return i
		}
	}
	return -1
}
