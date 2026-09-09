package transcoder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/domain"
	"github.com/crimsonn/media_pipeline/internal/utils"
	"github.com/jackc/pgx/v5"
)

type WorkerState int

const (
	WorkerStateIdle WorkerState = iota
	WorkerStateActive
)

type Worker struct {
	id                string
	status            WorkerState
	logger            *slog.Logger
	queries           *queries.Queries
	heartbeat         chan struct{}
	heartbeatResponse chan string
	stop              chan struct{}
}

func NewWorker(
	id string,
	logger *slog.Logger,
	queries *queries.Queries,
	heartbeat chan struct{},
	heartbeatResponse chan string,
	stop chan struct{},
) *Worker {
	return &Worker{
		id:                id,
		status:            WorkerStateIdle,
		logger:            logger,
		queries:           queries,
		heartbeat:         heartbeat,
		heartbeatResponse: heartbeatResponse,
		stop:              stop,
	}
}

func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("worker", "id", w.id, "started", true)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stop:
				return
			case _, ok := <-w.heartbeat:
				if !ok {
					return
				}
				select {
				case w.heartbeatResponse <- w.id:
				case <-ctx.Done():
					return
				case <-w.stop:
					return
				}
			}
		}
	}()
	select {
	case <-ctx.Done():
	case <-w.stop:
	}
	w.logger.Info("worker", "id", w.id, "stopped", true)
}

func (w *Worker) StartProcessing(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Received a stop signal")
			return
		case <-w.stop:
			w.logger.Info("Shutting down worker", "id=", w.id)
			return
		case <-ticker.C:
			task, err := w.queries.ClaimJobTask(ctx)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				w.logger.Error("failed to claim job task", "worker", w.id, "error", err)
				continue
			}
			w.processTask(ctx, w.id, task)
		}
	}
}

func (w *Worker) processTask(ctx context.Context, workerID string, task queries.ClaimJobTaskRow) {
	fail := func(msg string) {
		if err := w.queries.FailJobTask(ctx, queries.FailJobTaskParams{
			ID:           task.TaskID,
			ErrorMessage: &msg,
		}); err != nil {
			w.logger.Error("failed to mark task failed", "task", task.TaskID, "error", err)
		}
		if err := w.finalizeJob(ctx, task); err != nil {
			w.logger.Error("failed to finalize job", "job", task.JobID, "error", err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			fail(fmt.Sprintf("worker %s panicked: %v", workerID, r))
		}
	}()

	if err := w.queries.MarkJobRunning(ctx, task.JobID); err != nil {
		fail(fmt.Sprintf("mark job running: %v", err))
	}
	stem := utils.FileTrimSuffix(task.FileName)
	variantDir := filepath.Join(task.OutputDir, stem, task.RenditionName)
	if err := os.MkdirAll(variantDir, 0o755); err != nil {
		fail(fmt.Sprintf("failed to create variant directory: %v", err))
		return
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", utils.RenditionArgs(task.SourcePath, variantDir, int(task.HlsSegmentTime), task)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fail(fmt.Sprintf("ffmpeg failed: %v", err))
		return
	}

	if err := w.queries.CompleteJobTask(ctx, task.TaskID); err != nil {
		w.logger.Error("failed to complete task", "task", task.TaskID, "error", err)
		return
	}

	if err := w.finalizeJob(ctx, task); err != nil {
		w.logger.Error("failed to finalize job", "job", task.JobID, "error", err)
	}
}

func (w *Worker) finalizeJob(ctx context.Context, task queries.ClaimJobTaskRow) error {
	stats, err := w.queries.JobTaskStats(ctx, task.JobID)
	if err != nil {
		return fmt.Errorf("task stats: %w", err)
	}

	total := stats.Pending + stats.Running + stats.Completed + stats.Failed
	if total > 0 {
		if err := w.queries.UpdateJobProgress(ctx, queries.UpdateJobProgressParams{
			ID:              task.JobID,
			ProgressPercent: float32(stats.Completed) / float32(total) * 100,
		}); err != nil {
			w.logger.Error("failed to update job progress", "job", task.JobID, "error", err)
		}
	}

	if stats.Pending > 0 || stats.Running > 0 {
		return nil
	}

	if stats.Failed > 0 {
		// TODO: We should re-insert the job into the queue so that it can be retried
		// at some poiint in the future.
		msg := fmt.Sprintf("%d rendition(s) failed", stats.Failed)
		return w.queries.FailJob(ctx, queries.FailJobParams{
			ID:           task.JobID,
			ErrorMessage: &msg,
		})
	}

	rows, err := w.queries.ListJobRenditions(ctx, task.JobID)
	if err != nil {
		return fmt.Errorf("list renditions: %w", err)
	}

	renditions := make([]domain.Rendition, 0, len(rows))
	for _, row := range rows {
		streamIndex := 0
		if row.StreamIndex != nil {
			streamIndex = int(*row.StreamIndex)
		}

		renditions = append(renditions, domain.Rendition{
			ID:           row.ID,
			Name:         row.Name,
			Width:        int(row.Width),
			Height:       int(row.Height),
			VideoBitrate: int(row.VideoBitrate),
			AudioBitrate: int(row.AudioBitrate),
			VideoCodec:   row.VideoCodec,
			AudioCodec:   row.AudioCodec,
			StreamIndex:  streamIndex,
		})
	}

	masterDir := filepath.Join(task.OutputDir, utils.FileTrimSuffix(task.FileName))
	if err := WriteMaster(masterDir, renditions); err != nil {
		msg := fmt.Sprintf("write master playlist: %v", err)
		return w.queries.FailJob(ctx, queries.FailJobParams{
			ID:           task.JobID,
			ErrorMessage: &msg,
		})
	}

	if err := w.queries.CompleteJob(ctx, task.JobID); err != nil {
		return fmt.Errorf("complete job: %w", err)
	}

	if err := os.Remove(task.SourcePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		w.logger.Warn("failed to remove source file", "path", task.SourcePath, "error", err)
	}
	return nil
}

func (w *Worker) Kill() {
	select {
	case <-w.stop:
		return
	default:
		close(w.stop)
	}
}
