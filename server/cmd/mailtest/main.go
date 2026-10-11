package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/Blair-Shang/ensis-cloud/server/internal/config"
	"github.com/Blair-Shang/ensis-cloud/server/internal/mail"
)

// 用法：
//
//	go run ./cmd/mailtest
//	go run ./cmd/mailtest shangjian@flux.com.cn
//	go run ./cmd/mailtest --otp register shangjian@flux.com.cn
//	go run ./cmd/mailtest --otp login shangjian@flux.com.cn
func main() {
	chdirRepoRoot()

	modeOTP := false
	purpose := "register"
	to := "shangjian@flux.com.cn"
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--otp":
			modeOTP = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				purpose = strings.TrimSpace(args[i])
			}
		case strings.HasPrefix(a, "-"):
			fatal("unknown flag: %s", a)
		default:
			to = strings.TrimSpace(a)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fatal("load config: %v", err)
	}
	logger, err := zap.NewDevelopment()
	if err != nil {
		fatal("logger: %v", err)
	}
	defer func() { _ = logger.Sync() }()

	m := mail.New(cfg, logger)
	logger.Info("sending test mail",
		zap.String("to", to),
		zap.String("from", cfg.SMTPFrom),
		zap.String("host", cfg.SMTPHost),
		zap.Int("port", cfg.SMTPPort),
		zap.Bool("otp_template", modeOTP),
		zap.String("purpose", purpose),
	)

	if modeOTP {
		if err := m.SendOTPPreview(to, purpose, "834291"); err != nil {
			fatal("send failed: %v", err)
		}
	} else {
		subject := "Ensis SMTP 测试"
		body := fmt.Sprintf(
			"这是一封来自 ensis-cloud 的测试邮件。\n\n时间：%s\n发件：%s\nSMTP：%s:%d\n",
			time.Now().Format(time.RFC3339),
			cfg.SMTPFrom,
			cfg.SMTPHost,
			cfg.SMTPPort,
		)
		if err := m.SendTest(to, subject, body); err != nil {
			fatal("send failed: %v", err)
		}
	}
	logger.Info("send ok", zap.String("to", to))
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

func chdirRepoRoot() {
	if _, err := os.Stat("server/go.mod"); err == nil {
		return
	}
	if _, err := os.Stat("go.mod"); err == nil {
		_ = os.Chdir("..")
	}
}
