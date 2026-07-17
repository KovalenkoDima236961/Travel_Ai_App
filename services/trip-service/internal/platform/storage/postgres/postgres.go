package postgres

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/golang-migrate/migrate/v4/source/github"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/zap"
)

// PoolIface is the subset of *pgxpool.Pool used by DB and its callers.
// Swap in pgxmock.NewPool() during tests.
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
	Pool               PoolIface
	Builder            squirrel.StatementBuilderType
	log                *zap.Logger
	slowQueryThreshold time.Duration
}

func New(ctx context.Context, cfg Config, logs ...*zap.Logger) (*DB, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	postgresCfg, err := pgxpool.ParseConfig(fmt.Sprintf("%s&pool_max_conns=%d&pool_min_conns=%d", connString, cfg.MaxConns, cfg.MinConns))
	if err != nil {
		return nil, fmt.Errorf("unable to parse database connection config: %w", err)
	}
	if cfg.QueryTimeoutSeconds > 0 {
		postgresCfg.ConnConfig.RuntimeParams["statement_timeout"] = strconv.Itoa(cfg.QueryTimeoutSeconds * 1000)
	}
	log := zap.NewNop()
	if len(logs) > 0 && logs[0] != nil {
		log = logs[0]
	}

	var pool *pgxpool.Pool
	pool, err = pgxpool.NewWithConfig(ctx, postgresCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %w", err)
	}

	db := &DB{
		Pool:               pool,
		Builder:            squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		log:                log,
		slowQueryThreshold: time.Duration(cfg.SlowQueryThresholdMS) * time.Millisecond,
	}

	if err = doMigrate(connString, cfg.MigPath); err != nil {
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
	started := time.Now()
	result, err := db.Pool.Exec(ctx, query, args...)
	db.recordQuery(query, started, err)
	return result, err
}

func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	started := time.Now()
	rows, err := db.Pool.Query(ctx, query, args...)
	db.recordQuery(query, started, err)
	return rows, err
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return &timedRow{row: db.Pool.QueryRow(ctx, query, args...), db: db, query: query, started: time.Now()}
}

type timedRow struct {
	row     pgx.Row
	db      *DB
	query   string
	started time.Time
}

func (r *timedRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	r.db.recordQuery(r.query, r.started, err)
	return err
}

func (db *DB) recordQuery(query string, started time.Time, err error) {
	duration := time.Since(started)
	operation := queryOperation(query)
	recordDBQuery(operation, duration, err, db.Stat())
	if db.slowQueryThreshold > 0 && duration >= db.slowQueryThreshold {
		db.log.Warn("slow database query",
			zap.String("operation", operation),
			zap.Duration("duration", duration),
			zap.Bool("failed", err != nil),
		)
	}
}

func queryOperation(query string) string {
	fields := strings.Fields(strings.TrimSpace(query))
	if len(fields) == 0 {
		return "unknown"
	}
	op := strings.ToLower(fields[0])
	switch op {
	case "select", "insert", "update", "delete", "with":
		return op
	default:
		return "other"
	}
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
