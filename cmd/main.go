package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"dclean/internal/domain"
	"dclean/internal/scanner"
	"dclean/internal/store"
	"dclean/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	var (
		listMode    bool
		dryRun      bool
		addPath     string
		removePath  string
		showPaths   bool
		showHistory bool
	)

	flag.BoolVar(&listMode, "list", false, "Non-interactive mode: list found directories")
	flag.BoolVar(&listMode, "l", false, "Non-interactive mode (shorthand)")
	flag.BoolVar(&dryRun, "dry-run", false, "Show what would be deleted")
	flag.StringVar(&addPath, "add", "", "Add a scan path")
	flag.StringVar(&removePath, "remove", "", "Remove a scan path")
	flag.BoolVar(&showPaths, "paths", false, "List configured scan paths")
	flag.BoolVar(&showHistory, "history", false, "Show deletion history by category")
	flag.Parse()

	db, err := store.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if !db.HasPaths() {
		db.SeedDefaults()
	}

	commands := map[bool]func(){
		addPath != "":      func() { cmdAddPath(db, addPath) },
		removePath != "":   func() { cmdRemovePath(db, removePath) },
		showPaths:          func() { cmdShowPaths(db) },
		showHistory:        func() { cmdShowHistory(db) },
		listMode || dryRun: func() { cmdList(db, dryRun) },
	}

	if cmd, exists := commands[true]; exists {
		cmd()
		return
	}

	program := tea.NewProgram(tui.NewModel(db), tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func cmdAddPath(db *store.Store, path string) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	info, err := os.Stat(absPath)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "Error: %s is not a valid directory\n", absPath)
		os.Exit(1)
	}

	if err := db.AddPath(absPath, filepath.Base(absPath)); err != nil {
		fmt.Fprintf(os.Stderr, "Error adding path: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Added: %s\n", absPath)
}

func cmdRemovePath(db *store.Store, path string) {
	paths, _ := db.ListPaths()
	for _, p := range paths {
		if p.Path == path {
			db.RemovePath(p.ID)
			fmt.Printf("Removed: %s\n", path)
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Path not found: %s\n", path)
	os.Exit(1)
}

func cmdShowPaths(db *store.Store) {
	paths, err := db.ListPaths()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(paths) == 0 {
		fmt.Println("No scan paths configured. Use --add to add paths.")
		return
	}

	statusLabels := map[bool]string{true: "active", false: "inactive"}

	fmt.Println("Configured scan paths:")
	for _, p := range paths {
		fmt.Printf("  [%s] %s (%s)\n", statusLabels[p.Active], p.Path, p.Label)
	}
}

func cmdShowHistory(db *store.Store) {
	summaries, err := db.DeletionSummaries()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if len(summaries) == 0 {
		fmt.Println("No deletion history yet.")
		return
	}

	var totalFreed int64
	fmt.Println("Deletion history by category:")
	for _, s := range summaries {
		fmt.Printf("  %-25s %10s  (%d dirs, last: %s)\n",
			s.Category, domain.FormatSize(s.TotalSize), s.DirCount, s.LastDelete)
		totalFreed += s.TotalSize
	}
	fmt.Printf("\n  Total freed: %s\n", domain.FormatSize(totalFreed))
}

func cmdList(db *store.Store, dryRun bool) {
	activePaths, _ := db.ActivePaths()
	if len(activePaths) == 0 {
		fmt.Println("No active scan paths. Use --add to configure paths.")
		return
	}

	fmt.Println("Scanning:")
	for _, p := range activePaths {
		fmt.Printf("  %s\n", p.Path)
	}
	fmt.Println("  + ~/.cache + ~/snap")
	fmt.Println()

	sources, snapDir := scanner.BuildSources(activePaths)
	sc := scanner.New(sources, snapDir)
	_ = sc.Scan(func(scanned int64) {
		fmt.Printf("\r  Scanned %d directories...", scanned)
	})
	fmt.Println()

	if len(sc.Result.Items) == 0 {
		fmt.Println("No temporary/cache directories found.")
		return
	}

	groups := tui.BuildGroups(sc.Result)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Size > groups[j].Size
	})

	for _, group := range groups {
		fmt.Printf("\n[%s] %s\n", group.Category, domain.FormatSize(group.Size))
		for _, item := range group.Items {
			fmt.Printf("  %-10s %s\n", domain.FormatSize(item.Size), item.Path)
		}
	}

	fmt.Printf("\nTotal: %s in %d directories\n",
		domain.FormatSize(sc.Result.TotalSize), len(sc.Result.Items))

	if dryRun {
		fmt.Println("(dry-run: nothing was deleted)")
	}
}
