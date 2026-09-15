package domain

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	maxProfileTextLength = 300
	maxBrandTerms        = 20
	maxDiscountBPS       = 10_000
)

var brandKeyPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,49}$`)
var phonePattern = regexp.MustCompile(`^[+0-9][0-9 ()+-]*$`)

func NormalizeOrderProfile(profile OrderProfile) (OrderProfile, error) {
	profile.LegalName = strings.TrimSpace(profile.LegalName)
	profile.Consignee = strings.TrimSpace(profile.Consignee)
	profile.Address = strings.TrimSpace(profile.Address)
	profile.ContactName = strings.TrimSpace(profile.ContactName)
	profile.ContactPhone = strings.TrimSpace(profile.ContactPhone)
	profile.Carrier = strings.TrimSpace(profile.Carrier)
	profile.DeliveryPayer = strings.TrimSpace(profile.DeliveryPayer)
	profile.DeliveryDestination = strings.TrimSpace(profile.DeliveryDestination)

	for name, value := range map[string]string{
		"legal_name": profile.LegalName, "consignee": profile.Consignee, "address": profile.Address,
		"contact_name": profile.ContactName, "contact_phone": profile.ContactPhone, "carrier": profile.Carrier,
		"delivery_payer": profile.DeliveryPayer, "delivery_destination": profile.DeliveryDestination,
	} {
		if len([]rune(value)) > maxProfileTextLength {
			return OrderProfile{}, fmt.Errorf("%w: %s is too long", ErrInvalid, name)
		}
	}
	if profile.ContactPhone != "" {
		digits := 0
		for _, char := range profile.ContactPhone {
			if char >= '0' && char <= '9' {
				digits++
			}
		}
		if !phonePattern.MatchString(profile.ContactPhone) || digits < 6 {
			return OrderProfile{}, fmt.Errorf("%w: invalid contact_phone", ErrInvalid)
		}
	}
	if len(profile.BrandTerms) > maxBrandTerms {
		return OrderProfile{}, fmt.Errorf("%w: too many brand terms", ErrInvalid)
	}
	seen := make(map[string]struct{}, len(profile.BrandTerms))
	terms := make([]BrandTerms, 0, len(profile.BrandTerms))
	for _, item := range profile.BrandTerms {
		item.Brand = strings.ToLower(strings.TrimSpace(item.Brand))
		item.DealerName = strings.TrimSpace(item.DealerName)
		item.PaymentMethod = strings.TrimSpace(item.PaymentMethod)
		item.PaymentControl = strings.TrimSpace(item.PaymentControl)
		item.CustomerType = strings.TrimSpace(item.CustomerType)
		if !brandKeyPattern.MatchString(item.Brand) {
			return OrderProfile{}, fmt.Errorf("%w: invalid brand", ErrInvalid)
		}
		if _, ok := seen[item.Brand]; ok {
			return OrderProfile{}, fmt.Errorf("%w: duplicate brand", ErrInvalid)
		}
		seen[item.Brand] = struct{}{}
		if !item.DiscountSet {
			item.DiscountBasisPoints = 0
		}
		if item.DiscountBasisPoints < 0 || item.DiscountBasisPoints > maxDiscountBPS {
			return OrderProfile{}, fmt.Errorf("%w: discount must be between 0 and 100", ErrInvalid)
		}
		for _, value := range []string{item.DealerName, item.PaymentMethod, item.PaymentControl, item.CustomerType} {
			if len([]rune(value)) > maxProfileTextLength {
				return OrderProfile{}, fmt.Errorf("%w: brand term is too long", ErrInvalid)
			}
		}
		terms = append(terms, item)
	}
	profile.BrandTerms = terms
	return profile, nil
}
