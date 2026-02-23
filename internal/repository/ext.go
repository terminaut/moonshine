package repository

import (
	"database/sql"

	"github.com/google/uuid"
)

type ExtHandle interface {
	Exec(query string, args ...any) (sql.Result, error)
	QueryRow(query string, args ...any) *sql.Row
	Get(dest any, query string, args ...any) error
	Select(dest any, query string, args ...any) error
}

func UUIDsToStrings(ids []uuid.UUID) []string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = id.String()
	}
	return s
}
