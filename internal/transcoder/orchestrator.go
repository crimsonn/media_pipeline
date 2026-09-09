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
	go o.workerManager.Start(ctx)
	go o.healthMonitor.Start(ctx)
	<-ctx.Done()
	o.logger.Info("shutting down orchestrator")
	o.workerManager.Stop(ctx)
	o.healthMonitor.Stop(ctx)
	o.wg.Wait()
	o.logger.Info("orchestrator stopped")

}

func (o *Orchestrator) Stop(ctx context.Context) {
	close(o.stop)
}
