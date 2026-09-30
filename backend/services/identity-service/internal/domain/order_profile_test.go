package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeOrderProfileTrimsAndNormalizesBrandTerms(t *testing.T) {
	t.Parallel()

	got, err := NormalizeOrderProfile(OrderProfile{
		LegalName: "  ООО Тест  ", ContactPhone: " +7 (999) 123-45-67 ",
		BrandTerms: []BrandTerms{{
			Brand: "  ANGIOPHARM ", DealerName: " Dealer ", PaymentMethod: " invoice ",
			PaymentControl: " control ", CustomerType: " clinic ", DiscountBasisPoints: 500, DiscountSet: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.LegalName != "ООО Тест" || got.ContactPhone != "+7 (999) 123-45-67" {
		t.Fatalf("profile fields = %+v", got)
	}
	wantTerm := BrandTerms{
		Brand: "angiopharm", DealerName: "Dealer", PaymentMethod: "invoice",
		PaymentControl: "control", CustomerType: "clinic", DiscountBasisPoints: 500, DiscountSet: true,
	}
	if got.BrandTerms[0] != wantTerm {
		t.Fatalf("brand terms = %+v, want %+v", got.BrandTerms[0], wantTerm)
	}

	unset, err := NormalizeOrderProfile(OrderProfile{BrandTerms: []BrandTerms{{Brand: "klapp", DiscountBasisPoints: 900}}})
	if err != nil {
		t.Fatal(err)
	}
	if unset.BrandTerms[0].DiscountBasisPoints != 0 {
		t.Fatalf("unset discount was retained: %+v", unset.BrandTerms[0])
	}
}

func TestNormalizeOrderProfileRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		profile OrderProfile
	}{
		{name: "long profile text", profile: OrderProfile{Address: strings.Repeat("a", 301)}},
		{name: "phone characters", profile: OrderProfile{ContactPhone: "phone-123456"}},
		{name: "short phone", profile: OrderProfile{ContactPhone: "+12345"}},
		{name: "too many brands", profile: OrderProfile{BrandTerms: makeBrandTerms(maxBrandTerms + 1)}},
		{name: "invalid brand key", profile: OrderProfile{BrandTerms: []BrandTerms{{Brand: "bad-brand"}}}},
		{name: "duplicate normalized brand", profile: OrderProfile{BrandTerms: []BrandTerms{{Brand: "KLAPP"}, {Brand: " klapp "}}}},
		{name: "negative discount", profile: OrderProfile{BrandTerms: []BrandTerms{{Brand: "klapp", DiscountSet: true, DiscountBasisPoints: -1}}}},
		{name: "discount above 100 percent", profile: OrderProfile{BrandTerms: []BrandTerms{{Brand: "klapp", DiscountSet: true, DiscountBasisPoints: maxDiscountBPS + 1}}}},
		{name: "long brand field", profile: OrderProfile{BrandTerms: []BrandTerms{{Brand: "klapp", DealerName: strings.Repeat("д", maxProfileTextLength+1)}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := NormalizeOrderProfile(tt.profile)
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("NormalizeOrderProfile() error = %v, want ErrInvalid", err)
			}
		})
	}
}

func makeBrandTerms(count int) []BrandTerms {
	terms := make([]BrandTerms, count)
	for index := range terms {
		terms[index].Brand = "brand"
	}
	return terms
}
