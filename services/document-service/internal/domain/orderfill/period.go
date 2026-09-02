// Package orderfill holds the order-fill business rules. It works against the
// spreadsheet port and never touches zip, XML, HTTP or storage.
package orderfill

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"order-fill/services/document-service/internal/domain/normalize"
	"order-fill/services/document-service/internal/domain/spreadsheet"
)

// ErrInvalidInput marks a problem in the uploaded workbooks that the user has
// to fix. Callers map it to a 4xx-style job failure instead of a crash.
var ErrInvalidInput = errors.New("invalid workbook")

var monthsRU = [...]string{"", "январь", "февраль", "март", "апрель", "май", "июнь", "июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь"}

var (
	orderMonthPattern = regexp.MustCompile(`^(\d{4})-(\d{2})$`)
	periodPattern     = regexp.MustCompile(`(\d{1,2})\.(\d{1,2})\.(\d{4}).*?(\d{1,2})\.(\d{1,2})\.(\d{4})`)
)

// Period is a closed date range printed in the 1C export header.
type Period struct {
	Start time.Time
	End   time.Time
}

// PeriodInfo describes which periods were expected and which were found.
type PeriodInfo struct {
	OrderMonthLabel      string
	ExpectedMainPeriod   string
	ExpectedPrevious     string
	ActualMainPeriod     string
	ActualPreviousPeriod string
}

// ParseOrderMonth reads the "YYYY-MM" value chosen in the UI.
func ParseOrderMonth(value string) (time.Time, error) {
	match := orderMonthPattern.FindStringSubmatch(value)
	if match == nil {
		return time.Time{}, fmt.Errorf("%w: выберите месяц заказа", ErrInvalidInput)
	}
	year, _ := strconv.Atoi(match[1])
	month, _ := strconv.Atoi(match[2])
	if month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("%w: выберите месяц заказа", ErrInvalidInput)
	}
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC), nil
}

// ExpectedPeriods derives the 1C export periods required for an order month.
func ExpectedPeriods(orderMonth string) (main Period, previous Period, label string, err error) {
	orderDate, err := ParseOrderMonth(orderMonth)
	if err != nil {
		return Period{}, Period{}, "", err
	}
	mainStart := addMonths(orderDate, -13)
	main = Period{Start: mainStart, End: lastDayOfMonth(addMonths(orderDate, -2))}
	previous = Period{Start: mainStart, End: lastDayOfMonth(addMonths(mainStart, 2))}
	label = fmt.Sprintf("%s %d", monthsRU[int(orderDate.Month())], orderDate.Year())
	return main, previous, label, nil
}

// FormatPeriod renders a period the way the 1C export prints it.
func FormatPeriod(period Period) string {
	return formatDate(period.Start) + " - " + formatDate(period.End)
}

// FindSourcePeriods scans the export header for the period captions.
func FindSourcePeriods(workbook spreadsheet.Workbook) (main *Period, previous *Period) {
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 40); row++ {
			for column := 1; column <= bounds.MaxColumn; column++ {
				text := normalize.AsText(sheet.Value(row, column))
				if text == "" {
					continue
				}
				parsed := parsePeriodRange(text)
				if parsed == nil {
					continue
				}
				header := normalize.NormalizeHeader(text)
				switch {
				case strings.Contains(header, "прошлый период"):
					previous = parsed
				case strings.Contains(header, "период"):
					main = parsed
				}
			}
			if main != nil && previous != nil {
				return main, previous
			}
		}
	}
	return main, previous
}

// InferOrderMonth reconstructs the order month from the 1C period captions.
// The main period always ends two months before the order month.
func InferOrderMonth(workbook spreadsheet.Workbook) (string, PeriodInfo, error) {
	actualMain, actualPrevious := FindSourcePeriods(workbook)
	if actualMain == nil || actualPrevious == nil {
		return "", PeriodInfo{}, fmt.Errorf("%w: не нашел в таблице параметры периода и прошлого периода. Проверьте выгрузку из 1С", ErrInvalidInput)
	}
	orderDate := addMonths(time.Date(actualMain.End.Year(), actualMain.End.Month(), 1, 0, 0, 0, 0, time.UTC), 2)
	orderMonth := fmt.Sprintf("%04d-%02d", orderDate.Year(), int(orderDate.Month()))
	info, err := ValidateSourcePeriods(workbook, orderMonth)
	if err != nil {
		return "", PeriodInfo{}, err
	}
	return orderMonth, info, nil
}

// ValidateSourcePeriods refuses exports built for a different order month.
func ValidateSourcePeriods(workbook spreadsheet.Workbook, orderMonth string) (PeriodInfo, error) {
	expectedMain, expectedPrevious, label, err := ExpectedPeriods(orderMonth)
	if err != nil {
		return PeriodInfo{}, err
	}
	actualMain, actualPrevious := FindSourcePeriods(workbook)
	if actualMain == nil || actualPrevious == nil {
		return PeriodInfo{}, fmt.Errorf("%w: не нашел в таблице параметры периода и прошлого периода. Проверьте выгрузку из 1С", ErrInvalidInput)
	}
	if !periodsEqual(*actualMain, expectedMain) || !periodsEqual(*actualPrevious, expectedPrevious) {
		return PeriodInfo{}, fmt.Errorf(
			"%w: таблица расчета заказа сформирована не за тот период. Для заказа на %s нужен период %s, прошлый период %s. В загруженной таблице: период %s, прошлый период %s. Переделайте выгрузку из 1С с правильными параметрами",
			ErrInvalidInput,
			label,
			FormatPeriod(expectedMain),
			FormatPeriod(expectedPrevious),
			FormatPeriod(*actualMain),
			FormatPeriod(*actualPrevious),
		)
	}
	return PeriodInfo{
		OrderMonthLabel:      label,
		ExpectedMainPeriod:   FormatPeriod(expectedMain),
		ExpectedPrevious:     FormatPeriod(expectedPrevious),
		ActualMainPeriod:     FormatPeriod(*actualMain),
		ActualPreviousPeriod: FormatPeriod(*actualPrevious),
	}, nil
}

func parsePeriodRange(text string) *Period {
	match := periodPattern.FindStringSubmatch(normalize.AsText(text))
	if match == nil {
		return nil
	}
	numbers := make([]int, 6)
	for index := 0; index < 6; index++ {
		value, err := strconv.Atoi(match[index+1])
		if err != nil {
			return nil
		}
		numbers[index] = value
	}
	return &Period{
		Start: time.Date(numbers[2], time.Month(numbers[1]), numbers[0], 0, 0, 0, 0, time.UTC),
		End:   time.Date(numbers[5], time.Month(numbers[4]), numbers[3], 0, 0, 0, 0, time.UTC),
	}
}

func periodsEqual(left Period, right Period) bool {
	return left.Start.Equal(right.Start) && left.End.Equal(right.End)
}

func addMonths(date time.Time, months int) time.Time {
	return time.Date(date.Year(), date.Month()+time.Month(months), 1, 0, 0, 0, 0, time.UTC)
}

func lastDayOfMonth(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month()+1, 0, 0, 0, 0, 0, time.UTC)
}

func formatDate(date time.Time) string {
	return fmt.Sprintf("%02d.%02d.%d", date.Day(), int(date.Month()), date.Year())
}
