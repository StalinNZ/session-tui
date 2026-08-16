package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// ── Deterministic per-agent color ──
// Uses FNV hash to distribute agents evenly around HSL color wheel.
// Every agent name gets a unique color. Zero maintenance.
func agentColorStyle(agent string) lipgloss.Color {
	// Named overrides
	switch agent {
	case "gis", "Kuri-QGIS":
		return lipgloss.Color("#228833") // dark green
	case "Kuri-Red":
		return lipgloss.Color("#CC3333") // red
	case "Kuri-Blue":
		return lipgloss.Color("#3366CC") // blue
	case "Kuri-Session", "session":
		return lipgloss.Color("#8844CC") // purple
	case "agy":
		return lipgloss.Color("#FF8800") // orange for Antigravity
	}
	if agent == "" {
		return lipgloss.Color("#888888")
	}
	h := fnvHash(agent)
	// Hue: 0-360 (skip red 340-20 for readability with dark bg)
	hue := int(h%300 + 20)
	if hue >= 340 {
		hue -= 60
	}
	// Saturation: 70-90%
	sat := 70 + int(h%21)
	// Lightness: 55-75%
	lit := 55 + int(h%21)
	r, g, b := hslToRGB(hue, sat, lit)
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func fnvHash(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}

func hslToRGB(h, s, l int) (r, g, b int) {
	// h in 0-360, s/l in 0-100
	H := float64(h) / 360.0
	S := float64(s) / 100.0
	L := float64(l) / 100.0

	var rF, gF, bF float64
	if S == 0 {
		rF, gF, bF = L, L, L
	} else {
		var hue2rgb func(p, q, t float64) float64
		hue2rgb = func(p, q, t float64) float64 {
			if t < 0 {
				t += 1
			}
			if t > 1 {
				t -= 1
			}
			if t < 1.0/6.0 {
				return p + (q-p)*6.0*t
			}
			if t < 1.0/2.0 {
				return q
			}
			if t < 2.0/3.0 {
				return p + (q-p)*(2.0/3.0-t)*6.0
			}
			return p
		}

		var q float64
		if L < 0.5 {
			q = L * (1.0 + S)
		} else {
			q = L + S - L*S
		}
		p := 2.0*L - q
		rF = hue2rgb(p, q, H+1.0/3.0)
		gF = hue2rgb(p, q, H)
		bF = hue2rgb(p, q, H-1.0/3.0)
	}

	r = int(rF * 255)
	g = int(gF * 255)
	b = int(bF * 255)
	if r > 255 {
		r = 255
	}
	if g > 255 {
		g = 255
	}
	if b > 255 {
		b = 255
	}
	return
}

var agentIcon = map[string]string{
	"kuri-osint":             "󰗃",
	"kuri":                   "",
	"build":                  "󱥊",
	"defender":               "󰒃",
	"jelly-claw":             "󰕥",
	"manager":                "",
	"session":                "󰃤",
	"Kuri-Session":           "󰈙",
	"kuri-scout":             "󰄱",
	"Wall&Port-Manager":      "󰒓",
	"explore":                "󰈎",
	"Kuri":                   "󰈙",
	"Kuri-Manager":           "󱂅",
	"Kuri-QGIS":              "󰛖",
	"Kuri-Red":               "󰃤",
	"Kuri-Blue":              "󱃂",
	"Kuri-Indexer":           "󰋁",
	"Kuri-Linux":             "",
	"Kuri-LLM":               "󱙷",
	"Kuri-Web-Scraper":       "󰖟",
	"Kuri-Windows":           "",
	"Kuri-OSINT":             "󰗃",
	"Kuri-Port-Man":          "󰒓",
	"general":                "󰛨",
	"gis":                    "󰈸",
	"overseer":               "󰼭",
	"logs/session_audit_report":"󰇧",
	"logs/app_audit_2026-07-18":"󰅟",
	"logs/sysinternals_audit_2026-07-18":"",
	"logs/mcp_hub_agents_audit":"󱂅",
	"logs/vault_mining_report":"󰋁",
	"logs/vault_sort_report":  "󰄱",
	"agy":                   "󰛖",
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
)

// Tab styles
var (
	tabActive   = lipgloss.NewStyle().Background(lipgloss.Color("#5588CC")).Foreground(lipgloss.Color("#FFFFFF")).Bold(true).Padding(0, 2)
	tabInactive = lipgloss.NewStyle().Background(lipgloss.Color("#333355")).Foreground(lipgloss.Color("#AAAAAA")).Padding(0, 2)
	tabSpacer   = lipgloss.NewStyle().Foreground(lipgloss.Color("#666688"))
)

func agentStyle(agent string) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(agentColorStyle(agent)).Bold(true)
}

