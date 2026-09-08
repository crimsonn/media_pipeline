package grpc

import (
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/crimsonn/media_pipeline/internal/domain"
	"github.com/crimsonn/media_pipeline/internal/transcoder"
	"github.com/crimsonn/media_pipeline/pkg/pb"
	"github.com/google/uuid"
	"google.golang.org/grpc/status"
)

type Done struct {
	TaskID uuid.UUID
	State  domain.TaskState
	Error  error
}

type TranscoderGRPCServer struct {
	pb.UnimplementedTranscoderServiceServer
	worker *transcoder.Worker
	logger *slog.Logger
}

func NewTranscoderGRPCServer(
	worker *transcoder.Worker,
	logger *slog.Logger,
) *TranscoderGRPCServer {
	return &TranscoderGRPCServer{
		worker: worker,
		logger: logger,
	}
}

func (s *TranscoderGRPCServer) TranscodeVideo(req *pb.TranscodeVideoRequest, stream pb.TranscoderService_TranscodeVideoServer) error {
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

	backLog := make(chan *domain.Task, len(req.Resolutions))

	filenameWithoutExtension := parts[0]
	renditions := []domain.Rendition{}
	for _, resolution := range req.Resolutions {
		rendition := domain.Rendition{
			Name:         resolution.Name,
			Width:        int(resolution.Width),
			Height:       int(resolution.Height),
			VideoBitrate: int(resolution.VideoBps),
			AudioBitrate: int(resolution.AudioBps),
		}
		renditions = append(renditions, rendition)
		task := &domain.Task{
			ID:              uuid.New(),
			SourceFile:      req.SourceFilePath,
			FilePath:        path.Join(req.OutputDirectory, filenameWithoutExtension, resolution.Name),
			OutputDirectory: req.OutputDirectory,
			Rendition:       rendition,
			BackLog:         backLog,
		}
		s.worker.SubmitTask(ctx, task)
	}

	done := make(map[string]Done, len(req.Resolutions))
	for len(done) < len(req.Resolutions) {
		select {
		case l := <-backLog:
			if _, ok := done[l.ID.String()]; ok {
				continue
			}
			done[l.ID.String()] = Done{TaskID: l.ID, State: l.State, Error: l.Error}
		case <-ctx.Done():
			return status.FromContextError(ctx.Err()).Err()
		}
	}
	someFailed := false
	for _, d := range done {
		if d.State == domain.TaskStateError {
			someFailed = true
			break
		}
	}
	if someFailed {
		failed := []string{}
		for _, d := range done {
			if d.State == domain.TaskStateError {
				failed = append(failed, d.Error.Error())
			}
		}
		return stream.Send(&pb.TranscodeVideoResponse{
			TaskId:       req.FileId,
			State:        pb.TranscodeState_TRANSCODE_STATE_FAILED,
			ErrorMessage: strings.Join(failed, ", "),
		})
	}

	transcoder.WriteMaster(filepath.Join(req.OutputDirectory, filenameWithoutExtension), renditions)

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
