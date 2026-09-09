package transcoder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/domain"
	"github.com/jackc/pgx/v5"
)

type Worker struct {
	numWorkers int
	wg         sync.WaitGroup
	logger     *slog.Logger
	queries    *queries.Queries
}

func NewWorkers(numWorkers int, logger *slog.Logger, q *queries.Queries) *Worker {
	return &Worker{
		numWorkers: numWorkers,
		logger:     logger,
		queries:    q,
	}
}

func (w *Worker) Start(ctx context.Context) {
	for i := 1; i <= w.numWorkers; i++ {
		w.wg.Add(1)
		go w.worker(ctx, i)
	}
}

func (w *Worker) worker(ctx context.Context, id int) {
	defer w.wg.Done()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			task, err := w.queries.ClaimJobTask(ctx)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					continue
				}
				w.logger.Error("failed to claim job task", "worker", id, "error", err)
				continue
			}
			w.processTask(ctx, id, task)
		}
	}
}

func (w *Worker) processTask(ctx context.Context, workerID int, task queries.ClaimJobTaskRow) {
	fail := func(msg string) {
		if err := w.queries.FailJobTask(ctx, queries.FailJobTaskParams{
			ID:           task.TaskID,
			ErrorMessage: &msg,
		}); err != nil {
			w.logger.Error("failed to mark task failed", "task", task.TaskID, "error", err)
		}
		if err := w.maybeFinalizeJob(ctx, task); err != nil {
			w.logger.Error("failed to finalize job", "job", task.JobID, "error", err)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			fail(fmt.Sprintf("worker %d panicked: %v", workerID, r))
		}
	}()

	if err := w.queries.MarkJobRunning(ctx, task.JobID); err != nil {
		w.logger.Error("failed to mark job running", "job", task.JobID, "error", err)
	}

	stem := fileTrimSuffix(task.FileName)
	variantDir := filepath.Join(task.OutputDir, stem, task.RenditionName)
	if err := os.MkdirAll(variantDir, 0o755); err != nil {
		fail(fmt.Sprintf("create variant directory: %v", err))
		return
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", renditionArgs(task.SourcePath, variantDir, int(task.HlsSegmentTime), task)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fail(fmt.Sprintf("ffmpeg failed: %v", err))
		return
	}

	if err := w.queries.CompleteJobTask(ctx, task.TaskID); err != nil {
		w.logger.Error("failed to complete task", "task", task.TaskID, "error", err)
		fail(fmt.Sprintf("complete task: %v", err))
		return
	}

	if err := w.maybeFinalizeJob(ctx, task); err != nil {
		w.logger.Error("failed to finalize job", "job", task.JobID, "error", err)
	}
}

func (w *Worker) maybeFinalizeJob(ctx context.Context, task queries.ClaimJobTaskRow) error {
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

	masterDir := filepath.Join(task.OutputDir, fileTrimSuffix(task.FileName))
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

func (w *Worker) Stop() {
	w.wg.Wait()
}

func fileTrimSuffix(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))

}

func renditionArgs(src string, variantDir string, segmentSeconds int, r queries.ClaimJobTaskRow) []string {
	if segmentSeconds <= 0 {
		segmentSeconds = 6
	}
	vBitrate := fmt.Sprintf("%dk", r.VideoBitrate)
	vMaxRate := fmt.Sprintf("%dk", r.VideoBitrate*11/10)
	vBufSize := fmt.Sprintf("%dk", r.VideoBitrate*2)
	aBitrate := fmt.Sprintf("%dk", r.AudioBitrate)
	return []string{
		"-threads", "2",
		"-i", src,
		"-map", "0:v:0",
		"-vf", fmt.Sprintf("scale=w=%d:h=%d", r.Width, r.Height),
		"-c:v", "libx264",
		"-profile:v", "high", "-level", "4.0",
		"-preset", "veryfast",
		"-b:v", vBitrate,
		"-maxrate", vMaxRate,
		"-bufsize", vBufSize,
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentSeconds),
		"-sc_threshold", "0",
		"-map", "0:a:0?",
		"-c:a", "aac",
		"-b:a", aBitrate,
		"-ac", "2",
		"-f", "hls",
		"-hls_time", strconv.Itoa(segmentSeconds),
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(variantDir, "seg_%05d.ts"),
		filepath.Join(variantDir, "manifest.m3u8"),
	}
}
