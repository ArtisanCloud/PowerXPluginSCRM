//go:build cgo

package db

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLiteDialector returns a SQLite dialector for the current build.
// In cgo builds we use the default gorm sqlite dialector (go-sqlite3).
func SQLiteDialector(dsn string) gorm.Dialector {
	return sqlite.Open(dsn)
}

