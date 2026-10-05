package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func RandomTokenHex(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func HashSHA256Hex(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
