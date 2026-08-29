package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempAppYAML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "app.yaml"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_DIR", dir)
	return dir
}

func TestLoadRejectsExampleJWT(t *testing.T) {
	writeTempAppYAML(t, `
http_addr: "127.0.0.1:8090"
db:
  driver: sqlite
  dsn: cloud.db
jwt:
  secret: "dev-niuma-cloud-jwt-secret-change-me"
`)
	t.Setenv("NIUMA_ALLOW_INSECURE_DEV", "")
	t.Setenv("NIUMA_JWT_SECRET", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected example jwt.secret to be rejected")
	}
}

func TestLoadEnvOverridesSecrets(t *testing.T) {
	dir := writeTempAppYAML(t, `
http_addr: "127.0.0.1:8090"
http_timeout_min: 30
db:
  driver: sqlite
  dsn: from-file.db
jwt:
  secret: "dev-niuma-cloud-jwt-secret-change-me"
smtp:
  password: file-smtp
  log_otp: true
pipeline:
  token: "file-pipeline-token"
`)
	t.Setenv("NIUMA_JWT_SECRET", "env-jwt-secret-at-least-16")
	t.Setenv("NIUMA_DB_DSN", filepath.Join(dir, "from-env.db"))
	t.Setenv("NIUMA_SMTP_PASSWORD", "env-smtp")
	t.Setenv("NIUMA_PIPELINE_TOKEN", "env-pipeline-token-16")
	t.Setenv("NIUMA_ALLOW_INSECURE_DEV", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.JWTSecret != "env-jwt-secret-at-least-16" {
		t.Fatalf("jwt: %s", cfg.JWTSecret)
	}
	if !strings.HasSuffix(cfg.DBDSN, "from-env.db") {
		t.Fatalf("dsn: %s", cfg.DBDSN)
	}
	if cfg.SMTPPassword != "env-smtp" {
		t.Fatalf("smtp: %s", cfg.SMTPPassword)
	}
	if cfg.PipelineToken != "env-pipeline-token-16" {
		t.Fatalf("pipeline: %s", cfg.PipelineToken)
	}
	if !cfg.MailLogOTP {
		t.Fatal("log_otp should follow yaml true")
	}
	if cfg.HTTPTimeout.Minutes() != 30 {
		t.Fatalf("timeout: %s", cfg.HTTPTimeout)
	}
}

func TestLoadDefaultLogOTPOff(t *testing.T) {
	writeTempAppYAML(t, `
http_addr: "127.0.0.1:8090"
db:
  driver: sqlite
  dsn: cloud.db
jwt:
  secret: "a-sufficiently-long-random-secret"
`)
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MailLogOTP {
		t.Fatal("log_otp should default to false")
	}
	if cfg.HTTPAddr != "127.0.0.1:8090" {
		t.Fatalf("addr: %s", cfg.HTTPAddr)
	}
}
