package db

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// qi: quote identifier，给 schema/table/policy 名加安全双引号
func qi(ident string) string {
	ident = strings.TrimSpace(ident)
	ident = strings.ReplaceAll(ident, `"`, `""`)
	return `"` + ident + `"`
}

// createSchema: CREATE SCHEMA IF NOT EXISTS
func createSchema(schema string) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	if strings.TrimSpace(schema) == "" {
		return errors.New("empty schema")
	}
	sqlText := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", qi(schema))
	if err := db.Exec(sqlText).Error; err != nil {
		if isPermissionDenied(err) {
			exists, checkErr := schemaExists(schema)
			if checkErr != nil {
				return checkErr
			}
			if exists {
				return nil
			}
		}
		return err
	}
	return nil
}

// setDefaultSchema: SET search_path（会话级）
func setDefaultSchema(schema string) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	if strings.TrimSpace(schema) == "" {
		return errors.New("empty schema")
	}
	sqlText := fmt.Sprintf("SET search_path TO %s", qi(schema))
	return db.Exec(sqlText).Error
}

// Health 检查数据库健康状态
func Health() error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return sqlDB.PingContext(ctx)
}

// Migrate 执行数据库迁移
func Migrate(models ...interface{}) error {
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	return db.AutoMigrate(models...)
}

// EnableRLS 为表启用行级安全
func EnableRLS(tableName string) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	sql := fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY", tableName)
	return db.Exec(sql).Error
}

// CreateRLSPolicy 创建 RLS 策略
func CreateRLSPolicy(tableName, policyName string) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	sql := fmt.Sprintf(`
		CREATE POLICY IF NOT EXISTS %s ON %s
		USING (tenant_uuid::text = current_setting('app.tenant_uuid', true))
	`, policyName, tableName)
	return db.Exec(sql).Error
}

// DropRLSPolicy 删除 RLS 策略
func DropRLSPolicy(tableName, policyName string) error {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	sql := fmt.Sprintf("DROP POLICY IF EXISTS %s ON %s", policyName, tableName)
	return db.Exec(sql).Error
}

type sqlStateError interface {
	SQLState() string
}

func isPermissionDenied(err error) bool {
	var sqlErr sqlStateError
	if errors.As(err, &sqlErr) {
		return sqlErr.SQLState() == "42501"
	}
	return strings.Contains(err.Error(), "42501")
}

func schemaExists(schema string) (bool, error) {
	if db == nil || db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return false, errors.New("database not initialized")
	}
	var count int64
	query := `SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = ?`
	if err := db.Raw(query, schema).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}
