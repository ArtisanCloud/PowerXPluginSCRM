package logger

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

// Logger 全局日志实例。业务代码仍使用 logrus API，实际输出统一转发到 slog。
var Logger *logrus.Logger

var (
	backendMu     sync.RWMutex
	backend       *slog.Logger
	loggerOutput  io.Writer    = os.Stdout
	backendLevel  logrus.Level = logrus.InfoLevel
	backendFormat string       = "json"
)

// Fields 日志字段类型别名
type Fields = logrus.Fields
type Entry = logrus.Entry
type LoggerType = logrus.Logger
type Level = logrus.Level

const (
	PanicLevel Level = logrus.PanicLevel
	FatalLevel Level = logrus.FatalLevel
	ErrorLevel Level = logrus.ErrorLevel
	WarnLevel  Level = logrus.WarnLevel
	InfoLevel  Level = logrus.InfoLevel
	DebugLevel Level = logrus.DebugLevel
	TraceLevel Level = logrus.TraceLevel
)

// Init 初始化日志配置
func Init(level string) {
	InitWithOptions(level, "", "", "")
}

// InitWithOptions initializes logger by logging config fields.
// - level: debug/info/warn/error
// - format: text/json (empty means legacy auto by level)
// - output: stdout/stderr/file (empty means stdout)
// - filePath: required when output=file
func InitWithOptions(level, format, output, filePath string) {
	Logger = logrus.New()

	normalizedLevel := strings.ToLower(strings.TrimSpace(level))
	normalizedFormat := strings.ToLower(strings.TrimSpace(format))
	normalizedOutput := strings.ToLower(strings.TrimSpace(output))
	if normalizedFormat == "" {
		normalizedFormat = "json"
	}

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(normalizedLevel)
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	Logger.SetLevel(logLevel)

	// 设置输出目标，默认 stdout
	switch normalizedOutput {
	case "", "stdout":
		loggerOutput = os.Stdout
	case "stderr":
		loggerOutput = os.Stderr
	case "file":
		fp := strings.TrimSpace(filePath)
		if fp == "" {
			fmt.Fprintln(os.Stderr, "logger: output=file but file_path is empty, fallback to stdout")
			loggerOutput = os.Stdout
			break
		}
		if err := ensureLogParentDir(fp); err != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to create log dir for %q: %v, fallback to stdout\n", fp, err)
			loggerOutput = os.Stdout
			break
		}
		f, openErr := os.OpenFile(fp, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
		if openErr != nil {
			fmt.Fprintf(os.Stderr, "logger: failed to open log file %q: %v, fallback to stdout\n", fp, openErr)
			loggerOutput = os.Stdout
			break
		}
		loggerOutput = f
	default:
		loggerOutput = os.Stdout
	}

	backendLevel = logLevel
	backendFormat = normalizedFormat
	setBackendLogger(newSlogLogger(loggerOutput, backendFormat, backendLevel))
	slog.SetDefault(Slog())

	// logrus 仅作为兼容层，统一转发到 slog backend。
	Logger.SetOutput(io.Discard)
	Logger.SetFormatter(&logrus.JSONFormatter{})
	Logger.ReplaceHooks(make(logrus.LevelHooks))
	Logger.AddHook(&slogForwardHook{backend: Slog()})

	// 添加调用位置信息（仅在 debug/trace 模式）
	if logLevel == logrus.DebugLevel || logLevel == logrus.TraceLevel {
		Logger.SetReportCaller(true)
	}
	configureStandardLogger(Logger)
}

func ConfigureFromConfig(level, format, output, filePath string) {
	InitWithOptions(level, format, output, filePath)
}

func configureStandardLogger(source *logrus.Logger) {
	if source == nil {
		return
	}
	std := logrus.StandardLogger()
	std.SetLevel(source.Level)
	std.SetOutput(io.Discard)
	std.SetFormatter(&logrus.JSONFormatter{})
	std.SetReportCaller(source.ReportCaller)
	std.ReplaceHooks(make(logrus.LevelHooks))
	std.AddHook(&slogForwardHook{backend: Slog()})
	log.SetOutput(loggerOutput)
	log.SetFlags(0)
}

