package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// DefaultChatTimeout 是单次对话转发超时。
	DefaultChatTimeout = 5 * time.Minute
	// MaxChatBodyBytes 是客户端请求体上限。
	MaxChatBodyBytes = 2 << 20
	// MaxChatCompletionTokens 限制单次补全 token（max_tokens / max_completion_tokens）。
	MaxChatCompletionTokens = 8192
	// MaxChatCompletionN 限制一次请求的补全条数。
	MaxChatCompletionN = 1
)

// Upstream 描述 OpenAI 兼容上游。
type Upstream struct {
	BaseURL string
	APIKey  string
}

// ChatCompletionsURL 把上游 Base URL 拼成 /chat/completions。
func ChatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}

// RewriteChatBody 强制使用目录内的模型，去掉客户端伪造的密钥字段。
func RewriteChatBody(raw []byte, model string) ([]byte, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, fmt.Errorf("empty body")
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, fmt.Errorf("invalid json")
	}
	delete(payload, "api_key")
	delete(payload, "apiKey")
	delete(payload, "authorization")
	delete(payload, "Authorization")
	if strings.TrimSpace(model) != "" {
		payload["model"] = model
	}
	clampChatNumber(payload, "max_tokens", MaxChatCompletionTokens)
	clampChatNumber(payload, "max_completion_tokens", MaxChatCompletionTokens)
	clampChatNumber(payload, "n", MaxChatCompletionN)
	out, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal body")
	}
	return out, nil
}

// PickModel 仅允许目录内模型；空或不在名单则回落默认模型。
func PickModel(requested string, allowed []string, fallback string) string {
	req := strings.TrimSpace(requested)
	if req != "" {
		for _, m := range allowed {
			if strings.TrimSpace(m) == req {
				return req
			}
		}
	}
	fb := strings.TrimSpace(fallback)
	if fb != "" {
		return fb
	}
	if len(allowed) > 0 {
		return strings.TrimSpace(allowed[0])
	}
	return ""
}

func clampChatNumber(payload map[string]any, key string, max float64) {
	v, ok := payload[key]
	if !ok || max <= 0 {
		return
	}
	n, ok := jsonNumber(v)
	if !ok {
		delete(payload, key)
		return
	}
	if n > max {
		n = max
	}
	if n < 1 {
		n = 1
	}
	payload[key] = n
}

func jsonNumber(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

// RequestedModel 从 OpenAI chat 请求体取出 model。
func RequestedModel(raw []byte) string {
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return strings.TrimSpace(payload.Model)
}

// ProxyChat 把改写后的请求转到上游，并把响应原样写回（含 SSE）。
func ProxyChat(ctx context.Context, w http.ResponseWriter, upstream Upstream, body []byte, client *http.Client) error {
	endpoint := ChatCompletionsURL(upstream.BaseURL)
	if endpoint == "" {
		return fmt.Errorf("upstream not configured")
	}
	if client == nil {
		client = &http.Client{Timeout: DefaultChatTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	if key := strings.TrimSpace(upstream.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	if cc := resp.Header.Get("Cache-Control"); cc != "" {
		w.Header().Set("Cache-Control", cc)
	} else {
		w.Header().Set("Cache-Control", "no-store")
	}
	w.WriteHeader(resp.StatusCode)

	flusher, _ := w.(http.Flusher)
	buf := make([]byte, 32*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				return nil
			}
			return readErr
		}
	}
}
