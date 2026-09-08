package transcoder

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/crimsonn/media_pipeline/internal/domain"
)

func WriteMaster(outDir string, rs []domain.Rendition) error {
	var b strings.Builder
	b.WriteString("#EXTM3U\n#EXT-X-VERSION:6\n#EXT-X-INDEPENDENT-SEGMENTS\n")
	for _, r := range rs {
		fmt.Fprintf(&b,
			"#EXT-X-STREAM-INF:BANDWIDTH=%d,RESOLUTION=%dx%d,CODECS=\"avc1.640028,mp4a.40.2\"\n%s/manifest.m3u8\n",
			r.VideoBitrate+r.AudioBitrate, r.Width, r.Height, r.Name)
	}
	return os.WriteFile(filepath.Join(outDir, "master.m3u8"), []byte(b.String()), 0o644)
}
