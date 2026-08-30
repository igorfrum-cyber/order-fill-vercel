package xlsx

import (
	"fmt"
	"strings"
	"sync"

	"github.com/beevik/etree"

	"order-fill/services/document-service/internal/domain/spreadsheet"
)

// calcPrFollowers are the workbook elements that the schema places after
// calcPr; they position a calcPr element created for a workbook without one.
var calcPrFollowers = []string{
	"oleSize", "customWorkbookViews", "pivotCaches", "smartTagPr", "smartTagTypes",
	"webPublishing", "fileRecoveryPr", "webPublishObjects", "extLst",
}

type workbook struct {
	parts    *archive
	document *etree.Document
	sheets   []*sheet
	byName   map[string]*sheet
}

func (w *workbook) Sheets() []spreadsheet.Sheet {
	sheets := make([]spreadsheet.Sheet, 0, len(w.sheets))
	for _, item := range w.sheets {
		sheets = append(sheets, item)
	}
	return sheets
}

func (w *workbook) Sheet(name string) (spreadsheet.Sheet, bool) {
	found, ok := w.byName[name]
	if !ok {
		return nil, false
	}
	return found, true
}

func (w *workbook) Save() ([]byte, error) {
	var firstErr error
	var errOnce sync.Once
	runWorkers(len(w.sheets), func(index int) {
		item := w.sheets[index]
		if err := item.serialize(); err != nil {
			errOnce.Do(func() { firstErr = fmt.Errorf("xlsx: serialize sheet %q: %w", item.name, err) })
		}
	})
	if firstErr != nil {
		return nil, firstErr
	}
	if err := w.forceFormulaRecalculation(); err != nil {
		return nil, err
	}
	data, err := w.parts.write()
	if err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	return data, nil
}

// forceFormulaRecalculation makes Excel recompute every formula when the saved
// document is opened, and drops the cached calculation chain, which no longer
// matches the cells the domain changed.
func (w *workbook) forceFormulaRecalculation() error {
	entry, ok := w.parts.get(workbookPath)
	if !ok {
		return fmt.Errorf("xlsx: %s is missing", workbookPath)
	}
	calcPr := ensureChild(w.document.Root(), "calcPr", calcPrFollowers...)
	calcPr.CreateAttr("calcMode", "auto")
	calcPr.CreateAttr("fullCalcOnLoad", "1")
	calcPr.CreateAttr("forceFullCalc", "1")
	data, err := w.document.WriteToBytes()
	if err != nil {
		return fmt.Errorf("xlsx: serialize %s: %w", workbookPath, err)
	}
	entry.replace(data)

	w.parts.remove(calcChainPath)
	if err := w.removeElements(workbookRelsPath, "Relationship", "Type", "calcChain"); err != nil {
		return err
	}
	return w.removeElements(contentTypesPath, "Override", "PartName", "/"+calcChainPath)
}

func (w *workbook) removeElements(path string, tag string, attribute string, value string) error {
	entry, ok := w.parts.get(path)
	if !ok {
		return nil
	}
	data, err := entry.bytes()
	if err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}
	document, err := parseDocument(data)
	if err != nil {
		return fmt.Errorf("xlsx: parse %s: %w", path, err)
	}
	removed := false
	for _, element := range descendants(document.Root(), tag) {
		if !strings.Contains(element.SelectAttrValue(attribute, ""), value) {
			continue
		}
		if parent := element.Parent(); parent != nil {
			parent.RemoveChild(element)
			removed = true
		}
	}
	if !removed {
		return nil
	}
	serialized, err := document.WriteToBytes()
	if err != nil {
		return fmt.Errorf("xlsx: serialize %s: %w", path, err)
	}
	entry.replace(serialized)
	return nil
}
