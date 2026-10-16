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
	Auth     AuthConfig
	Monitor  MonitorConfig
}

// AuthConfig holds JWT signing settings. Tokens are self-contained; Redis is not used for sessions.
type AuthConfig struct {
	JWTSecret   string
	TokenTTL    time.Duration
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

// MonitorConfig controls in-process metrics, alerts, and request protection.
// Rate limiting applies inside this process only.
type MonitorConfig struct {
	Window         time.Duration
	ErrorCount     int
	TimeoutCount   int
	P95            time.Duration
	CPUPercent     int
	MemoryPercent  int
	DiskPercent    int
	RequestTimeout time.Duration
	MaxBodyBytes   int64
	RateLimitRPS   int
	RateLimitBurst int
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
	require("JWT_SECRET")

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
	cfg.Auth.JWTSecret = env.getenv("JWT_SECRET", "")
	if cfg.Auth.TokenTTL, err = parseDuration("JWT_EXPIRE", env.getenv("JWT_EXPIRE", ""), "24h"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.Window, err = parseDuration("MONITOR_WINDOW", env.getenv("MONITOR_WINDOW", ""), "5m"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.ErrorCount, err = parseNonNegativeInt("MONITOR_ERROR_COUNT", env.getenv("MONITOR_ERROR_COUNT", ""), "20"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.TimeoutCount, err = parseNonNegativeInt("MONITOR_TIMEOUT_COUNT", env.getenv("MONITOR_TIMEOUT_COUNT", ""), "1"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.P95, err = parseDuration("MONITOR_P95", env.getenv("MONITOR_P95", ""), "2s"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.CPUPercent, err = parsePercent("MONITOR_CPU_PERCENT", env.getenv("MONITOR_CPU_PERCENT", ""), "90"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.MemoryPercent, err = parsePercent("MONITOR_MEMORY_PERCENT", env.getenv("MONITOR_MEMORY_PERCENT", ""), "90"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.DiskPercent, err = parsePercent("MONITOR_DISK_PERCENT", env.getenv("MONITOR_DISK_PERCENT", ""), "90"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.RequestTimeout, err = parseDuration("HTTP_REQUEST_TIMEOUT", env.getenv("HTTP_REQUEST_TIMEOUT", ""), "10s"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.MaxBodyBytes, err = parsePositiveInt64("HTTP_MAX_BODY_BYTES", env.getenv("HTTP_MAX_BODY_BYTES", ""), "1048576"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.RateLimitRPS, err = parseNonNegativeInt("HTTP_RATE_LIMIT_RPS", env.getenv("HTTP_RATE_LIMIT_RPS", ""), "50"); err != nil {
		return Config{}, err
	}
	if cfg.Monitor.RateLimitBurst, err = parseNonNegativeInt("HTTP_RATE_LIMIT_BURST", env.getenv("HTTP_RATE_LIMIT_BURST", ""), "100"); err != nil {
		return Config{}, err
	}
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
	if len(cfg.Auth.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if cfg.Monitor.Window < 30*time.Second || cfg.Monitor.Window > 24*time.Hour {
		return fmt.Errorf("MONITOR_WINDOW must be between 30s and 24h")
	}
	if cfg.Monitor.RequestTimeout > cfg.HTTP.WriteTimeout {
		return fmt.Errorf("HTTP_REQUEST_TIMEOUT cannot exceed HTTP_WRITE_TIMEOUT")
	}
	if cfg.Monitor.MaxBodyBytes < 1024 || cfg.Monitor.MaxBodyBytes > 32<<20 {
		return fmt.Errorf("HTTP_MAX_BODY_BYTES must be between 1024 and 33554432")
	}
	if cfg.Monitor.RateLimitRPS > 10000 {
		return fmt.Errorf("HTTP_RATE_LIMIT_RPS must be at most 10000")
	}
	if cfg.Monitor.RateLimitRPS > 0 && (cfg.Monitor.RateLimitBurst < cfg.Monitor.RateLimitRPS || cfg.Monitor.RateLimitBurst > 20000) {
		return fmt.Errorf("HTTP_RATE_LIMIT_BURST must be between the rate and 20000")
	}
	if cfg.Monitor.ErrorCount > 100000 || cfg.Monitor.TimeoutCount > 100000 {
		return fmt.Errorf("monitor thresholds are too large")
	}
	if cfg.AppEnv == EnvProduction {
		if cfg.Postgres.Password == "change-me" {
			return fmt.Errorf("POSTGRES_PASSWORD must be changed when APP_ENV is production")
		}
		if cfg.Postgres.SSLMode == "disable" {
			return fmt.Errorf("POSTGRES_SSLMODE cannot be disable when APP_ENV is production")
		}
		if cfg.Auth.JWTSecret == "change-me-jwt-secret" || cfg.Auth.JWTSecret == "change-me" {
			return fmt.Errorf("JWT_SECRET must be changed when APP_ENV is production")
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

func parsePercent(key, value, fallback string) (int, error) {
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	number, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || number < 0 || number > 100 {
		return 0, fmt.Errorf("%s must be an integer from 0 to 100", key)
	}
	return number, nil
}

func parsePositiveInt64(key, value, fallback string) (int64, error) {
	if strings.TrimSpace(value) == "" {
		value = fallback
	}
	number, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
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
