package logger

import (
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger 全局日志实例
var Logger *logrus.Logger

// Fields 日志字段类型别名
type Fields = logrus.Fields

// Init 初始化日志配置
func Init(level string) {
	Logger = logrus.New()

	// 设置日志级别
	logLevel, err := logrus.ParseLevel(strings.ToLower(level))
	if err != nil {
		logLevel = logrus.InfoLevel
	}
	Logger.SetLevel(logLevel)

	// 设置输出格式
	if level == "debug" {
		// 开发模式使用文本格式
		Logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	} else {
		// 生产模式使用 JSON 格式
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// 设置输出
	Logger.SetOutput(os.Stdout)

	// 添加调用位置信息（仅在 debug 模式）
	if logLevel == logrus.DebugLevel || logLevel == logrus.TraceLevel {
		tf := &logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
			CallerPrettyfier: func(f *runtime.Frame) (function string, file string) {
				funcName := path.Base(f.Function)
				fileName := path.Base(f.File)
				return funcName, fmt.Sprintf("%s:%d", fileName, f.Line)
			},
		}
		Logger.SetFormatter(tf)
		Logger.SetReportCaller(true)
	}
}

// SetOutput 设置日志输出
func SetOutput(output io.Writer) {
	if Logger != nil {
		Logger.SetOutput(output)
	}
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
