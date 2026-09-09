package watcher

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"path"
	"sync"
	"syscall"
	"time"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	filesChecked    map[string]bool
	mu              sync.Mutex
	db              *queries.Queries
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
	db *queries.Queries,
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
		filesChecked:    make(map[string]bool),
		db:              db,
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
				existing, err := o.db.GetPendingFile(ctx, task.fileName)
				if err == nil {
					o.logger.Info("file already exists, skipping", "file_id", existing.ID)
					return
				}

				if !errors.Is(err, pgx.ErrNoRows) {
					o.logger.Error("failed to get pending file", "error", err)
					task.status = Failed
					task.error = err
					return
				}
				results, err := o.db.CreatePendingFile(ctx, task.fileName)
				if err != nil {
					o.logger.Error("failed to create pending file", "error", err)
					task.status = Failed
					task.error = err
					return
				}
				o.logger.Info("pending file created", "file_id", results.ID)

				task.status = Processing
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
			o.walkDirectory(ctx)
		case <-ctx.Done():
			o.logger.Info("polling stopped")
			tick.Stop()
			return
		}
	}
}

func (o *Orchestrator) walkDirectory(ctx context.Context) {
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
		if alreadyChecked {
			continue
		}

		existing, err := o.db.GetPendingFile(ctx, file.Name())
		if err == nil {
			delete(o.filesChecked, file.Name())
			o.logger.Info("file already exists, skipping", "file_id", existing.ID)
			o.mu.Unlock()
			continue
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			delete(o.filesChecked, file.Name())
			o.logger.Error("failed to get pending file", "error", err)
			o.mu.Unlock()
			continue
		}
		submitted := o.Submit(&Task{
			id:            uuid.New(),
			status:        Pending,
			fileName:      file.Name(),
			lastCheckedAt: time.Now(),
		})
		o.mu.Unlock()

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
