package brands

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"order-fill/backend/services/brand-service/internal/domain"
	"order-fill/backend/services/brand-service/internal/storage/static"
)

type Store interface {
	Get(ctx context.Context, brand string) (domain.Policy, bool, error)
	Save(ctx context.Context, policy domain.Policy, actorID string) error
}

type Service struct{ store Store }

func New(store ...Store) *Service {
	var selected Store
	if len(store) > 0 {
		selected = store[0]
	}
	return &Service{store: selected}
}

func (s *Service) GetPolicy(ctx context.Context, brand, variant string) (domain.Policy, error) {
	key := strings.ToLower(strings.TrimSpace(brand))
	p := static.Policy(key)
	if s.store != nil {
		stored, ok, err := s.store.Get(ctx, key)
		if err != nil {
			return domain.Policy{}, err
		}
		if ok {
			p = stored
		}
	}
	p.Variant = strings.TrimSpace(variant)
	return p, nil
}

func (s *Service) List(_ context.Context) []string {
	return static.List()
}

func (s *Service) UpdatePolicy(ctx context.Context, policy domain.Policy, actorID string) (domain.Policy, error) {
	policy.Key = strings.ToLower(strings.TrimSpace(policy.Key))
	policy.Label = strings.TrimSpace(policy.Label)
	policy.Variant = ""
	policy.AdjustmentLabel = strings.TrimSpace(policy.AdjustmentLabel)
	policy.AdjustmentComment = strings.TrimSpace(policy.AdjustmentComment)
	policy.BlankQuantityHeader = strings.TrimSpace(policy.BlankQuantityHeader)
	policy.BlankBoxHeader = strings.TrimSpace(policy.BlankBoxHeader)
	policy.BlankLayout = strings.TrimSpace(policy.BlankLayout)
	for i := range policy.ArticlePrefixAliases {
		policy.ArticlePrefixAliases[i] = strings.TrimSpace(policy.ArticlePrefixAliases[i])
	}
	policy.ArticlePrefixAliases = slices.Compact(policy.ArticlePrefixAliases)
	if err := validatePolicy(policy); err != nil {
		return domain.Policy{}, err
	}
	if s.store == nil {
		return domain.Policy{}, fmt.Errorf("save brand policy: %w", domain.ErrInvalidPolicy)
	}
	if err := s.store.Save(ctx, policy, actorID); err != nil {
		return domain.Policy{}, err
	}
	return policy, nil
}

func validatePolicy(policy domain.Policy) error {
	if !slices.Contains(static.List(), policy.Key) {
		return fmt.Errorf("%w: unknown brand", domain.ErrInvalidPolicy)
	}
	if policy.Label == "" || len([]rune(policy.Label)) > 80 {
		return fmt.Errorf("%w: label is required and must be at most 80 characters", domain.ErrInvalidPolicy)
	}
	adjustments := []domain.Adjustment{domain.AdjustmentNone, domain.AdjustmentBox, domain.AdjustmentMultiple, domain.AdjustmentNearestMultiple, domain.AdjustmentMinimum}
	if !slices.Contains(adjustments, policy.Adjustment) {
		return fmt.Errorf("%w: unsupported adjustment", domain.ErrInvalidPolicy)
	}
	if policy.Multiple < 0 || policy.Multiple > 10000 || policy.MinQuantity < 0 || policy.MinQuantity > 1000000 {
		return fmt.Errorf("%w: quantity values are out of range", domain.ErrInvalidPolicy)
	}
	if (policy.Adjustment == domain.AdjustmentMultiple || policy.Adjustment == domain.AdjustmentNearestMultiple) && policy.Multiple == 0 {
		return fmt.Errorf("%w: quantity multiple is required", domain.ErrInvalidPolicy)
	}
	if policy.Adjustment == domain.AdjustmentMinimum && policy.MinQuantity == 0 {
		return fmt.Errorf("%w: minimum quantity is required", domain.ErrInvalidPolicy)
	}
	stringsToCheck := []string{policy.AdjustmentLabel, policy.AdjustmentComment, policy.BlankQuantityHeader, policy.BlankBoxHeader, policy.BlankLayout}
	for _, value := range stringsToCheck {
		if len([]rune(value)) > 120 {
			return fmt.Errorf("%w: text value is too long", domain.ErrInvalidPolicy)
		}
	}
	if len(policy.ArticlePrefixAliases) > 20 {
		return fmt.Errorf("%w: too many article prefixes", domain.ErrInvalidPolicy)
	}
	for _, alias := range policy.ArticlePrefixAliases {
		if alias == "" || len([]rune(alias)) > 32 {
			return fmt.Errorf("%w: invalid article prefix", domain.ErrInvalidPolicy)
		}
	}
	return nil
}