func (m Model) View() string {
	var b strings.Builder
	termW := m.Width
	if termW < 60 {
		termW = 60
	}

	// ── Tab bar ──
	vis := m.VisibleAgents()
	tabs := []struct {
		name  string
		glyph string
		count int
	}{
		{name: "Nothing", glyph: "󱫟", count: 0},
	}
	tabs = tabs[:0]
	for _, a := range vis {
		tabs = append(tabs, struct {
			name  string
			glyph string
			count int
		}{a.Name, a.Glyph, len(a.Sessions)})
	}

	b.WriteString(styleBorder.Render("╭" + strings.Repeat("─", termW-2) + "╮") + "\n")
	b.WriteString("│")
	// Track tab content width so we can fill remaining space
	tabContentW := 0
	for i, tab := range tabs {
		num := i + 1
		label := fmt.Sprintf(" %d %s %s (%d) ", num, tab.glyph, tab.name, tab.count)
		if m.ActiveTab == i {
			b.WriteString(tabActive.Render(label))
		} else {
			b.WriteString(tabInactive.Render(label))
		}
		tabContentW += displayWidth(label) + 4
		if i < len(tabs)-1 {
			b.WriteString(tabSpacer.Render(" "))
			tabContentW += 1
		}
	}
	// Fill remaining width
	remain := termW - 2 - tabContentW
	if remain < 0 {
		remain = 0
	}
	b.WriteString(strings.Repeat(" ", remain))
	b.WriteString("│\n")

	// ── Header ──
	b.WriteString(styleBorder.Render("├" + strings.Repeat("─", termW-2) + "┤") + "\n")

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

	sub := fmt.Sprintf(" %s %s    %d sessions     s=sort  /=filter  d=delete  r=rename  R=refresh  q=quit  Tab=cycle",
		sortIcon[m.SortMode], m.SortMode, len(m.FilteredSessions()))
	b.WriteString(fmt.Sprintf("│%s│\n", styleMuted.Render(" "+truncate(sub, termW-3))))

	b.WriteString(styleBorder.Render("├" + strings.Repeat("─", termW-2) + "┤") + "\n")

	// ── Session list ──
	sessions := m.FilteredSessions()
	start := 0
	end := len(sessions)
	// Tab bar + top border + header + subheader + divider = 5 lines extra now
	maxVisible := m.Height - 12
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

	// Column widths — order: num  glype  agent  date  msgs  tokens  chat
	cw := struct {
		pre   int
		num   int
		icon  int
		agent int
		date  int
		msgs  int
		tok   int
		title int
	}{
		pre:   1,
		num:   4,
		icon:  1,
		agent: 11,
		date:  14,
		msgs:  5,
		tok:   7,
	}
	// Fixed: cursor+space+glyph+space+sep+space+agent+space+sep+space+date+space+sep+space+msgs+space+sep+space+tokens+space+sep+space+title+space
	fixed := 1 + 1 + 1 + 1 + 1 + 1 + cw.num + 1 + 1 + 1 + cw.icon + 1 + 1 + 1 + cw.agent + 1 + 1 + 1 + cw.date + 1 + 1 + 1 + cw.msgs + 1 + 1 + 1 + cw.tok + 1
	cw.title = termW - fixed - 2
	if cw.title < 10 {
		cw.title = 10
	}

	sep := lipgloss.NewStyle().Foreground(lipgloss.Color("#663399")).Render("")

	// Header
	hdr := fmt.Sprintf("│%*s %s %-*s %s %-*s %s %-*s %s %*s %s %*s %s %-*s │",
		cw.pre, "#", sep,
		cw.num, "session", sep,
		cw.icon, "glype", sep,
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
			cursorMark = styleCursor.Render("󰜴")
		}

		// Row number (visible index)
		rowNum := i - start + 1
		numStr := fmt.Sprintf("%-*d", cw.num, rowNum)

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
		var tokStr string
		if s.TotalTokens > 0 {
			tokens := fmt.Sprintf("%.1fM", float64(s.TotalTokens)/1_000_000)
			if s.TotalTokens < 1_000_000 {
				tokStr = fmt.Sprintf("%dK", s.TotalTokens/1000)
				if s.TotalTokens < 1000 {
					tokStr = fmt.Sprintf("%d", s.TotalTokens)
				}
			} else {
				tokStr = tokens
			}
		} else {
			tokStr = "—"
		}
		tokStr = fmt.Sprintf("%*s", cw.tok, tokStr)

		// Chat title
		title := truncate(s.Title, cw.title)

		line := fmt.Sprintf("│%s %s %s %s %s %s %s %s %s %s %s %s %s │",
			cursorMark,
			styleMuted.Render(numStr), sep,
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
		var empty string
		if m.FilterText != "" {
			empty = " 󰱟 No sessions match filter"
		} else if m.isAGY() {
			empty = " 󰱟 No AGY conversations found"
		} else {
			empty = " 󰱟 No sessions"
		}
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
