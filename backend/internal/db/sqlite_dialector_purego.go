//go:build !cgo

package db

import (
	_ "modernc.org/sqlite"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLiteDialector returns a SQLite dialector for the current build.
// In non-cgo builds we use the pure Go modernc SQLite driver via database/sql.
func SQLiteDialector(dsn string) gorm.Dialector {
	return sqlite.Dialector{
		DriverName: "sqlite",
		DSN:        dsn,
	}
}

