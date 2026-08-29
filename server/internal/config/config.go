package config

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	HTTPAddr string
	// HTTPTimeout 前后端单次请求读写超时（ReadTimeout / WriteTimeout）
	HTTPTimeout time.Duration
	// TrustProxy 为 true 时采用 X-Real-IP / X-Forwarded-For 末跳（须在可信反代之后）
	TrustProxy bool

	ConfigDir string
	LogDir    string
	DataDir   string

	DBDriver string
	DBDSN    string

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	OTPTTL          time.Duration
	OTPCooldown     time.Duration
	OTPDailyLimit   int

	CORSOrigins []string

	StaffEmail    string
	StaffPassword string

	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	MailLogOTP   bool

	FeedbackWebhookURL string

	// DownloadURLHosts 发布记录 download_url 允许的主机后缀；空则仅拒绝内网/IP/非 HTTPS。
	DownloadURLHosts []string
	// ReleasesDir 安装包落盘目录，默认 {data_dir}/releases。
	ReleasesDir string
	// PublicBase 对外 API 根（含 /niuma/cloud），用于生成国内下载 URL。
	PublicBase string
	// MaxInstallerBytes 单个安装包上限。
	MaxInstallerBytes int64

	// PipelineToken GitHub Actions 等流水线接收发布的共享密钥；空则关闭该接口。
	PipelineToken string

	LogLevel      string
	LogToConsole  bool
	LogMaxSizeMB  int
	LogMaxBackups int
	LogMaxAgeDays int
}

