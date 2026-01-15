// package models
package models

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"gorm.io/gorm"
)

const defaultSchema = "public"

var (
	schemaName = defaultSchema // 默认 schema（本地开发使用 public）
	once       sync.Once
)

// InitSchemaFrom 仅在首次调用时生效：把配置文件里的 schema 注入进来
func InitSchemaFrom(schema string) {
	once.Do(func() {
		if s := strings.TrimSpace(schema); s != "" {
			if !isValidSchema(s) {
				panic(fmt.Errorf("invalid schema name: %q (only [A-Za-z_][A-Za-z0-9_]* allowed)", s))
			}
			schemaName = s
			return
		}
		// 空字符串代表不使用 schema 前缀（SQLite/memory 模式）
		schemaName = ""
	})
}

// Schema 返回当前有效的 schema 名
func Schema() string {
	return schemaName
}

// S 返回带 schema 前缀的完整表名："schema".table
// 注意：对 schema 加引号保留大小写、避免关键字冲突；表名建议全小写下划线，不加引号。
func S(table string) string {
	schema := Schema()
	if strings.TrimSpace(schema) == "" {
		return table
	}
	return fmt.Sprintf(`"%s".%s`, schema, table)
}

// EnsureSchema 若不存在则创建 schema（PostgreSQL）
func EnsureSchema(db *gorm.DB) error {
	q := fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS "%s"`, Schema())
	return db.Exec(q).Error
}

// 仅允许以字母或下划线开头、其后字母/数字/下划线
var schemaRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func isValidSchema(s string) bool { return schemaRe.MatchString(s) }

// ForceSchemaForTests overrides schema configuration for tests and resets InitSchemaFrom guard.
func ForceSchemaForTests(schema string) {
	schemaName = schema
	once = sync.Once{}
}
