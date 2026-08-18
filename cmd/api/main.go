// Package main is the entrypoint for the Nandi API.
//
//	@title						Nandi API
//	@version					1.0
//	@description				Multi-tenant Customer Engagement Platform API (Africa-first).
//	@host						localhost:8080
//	@BasePath					/
//	@schemes					http
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer" followed by a space and the access token.
//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						X-API-Key
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"github.com/yourorg/nandi/internal/config"
	"github.com/yourorg/nandi/internal/database"
	"github.com/yourorg/nandi/internal/handlers"
	"github.com/yourorg/nandi/internal/realtime"
	"github.com/yourorg/nandi/internal/utils"

	_ "github.com/yourorg/nandi/docs"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := utils.NewLogger(cfg.Log.Level, cfg.App.Env)
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.NewPostgres(cfg.Database, log)
	if err != nil {
		log.Warn().Err(err).Msg("postgres unavailable; starting without a database connection")
	}
	defer func() {
		if closeErr := database.ClosePostgres(db); closeErr != nil {
			log.Error().Err(closeErr).Msg("close postgres")
		}
	}()
	if db != nil {
		if err := database.AutoMigrate(db, log); err != nil {
			log.Fatal().Err(err).Msg("automigrate")
		}
	}

	rdb, err := database.NewRedis(cfg.Redis, log)
	if err != nil {
		log.Warn().Err(err).Msg("redis unavailable; starting without a redis connection")
	}
	defer func() {
		if closeErr := database.CloseRedis(rdb); closeErr != nil {
			log.Error().Err(closeErr).Msg("close redis")
		}
	}()

	appCtx, appCancel := context.WithCancel(context.Background())
	defer appCancel()

	hub := realtime.NewHub(rdb, log)
	go hub.Run(appCtx)

	deps := handlers.Dependencies{DB: db, Redis: rdb, Config: cfg, Log: log, Hub: hub}
	svc := handlers.BuildServices(deps)
	svc.Campaigns.StartWorker(appCtx, 5*time.Second)

	router := handlers.NewRouter(log, deps, svc)
	server := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info().Str("addr", cfg.Server.Addr()).Msg("nandi api listening")
		if serveErr := server.ListenAndServe(); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Fatal().Err(serveErr).Msg("listen")
		}
	}()

	waitForShutdown(server, appCancel, log, cfg)
}

func waitForShutdown(server *http.Server, cancel context.CancelFunc, log zerolog.Logger, cfg *config.Config) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	log.Info().Str("signal", sig.String()).Msg("shutting down")
	cancel()

	ctx, stop := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer stop()

	if err := server.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("graceful shutdown failed")
		return
	}

	log.Info().Msg("server stopped")
}
