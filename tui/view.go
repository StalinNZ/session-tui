package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Color palette per agent ──
var agentColor = map[string]lipgloss.Color{
	"kuri-osint":  lipgloss.Color("#FF4444"),
	"kuri":        lipgloss.Color("#44BBFF"),
	"build":       lipgloss.Color("#FFAA00"),
	"defender":    lipgloss.Color("#44DD44"),
	"jelly-claw":  lipgloss.Color("#FF66FF"),
	"manager":     lipgloss.Color("#88FF88"),
	"session":     lipgloss.Color("#6688FF"),
	"kuri-scout":  lipgloss.Color("#88AAAA"),
	"recon":       lipgloss.Color("#888888"),
}

var agentIcon = map[string]string{
	"kuri-osint":       "󰗃",
	"kuri":             "",
	"build":            "󱥊",
	"defender":         "󰒃",
	"jelly-claw":       "󰕥",
	"manager":          "",
	"session":          "󰃤",
	"kuri-scout":       "󰄱",
	"Wall&Port-Manager":"󰒓",
	"explore":          "󰈎",
	"recon":            "󰈎",
}

var sortIcon = map[SortMode]string{
	SortTimeDesc:  "󰔠",
	SortTokensDesc:"󰘦",
	SortMsgsDesc:  "",
	SortAgentAsc:  "",
	SortTitleAsc:  "󰉖",
}

// ── Lipgloss styles ──
var (
	styleBorder    = lipgloss.NewStyle().Foreground(lipgloss.Color("#666688"))
	styleHeader    = lipgloss.NewStyle().Foreground(lipgloss.Color("#44CCFF")).Bold(true)
	styleSelected  = lipgloss.NewStyle().Background(lipgloss.Color("#334466")).Foreground(lipgloss.Color("#FFFFFF"))
	styleCursor    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCC00"))
	styleMuted     = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	styleTokens    = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8800"))
	styleMsgs      = lipgloss.NewStyle().Foreground(lipgloss.Color("#66DDFF"))
	styleHelp      = lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	styleStatus    = lipgloss.NewStyle().Foreground(lipgloss.Color("#44FF44"))
	styleAgentTag  = lipgloss.NewStyle().Bold(true)
)

func agentStyle(agent string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(agentColorStyle(agent)).Bold(true)
}

func agentColorStyle(agent string) lipgloss.Color {
	c, ok := agentColor[agent]
	if !ok {
		return lipgloss.Color("#AAAAAA")
	}
	return c
}