// fileConfig 对应 config/app.yaml 结构。
type fileConfig struct {
	HTTPAddr       string `yaml:"http_addr"`
	HTTPTimeoutMin int    `yaml:"http_timeout_min"` // 前后端交互超时（分钟），默认 5
	TrustProxy     *bool  `yaml:"trust_proxy"`      // 默认 true；仅当 :8090 直连公网时设 false

	DB struct {
		Driver string `yaml:"driver"`
		DSN    string `yaml:"dsn"`
	} `yaml:"db"`

	JWT struct {
		Secret              string `yaml:"secret"`
		AccessTokenTTLMin   int    `yaml:"access_token_ttl_min"`
		RefreshTokenTTLDays int    `yaml:"refresh_token_ttl_days"`
	} `yaml:"jwt"`

	CORSOrigins []string `yaml:"cors_origins"`

	Staff struct {
		Email    string `yaml:"email"`
		Password string `yaml:"password"`
	} `yaml:"staff"`

	SMTP struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		From     string `yaml:"from"`
		LogOTP   *bool  `yaml:"log_otp"`
	} `yaml:"smtp"`

	OTP struct {
		TTLMin      int `yaml:"ttl_min"`
		CooldownSec int `yaml:"cooldown_sec"`
		DailyLimit  int `yaml:"daily_limit"`
	} `yaml:"otp"`

	Feedback struct {
		WebhookURL string `yaml:"webhook_url"`
	} `yaml:"feedback"`

	Updates struct {
		// DownloadURLHosts 如 ["cdn.niuma007.com","niuma007.com"]；空=不限制公网域名（仍禁内网）
		DownloadURLHosts []string `yaml:"download_url_hosts"`
		PublicBase       string   `yaml:"public_base"`
		Dir              string   `yaml:"dir"`
		MaxFileMB        int      `yaml:"max_file_mb"`
	} `yaml:"updates"`

	Pipeline struct {
		Token string `yaml:"token"`
	} `yaml:"pipeline"`

	Log struct {
		Level      string `yaml:"level"`
		ToConsole  *bool  `yaml:"to_console"`
		Dir        string `yaml:"dir"`
		MaxSizeMB  int    `yaml:"max_size_mb"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAgeDays int    `yaml:"max_age_days"`
	} `yaml:"log"`

	DataDir string `yaml:"data_dir"`
}

func Load() (Config, error) {
	cfgDir := getenv("CONFIG_DIR", "./config")
	if err := ensureRuntimeDirs(cfgDir); err != nil {
		return Config{}, err
	}
	if err := ensureConfigFile(cfgDir); err != nil {
		return Config{}, err
	}

	raw, err := os.ReadFile(filepath.Join(cfgDir, "app.yaml"))
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var fc fileConfig
	if err := yaml.Unmarshal(raw, &fc); err != nil {
		return Config{}, fmt.Errorf("parse app.yaml: %w", err)
	}

	logDir := firstNonEmpty(fc.Log.Dir, "./logs")
	dataDir := firstNonEmpty(fc.DataDir, "./data")
	if err := ensureRuntimeDirs(logDir, dataDir); err != nil {
		return Config{}, err
	}

	mailLogOTP := true
	if fc.SMTP.LogOTP != nil {
		mailLogOTP = *fc.SMTP.LogOTP
	}
	logToConsole := true
	if fc.Log.ToConsole != nil {
		logToConsole = *fc.Log.ToConsole
	}

	trustProxy := true
	if fc.TrustProxy != nil {
		trustProxy = *fc.TrustProxy
	}

	cfg := Config{
		HTTPAddr:           firstNonEmpty(fc.HTTPAddr, ":8090"),
		HTTPTimeout:        time.Duration(intOr(fc.HTTPTimeoutMin, 5)) * time.Minute,
		TrustProxy:         trustProxy,
		ConfigDir:          cfgDir,
		LogDir:             logDir,
		DataDir:            dataDir,
		DBDriver:           strings.ToLower(firstNonEmpty(fc.DB.Driver, "mysql")),
		DBDSN:              strings.TrimSpace(fc.DB.DSN),
		JWTSecret:          strings.TrimSpace(fc.JWT.Secret),
		AccessTokenTTL:     time.Duration(intOr(fc.JWT.AccessTokenTTLMin, 30)) * time.Minute,
		RefreshTokenTTL:    time.Duration(intOr(fc.JWT.RefreshTokenTTLDays, 30)) * 24 * time.Hour,
		OTPTTL:             time.Duration(intOr(fc.OTP.TTLMin, 5)) * time.Minute,
		OTPCooldown:        time.Duration(intOr(fc.OTP.CooldownSec, 60)) * time.Second,
		OTPDailyLimit:      intOr(fc.OTP.DailyLimit, 20),
		StaffEmail:         strings.ToLower(strings.TrimSpace(firstNonEmpty(fc.Staff.Email, "admin@niuma007.com"))),
		StaffPassword:      firstNonEmpty(fc.Staff.Password, "ChangeMeAdmin1!"),
		SMTPHost:           strings.TrimSpace(fc.SMTP.Host),
		SMTPPort:           intOr(fc.SMTP.Port, 587),
		SMTPUser:           strings.TrimSpace(fc.SMTP.User),
		SMTPPassword:       fc.SMTP.Password,
		SMTPFrom:           firstNonEmpty(fc.SMTP.From, "noreply@niuma007.com"),
		MailLogOTP:         mailLogOTP,
		FeedbackWebhookURL: strings.TrimSpace(fc.Feedback.WebhookURL),
		PipelineToken:      firstNonEmpty(getenv("NIUMA_PIPELINE_TOKEN", ""), strings.TrimSpace(fc.Pipeline.Token)),
		ReleasesDir:        firstNonEmpty(fc.Updates.Dir, filepath.Join(dataDir, "releases")),
		PublicBase:         firstNonEmpty(fc.Updates.PublicBase, "https://www.niuma007.com/niuma/cloud"),
		MaxInstallerBytes:  int64(intOr(fc.Updates.MaxFileMB, 2048)) << 20,
		LogLevel:           firstNonEmpty(fc.Log.Level, "info"),
		LogToConsole:       logToConsole,
		LogMaxSizeMB:       intOr(fc.Log.MaxSizeMB, 100),
		LogMaxBackups:      intOr(fc.Log.MaxBackups, 10),
		LogMaxAgeDays:      intOr(fc.Log.MaxAgeDays, 30),
	}
	for _, h := range fc.Updates.DownloadURLHosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h != "" {
			cfg.DownloadURLHosts = append(cfg.DownloadURLHosts, h)
		}
	}

	if cfg.JWTSecret == "" || len(cfg.JWTSecret) < 16 {
		return Config{}, fmt.Errorf("jwt.secret must be at least 16 characters")
	}
	if cfg.PipelineToken != "" && len(cfg.PipelineToken) < 16 {
		return Config{}, fmt.Errorf("pipeline.token must be at least 16 characters")
	}
	if err := ensureRuntimeDirs(cfg.ReleasesDir); err != nil {
		return Config{}, err
	}

	if len(fc.CORSOrigins) == 0 {
		cfg.CORSOrigins = []string{
			"http://localhost:5174",
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"https://www.niuma007.com",
			"https://niuma007.com",
		}
	} else {
		for _, o := range fc.CORSOrigins {
			o = strings.TrimSpace(o)
			if o != "" {
				cfg.CORSOrigins = append(cfg.CORSOrigins, o)
			}
		}
	}

	switch cfg.DBDriver {
	case "mysql":
		// 主存储：MySQL
	case "sqlite":
		// 仅本地临时调试；生产勿用
		if cfg.DBDSN == "" {
			cfg.DBDSN = filepath.Join(dataDir, "cloud.db")
		}
	default:
		return Config{}, fmt.Errorf("unsupported db.driver %q (want mysql|sqlite)", cfg.DBDriver)
	}
	if cfg.DBDSN == "" {
		return Config{}, fmt.Errorf("db.dsn is required (mysql example: user:pass@tcp(host:3306)/niuma_cloud?parseTime=true&charset=utf8mb4&loc=UTC)")
	}
	return cfg, nil
}

func ensureRuntimeDirs(dirs ...string) error {
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}

func ensureConfigFile(cfgDir string) error {
	dst := filepath.Join(cfgDir, "app.yaml")
	if _, err := os.Stat(dst); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	candidates := []string{
		filepath.Join(cfgDir, "app.yaml.example"),
		"config/app.yaml.example",
	}
	var src string
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			src = c
			break
		}
	}
	if src == "" {
		content := defaultYAML()
		return os.WriteFile(dst, []byte(content), 0o644)
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func defaultYAML() string {
	return `http_addr: ":8090"

db:
  driver: mysql
  dsn: "user:pass@tcp(127.0.0.1:3306)/niuma_cloud?parseTime=true&charset=utf8mb4&loc=UTC"

jwt:
  secret: "dev-niuma-cloud-jwt-secret-change-me"
`
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func intOr(v, fallback int) int {
	if v == 0 {
		return fallback
	}
	return v
}
