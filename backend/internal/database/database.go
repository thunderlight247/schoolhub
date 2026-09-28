// Package database owns the PostgreSQL connection pool. Every other internal
// package that needs the database receives a *pgxpool.Pool rather than
// opening its own connections, so the pool's lifecycle stays in one place.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool creates a pgx connection pool for the given database URL and
// verifies connectivity with a ping before returning. Failing fast here
// means a misconfigured DATABASE_URL is caught at startup, not on the first
// request.
func NewPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: parsing connection string: %w", err)
	}

	// Conservative defaults for a small prototype; these can be tuned once
	// real usage patterns are known.
	poolCfg.MaxConns = 10
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: creating pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping failed: %w", err)
	}

	return pool, nil
}
