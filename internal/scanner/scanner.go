package scanner

import (
	"sync"
	"sync/atomic"

	"dclean/internal/domain"
)

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

func (ms *MultiScanner) Scan(onProgress func(int64)) {
	for _, source := range ms.sources {
		if source.Direct {
			ms.scanDirectChildren(source, onProgress)
		} else {
			ms.scanRecursive(source, onProgress)
		}
	}

	if ms.snapDir != "" {
		ms.scanSnapCaches()
	}

	ms.scanDocker()
	ms.scanSystem()
}
