package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"slices"
)

const KeySize = 32

const kekInfo = "order-fill twofa kek"

type Box struct {
	gcm    cipher.AEAD
	legacy cipher.AEAD
	key    []byte
}

func NewBox(key []byte) (*Box, error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("aes key must be %d bytes", KeySize)
	}
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	return &Box{gcm: gcm, key: slices.Clone(key)}, nil
}

func NewBoxFromMaster(master string) (*Box, error) {
	key, err := hkdf.Key(sha256.New, []byte(master), nil, kekInfo, KeySize)
	if err != nil {
		return nil, fmt.Errorf("derive kek: %w", err)
	}
	box, err := NewBox(key)
	if err != nil {
		return nil, err
	}
	legacySum := sha256.Sum256([]byte(master))
	legacy, err := newGCM(legacySum[:])
	if err != nil {
		return nil, err
	}
	box.legacy = legacy
	return box, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return gcm, nil
}

func (b *Box) HMAC(raw string) string {
	mac := hmac.New(sha256.New, b.key)
	_, _ = io.WriteString(mac, raw)
	return hex.EncodeToString(mac.Sum(nil))
}

func (b *Box) Seal(plaintext string) (string, error) {
	nonce := make([]byte, b.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	sealed := b.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (b *Box) Open(blob string) (string, error) {
	plain, err := openWith(b.gcm, blob)
	if err == nil {
		return plain, nil
	}
	if b.legacy == nil {
		return "", err
	}
	return openWith(b.legacy, blob)
}

func openWith(aead cipher.AEAD, blob string) (string, error) {
	raw, err := base64.RawStdEncoding.DecodeString(blob)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	ns := aead.NonceSize()
	if len(raw) < ns {
		return "", fmt.Errorf("ciphertext too short")
	}
	plain, err := aead.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
