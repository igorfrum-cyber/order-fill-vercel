package orderfill

import (
	"regexp"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

// ChristinaLine describes one PROFF product line from the supplier blank.
// The full required-article list comes from the blank itself, not just the
// matched rows — so the planner knows whether a line is complete.
type ChristinaLine struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Article  string   `json:"article"`
	Required []string `json:"required"`
}

// ChristinaProffLinesByRow returns a map from blank row number to the
// ChristinaLine it belongs to. Only rows inside a PROFF line group are mapped;
// HOME rows and everything else return nothing. It is exported so the North
// pipeline can attach the same line metadata to its rows.
//
// The algorithm mirrors origin/main christinaProffLines: coloured header rows
// introduce a new line group, and CHR-article rows are members of the current
// group. A row with the special header "ОБЩАЯ ЛИНИЯ" ends the current group.
//
// Colour detection uses the Styled type-assertion available on xlsx sheets.
// When the sheet is not Styled (e.g. in unit tests with a plain stub), the
// function falls back to detecting groups solely by article pattern.
func ChristinaProffLinesByRow(sheet spreadsheet.Sheet) map[int]*ChristinaLine {
	styled, hasStyles := sheet.(spreadsheet.Styled)

	// Collect all articles per prospective line so Required is populated once
	// we finish reading the sheet.
	type lineEntry struct {
		line     ChristinaLine
		rows     []int
		articles []string
	}

	var lines []*lineEntry
	var current *lineEntry

	bounds := sheet.Bounds()
	for row := 1; row <= bounds.MaxRow; row++ {
		article := strings.ToUpper(strings.TrimSpace(normalize.AsText(sheet.Value(row, 1))))
		name := strings.TrimSpace(normalize.AsText(sheet.Value(row, 2)))

		// A non-article row with a non-empty name may be a line header.
		if article == "" && name != "" {
			if isChristinaGeneralLine(name) {
				current = nil
				continue
			}
			if isChrominaLineHeader(sheet, styled, hasStyles, row) {
				label := lineHeaderLabel(name)
				current = &lineEntry{
					line: ChristinaLine{
						ID:   strings.ToUpper(label),
						Name: label,
					},
				}
				lines = append(lines, current)
			}
			continue
		}

		if !isChristinaArticle(article) || name == "" {
			continue
		}

		// The approved September PROFF blank has no Nuance heading — product
		// names identify the line (mirrors origin/main logic exactly).
		if isNuanceName(name) && (current == nil || current.line.ID != "NUANCE") {
			nuance := &lineEntry{
				line: ChristinaLine{ID: "NUANCE", Name: "Nuance"},
			}
			lines = append(lines, nuance)
			current = nuance
		}

		if current == nil {
			continue
		}
		current.rows = append(current.rows, row)
		current.articles = append(current.articles, article)
	}

	// Build required list and row→line map.
	result := make(map[int]*ChristinaLine)
	for _, entry := range lines {
		if len(entry.rows) == 0 {
			continue
		}
		entry.line.Required = entry.articles
		for i, row := range entry.rows {
			cl := entry.line // copy
			cl.Article = entry.articles[i]
			clPtr := cl
			result[row] = &clPtr
		}
	}
	return result
}

var christinaArticleRe = regexp.MustCompile(`(?i)^CHR\d+$`)

func isChristinaArticle(article string) bool {
	return christinaArticleRe.MatchString(article)
}

func isChristinaGeneralLine(name string) bool {
	return regexp.MustCompile(`(?i)ОБЩАЯ ЛИНИЯ`).MatchString(name)
}

func isNuanceName(name string) bool {
	return regexp.MustCompile(`(?i)^Nuance\b`).MatchString(name)
}

// isChrominaLineHeader decides whether the current row is a coloured band
// introducing a new product line. When styles are available it checks that
// every cell in columns 1–6 has a non-default fill (fillId > 1), matching
// origin/main. Without styles it falls back to detecting section headings by
// checking that the name does not look like an article row.
func isChrominaLineHeader(sheet spreadsheet.Sheet, styled spreadsheet.Styled, hasStyles bool, row int) bool {
	if !hasStyles {
		// Fallback: any non-article, non-empty name row inside PROFF is a header.
		return true
	}
	styles := styled.Styles()
	// All 6 columns must have a non-default fill.
	for col := 1; col <= 6; col++ {
		idx := styled.StyleIndex(row, col)
		if idx < 0 || idx >= len(styles) {
			return false
		}
		if styles[idx].Fill == "" || styles[idx].Fill == "#FFFFFF" || styles[idx].Fill == "none" {
			return false
		}
	}
	return true
}

// lineHeaderLabel strips the trailing "— N линий" / "– N линий" suffix
// that the blank sometimes appends to the line name.
func lineHeaderLabel(name string) string {
	idx := strings.IndexAny(name, "–—-")
	if idx > 0 {
		candidate := strings.TrimSpace(name[:idx])
		if candidate != "" {
			return candidate
		}
	}
	return name
}
