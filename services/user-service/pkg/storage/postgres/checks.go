package postgres

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// UniqueConstraintViolation reports whether err is a Postgres unique violation.
func UniqueConstraintViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation
}

// NoRowsFound reports whether err is pgx.ErrNoRows.
func NoRowsFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
