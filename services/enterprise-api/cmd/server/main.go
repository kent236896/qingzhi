package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/dict"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/knowledge"
	"qingzhi/enterprise-api/internal/modelcfg"
	"qingzhi/enterprise-api/internal/monitor"
	"qingzhi/enterprise-api/internal/platform/database"
	"qingzhi/enterprise-api/internal/platform/logger"
	"qingzhi/enterprise-api/internal/platform/redis"
	"qingzhi/enterprise-api/internal/security"
	"qingzhi/enterprise-api/internal/server"
	"qingzhi/enterprise-api/internal/settings"
	"qingzhi/enterprise-api/internal/weknora"
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

	tokens, err := auth.NewTokenIssuer(cfg.Auth.JWTSecret, cfg.Auth.TokenTTL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration error: %s\n", err)
		return 1
	}

	var databasePinger interface{ Ping(context.Context) error }
	var identityService *identity.Service
	var knowledgeService *knowledge.Service
	var modelService *modelcfg.Service
	var dictService *dict.Service
	var securityService *security.Service
	var settingsService *settings.Service
	var alertStore monitor.AlertStore
	db, err := database.OpenPostgres(cfg.Postgres)
	if err != nil {
		log.Error("postgres initialization failed",
			slog.String("component", "app"),
			slog.String("error", err.Error()),
		)
	} else {
		databasePinger = db
		identityService = identity.NewService(identity.NewGormStore(db.GORM()))
		remoteClient := weknora.NewHTTPClient(cfg.WeKnora.BaseURL, cfg.WeKnora.APIKey, cfg.WeKnora.Timeout)
		settingsService = settings.NewService(settings.NewGormStore(db.GORM()), log)
		if conn, connErr := settingsService.Ensure(context.Background(), cfg.WeKnora.BaseURL, cfg.WeKnora.APIKey); connErr != nil {
			log.Warn("weknora settings unavailable", slog.String("component", "app"), slog.String("error", connErr.Error()))
		} else {
			cfg.WeKnora.BaseURL = conn.BaseURL
			cfg.WeKnora.APIKey = conn.APIKey
			remoteClient.Configure(conn.BaseURL, conn.APIKey)
		}
		var remote weknora.Client = remoteClient
		if cfg.WeKnora.Enabled() {
			info, infoErr := remote.GetSystemInfo(context.Background())
			if infoErr != nil {
				log.Warn("weknora is not reachable", slog.String("component", "app"), slog.String("error", infoErr.Error()))
			} else {
				log.Info("weknora is reachable", slog.String("component", "app"), slog.String("version", info.Version), slog.String("edition", info.Edition))
			}
			if caps, capsErr := remote.GetDeploymentCapabilities(context.Background()); capsErr == nil {
				log.Info("weknora capabilities loaded", slog.String("component", "app"), slog.String("edition", caps.Edition), slog.Int("count", len(caps.Capabilities)))
			}
		}
		knowledgeService = knowledge.NewService(knowledge.NewGormStore(db.GORM()), identityService, remote, cfg.Knowledge, cfg.WeKnora, log)
		modelService = modelcfg.NewService(modelcfg.NewGormStore(db.GORM()), modelcfg.NewHTTPRemote(remoteClient), cfg.WeKnora, log)
		knowledgeService.SetModelDirectory(modelService)
		settingsService.SetHooks(settings.Hooks{
			Apply: func(baseURL, apiKey string) {
				remoteClient.Configure(baseURL, apiKey)
				modelService.SetConnection(baseURL, apiKey)
				knowledgeService.SetConnection(baseURL, apiKey)
			},
			Probe: func(ctx context.Context) (string, error) {
				info, err := remoteClient.GetSystemInfo(ctx)
				if err != nil {
					return "", err
				}
				return info.Version, nil
			},
		})
		dictService = dict.NewService(dict.NewGormStore(db.GORM()))
		securityService = security.NewService(security.NewGormStore(db.GORM()))
		alertStore = monitor.NewGormStore(db.GORM())
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

	monitorService := monitor.New(cfg.Monitor, log, databasePinger, redisClient, alertStore)
	monitorCtx, monitorCancel := context.WithCancel(context.Background())
	defer monitorCancel()
	go monitorService.Run(monitorCtx)

	srv := server.New(server.Dependencies{
		Config:    cfg,
		Logger:    log,
		Database:  databasePinger,
		Redis:     redisClient,
		Identity:  identityService,
		Knowledge: knowledgeService,
		Models:    modelService,
		Dict:      dictService,
		Security:  securityService,
		Settings:  settingsService,
		Monitor:   monitorService,
		Tokens:    tokens,
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
