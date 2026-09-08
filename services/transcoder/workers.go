package transcoder

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type TaskState int

const (
	TaskStatePending TaskState = iota
	TaskStateInProgress
	TaskStateCompleted
	TaskStateError
)

type Rendition struct {
	Name     string
	Width    int
	Height   int
	VideoBps int
	AudioBps int
}

type Task struct {
	id              uuid.UUID
	sourceFile      string
	filePath        string
	outputDirectory string
	rendition       Rendition
	progress        int
	state           TaskState
	backLog         chan *Task
	error           error
}

type Worker struct {
	numWorkers int
	tasks      chan *Task
	wg         sync.WaitGroup
	logger     *slog.Logger
}

const segmentSeconds = 6

func NewWorkers(numWorkers int, logger *slog.Logger) *Worker {
	return &Worker{
		numWorkers: numWorkers,
		tasks:      make(chan *Task),
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

func (w *Worker) processTask(ctx context.Context, workerid int, task *Task) {
	task.state = TaskStateInProgress
	defer func() {
		if r := recover(); r != nil {
			task.state = TaskStateError
			task.error = fmt.Errorf("worker %d panicked: %v", workerid, r)
		}
		select {
		case task.backLog <- task:
		case <-ctx.Done():
			w.logger.Warn("dropping result, context cancelled", "task", task.id)
		}
	}()

	if err := os.MkdirAll(task.filePath, 0o755); err != nil {
		task.state = TaskStateError
		task.error = fmt.Errorf("create variant directory: %w", err)
		return
	}

	renditions := renditionArgs(task.sourceFile, task.filePath, task.rendition)
	cmd := exec.CommandContext(ctx, "ffmpeg", renditions...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		task.state = TaskStateError
		task.error = fmt.Errorf("ffmpeg failed: %w", err)
		return
	}
	task.state = TaskStateCompleted
}

func (w *Worker) SubmitTask(ctx context.Context, t *Task) error {
	select {
	case w.tasks <- t:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (t *Task) GetProgress() int {
	return t.progress
}

func (w *Worker) Stop() {
	close(w.tasks)
	w.wg.Wait()
}

func renditionArgs(src string, variantDir string, r Rendition) []string {
	return []string{
		"-i", src,
		"-map", "0:v:0",
		"-vf", fmt.Sprintf("scale=w=%d:h=%d", r.Width, r.Height),
		"-c:v", "libx264",
		"-profile:v", "high", "-level", "4.0",
		"-preset", "veryfast",
		"-b:v", strconv.Itoa(r.VideoBps),
		"-maxrate", strconv.Itoa(r.VideoBps * 11 / 10),
		"-bufsize", strconv.Itoa(r.VideoBps * 2),
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentSeconds),
		"-sc_threshold", "0",
		"-map", "0:a:0?",
		"-c:a", "aac", "-b:a", strconv.Itoa(r.AudioBps), "-ac", "2",
		"-f", "hls",
		"-hls_time", strconv.Itoa(segmentSeconds),
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(variantDir, "seg_%05d.ts"),
		filepath.Join(variantDir, "manifest.m3u8"),
	}
}

func writeMaster(outDir string, rs []Rendition) error {
	var b strings.Builder
	b.WriteString("#EXTM3U\n#EXT-X-VERSION:6\n#EXT-X-INDEPENDENT-SEGMENTS\n")
	for _, r := range rs {
		fmt.Fprintf(&b,
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"avc1.640028,mp4a.40.2\"\n%s/manifest.m3u8\n",
			r.VideoBps+r.AudioBps, r.Width, r.Height, r.Name)
	}
	return os.WriteFile(filepath.Join(outDir, "master.m3u8"), []byte(b.String()), 0o644)
}
