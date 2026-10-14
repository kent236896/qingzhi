package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/platform/database"
	"qingzhi/enterprise-api/internal/platform/logger"
	"qingzhi/enterprise-api/internal/platform/redis"
	"qingzhi/enterprise-api/internal/server"
)

func main() {
	os.Exit(run())
}

func run() int {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %s\n", err)
		return 1
	}
	log := logger.New(cfg.LogLevel, cfg.AppName)
	log.Info("configuration loaded",
		slog.String("component", "app"),
		slog.String("app_env", cfg.AppEnv),
		slog.String("http_addr", cfg.HTTP.Addr()),
		slog.String("postgres_host", cfg.Postgres.Host),
		slog.Int("postgres_port", cfg.Postgres.Port),
		slog.String("postgres_db", cfg.Postgres.Name),
		slog.String("redis_host", cfg.Redis.Host),
		slog.Int("redis_port", cfg.Redis.Port),
	)

	var databasePinger interface{ Ping(context.Context) error }
	db, err := database.OpenPostgres(cfg.Postgres)
	if err != nil {
		log.Error("postgres initialization failed",
			slog.String("component", "app"),
			slog.String("error", err.Error()),
		)
	} else {
		databasePinger = db
		defer func() {
			if closeErr := db.Close(); closeErr != nil {
				log.Error("postgres close failed", slog.String("component", "app"), slog.String("error", closeErr.Error()))
			}
		}()
		logDependency(log, "postgres", db.Ping)
	}

	redisClient := redis.Open(cfg.Redis)
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Error("redis close failed", slog.String("component", "app"), slog.String("error", closeErr.Error()))
		}
	}()
	logDependency(log, "redis", redisClient.Ping)

	srv := server.New(server.Dependencies{
		Config:   cfg,
		Logger:   log,
		Database: databasePinger,
		Redis:    redisClient,
	})
	if err := srv.Run(); err != nil {
		log.Error("server stopped", slog.String("component", "app"), slog.String("error", err.Error()))
		return 1
	}
	return 0
}

func logDependency(log *slog.Logger, name string, ping func(context.Context) error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := ping(ctx)
	if err == nil {
		log.Info(name+" is reachable", slog.String("component", "app"))
		return
	}
	detail := ""
	var connErr *database.ConnectionError
	if errors.As(err, &connErr) {
		detail = connErr.Diagnostic()
	}
	log.Warn(name+" is not reachable",
		slog.String("component", "app"),
		slog.String("error", err.Error()),
		slog.String("detail", detail),
	)
}
