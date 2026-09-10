package xlsx

import (
	"errors"
	"strconv"
	"strings"

	"github.com/beevik/etree"

	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

type cellKey struct {
	row    int
	column int
}

func parseDocument(data []byte) (*etree.Document, error) {
	document := etree.NewDocument()
	document.ReadSettings.PreserveCData = true
	if err := document.ReadFromBytes(data); err != nil {
		return nil, err
	}
	if document.Root() == nil {
		return nil, errors.New("document has no root element")
	}
	return document, nil
}

// newChild creates an element inside parent, before the given sibling when it is
// not nil. The parent namespace prefix is inherited so documents written with an
// explicit prefix stay consistent with documents using the default namespace.
func newChild(parent *etree.Element, tag string, before *etree.Element) *etree.Element {
	if parent.Space != "" {
		tag = parent.Space + ":" + tag
	}
	element := etree.NewElement(tag)
	if before == nil {
		parent.AddChild(element)
		return element
	}
	parent.InsertChildAt(before.Index(), element)
	return element
}

// ensureChild returns the child of parent named tag, creating it before the
// first of followers when absent. OOXML parts are sequences, so an element
// appended at the end of the part can make the document invalid.
func ensureChild(parent *etree.Element, tag string, followers ...string) *etree.Element {
	if existing := parent.SelectElement(tag); existing != nil {
		return existing
	}
	var before *etree.Element
	for _, child := range parent.ChildElements() {
		if containsTag(followers, child.Tag) {
			before = child
			break
		}
	}
	return newChild(parent, tag, before)
}

func containsTag(tags []string, tag string) bool {
	for _, candidate := range tags {
		if candidate == tag {
			return true
		}
	}
	return false
}

func descendants(element *etree.Element, tag string) []*etree.Element {
	var found []*etree.Element
	for _, child := range element.ChildElements() {
		if child.Tag == tag {
			found = append(found, child)
		}
		found = append(found, descendants(child, tag)...)
	}
	return found
}

func elementText(element *etree.Element) string {
	var builder strings.Builder
	for _, token := range element.Child {
		if data, ok := token.(*etree.CharData); ok {
			builder.WriteString(data.Data)
		}
	}
	return builder.String()
}

// relationshipID reads the r:id attribute of a sheet entry without depending on
// the namespace prefix the document happens to use.
func relationshipID(element *etree.Element) string {
	for _, attr := range element.Attr {
		if attr.Key == "id" {
			return attr.Value
		}
	}
	return ""
}

func parseCellRef(reference string) (cellKey, bool) {
	letters := 0
	for letters < len(reference) && isColumnLetter(reference[letters]) {
		letters++
	}
	if letters == 0 || letters == len(reference) {
		return cellKey{}, false
	}
	column := spreadsheet.ParseColumnName(reference[:letters])
	row, err := strconv.Atoi(reference[letters:])
	if column == 0 || err != nil || row <= 0 {
		return cellKey{}, false
	}
	return cellKey{row: row, column: column}, true
}

func formatCellRef(row int, column int) string {
	return spreadsheet.ColumnName(column) + strconv.Itoa(row)
}

func isColumnLetter(char byte) bool {
	return char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z'
}
