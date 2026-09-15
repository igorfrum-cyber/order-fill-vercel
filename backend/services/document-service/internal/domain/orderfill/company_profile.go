package orderfill

import (
	"strings"
	"time"

	"order-fill/backend/services/document-service/internal/domain/normalize"
	"order-fill/backend/services/document-service/internal/domain/spreadsheet"
)

type CompanyOrderProfile struct {
	LegalName           string
	Consignee           string
	Address             string
	ContactName         string
	ContactPhone        string
	Carrier             string
	DeliveryPayer       string
	DeliveryDestination string
	BrandTerms          []CompanyBrandTerms
}

type CompanyBrandTerms struct {
	Brand               string
	DealerName          string
	PaymentMethod       string
	PaymentControl      string
	CustomerType        string
	DiscountBasisPoints int32
	DiscountSet         bool
}

func ApplyCompanyOrderProfile(workbook spreadsheet.Workbook, brand string, profile CompanyOrderProfile, at time.Time) {
	terms := termsForBrand(profile.BrandTerms, brand)
	switch brand {
	case "angiopharm":
		fillAngiopharmProfile(workbook, profile, terms, at)
	case "skin_synergy":
		fillSkinSynergyProfile(workbook, profile, terms)
	case "klapp":
		fillKlappProfile(workbook, profile, terms)
	}
}

func fillAngiopharmProfile(workbook spreadsheet.Workbook, profile CompanyOrderProfile, terms CompanyBrandTerms, at time.Time) {
	values := map[string]string{
		"грузополучатель":                           firstFilled(profile.Consignee, profile.LegalName),
		"дата составления заявки":                   at.Format("02.01.2006"),
		"способ оплаты покупателем":                 terms.PaymentMethod,
		"контроль оплаты":                           terms.PaymentControl,
		"адрес":                                     profile.Address,
		"контактное лицо получателя":                profile.ContactName,
		"телефон контактного лица получателя":       profile.ContactPhone,
		"транспортная компания":                     profile.Carrier,
		"кто оплачивает доставку":                   profile.DeliveryPayer,
		"доставка до адреса терминала транспортной": profile.DeliveryDestination,
		"укажите в списке если вы стоматолог":       terms.CustomerType,
	}
	for _, sheet := range workbook.Sheets() {
		for label, value := range values {
			setAfterLabel(sheet, label, value)
		}
		if terms.DiscountSet {
			setNumberAfterLabel(sheet, "укажите размер вашей скидки %", float64(terms.DiscountBasisPoints)/100)
		}
	}
}

func fillSkinSynergyProfile(workbook spreadsheet.Workbook, profile CompanyOrderProfile, terms CompanyBrandTerms) {
	for _, sheet := range workbook.Sheets() {
		setAfterLabel(sheet, "дилер", firstFilled(terms.DealerName, profile.LegalName))
		if terms.DiscountSet {
			setNumberAfterLabel(sheet, "проставьте вашу скидку", float64(terms.DiscountBasisPoints)/10_000)
		}
	}
}

func fillKlappProfile(workbook spreadsheet.Workbook, profile CompanyOrderProfile, terms CompanyBrandTerms) {
	for _, sheet := range workbook.Sheets() {
		bounds := sheet.Bounds()
		for row := 1; row <= min(bounds.MaxRow, 20); row++ {
			for column := 1; column <= bounds.MaxColumn; column++ {
				label := normalize.NormalizeHeader(sheet.Value(row, column))
				switch {
				case strings.HasPrefix(label, "заказ от просьба указать юр лицо") && profile.LegalName != "":
					sheet.SetText(row, column, "Заказ от (просьба указать юр. лицо): "+profile.LegalName)
				case label == "ваша скидка" && terms.DiscountSet:
					sheet.SetNumber(row, column+3, float64(terms.DiscountBasisPoints)/10_000)
				}
			}
		}
	}
}

func setAfterLabel(sheet spreadsheet.Sheet, wanted, value string) {
	if value == "" {
		return
	}
	if row, column, ok := findLabel(sheet, wanted); ok {
		sheet.SetText(row, columnAfterMerge(sheet, row, column), value)
	}
}

func setNumberAfterLabel(sheet spreadsheet.Sheet, wanted string, value float64) {
	if row, column, ok := findLabel(sheet, wanted); ok {
		sheet.SetNumber(row, columnAfterMerge(sheet, row, column), value)
	}
}

func findLabel(sheet spreadsheet.Sheet, wanted string) (int, int, bool) {
	wanted = normalize.NormalizeHeader(wanted)
	bounds := sheet.Bounds()
	for row := 1; row <= min(bounds.MaxRow, 30); row++ {
		for column := 1; column <= bounds.MaxColumn; column++ {
			if strings.HasPrefix(normalize.NormalizeHeader(sheet.Value(row, column)), wanted) {
				return row, column, true
			}
		}
	}
	return 0, 0, false
}

func columnAfterMerge(sheet spreadsheet.Sheet, row, column int) int {
	styled, ok := sheet.(spreadsheet.Styled)
	if !ok {
		return column + 1
	}
	for _, merge := range styled.Merges() {
		if merge.Row == row && merge.Column == column {
			return column + merge.Width
		}
	}
	return column + 1
}

func termsForBrand(items []CompanyBrandTerms, brand string) CompanyBrandTerms {
	for _, item := range items {
		if item.Brand == brand {
			return item
		}
	}
	return CompanyBrandTerms{}
}

func firstFilled(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
