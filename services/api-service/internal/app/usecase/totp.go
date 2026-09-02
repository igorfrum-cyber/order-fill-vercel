package usecase

import (
	"context"
	"encoding/base64"
	"errors"

	"order-fill/services/api-service/internal/domain/identity"
)

type TOTPSetup struct {
	Secret      string
	AuthURL     string
	QRPNGBase64 string
}

func (a *Auth) StartTOTPSetup(ctx context.Context, actor identity.User) (TOTPSetup, error) {
	user, err := a.store.GetUserByID(ctx, actor.ID)
	if err != nil || user.Disabled() {
		return TOTPSetup{}, identity.ErrUnauthorized
	}
	existing, err := a.store.GetTOTP(ctx, user.ID)
	if err != nil && !errors.Is(err, identity.ErrNotFound) {
		return TOTPSetup{}, err
	}
	if err == nil && existing.Enabled() {
		return TOTPSetup{}, identity.ErrConflict
	}
	secret, err := identity.NewTOTPSecret()
	if err != nil {
		return TOTPSetup{}, err
	}
	authURL, err := identity.TOTPAuthURL(secret, user.Login)
	if err != nil {
		return TOTPSetup{}, err
	}
	png, err := identity.TOTPQR(secret, user.Login)
	if err != nil {
		return TOTPSetup{}, err
	}
	if err := a.store.SaveTOTPSetup(ctx, identity.TOTP{
		UserID:             user.ID,
		Secret:             secret,
		RecoveryCodeHashes: []string{},
	}); err != nil {
		return TOTPSetup{}, err
	}
	return TOTPSetup{
		Secret:      secret,
		AuthURL:     authURL,
		QRPNGBase64: base64.StdEncoding.EncodeToString(png),
	}, nil
}

func (a *Auth) EnableTOTP(ctx context.Context, actor identity.User, code string) ([]string, error) {
	settings, err := a.store.GetTOTP(ctx, actor.ID)
	if err != nil {
		return nil, identity.ErrNotFound
	}
	if settings.Enabled() {
		return nil, identity.ErrConflict
	}
	if err := identity.VerifyTOTP(settings.Secret, code, a.now()); err != nil {
		return nil, identity.ErrUnauthorized
	}
	raw, hashes, err := identity.GenerateRecoveryCodes(8)
	if err != nil {
		return nil, err
	}
	if err := a.store.ReplaceRecoveryCodes(ctx, actor.ID, hashes); err != nil {
		return nil, err
	}
	if err := a.store.EnableTOTP(ctx, actor.ID, a.now()); err != nil {
		return nil, err
	}
	return raw, nil
}

func (a *Auth) DisableTOTP(ctx context.Context, actor identity.User, password string) error {
	user, err := a.store.GetUserByID(ctx, actor.ID)
	if err != nil || user.Disabled() || user.PasswordHash == "" {
		_ = identity.VerifyPassword(identity.DummyPasswordHash(), password)
		return identity.ErrUnauthorized
	}
	if err := identity.VerifyPassword(user.PasswordHash, password); err != nil {
		return identity.ErrUnauthorized
	}
	if err := a.store.DisableTOTP(ctx, actor.ID); err != nil && !errors.Is(err, identity.ErrNotFound) {
		return err
	}
	return nil
}
