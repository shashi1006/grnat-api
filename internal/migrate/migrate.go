package migrate

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// Up applies all pending up migrations.
func Up(databaseURL, migrationsDir string) error {
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsDir),
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

// Down rolls back all migrations.
func Down(databaseURL, migrationsDir string) error {
	if migrationsDir == "" {
		migrationsDir = "migrations"
	}

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsDir),
		databaseURL,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}
	defer m.Close()

	return m.Down()
}
