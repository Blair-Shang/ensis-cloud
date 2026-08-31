package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const (
	// MaxListModels 是一次从上游收下的模型条数上限。
	MaxListModels = 300
	// MaxListModelsBodyBytes 是 /models 响应体上限。
	MaxListModelsBodyBytes = 512 << 10
)

// RemoteModel 是上游模型目录中的一条。
type RemoteModel struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ListModelsResult 是拉取上游模型目录的结果。
type ListModelsResult struct {
	OK      bool
	Status  int
	Latency time.Duration
	Message string
	Models  []RemoteModel
}

// ModelsURL 把上游 Base URL 拼成 OpenAI 兼容的 /models。
func ModelsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return ""
	}
	if strings.HasSuffix(base, "/models") && !strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	base = strings.TrimSuffix(base, "/chat/completions")
	base = strings.TrimRight(base, "/")
	if strings.HasSuffix(base, "/v1") {
		return base + "/models"
	}
	return base + "/v1/models"
}

// ListModels 请求上游 GET /models，解析 OpenAI 兼容目录。
func ListModels(ctx context.Context, upstream Upstream, client *http.Client) ListModelsResult {
	endpoint := ModelsURL(upstream.BaseURL)
	if endpoint == "" {
		return ListModelsResult{Message: "upstream not configured"}
	}
	if client == nil {
		client = &http.Client{Timeout: ProbeTimeout}
	}
	ctx, cancel := context.WithTimeout(ctx, ProbeTimeout)
	defer cancel()

	started := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return ListModelsResult{Latency: time.Since(started), Message: SanitizeDialError(err)}
	}
	req.Header.Set("Accept", "application/json")
	if key := strings.TrimSpace(upstream.APIKey); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	resp, err := client.Do(req)
	latency := time.Since(started)
	if err != nil {
		return ListModelsResult{Latency: latency, Message: SanitizeDialError(err)}
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, MaxListModelsBodyBytes+1))
	if len(raw) > MaxListModelsBodyBytes {
		raw = raw[:MaxListModelsBodyBytes]
	}
	out := ListModelsResult{
		Status:  resp.StatusCode,
		Latency: latency,
		Message: probeMessage(resp.StatusCode, raw),
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out
	}
	models := parseRemoteModels(raw)
	if len(models) == 0 {
		out.Message = "no models"
		return out
	}
	out.OK = true
	out.Models = models
	if strings.TrimSpace(out.Message) == "" {
		out.Message = "ok"
	}
	return out
}

func parseRemoteModels(raw []byte) []RemoteModel {
	var body struct {
		Data   []json.RawMessage `json:"data"`
		Models []json.RawMessage `json:"models"`
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil
	}
	chunks := body.Data
	if len(chunks) == 0 {
		chunks = body.Models
	}
	seen := make(map[string]struct{}, len(chunks))
	out := make([]RemoteModel, 0, len(chunks))
	for _, chunk := range chunks {
		id, label := remoteModelID(chunk)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if label == "" {
			label = id
		}
		out = append(out, RemoteModel{ID: id, Label: label})
		if len(out) >= MaxListModels {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ID) < strings.ToLower(out[j].ID)
	})
	return out
}

func remoteModelID(raw json.RawMessage) (id, label string) {
	var obj struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Model string `json:"model"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		id = firstNonEmpty(obj.ID, obj.Name, obj.Model)
		label = firstNonEmpty(obj.Name, obj.ID, obj.Model)
		if strings.HasPrefix(id, "models/") {
			id = strings.TrimPrefix(id, "models/")
		}
		return strings.TrimSpace(id), strings.TrimSpace(label)
	}
	var text string
	if err := json.Unmarshal(raw, &text); err == nil {
		text = strings.TrimSpace(text)
		return text, text
	}
	return "", ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
