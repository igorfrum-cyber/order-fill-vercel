package xlsx

// CopyCellStyle copies only the supplier's style, retaining the target value.
func (s *sheet) CopyCellStyle(row, sourceColumn, targetColumn int) {
	source := s.cells[cellKey{row: row, column: sourceColumn}]
	if source == nil {
		return
	}
	target := s.findOrCreateCell(row, targetColumn)
	target.xf = source.xf
	if style := source.element.SelectAttrValue("s", ""); style != "" {
		target.element.CreateAttr("s", style)
	} else {
		target.element.RemoveAttr("s")
	}
	s.modified = true
}
