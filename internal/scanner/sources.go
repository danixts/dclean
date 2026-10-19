package scanner

import (
	"os"
	"path/filepath"

	"dclean/internal/domain"
)

func BuildSources(activePaths []domain.ScanPath) ([]domain.ScanSource, string) {
	home, _ := os.UserHomeDir()
	snapDir := filepath.Join(home, "snap")

	var sources []domain.ScanSource
	for _, p := range activePaths {
		sources = append(sources, domain.ScanSource{
			Root:       p.Path,
			Categories: domain.ProjectCategories,
			Direct:     false,
		})
	}

	cachePath := filepath.Join(home, ".cache")
	if isDir(cachePath) {
		sources = append(sources, domain.ScanSource{
			Root:       cachePath,
			Categories: domain.HomeCacheCategories,
			Direct:     true,
		})
	}

	for _, hc := range domain.HomeDirCaches {
		root := filepath.Join(home, hc.Root)
		if isDir(root) {
			sources = append(sources, domain.ScanSource{
				Root:       root,
				Categories: []domain.Category{hc.Category},
				Direct:     true,
			})
		}
	}

	return sources, snapDir
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
