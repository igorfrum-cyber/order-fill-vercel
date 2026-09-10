package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

const (
	MinPasswordLength = 10
	MaxPasswordLength = 1024
	argonTime         = 1
	argonMemory       = 64 * 1024
	argonThreads      = 4
	argonKeyLen       = 32
	argonSaltLen      = 16
)

var (
	ErrPassword = errors.New("invalid password")
	ErrHash     = errors.New("invalid password hash")

	dummyOnce sync.Once
	dummyHash string
)

func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength || len(password) > MaxPasswordLength {
		return "", ErrPassword
	}
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func VerifyPassword(encoded string, password string) error {
	salt, want, timeCost, memory, threads, err := parseArgonHash(encoded)
	if err != nil {
		return err
	}
	keyLen := len(want)
	if keyLen < 16 || keyLen > 64 {
		return ErrHash
	}
	got := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(keyLen)) // #nosec G115 -- keyLen is 16..64
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return ErrPassword
	}
	return nil
}

// DummyPasswordHash is a real argon2id hash used on unknown-login paths so
// response timing does not reveal whether the account exists.
func DummyPasswordHash() string {
	dummyOnce.Do(func() {
		hash, err := HashPassword("dummy-password-for-timing")
		if err != nil {
			panic(err)
		}
		dummyHash = hash
	})
	return dummyHash
}

func parseArgonHash(encoded string) (salt []byte, key []byte, timeCost uint32, memory uint32, threads uint8, err error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, 0, 0, 0, ErrHash
	}
	var version int
	if _, err = fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return nil, nil, 0, 0, 0, ErrHash
	}
	for _, param := range strings.Split(parts[3], ",") {
		keyVal := strings.SplitN(param, "=", 2)
		if len(keyVal) != 2 {
			return nil, nil, 0, 0, 0, ErrHash
		}
		switch keyVal[0] {
		case "m":
			parsed, convErr := strconv.ParseUint(keyVal[1], 10, 32)
			if convErr != nil {
				return nil, nil, 0, 0, 0, ErrHash
			}
			memory = uint32(parsed)
		case "t":
			parsed, convErr := strconv.ParseUint(keyVal[1], 10, 32)
			if convErr != nil {
				return nil, nil, 0, 0, 0, ErrHash
			}
			timeCost = uint32(parsed)
		case "p":
			parsed, convErr := strconv.ParseUint(keyVal[1], 10, 8)
			if convErr != nil {
				return nil, nil, 0, 0, 0, ErrHash
			}
			threads = uint8(parsed)
		default:
			return nil, nil, 0, 0, 0, ErrHash
		}
	}
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return nil, nil, 0, 0, 0, ErrHash
	}
	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return nil, nil, 0, 0, 0, ErrHash
	}
	if timeCost == 0 || memory == 0 || threads == 0 || len(salt) == 0 || len(key) == 0 {
		return nil, nil, 0, 0, 0, ErrHash
	}
	return salt, key, timeCost, memory, threads, nil
}
