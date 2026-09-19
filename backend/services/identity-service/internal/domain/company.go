package domain

import "time"

type Company struct {
	ID                 string
	Name               string
	LoginSlug          string
	LogoContentType    string
	MatchingMode       MatchingMode
	ChristinaProffMode ChristinaProffMode
	OrderProfile       OrderProfile
	CreatedAt          time.Time
	DisabledAt         *time.Time
}

type OrderProfile struct {
	LegalName           string       `json:"legal_name,omitempty"`
	Consignee           string       `json:"consignee,omitempty"`
	Address             string       `json:"address,omitempty"`
	ContactName         string       `json:"contact_name,omitempty"`
	ContactPhone        string       `json:"contact_phone,omitempty"`
	Carrier             string       `json:"carrier,omitempty"`
	DeliveryPayer       string       `json:"delivery_payer,omitempty"`
	DeliveryDestination string       `json:"delivery_destination,omitempty"`
	BrandTerms          []BrandTerms `json:"brand_terms,omitempty"`
}

type BrandTerms struct {
	Brand               string `json:"brand"`
	DealerName          string `json:"dealer_name,omitempty"`
	PaymentMethod       string `json:"payment_method,omitempty"`
	PaymentControl      string `json:"payment_control,omitempty"`
	CustomerType        string `json:"customer_type,omitempty"`
	DiscountBasisPoints int32  `json:"discount_basis_points,omitempty"`
	DiscountSet         bool   `json:"discount_set,omitempty"`
}

func (c Company) Disabled() bool {
	return c.DisabledAt != nil
}

func (c Company) HasLogo() bool {
	return c.LogoContentType != ""
}
