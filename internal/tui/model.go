package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"dclean/internal/domain"
	"dclean/internal/scanner"
	"dclean/internal/store"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type ViewMode int

const (
	ModeScanning ViewMode = iota
	ModeSelect
	ModeConfirm
	ModeDeleting
	ModeDone
	ModePaths
)

type GroupedItem struct {
	Category string
	Color    string
	Items    []domain.FoundDir
	Size     int64
	Selected bool
	Expanded bool
}

type Model struct {
	Groups      []GroupedItem
	Cursor      int
	Mode        ViewMode
	TotalSize   int64
	FreedSize   int64
	DeleteCount int
	Errors      []string
	Width       int
	Height      int
	Store       *store.Store
	ScanPaths   []domain.ScanPath

	AllPaths        []domain.ScanPath
	PathCursor      int
	PathInput       textinput.Model
	PathInputActive bool
	PathError       string
}

type ScanDoneMsg struct {
	Result *scanner.Result
}

type DeleteDoneMsg struct {
	Freed   int64
	Errors  []string
	Deleted []domain.FoundDir
}

type PathsLoadedMsg struct {
	Paths []domain.ScanPath
}

func NewModel(s *store.Store) Model {
	paths, _ := s.ActivePaths()

	ti := textinput.New()
	ti.Placeholder = "Enter absolute path..."
	ti.CharLimit = 256
	ti.Width = 60

	return Model{
		Mode:      ModeScanning,
		Store:     s,
		ScanPaths: paths,
		PathInput: ti,
	}
}

func (m Model) Init() tea.Cmd {
	return m.scanCmd()
}

func (m Model) scanCmd() tea.Cmd {
	paths := m.ScanPaths
	return func() tea.Msg {
		sources, snapDir := scanner.BuildSources(paths)
		sc := scanner.New(sources, snapDir)
		_ = sc.Scan(nil)
		return ScanDoneMsg{Result: sc.Result}
	}
}

func (m Model) loadPathsCmd() tea.Cmd {
	s := m.Store
	return func() tea.Msg {
		paths, _ := s.ListPaths()
		return PathsLoadedMsg{Paths: paths}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case ScanDoneMsg:
		m.Mode = ModeSelect
		m.Groups = BuildGroups(msg.Result)
		m.TotalSize = msg.Result.TotalSize
		return m, nil

	case DeleteDoneMsg:
		m.Mode = ModeDone
		m.FreedSize = msg.Freed
		m.Errors = msg.Errors
		for _, item := range msg.Deleted {
			m.Store.RecordDeletion(domain.DeletionRecord{
				Path:      item.Path,
				Category:  item.Category,
				SizeBytes: item.Size,
			})
		}
		return m, nil

	case PathsLoadedMsg:
		m.AllPaths = msg.Paths
		if m.PathCursor >= len(m.AllPaths) {
			m.PathCursor = max(0, len(m.AllPaths)-1)
		}
		return m, nil

	case tea.KeyMsg:
		handlers := map[ViewMode]func(tea.KeyMsg) (Model, tea.Cmd){
			ModeSelect:  m.handleSelectKeys,
			ModeConfirm: m.handleConfirmKeys,
			ModeDone:    func(_ tea.KeyMsg) (Model, tea.Cmd) { return m, tea.Quit },
			ModePaths:   m.handlePathsKeys,
		}
		if handler, exists := handlers[m.Mode]; exists {
			return handler(msg)
		}
	}

	return m, nil
}

func (m Model) handleSelectKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	if key == "q" || key == "ctrl+c" {
		return m, tea.Quit
	}

	if key == "enter" || key == "d" {
		return m.startDelete()
	}

	if key == "p" {
		m.Mode = ModePaths
		m.PathCursor = 0
		m.PathError = ""
		m.PathInputActive = false
		return m, m.loadPathsCmd()
	}

	actions := map[string]func(){
		"up":   func() { m.moveCursor(-1) },
		"k":    func() { m.moveCursor(-1) },
		"down": func() { m.moveCursor(1) },
		"j":    func() { m.moveCursor(1) },
		" ":    func() { m.toggleCurrentGroup() },
		"a":    func() { m.toggleAllGroups() },
		"tab":  func() { m.toggleExpand() },
		"e":    func() { m.toggleExpand() },
	}

	if action, exists := actions[key]; exists {
		action()
	}

	return m, nil
}

func (m Model) handlePathsKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.PathInputActive {
		return m.handlePathInput(msg)
	}

	key := msg.String()

	if key == "esc" || key == "q" {
		m.ScanPaths, _ = m.Store.ActivePaths()
		m.Mode = ModeScanning
		return m, m.scanCmd()
	}

	if key == "ctrl+c" {
		return m, tea.Quit
	}

	if key == "a" {
		m.PathInputActive = true
		m.PathInput.Reset()
		m.PathInput.Focus()
		m.PathError = ""
		return m, textinput.Blink
	}

	if len(m.AllPaths) == 0 {
		return m, nil
	}

	actions := map[string]func() tea.Cmd{
		"up":   func() tea.Cmd { m.movePathCursor(-1); return nil },
		"k":    func() tea.Cmd { m.movePathCursor(-1); return nil },
		"down": func() tea.Cmd { m.movePathCursor(1); return nil },
		"j":    func() tea.Cmd { m.movePathCursor(1); return nil },
		" ": func() tea.Cmd {
			m.Store.TogglePath(m.AllPaths[m.PathCursor].ID)
			return m.loadPathsCmd()
		},
		"x": func() tea.Cmd {
			m.Store.RemovePath(m.AllPaths[m.PathCursor].ID)
			return m.loadPathsCmd()
		},
	}

	if action, exists := actions[key]; exists {
		cmd := action()
		return m, cmd
	}

	return m, nil
}

