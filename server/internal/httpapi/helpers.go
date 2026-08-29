package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]string{"error": code})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	return decodeJSONLimit(w, r, dst, 64<<10)
}

func decodeJSONLimit(w http.ResponseWriter, r *http.Request, dst any, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = 64 << 10
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func (s *Server) requestIP(r *http.Request) string {
	return requestIP(r, s != nil && s.cfg.TrustProxy)
}

// requestIP 在 trustProxy 时优先 X-Real-IP，否则取 X-Forwarded-For 末跳（反代追加的真实客户端）。
// 不信任反代时只用 TCP RemoteAddr，避免客户端伪造 XFF 绕过限流。
func requestIP(r *http.Request, trustProxy bool) string {
	if trustProxy {
		if rip := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(rip) != nil {
			return rip
		}
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			last := strings.TrimSpace(parts[len(parts)-1])
			if net.ParseIP(last) != nil {
				return last
			}
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:8])
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const p = "Bearer "
	if !strings.HasPrefix(h, p) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, p))
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validPassword(pw string) bool {
	n := utf8.RuneCountInString(pw)
	if n < 10 || n > 128 {
		return false
	}
	hasLetter, hasDigit := false, false
	for _, r := range pw {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if hasLetter && hasDigit {
			return true
		}
	}
	return false
}
