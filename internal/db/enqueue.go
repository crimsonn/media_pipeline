package db

import (
	"context"
	"fmt"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/jackc/pgx/v5/pgxpool"
)

func EnqueueTranscode(ctx context.Context, pool *pgxpool.Pool, arg queries.EnqueueJobParams) (queries.Job, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return queries.Job{}, fmt.Errorf("begin enqueue tx: %w", err)
	}
	defer tx.Rollback(ctx)

	q := queries.New(tx)
	rows, err := q.GetProfileWithRenditions(ctx, arg.ProfileID)
	if err != nil {
		return queries.Job{}, fmt.Errorf("load profile %d: %w", arg.ProfileID, err)
	}
	if len(rows) == 0 {
		return queries.Job{}, fmt.Errorf("profile %d has no renditions", arg.ProfileID)
	}

	job, err := q.EnqueueJob(ctx, arg)
	if err != nil {
		return queries.Job{}, fmt.Errorf("enqueue job: %w", err)
	}

	for _, row := range rows {
		if _, err := q.InsertJobTask(ctx, queries.InsertJobTaskParams{
			JobID:       job.ID,
			RenditionID: row.RenditionID,
		}); err != nil {
			return queries.Job{}, fmt.Errorf("enqueue rendition %s: %w", row.RenditionName, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return queries.Job{}, fmt.Errorf("commit enqueue tx: %w", err)
	}
	return job, nil
}
