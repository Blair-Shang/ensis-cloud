package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatCompletionsURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"https://api.example.com/v1", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com/v1/", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com/v1/chat/completions", "https://api.example.com/v1/chat/completions"},
		{"https://api.example.com", "https://api.example.com/v1/chat/completions"},
	}
	for _, c := range cases {
		if got := ChatCompletionsURL(c.in); got != c.want {
			t.Fatalf("ChatCompletionsURL(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestPickModel(t *testing.T) {
	allowed := []string{"niuma-fast", "niuma-pro"}
	if got := PickModel("niuma-pro", allowed, "niuma-fast"); got != "niuma-pro" {
		t.Fatalf("allow: %s", got)
	}
	if got := PickModel("gpt-4o", allowed, "niuma-fast"); got != "niuma-fast" {
		t.Fatalf("fallback: %s", got)
	}
	if got := PickModel("", nil, "niuma-fast"); got != "niuma-fast" {
		t.Fatalf("empty allow: %s", got)
	}
	if got := PickModel("gpt-4o", nil, ""); got != "" {
		t.Fatalf("no allow no fallback: %s", got)
	}
}

func TestRewriteChatBody(t *testing.T) {
	raw := []byte(`{"model":"x","messages":[{"role":"user","content":"hi"}],"api_key":"leak","authorization":"sk","n":8,"max_tokens":99999,"stream":true}`)
	out, err := RewriteChatBody(raw, "niuma-fast")
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["model"] != "niuma-fast" {
		t.Fatalf("model: %v", payload["model"])
	}
	if _, ok := payload["api_key"]; ok {
		t.Fatal("api_key should be stripped")
	}
	if _, ok := payload["authorization"]; ok {
		t.Fatal("authorization should be stripped")
	}
	if payload["n"] != float64(MaxChatCompletionN) {
		t.Fatalf("n: %v", payload["n"])
	}
	if payload["max_tokens"] != float64(MaxChatCompletionTokens) {
		t.Fatalf("max_tokens: %v", payload["max_tokens"])
	}
}

func TestProxyChatForwardsAuthAndBody(t *testing.T) {
	var gotAuth, gotBody string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
	}))
	t.Cleanup(up.Close)

	rec := httptest.NewRecorder()
	body := []byte(`{"model":"niuma-fast","stream":true}`)
	if err := ProxyChat(t.Context(), rec, Upstream{BaseURL: up.URL + "/v1", APIKey: "sk-up"}, body, up.Client()); err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer sk-up" {
		t.Fatalf("auth: %q", gotAuth)
	}
	if !strings.Contains(gotBody, `"model":"niuma-fast"`) {
		t.Fatalf("body: %s", gotBody)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status: %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ok") {
		t.Fatalf("resp: %s", rec.Body.String())
	}
}
