package playback

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/crimsonn/media_pipeline/internal/config"
	"github.com/crimsonn/media_pipeline/internal/domain"
)

const masterPlaylist = "master.m3u8"

type Service interface {
	ListOutputs(ctx context.Context) ([]*domain.PlaybackOutput, error)
	ResolveFile(rel string) (string, error)
}

type service struct {
	outputDir string
}

func NewService(cfg *config.Config) Service {
	return &service{outputDir: cfg.OutputDirectory}
}

func (s *service) ListOutputs(_ context.Context) ([]*domain.PlaybackOutput, error) {
	entries, err := os.ReadDir(s.outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*domain.PlaybackOutput{}, nil
		}
		return nil, fmt.Errorf("read output dir: %w", err)
	}

	outputs := make([]*domain.PlaybackOutput, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(s.outputDir, entry.Name())
		masterInfo, err := os.Stat(filepath.Join(dir, masterPlaylist))
		if err != nil {
			continue
		}

		renditions := make([]string, 0)
		if variants, err := os.ReadDir(dir); err == nil {
			for _, v := range variants {
				if v.IsDir() {
					renditions = append(renditions, v.Name())
				}
			}
		}
		sort.Strings(renditions)

		outputs = append(outputs, &domain.PlaybackOutput{
			Name:       entry.Name(),
			MasterPath: fmt.Sprintf("playback/hls/%s/%s", entry.Name(), masterPlaylist),
			Renditions: renditions,
			ModifiedAt: masterInfo.ModTime(),
		})
	}

	sort.Slice(outputs, func(a, b int) bool {
		return outputs[a].ModifiedAt.After(outputs[b].ModifiedAt)
	})
	return outputs, nil
}

func (s *service) ResolveFile(rel string) (string, error) {
	root, err := filepath.Abs(s.outputDir)
	if err != nil {
		return "", fmt.Errorf("resolve output dir: %w", err)
	}

	clean := filepath.Clean("/" + filepath.FromSlash(rel))
	full := filepath.Join(root, clean)
	if full != root && !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", fmt.Errorf("path escapes output directory")
	}
	return full, nil
}
