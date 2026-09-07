package watchdog

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/crimsonn/media_pipeline/pkg/pb"
	"google.golang.org/grpc"
)

type Watcher struct {
	WatchDirectory string
	Delay          time.Duration
	Client         *grpc.ClientConn
	Transcoder     pb.TranscoderServiceClient
	lastChecked    map[string]int64
}

func NewWatcher(
	watchDirectory string,
	delay time.Duration,
	client *grpc.ClientConn,
) *Watcher {
	transcoder := pb.NewTranscoderServiceClient(client)
	return &Watcher{
		WatchDirectory: watchDirectory,
		Delay:          delay,
		Client:         client,
		Transcoder:     transcoder,
	}
}

func (s *Watcher) Start() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	errCh := make(chan error, 1)
	go func() {
		slog.Info("watch dog is up and running")
		s.startPooling(ctx)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutting down watch dog")
		stopped := make(chan struct{})
		close(stopped)
	}
	return nil
}

func (s *Watcher) startPooling(ctx context.Context) error {
	tick := time.NewTicker(s.Delay)
	for {
		select {
		case <-tick.C:
			fmt.Printf("Ticking\n")
		case <-ctx.Done():
			return fmt.Errorf("shutting down pooling")
		}
	}
}
