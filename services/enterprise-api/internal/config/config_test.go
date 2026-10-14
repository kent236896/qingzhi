package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLoadFromEnvironment(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENV_FILE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AppEnv != EnvTest || cfg.HTTP.Port != 8080 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
	if cfg.HTTP.ReadTimeout != 15*time.Second || cfg.Postgres.MaxOpenConns != 25 {
		t.Fatalf("defaults were not applied: %+v", cfg)
	}
	if cfg.Postgres.Password != "change-me" {
		t.Fatal("password was not loaded")
	}
}

func TestLoadMissingRequired(t *testing.T) {
	clearRequiredEnv(t)
	t.Setenv("ENV_FILE", "")
	t.Setenv("APP_ENV", EnvTest)

	_, err := Load()
	if err == nil {
		t.Fatal("expected missing configuration error")
	}
	if !strings.Contains(err.Error(), "POSTGRES_PASSWORD") {
		t.Fatalf("error = %v, want POSTGRES_PASSWORD", err)
	}
}

func TestLoadRejectsUnsafeProductionSettings(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ENV_FILE", "")
	t.Setenv("APP_ENV", EnvProduction)
	t.Setenv("POSTGRES_SSLMODE", "require")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "POSTGRES_PASSWORD") {
		t.Fatalf("error = %v", err)
	}
}

func TestLoadEnvFileDoesNotOverrideProcess(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.env")
	content := "" +
		"APP_ENV=production\n" +
		"APP_NAME=from-file\n" +
		"HTTP_PORT=9090\n" +
		"CORS_ALLOWED_ORIGINS=http://localhost:5173\n" +
		"POSTGRES_HOST=127.0.0.1\n" +
		"POSTGRES_PORT=5432\n" +
		"POSTGRES_USER=enterprise\n" +
		"POSTGRES_PASSWORD=from-file-secret\n" +
		"POSTGRES_DB=enterprise\n" +
		"POSTGRES_SSLMODE=disable\n" +
		"REDIS_HOST=127.0.0.1\n" +
		"REDIS_PORT=6379\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	clearRequiredEnv(t)
	t.Setenv("ENV_FILE", path)
	t.Setenv("APP_ENV", EnvTest)
	t.Setenv("POSTGRES_PASSWORD", "process-secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.AppEnv != EnvTest {
		t.Fatalf("APP_ENV = %s, want process value", cfg.AppEnv)
	}
	if cfg.Postgres.Password != "process-secret" {
		t.Fatal("process password was overridden by the env file")
	}
	if cfg.HTTP.Port != 9090 || cfg.AppName != "from-file" {
		t.Fatalf("file values were not used: %+v", cfg)
	}
}

func TestParseEnvFileQuotesAndComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.env")
	content := "# comment\n\nexport APP_ENV=test\nPOSTGRES_PASSWORD=\"change-me\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	values, err := parseEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if values["APP_ENV"] != "test" || values["POSTGRES_PASSWORD"] != "change-me" {
		t.Fatalf("values = %#v", values)
	}
}

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_ENV", EnvTest)
	t.Setenv("HTTP_PORT", "8080")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")
	t.Setenv("POSTGRES_HOST", "127.0.0.1")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_USER", "enterprise")
	t.Setenv("POSTGRES_PASSWORD", "change-me")
	t.Setenv("POSTGRES_DB", "enterprise")
	t.Setenv("POSTGRES_SSLMODE", "disable")
	t.Setenv("REDIS_HOST", "127.0.0.1")
	t.Setenv("REDIS_PORT", "6379")
}

func clearRequiredEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{
		"APP_ENV",
		"APP_NAME",
		"LOG_LEVEL",
		"HTTP_HOST",
		"HTTP_PORT",
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_SHUTDOWN_TIMEOUT",
		"CORS_ALLOWED_ORIGINS",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
		"POSTGRES_DB",
		"POSTGRES_SSLMODE",
		"POSTGRES_MAX_OPEN_CONNS",
		"POSTGRES_MAX_IDLE_CONNS",
		"POSTGRES_CONN_MAX_LIFETIME",
		"REDIS_HOST",
		"REDIS_PORT",
		"REDIS_PASSWORD",
		"REDIS_DB",
	} {
		t.Setenv(key, "")
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
}
