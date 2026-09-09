package domain

import (
	"time"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
)

type JobResponse struct {
	ID           int64             `json:"id"`
	Status       string            `json:"status"`
	ErrorMessage string            `json:"error_message"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	ProfileID    int64             `json:"profile_id"`
	SourcePath   string            `json:"source_path"`
	OutputDir    string            `json:"output_dir"`
	FileName     string            `json:"file_name"`
	Tasks        []JobTaskResponse `json:"tasks"`
}

type JobTaskResponse struct {
	ID           int64             `json:"id"`
	Status       string            `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	ErrorMessage string            `json:"error_message"`
	Rendition    RenditionResponse `json:"rendition"`
}

func ToJobResponses(rows []queries.GetLatestJobTasksRow) []*JobResponse {
	byID := make(map[int64]*JobResponse, len(rows))
	order := make([]int64, 0)

	for _, row := range rows {
		job, ok := byID[row.JobID]
		if !ok {
			job = &JobResponse{
				ID:           row.JobID,
				Status:       row.JobStatus,
				ErrorMessage: derefString(row.JobErrorMessage),
				CreatedAt:    row.JobCreatedAt.Time,
				UpdatedAt:    row.JobUpdatedAt.Time,
				ProfileID:    row.JobProfileID,
				SourcePath:   row.JobSourcePath,
				OutputDir:    row.JobOutputDir,
				FileName:     row.JobFileName,
				Tasks:        make([]JobTaskResponse, 0),
			}
			byID[row.JobID] = job
			order = append(order, row.JobID)
		}

		job.Tasks = append(job.Tasks, JobTaskResponse{
			ID:           row.TaskID,
			Status:       row.TaskStatus,
			CreatedAt:    row.TaskCreatedAt.Time,
			UpdatedAt:    row.TaskUpdatedAt.Time,
			ErrorMessage: derefString(row.TaskErrorMessage),
			Rendition: RenditionResponse{
				ID:           row.RenditionID,
				Name:         row.RenditionName,
				Width:        int(row.RenditionWidth),
				Height:       int(row.RenditionHeight),
				VideoBitrate: int(row.RenditionVideoBitrate),
				AudioBitrate: int(row.RenditionAudioBitrate),
				VideoCodec:   row.RenditionVideoCodec,
				AudioCodec:   row.RenditionAudioCodec,
				Fps:          int(row.RenditionFps),
			},
		})
	}

	responses := make([]*JobResponse, 0, len(order))
	for _, id := range order {
		responses = append(responses, byID[id])
	}
	return responses
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
