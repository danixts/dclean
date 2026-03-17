package tui

import (
	"fmt"
	"strings"

	"dclean/internal/domain"

	"github.com/charmbracelet/lipgloss"
)

const maxPathLength = 80

var (
	ActiveBadge   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff88")).Bold(true)
	InactiveBadge = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff4444"))
)

func (m Model) viewScanning() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("dclean"))
	b.WriteString("\n\n")
	b.WriteString(StatusStyle.Render("  Scanning:"))
	b.WriteString("\n")
	for _, p := range m.ScanPaths {
		fmt.Fprintf(&b, "    %s\n", DimStyle.Render(p.Path))
	}
	b.WriteString(DimStyle.Render("    + ~/.cache + ~/snap"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) viewSelect() string {
	var b strings.Builder

	var pathLabels []string
	for _, p := range m.ScanPaths {
		pathLabels = append(pathLabels, p.Label)
	}
	header := fmt.Sprintf("dclean — %s + cache + snap  |  Total: %s",
		DimStyle.Render(strings.Join(pathLabels, ", ")),
		SizeStyle.Render(domain.FormatSize(m.TotalSize)))
	b.WriteString(TitleStyle.Render(header))
	b.WriteString("\n")

	if len(m.Groups) == 0 {
		b.WriteString("\n  No temporary/cache directories found.\n")
		b.WriteString(HelpStyle.Render("  Press q to quit"))
		return b.String()
	}

	selectedCount, selectedSize := m.SelectedStats()

	maxLines := m.Height - 8
	if maxLines < 5 {
		maxLines = 30
	}

	renderedLines := 0
	for i, group := range m.Groups {
		if renderedLines >= maxLines {
			fmt.Fprintf(&b, "  %s\n", DimStyle.Render(fmt.Sprintf("... %d more groups", len(m.Groups)-i)))
			break
		}

		renderedLines += renderGroup(&b, i, m.Cursor, group, maxLines-renderedLines)
	}

	b.WriteString("\n")
	if selectedCount > 0 {
		fmt.Fprintf(&b, "  Selected: %s (%d dirs)\n",
			DangerStyle.Render(domain.FormatSize(selectedSize)), selectedCount)
	}

	b.WriteString(HelpStyle.Render("  space: toggle  a: all  tab: expand  enter/d: delete  p: paths  q: quit"))
	b.WriteString("\n")

	return b.String()
}

func renderGroup(b *strings.Builder, index, cursor int, group GroupedItem, remaining int) int {
	cursorIcon := "  "
	if index == cursor {
		cursorIcon = "> "
	}

	checkbox := "[ ]"
	if group.Selected {
		checkbox = SelectedStyle.Render("[x]")
	}

	categoryColor := lipgloss.NewStyle().Foreground(lipgloss.Color(group.Color)).Bold(true)
	expandIcon := "▸"
	if group.Expanded {
		expandIcon = "▾"
	}

	fmt.Fprintf(b, "%s%s %s %s  %s  %s\n",
		cursorIcon, checkbox, expandIcon,
		categoryColor.Render(group.Category),
		SizeStyle.Render(domain.FormatSize(group.Size)),
		DimStyle.Render(fmt.Sprintf("(%d dirs)", len(group.Items))))

	lines := 1

	if group.Expanded {
		for _, item := range group.Items {
			if lines >= remaining {
				fmt.Fprintf(b, "      %s\n", DimStyle.Render(fmt.Sprintf("... %d more items", len(group.Items))))
				lines++
				break
			}
			fmt.Fprintf(b, "      %s  %s\n",
				DimStyle.Render(truncatePath(item.Path)),
				SizeStyle.Render(domain.FormatSize(item.Size)))
			lines++
		}
	}

	return lines
}

func (m Model) viewPaths() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("dclean — Scan Paths"))
	b.WriteString("\n\n")

	if len(m.AllPaths) == 0 {
		b.WriteString(DimStyle.Render("  No paths configured. Press 'a' to add one."))
		b.WriteString("\n")
	}

	for i, p := range m.AllPaths {
		cursor := "  "
		if i == m.PathCursor {
			cursor = "> "
		}

		var badge string
		if p.Active {
			badge = ActiveBadge.Render("active")
		} else {
			badge = InactiveBadge.Render("inactive")
		}

		fmt.Fprintf(&b, "%s[%s] %s  %s\n",
			cursor,
			badge,
			p.Path,
			DimStyle.Render("("+p.Label+")"))
	}

	if m.PathInputActive {
		b.WriteString("\n")
		fmt.Fprintf(&b, "  %s %s\n", StatusStyle.Render("New path:"), m.PathInput.View())
	}

	if m.PathError != "" {
		b.WriteString("\n")
		fmt.Fprintf(&b, "  %s\n", DangerStyle.Render(m.PathError))
	}

	b.WriteString("\n")
	if m.PathInputActive {
		b.WriteString(HelpStyle.Render("  enter: confirm  esc: cancel"))
	} else {
		b.WriteString(HelpStyle.Render("  space: toggle  a: add  x: delete  esc: back"))
	}
	b.WriteString("\n")

	return b.String()
}

func (m Model) viewConfirm() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("dclean"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "  %s\n\n",
		DangerStyle.Render(fmt.Sprintf("Delete %d directories? (%s will be freed)",
			m.DeleteCount, domain.FormatSize(m.FreedSize))))

	for _, group := range m.Groups {
		if !group.Selected {
			continue
		}
		categoryColor := lipgloss.NewStyle().Foreground(lipgloss.Color(group.Color)).Bold(true)
		fmt.Fprintf(&b, "  %s  %s\n", categoryColor.Render(group.Category), SizeStyle.Render(domain.FormatSize(group.Size)))
		for _, item := range group.Items {
			fmt.Fprintf(&b, "    %s\n", DimStyle.Render(item.Path))
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  y: confirm  n: cancel"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) viewDeleting() string {
	return TitleStyle.Render("dclean") + "\n\n" +
		StatusStyle.Render("  Deleting...") + "\n"
}

func (m Model) viewDone() string {
	var b strings.Builder
	b.WriteString(TitleStyle.Render("dclean"))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "  %s\n", SelectedStyle.Render(fmt.Sprintf("Freed %s", domain.FormatSize(m.FreedSize))))

	if len(m.Errors) > 0 {
		fmt.Fprintf(&b, "\n  %s\n", DangerStyle.Render(fmt.Sprintf("%d errors:", len(m.Errors))))
		for _, e := range m.Errors {
			fmt.Fprintf(&b, "    %s\n", e)
		}
	}

	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  Press any key to exit"))
	b.WriteString("\n")
	return b.String()
}

func truncatePath(path string) string {
	if len(path) <= maxPathLength {
		return path
	}
	return "..." + path[len(path)-(maxPathLength-3):]
}
