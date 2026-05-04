package acquisition

import (
	"errors"
	"strings"

	"github.com/jackc/pgconn"
)

func isUndefinedRelationErr(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "sqlstate 42p01") ||
		(strings.Contains(msg, "relation") && strings.Contains(msg, "does not exist"))
}
