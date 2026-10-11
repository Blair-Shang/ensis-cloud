package httpapi

import (
	"context"
	"net/http"

	"go.uber.org/zap"

	"github.com/Blair-Shang/ensis-cloud/server/internal/auth"
	"github.com/Blair-Shang/ensis-cloud/server/internal/store"
)

// writeAudit 写入审计日志；失败只打警告，不阻断主流程。
func (s *Server) writeAudit(ctx context.Context, r *http.Request, a store.AuditLog) {
	if s.store == nil {
		return
	}
	if a.ID == "" {
		id, err := auth.NewID("aud")
		if err != nil {
			s.log.Warn("audit id", zap.Error(err))
			return
		}
		a.ID = id
	}
	if a.SourceIPHash == "" && r != nil {
		a.SourceIPHash = hashIP(s.requestIP(r))
	}
	if err := s.store.InsertAuditLog(ctx, a); err != nil {
		s.log.Warn("audit insert", zap.Error(err), zap.String("action", a.ActionCode))
	}
}
