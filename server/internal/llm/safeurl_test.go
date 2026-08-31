package llm

import (
	"net"
	"testing"
)

func TestValidateUpstreamURLRejectsUnsafe(t *testing.T) {
	cases := []string{
		"",
		"http://api.openai.com/v1",
		"https://127.0.0.1/v1",
		"https://localhost/v1",
		"https://10.0.0.8/v1",
		"https://192.168.1.1/v1",
		"https://169.254.169.254/latest/meta-data",
		"https://100.64.0.1/v1",
		"https://user:pass@api.openai.com/v1",
		"https://metadata.google.internal/",
	}
	for _, in := range cases {
		if err := ValidateUpstreamURL(in); err == nil {
			t.Fatalf("expected reject: %s", in)
		}
	}
}

func TestIsPublicIP(t *testing.T) {
	if isPublicIP(net.ParseIP("8.8.8.8")) != true {
		t.Fatal("8.8.8.8")
	}
	if isPublicIP(net.ParseIP("127.0.0.1")) {
		t.Fatal("loopback")
	}
	if isPublicIP(net.ParseIP("100.64.1.1")) {
		t.Fatal("cgnat")
	}
}

func TestSanitizeDialError(t *testing.T) {
	if got := SanitizeDialError(ErrUnsafeUpstream); got != ErrUnsafeUpstream.Error() {
		t.Fatalf("unsafe: %s", got)
	}
	if got := SanitizeDialError(assertError("dial tcp 10.1.2.3:443: connect: connection refused")); got != "upstream unreachable" {
		t.Fatalf("dial: %s", got)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }
