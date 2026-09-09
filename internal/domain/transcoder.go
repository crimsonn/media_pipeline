package domain

import (
	"github.com/crimsonn/media_pipeline/internal/db/queries"
	"github.com/google/uuid"
)

type Rendition struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	VideoBitrate int    `json:"video_bitrate"`
	AudioBitrate int    `json:"audio_bitrate"`
	VideoCodec   string `json:"video_codec"`
	AudioCodec   string `json:"audio_codec"`
	StreamIndex  int    `json:"stream_index"`
}

type Profile struct {
	ID             int64       `json:"id"`
	Name           string      `json:"name"`
	Description    string      `json:"description"`
	HlsSegmentTime int         `json:"hls_segment_time"`
	Renditions     []Rendition `json:"renditions"`
}

type TaskState int

const (
	TaskStatePending TaskState = iota
	TaskStateInProgress
	TaskStateCompleted
	TaskStateError
)

type Task struct {
	ID              uuid.UUID
	SourceFile      string
	FilePath        string
	OutputDirectory string
	Rendition       Rendition
	Progress        int
	State           TaskState
	BackLog         chan *Task
	Error           error
}

type CreateRenditionRequest struct {
	Name         string `json:"name" binding:"required"`
	Width        int    `json:"width" binding:"required"`
	Height       int    `json:"height" binding:"required"`
	VideoBitrate int    `json:"video_bitrate" binding:"required"`
	AudioBitrate int    `json:"audio_bitrate" binding:"required"`
	VideoCodec   string `json:"video_codec" binding:"required"`
	AudioCodec   string `json:"audio_codec" binding:"required"`
	Fps          int    `json:"fps" binding:"required"`
}

type RenditionResponse struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	VideoBitrate int    `json:"video_bitrate"`
	AudioBitrate int    `json:"audio_bitrate"`
	VideoCodec   string `json:"video_codec"`
	AudioCodec   string `json:"audio_codec"`
	Fps          int    `json:"fps"`
}

type CreateProfileRequest struct {
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description" binding:"required"`
	HlsSegmentTime int    `json:"hls_segment_time" binding:"required"`
	IsDefault      bool   `json:"is_default" binding:"required"`
	Renditions     []int  `json:"renditions" binding:"required"`
}

type ProfileResponse struct {
	ID             int64               `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	HlsSegmentTime int                 `json:"hls_segment_time"`
	Renditions     []RenditionResponse `json:"renditions"`
}

func ToProfileResponse(p queries.TranscodeProfile, renditions []queries.Rendition) ProfileResponse {
	description := ""
	if p.Description != nil {
		description = *p.Description
	}
	renditionResponses := make([]RenditionResponse, 0, len(renditions))
	for _, rendition := range renditions {
		renditionResponses = append(renditionResponses, ToRenditionResponse(rendition))
	}
	return ProfileResponse{
		ID:             p.ID,
		Name:           p.Name,
		Description:    description,
		HlsSegmentTime: int(p.HlsSegmentTime),
		Renditions:     renditionResponses,
	}
}

func ToRenditionResponse(r queries.Rendition) RenditionResponse {
	return RenditionResponse{
		ID:           r.ID,
		Name:         r.Name,
		Width:        int(r.Width),
		Height:       int(r.Height),
		VideoBitrate: int(r.VideoBitrate),
		AudioBitrate: int(r.AudioBitrate),
		VideoCodec:   r.VideoCodec,
		AudioCodec:   r.AudioCodec,
		Fps:          int(r.Fps),
	}
}
