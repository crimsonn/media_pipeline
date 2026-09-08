package transcoder

import (
	"context"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/crimsonn/media_pipeline/internal/domain"
)

type Service interface {
	GetFullProfile(ctx context.Context, profileID int64) (*domain.Profile, error)
}

type service struct {
	queries *queries.Queries
}

// GetFullProfile implements [Service].
func (s *service) GetFullProfile(ctx context.Context, profileID int64) (*domain.Profile, error) {
	panic("unimplemented")
}

func NewService(queries *queries.Queries) Service {
	return &service{queries}
}
