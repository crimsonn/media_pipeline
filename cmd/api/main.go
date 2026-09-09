package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crimsonn/media_pipeline/internal/api/jobs"
	"github.com/crimsonn/media_pipeline/internal/api/pending"
	"github.com/crimsonn/media_pipeline/internal/api/playback"
	"github.com/crimsonn/media_pipeline/internal/api/transcoder"
	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/db"
	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/server"
)

func main() {
	logger := slog.Default()
	config := config.LoadConfig()
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second)
	database, err := db.OpenDatabase(dbCtx, config.DatabaseURL)
	dbCancel()
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	q := queries.New(database)
	transcoderService := transcoder.NewService(q)
	transcoderHandler := transcoder.NewHandler(transcoderService)
	pendingService := pending.NewService(database, config)
	pendingHandler := pending.NewHandler(pendingService)
	jobsService := jobs.NewService(q)
	jobsHandler := jobs.NewHandler(jobsService)
	playbackService := playback.NewService(config)
	playbackHandler := playback.NewHandler(playbackService)
	router := server.SetupRouter(
		logger,
		config,
		transcoderHandler,
		pendingHandler,
		jobsHandler,
		playbackHandler,
	)
	srv := &http.Server{
		Addr:    config.APIAddr + ":" + config.APIPort,
		Handler: router,
	}

	go func() {
		logger.Info("Starting server", "addr", srv.Addr, "port", config.APIPort, "env", config.Environment)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	logger.Info("Received shutdown signal", "signal", sig)
	logger.Info("Shutting down server gracefully...")
	shutdownTimeout := time.Duration(10 * time.Second)
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Failed to shutdown server", "error", err)
		return
	}

	logger.Info("Server shutdown complete")
}
