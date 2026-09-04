package orderfill

import (
	"fmt"
	"regexp"
	"strings"

	"order-fill/services/document-service/internal/domain/brand"
	"order-fill/services/document-service/internal/domain/normalize"
	"order-fill/services/document-service/internal/domain/spreadsheet"
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

// PlanBlanks checks how many blanks the brand needs and labels CHRISTINA files.
func PlanBlanks(brandKey string, names []string) ([]BlankPlan, error) {
	if brandKey == "christina" {
		return planChristinaBlanks(names)
	}
	if len(names) != 1 {
		label := brand.Rule(brandKey).Label
		return nil, fmt.Errorf("%w: для %s нужен один бланк поставщика. Сейчас загружено несколько файлов — оставьте один", ErrInvalidInput, label)
	}
	return []BlankPlan{{Index: 0, ID: blankPlanID(0), Label: names[0]}}, nil
}

func blankPlanID(index int) string {
	return fmt.Sprintf("blank-%d", index+1)
}

func planChristinaBlanks(names []string) ([]BlankPlan, error) {
	if len(names) != 2 {
		return nil, fmt.Errorf("%w: для Christina нужны два бланка: HOME и PROFF. Сейчас выбран другой набор файлов", ErrInvalidInput)
	}
	first := blankLineKind(names[0])
	second := blankLineKind(names[1])
	if first == "" && second == "" {
		first, second = "home", "proff"
	}
	if first == "" && second != "" {
		first = remainingChristinaLine(second)
	}
	if second == "" && first != "" {
		second = remainingChristinaLine(first)
	}
	if first == "" || second == "" || first == second {
		return nil, fmt.Errorf("%w: не поняли, какой бланк HOME, а какой PROFF. Назовите файлы HOME и PROFF", ErrInvalidInput)
	}
	return []BlankPlan{
		{Index: 0, ID: blankPlanID(0), Label: strings.ToUpper(first)},
		{Index: 1, ID: blankPlanID(1), Label: strings.ToUpper(second)},
	}, nil
}

func remainingChristinaLine(kind string) string {
	if kind == "home" {
		return "proff"
	}
	if kind == "proff" {
		return "home"
	}
	return ""
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
