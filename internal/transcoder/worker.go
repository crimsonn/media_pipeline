package transcoder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/crimsonn/media_pipeline/internal/domain"
)

type Worker struct {
	numWorkers int
	tasks      chan *domain.Task
	wg         sync.WaitGroup
	logger     *slog.Logger
}

const segmentSeconds = 6

func NewWorkers(numWorkers int, logger *slog.Logger) *Worker {
	return &Worker{
		numWorkers: numWorkers,
		tasks:      make(chan *domain.Task),
		wg:         sync.WaitGroup{},
		logger:     logger,
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
	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker shutting down")
			return
		case task, ok := <-w.tasks:
			if !ok {
				return
			}
			w.processTask(ctx, id, task)
		}
	}
}

func (w *Worker) processTask(ctx context.Context, workerid int, task *domain.Task) {
	task.State = domain.TaskStateInProgress
	defer func() {
		if r := recover(); r != nil {
			task.State = domain.TaskStateError
			task.Error = fmt.Errorf("worker %d panicked: %v", workerid, r)
		}
		select {
		case task.BackLog <- task:
		case <-ctx.Done():
			w.logger.Warn("dropping result, context cancelled", "task", task.ID)
		}
	}()

	if err := os.MkdirAll(task.FilePath, 0o755); err != nil {
		task.State = domain.TaskStateError
		task.Error = fmt.Errorf("create variant directory: %w", err)
		return
	}

	renditions := renditionArgs(task.SourceFile, task.FilePath, task.Rendition)
	cmd := exec.CommandContext(ctx, "ffmpeg", renditions...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		task.State = domain.TaskStateError
		task.Error = fmt.Errorf("ffmpeg failed: %w", err)
		return
	}
	task.State = domain.TaskStateCompleted
}

func (w *Worker) SubmitTask(ctx context.Context, t *domain.Task) error {
	select {
	case w.tasks <- t:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *Worker) Stop() {
	close(w.tasks)
	w.wg.Wait()
}

func renditionArgs(src string, variantDir string, r domain.Rendition) []string {
	return []string{
		"-i", src,
		"-map", "0:v:0",
		"-vf", fmt.Sprintf("scale=w=%d:h=%d", r.Width, r.Height),
		"-c:v", "libx264",
		"-profile:v", "high", "-level", "4.0",
		"-preset", "veryfast",
		"-b:v", strconv.Itoa(r.VideoBitrate),
		"-maxrate", strconv.Itoa(r.VideoBitrate * 11 / 10),
		"-bufsize", strconv.Itoa(r.VideoBitrate * 2),
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentSeconds),
		"-sc_threshold", "0",
		"-map", "0:a:0?",
		"-c:a", "aac", "-b:a", strconv.Itoa(r.AudioBitrate), "-ac", "2",
		"-f", "hls",
		"-hls_time", strconv.Itoa(segmentSeconds),
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(variantDir, "seg_%05d.ts"),
		filepath.Join(variantDir, "manifest.m3u8"),
	}
}
