package transcoder

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
)

type Orchestrator struct {
	numWorkers    int
	logger        *slog.Logger
	queries       *queries.Queries
	stop          chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
	workerManager *WorkerManager
	healthMonitor *HealthMonitor
}

func NewOrchestrator(
	numWorkers int,
	logger *slog.Logger,
	queries *queries.Queries,
) *Orchestrator {
	orchestrator := &Orchestrator{
		numWorkers: numWorkers,
		logger:     logger,
		queries:    queries,
		stop:       make(chan struct{}),
		wg:         sync.WaitGroup{},
	}
	return orchestrator
}

func (o *Orchestrator) Start(ctx context.Context) {
	heartbeatResponse := make(chan string)
	notify := make(chan WorkerNotification)
	o.healthMonitor = NewHealthMonitor(o.logger, ctx, 2, heartbeatResponse)
	o.healthMonitor.workerManagerChannel = notify
	o.workerManager = NewWorkerManager(o.logger, o.queries, o.numWorkers, heartbeatResponse)
	o.workerManager.notificationChannel = notify
	for i := range o.numWorkers {
		worker := o.workerManager.CreateWorker(ctx, fmt.Sprintf("worker-%d", i))
		o.healthMonitor.RegisterWorker(&WorkerHeartbeat{
			id:        worker.id,
			heartbeat: worker.heartbeat,
		})
	}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		o.workerManager.Start(ctx)
	}()
	go func() {
		defer wg.Done()
		o.healthMonitor.Start(ctx)
	}()

	<-ctx.Done()
	o.logger.Info("shutting down orchestrator")
	wg.Wait()
	o.workerManager.Stop()
	o.healthMonitor.Stop()
	o.logger.Info("orchestrator stopped")
}

func (o *Orchestrator) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()
	select {
	case <-o.stop:
	default:
		close(o.stop)
	}
}
