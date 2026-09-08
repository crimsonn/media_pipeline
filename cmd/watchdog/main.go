package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/pkg/pb"
	"github.com/crimsonn/media_pipeline/services/watchdog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	transcoderAddr := config.GetEnvString("TRANSCODER_ADDR", ":50051")
	if err := run(transcoderAddr); err != nil {
		slog.Error("watchdog failed", "err", err)
		os.Exit(1)
	}
}

func run(addr string) error {
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	conn, err := grpc.NewClient(addr, opts...)
	if err != nil {
		log.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()
	// client := pb.NewTranscoderServiceClient(conn)
	// resp, err := client.TranscodeVideo(context.Background(), &pb.TranscodeRequest{
	// 	FileId:            "123",
	// 	SourceFilePath:    "123",
	// 	OutputDirectory:   "123",
	// 	TargetResolutions: []string{"123", "123", "123"},
	// })
	// if err != nil {
	// 	return err
	// }

	watchDogFolder := config.GetEnvString("WATCHDOG_FOLDER", "./watch")
	watchDogDelay := config.GetEnvString("WATCHDOG_DELAY", "5s")
	delay, err := time.ParseDuration(watchDogDelay)
	if err != nil {
		return fmt.Errorf("cant parse duration from delay: %w", err)
	}
	numWorkers := config.GetEnvInt("NUM_WORKERS", 10)
	queueCapacity := config.GetEnvInt("QUEUE_CAPACITY", 100)
	transcoder := pb.NewTranscoderServiceClient(conn)
	orchestrator := watchdog.NewOrchestrator(numWorkers, queueCapacity, delay, watchDogFolder, transcoder)
	return orchestrator.Start()
}
