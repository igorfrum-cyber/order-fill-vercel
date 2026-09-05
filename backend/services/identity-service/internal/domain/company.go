package domain

import "time"

type Company struct {
	ID              string
	Name            string
	LoginSlug       string
	LogoContentType string
	MatchingMode    MatchingMode
	CreatedAt       time.Time
	DisabledAt      *time.Time
}

func (c Company) Disabled() bool {
	return c.DisabledAt != nil
}

func (c Company) HasLogo() bool {
	return c.LogoContentType != ""
}
