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
	"github.com/crimsonn/media_pipeline/pkg/pb"
	"github.com/crimsonn/media_pipeline/services/transcoder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	pb.RegisterTranscoderServiceServer(srv, transcoder.NewTranscoderServer())
	reflection.Register(srv)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		slog.Info("transcoder gRPC server listening", "addr", lis.Addr().String())
		errCh <- srv.Serve(lis)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		stopped := make(chan struct{})
		go func() {
			srv.GracefulStop()
			close(stopped)
		}()
		select {
		case <-stopped:
		case <-time.After(10 * time.Second):
			srv.Stop()
		}
		return nil
	}
}
