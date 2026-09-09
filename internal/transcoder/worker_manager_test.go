package transcoder

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/jackc/pgx/v5"
)

type idleJobStore struct{}

func (idleJobStore) ClaimJobTask(context.Context) (queries.ClaimJobTaskRow, error) {
	return queries.ClaimJobTaskRow{}, pgx.ErrNoRows
}

func (idleJobStore) FailJobTask(context.Context, queries.FailJobTaskParams) error { return nil }

func (idleJobStore) MarkJobRunning(context.Context, int64) error { return nil }

func (idleJobStore) CompleteJobTask(context.Context, int64) error { return nil }

func (idleJobStore) JobTaskStats(context.Context, int64) (queries.JobTaskStatsRow, error) {
	return queries.JobTaskStatsRow{}, nil
}

func (idleJobStore) UpdateJobProgress(context.Context, queries.UpdateJobProgressParams) error {
	return nil
}

func (idleJobStore) FailJob(context.Context, queries.FailJobParams) error { return nil }

func (idleJobStore) ListJobRenditions(context.Context, int64) ([]queries.ListJobRenditionsRow, error) {
	return nil, nil
}

func (idleJobStore) CompleteJob(context.Context, int64) error { return nil }

func newTestWorkerManager(t *testing.T, numWorkers int) (*WorkerManager, context.Context) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	logger := slog.New(slog.DiscardHandler)
	wm := NewWorkerManager(logger, idleJobStore{}, numWorkers, make(chan string, 8))
	t.Cleanup(wm.Stop)
	return wm, ctx
}

func TestNewWorkerManager(t *testing.T) {
	wm, _ := newTestWorkerManager(t, 3)

	if wm.numWorkers != 3 {
		t.Fatalf("numWorkers = %d, want 3", wm.numWorkers)
	}
	if wm.nextWorkerID != 3 {
		t.Fatalf("nextWorkerID = %d, want 3", wm.nextWorkerID)
	}
	if wm.workers == nil {
		t.Fatal("workers map is nil")
	}
	if wm.fromMonitor == nil || wm.toMonitor == nil {
		t.Fatal("notify channels are nil")
	}
	if wm.workerCount() != 0 {
		t.Fatalf("worker count = %d, want 0", wm.workerCount())
	}
}

func TestCreateWorkerRegistersAndHeartbeats(t *testing.T) {
	wm, ctx := newTestWorkerManager(t, 1)
	w := wm.CreateWorker(ctx, "worker-0")

	if got := wm.lookupWorker("worker-0"); got != w {
		t.Fatal("worker was not registered")
	}

	select {
	case w.heartbeat <- struct{}{}:
	case <-time.After(time.Second):
		t.Fatal("timed out sending heartbeat")
	}

	select {
	case id := <-wm.heartbeatResponse:
		if id != "worker-0" {
			t.Fatalf("heartbeat id = %q, want worker-0", id)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for heartbeat response")
	}
}

func TestStopKillsWorkersAndIsIdempotent(t *testing.T) {
	wm, ctx := newTestWorkerManager(t, 1)
	w := wm.CreateWorker(ctx, "worker-0")

	wm.Stop()
	wm.Stop()

	select {
	case <-w.stop:
	default:
		t.Fatal("worker stop channel was not closed")
	}
}

func TestStartReturnsOnCancel(t *testing.T) {
	wm, ctx := newTestWorkerManager(t, 1)
	ctx, cancel := context.WithCancel(ctx)

	done := make(chan struct{})
	go func() {
		wm.Start(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Start did not return after cancel")
	}
}

func TestNotifyReplaceSpawnsWorkerAndForwards(t *testing.T) {
	wm, ctx := newTestWorkerManager(t, 1)
	old := wm.CreateWorker(ctx, "worker-0")

	go wm.Notify(ctx)

	wm.fromMonitor <- WorkerNotification{
		worker: &WorkerHeartbeat{
			id:        old.id,
			heartbeat: old.heartbeat,
		},
		action: "replace",
	}

	unregister := recvNotification(t, wm.toMonitor, "unregister")
	if unregister.worker.id != "worker-0" {
		t.Fatalf("unregister id = %q, want worker-0", unregister.worker.id)
	}

	register := recvNotification(t, wm.toMonitor, "register")
	if register.worker.id != "worker-1" {
		t.Fatalf("register id = %q, want worker-1", register.worker.id)
	}
	if register.worker.heartbeat == nil {
		t.Fatal("register heartbeat is nil")
	}

	if wm.lookupWorker("worker-0") != nil {
		t.Fatal("old worker is still registered")
	}
	replacement := wm.lookupWorker("worker-1")
	if replacement == nil {
		t.Fatal("replacement worker was not registered")
	}

	select {
	case <-old.stop:
	case <-time.After(time.Second):
		t.Fatal("old worker was not killed")
	}

	select {
	case replacement.heartbeat <- struct{}{}:
	case <-time.After(time.Second):
		t.Fatal("timed out sending heartbeat to replacement")
	}
	select {
	case id := <-wm.heartbeatResponse:
		if id != "worker-1" {
			t.Fatalf("replacement heartbeat id = %q, want worker-1", id)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for replacement heartbeat")
	}
}

func TestNotifyHealthMonitorSendsToMonitor(t *testing.T) {
	wm, ctx := newTestWorkerManager(t, 1)
	n := WorkerNotification{
		worker: &WorkerHeartbeat{id: "worker-0"},
		action: "register",
	}

	done := make(chan struct{})
	go func() {
		wm.NotifyHealthMonitor(ctx, n)
		close(done)
	}()

	got := recvNotification(t, wm.toMonitor, "register")
	if got.worker.id != "worker-0" {
		t.Fatalf("id = %q, want worker-0", got.worker.id)
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("NotifyHealthMonitor did not return")
	}
}

func TestCreateWorkerConcurrent(t *testing.T) {
	wm, ctx := newTestWorkerManager(t, 8)
	const n = 8

	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			wm.CreateWorker(ctx, fmt.Sprintf("worker-%d", i))
		}(i)
	}
	wg.Wait()

	if got := wm.workerCount(); got != n {
		t.Fatalf("worker count = %d, want %d", got, n)
	}
}

func recvNotification(t *testing.T, ch <-chan WorkerNotification, wantAction string) WorkerNotification {
	t.Helper()
	select {
	case n := <-ch:
		if n.action != wantAction {
			t.Fatalf("action = %q, want %q", n.action, wantAction)
		}
		if n.worker == nil {
			t.Fatal("notification worker is nil")
		}
		return n
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s notification", wantAction)
		return WorkerNotification{}
	}
}
