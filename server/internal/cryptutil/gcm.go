package cryptutil

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

// Seal 用 master 派生 AES-256-GCM 加密明文，返回 base64 密文。
func Seal(master, plaintext string) (string, error) {
	if strings.TrimSpace(plaintext) == "" {
		return "", nil
	}
	if strings.TrimSpace(master) == "" {
		return "", fmt.Errorf("empty master")
	}
	block, err := aes.NewCipher(deriveKey(master))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Open 解密 Seal 产出的密文。
func Open(master, cipherText string) (string, error) {
	raw := strings.TrimSpace(cipherText)
	if raw == "" {
		return "", nil
	}
	if strings.TrimSpace(master) == "" {
		return "", fmt.Errorf("empty master")
	}
	bin, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(deriveKey(master))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(bin) < gcm.NonceSize() {
		return "", fmt.Errorf("cipher too short")
	}
	nonce, payload := bin[:gcm.NonceSize()], bin[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, payload, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func deriveKey(master string) []byte {
	sum := sha256.Sum256([]byte(master))
	return sum[:]
}
