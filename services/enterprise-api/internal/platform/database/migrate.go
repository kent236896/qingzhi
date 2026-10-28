package database

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/platform/redact"
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
			return "", migrateFailure(err, cfg.Secrets()...)
		}
		return "database migrations applied", nil
	case "down":
		err = migrator.Steps(-1)
		if errors.Is(err, migrate.ErrNoChange) {
			return "no migrations to roll back", nil
		}
		if err != nil {
			return "", migrateFailure(err, cfg.Secrets()...)
		}
		return "rolled back one migration", nil
	case "version":
		version, dirty, versionErr := migrator.Version()
		if errors.Is(versionErr, migrate.ErrNilVersion) {
			return "database schema version: none", nil
		}
		if versionErr != nil {
			return "", migrateFailure(versionErr, cfg.Secrets()...)
		}
		return fmt.Sprintf("database schema version: %d dirty: %t", version, dirty), nil
	default:
		return "", fmt.Errorf("unknown migrate command %q", command)
	}
}

// migrateFailure keeps dirty/SQL failures readable; only real driver/connect issues become ConnectionError.
func migrateFailure(err error, secrets ...string) error {
	var dirty *migrate.ErrDirty
	if errors.As(err, &dirty) {
		return fmt.Errorf(
			"dirty database schema at version %d (previous migration failed); fix schema then set schema_migrations to version %d dirty=false and re-run up",
			dirty.Version,
			dirty.Version-1,
		)
	}
	msg := redact.Apply(err.Error(), secrets...)
	// Connection/driver style failures stay opaque; migration SQL errors stay explicit.
	if looksLikeConnectionFailure(msg) {
		return unavailable("database", err, secrets...)
	}
	return fmt.Errorf("%s", msg)
}

func looksLikeConnectionFailure(msg string) bool {
	lower := strings.ToLower(msg)
	for _, needle := range []string{
		"connection refused",
		"connect: ",
		"no such host",
		"i/o timeout",
		"timeout expired",
		"server closed the connection",
		"driver: bad connection",
		"pq: sorry, too many clients",
		"dial tcp",
	} {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
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
