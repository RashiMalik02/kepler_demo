package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	maxConns       = 10
	acquireTimeout = 2 * time.Second
)

var errPoolExhausted = errors.New("db pool exhausted")

type DB struct {
	pool *pgxpool.Pool
}

func openDB(ctx context.Context, url string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = maxConns

	// orders-db may still be starting when we roll out, so retry for about a minute.
	for attempt := 1; ; attempt++ {
		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			if err = pool.Ping(ctx); err == nil {
				db := &DB{pool: pool}
				return db, db.migrate(ctx)
			}
			pool.Close()
		}
		if attempt == 30 {
			return nil, err
		}
		slog.Warn("orders-db not ready, retrying", "attempt", attempt, "err", err)
		time.Sleep(2 * time.Second)
	}
}

func (db *DB) Close() { db.pool.Close() }

// acquire waits at most acquireTimeout for a free connection, so a slow
// database turns into fast 500s instead of an ever-growing request queue.
func (db *DB) acquire(ctx context.Context) (*pgxpool.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, acquireTimeout)
	defer cancel()
	conn, err := db.pool.Acquire(ctx)
	if err != nil {
		s := db.pool.Stat()
		return nil, fmt.Errorf("%w: waited %s for a connection (in_use=%d/%d)",
			errPoolExhausted, acquireTimeout, s.AcquiredConns(), s.MaxConns())
	}
	return conn, nil
}

func (db *DB) migrate(ctx context.Context) error {
	for _, stmt := range []string{
		`CREATE TABLE IF NOT EXISTS charges (
			id         TEXT PRIMARY KEY,
			account_id INT NOT NULL,
			amount     BIGINT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS ledger_entries (
			id         BIGSERIAL PRIMARY KEY,
			charge_id  TEXT NOT NULL,
			account_id INT NOT NULL,
			line       INT NOT NULL,
			amount     BIGINT NOT NULL,
			balance    BIGINT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	} {
		if _, err := db.pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}
