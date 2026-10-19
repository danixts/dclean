package scanner

import (
	"os"
	"path/filepath"
	"runtime"

	"dclean/internal/domain"

	"golang.org/x/sync/errgroup"
)

const maxSizeWorkers = 8

type sizeCandidate struct {
	path     string
	category string
	target   string
}

type targetMatch struct {
	category string
	target   string
}

var versionControlDirs = map[string]bool{
	".git": true,
	".hg":  true,
	".svn": true,
}

func (ms *MultiScanner) scanDirectChildren(source domain.ScanSource, onProgress func(int64)) {
	targetLookup := buildTargetLookup(source.Categories)

	entries, err := os.ReadDir(source.Root)
	if err != nil {
		return
	}

	var candidates []sizeCandidate
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		ms.Result.ScannedDirs.Add(1)
		ms.notifyProgress(onProgress)

		match, found := targetLookup[entry.Name()]
		if !found {
			continue
		}

		candidates = append(candidates, sizeCandidate{
			path:     filepath.Join(source.Root, entry.Name()),
			category: match.category,
			target:   match.target,
		})
	}

	ms.resolveSizes(candidates)
}

func (ms *MultiScanner) scanRecursive(source domain.ScanSource, onProgress func(int64)) {
	targetLookup := buildTargetLookup(source.Categories)

	var candidates []sizeCandidate
	_ = filepath.WalkDir(source.Root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || !entry.IsDir() {
			return nil
		}

		ms.Result.ScannedDirs.Add(1)
		ms.notifyProgress(onProgress)

		dirName := entry.Name()
		if versionControlDirs[dirName] {
			return filepath.SkipDir
		}

		match, found := targetLookup[dirName]
		if !found {
			return nil
		}

		candidates = append(candidates, sizeCandidate{path: path, category: match.category, target: match.target})
		return filepath.SkipDir
	})

	ms.resolveSizes(candidates)
}

func (ms *MultiScanner) resolveSizes(candidates []sizeCandidate) {
	if len(candidates) == 0 {
		return
	}

	var g errgroup.Group
	g.SetLimit(min(maxSizeWorkers, runtime.GOMAXPROCS(0)*2))

	for _, c := range candidates {
		g.Go(func() error {
			size := calculateDirSize(c.path)
			if size > 0 {
				ms.Result.add(domain.FoundDir{
					Path:     c.path,
					Size:     size,
					Category: c.category,
					Target:   c.target,
				})
			}
			return nil
		})
	}
	_ = g.Wait()
}

func (ms *MultiScanner) notifyProgress(onProgress func(int64)) {
	if onProgress != nil && ms.Result.ScannedDirs.Load()%500 == 0 {
		onProgress(ms.Result.ScannedDirs.Load())
	}
}

func buildTargetLookup(categories []domain.Category) map[string]targetMatch {
	lookup := make(map[string]targetMatch)
	for _, cat := range categories {
		for _, target := range cat.Targets {
			lookup[target] = targetMatch{category: cat.Name, target: target}
		}
	}
	return lookup
}

func calculateDirSize(path string) int64 {
	var totalSize int64
	_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		if info, err := entry.Info(); err == nil {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize
}
