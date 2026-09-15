package orderfill

import (
	"testing"
	"time"
)

func TestApplyCompanyOrderProfileFillsAngiopharmLabels(t *testing.T) {
	t.Parallel()
	workbook := newFakeWorkbook("Бланк", [][]string{
		{"Грузополучатель", ""},
		{"Дата составления заявки", ""},
		{"Способ оплаты покупателем", ""},
		{"Контроль оплаты", ""},
		{"Адрес", ""},
		{"Контактное лицо получателя", ""},
		{"Телефон контактного лица получателя", ""},
		{"Транспортная компания", ""},
		{"Кто оплачивает доставку", ""},
		{"Доставка до адреса/терминала транспортной", ""},
		{"Укажите в списке если вы — Стоматолог", ""},
		{"Укажите размер Вашей скидки (%)", "35"},
	})
	ApplyCompanyOrderProfile(workbook, "angiopharm", CompanyOrderProfile{
		LegalName: "ООО Тест", Address: "Тюмень", ContactName: "Иван", ContactPhone: "+7 900 000-00-00",
		BrandTerms: []CompanyBrandTerms{{Brand: "angiopharm", PaymentMethod: "Безналичный", CustomerType: "Дистрибьютор", DiscountBasisPoints: 3_025, DiscountSet: true}},
	}, time.Date(2026, 9, 14, 0, 0, 0, 0, time.UTC))
	sheet := workbook.sheets[0]
	for row, want := range map[int]string{1: "ООО Тест", 2: "14.09.2026", 3: "Безналичный", 5: "Тюмень", 6: "Иван", 7: "+7 900 000-00-00", 11: "Дистрибьютор", 12: "30.25"} {
		if got := sheet.Value(row, 2); got != want {
			t.Fatalf("row %d = %q, want %q", row, got, want)
		}
	}
}

func TestApplyCompanyOrderProfileKeepsTemplateDiscountWhenUnset(t *testing.T) {
	t.Parallel()
	workbook := newFakeWorkbook("Бланк", [][]string{{"Укажите размер Вашей скидки (%)", "35"}})
	ApplyCompanyOrderProfile(workbook, "angiopharm", CompanyOrderProfile{}, time.Now())
	if got := workbook.sheets[0].Value(1, 2); got != "35" {
		t.Fatalf("discount = %q", got)
	}
}
