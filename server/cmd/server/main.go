package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/niuma-cloud/server/internal/config"
	"github.com/Blair-Shang/niuma-cloud/server/internal/db"
	"github.com/Blair-Shang/niuma-cloud/server/internal/httpapi"
	"github.com/Blair-Shang/niuma-cloud/server/internal/logging"
	"github.com/Blair-Shang/niuma-cloud/server/internal/mail"
	"github.com/Blair-Shang/niuma-cloud/server/internal/store"
	"github.com/Blair-Shang/niuma-cloud/server/internal/web"
)

// version 由打包脚本 -ldflags "-X main.version=x.y.z" 注入；本地构建为 dev。
var version = "dev"

func main() {
	if wantsVersion(os.Args[1:]) {
		fmt.Println(version)
		return
	}

	chdirRepoRoot()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	logger, err := logging.New(logging.Options{
		Dir:        cfg.LogDir,
		Level:      cfg.LogLevel,
		ToConsole:  cfg.LogToConsole,
		MaxSizeMB:  cfg.LogMaxSizeMB,
		MaxBackups: cfg.LogMaxBackups,
		MaxAgeDays: cfg.LogMaxAgeDays,
	})
	if err != nil {
		panic(err)
	}
	defer func() { _ = logger.Sync() }()

	sqlDB, err := db.Open(cfg.DBDriver, cfg.DBDSN)
	if err != nil {
		logger.Fatal("open db", zap.Error(err))
	}
	defer sqlDB.Close()

	st := store.New(sqlDB, cfg.DBDriver)
	mailer := mail.New(cfg, logger)
	api := httpapi.New(cfg, st, mailer, logger)

	if err := api.SeedStaff(context.Background()); err != nil {
		logger.Fatal("seed staff", zap.Error(err))
	}

	static, err := web.Handler()
	if err != nil {
		logger.Fatal("embed web", zap.Error(err))
	}
	api.SetStatic(static)

	// 前后端交互超时来自 app.yaml http_timeout_min（默认 30 分钟）
	reqTimeout := cfg.HTTPTimeout
	if reqTimeout <= 0 {
		reqTimeout = 30 * time.Minute
	}
	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       reqTimeout,
		WriteTimeout:      reqTimeout,
		IdleTimeout:       reqTimeout,
	}

	go func() {
		logger.Info("server listening",
			zap.String("version", version),
			zap.String("addr", cfg.HTTPAddr),
			zap.String("db", cfg.DBDriver),
			zap.Duration("http_timeout", reqTimeout),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen", zap.Error(err))
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Warn("shutdown", zap.Error(err))
	} else {
		logger.Info("shutdown complete")
	}
}

func wantsVersion(args []string) bool {
	for _, a := range args {
		if a == "--version" || a == "-version" || a == "-V" {
			return true
		}
	}
	return false
}

func chdirRepoRoot() {
	if _, err := os.Stat("server/go.mod"); err == nil {
		return
	}
	if _, err := os.Stat("go.mod"); err == nil {
		_ = os.Chdir("..")
	}
}
