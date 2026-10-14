package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/migrations"
)

// Migrate runs one migration command: up, down, or version.
// down rolls back a single step.
func Migrate(cfg config.PostgresConfig, command string) (string, error) {
	switch command {
	case "up", "down", "version":
	default:
		return "", fmt.Errorf("unknown migrate command %q", command)
	}

	databaseURL := postgresURL(cfg)
	sqlDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return "", unavailable("database", err, cfg.Secrets()...)
	}
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		return "", unavailable("database", err, cfg.Secrets()...)
	}
	source, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return "", fmt.Errorf("open migrations: %w", err)
	}
	migrator, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return "", unavailable("database", err, cfg.Secrets()...)
	}
	defer migrator.Close()

	switch command {
	case "up":
		err = migrator.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			return "database schema is already up to date", nil
		}
		if err != nil {
			return "", unavailable("database", err, cfg.Secrets()...)
		}
		return "database migrations applied", nil
	case "down":
		err = migrator.Steps(-1)
		if errors.Is(err, migrate.ErrNoChange) {
			return "no migrations to roll back", nil
		}
		if err != nil {
			return "", unavailable("database", err, cfg.Secrets()...)
		}
		return "rolled back one migration", nil
	case "version":
		version, dirty, versionErr := migrator.Version()
		if errors.Is(versionErr, migrate.ErrNilVersion) {
			return "database schema version: none", nil
		}
		if versionErr != nil {
			return "", unavailable("database", versionErr, cfg.Secrets()...)
		}
		return fmt.Sprintf("database schema version: %d dirty: %t", version, dirty), nil
	default:
		return "", fmt.Errorf("unknown migrate command %q", command)
	}
}

func postgresURL(cfg config.PostgresConfig) string {
	query := url.Values{}
	query.Set("sslmode", cfg.SSLMode)
	query.Set("connect_timeout", "5")
	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     cfg.Host + ":" + strconv.Itoa(cfg.Port),
		Path:     "/" + cfg.Name,
		RawQuery: query.Encode(),
	}
	return u.String()
}
