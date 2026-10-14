package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"qingzhi/enterprise-api/internal/config"
)

// Postgres is a PostgreSQL connection pool.
type Postgres struct {
	gorm     *gorm.DB
	sql      *sql.DB
	user     string
	password string
}

// OpenPostgres creates a pool. It does not require the server to be reachable.
func OpenPostgres(cfg config.PostgresConfig) (*Postgres, error) {
	db, err := gorm.Open(postgres.Open(dsn(cfg)), &gorm.Config{
		Logger:               gormlogger.Default.LogMode(gormlogger.Silent),
		DisableAutomaticPing: true,
	})
	if err != nil {
		return nil, unavailable("database", err, cfg.Secrets()...)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, unavailable("database", err, cfg.Secrets()...)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	return &Postgres{gorm: db, sql: sqlDB, user: cfg.User, password: cfg.Password}, nil
}

// Ping checks that PostgreSQL can serve a connection.
func (p *Postgres) Ping(ctx context.Context) error {
	if p == nil || p.sql == nil {
		return &ConnectionError{Dependency: "database", diagnostic: "database handle is not initialized"}
	}
	if err := p.sql.PingContext(ctx); err != nil {
		return unavailable("database", err, p.password, p.user)
	}
	return nil
}

// Close releases the pool.
func (p *Postgres) Close() error {
	if p == nil || p.sql == nil {
		return nil
	}
	return p.sql.Close()
}

func dsn(cfg config.PostgresConfig) string {
	timeout := 5
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC connect_timeout=%d",
		quoteDSN(cfg.Host),
		cfg.Port,
		quoteDSN(cfg.User),
		quoteDSN(cfg.Password),
		quoteDSN(cfg.Name),
		quoteDSN(cfg.SSLMode),
		timeout,
	)
}

func quoteDSN(value string) string {
	if value == "" || !strings.ContainsAny(value, " '\\") {
		return value
	}
	escaped := strings.ReplaceAll(value, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `'`, `\'`)
	return "'" + escaped + "'"
}
