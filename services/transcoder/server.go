package transcoder

import (
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	pb "github.com/crimsonn/media_pipeline/pkg/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/status"
)

type Done struct {
	taskId uuid.UUID
	state  TaskState
	error  error
}

type Server struct {
	pb.UnimplementedTranscoderServiceServer
	worker *Worker
	logger *slog.Logger
}

func NewTranscoderServer(worker *Worker, logger *slog.Logger) *Server {
	return &Server{
		worker: worker,
		logger: logger,
	}
}

func (s *Server) TranscodeVideo(req *pb.TranscodeVideoRequest, stream pb.TranscoderService_TranscodeVideoServer) error {
	ctx := stream.Context()
	s.logger.Info("received transcode video request", "file_id", req.FileId)
	if len(req.Resolutions) == 0 {
		return stream.Send(&pb.TranscodeVideoResponse{
			TaskId:       req.FileId,
			State:        pb.TranscodeState_TRANSCODE_STATE_FAILED,
			ErrorMessage: "invalid renditions length",
		})
	}

	parts := strings.Split(req.FileName, ".")
	if len(parts) < 2 {
		return stream.Send(&pb.TranscodeVideoResponse{
			TaskId:       req.FileId,
			State:        pb.TranscodeState_TRANSCODE_STATE_FAILED,
			ErrorMessage: "invalid file name",
		})
	}

	backLog := make(chan *Task, len(req.Resolutions))

	filenameWithoutExtension := parts[0]
	renditions := []Rendition{}
	for _, resolution := range req.Resolutions {
		rendition := Rendition{
			Name:     resolution.Name,
			Width:    int(resolution.Width),
			Height:   int(resolution.Height),
			VideoBps: int(resolution.VideoBps),
			AudioBps: int(resolution.AudioBps),
		}
		renditions = append(renditions, rendition)
		task := &Task{
			id:              uuid.New(),
			sourceFile:      req.SourceFilePath,
			filePath:        path.Join(req.OutputDirectory, filenameWithoutExtension, resolution.Name),
			outputDirectory: req.OutputDirectory,
			rendition:       rendition,
			backLog:         backLog,
		}
		s.worker.SubmitTask(ctx, task)
	}

	done := make(map[string]Done, len(req.Resolutions))
	for len(done) < len(req.Resolutions) {
		select {
		case l := <-backLog:
			if _, ok := done[l.id.String()]; ok {
				continue
			}
			done[l.id.String()] = Done{taskId: l.id, state: l.state, error: l.error}
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		}
	}
	someFailed := false
	for _, d := range done {
		if d.state == TaskStateError {
			someFailed = true
			break
		}
	}
	if someFailed {
		failed := []string{}
		for _, d := range done {
			if d.state == TaskStateError {
				failed = append(failed, d.error.Error())
			}
		}
		return stream.Send(&pb.TranscodeVideoResponse{
			TaskId:       req.FileId,
			State:        pb.TranscodeState_TRANSCODE_STATE_FAILED,
			ErrorMessage: strings.Join(failed, ", "),
		})
	}

	writeMaster(filepath.Join(req.OutputDirectory, filenameWithoutExtension), renditions)

	if err := os.Remove(req.SourceFilePath); err != nil {
		return stream.Send(&pb.TranscodeVideoResponse{
			TaskId:       req.FileId,
			State:        pb.TranscodeState_TRANSCODE_STATE_FAILED,
			ErrorMessage: err.Error(),
		})
	}
	return stream.Send(&pb.TranscodeVideoResponse{
		TaskId:          req.FileId,
		State:           pb.TranscodeState_TRANSCODE_STATE_COMPLETED,
		PercentComplete: 100,
	})
}
