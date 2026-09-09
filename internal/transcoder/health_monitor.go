package transcoder

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Design document:
// The health monitor is responsible for monitoring the health of the workers.
// It will be started as a goroutine and will run until the stop channel is closed.
// It will use the heartbeat channel to send a heartbeat to the workers.
// It will collect the response from the workers and check if they are alive.
// If a worker is not responding, it will be considered dead/unhealthy and a new worker will be spawned in its place.

type WorkerHeartbeat struct {
	id        string
	heartbeat chan struct{}
}

type HealthMonitor struct {
	logger               *slog.Logger
	ctx                  context.Context
	workers              map[string]*WorkerHeartbeat
	consecutiveLosses    map[string]int
	maxLosses            int
	mu                   sync.Mutex
	heartbeatResponse    chan string
	workerManagerChannel chan WorkerNotification
}

type WorkerNotification struct {
	worker *WorkerHeartbeat
	action string
}

func NewHealthMonitor(
	logger *slog.Logger,
	ctx context.Context,
	maxLosses int,
	heartbeatResponse chan string,
) *HealthMonitor {
	return &HealthMonitor{
		logger:            logger,
		ctx:               ctx,
		workers:           make(map[string]*WorkerHeartbeat),
		consecutiveLosses: make(map[string]int),
		maxLosses:         maxLosses,
		heartbeatResponse: heartbeatResponse,
	}
}

func (h *HealthMonitor) Start(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		h.CheckHealth()
	}()
	go func() {
		defer wg.Done()
		h.Notify()
	}()
	wg.Wait()
	h.logger.Info("shutting down health monitor")
}

func (h *HealthMonitor) RegisterWorker(worker *WorkerHeartbeat) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, w := range h.workers {
		if w.id == worker.id {
			h.logger.Warn("worker already registered", "id", worker.id)
			return
		}
	}
	h.workers[worker.id] = worker
	h.logger.Info("registered worker", "id", worker.id)
}

func (h *HealthMonitor) UnregisterWorker(worker *WorkerHeartbeat) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.workers, worker.id)
	h.logger.Info("unregistered worker", "id", worker.id)
}

func (h *HealthMonitor) Notify() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case notification := <-h.workerManagerChannel:
			switch notification.action {
			case "register":
				h.RegisterWorker(&WorkerHeartbeat{
					id:        notification.worker.id,
					heartbeat: notification.worker.heartbeat,
				})
			case "unregister":
				h.mu.Lock()
				delete(h.consecutiveLosses, notification.worker.id)
				h.mu.Unlock()
				h.UnregisterWorker(&WorkerHeartbeat{
					id:        notification.worker.id,
					heartbeat: notification.worker.heartbeat,
				})
			}
		}
	}
}

func (h *HealthMonitor) CheckHealth() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			pending := make(map[string]bool)
			for _, w := range h.workers {
				pending[w.id] = true
				select {
				case w.heartbeat <- struct{}{}:
				default:
				}
			}

			timeout := time.NewTimer(1 * time.Second)
			collecting := true

			for collecting {
				select {
				case <-h.ctx.Done():
					timeout.Stop()
					return
				case workerId := <-h.heartbeatResponse:
					h.logger.Info("health monitor received heartbeat from", "id", workerId)
					delete(pending, workerId)
					if _, ok := h.consecutiveLosses[workerId]; ok {
						h.logger.Info("worker recovered", "id", workerId)
						delete(h.consecutiveLosses, workerId)
					}
				case <-timeout.C:
					collecting = false
				}
			}

			for workerId := range pending {
				h.logger.Warn("worker missed heartbeat", "id", workerId)
				h.consecutiveLosses[workerId]++
				if h.consecutiveLosses[workerId] > h.maxLosses {
					notification := WorkerNotification{
						worker: &WorkerHeartbeat{
							id:        workerId,
							heartbeat: nil,
						},
						action: "replace",
					}
					select {
					case h.workerManagerChannel <- notification:
					case <-h.ctx.Done():
						return
					}
					delete(pending, workerId)
					h.logger.Warn("worker lost, spawning a new one at the same position", "retries", h.consecutiveLosses[workerId], "maxLosses", h.maxLosses)
				}
			}
		}
	}
}

func (h *HealthMonitor) Stop() {
	h.logger.Info("health monitor stopped")
}
