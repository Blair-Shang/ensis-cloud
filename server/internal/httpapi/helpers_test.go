package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestValidPassword(t *testing.T) {
	ok := []string{"ChangeMeAdmin1!", "abcdefghij1"}
	for _, pw := range ok {
		if !validPassword(pw) {
			t.Fatalf("want valid: %q", pw)
		}
	}
	bad := []string{"", "short1A", "abcdefghij", "1234567890", "OnlyLetters"}
	for _, pw := range bad {
		if validPassword(pw) {
			t.Fatalf("want invalid: %q", pw)
		}
	}
}

func TestRequestIPTrustProxy(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "10.0.0.8:1234"
	r.Header.Set("X-Forwarded-For", "1.2.3.4, 203.0.113.9")
	r.Header.Set("X-Real-IP", "198.51.100.7")

	if got := requestIP(r, false); got != "10.0.0.8" {
		t.Fatalf("no trust: got %s", got)
	}
	if got := requestIP(r, true); got != "198.51.100.7" {
		t.Fatalf("real-ip: got %s", got)
	}
	r.Header.Del("X-Real-IP")
	if got := requestIP(r, true); got != "203.0.113.9" {
		t.Fatalf("xff last: got %s", got)
	}
}
