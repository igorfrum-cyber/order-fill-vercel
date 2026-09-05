package brands

import (
	"context"
	"strings"

	"order-fill/backend/services/brand-service/internal/domain"
	"order-fill/backend/services/brand-service/internal/storage/static"
)

type Service struct{}

func New() *Service { return &Service{} }

func (s *Service) GetPolicy(_ context.Context, brand, variant string) domain.Policy {
	p := static.Policy(strings.ToLower(strings.TrimSpace(brand)))
	p.Variant = strings.TrimSpace(variant)
	return p
}

func (s *Service) List(_ context.Context) []string {
	return static.List()
}
