// Package logger 基于 zap 提供结构化日志。
package logger

import (
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/pichub/pichub/internal/config"
)

// Field 是日志字段别名。
type Field = zap.Field

// Logger 是应用日志器，支持链式字段。
type Logger struct {
	z         *zap.Logger
	atomicLvl zap.AtomicLevel
}

// New 根据配置创建日志器。文件输出目录不存在时自动创建。
func New(cfg *config.LogConfig) *Logger {
	lvl := zap.NewAtomicLevelAt(parseLevel(cfg.Level))

	encCfg := zap.NewProductionEncoderConfig()
	encCfg.TimeKey = "time"
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encCfg.EncodeLevel = zapcore.LowercaseLevelEncoder

	var encoder zapcore.Encoder
	if cfg.Format == "console" {
		if cfg.Color {
			encCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
		encoder = zapcore.NewConsoleEncoder(encCfg)
	} else {
		encoder = zapcore.NewJSONEncoder(encCfg)
	}

	var cores []zapcore.Core
	if cfg.Output == "stdout" || cfg.Output == "both" || cfg.Output == "" {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), lvl))
	}
	if (cfg.Output == "file" || cfg.Output == "both") && cfg.FilePath != "" {
		_ = os.MkdirAll(filepath.Dir(cfg.FilePath), 0o755)
		writer := &lumberjack.Logger{
			Filename:   cfg.FilePath,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(writer), lvl))
	}
	if len(cores) == 0 {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), lvl))
	}

	opts := []zap.Option{zap.AddStacktrace(zapcore.ErrorLevel)}
	if cfg.Caller {
		opts = append(opts, zap.AddCaller(), zap.AddCallerSkip(0))
	}
	return &Logger{z: zap.New(zapcore.NewTee(cores...), opts...), atomicLvl: lvl}
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// SetLevel 动态调整日志级别。
func (l *Logger) SetLevel(s string) { l.atomicLvl.SetLevel(parseLevel(s)) }

// Zap 返回底层 zap 实例。
func (l *Logger) Zap() *zap.Logger { return l.z }

// With 附加字段返回新日志器。
func (l *Logger) With(fields ...Field) *Logger {
	return &Logger{z: l.z.With(fields...), atomicLvl: l.atomicLvl}
}

// Debug 输出 debug 日志。
func (l *Logger) Debug(msg string, fields ...Field) { l.z.Debug(msg, fields...) }

// Info 输出 info 日志。
func (l *Logger) Info(msg string, fields ...Field) { l.z.Info(msg, fields...) }

// Warn 输出 warn 日志。
func (l *Logger) Warn(msg string, fields ...Field) { l.z.Warn(msg, fields...) }

// Error 输出 error 日志。
func (l *Logger) Error(msg string, fields ...Field) { l.z.Error(msg, fields...) }

// Fatal 输出 fatal 日志并退出。
func (l *Logger) Fatal(msg string, fields ...Field) { l.z.Fatal(msg, fields...) }

// Sync 刷新缓冲。
func (l *Logger) Sync() { _ = l.z.Sync() }

// String 构造字符串字段。
func String(key, val string) Field { return zap.String(key, val) }

// Int 构造整型字段。
func Int(key string, val int) Field { return zap.Int(key, val) }

// Int64 构造整型字段。
func Int64(key string, val int64) Field { return zap.Int64(key, val) }

// Uint64 构造无符号整型字段。
func Uint64(key string, val uint64) Field { return zap.Uint64(key, val) }

// Bool 构造布尔字段。
func Bool(key string, val bool) Field { return zap.Bool(key, val) }

// Duration 构造时长字段。
func Duration(key string, val time.Duration) Field { return zap.Duration(key, val) }

// Err 构造错误字段。
func Err(err error) Field { return zap.Error(err) }

// Any 构造任意字段。
func Any(key string, val any) Field { return zap.Any(key, val) }