func newSlogLogger(w io.Writer, format string, level logrus.Level) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level:     mapSlogLevel(level),
		AddSource: level == logrus.DebugLevel || level == logrus.TraceLevel,
	}
	if strings.EqualFold(strings.TrimSpace(format), "text") {
		return slog.New(slog.NewTextHandler(w, opts))
	}
	return slog.New(slog.NewJSONHandler(w, opts))
}

func mapSlogLevel(level logrus.Level) slog.Leveler {
	switch level {
	case logrus.TraceLevel:
		return slog.LevelDebug - 4
	case logrus.DebugLevel:
		return slog.LevelDebug
	case logrus.WarnLevel:
		return slog.LevelWarn
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func mapEntryLevel(level logrus.Level) slog.Level {
	switch level {
	case logrus.TraceLevel:
		return slog.LevelDebug - 4
	case logrus.DebugLevel:
		return slog.LevelDebug
	case logrus.InfoLevel:
		return slog.LevelInfo
	case logrus.WarnLevel:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

type slogForwardHook struct {
	backend *slog.Logger
}

func (h *slogForwardHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *slogForwardHook) Fire(entry *logrus.Entry) error {
	l := h.backend
	if l == nil {
		l = Slog()
	}

	attrs := make([]slog.Attr, 0, len(entry.Data)+2)
	for k, v := range entry.Data {
		attrs = append(attrs, slog.Any(k, v))
	}
	if entry.Caller != nil {
		attrs = append(attrs, slog.String("caller", fmt.Sprintf("%s:%d", path.Base(entry.Caller.File), entry.Caller.Line)))
		attrs = append(attrs, slog.String("func", path.Base(entry.Caller.Function)))
	}

	ctx := entry.Context
	if ctx == nil {
		ctx = context.Background()
	}
	l.LogAttrs(ctx, mapEntryLevel(entry.Level), entry.Message, attrs...)
	return nil
}

func setBackendLogger(l *slog.Logger) {
	backendMu.Lock()
	defer backendMu.Unlock()
	backend = l
}

// Slog 返回统一日志后端，PowerXPlugin framework 的 slog 默认日志也会接到这里。
func Slog() *slog.Logger {
	backendMu.RLock()
	defer backendMu.RUnlock()
	if backend == nil {
		return slog.Default()
	}
	return backend
}

// SetOutput 设置日志输出
func SetOutput(output io.Writer) {
	if output == nil {
		output = io.Discard
	}
	loggerOutput = output
	setBackendLogger(newSlogLogger(loggerOutput, backendFormat, backendLevel))
	slog.SetDefault(Slog())
	log.SetOutput(loggerOutput)
	if Logger != nil {
		Logger.ReplaceHooks(make(logrus.LevelHooks))
		Logger.AddHook(&slogForwardHook{backend: Slog()})
	}
	std := logrus.StandardLogger()
	std.SetOutput(io.Discard)
	std.ReplaceHooks(make(logrus.LevelHooks))
	std.AddHook(&slogForwardHook{backend: Slog()})
}

func SetLevel(level logrus.Level) {
	backendLevel = level
	if Logger != nil {
		Logger.SetLevel(level)
	}
	logrus.SetLevel(level)
	setBackendLogger(newSlogLogger(loggerOutput, backendFormat, backendLevel))
	slog.SetDefault(Slog())
}

func Output() io.Writer {
	if loggerOutput == nil {
		return os.Stdout
	}
	return loggerOutput
}

func ensureLogParentDir(filePath string) error {
	parent := strings.TrimSpace(filepathDir(strings.TrimSpace(filePath)))
	if parent == "" || parent == "." {
		return nil
	}
	return os.MkdirAll(parent, 0o755)
}

func filepathDir(p string) string {
	idx := strings.LastIndex(p, "/")
	if idx < 0 {
		idx = strings.LastIndex(p, "\\")
	}
	if idx < 0 {
		return "."
	}
	if idx == 0 {
		return p[:1]
	}
	return p[:idx]
}

// WithFields 创建带字段的日志条目
func WithFields(fields logrus.Fields) *logrus.Entry {
	if Logger == nil {
		return logrus.WithFields(fields)
	}
	return Logger.WithFields(fields)
}

// WithField 创建带单个字段的日志条目
func WithField(key string, value interface{}) *logrus.Entry {
	if Logger == nil {
		return logrus.WithField(key, value)
	}
	return Logger.WithField(key, value)
}

// WithError 创建带错误的日志条目
func WithError(err error) *logrus.Entry {
	if Logger == nil {
		return logrus.WithError(err)
	}
	return Logger.WithError(err)
}

func New() *logrus.Logger {
	if Logger != nil {
		return Logger
	}
	return logrus.New()
}

func StandardLogger() *logrus.Logger {
	if Logger != nil {
		return Logger
	}
	return logrus.StandardLogger()
}

func NewEntry(l *logrus.Logger) *logrus.Entry {
	if l == nil {
		l = StandardLogger()
	}
	return logrus.NewEntry(l)
}

// Debug 调试日志
func Debug(args ...interface{}) {
	if Logger == nil {
		logrus.Debug(args...)
		return
	}
	Logger.Debug(args...)
}

// Debugf 格式化调试日志
func Debugf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Debugf(format, args...)
		return
	}
	Logger.Debugf(format, args...)
}

// Info 信息日志
func Info(args ...interface{}) {
	if Logger == nil {
		logrus.Info(args...)
		return
	}
	Logger.Info(args...)
}

// Infof 格式化信息日志
func Infof(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Infof(format, args...)
		return
	}
	Logger.Infof(format, args...)
}

// Warn 警告日志
func Warn(args ...interface{}) {
	if Logger == nil {
		logrus.Warn(args...)
		return
	}
	Logger.Warn(args...)
}

// Warnf 格式化警告日志
func Warnf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Warnf(format, args...)
		return
	}
	Logger.Warnf(format, args...)
}

