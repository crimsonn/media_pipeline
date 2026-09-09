package transcoder

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
)

type WorkerManager struct {
	logger              *slog.Logger
	workers             map[string]*Worker
	queries             *queries.Queries
	numWorkers          int
	mu                  sync.Mutex
	wg                  sync.WaitGroup
	stop                chan struct{}
	heartbeatResponse   chan string
	notificationChannel chan WorkerNotification
}

type Notification struct {
	workerId     string
	workerStatus WorkerState
	action       string
}

func NewWorkerManager(
	logger *slog.Logger,
	queries *queries.Queries,
	numWorkers int,
	heartbeatResponse chan string,
) *WorkerManager {
	return &WorkerManager{
		logger:            logger,
		workers:           make(map[string]*Worker),
		queries:           queries,
		numWorkers:        numWorkers,
		stop:              make(chan struct{}),
		heartbeatResponse: heartbeatResponse,
	}
}

func randRange(min, max int) int {
	return rand.IntN(max-min) + min
}

func (wm *WorkerManager) CreateWorker(ctx context.Context, id string) *Worker {
	heartbeat := make(chan struct{})
	stop := make(chan struct{})
	w := NewWorker(id, wm.logger, wm.queries, heartbeat, wm.heartbeatResponse, stop)
	wm.workers[id] = w
	go w.Start(ctx)
	go w.StartProcessing(ctx)
	return w
}

func (wm *WorkerManager) Start(ctx context.Context) {
	go wm.Notify(ctx)
	<-ctx.Done()
	wm.logger.Info("shutting down worker manager")
	for _, w := range wm.workers {
		close(w.heartbeat)
	}
	wm.wg.Wait()
	wm.logger.Info("worker manager stopped")
}

func (wm *WorkerManager) Notify(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case notification := <-wm.notificationChannel:
			if notification.action == "replace" {
				wm.logger.Info("replacing worker", "id", notification.worker.id)
				worker := wm.CreateWorker(ctx, fmt.Sprintf("worker-%d", wm.numWorkers+1))
				registerNotification := WorkerNotification{
					worker: &WorkerHeartbeat{
						id:        worker.id,
						heartbeat: worker.heartbeat,
					},
					action: "register",
				}
				unregisterNotification := WorkerNotification{
					worker: &WorkerHeartbeat{
						id:        notification.worker.id,
						heartbeat: notification.worker.heartbeat,
					},
					action: "unregister",
				}
				wm.NotifyHealthMonitor(ctx, unregisterNotification)
				wm.NotifyHealthMonitor(ctx, registerNotification)
			}
		}
	}
}

func (wm *WorkerManager) NotifyHealthMonitor(ctx context.Context, notification WorkerNotification) {
	wm.notificationChannel <- notification
}

func (wm *WorkerManager) Stop(ctx context.Context) {
	close(wm.stop)
	for _, w := range wm.workers {
		close(w.heartbeat)
	}
	wm.wg.Wait()
	wm.logger.Info("worker manager stopped")
}
