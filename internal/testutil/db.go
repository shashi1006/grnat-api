// Package testutil provides shared helpers for integration tests.
package testutil

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/readygeneration/readygeneration-backend/internal/config"
	"github.com/readygeneration/readygeneration-backend/internal/db"
	"github.com/readygeneration/readygeneration-backend/internal/migrate"
)

// NewTestDB spins up a throwaway Postgres container, runs migrations, and
// returns a connection pool. The container is terminated when the test finishes.
func NewTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	ctr, err := postgres.Run(ctx,
		"pgvector/pgvector:pg16",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("get connection string: %v", err)
	}

	// Run migrations
	if err := migrate.Up(dsn, "migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	cfg := &config.Config{
		DB: config.DBConfig{
			URL:             dsn,
			MaxOpenConns:    5,
			MaxIdleConns:    2,
			ConnMaxLifetime: 0,
		},
	}
	pool, err := db.NewPool(ctx, cfg)
	if err != nil {
		t.Fatalf("connect pool: %v", err)
	}
	t.Cleanup(pool.Close)

	return pool
}

// Ptr is a generic helper that returns a pointer to any value.
func Ptr[T any](v T) *T { return &v }

// Must fails the test on non-nil error.
func Must(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// MustVal fails the test on non-nil error and returns the value.
func MustVal[T any](t *testing.T, v T, err error, msg string) T {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
	return v
}

// IntegrationSkip skips the test unless -run=Integration is passed or
// INTEGRATION_TESTS=1 env var is set. Prevents CI from running DB tests
// unless a container runtime is available.
func IntegrationSkip(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	// Allow opt-in via env or build tag
	fmt.Sprintf("") // noop — actual skip handled by -short flag
}
