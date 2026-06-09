package db

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/readygeneration/readygeneration-backend/internal/config"
)

// EnsureDB connects to the postgres maintenance database and creates the
// target database if it does not already exist. Safe to call on every start.
func EnsureDB(ctx context.Context, dbURL string) error {
	u, err := url.Parse(dbURL)
	if err != nil {
		return fmt.Errorf("parse db url: %w", err)
	}

	dbName := u.Path
	if len(dbName) > 0 && dbName[0] == '/' {
		dbName = dbName[1:]
	}

	// Connect to the postgres maintenance database instead
	u.Path = "/postgres"
	maintenanceURL := u.String()

	conn, err := pgx.Connect(ctx, maintenanceURL)
	if err != nil {
		return fmt.Errorf("connect to postgres maintenance db: %w", err)
	}
	defer conn.Close(ctx)

	var exists bool
	err = conn.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", dbName,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check db existence: %w", err)
	}

	if !exists {
		// Database names cannot be parameterised in DDL
		_, err = conn.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, dbName))
		if err != nil {
			return fmt.Errorf("create database %q: %w", dbName, err)
		}
		fmt.Printf("created database %q\n", dbName)
	}

	return nil
}

// NewPool creates a new pgx connection pool.
func NewPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DB.URL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.DB.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.DB.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.DB.ConnMaxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}
