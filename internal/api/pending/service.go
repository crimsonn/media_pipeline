package pending

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/db"
	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service interface {
	GetPendingFiles(ctx context.Context) ([]*domain.PendingFileResponse, error)
	EnqueuePendingFile(ctx context.Context, req domain.PendingEnqueueRequest) error
}

type service struct {
	db     *queries.Queries
	pool   *pgxpool.Pool
	config *config.Config
}

func NewService(pool *pgxpool.Pool, config *config.Config) Service {
	return &service{pool: pool, db: queries.New(pool), config: config}
}

func (s *service) GetPendingFiles(ctx context.Context) ([]*domain.PendingFileResponse, error) {
	pending, err := s.db.GetPendingFiles(ctx)
	if err != nil {
		return nil, err
	}
	responses := make([]*domain.PendingFileResponse, len(pending))
	for i, p := range pending {
		responses[i] = domain.ToPendingResponse(p)
	}
	return responses, nil
}

func (s *service) EnqueuePendingFile(ctx context.Context, req domain.PendingEnqueueRequest) error {
	sourcePath := path.Join(s.config.WatchDirectory, req.FileName)
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", sourcePath)
	}

	db.EnqueueTranscode(ctx, s.pool, queries.EnqueueJobParams{
		FileName:   req.FileName,
		ProfileID:  req.TranscodeProfileID,
		SourcePath: sourcePath,
		OutputDir:  s.config.OutputDirectory,
	})
	err := s.db.UpdatePendingFileStatus(ctx, queries.UpdatePendingFileStatusParams{
		ID:     req.ID,
		Status: "processing",
	})
	if err != nil {
		return fmt.Errorf("failed to update pending file status: %w", err)
	}
	return nil
}