func (m Model) handlePathInput(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	if key == "esc" {
		m.PathInputActive = false
		m.PathError = ""
		return m, nil
	}

	if key == "enter" {
		inputPath := m.PathInput.Value()
		if inputPath == "" {
			m.PathInputActive = false
			return m, nil
		}

		absPath, err := filepath.Abs(inputPath)
		if err != nil {
			m.PathError = fmt.Sprintf("invalid path: %v", err)
			return m, nil
		}

		info, err := os.Stat(absPath)
		if err != nil || !info.IsDir() {
			m.PathError = fmt.Sprintf("'%s' is not a valid directory", absPath)
			return m, nil
		}

		m.Store.AddPath(absPath, filepath.Base(absPath))
		m.PathInputActive = false
		m.PathError = ""
		return m, m.loadPathsCmd()
	}

	var cmd tea.Cmd
	m.PathInput, cmd = m.PathInput.Update(msg)
	m.PathError = ""
	return m, cmd
}

func (m *Model) moveCursor(delta int) {
	newPos := m.Cursor + delta
	if newPos >= 0 && newPos < len(m.Groups) {
		m.Cursor = newPos
	}
}

func (m *Model) movePathCursor(delta int) {
	newPos := m.PathCursor + delta
	if newPos >= 0 && newPos < len(m.AllPaths) {
		m.PathCursor = newPos
	}
}

func (m *Model) toggleCurrentGroup() {
	if m.Cursor < len(m.Groups) {
		m.Groups[m.Cursor].Selected = !m.Groups[m.Cursor].Selected
	}
}

func (m *Model) toggleAllGroups() {
	allSelected := true
	for _, group := range m.Groups {
		if !group.Selected {
			allSelected = false
			break
		}
	}
	for i := range m.Groups {
		m.Groups[i].Selected = !allSelected
	}
}

func (m *Model) toggleExpand() {
	if m.Cursor < len(m.Groups) {
		m.Groups[m.Cursor].Expanded = !m.Groups[m.Cursor].Expanded
	}
}

func (m Model) startDelete() (Model, tea.Cmd) {
	count, size := m.SelectedStats()
	if count == 0 {
		return m, nil
	}
	m.DeleteCount = count
	m.FreedSize = size
	m.Mode = ModeConfirm
	return m, nil
}

func (m Model) handleConfirmKeys(msg tea.KeyMsg) (Model, tea.Cmd) {
	key := msg.String()

	if key == "y" || key == "Y" {
		m.Mode = ModeDeleting
		groups := m.Groups
		return m, func() tea.Msg {
			var freed int64
			var errors []string
			var deleted []domain.FoundDir
			for _, group := range groups {
				if !group.Selected {
					continue
				}
				for _, item := range group.Items {
					var err error
					if len(item.Cmd) > 0 {
						err = exec.Command(item.Cmd[0], item.Cmd[1:]...).Run()
					} else {
						err = os.RemoveAll(item.Path)
					}
					if err != nil {
						errors = append(errors, fmt.Sprintf("%s: %v", item.Path, err))
					} else {
						freed += item.Size
						deleted = append(deleted, item)
					}
				}
			}
			return DeleteDoneMsg{Freed: freed, Errors: errors, Deleted: deleted}
		}
	}

	cancelKeys := map[string]bool{"n": true, "N": true, "esc": true, "q": true}
	if cancelKeys[key] {
		m.Mode = ModeSelect
	}

	return m, nil
}

func (m Model) SelectedStats() (int, int64) {
	var count int
	var size int64
	for _, group := range m.Groups {
		if group.Selected {
			count += len(group.Items)
			size += group.Size
		}
	}
	return count, size
}

func (m Model) View() string {
	views := map[ViewMode]func() string{
		ModeScanning: m.viewScanning,
		ModeSelect:   m.viewSelect,
		ModeConfirm:  m.viewConfirm,
		ModeDeleting: m.viewDeleting,
		ModeDone:     m.viewDone,
		ModePaths:    m.viewPaths,
	}

	if render, exists := views[m.Mode]; exists {
		return render()
	}
	return ""
}

func BuildGroups(result *scanner.Result) []GroupedItem {
	colorMap := buildColorMap()
	groupMap := make(map[string]*GroupedItem)

	for _, item := range result.Items {
		group, exists := groupMap[item.Category]
		if !exists {
			color := colorMap[item.Category]
			if color == "" {
				color = "#aaaaaa"
			}
			group = &GroupedItem{Category: item.Category, Color: color}
			groupMap[item.Category] = group
		}
		group.Items = append(group.Items, item)
		group.Size += item.Size
	}

	groups := make([]GroupedItem, 0, len(groupMap))
	for _, group := range groupMap {
		sort.Slice(group.Items, func(i, j int) bool {
			return group.Items[i].Size > group.Items[j].Size
		})
		groups = append(groups, *group)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Size > groups[j].Size
	})

	return groups
}

func buildColorMap() map[string]string {
	colorMap := make(map[string]string)
	allCategories := append(domain.ProjectCategories, domain.HomeCacheCategories...)
	for _, cat := range allCategories {
		colorMap[cat.Name] = cat.Color
	}
	for _, hc := range domain.HomeDirCaches {
		colorMap[hc.Category.Name] = hc.Category.Color
	}
	colorMap[domain.SnapOldRevisionsCategory] = domain.SnapOldRevisionsColor
	colorMap[domain.SnapCacheCategory] = domain.SnapCacheColor
	colorMap[domain.DockerOrphanVolumeCategory] = domain.DockerOrphanVolumeColor
	colorMap[domain.DockerSystemPruneCategory] = domain.DockerSystemPruneColor
	return colorMap
}
