// package config 或者你原来的包
package config

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type DatabaseConfig struct {
	// Driver 指定数据库驱动：postgres、sqlite、memory（别名 sqlite 内存）
	Driver string `yaml:"driver" json:"driver"`
	// 连接串：postgres://user:pass@host:5432/dbname?sslmode=disable
	DSN string `yaml:"dsn" json:"dsn"`
	// 默认 schema（会在连接后执行 CREATE SCHEMA IF NOT EXISTS + SET search_path）
	Schema string `yaml:"schema" json:"schema"`

	// ---------- 连接池 ----------
	MaxIdleConns    int           `yaml:"maxIdleConns" json:"maxIdleConns"`       // 默认 10
	MaxOpenConns    int           `yaml:"maxOpenConns" json:"maxOpenConns"`       // 默认 100
	ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" json:"connMaxLifetime"` // 默认 1h
	ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" json:"connMaxIdleTime"` // 默认 30m

	// ---------- 观测 & 性能 ----------
	SlowThreshold        time.Duration `yaml:"slowThreshold" json:"slowThreshold"`               // 默认 200ms（gorm 慢查询阈值）
	PreferSimpleProtocol bool          `yaml:"preferSimpleProtocol" json:"preferSimpleProtocol"` // 默认 true
	PrepareStmt          bool          `yaml:"prepareStmt" json:"prepareStmt"`                   // 默认 false（按需开启）
	SkipDefaultTx        bool          `yaml:"skipDefaultTx" json:"skipDefaultTx"`               // 默认 false

	// ---------- 日志 ----------
	// debug|info|warn|error|silent（不区分大小写）
	LogLevel string `yaml:"logLevel" json:"logLevel"` // 默认 "silent"
	DevMode  bool   `yaml:"devMode" json:"devMode"`   // 默认 false（为 true 时，LogLevel 自动提升到 info）

	// ---------- 运维 ----------
	HealthTimeout time.Duration `yaml:"healthTimeout" json:"healthTimeout"` // 默认 5s
}

func (c *DatabaseConfig) ApplyDefaults() {
	driver := strings.ToLower(strings.TrimSpace(c.Driver))
	if driver == "" {
		driver = "memory"
	}
	c.Driver = driver

	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = 10
	}
	if c.MaxOpenConns == 0 {
		c.MaxOpenConns = 100
	}
	if c.ConnMaxLifetime == 0 {
		c.ConnMaxLifetime = time.Hour
	}
	if c.ConnMaxIdleTime == 0 {
		c.ConnMaxIdleTime = 30 * time.Minute
	}
	if c.SlowThreshold == 0 {
		c.SlowThreshold = 200 * time.Millisecond
	}
	if c.LogLevel == "" {
		c.LogLevel = "silent"
	}
	if c.HealthTimeout == 0 {
		c.HealthTimeout = 5 * time.Second
	}
	// Dev 模式自动抬升日志
	if c.DevMode && (c.LogLevel == "silent" || c.LogLevel == "error") {
		c.LogLevel = "info"
	}

	switch c.Driver {
	case "memory", "sqlite":
		if c.DSN == "" {
			// 使用 shared cache 便于多连接共享数据
			c.DSN = "file:powerxplugin?mode=memory&cache=shared"
		}
		// SQLite/memory 模式不需要 schema 前缀，保持为空即可
		c.Schema = strings.TrimSpace(c.Schema)
	default:
		// 默认更轻的协议，降低 CPU（Postgres 专用）
		if !c.PreferSimpleProtocol {
			c.PreferSimpleProtocol = true
		}
	}
}

func (c *DatabaseConfig) Validate() error {
	switch c.Driver {
	case "memory", "sqlite":
		// 允许使用内存/SQLite，不强制 DSN/schema
		return nil
	case "", "postgres":
		// 兼容旧配置：缺省驱动等同 postgres
		c.Driver = "postgres"
	default:
		return fmt.Errorf("unsupported database driver: %s", c.Driver)
	}

	if c.DSN == "" {
		return errors.New("database dsn is required")
	}
	if c.Schema == "" {
		return errors.New("database schema is required")
	}
	if c.MaxIdleConns < 0 || c.MaxOpenConns < 0 {
		return errors.New("maxIdleConns/maxOpenConns cannot be negative")
	}
	if c.MaxOpenConns > 0 && c.MaxIdleConns > c.MaxOpenConns {
		return fmt.Errorf("maxIdleConns(%d) cannot be greater than maxOpenConns(%d)", c.MaxIdleConns, c.MaxOpenConns)
	}
	return nil
}

// ResolvePaths 将 SQLite DSN 中的相对文件路径转换成以配置文件目录为基准的绝对路径。
func (c *DatabaseConfig) ResolvePaths(baseDir string) {
	if c == nil || baseDir == "" {
		return
	}

	driver := strings.ToLower(strings.TrimSpace(c.Driver))
	if driver != "sqlite" && driver != "memory" {
		return
	}

	dsn := strings.TrimSpace(c.DSN)
	if dsn == "" || !strings.HasPrefix(dsn, "file:") {
		return
	}

	body := strings.TrimPrefix(dsn, "file:")
	// 内存模式（file::memory:?cache=shared）无需处理。
	if strings.HasPrefix(body, ":") {
		return
	}

	var pathPart, suffix string
	if idx := strings.IndexAny(body, "?#"); idx >= 0 {
		pathPart = body[:idx]
		suffix = body[idx:]
	} else {
		pathPart = body
	}

	if pathPart == "" || filepath.IsAbs(pathPart) {
		return
	}

	resolved := filepath.Clean(filepath.Join(baseDir, pathPart))
	c.DSN = "file:" + resolved + suffix
}