func (m Model) View() string {
	var b strings.Builder
	termW := m.Width
	if termW < 60 {
		termW = 60
	}

	// ── Header ──
	b.WriteString(styleBorder.Render("╭" + strings.Repeat("─", termW-2) + "╮") + "\n")

	headerLeft := fmt.Sprintf(" %s 󰃤 SESSION BROWSER", sortIcon[m.SortMode])
	if m.Mem0Enabled {
		headerLeft += "   mem0"
	}
	headerRight := fmt.Sprintf("󰉖 %s", m.SortMode)
	padding := termW - 2 - displayWidth(headerLeft) - displayWidth(headerRight) - 2
	if padding < 1 {
		padding = 1
	}
	b.WriteString(fmt.Sprintf("│%s%s%s│\n",
		styleHeader.Render(headerLeft),
		strings.Repeat(" ", padding),
		styleMuted.Render(headerRight)))

	sub := fmt.Sprintf(" %s %s    %d sessions     s=sort  /=filter  d=delete  r=rename  R=refresh  q=quit",
		sortIcon[m.SortMode], m.SortMode, len(m.Sessions))
	b.WriteString(fmt.Sprintf("│%s│\n", styleMuted.Render(" "+truncate(sub, termW-3))))

	b.WriteString(styleBorder.Render("├" + strings.Repeat("─", termW-2) + "┤") + "\n")

	// ── Session list ──
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

	// Column widths — order: glype  agent  date  msgs  tokens  chat
	cw := struct {
		pre   int
		icon  int
		agent int
		date  int
		msgs  int
		tok   int
		title int
	}{
		pre:   1,
		icon:  1,
		agent: 11,
		date:  14,
		msgs:  5,
		tok:   7,
	}
	// Fixed: cursor+space+glyph+space+sep+space+agent+space+sep+space+date+space+sep+space+msgs+space+sep+space+tokens+space+sep+space+title+space
	fixed := 1 + 1 + 1 + 1 + 1 + 1 + cw.icon + 1 + 1 + 1 + cw.agent + 1 + 1 + 1 + cw.date + 1 + 1 + 1 + cw.msgs + 1 + 1 + 1 + cw.tok + 1
	cw.title = termW - fixed - 2
	if cw.title < 10 {
		cw.title = 10
	}

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#663399")).Render("")

	// Header
	hdr := fmt.Sprintf("│%*s %s %-*s %s %-*s %s %*s %s %*s %s %-*s │",
		cw.pre, "glype", sep,
		cw.agent, "agent", sep,
		cw.date, "date", sep,
		cw.msgs, "msgs", sep,
		cw.tok, "tokens", sep,
		cw.title, "chat")
	b.WriteString(styleMuted.Render(hdr) + "\n")
	b.WriteString(styleBorder.Render("├" + strings.Repeat("━", termW-2) + "┤") + "\n")

	for i, s := range sessions {
		if i < start || i >= end {
			continue
		}

		cursorMark := " "
		if i == m.Cursor {
			cursorMark = styleCursor.Render("")
		}

		// Glype + agent
		agent := s.Agent
		if agent == "" {
			agent = "?"
		}
		icon := agentIcon[agent]
		if icon == "" {
			icon = "󰄱"
		}
		glyph := agentStyle(agent).Render(fmt.Sprintf("%-*s", cw.icon, icon))
		agentName := truncate(agent, cw.agent)
		if len([]rune(agentName)) < cw.agent {
			agentName += strings.Repeat(" ", cw.agent-len([]rune(agentName)))
		}

		// Date
		date := s.UpdatedAt

		// Messages
		msgsStr := fmt.Sprintf("%*d", cw.msgs, s.MsgCount)

		// Tokens
		tokens := fmt.Sprintf("%.1fM", float64(s.TotalTokens)/1_000_000)
		if s.TotalTokens < 1_000_000 {
			tokens = fmt.Sprintf("%dK", s.TotalTokens/1000)
			if s.TotalTokens < 1000 {
				tokens = fmt.Sprintf("%d", s.TotalTokens)
			}
		}
		tokStr := fmt.Sprintf("%*s", cw.tok, tokens)

		// Chat title (last, takes remaining space)
		title := truncate(s.Title, cw.title)

		line := fmt.Sprintf("│%s %s %s %s %s %s %s %s %s %s %s %s │",
			cursorMark,
			glyph, sep,
			agentStyle(agent).Render(agentName), sep,
			styleMuted.Render(date), sep,
			styleMsgs.Render(msgsStr), sep,
			styleTokens.Render(tokStr), sep,
			title)

		if i == m.Cursor {
			b.WriteString(styleSelected.Render(line) + "\n")
		} else {
			b.WriteString(line + "\n")
		}
	}

	// Empty state
	if len(sessions) == 0 {
		empty := " 󰱟 No sessions match filter"
		b.WriteString(fmt.Sprintf("│%s│\n", styleMuted.Render(empty)))
	}

	// ── Footer ──
	b.WriteString(styleBorder.Render("╰" + strings.Repeat("─", termW-2) + "╯") + "\n")

	// Status / input line
	switch m.Mode {
	case ModeFilter:
		b.WriteString(fmt.Sprintf("  %s\n", m.FilterInput.View()))
	case ModeConfirmDelete:
		b.WriteString(fmt.Sprintf(" 󰩹 %s\n", m.ConfirmMsg))
	case ModeRename:
		b.WriteString(fmt.Sprintf(" 󰊕 %s\n", m.RenameInput.View()))
	default:
		if m.FilterText != "" {
			b.WriteString(styleMuted.Render(fmt.Sprintf("  filter: %s", m.FilterText)) + "\n")
		}
		if m.StatusMsg != "" {
			b.WriteString(styleStatus.Render(fmt.Sprintf(" %s", m.StatusMsg)) + "\n")
		}
	}

	// ── Button bar ──
	btnLabels := []string{"Sort", "Filter", "Delete", "Rename", "Refresh", "Quit"}
	btnKeys := []string{"s", "/", "d", "r", "R", "q"}

	btnStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#5588CC")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true)

	// Calculate button widths to fit terminal
	if termW < 60 {
		termW = 60
	}
	totalBtns := len(btnLabels)
	btnW := (termW - 2) / totalBtns
	if btnW < 8 {
		btnW = 8
	}
	// Distribute remainder
	rem := (termW - 2) - (btnW * totalBtns)

	b.WriteString(styleBorder.Render("├" + strings.Repeat("─", termW-2) + "┤") + "\n")
	b.WriteString("│")
	for i, label := range btnLabels {
		btnText := fmt.Sprintf(" %s %s ", btnKeys[i], label)
		w := btnW
		if i < rem {
			w++
		}
		padded := fmt.Sprintf("%-*s", w, truncate(btnText, w))
		b.WriteString(btnStyle.Render(padded))
	}
	b.WriteString("│\n")
	b.WriteString(styleBorder.Render("╰" + strings.Repeat("─", termW-2) + "╯") + "\n")

	return b.String()
}

func truncate(s string, n int) string {
	// Simple rune-aware truncation
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return string(runes[:n-1]) + "…"
}

func displayWidth(s string) int {
	// Count runes, CJK as 2, but NOT NerdFont PUA (E000-F8FF) which is 1
	w := 0
	for _, r := range s {
		if r >= 0x1100 && r <= 0x11FF { // Hangul
			w += 2
		} else if r >= 0x2E80 && r <= 0x2FFF { // CJK Radicals
			w += 2
		} else if r >= 0x3000 && r <= 0x303F { // CJK Symbols
			w += 2
		} else if r >= 0x3040 && r <= 0x9FFF { // Hiragana-Katakana-CJK Unified
			w += 2
		} else if r >= 0xAC00 && r <= 0xD7AF { // Hangul Syllables
			w += 2
		} else if r >= 0xF900 && r <= 0xFAFF { // CJK Compatibility
			w += 2
		} else if r >= 0xFE30 && r <= 0xFE6F { // CJK Compat Forms
			w += 2
		} else if r >= 0xFF01 && r <= 0xFF60 { // Fullwidth forms
			w += 2
		} else if r >= 0xFFE0 && r <= 0xFFE6 { // Fullwidth signs
			w += 2
		} else if r >= 0x20000 && r <= 0x2FFFF { // CJK Ext B+
			w += 2
		} else {
			w++
		}
	}
	return w
}
