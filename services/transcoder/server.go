package transcoder

import (
	"context"
	"log/slog"
	"path"
	"strings"

	pb "github.com/crimsonn/media_pipeline/pkg/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedTranscoderServiceServer
}

func NewTranscoderServer() *Server {
	return &Server{}
}

func (s *Server) TranscodeVideo(ctx context.Context, req *pb.TranscodeRequest) (*pb.TranscodeResponse, error) {
	if req.GetFileId() == "" {
		return nil, status.Error(codes.InvalidArgument, "file_id is required")
	}
	if req.GetSourceFilePath() == "" {
		return nil, status.Error(codes.InvalidArgument, "source_file_path is required")
	}
	if req.GetOutputDirectory() == "" {
		return nil, status.Error(codes.InvalidArgument, "output_directory is required")
	}

	slog.InfoContext(ctx, "transcode requested",
		"file_id", req.GetFileId(),
		"source", req.GetSourceFilePath(),
		"output", req.GetOutputDirectory(),
		"resolutions", strings.Join(req.GetTargetResolutions(), ","),
	)

	return &pb.TranscodeResponse{
		FileId:            req.GetFileId(),
		Success:           true,
		MasterPlaylistUrl: path.Join(req.GetOutputDirectory(), req.GetFileId(), "master.m3u8"),
	}, nil
}
