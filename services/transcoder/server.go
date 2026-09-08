package transcoder

import (
	"time"

	pb "github.com/crimsonn/media_pipeline/pkg/pb"
)

type Server struct {
	pb.UnimplementedTranscoderServiceServer
}

func NewTranscoderServer() *Server {
	return &Server{}
}

func (s *Server) TranscodeVideo(req *pb.TranscodeRequest, stream pb.TranscoderService_TranscodeVideoServer) error {
	ctx := stream.Context()
	stream.Send(&pb.TranscodeProgress{
		TaskId:          req.FileId,
		State:           pb.TranscodeState_STATE_PROCESSING,
		PercentComplete: 0,
	})

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	time.Sleep(10 * time.Second)
	for i := 0; i < 100; i++ {
		stream.Send(&pb.TranscodeProgress{
			TaskId:          req.FileId,
			State:           pb.TranscodeState_STATE_PROCESSING,
			PercentComplete: int32(i),
		})
		time.Sleep(100 * time.Millisecond)
	}

	return stream.Send(&pb.TranscodeProgress{
		TaskId:          req.FileId,
		State:           pb.TranscodeState_STATE_COMPLETED,
		PercentComplete: 100,
	})
}
