package database

import (
	"context"
	"strings"
	"testing"
	"time"

	"qingzhi/enterprise-api/internal/config"
)

func TestPingFailureDoesNotLeakPassword(t *testing.T) {
	const password = "super-secret-db-password"
	db, err := OpenPostgres(config.PostgresConfig{
		Host:            "127.0.0.1",
		Port:            1,
		User:            "enterprise",
		Password:        password,
		Name:            "enterprise",
		SSLMode:         "disable",
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	})
	if err != nil {
		t.Fatalf("OpenPostgres() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if db.sql.Stats().MaxOpenConnections != 2 {
		t.Fatalf("max open connections = %d", db.sql.Stats().MaxOpenConnections)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err = db.Ping(ctx)
	if err == nil {
		t.Fatal("expected ping to fail")
	}
	assertNoSecret(t, err, password)
}

func TestUnavailableRedactsPassword(t *testing.T) {
	const password = "super-secret-db-password"
	err := unavailable("database", errWithSecret(password), password)
	assertNoSecret(t, err, password)
	connErr, ok := err.(*ConnectionError)
	if !ok {
		t.Fatalf("error type = %T", err)
	}
	if !strings.Contains(connErr.Diagnostic(), "[REDACTED]") {
		t.Fatalf("diagnostic = %s", connErr.Diagnostic())
	}
}

func TestMigrateUpReportsUnavailableWithoutSecret(t *testing.T) {
	const password = "super-secret-db-password"
	_, err := Migrate(config.PostgresConfig{
		Host:     "127.0.0.1",
		Port:     1,
		User:     "enterprise",
		Password: password,
		Name:     "enterprise",
		SSLMode:  "disable",
	}, "up")
	if err == nil {
		t.Fatal("expected migrate to fail when postgres is closed")
	}
	if strings.Contains(err.Error(), "unknown driver") {
		t.Fatalf("postgres driver is not registered: %v", err)
	}
	assertNoSecret(t, err, password)
	if _, err := Migrate(config.PostgresConfig{}, "sideways"); err == nil {
		t.Fatal("expected unknown command error")
	}
}

func errWithSecret(password string) error {
	return &secretError{text: "pq: password authentication failed for " + password + " postgres://" + password + "@db/enterprise"}
}

type secretError struct{ text string }

func (e *secretError) Error() string { return e.text }

func assertNoSecret(t *testing.T, err error, secret string) {
	t.Helper()
	text := err.Error()
	if strings.Contains(text, secret) || strings.Contains(text, "postgres://") {
		t.Fatalf("error leaked secret or dsn: %s", text)
	}
	if connErr, ok := err.(*ConnectionError); ok && strings.Contains(connErr.Diagnostic(), secret) {
		t.Fatalf("diagnostic leaked secret: %s", connErr.Diagnostic())
	}
}
