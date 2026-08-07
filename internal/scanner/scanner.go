package scanner

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"

	"dclean/internal/domain"

	"golang.org/x/sync/errgroup"
)

// maxSizeWorkers bounds how many calculateDirSize calls run at once. It is
// I/O-bound work (walking + stat-ing every file), so a small multiple of
// GOMAXPROCS overlaps disk waits without flooding the OS with goroutines.
const maxSizeWorkers = 8

type Result struct {
	Items       []domain.FoundDir
	TotalSize   int64
	ScannedDirs atomic.Int64
	mu          sync.Mutex
}

func (r *Result) add(found domain.FoundDir) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Items = append(r.Items, found)
	r.TotalSize += found.Size
}

type MultiScanner struct {
	sources []domain.ScanSource
	snapDir string
	Result  *Result
}

func New(sources []domain.ScanSource, snapDir string) *MultiScanner {
	return &MultiScanner{
		sources: sources,
		snapDir: snapDir,
		Result:  &Result{},
	}
}

// Scan walks every source and populates ms.Result. Individual source
// failures (a missing dir, a command not installed) are skipped silently
// by the per-source scanners, so this never fails as a whole.
func (ms *MultiScanner) Scan(onProgress func(int64)) {
	for _, source := range ms.sources {
		if source.Direct {
			ms.scanDirectChildren(source, onProgress)
		} else {
			ms.scanRecursive(source, onProgress)
		}
	}

	if ms.snapDir != "" {
		ms.scanSnapRevisions(onProgress)
	}

	ms.scanDocker()
	ms.scanSystem()
}

// sizeCandidate is a matched directory whose size still needs to be
// calculated; separating discovery from sizing lets the (cheap) walk stay
// serial while the (expensive) size calculation runs concurrently.
type sizeCandidate struct {
	path     string
	category string
	target   string
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
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			return nil
		}

		ms.Result.ScannedDirs.Add(1)
		ms.notifyProgress(onProgress)

		dirName := entry.Name()

		if isVersionControlDir(dirName) {
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

// resolveSizes calculates the size of every candidate concurrently, bounded
// by maxSizeWorkers, and adds the non-empty ones to the result.
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

func (ms *MultiScanner) scanSnapRevisions(onProgress func(int64)) {
	apps, err := os.ReadDir(ms.snapDir)
	if err != nil {
		return
	}

	for _, app := range apps {
		if !app.IsDir() {
			continue
		}

		appPath := filepath.Join(ms.snapDir, app.Name())
		activeRevision := readCurrentRevision(appPath)
		if activeRevision == "" {
			continue
		}

		ms.collectOldRevisions(app.Name(), appPath, activeRevision, onProgress)
		ms.collectSnapCache(app.Name(), appPath)
	}
}

func (ms *MultiScanner) collectOldRevisions(appName, appPath, activeRevision string, onProgress func(int64)) {
	entries, err := os.ReadDir(appPath)
	if err != nil {
		return
	}

	skipDirs := map[string]bool{"common": true, "current": true, activeRevision: true}

	for _, entry := range entries {
		if !entry.IsDir() || skipDirs[entry.Name()] || !isNumericString(entry.Name()) {
			continue
		}

		ms.Result.ScannedDirs.Add(1)
		ms.notifyProgress(onProgress)

		revisionPath := filepath.Join(appPath, entry.Name())
		revisionSize := calculateDirSize(revisionPath)
		if revisionSize > 0 {
			ms.Result.add(domain.FoundDir{
				Path:     revisionPath,
				Size:     revisionSize,
				Category: domain.SnapOldRevisionsCategory,
				Target:   fmt.Sprintf("%s (rev %s, active: %s)", appName, entry.Name(), activeRevision),
			})
		}
	}
}

// browser caches hold session data, wiping them logs the user out
var browserSnaps = map[string]bool{
	"brave":          true,
	"firefox":        true,
	"chromium":       true,
	"opera":          true,
	"vivaldi":        true,
	"microsoft-edge": true,
}

func (ms *MultiScanner) collectSnapCache(appName, appPath string) {
	if browserSnaps[appName] {
		return
	}

	cachePath := filepath.Join(appPath, "common", ".cache")
	info, err := os.Stat(cachePath)
	if err != nil || !info.IsDir() {
		return
	}

	cacheSize := calculateDirSize(cachePath)
	if cacheSize > domain.SnapCacheMinSize {
		ms.Result.add(domain.FoundDir{
			Path:     cachePath,
			Size:     cacheSize,
			Category: domain.SnapCacheCategory,
			Target:   appName,
		})
	}
}

func (ms *MultiScanner) notifyProgress(onProgress func(int64)) {
	if onProgress != nil && ms.Result.ScannedDirs.Load()%500 == 0 {
		onProgress(ms.Result.ScannedDirs.Load())
	}
}

type targetMatch struct {
	category string
	target   string
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

var versionControlDirs = map[string]bool{
	".git": true,
	".hg":  true,
	".svn": true,
}

func isVersionControlDir(name string) bool {
	return versionControlDirs[name]
}

func readCurrentRevision(appPath string) string {
	target, err := os.Readlink(filepath.Join(appPath, "current"))
	if err != nil {
		return ""
	}
	return target
}

func isNumericString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func calculateDirSize(path string) int64 {
	var totalSize int64
	_ = filepath.WalkDir(path, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !entry.IsDir() {
			info, err := entry.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})
	return totalSize
}

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
	if info, err := os.Stat(cachePath); err == nil && info.IsDir() {
		sources = append(sources, domain.ScanSource{
			Root:       cachePath,
			Categories: domain.HomeCacheCategories,
			Direct:     true,
		})
	}

	for _, hc := range domain.HomeDirCaches {
		root := filepath.Join(home, hc.Root)
		if info, err := os.Stat(root); err == nil && info.IsDir() {
			sources = append(sources, domain.ScanSource{
				Root:       root,
				Categories: []domain.Category{hc.Category},
				Direct:     true,
			})
		}
	}

	return sources, snapDir
}
