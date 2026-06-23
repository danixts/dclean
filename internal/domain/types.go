package domain

import "time"

type Category struct {
	Name    string
	Color   string
	Targets []string
}

type FoundDir struct {
	Path     string
	Size     int64
	Category string
	Target   string
	Cmd      []string
}

type ScanPath struct {
	ID        int64
	Path      string
	Label     string
	Active    bool
	CreatedAt time.Time
}

type DeletionRecord struct {
	Path      string
	Category  string
	SizeBytes int64
}

type DeletionSummary struct {
	Category   string
	TotalSize  int64
	DirCount   int
	LastDelete string
}

type ScanSource struct {
	Root       string
	Categories []Category
	Direct     bool
}
