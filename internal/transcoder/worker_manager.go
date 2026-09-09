package transcoder

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
)

type WorkerManager struct {
	logger            *slog.Logger
	workers           map[string]*Worker
	store             JobStore
	numWorkers        int
	nextWorkerID      int
	mu                sync.Mutex
	stopOnce          sync.Once
	stop              chan struct{}
	heartbeatResponse chan string
	fromMonitor       chan WorkerNotification
	toMonitor         chan WorkerNotification
}

type Notification struct {
	workerId     string
	workerStatus WorkerState
	action       string
}

func NewWorkerManager(
	logger *slog.Logger,
	store JobStore,
	numWorkers int,
	heartbeatResponse chan string,
) *WorkerManager {
	return &WorkerManager{
		logger:            logger,
		workers:           make(map[string]*Worker),
		store:             store,
		numWorkers:        numWorkers,
		nextWorkerID:      numWorkers,
		stop:              make(chan struct{}),
		heartbeatResponse: heartbeatResponse,
		fromMonitor:       make(chan WorkerNotification),
		toMonitor:         make(chan WorkerNotification),
	}
}

func (wm *WorkerManager) CreateWorker(ctx context.Context, id string) *Worker {
	heartbeat := make(chan struct{})
	stop := make(chan struct{})
	w := NewWorker(id, wm.logger, wm.store, heartbeat, wm.heartbeatResponse, stop)
	wm.mu.Lock()
	wm.workers[id] = w
	wm.mu.Unlock()
	go w.Start(ctx)
	go w.StartProcessing(ctx)
	return w
}

func (wm *WorkerManager) Start(ctx context.Context) {
	wm.Notify(ctx)
	wm.logger.Info("shutting down worker manager")
}

func (wm *WorkerManager) Notify(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case notification := <-wm.fromMonitor:
			if notification.action == "replace" {
				wm.replaceWorker(ctx, notification)
			}
		}
	}
}

func (wm *WorkerManager) replaceWorker(ctx context.Context, notification WorkerNotification) {
	oldID := ""
	if notification.worker != nil {
		oldID = notification.worker.id
	}
	wm.logger.Info("replacing worker", "id", oldID)

	wm.mu.Lock()
	old := wm.workers[oldID]
	delete(wm.workers, oldID)
	id := fmt.Sprintf("worker-%d", wm.nextWorkerID)
	wm.nextWorkerID++
	wm.mu.Unlock()

	if old != nil {
		old.Kill()
	}

	worker := wm.CreateWorker(ctx, id)
	unregisterNotification := WorkerNotification{
		worker: &WorkerHeartbeat{
			id:        oldID,
			heartbeat: nil,
		},
		action: "unregister",
	}
	if notification.worker != nil {
		unregisterNotification.worker.heartbeat = notification.worker.heartbeat
	}
	registerNotification := WorkerNotification{
		worker: &WorkerHeartbeat{
			id:        worker.id,
			heartbeat: worker.heartbeat,
		},
		action: "register",
	}
	wm.NotifyHealthMonitor(ctx, unregisterNotification)
	wm.NotifyHealthMonitor(ctx, registerNotification)
}

func (wm *WorkerManager) NotifyHealthMonitor(ctx context.Context, notification WorkerNotification) {
	select {
	case wm.toMonitor <- notification:
	case <-ctx.Done():
	}
}

func (wm *WorkerManager) workerCount() int {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	return len(wm.workers)
}

func (wm *WorkerManager) lookupWorker(id string) *Worker {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	return wm.workers[id]
}

func (wm *WorkerManager) Stop() {
	wm.stopOnce.Do(func() {
		close(wm.stop)
		wm.mu.Lock()
		workers := make([]*Worker, 0, len(wm.workers))
		for _, w := range wm.workers {
			workers = append(workers, w)
		}
		wm.mu.Unlock()
		for _, w := range workers {
			w.Kill()
		}
		wm.logger.Info("worker manager stopped")
	})
}
