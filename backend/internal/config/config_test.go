package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesEnvOverrides(t *testing.T) {
	const (
		dsn    = "postgres://user:pass@127.0.0.1:5432/powerx_test?sslmode=disable"
		schema = "px_override"
	)

	t.Setenv("POWERX_DB_DSN", dsn)
	t.Setenv("POWERX_DB_SCHEMA", schema)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_LOG_LEVEL", "INFO")

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yaml")
	configContent := "server:\n  bind_addr: \"127.0.0.1:0\"\nlogging:\n  level: WARN\n  format: TEXT\n  output: STDOUT\n"
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	if cfg.Database == nil {
		t.Fatal("Database 配置未初始化")
	}
	if cfg.Database.DSN != dsn {
		t.Fatalf("POWERX_DB_DSN 未生效，期望 %q 实际 %q", dsn, cfg.Database.DSN)
	}
	if cfg.Database.Schema != schema {
		t.Fatalf("POWERX_DB_SCHEMA 未生效，期望 %q 实际 %q", schema, cfg.Database.Schema)
	}
	if cfg.Logging.Level != "info" || cfg.Server.LogLevel != "info" {
		t.Fatalf("POWERX_LOG_LEVEL 未归一化为小写 info, server=%q logging=%q", cfg.Server.LogLevel, cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" || cfg.Logging.Output != "stdout" {
		t.Fatalf("日志配置未归一化: format=%q output=%q", cfg.Logging.Format, cfg.Logging.Output)
	}
}

func TestLoadNormalizesLoggingFromYAML(t *testing.T) {
	tempDir := t.TempDir()
	configContent := "logging:\n  level: ERROR\n  format: JSON\n  output: STDERR\n"
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Logging.Level != "error" || cfg.Logging.Format != "json" || cfg.Logging.Output != "stderr" {
		t.Fatalf("YAML 归一化失败: level=%q format=%q output=%q", cfg.Logging.Level, cfg.Logging.Format, cfg.Logging.Output)
	}
}

func TestLoadResolvesPlaceholderDefaults(t *testing.T) {
	tempDir := t.TempDir()
	configContent := "server:\n  bind_addr: \"${POWERX_BIND_ADDR:-:9000}\"\nlogging:\n  level: \"${POWERX_LOG_LEVEL:-INFO}\"\n  format: \"${POWERX_LOG_FORMAT:-JSON}\"\n  output: \"${POWERX_LOG_OUTPUT:-STDOUT}\"\n"
	configFile := filepath.Join(tempDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	t.Setenv("CONFIG_PATH", tempDir)
	t.Setenv("POWERX_DEV_MODE", "true")
	t.Setenv("POWERX_DB_DSN", "postgres://user:pass@127.0.0.1:5432/test?sslmode=disable")
	t.Setenv("POWERX_DB_SCHEMA", "px_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Server.BindAddr != ":9000" {
		t.Fatalf("server.bind_addr 占位符未解析，得到 %q", cfg.Server.BindAddr)
	}
	if cfg.Logging.Level != "info" {
		t.Fatalf("logging.level 占位符未解析，得到 %q", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" || cfg.Logging.Output != "stdout" {
		t.Fatalf("logging format/output 占位符未解析，format=%q output=%q", cfg.Logging.Format, cfg.Logging.Output)
	}
}

func TestLoadUsesConfigPathPlaceholder(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "config")
	if err := os.Mkdir(configDir, 0o755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}
	configContent := "server:\n  bind_addr: \":0\"\n  log_level: \"INFO\"\n  dev_mode: true\ndatabase:\n  dsn: \"postgres://user:pass@127.0.0.1:5432/test?sslmode=disable\"\n  schema: \"px_test\"\nlogging:\n  level: \"INFO\"\n  format: \"JSON\"\n  output: \"STDOUT\"\n"
	configFile := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configFile, []byte(configContent), 0o644); err != nil {
		t.Fatalf("写入 host-values 配置失败: %v", err)
	}
	t.Setenv("POWERX_PLUGIN_CONFIG_DIR", configDir)
	t.Setenv("CONFIG_PATH", "${POWERX_PLUGIN_CONFIG_DIR:-./backend/etc}")
	t.Setenv("POWERX_DEV_MODE", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}
	if cfg.Database == nil || cfg.Database.DSN == "" {
		t.Fatal("未从 CONFIG_PATH 提供的 YAML 中读取到数据库 DSN")
	}
	if cfg.Server.LogLevel != "info" {
		t.Fatalf("CONFIG_PATH 配置未生效，log level=%q", cfg.Server.LogLevel)
	}
}
