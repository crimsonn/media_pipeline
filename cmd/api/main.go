package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crimsonn/media_pipeline/internal/api/transcoder"
	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/db"
	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/server"
	"github.com/crimsonn/media_pipeline/pkg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	logger := slog.Default()
	config := config.LoadConfig()
	transcoderAddr := config.TranscoderAddr
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	conn, err := grpc.NewClient(transcoderAddr, opts...)
	if err != nil {
		logger.Error("Failed to connect to transcoder", "error", err)
		os.Exit(1)
	}
	defer conn.Close()
	trClient := pb.NewTranscoderServiceClient(conn)
	healthClient := healthpb.NewHealthClient(conn)
	database, err := db.OpenDatabase()
	if err != nil {
		logger.Error("Failed to open database", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	q := queries.New(database)
	transcoderService := transcoder.NewService(q)
	transcoderHandler := transcoder.NewHandler(transcoderService)
	router := server.SetupRouter(
		logger,
		config,
		trClient,
		healthClient,
		database,
		transcoderHandler,
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
