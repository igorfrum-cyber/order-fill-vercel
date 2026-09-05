package twofa

import (
	"context"
	"errors"
	"slices"
	"strings"
	"time"

	"order-fill/backend/services/twofa-service/internal/domain"
	"order-fill/backend/services/twofa-service/internal/ratelimit"
	"order-fill/backend/services/twofa-service/internal/secret"
	"order-fill/backend/services/twofa-service/internal/totp"
)

type Store interface {
	Save(ctx context.Context, cred domain.Credential) error
	Get(ctx context.Context, userID string) (domain.Credential, error)
	Delete(ctx context.Context, userID string) error
}

type Service struct {
	store Store
	box   *secret.Box
	limit *ratelimit.Limiter
	now   func() time.Time
}

func New(store Store, box *secret.Box, limit *ratelimit.Limiter, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{store: store, box: box, limit: limit, now: now}
}

type Setup struct {
	Secret     string
	OtpauthURL string
	QRPNG      []byte
}

func (s *Service) Setup(ctx context.Context, userID, accountName string) (Setup, error) {
	existing, err := s.load(ctx, userID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return Setup{}, err
	}
	if err == nil && existing.IsEnabled() {
		return Setup{}, domain.ErrConflict
	}
	plain, err := totp.NewTOTPSecret()
	if err != nil {
		return Setup{}, err
	}
	authURL, err := totp.TOTPAuthURL(plain, accountName)
	if err != nil {
		return Setup{}, err
	}
	png, err := totp.TOTPQR(plain, accountName)
	if err != nil {
		return Setup{}, err
	}
	if err := s.save(ctx, domain.Credential{UserID: userID, Secret: plain}); err != nil {
		return Setup{}, err
	}
	return Setup{Secret: plain, OtpauthURL: authURL, QRPNG: png}, nil
}

func (s *Service) Enable(ctx context.Context, userID, code string) ([]string, error) {
	cred, err := s.load(ctx, userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if cred.IsEnabled() {
		return nil, domain.ErrConflict
	}
	if err := totp.VerifyTOTP(cred.Secret, code, s.now()); err != nil {
		return nil, domain.ErrUnauthorized
	}
	raw, hashes, err := totp.GenerateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}
	cred.Enabled = true
	cred.RecoveryCodeHashes = hashes
	if err := s.save(ctx, cred); err != nil {
		return nil, err
	}
	return raw, nil
}

func (s *Service) Disable(ctx context.Context, userID, code string) error {
	if strings.TrimSpace(code) == "" {
		// ponytail: gateway already checked the account password; twofa has no actor auth.
		return s.store.Delete(ctx, userID)
	}
	if _, err := s.Verify(ctx, userID, code); err != nil {
		return err
	}
	return s.store.Delete(ctx, userID)
}

func (s *Service) IsEnabled(ctx context.Context, userID string) (bool, error) {
	cred, err := s.load(ctx, userID)
	if errors.Is(err, domain.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return cred.IsEnabled(), nil
}

func (s *Service) Verify(ctx context.Context, userID, code string) (usedRecovery bool, err error) {
	if !s.limit.Allow(userID) {
		return false, domain.ErrLocked
	}
	cred, err := s.load(ctx, userID)
	if err != nil || !cred.IsEnabled() {
		s.limit.Fail(userID)
		return false, domain.ErrUnauthorized
	}
	if err := totp.VerifyTOTP(cred.Secret, code, s.now()); err == nil {
		s.limit.Clear(userID)
		return false, nil
	}
	remaining, recErr := totp.ConsumeRecoveryCode(slices.Clone(cred.RecoveryCodeHashes), code)
	if recErr != nil {
		s.limit.Fail(userID)
		return false, domain.ErrUnauthorized
	}
	cred.RecoveryCodeHashes = remaining
	if err := s.save(ctx, cred); err != nil {
		return false, err
	}
	s.limit.Clear(userID)
	return true, nil
}

func (s *Service) save(ctx context.Context, cred domain.Credential) error {
	sealed, err := s.box.Seal(cred.Secret)
	if err != nil {
		return err
	}
	cred.Secret = sealed
	return s.store.Save(ctx, cred)
}

func (s *Service) load(ctx context.Context, userID string) (domain.Credential, error) {
	cred, err := s.store.Get(ctx, userID)
	if err != nil {
		return domain.Credential{}, err
	}
	plain, err := s.box.Open(cred.Secret)
	if err != nil {
		return domain.Credential{}, err
	}
	cred.Secret = plain
	return cred, nil
}
