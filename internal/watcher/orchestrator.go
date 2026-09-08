package watcher

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/crimsonn/media_pipeline/pkg/pb"
	"github.com/google/uuid"
)

type TaskStatus string

const (
	Pending    TaskStatus = "PENDING"
	Waiting    TaskStatus = "WAITING"
	Processing TaskStatus = "PROCESSING"
	Failed     TaskStatus = "FAILED"
	Completed  TaskStatus = "COMPLETED"
)

type Orchestrator struct {
	numWorkers      int
	tasks           chan *Task
	wg              sync.WaitGroup
	logger          *slog.Logger
	watchDirectory  string
	outputDirectory string
	transcoder      pb.TranscoderServiceClient
	filesChecked    map[string]bool
	mu              sync.Mutex
}

type Task struct {
	id            uuid.UUID
	status        TaskStatus
	fileName      string
	lastCheckedAt time.Time
	tries         int
	result        any
	error         error
}

func NewOrchestrator(
	numWorkers int,
	queueCapacity int,
	pollingInterval time.Duration,
	watchDirectory string,
	outputDirectory string,
	transcoder pb.TranscoderServiceClient,
) *Orchestrator {
	logger := slog.Default()
	logger.Info("starting workers", "count", numWorkers)
	logger.Info("queue capacity", "capacity", queueCapacity)
	logger.Info("watch directory", "directory", watchDirectory)
	logger.Info("output directory", "directory", outputDirectory)
	return &Orchestrator{
		logger:          logger,
		tasks:           make(chan *Task, queueCapacity),
		numWorkers:      numWorkers,
		watchDirectory:  watchDirectory,
		outputDirectory: outputDirectory,
		transcoder:      transcoder,
		filesChecked:    make(map[string]bool),
	}
}

func (o *Orchestrator) Start() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	for i := 1; i <= o.numWorkers; i++ {
		o.wg.Add(1)
		go o.worker(ctx, i)
	}
	go o.StartPolling(ctx)
	<-ctx.Done()
	o.logger.Info("shutting down orchestrator")
	o.Stop()
	return nil
}

func (o *Orchestrator) Submit(task *Task) bool {
	select {
	case o.tasks <- task:
		o.logger.Info("task submitted", "id", task.id, "fileName", task.fileName)
		return true
	default:
		o.logger.Error("task queue is full", "id", task.id, "fileName", task.fileName)
		return false
	}
}

func (o *Orchestrator) worker(ctx context.Context, id int) {
	defer o.wg.Done()
	for {
		select {
		case <-ctx.Done():
			o.logger.Info("worker shutting down")
			return
		case task, ok := <-o.tasks:
			if !ok {
				return
			}
			o.processTask(ctx, id, task)
		}
	}

}

func (o *Orchestrator) processTask(ctx context.Context, workerid int, task *Task) {
	defer func() {
		o.mu.Lock()
		delete(o.filesChecked, task.fileName)
		defer o.mu.Unlock()
	}()

	o.logger.Info("worker processing task", "workerId", workerid, "taskId", task.id, "fileName", task.fileName)
	task.status = Waiting
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			o.logger.Warn("cancellation received during file watching", "taskId", task.id)
			return
		case <-tick.C:
			filePath := path.Join(o.watchDirectory, task.fileName)
			fileInfo, err := os.Stat(filePath)
			if err != nil {
				task.status = Failed
				task.error = err
				o.logger.Error("file not found", "error", err)
				return
			}

			if fileInfo.ModTime().Unix() == task.lastCheckedAt.Unix() {
				task.tries++
			}
			if fileInfo.ModTime().After(task.lastCheckedAt) {
				task.lastCheckedAt = fileInfo.ModTime()
				task.tries = 0
			}
			if fileInfo.ModTime().Before(task.lastCheckedAt) {
				// Means either the transcoding failed and didnt remove the file or something else went wrong
				task.tries++
			}

			if task.tries >= 2 {
				o.logger.Info("task exceeded max tries, submitting for transcoding", "file_id", task.id.String())
				stream, err := o.transcoder.TranscodeVideo(ctx, &pb.TranscodeVideoRequest{
					FileName:        task.fileName,
					FileId:          task.id.String(),
					SourceFilePath:  filePath,
					OutputDirectory: o.outputDirectory,
					Resolutions: []*pb.TranscodeResolution{
						{
							Name:     "1080p",
							Width:    1920,
							Height:   1080,
							VideoBps: 5_000_000,
							AudioBps: 128_000,
						},
						{
							Name:     "720p",
							Width:    1280,
							Height:   720,
							VideoBps: 2_000_000,
							AudioBps: 128_000,
						},
					},
				})
				if err != nil {
					o.logger.Error("transcoding failed or was cancelled", "error", err)
					task.status = Failed
					return
				}

				task.status = Processing
				for {
					progress, err := stream.Recv()
					if err == io.EOF {
						break
					}

					if err != nil {
						o.logger.Error("transcode stream failed", "error", err)
						task.status = Failed
						task.error = err
						return
					}

					switch progress.State {
					case pb.TranscodeState_TRANSCODE_STATE_IN_PROGRESS:
						o.logger.Info("transcoding in progress", "taskId", task.id, "percentComplete", progress.PercentComplete)
					case pb.TranscodeState_TRANSCODE_STATE_COMPLETED:
						o.logger.Info("transcoding completed", "taskId", task.id)
						task.status = Completed
					case pb.TranscodeState_TRANSCODE_STATE_FAILED:
						o.logger.Error("transcoding failed", "taskId", task.id, "error", progress.ErrorMessage)
						task.status = Failed
						task.error = errors.New(progress.ErrorMessage)
						return
					}
				}

				return
			}
		}
	}
}

func (o *Orchestrator) StartPolling(ctx context.Context) {
	tick := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-tick.C:
			o.walkDirectory()
		case <-ctx.Done():
			o.logger.Info("polling stopped")
			tick.Stop()
			return
		}
	}
}

func (o *Orchestrator) walkDirectory() {
	files, err := os.ReadDir(o.watchDirectory)
	if err != nil {
		o.logger.Error("failed to read directory", "error", err)
		return
	}
	for _, file := range files {
		o.mu.Lock()
		alreadyChecked := o.filesChecked[file.Name()]
		if !alreadyChecked {
			o.filesChecked[file.Name()] = true
		}
		o.mu.Unlock()
		if alreadyChecked {
			continue
		}
		submitted := o.Submit(&Task{
			id:            uuid.New(),
			status:        Pending,
			fileName:      file.Name(),
			lastCheckedAt: time.Now(),
		})

		if !submitted {
			o.mu.Lock()
			delete(o.filesChecked, file.Name())
			o.mu.Unlock()
		}
	}
}

func (o *Orchestrator) Stop() {
	close(o.tasks)
	o.wg.Wait()
}
