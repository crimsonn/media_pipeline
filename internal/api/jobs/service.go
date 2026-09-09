package jobs

import (
	"context"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/domain"
)

type Service interface {
	GetLatestJobs(ctx context.Context) ([]*domain.JobResponse, error)
}

type service struct {
	db *queries.Queries
}

func NewService(db *queries.Queries) Service {
	return &service{db: db}
}

func (s *service) GetLatestJobs(ctx context.Context) ([]*domain.JobResponse, error) {
	rows, err := s.db.GetLatestJobTasks(ctx)
	if err != nil {
		return nil, err
	}
	return domain.ToJobResponses(rows), nil
}
