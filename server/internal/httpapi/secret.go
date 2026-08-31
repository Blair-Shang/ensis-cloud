package httpapi

import (
	"strings"

	"github.com/Blair-Shang/niuma-cloud/server/internal/cryptutil"
)

func (s *Server) cryptoMaster() string {
	if s != nil && strings.TrimSpace(s.cfg.CryptoSecret) != "" {
		return s.cfg.CryptoSecret
	}
	return s.cfg.JWTSecret
}

func (s *Server) sealSecret(plain string) (string, error) {
	return cryptutil.Seal(s.cryptoMaster(), plain)
}

func (s *Server) openSecret(cipher string) (string, error) {
	if alt := strings.TrimSpace(s.cfg.CryptoSecret); alt != "" && alt != s.cfg.JWTSecret {
		if plain, err := cryptutil.Open(alt, cipher); err == nil {
			return plain, nil
		}
	}
	return cryptutil.Open(s.cfg.JWTSecret, cipher)
}
