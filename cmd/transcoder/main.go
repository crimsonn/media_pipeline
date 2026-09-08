package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/transcoder"
	transcoderGrpc "github.com/crimsonn/media_pipeline/internal/transport/grpc"
	"github.com/crimsonn/media_pipeline/pkg/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	addr := config.GetEnvString("TRANSCODER_ADDR", ":50051")
	if err := run(addr); err != nil {
		slog.Error("transcoder failed", "err", err)
		os.Exit(1)
	}
}

func run(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	srv := grpc.NewServer()
	healthServ := health.NewServer()
	healthpb.RegisterHealthServer(srv, healthServ)

	logger := slog.Default()
	worker := transcoder.NewWorkers(10, logger)
	transcoderGrpcServer := transcoderGrpc.NewTranscoderGRPCServer(worker, logger)
	pb.RegisterTranscoderServiceServer(srv, transcoderGrpcServer)

	healthServ.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	workerCtx, cancelWorkers := context.WithCancel(context.Background())
	defer cancelWorkers()
	worker.Start(workerCtx)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve(lis)
	}()
	var serveErr error
	select {
	case serveErr = <-errCh:
	case <-ctx.Done():
		slog.Info("shutting down")
	}

	stopped := make(chan struct{})
	go func() {
		srv.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-time.After(10 * time.Second):
		srv.Stop()
		<-stopped
	}

	cancelWorkers()

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
	return serveErr
}
