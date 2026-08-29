package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(code int) {
	if r.status == 0 {
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format("150405.000")))
	}
	return hex.EncodeToString(b[:])
}

// withAccessLog 记录 API 访问：method/path/status/耗时/request_id（跳过静态资源与 healthz 噪声可按需扩展）。
func (s *Server) withAccessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reqID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if reqID == "" {
			reqID = newRequestID()
		}
		w.Header().Set("X-Request-ID", reqID)

		rec := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}

		path := r.URL.Path
		// 预检 / 健康检查 / Admin SPA 不打访问日志，避免刷屏
		if r.Method == http.MethodOptions {
			return
		}
		if path == OpsPrefix || strings.HasPrefix(path, OpsPrefix+"/") {
			return
		}
		if path == CloudPrefix+"/healthz" {
			return
		}

		level := s.log.Info
		if status >= 500 {
			level = s.log.Error
		} else if status >= 400 {
			level = s.log.Warn
		}
		level("http request",
			zap.String("request_id", reqID),
			zap.String("method", r.Method),
			zap.String("path", path),
			zap.Int("status", status),
			zap.Int("bytes", rec.bytes),
			zap.Duration("duration", time.Since(start)),
			zap.String("ip", s.requestIP(r)),
			zap.String("user_agent", truncateUA(r.UserAgent())),
		)
	})
}

func truncateUA(ua string) string {
	ua = strings.TrimSpace(ua)
	if utf8Len := len([]rune(ua)); utf8Len > 120 {
		return string([]rune(ua)[:120])
	}
	return ua
}
