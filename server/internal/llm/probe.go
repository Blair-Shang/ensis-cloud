package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	// ProbeTimeout 是 Ops 探测上游的超时。
	ProbeTimeout = 20 * time.Second
	// MaxProbeBodyBytes 是探测响应体上限。
	MaxProbeBodyBytes = 8 << 10
)

// ProbeResult 是上游连通性探测结果。
type ProbeResult struct {
	OK      bool
	Status  int
	Latency time.Duration
	Message string
	Model   string
}

type probeErrorBody struct {
	Error json.RawMessage `json:"error"`
}

// ProbeChat 向 OpenAI 兼容上游发一条极短非流式对话，用于 Ops 测试连接。
func ProbeChat(ctx context.Context, upstream Upstream, model string, client *http.Client) ProbeResult {
	endpoint := ChatCompletionsURL(upstream.BaseURL)
	if endpoint == "" {
		return ProbeResult{Message: "upstream not configured"}
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return ProbeResult{Message: "model required"}
	}
	payload, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": "ping"},
		},
		"max_tokens": 1,
		"stream":     false,
	})
	if err != nil {
		return ProbeResult{Message: "marshal body", Model: model}
	}

	if client == nil {
		client = &http.Client{Timeout: ProbeTimeout}
	}
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()

	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return ProbeResult{Latency: time.Since(started), Message: SanitizeDialError(err), Model: model}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if key := strings.TrimSpace(upstream.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := client.Do(req)
	latency := time.Since(started)
	if err != nil {
		return ProbeResult{Latency: latency, Message: SanitizeDialError(err), Model: model}
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, MaxProbeBodyBytes+1))
	if len(raw) > MaxProbeBodyBytes {
		raw = raw[:MaxProbeBodyBytes]
	}
	out := ProbeResult{
		Status:  resp.StatusCode,
		Latency: latency,
		Message: probeMessage(resp.StatusCode, raw),
		Model:   model,
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		out.OK = true
		if strings.TrimSpace(out.Message) == "" {
			out.Message = "ok"
		}
	}
	return out
}

func probeMessage(status int, raw []byte) string {
	if msg := extractOpenAIError(raw); msg != "" {
		return msg
	}
	if status >= 200 && status < 300 {
		return "ok"
	}
	if status > 0 {
		return http.StatusText(status)
	}
	return "upstream error"
}

func extractOpenAIError(raw []byte) string {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return ""
	}
	var body probeErrorBody
	if err := json.Unmarshal(raw, &body); err != nil || len(body.Error) == 0 {
		return ""
	}
	var obj struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body.Error, &obj); err == nil && strings.TrimSpace(obj.Message) != "" {
		return strings.TrimSpace(obj.Message)
	}
	var text string
	if err := json.Unmarshal(body.Error, &text); err == nil {
		return strings.TrimSpace(text)
	}
	return ""
}
