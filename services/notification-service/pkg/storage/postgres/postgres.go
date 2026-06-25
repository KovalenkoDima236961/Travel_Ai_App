package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// PoolIface is the subset of *pgxpool.Pool used by DB and its callers.
// Swap in a compatible pool during tests.
type PoolIface interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	BeginTx(ctx context.Context, txOptions pgx.TxOptions) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Ping(ctx context.Context) error
	Close()
	Reset()
	Stat() *pgxpool.Stat
}

type DB struct {
	Pool    PoolIface
	Builder squirrel.StatementBuilderType
}

func New(ctx context.Context, cfg Config) (*DB, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	postgresCfg, err := pgxpool.ParseConfig(fmt.Sprintf("%s&pool_max_conns=%d&pool_min_conns=%d", connString, cfg.MaxConns, cfg.MinConns))
	if err != nil {
		return nil, fmt.Errorf("unable to parse database connection config: %w", err)
	}

	var pool *pgxpool.Pool
	pool, err = pgxpool.NewWithConfig(ctx, postgresCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	db := &DB{
		Pool:    pool,
		Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}

	if err = doMigrate(connString, cfg.MigPath); err != nil {
		pool.Close()
		return nil, err
	}

	return db, nil
}

func doMigrate(connStr, migPath string) error {
	m, err := migrate.New(fmt.Sprintf("file://%s", migPath), connStr)
	if err != nil {
		return fmt.Errorf("failed creating migrations: %w", err)
	}

	if err = m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed executing migrations: %w", err)
	}

	return nil
}

func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return db.Pool.Exec(ctx, query, args...)
}

func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return db.Pool.Query(ctx, query, args...)
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return db.Pool.QueryRow(ctx, query, args...)
}

func (db *DB) Close() {
	db.Pool.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

func (db *DB) Reset() {
	db.Pool.Reset()
}

func (db *DB) Stat() *pgxpool.Stat {
	return db.Pool.Stat()
}

func (db *DB) BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error) {
	return db.Pool.BeginTx(ctx, opts)
}
