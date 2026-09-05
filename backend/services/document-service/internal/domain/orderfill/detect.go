package orderfill

import (
	"cmp"
	"fmt"
	"regexp"
	"strings"

	"order-fill/backend/services/document-service/internal/domain/brand"
	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

var nomenclatureGroupPattern = regexp.MustCompile(`(?i)в группе\s*[«"']([^»"']+)[»"']`)

// DetectBrand reads the 1C nomenclature-group filter from the export header.
func DetectBrand(workbook spreadsheet.Workbook) (string, error) {
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 40); row++ {
			for column := 1; column <= bounds.MaxColumn; column++ {
				text := normalize.AsText(sheet.Value(row, column))
				if text == "" {
					continue
				}
				match := nomenclatureGroupPattern.FindStringSubmatch(text)
				if match == nil {
					continue
				}
				group := strings.TrimSpace(match[1])
				key, ok := brand.KeyFromNomenclatureGroup(group)
				if !ok {
					return "", fmt.Errorf("%w: не узнали бренд «%s». Проверьте отбор номенклатуры в выгрузке 1С", ErrInvalidInput, group)
				}
				return key, nil
			}
		}
	}
	return "", fmt.Errorf("%w: этот файл не похож на таблицу продаж из 1С. Загрузите выгрузку с отбором номенклатуры, а не бланк поставщика", ErrInvalidInput)
}

// BlankPlan is one uploaded supplier blank after brand-specific checks.
type BlankPlan struct {
	Index int
	ID    string
	Label string
}

// PlanBlanks checks that the brand got exactly one supplier blank.
func PlanBlanks(brandKey string, names []string) ([]BlankPlan, error) {
	if len(names) != 1 {
		label := brand.Rule(brandKey).Label
		return nil, fmt.Errorf("%w: для %s нужен один бланк поставщика. Сейчас загружено несколько файлов — оставьте один", ErrInvalidInput, label)
	}
	return []BlankPlan{{Index: 0, ID: blankPlanID(0), Label: names[0]}}, nil
}

func blankPlanID(index int) string {
	return fmt.Sprintf("blank-%d", index+1)
}

// LabelChristinaBlank names the blank HOME or PROFF from section headers,
// then from the file name.
func LabelChristinaBlank(workbook spreadsheet.Workbook, fileName string) string {
	kind := cmp.Or(christinaLineFromWorkbook(workbook), blankLineKind(fileName))
	if kind == "" {
		return fileName
	}
	return strings.ToUpper(kind)
}

func christinaLineFromWorkbook(workbook spreadsheet.Workbook) string {
	var home, proff int
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= bounds.MaxRow; row++ {
			for column := 1; column <= bounds.MaxColumn; column++ {
				switch christinaSectionKind(sheet.Value(row, column)) {
				case "proff":
					proff++
				case "home":
					home++
				}
			}
		}
	}
	if proff > home {
		return "proff"
	}
	if home > proff {
		return "home"
	}
	return ""
}

func christinaSectionKind(text string) string {
	switch normalize.NormalizeHeader(text) {
	case "профессиональный уход", "профессиональная линия":
		return "proff"
	case "домашний уход", "домашняя линия", "home care":
		return "home"
	default:
		return ""
	}
}

func blankLineKind(name string) string {
	value := strings.ToLower(strings.ReplaceAll(name, "ё", "е"))
	proff := strings.Contains(value, "proff") || strings.Contains(value, "prof") || strings.Contains(value, "проф")
	home := strings.Contains(value, "home") || strings.Contains(value, "дом")
	if proff && !home {
		return "proff"
	}
	if home && !proff {
		return "home"
	}
	return ""
}
