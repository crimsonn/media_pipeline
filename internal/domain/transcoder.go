package domain

import "github.com/google/uuid"

type Rendition struct {
	ID           int64
	Name         string
	Width        int
	Height       int
	VideoBitrate int
	AudioBitrate int
	VideoCodec   string
	AudioCodec   string
	StreamIndex  int
}

type Profile struct {
	ID             int64
	Name           string
	Description    string
	HlsSegmentTime int
	Renditions     []Rendition
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
