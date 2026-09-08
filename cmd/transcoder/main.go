package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/db"
	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/transcoder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	run()
}

func run() {

	srv := grpc.NewServer()
	healthServ := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthServ)

	logger := slog.Default()
	config := config.LoadConfig()
	database, err := db.OpenDatabase(context.Background(), config.DatabaseURL)
	if err != nil {
		logger.Error("open database", "err", err)
		os.Exit(1)
	}
	q := queries.New(database)

	worker := transcoder.NewWorkers(10, logger, q)
	logger.Info("transcoder workers started", "count", 10)

	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	worker.Start(workerCtx)

	done := make(chan struct{})
	go func() {
		worker.Stop()
		close(done)
	}()
	select {
	case <-done:
		slog.Info("all workers stopped")
	case <-time.After(15 * time.Second):
		slog.Warn("wrokers did not finish in time, exiting anyway")
	}
}
