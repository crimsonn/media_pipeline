package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/db"
	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/transcoder"
)

func main() {
	logger := slog.Default()
	config := config.LoadConfig()

	database, err := db.OpenDatabase(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Error("open database", "err", err)
		os.Exit(1)
	}

	q := queries.New(database)
	numWorkers := config.TranscoderWorkers
	worker := transcoder.NewWorkers(numWorkers, logger, q)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	worker.Start(ctx)
	logger.Info("transcoder workers started", "count", numWorkers)

	<-ctx.Done()
	logger.Info("shutdown signal received, initiating graceful shutdown...")

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()

	shutdownDone := make(chan struct{})
	go func() {
		worker.Stop()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		logger.Info("all workers stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Warn("workers did not finish in time, forcing exit")
	}
}
