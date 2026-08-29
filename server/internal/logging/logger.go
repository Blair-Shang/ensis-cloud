package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type Options struct {
	Dir        string
	Level      string
	ToConsole  bool
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
}

func New(opt Options) (*zap.Logger, error) {
	if strings.TrimSpace(opt.Dir) == "" {
		opt.Dir = "./logs"
	}
	if err := os.MkdirAll(opt.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir log dir: %w", err)
	}
	if opt.MaxSizeMB <= 0 {
		opt.MaxSizeMB = 100
	}
	if opt.MaxBackups <= 0 {
		opt.MaxBackups = 10
	}
	if opt.MaxAgeDays <= 0 {
		opt.MaxAgeDays = 30
	}

	level := zap.InfoLevel
	switch strings.ToLower(strings.TrimSpace(opt.Level)) {
	case "debug":
		level = zap.DebugLevel
	case "warn", "warning":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	case "info", "":
		level = zap.InfoLevel
	default:
		return nil, fmt.Errorf("unknown log level %q", opt.Level)
	}

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "ts"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.CallerKey = "caller"
	encCfg.EncodeCaller = zapcore.ShortCallerEncoder

	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(opt.Dir, "niuma-cloud.log"),
		MaxSize:    opt.MaxSizeMB,
		MaxBackups: opt.MaxBackups,
		MaxAge:     opt.MaxAgeDays,
		Compress:   true,
		LocalTime:  true,
	})

	cores := []zapcore.Core{
		zapcore.NewCore(zapcore.NewJSONEncoder(encCfg), fileWriter, level),
	}
	if opt.ToConsole {
		consoleEnc := zap.NewDevelopmentEncoderConfig()
		consoleEnc.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cores = append(cores, zapcore.NewCore(
			zapcore.NewConsoleEncoder(consoleEnc),
			zapcore.AddSync(os.Stdout),
			level,
		))
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller()), nil
}
