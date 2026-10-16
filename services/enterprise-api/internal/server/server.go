// Package server assembles the HTTP service.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"qingzhi/enterprise-api/internal/auth"
	"qingzhi/enterprise-api/internal/config"
	"qingzhi/enterprise-api/internal/dict"
	"qingzhi/enterprise-api/internal/handler"
	"qingzhi/enterprise-api/internal/identity"
	"qingzhi/enterprise-api/internal/middleware"
	"qingzhi/enterprise-api/internal/monitor"
	"qingzhi/enterprise-api/internal/response"
	"qingzhi/enterprise-api/internal/security"
)

// Dependencies are the runtime collaborators of the HTTP server.
type Dependencies struct {
	Config   config.Config
	Logger   *slog.Logger
	Database handler.Pinger
	Redis    handler.Pinger
	Identity *identity.Service
	Dict     *dict.Service
	Security *security.Service
	Monitor  *monitor.Service
	Tokens   *auth.TokenIssuer
}

// Server is the HTTP API process.
type Server struct {
	cfg        config.Config
	log        *slog.Logger
	engine     *gin.Engine
	httpServer *http.Server
}

// New builds the router and server. It does not listen.
func New(deps Dependencies) *Server {
	switch deps.Config.AppEnv {
	case config.EnvProduction:
		gin.SetMode(gin.ReleaseMode)
	case config.EnvTest:
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()
	engine.Use(middleware.RequestID())
	engine.Use(middleware.AccessLog(deps.Logger))
	engine.Use(middleware.Recovery(deps.Logger))
	engine.Use(middleware.CORS(deps.Config.CORS))
	if deps.Monitor != nil {
		engine.Use(deps.Monitor.Protect())
	}

	health := handler.Health{
		Logger:   deps.Logger,
		Database: deps.Database,
		Redis:    deps.Redis,
		Timeout:  2 * time.Second,
	}
	api := engine.Group("/api/v1")
	api.GET("/health/live", health.Live)
	api.GET("/health/ready", health.Ready)
	api.GET("/openapi.json", handler.OpenAPI)

	authHandler := handler.Auth{
		Logger:   deps.Logger,
		Database: deps.Database,
		Service:  deps.Identity,
		Security: deps.Security,
		Tokens:   deps.Tokens,
	}
	authHandler.RegisterPublic(api)

	protected := api.Group("")
	enforce := deps.Tokens != nil && deps.Identity != nil
	if enforce {
		protected.Use(auth.Guard{Tokens: deps.Tokens, Service: deps.Identity}.Authenticate())
	}
	protected.Use(security.OperateAudit(deps.Security, deps.Logger))
	authHandler.RegisterProtected(protected)
	handler.Identity{
		Logger:   deps.Logger,
		Database: deps.Database,
		Service:  deps.Identity,
		Security: deps.Security,
	}.Register(protected, enforce)
	handler.Dict{
		Logger:   deps.Logger,
		Database: deps.Database,
		Service:  deps.Dict,
	}.Register(protected, enforce)
	handler.Security{
		Logger:   deps.Logger,
		Database: deps.Database,
		Service:  deps.Security,
	}.Register(protected, enforce)
	handler.Monitor{
		Logger:  deps.Logger,
		Service: deps.Monitor,
	}.Register(protected, enforce)

	engine.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "资源不存在", nil)
	})
	engine.NoMethod(func(c *gin.Context) {
		response.Fail(c, http.StatusMethodNotAllowed, response.CodeMethodNotAllowed, "请求方法不允许", nil)
	})

	return &Server{
		cfg:    deps.Config,
		log:    deps.Logger,
		engine: engine,
	}
}

// Engine exposes the router for tests.
func (s *Server) Engine() *gin.Engine {
	return s.engine
}

// Routes returns the registered Gin routes.
func (s *Server) Routes() gin.RoutesInfo {
	return s.engine.Routes()
}

// Run listens until the process is interrupted or the listener fails.
func (s *Server) Run() error {
	s.httpServer = &http.Server{
		Addr:              s.cfg.HTTP.Addr(),
		Handler:           s.engine,
		ReadTimeout:       s.cfg.HTTP.ReadTimeout,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      s.cfg.HTTP.WriteTimeout,
		IdleTimeout:       s.cfg.HTTP.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		s.log.Info("http server listening",
			slog.String("component", "app"),
			slog.String("addr", s.httpServer.Addr),
		)
		err := s.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(stop)

	select {
	case err := <-errCh:
		return err
	case <-stop:
		s.log.Info("http server shutting down", slog.String("component", "app"))
		ctx, cancel := context.WithTimeout(context.Background(), s.cfg.HTTP.ShutdownTimeout)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
}
