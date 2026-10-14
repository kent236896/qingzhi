// Package config loads process configuration from the environment.
package config

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// Config is the validated runtime configuration.
type Config struct {
	AppName  string
	AppEnv   string
	LogLevel string
	HTTP     HTTPConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	CORS     CORSConfig
}

// HTTPConfig controls the public HTTP server.
type HTTPConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// Addr returns the listen address.
func (h HTTPConfig) Addr() string {
	return fmt.Sprintf("%s:%d", h.Host, h.Port)
}

// PostgresConfig holds PostgreSQL connection settings.
type PostgresConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Secrets returns values that must never appear in client responses or logs.
func (p PostgresConfig) Secrets() []string {
	return []string{p.Password, p.User}
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// Addr returns the Redis address.
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// Secrets returns values that must be redacted.
func (r RedisConfig) Secrets() []string {
	if r.Password == "" {
		return nil
	}
	return []string{r.Password}
}

// CORSConfig is the allowed browser origin list.
type CORSConfig struct {
	AllowedOrigins []string
}

// Load reads environment variables and validates them.
// An empty ENV_FILE disables dotenv loading. When ENV_FILE is unset, the first
// existing file among .env, ../.env and ../../.env is loaded without overriding
// variables that are already set.
func Load() (Config, error) {
	env, err := loadEnvFiles()
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		AppName:  env.getenv("APP_NAME", "enterprise-api"),
		AppEnv:   strings.TrimSpace(env.getenv("APP_ENV", "")),
		LogLevel: strings.ToLower(env.getenv("LOG_LEVEL", "info")),
	}

	var missing []string
	require := func(key string) string {
		value := strings.TrimSpace(env.getenv(key, ""))
		if value == "" {
			missing = append(missing, key)
		}
		return value
	}

	httpPortRaw := require("HTTP_PORT")
	require("CORS_ALLOWED_ORIGINS")
	require("POSTGRES_HOST")
	postgresPortRaw := require("POSTGRES_PORT")
	require("POSTGRES_USER")
	require("POSTGRES_PASSWORD")
	require("POSTGRES_DB")
	require("POSTGRES_SSLMODE")
	require("REDIS_HOST")
	redisPortRaw := require("REDIS_PORT")
	require("APP_ENV")

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	cfg.HTTP.Host = env.getenv("HTTP_HOST", "0.0.0.0")
	if cfg.HTTP.Port, err = parsePort("HTTP_PORT", httpPortRaw); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ReadTimeout, err = parseDuration("HTTP_READ_TIMEOUT", env.getenv("HTTP_READ_TIMEOUT", ""), "15s"); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.WriteTimeout, err = parseDuration("HTTP_WRITE_TIMEOUT", env.getenv("HTTP_WRITE_TIMEOUT", ""), "15s"); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.IdleTimeout, err = parseDuration("HTTP_IDLE_TIMEOUT", env.getenv("HTTP_IDLE_TIMEOUT", ""), "60s"); err != nil {
		return Config{}, err
	}
	if cfg.HTTP.ShutdownTimeout, err = parseDuration("HTTP_SHUTDOWN_TIMEOUT", env.getenv("HTTP_SHUTDOWN_TIMEOUT", ""), "10s"); err != nil {
		return Config{}, err
	}

	cfg.Postgres.Host = strings.TrimSpace(env.getenv("POSTGRES_HOST", ""))
	if cfg.Postgres.Port, err = parsePort("POSTGRES_PORT", postgresPortRaw); err != nil {
		return Config{}, err
	}
	cfg.Postgres.User = env.getenv("POSTGRES_USER", "")
	cfg.Postgres.Password = env.getenv("POSTGRES_PASSWORD", "")
	cfg.Postgres.Name = strings.TrimSpace(env.getenv("POSTGRES_DB", ""))
	cfg.Postgres.SSLMode = strings.TrimSpace(env.getenv("POSTGRES_SSLMODE", ""))
	if cfg.Postgres.MaxOpenConns, err = parsePositiveInt("POSTGRES_MAX_OPEN_CONNS", env.getenv("POSTGRES_MAX_OPEN_CONNS", ""), "25"); err != nil {
		return Config{}, err
	}
	if cfg.Postgres.MaxIdleConns, err = parsePositiveInt("POSTGRES_MAX_IDLE_CONNS", env.getenv("POSTGRES_MAX_IDLE_CONNS", ""), "5"); err != nil {
		return Config{}, err
	}
	if cfg.Postgres.ConnMaxLifetime, err = parseDuration("POSTGRES_CONN_MAX_LIFETIME", env.getenv("POSTGRES_CONN_MAX_LIFETIME", ""), "30m"); err != nil {
		return Config{}, err
	}

	cfg.Redis.Host = strings.TrimSpace(env.getenv("REDIS_HOST", ""))
	if cfg.Redis.Port, err = parsePort("REDIS_PORT", redisPortRaw); err != nil {
		return Config{}, err
	}
	cfg.Redis.Password = env.getenv("REDIS_PASSWORD", "")
	if cfg.Redis.DB, err = parseNonNegativeInt("REDIS_DB", env.getenv("REDIS_DB", ""), "0"); err != nil {
		return Config{}, err
	}

	cfg.CORS.AllowedOrigins = splitCSV(env.getenv("CORS_ALLOWED_ORIGINS", ""))
	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (cfg Config) validate() error {
	switch cfg.AppEnv {
	case EnvDevelopment, EnvTest, EnvProduction:
	default:
		return fmt.Errorf("APP_ENV must be development, test, or production")
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "warning", "error":
	default:
		return fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}
	switch cfg.Postgres.SSLMode {
	case "disable", "require", "verify-ca", "verify-full":
	default:
		return fmt.Errorf("POSTGRES_SSLMODE must be disable, require, verify-ca, or verify-full")
	}
	if cfg.Postgres.MaxIdleConns > cfg.Postgres.MaxOpenConns {
		return fmt.Errorf("POSTGRES_MAX_IDLE_CONNS cannot exceed POSTGRES_MAX_OPEN_CONNS")
	}
	if len(cfg.CORS.AllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	wildcard := false
	for _, origin := range cfg.CORS.AllowedOrigins {
		if origin == "*" {
			wildcard = true
			continue
		}
		parsed, err := url.Parse(origin)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS contains an invalid origin")
		}
	}
	if wildcard && len(cfg.CORS.AllowedOrigins) != 1 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS cannot mix * with explicit origins")
	}
	if cfg.AppEnv == EnvProduction {
		if cfg.Postgres.Password == "change-me" {
			return fmt.Errorf("POSTGRES_PASSWORD must be changed when APP_ENV is production")
		}
		if cfg.Postgres.SSLMode == "disable" {
			return fmt.Errorf("POSTGRES_SSLMODE cannot be disable when APP_ENV is production")
		}
	}
	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parsePort(key, value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%s must be a port between 1 and 65535", key)
	}
	return port, nil
}

func parsePositiveInt(key, value, fallback string) (int, error) {
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return number, nil
}

func parseNonNegativeInt(key, value, fallback string) (int, error) {
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return number, nil
}

func parseDuration(key, value, fallback string) (time.Duration, error) {
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	parsed, err := time.ParseDuration(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}
