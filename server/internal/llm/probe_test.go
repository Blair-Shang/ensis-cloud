package llm

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProbeChatOK(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-up" {
			t.Fatalf("auth: %q", r.Header.Get("Authorization"))
		}
		b, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(b), `"model":"deepseek-chat"`) || strings.Contains(string(b), `"stream":true`) {
			t.Fatalf("body: %s", b)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"ok"}}]}`)
	}))
	t.Cleanup(up.Close)

	got := ProbeChat(t.Context(), Upstream{BaseURL: up.URL + "/v1", APIKey: "sk-up"}, "deepseek-chat", up.Client())
	if !got.OK || got.Status != http.StatusOK {
		t.Fatalf("probe: %+v", got)
	}
}

func TestProbeChatUpstreamError(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"message":"Incorrect API key"}}`)
	}))
	t.Cleanup(up.Close)

	got := ProbeChat(t.Context(), Upstream{BaseURL: up.URL + "/v1", APIKey: "bad"}, "m", up.Client())
	if got.OK {
		t.Fatal("expected failure")
	}
	if got.Status != http.StatusUnauthorized || !strings.Contains(got.Message, "Incorrect API key") {
		t.Fatalf("probe: %+v", got)
	}
}

func TestProbeChatRequiresModel(t *testing.T) {
	got := ProbeChat(t.Context(), Upstream{BaseURL: "https://api.example.com/v1"}, "  ", nil)
	if got.OK || got.Message != "model required" {
		t.Fatalf("probe: %+v", got)
	}
}
