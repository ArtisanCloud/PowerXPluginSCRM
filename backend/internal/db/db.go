// package db
package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
)

var db *gorm.DB

func GetGlobalDB() *gorm.DB { return db }

func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	// 默认值/校验
	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 日志等级
	level := gormLogger.Silent
	switch strings.ToLower(cfg.LogLevel) {
	case "debug":
		level = gormLogger.Info // gorm 没有 Debug 级别，用 Info 近似
	case "info":
		level = gormLogger.Info
	case "warn":
		level = gormLogger.Warn
	case "error":
		level = gormLogger.Error
	case "silent":
		level = gormLogger.Silent
	}

	gLogger := gormLogger.New(
		NewGormWriter(), // 适配到你的 logger
		gormLogger.Config{
			SlowThreshold:             cfg.SlowThreshold,
			LogLevel:                  level,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	gormConfig := &gorm.Config{
		Logger:               gLogger,
		NowFunc:              func() time.Time { return time.Now().UTC() },
		DisableAutomaticPing: true,
		PrepareStmt:          cfg.PrepareStmt,
	}

	var dialector gorm.Dialector
	switch strings.ToLower(cfg.Driver) {
	case "memory", "sqlite":
		if err := ensureSQLiteDir(cfg.DSN); err != nil {
			return nil, err
		}
		// SQLite 内存模式：禁用外键自动迁移，保留事务默认值
		gormConfig.SkipDefaultTransaction = cfg.SkipDefaultTx
		gormConfig.DisableForeignKeyConstraintWhenMigrating = true
		dialector = SQLiteDialector(cfg.DSN)
	case "postgres":
		gormConfig.SkipDefaultTransaction = cfg.SkipDefaultTx
		dialector = postgres.New(postgres.Config{
			DSN:                  cfg.DSN,
			PreferSimpleProtocol: cfg.PreferSimpleProtocol,
		})
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	// 打开 + 简单重试
	var err error
	backoff := []time.Duration{0, 500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
	for i, d := range backoff {
		if d > 0 {
			time.Sleep(d)
		}
		db, err = gorm.Open(dialector, gormConfig)
		if err == nil {
			break
		}
		logger.Errorf("[DB] open failed (try %d/%d): %v", i+1, len(backoff), err)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %w", err)
	}
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// 健康检查
	ctx, cancel := context.WithTimeout(context.Background(), cfg.HealthTimeout)
	defer cancel()
	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// schema 准备
	if strings.ToLower(cfg.Driver) == "postgres" {
		if err := createSchema(cfg.Schema); err != nil {
			return nil, fmt.Errorf("failed to create schema: %w", err)
		}
		if err := setDefaultSchema(cfg.Schema); err != nil {
			return nil, fmt.Errorf("failed to set default schema: %w", err)
		}
	}

	logger.Infof("Database connected. schema=%s pool{idle=%d open=%d}", cfg.Schema, cfg.MaxIdleConns, cfg.MaxOpenConns)
	return db, nil
}

// Close 关闭连接池（GORM v2 需关闭底层 *sql.DB）
func Close() error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}
	// 关闭后：禁止新查询、等待在途查询完成，再释放连接
	err = sqlDB.Close()
	db = nil // 避免误用
	return err
}

func SQL() (*sql.DB, error) {
	if db == nil {
		return nil, fmt.Errorf("db not initialized")
	}
	return db.DB()
}

func ensureSQLiteDir(dsn string) error {
	if dsn == "" || !strings.HasPrefix(dsn, "file:") {
		return nil
	}

	body := strings.TrimPrefix(dsn, "file:")
	if strings.HasPrefix(body, ":") {
		// 内存模式无需处理
		return nil
	}

	stopIdx := len(body)
	if idx := strings.IndexAny(body, "?#"); idx >= 0 {
		stopIdx = idx
	}
	pathPart := body[:stopIdx]
	if pathPart == "" {
		return nil
	}
	if !filepath.IsAbs(pathPart) {
		abs, err := filepath.Abs(pathPart)
		if err == nil {
			pathPart = abs
		}
	}

	dir := filepath.Dir(pathPart)
	if dir == "." || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}
