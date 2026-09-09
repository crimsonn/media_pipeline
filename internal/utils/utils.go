package utils

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/crimsonn/media_pipeline/internal/db/queries"
)

func FileTrimSuffix(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))

}

func RenditionArgs(src string, variantDir string, segmentSeconds int, r queries.ClaimJobTaskRow) []string {
	if segmentSeconds <= 0 {
		segmentSeconds = 6
	}
	vBitrate := fmt.Sprintf("%dk", r.VideoBitrate)
	vMaxRate := fmt.Sprintf("%dk", r.VideoBitrate*11/10)
	vBufSize := fmt.Sprintf("%dk", r.VideoBitrate*2)
	aBitrate := fmt.Sprintf("%dk", r.AudioBitrate)
	return []string{
		"-threads", "2",
		"-i", src,
		"-map", "0:v:0",
		"-vf", fmt.Sprintf("scale=w=%d:h=%d", r.Width, r.Height),
		"-c:v", "libx264",
		"-profile:v", "high", "-level", "4.0",
		"-preset", "veryfast",
		"-b:v", vBitrate,
		"-maxrate", vMaxRate,
		"-bufsize", vBufSize,
		"-force_key_frames", fmt.Sprintf("expr:gte(t,n_forced*%d)", segmentSeconds),
		"-sc_threshold", "0",
		"-map", "0:a:0?",
		"-c:a", "aac",
		"-b:a", aBitrate,
		"-ac", "2",
		"-f", "hls",
		"-hls_time", strconv.Itoa(segmentSeconds),
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(variantDir, "seg_%05d.ts"),
		filepath.Join(variantDir, "manifest.m3u8"),
	}
}
