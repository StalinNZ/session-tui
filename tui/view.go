package tui

import (
	"fmt"
	"strings"
)

func (m Model) View() string {
	var b strings.Builder

	// Header
	b.WriteString("╭─────────────────────────────────────────────────────────────╮\n")
	header := fmt.Sprintf(" 󰃤 SESSION BROWSER")
	if m.Mem0Enabled {
		header += "   mem0"
	}
	b.WriteString(fmt.Sprintf("│ %-61s │\n", header))
	b.WriteString(fmt.Sprintf("│  %d sessions           /q=quit /?=help           │\n", len(m.Sessions)))
	b.WriteString("├─────────────────────────────────────────────────────────────┤\n")

	// Session list
	sessions := m.FilteredSessions()
	start := 0
	end := len(sessions)
	maxVisible := m.Height - 10
	if maxVisible > 0 && len(sessions) > maxVisible {
		half := maxVisible / 2
		start = m.Cursor - half
		if start < 0 {
			start = 0
		}
		end = start + maxVisible
		if end > len(sessions) {
			end = len(sessions)
			start = end - maxVisible
			if start < 0 {
				start = 0
			}
		}
	}

	for i, s := range sessions {
		if i < start || i >= end {
			continue
		}
		cursor := " "
		sel := " "
		if i == m.Cursor {
			cursor = "▶"
			sel = "▓"
		}
		tokens := fmt.Sprintf("%.1fM", float64(s.TotalTokens)/1_000_000)
		if s.TotalTokens < 1_000_000 {
			tokens = fmt.Sprintf("%dK", s.TotalTokens/1000)
			if s.TotalTokens < 1000 {
				tokens = fmt.Sprintf("%d", s.TotalTokens)
			}
		}

		line := fmt.Sprintf("│ %s %s %-50s %s │",
			cursor, sel, truncate(s.Title, 50),
			fmt.Sprintf("%4d msgs %8s", s.MsgCount, tokens))

		if i == m.Cursor {
			b.WriteString(reverseLine(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	// Status / input line
	b.WriteString("╰─────────────────────────────────────────────────────────────╯\n")

	switch m.Mode {
	case ModeFilter:
		b.WriteString(fmt.Sprintf("  %s\n", m.FilterInput.View()))
	case ModeConfirmDelete:
		b.WriteString(fmt.Sprintf(" 󰩹 %s\n", m.ConfirmMsg))
	case ModeRename:
		b.WriteString(fmt.Sprintf(" 󰊕 %s\n", m.RenameInput.View()))
	default:
		if m.FilterText != "" {
			b.WriteString(fmt.Sprintf("  filter: %s\n", m.FilterText))
		}
		if m.StatusMsg != "" {
			b.WriteString(fmt.Sprintf(" %s\n", m.StatusMsg))
		} else {
			b.WriteString(" 󰌨 j/k=nav  /=filter  d=delete  r=rename  R=refresh  q=quit\n")
		}
	}

	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func reverseLine(s string) string {
	return "\033[7m" + s + "\033[0m"
}
