// Package postgres stores job metadata and reads the review report that
// document-service writes back.
package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	poolMaxConns          = 10
	poolMinConns          = 1
	poolMaxConnLifetime   = time.Hour
	poolMaxConnIdleTime   = 30 * time.Minute
	poolHealthCheckPeriod = time.Minute
	poolConnectTimeout    = 5 * time.Second
)

// OpenPool builds a connection pool from a DATABASE_URL and verifies it is
// reachable before returning.
func OpenPool(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	config.MaxConns = poolMaxConns
	config.MinConns = poolMinConns
	config.MaxConnLifetime = poolMaxConnLifetime
	config.MaxConnIdleTime = poolMaxConnIdleTime
	config.HealthCheckPeriod = poolHealthCheckPeriod
	config.ConnConfig.ConnectTimeout = poolConnectTimeout

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}
