// Package logger 封装基于 zap 的日志系统。
//
// 文件输出规则：
//   - 目录：cfg.Dir（默认 logs/）
//   - 文件名：chatroom_YYYY-MM-DD.log 或 chatroom_YYYY-MM-DD_N.log
//     其中 N 是当天第几次启动（N=1 时省略后缀，N>=2 时追加 _N）
//   - 日期轮替：每条日志写入前检查当前日期，若跨天则关闭旧文件并新建当天文件（序号重置为 1）
//
// 调试模式（cfg.Debug=true）：
//   - 日志级别强制设为 Debug
//   - 中间件会将每条 HTTP 请求也写入日志
package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/hui0882/chatroom/pkg/config"
)

var (
	global *zap.Logger
	once   sync.Once
)

// Init 初始化全局日志，必须在程序启动时调用一次
func Init(cfg *config.Config) error {
	var initErr error
	once.Do(func() {
		l, err := build(cfg)
		if err != nil {
			initErr = err
			return
		}
		global = l
	})
	return initErr
}

// L 返回全局 logger，Init 之前调用会 panic
func L() *zap.Logger {
	if global == nil {
		panic("logger not initialized, call logger.Init first")
	}
	return global
}

// Sync 在程序退出前调用，确保缓冲日志落盘
func Sync() {
	if global != nil {
		_ = global.Sync()
	}
}

// ---------------------------------------------------------------------------
// 内部实现
// ---------------------------------------------------------------------------

func build(cfg *config.Config) (*zap.Logger, error) {
	level := parseLevel(cfg.Log.Level)
	// 调试模式强制 Debug 级别
	if cfg.App.Debug {
		level = zapcore.DebugLevel
	}

	encoderCfg := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core

	// stdout core（始终开启）
	consoleEnc := zapcore.NewConsoleEncoder(encoderCfg)
	consoleCore := zapcore.NewCore(consoleEnc, zapcore.AddSync(os.Stdout), level)
	cores = append(cores, consoleCore)

	// file core（output=file 时开启）
	if cfg.Log.Output == "file" {
		fileWriter, err := newDateRotateWriter(cfg.Log.Dir)
		if err != nil {
			return nil, fmt.Errorf("create log file writer: %w", err)
		}
		fileEnc := zapcore.NewJSONEncoder(encoderCfg)
		fileCore := zapcore.NewCore(fileEnc, zapcore.AddSync(fileWriter), level)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(0))
	return logger, nil
}

func parseLevel(s string) zapcore.Level {
	switch s {
	case "debug":
		return zapcore.DebugLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// ---------------------------------------------------------------------------
// dateRotateWriter：基于日期的轮替写入器
// ---------------------------------------------------------------------------

// dateRotateWriter 实现 io.Writer，在跨天时自动切换到新日志文件。
// 文件命名规则：
//
//	第 1 次启动：chatroom_2025-04-09.log
//	第 2 次启动：chatroom_2025-04-09_2.log
//	跨天后新文件：chatroom_2025-04-10.log（序号重置，当天第 1 次写入时新建）
type dateRotateWriter struct {
	dir     string
	mu      sync.Mutex
	file    *os.File
	curDate string // 当前文件对应的日期 "2006-01-02"
}

func newDateRotateWriter(dir string) (*dateRotateWriter, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("mkdir %s: %w", dir, err)
	}
	w := &dateRotateWriter{dir: dir}
	if err := w.openFile(); err != nil {
		return nil, err
	}
	return w, nil
}

// Write 实现 io.Writer；每次写入前检查是否需要轮替
func (w *dateRotateWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("2006-01-02")
	if today != w.curDate {
		// 跨天，关闭旧文件，重新开一个
		if w.file != nil {
			_ = w.file.Close()
			w.file = nil
		}
		if err = w.openFileLocked(today); err != nil {
			return 0, err
		}
	}

	return w.file.Write(p)
}

// openFile 在启动时调用（无锁版本，只在构造时使用）
func (w *dateRotateWriter) openFile() error {
	today := time.Now().Format("2006-01-02")
	return w.openFileLocked(today)
}

// openFileLocked 创建或追加打开日志文件（调用时已持锁，或处于初始化阶段）
// 命名逻辑：先尝试 chatroom_<date>.log，若已存在则依次尝试 _2, _3 …
func (w *dateRotateWriter) openFileLocked(date string) error {
	path := w.buildPath(date, 0)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// 当天第一次启动，直接创建
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return fmt.Errorf("create log file %s: %w", path, err)
		}
		w.file = f
		w.curDate = date
		return nil
	}

	// 文件已存在（当天已有启动过），找下一个可用序号
	for n := 2; ; n++ {
		path = w.buildPath(date, n)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return fmt.Errorf("create log file %s: %w", path, err)
			}
			w.file = f
			w.curDate = date
			return nil
		}
	}
}

// buildPath 构造日志文件路径
//
//	n=0 → chatroom_2025-04-09.log
//	n=2 → chatroom_2025-04-09_2.log
func (w *dateRotateWriter) buildPath(date string, n int) string {
	var name string
	if n <= 1 {
		name = fmt.Sprintf("chatroom_%s.log", date)
	} else {
		name = fmt.Sprintf("chatroom_%s_%d.log", date, n)
	}
	return filepath.Join(w.dir, name)
}
