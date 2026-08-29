package feedbackmedia

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Blair-Shang/niuma-cloud/server/internal/auth"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
)

const (
	MaxImages     = 3
	MaxImageBytes = 2 << 20 // 单张 2MB
)

var allowedTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/jpg":  ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/gif":  ".gif",
}

type ImageInput struct {
	Name        string `json:"name"`
	ContentType string `json:"contentType"`
	Data        string `json:"data"` // raw base64 或 data URL
}

func Dir(dataDir, feedbackID string) string {
	return filepath.Join(dataDir, "feedback", feedbackID)
}

func FilePath(dataDir, feedbackID, fileName string) string {
	return filepath.Join(Dir(dataDir, feedbackID), filepath.Base(fileName))
}

// SaveImages 校验并写入截图，返回附件元数据；失败时清理目录。
func SaveImages(dataDir, feedbackID string, images []ImageInput) ([]store.FeedbackAttachment, error) {
	if len(images) == 0 {
		return nil, nil
	}
	if len(images) > MaxImages {
		return nil, fmt.Errorf("too_many_images")
	}
	dir := Dir(dataDir, feedbackID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	out := make([]store.FeedbackAttachment, 0, len(images))
	for _, img := range images {
		att, err := materializeOne(dataDir, feedbackID, img)
		if err != nil {
			_ = os.RemoveAll(dir)
			return nil, err
		}
		out = append(out, *att)
	}
	return out, nil
}

func materializeOne(dataDir, feedbackID string, img ImageInput) (*store.FeedbackAttachment, error) {
	raw, detected, err := decodeImageData(img.Data)
	if err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("invalid_image")
	}
	if int64(len(raw)) > MaxImageBytes {
		return nil, fmt.Errorf("image_too_large")
	}

	ct := strings.ToLower(strings.TrimSpace(img.ContentType))
	if ct == "" {
		ct = detected
	}
	if ct == "image/jpg" {
		ct = "image/jpeg"
	}
	if detected != "" && detected != ct {
		if _, ok := allowedTypes[detected]; ok {
			ct = detected
		}
	}
	ext, ok := allowedTypes[ct]
	if !ok {
		return nil, fmt.Errorf("unsupported_image_type")
	}

	id, err := auth.NewID("att")
	if err != nil {
		return nil, err
	}
	fileName := id + ext
	if err := os.WriteFile(FilePath(dataDir, feedbackID, fileName), raw, 0o644); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(img.Name)
	if name == "" {
		name = fileName
	}
	return &store.FeedbackAttachment{
		ID:          id,
		Name:        name,
		ContentType: ct,
		Size:        int64(len(raw)),
		File:        fileName,
	}, nil
}

func decodeImageData(data string) (raw []byte, contentType string, err error) {
	data = strings.TrimSpace(data)
	if data == "" {
		return nil, "", fmt.Errorf("empty")
	}
	if strings.HasPrefix(data, "data:") {
		comma := strings.IndexByte(data, ',')
		if comma < 0 {
			return nil, "", fmt.Errorf("bad data url")
		}
		meta := data[5:comma]
		payload := data[comma+1:]
		if i := strings.Index(meta, ";"); i >= 0 {
			contentType = strings.ToLower(strings.TrimSpace(meta[:i]))
		} else {
			contentType = strings.ToLower(strings.TrimSpace(meta))
		}
		raw, err = base64.StdEncoding.DecodeString(payload)
		return raw, contentType, err
	}
	raw, err = base64.StdEncoding.DecodeString(data)
	return raw, "", err
}

func MarshalAttachments(atts []store.FeedbackAttachment) (string, error) {
	if atts == nil {
		atts = []store.FeedbackAttachment{}
	}
	b, err := json.Marshal(atts)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func ParseAttachments(raw string) []store.FeedbackAttachment {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil
	}
	var atts []store.FeedbackAttachment
	if err := json.Unmarshal([]byte(raw), &atts); err != nil {
		return nil
	}
	return atts
}

func FindAttachment(atts []store.FeedbackAttachment, attID string) *store.FeedbackAttachment {
	attID = strings.TrimSpace(attID)
	for i := range atts {
		if atts[i].ID == attID {
			return &atts[i]
		}
	}
	return nil
}

func RemoveDir(dataDir, feedbackID string) {
	_ = os.RemoveAll(Dir(dataDir, feedbackID))
}
