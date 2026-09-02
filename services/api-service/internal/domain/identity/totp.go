package identity

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"image/png"
	"net/url"
	"strings"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

const (
	totpIssuer         = "Order Fill"
	totpPeriod         = uint(30)
	recoveryCodeCount  = 8
	recoveryCodeBytes  = 5
	recoveryCodeDigits = 8
)

func NewTOTPSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
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
	code, err := totp.GenerateCodeCustom(secret, at, totpOpts())
	if err != nil {
		return "", fmt.Errorf("generate totp code: %w", err)
	}
	return code, nil
}

func VerifyTOTP(secret string, code string, at time.Time) error {
	ok, err := totp.ValidateCustom(strings.TrimSpace(code), secret, at, totpOpts())
	if err != nil || !ok {
		return ErrInvalidTOTP
	}
	return nil
}

func GenerateRecoveryCodes(count int) ([]string, []string, error) {
	if count <= 0 {
		count = recoveryCodeCount
	}
	raw := make([]string, count)
	hashes := make([]string, count)
	for i := 0; i < count; i++ {
		buf := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(buf); err != nil {
			return nil, nil, fmt.Errorf("generate recovery code: %w", err)
		}
		encoded := strings.ToUpper(fmt.Sprintf("%x", buf))
		if len(encoded) < recoveryCodeDigits {
			return nil, nil, fmt.Errorf("generate recovery code: too short")
		}
		code := encoded[:4] + "-" + encoded[4:recoveryCodeDigits]
		raw[i] = code
		hashes[i] = HashSecret(normalizeRecoveryCode(code))
	}
	return raw, hashes, nil
}

func ConsumeRecoveryCode(hashes []string, code string) ([]string, error) {
	want := HashSecret(normalizeRecoveryCode(code))
	found := -1
	for i, hash := range hashes {
		if SecretEqual(hash, want) {
			found = i
			break
		}
	}
	if found < 0 {
		return nil, ErrInvalidTOTP
	}
	remaining := make([]string, 0, len(hashes)-1)
	remaining = append(remaining, hashes[:found]...)
	remaining = append(remaining, hashes[found+1:]...)
	return remaining, nil
}

func normalizeRecoveryCode(code string) string {
	trimmed := strings.TrimSpace(code)
	trimmed = strings.ReplaceAll(trimmed, "-", "")
	trimmed = strings.ReplaceAll(trimmed, " ", "")
	return strings.ToUpper(trimmed)
}

func totpOpts() totp.ValidateOpts {
	return totp.ValidateOpts{
		Period:    totpPeriod,
		Skew:      1,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	}
}

func TOTPAuthURL(secret string, accountName string) (string, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return "", ErrInvalidTOTP
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
