package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var distFS embed.FS

// Handler 返回 Ops SPA 静态处理器；调用方负责把 /ops 前缀剥掉后再传入。
func Handler() (http.Handler, error) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		return nil, err
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path != "/" && !hasFileExt(path) {
			rel := strings.TrimPrefix(path, "/")
			if _, err := fs.Stat(sub, rel); err != nil {
				r2 := r.Clone(r.Context())
				r2.URL.Path = "/"
				fileServer.ServeHTTP(w, r2)
				return
			}
		}
		fileServer.ServeHTTP(w, r)
	}), nil
}

func hasFileExt(path string) bool {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return true
		}
	}
	return false
}
