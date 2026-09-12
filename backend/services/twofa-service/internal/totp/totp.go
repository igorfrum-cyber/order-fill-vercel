package totp

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image/png"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/pquerna/otp"
	otptotp "github.com/pquerna/otp/totp"

	"order-fill/backend/services/twofa-service/internal/domain"
	"order-fill/backend/services/twofa-service/internal/secret"
)

const (
	totpIssuer         = "Order Fill"
	totpPeriod         = uint(30)
	recoveryCodeCount  = 8
	recoveryCodeBytes  = 10
	recoveryCodeDigits = 20
)

func NewTOTPSecret() (string, error) {
	key, err := otptotp.Generate(otptotp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: "pending",
		Period:      totpPeriod,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return "", fmt.Errorf("generate totp secret: %w", err)
	}
	return key.Secret(), nil
}

func CurrentTOTPCode(secret string, at time.Time) (string, error) {
	code, err := otptotp.GenerateCodeCustom(secret, at, totpOpts())
	if err != nil {
		return "", fmt.Errorf("generate totp code: %w", err)
	}
	return code, nil
}

func VerifyTOTP(secret string, code string, at time.Time) error {
	ok, err := otptotp.ValidateCustom(strings.TrimSpace(code), secret, at, totpOpts())
	if err != nil || !ok {
		return domain.ErrInvalidTOTP
	}
	return nil
}

func GenerateRecoveryCodes(count int, hash func(string) string) ([]string, []string, error) {
	if count <= 0 {
		count = recoveryCodeCount
	}
	if hash == nil {
		hash = secret.HashSecret
	}
	raw := make([]string, count)
	hashes := make([]string, count)
	for i := range count {
		buf := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, fmt.Errorf("generate recovery code: %w", err)
		}
		encoded := strings.ToUpper(fmt.Sprintf("%x", buf))
		if len(encoded) < recoveryCodeDigits {
			return nil, nil, fmt.Errorf("generate recovery code: too short")
		}
		code := formatRecoveryCode(encoded[:recoveryCodeDigits])
		raw[i] = code
		hashes[i] = hash(normalizeRecoveryCode(code))
	}
	return raw, hashes, nil
}

func ConsumeRecoveryCode(hashes []string, code string, hash func(string) string) ([]string, error) {
	norm := normalizeRecoveryCode(code)
	legacy := secret.HashSecret(norm)
	found := slices.IndexFunc(hashes, func(stored string) bool {
		if hash != nil && secret.SecretEqual(stored, hash(norm)) {
			return true
		}
		return secret.SecretEqual(stored, legacy)
	})
	if found < 0 {
		return nil, domain.ErrInvalidTOTP
	}
	return append(slices.Clone(hashes[:found]), hashes[found+1:]...), nil
}

func formatRecoveryCode(encoded string) string {
	var b strings.Builder
	for i := range encoded {
		if i > 0 && i%4 == 0 {
			b.WriteByte('-')
		}
		b.WriteByte(encoded[i])
	}
	return b.String()
}

func normalizeRecoveryCode(code string) string {
	trimmed := strings.TrimSpace(code)
	trimmed = strings.ReplaceAll(trimmed, "-", "")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	return strings.ToUpper(trimmed)
}

func totpOpts() otptotp.ValidateOpts {
	return otptotp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
}

func TOTPAuthURL(secret string, accountName string) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", domain.ErrInvalidTOTP
	}
	account := strings.TrimSpace(accountName)
	if account == "" {
		account = "user"
	}
	values := url.Values{}
	values.Set("secret", secret)
	values.Set("issuer", totpIssuer)
	values.Set("algorithm", "SHA1")
	values.Set("digits", "6")
	values.Set("period", "30")
	return "otpauth://totp/" + url.PathEscape(totpIssuer+":"+account) + "?" + values.Encode(), nil
}

func TOTPQR(secret string, accountName string) ([]byte, error) {
	authURL, err := TOTPAuthURL(secret, accountName)
	if err != nil {
		return nil, err
	}
	key, err := otp.NewKeyFromURL(authURL)
	if err != nil {
		return nil, fmt.Errorf("totp url: %w", err)
	}
	img, err := key.Image(200, 200)
	if err != nil {
		return nil, fmt.Errorf("totp qr: %w", err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("encode totp qr: %w", err)
	}
	return buf.Bytes(), nil
}
