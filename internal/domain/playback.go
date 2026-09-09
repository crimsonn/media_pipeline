package domain

import "time"

// PlaybackOutput is a finished HLS transcode available under the output ("done")
// directory: a folder containing a master.m3u8 plus one sub-folder per rendition.
type PlaybackOutput struct {
	// Name is the output folder name (the source file stem, e.g. "clip").
	Name string `json:"name"`
	// MasterPath is the master playlist location relative to the API v1 base,
	// e.g. "playback/hls/clip/master.m3u8".
	MasterPath string `json:"master_path"`
	// Renditions lists the per-rendition sub-folder names (e.g. "1080p", "720p").
	Renditions []string `json:"renditions"`
	// ModifiedAt is the master playlist's mtime, i.e. when the job finished.
	ModifiedAt time.Time `json:"modified_at"`
}
