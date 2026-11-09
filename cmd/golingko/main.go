package main

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golingko/internal/api"
	"golingko/internal/config"
	"golingko/internal/transit"
	"golingko/internal/worker"
	"golingko/pkg/database"
	"golingko/pkg/logger"
)

func main() {
	cfg := config.Load()

	appLogger, logCleanup, err := logger.New(logger.Config{
		Driver: logger.Driver(cfg.Logging.Driver),
		Level:  logger.Level(cfg.Logging.Verbosity),
		Format: logger.Format(cfg.Logging.Format),
	})
	if err != nil {
		slog.Error("Failed to initialize configured logging driver", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = logCleanup()
	}()
	slog.SetDefault(appLogger)

	appLogger.Info("Configuration loaded",
		"server_addr", cfg.ServerAddr,
		"db_host", cfg.DB.Host,
		"db_port", cfg.DB.Port,
		"db_name", cfg.DB.DBName,
		"log_driver", cfg.Logging.Driver,
		"log_verbosity", cfg.Logging.Verbosity,
		"log_format", cfg.Logging.Format,
	)

	var db *sql.DB
	var repo transit.TransitRepository
	conn, err := database.InitDB(cfg.DB)
	if err != nil {
		appLogger.Warn("Failed to connect to database at startup; running in degraded mode",
			"error", err,
			"hint", "Run 'docker compose up -d' or 'make up' to start PostgreSQL and Redis",
		)
	} else {
		db = conn
		defer db.Close()
		repo = transit.NewRepository(db)
	}

	var cardSvc transit.CardService
	var stationSvc transit.StationService
	var routeSvc transit.RouteService
	var journeyOrch transit.JourneyOrchestrator
	var ledgerSvc transit.LedgerService

	if repo != nil {
		cardSvc = transit.NewCardService(repo)
		stationSvc = transit.NewStationService(repo)
		routeSvc = transit.NewRouteService(repo)
		journeyOrch = transit.NewJourneyOrchestrator(repo, cfg.MinTapBalance)
		ledgerSvc = transit.NewLedgerService(repo)
	}

	sweeper := worker.NewSweeper(repo, 5*time.Minute, cfg.JourneyExpiryHours, appLogger)
	sweeper.Start()
	defer sweeper.Stop()

	engine, err := api.ParseRouterEngine(cfg.HTTPRouter)
	if err != nil {
		appLogger.Warn("Unknown router engine configured; falling back to net/http",
			"engine", cfg.HTTPRouter,
			"error", err,
		)
		engine = api.RouterEngineNetHTTP
	}
	appLogger.Info("Configured HTTP router engine", "engine", engine)

	handler := api.NewHandler(cardSvc, stationSvc, routeSvc, journeyOrch, ledgerSvc, db, appLogger)
	router, err := api.SetupRouterWithEngine(engine, handler, appLogger)
	if err != nil {
		appLogger.Error("Failed to initialize HTTP router", "error", err)
		os.Exit(1)
	}
	server := api.NewServer(cfg, router)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		appLogger.Info("Golingko HTTP server listening", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	appLogger.Info("Shutdown signal received, shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("Graceful shutdown failed, forcing close", "error", err)
	} else {
		appLogger.Info("Server stopped cleanly")
	}
}