// Error 错误日志
func Error(args ...interface{}) {
	if Logger == nil {
		logrus.Error(args...)
		return
	}
	Logger.Error(args...)
}

// Errorf 格式化错误日志
func Errorf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Errorf(format, args...)
		return
	}
	Logger.Errorf(format, args...)
}

// Fatal 致命错误日志
func Fatal(args ...interface{}) {
	if Logger == nil {
		logrus.Fatal(args...)
		return
	}
	Logger.Fatal(args...)
}

// Fatalf 格式化致命错误日志
func Fatalf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Fatalf(format, args...)
		return
	}
	Logger.Fatalf(format, args...)
}

// Panic panic 日志
func Panic(args ...interface{}) {
	if Logger == nil {
		logrus.Panic(args...)
		return
	}
	Logger.Panic(args...)
}

// Panicf 格式化 panic 日志
func Panicf(format string, args ...interface{}) {
	if Logger == nil {
		logrus.Panicf(format, args...)
		return
	}
	Logger.Panicf(format, args...)
}

// HTTPMiddleware 创建 HTTP 中间件日志
func HTTPMiddleware() *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "http",
	})
}

// DBMiddleware 创建数据库中间件日志
func DBMiddleware() *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "database",
	})
}

// AuthMiddleware 创建认证中间件日志
func AuthMiddleware() *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "auth",
	})
}

// ServiceLogger 创建服务层日志
func ServiceLogger(service string) *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "service",
		"service":   service,
	})
}

// RepoLogger 创建仓储层日志
func RepoLogger(repo string) *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "repository",
		"repo":      repo,
	})
}

// HandlerLogger 创建处理器日志
func HandlerLogger(handler string) *logrus.Entry {
	return WithFields(logrus.Fields{
		"component": "handler",
		"handler":   handler,
	})
}

func callerPrettyForDebug(f *runtime.Frame) (function string, file string) {
	funcName := path.Base(f.Function)
	fileName := path.Base(f.File)
	return funcName, fmt.Sprintf("%s:%d", fileName, f.Line)
}
