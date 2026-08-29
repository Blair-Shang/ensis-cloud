package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func NewID(prefix string) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("id: %w", err)
	}
	return prefix + "_" + hex.EncodeToString(b), nil
}
