// Package secret encrypts the credentials the control plane has to keep.
//
// Storing an SSH key or password in plain text would mean a copy of the
// database is a copy of every server. Values are sealed with AES-256-GCM
// under a key supplied by the environment, so the database alone is not enough.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

const KeyBytes = 32

type Sealer struct {
	aead cipher.AEAD
}

// NewSealer takes a base64-encoded 32-byte key.
func NewSealer(encodedKey string) (*Sealer, error) {
	key, err := base64.StdEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("secret: key is not valid base64: %w", err)
	}
	if len(key) != KeyBytes {
		return nil, fmt.Errorf("secret: key must be %d bytes, got %d", KeyBytes, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secret: new cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secret: new gcm: %w", err)
	}
	return &Sealer{aead: aead}, nil
}

// GenerateKey returns a fresh base64 key, for operators setting this up.
func GenerateKey() (string, error) {
	key := make([]byte, KeyBytes)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// Seal returns base64(nonce || ciphertext). Empty input stays empty so callers
// can store "no credential" without a special case.
func (s *Sealer) Seal(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("secret: nonce: %w", err)
	}

	sealed := s.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(sealed), nil
}

func (s *Sealer) Open(encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("secret: stored value is not valid base64: %w", err)
	}
	if len(raw) < s.aead.NonceSize() {
		return "", errors.New("secret: stored value is too short to be valid")
	}

	nonce, ciphertext := raw[:s.aead.NonceSize()], raw[s.aead.NonceSize():]
	plaintext, err := s.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("secret: could not decrypt — wrong key, or the value was tampered with")
	}
	return string(plaintext), nil
}
