package domain

import (
	"time"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
)

type PendingFileResponse struct {
	ID        int64     `json:"id"`
	FileName  string    `json:"file_name"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type PendingEnqueueRequest struct {
	ID                 int64  `json:"id" binding:"required"`
	FileName           string `json:"file_name" binding:"required"`
	TranscodeProfileID int64  `json:"transcode_profile_id" binding:"required"`
}

func ToPendingResponse(pending queries.PendingFile) *PendingFileResponse {
	return &PendingFileResponse{
		ID:        pending.ID,
		FileName:  pending.FileName,
		Status:    pending.Status,
		CreatedAt: pending.CreatedAt.Time,
		UpdatedAt: pending.UpdatedAt.Time,
	}
}
