package llm

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelsURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://api.deepseek.com/v1", "https://api.deepseek.com/v1/models"},
		{"https://api.openai.com/v1/", "https://api.openai.com/v1/models"},
		{"https://api.openai.com/v1/chat/completions", "https://api.openai.com/v1/models"},
		{"https://api.example.com/v1/models", "https://api.example.com/v1/models"},
		{"https://api.example.com", "https://api.example.com/v1/models"},
		{"", ""},
	}
	for _, c := range cases {
		if got := ModelsURL(c.in); got != c.want {
			t.Fatalf("ModelsURL(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestListModelsOK(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" {
			t.Fatalf("req %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk-up" {
			t.Fatalf("auth: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"data":[{"id":"fast"},{"id":"chat","name":"Chat"},{"id":"fast"}]}`)
	}))
	t.Cleanup(up.Close)

	got := ListModels(t.Context(), Upstream{BaseURL: up.URL + "/v1", APIKey: "sk-up"}, up.Client())
	if !got.OK || len(got.Models) != 2 {
		t.Fatalf("list: %+v", got)
	}
	if got.Models[0].ID != "chat" || got.Models[0].Label != "Chat" {
		t.Fatalf("first: %+v", got.Models[0])
	}
	if got.Models[1].ID != "fast" {
		t.Fatalf("second: %+v", got.Models[1])
	}
}

func TestListModelsGeminiName(t *testing.T) {
	raw := []byte(`{"models":[{"name":"models/gemini-2.0-flash"}]}`)
	got := parseRemoteModels(raw)
	if len(got) != 1 || got[0].ID != "gemini-2.0-flash" {
		t.Fatalf("parse: %+v", got)
	}
}

func TestListModelsUpstreamError(t *testing.T) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":{"message":"Incorrect API key"}}`)
	}))
	t.Cleanup(up.Close)

	got := ListModels(t.Context(), Upstream{BaseURL: up.URL + "/v1", APIKey: "bad"}, up.Client())
	if got.OK || got.Status != http.StatusUnauthorized {
		t.Fatalf("list: %+v", got)
	}
}
