package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

const APIKeyPrefix = "nandi_"

func RandomHex(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := io.ReadFull(rand.Reader, buf); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func SHA256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// NewAPIKey returns a high-entropy key, the public prefix, and the hash to persist.
func NewAPIKey() (raw, prefix, hash string, err error) {
	secret, err := RandomHex(32)
	if err != nil {
		return "", "", "", err
	}
	raw = APIKeyPrefix + secret
	prefix = raw[:12]
	hash = SHA256Hex(raw)
	return raw, prefix, hash, nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
