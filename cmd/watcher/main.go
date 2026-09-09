package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/db"
	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/watcher"
)

func main() {
	cfg := config.LoadConfig()
	database, err := db.OpenDatabase(context.Background(), cfg.DatabaseURL)
	if err != nil {
		slog.Error("open database", "err", err)
		os.Exit(1)
	}
	defer database.Close()
	if _, err := os.Stat(cfg.WatchDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(cfg.WatchDirectory, 0o755); err != nil {
			slog.Error("create watch directory", "err", err)
			os.Exit(1)
		}
	}
	if _, err := os.Stat(cfg.OutputDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(cfg.OutputDirectory, 0o755); err != nil {
			slog.Error("create output directory", "err", err)
			os.Exit(1)
		}
	}

	numWorkers := config.GetEnvInt("NUM_WORKERS", 10)
	queueCapacity := config.GetEnvInt("QUEUE_CAPACITY", 100)
	q := queries.New(database)
	orchestrator := watcher.NewOrchestrator(numWorkers, queueCapacity, cfg.PollingInterval, cfg.WatchDirectory, cfg.OutputDirectory, q)
	if err := orchestrator.Start(); err != nil {
		slog.Error("start orchestrator", "err", err)
		os.Exit(1)
	}
}
