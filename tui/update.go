package tui

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

type (
	sessionsUpdatedMsg struct{}
	agyDeletedMsg      struct{}
	statusMsg          string
)

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	case sessionsUpdatedMsg:
		sessions, err := m.DB.ListSessions(0, 0)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Error refreshing: %v", err)
			return m, nil
		}
		m.OpenCodeSessions = sessions
		if m.ActiveTab == TabOpenCode {
			m.SortSessions()
		}
		if m.Cursor >= len(m.FilteredSessions()) {
			m.Cursor = 0
		}
		return m, nil

	case agyDeletedMsg:
		m.ReloadAGY()
		if m.Cursor >= len(m.FilteredSessions()) {
			m.Cursor = 0
		}
		return m, nil

	case statusMsg:
		m.StatusMsg = string(msg)
		return m, nil
	}

	return m, nil
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.Mode {
	case ModeFilter:
		return m.handleFilterMode(msg)
	case ModeConfirmDelete:
		return m.handleConfirmMode(msg)
	case ModeRename:
		return m.handleRenameMode(msg)
	default:
		return m.handleBrowseMode(msg)
	}
}

// ── Mouse handling ──
func (m *Model) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	x, y := msg.X, msg.Y

	switch msg.Type {
	case tea.MouseLeft:
		// Tab bar click (y=1 = the tab row, 0-indexed within borders)
		if y == 1 {
			// Build tab list dynamically — same as View()
			tabs := []struct {
				mode TabMode
				width int
			}{
				{TabOpenCode, 0},
				{TabAGY, 0},
				{TabOMP, 0},
				{TabClaude, 0},
				{TabHermes, 0},
			}
			for range m.WorktreeAgents {
				tabs = append(tabs, struct {
					mode TabMode
					width int
				}{TabWorktree, 0})
			}
			ocLabel := fmt.Sprintf(" 1 %s OpenCode (%d) ", "󰈙", len(m.OpenCodeSessions))
			tabs[0].width = displayWidth(ocLabel) + 4
			agyLabel := fmt.Sprintf(" 2 %s AGY (%d) ", "󰛖", len(m.AGYSessions))
			tabs[1].width = displayWidth(agyLabel) + 4
			ompLabel := fmt.Sprintf(" 3 %s OMP (%d) ", "󰛖", len(m.OMPSessions))
			tabs[2].width = displayWidth(ompLabel) + 4
			claudeLabel := fmt.Sprintf(" 4 %s Claude (%d) ", "󰚩", len(m.ClaudeSessions))
			tabs[3].width = displayWidth(claudeLabel) + 4
			hermesLabel := fmt.Sprintf(" 5 %s Hermes (%d) ", "󰗃", len(m.HermesSessions))
			tabs[4].width = displayWidth(hermesLabel) + 4
			for i, wa := range m.WorktreeAgents {
				lbl := fmt.Sprintf(" %d %s %s (%d) ", 6+i, wa.Glyph, wa.Name, len(wa.Sessions))
				tabs[5+i].width = displayWidth(lbl) + 4
			}
			cx := 1
			for i, tab := range tabs {
				if x >= cx && x < cx+tab.width {
					if tab.mode == TabWorktree {
						idx := i - 5
						m.SwitchWorktreeTab(idx)
						m.StatusMsg = fmt.Sprintf("Tab: %s (%d sessions)", m.WorktreeAgents[idx].Name, len(m.WorktreeAgents[idx].Sessions))
					} else {
						m.SwitchTab(tab.mode)
						switch tab.mode {
						case TabOpenCode:
							m.StatusMsg = fmt.Sprintf("Tab: OpenCode (%d sessions)", len(m.OpenCodeSessions))
						case TabAGY:
							m.ReloadAGY()
							m.StatusMsg = fmt.Sprintf("Tab: AGY (%d conversations)", len(m.AGYSessions))
						case TabOMP:
							m.StatusMsg = fmt.Sprintf("Tab: OmniRoute (%d sessions)", len(m.OMPSessions))
						case TabClaude:
							m.StatusMsg = fmt.Sprintf("Tab: Claude (%d sessions)", len(m.ClaudeSessions))
						case TabHermes:
							m.StatusMsg = fmt.Sprintf("Tab: Hermes (%d sessions)", len(m.HermesSessions))
						}
					}
					return m, nil
				}
				cx += tab.width
				if i < len(tabs)-1 {
					cx += 1 // spacer
				}
			}
			return m, nil
		}

		// Click on session list row — move cursor
		sessions := m.FilteredSessions()
		h := m.Height
		maxVisible := h - 12
		if maxVisible < 1 {
			maxVisible = 1
		}
		listStartY := 5 // tab bar(1) + border(1) + header(1) + subheader(1) + divider(1) = 5
		if y >= listStartY && y < listStartY+len(sessions) && y-listStartY < len(sessions) {
			idx := y - listStartY
			// Account for scrolling offset
			half := maxVisible / 2
			scrollOff := m.Cursor - half
			if scrollOff < 0 {
				scrollOff = 0
			}
			idx += scrollOff
			if idx >= 0 && idx < len(sessions) {
				m.Cursor = idx
				return m, nil
			}
		}

		// Bottom button bar clicks
		btnY := h - 3
		if y == btnY || y == btnY-1 {
			btnW := (m.Width - 2) / 6
			if btnW < 8 {
				btnW = 8
			}
			if x >= 1 && x < 1+btnW {
				// Sort
				m.SortMode = (m.SortMode + 1) % 5
				m.SortSessions()
				m.StatusMsg = fmt.Sprintf("Sort: %s", m.SortMode)
				m.Cursor = 0
			} else if x >= 1+btnW && x < 1+btnW*2 {
				// Filter
				m.Mode = ModeFilter
				m.FilterInput.Focus()
				m.FilterInput.SetValue("")
			} else if x >= 1+btnW*2 && x < 1+btnW*3 {
				// Delete
				sel := m.SelectedSession()
				if sel != nil {
					if m.ActiveTab == TabAGY {
						m.ConfirmMsg = fmt.Sprintf("Delete AGY '%s' (%d steps)? (y/N)", sel.Title, sel.MsgCount)
						id := sel.ID // capture
						m.Mode = ModeConfirmDelete
						m.ConfirmAction = func() tea.Cmd {
							return func() tea.Msg {
								if err := m.DeleteAGY(id); err != nil {
									return statusMsg(fmt.Sprintf("Delete failed: %v", err))
								}
								return agyDeletedMsg{}
							}
						}
					} else {
						m.ConfirmMsg = fmt.Sprintf("Delete '%s' (%d msgs)? (y/N)", sel.Title, sel.MsgCount)
						m.Mode = ModeConfirmDelete
						m.ConfirmAction = func() tea.Cmd {
							return func() tea.Msg {
								if err := m.DB.DeleteSession(sel.ID); err != nil {
									return statusMsg(fmt.Sprintf("Delete failed: %v", err))
								}
								return sessionsUpdatedMsg{}
							}
						}
					}
				}
			} else if x >= 1+btnW*3 && x < 1+btnW*4 {
				// Rename
				if m.ActiveTab == TabAGY {
					m.StatusMsg = "Cannot rename AGY conversations"
					return m, nil
				}
				sel := m.SelectedSession()
				if sel != nil {
					m.Mode = ModeRename
					m.RenameID = sel.ID
					m.RenameInput.SetValue(sel.Title)
					m.RenameInput.Focus()
				}
			} else if x >= 1+btnW*4 && x < 1+btnW*5 {
				// Refresh
				m.refresh()
			} else if x >= 1+btnW*5 && x <= m.Width-1 {
				// Quit
				return m, tea.Quit
			}
		}
		return m, nil

	case tea.MouseWheelUp:
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, nil

	case tea.MouseWheelDown:
		sessions := m.FilteredSessions()
		if m.Cursor < len(sessions)-1 {
			m.Cursor++
		}
		return m, nil
	}

	return m, nil
}

func (m Model) handleBrowseMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		sessions := m.FilteredSessions()
		if m.Cursor < len(sessions)-1 {
			m.Cursor++
		}
		return m, nil

	case "k", "up":
		if m.Cursor > 0 {
			m.Cursor--
		}
		return m, nil

	case "g", "home":
		m.Cursor = 0
		return m, nil

	case "G", "end":
		m.Cursor = len(m.FilteredSessions()) - 1
		return m, nil

	case "/":
		m.Mode = ModeFilter
		m.FilterInput.Focus()
		m.FilterInput.SetValue("")
		return m, nil

	case "d":
		sel := m.SelectedSession()
		if sel == nil {
			return m, nil
		}
		if m.ActiveTab == TabAGY {
			m.ConfirmMsg = fmt.Sprintf("Delete AGY '%s' (%d steps)? (y/N)", sel.Title, sel.MsgCount)
			id := sel.ID // capture
			m.Mode = ModeConfirmDelete
			m.ConfirmAction = func() tea.Cmd {
				return func() tea.Msg {
					if err := m.DeleteAGY(id); err != nil {
						log.Printf("agy delete error: %v", err)
						return statusMsg(fmt.Sprintf("Delete failed: %v", err))
					}
					return agyDeletedMsg{}
				}
			}
		} else {
			m.ConfirmMsg = fmt.Sprintf("Delete '%s' (%d msgs)? (y/N)", sel.Title, sel.MsgCount)
			m.Mode = ModeConfirmDelete
			m.ConfirmAction = func() tea.Cmd {
				return func() tea.Msg {
					if err := m.DB.DeleteSession(sel.ID); err != nil {
						log.Printf("delete error: %v", err)
						return statusMsg(fmt.Sprintf("Delete failed: %v", err))
					}
					return sessionsUpdatedMsg{}
				}
			}
		}
		return m, nil

	case "r":
		if m.ActiveTab == TabAGY {
			m.StatusMsg = "Cannot rename AGY conversations"
			return m, nil
		}
		sel := m.SelectedSession()
		if sel == nil {
			return m, nil
		}
		m.Mode = ModeRename
		m.RenameID = sel.ID
		m.RenameInput.SetValue(sel.Title)
		m.RenameInput.Focus()
		return m, nil

	case "R":
		m.refresh()
		return m, nil

	case "s":
		m.SortMode = (m.SortMode + 1) % 5
		m.SortSessions()
		m.StatusMsg = fmt.Sprintf("Sort: %s", m.SortMode)
		m.Cursor = 0
		return m, nil

	// Tab switching: 1=OC, 2=AGY, 3=OMP, 4=Claude, 5=Hermes, 6+=worktrees
	case "1":
		if m.ActiveTab != TabOpenCode {
			m.SwitchTab(TabOpenCode)
			m.StatusMsg = fmt.Sprintf("Tab: OpenCode (%d sessions)", len(m.OpenCodeSessions))
		}
		return m, nil

	case "2":
		if m.ActiveTab != TabAGY {
			m.ReloadAGY()
			m.SwitchTab(TabAGY)
			m.StatusMsg = fmt.Sprintf("Tab: AGY (%d conversations)", len(m.AGYSessions))
		}
		return m, nil

	case "3":
		if m.ActiveTab != TabOMP {
			m.SwitchTab(TabOMP)
			m.StatusMsg = fmt.Sprintf("Tab: OmniRoute (%d sessions)", len(m.OMPSessions))
		}
		return m, nil

	case "4":
		if m.ActiveTab != TabClaude {
			m.SwitchTab(TabClaude)
			m.StatusMsg = fmt.Sprintf("Tab: Claude (%d sessions)", len(m.ClaudeSessions))
		}
		return m, nil

	case "5":
		if m.ActiveTab != TabHermes {
			m.SwitchTab(TabHermes)
			m.StatusMsg = fmt.Sprintf("Tab: Hermes (%d sessions)", len(m.HermesSessions))
		}
return m, nil
	}

	// Tab/Shift+Tab — cycle tabs forward/backward
	if msg.String() == "tab" || msg.String() == "shift+tab" {
		forward := msg.String() == "tab"
		m.cycleTab(forward)
		return m, nil
	}

	return m, nil
}

func (m Model) handleFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.FilterText = m.FilterInput.Value()
		m.Mode = ModeBrowse
		m.FilterInput.Blur()
		m.Cursor = 0
		return m, nil
	}

	var cmd tea.Cmd
	m.FilterInput, cmd = m.FilterInput.Update(msg)
	return m, cmd
}

func (m Model) handleConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.Mode = ModeBrowse
		return m, m.ConfirmAction()
	case "n", "N", "esc", "enter":
		m.Mode = ModeBrowse
		return m, nil
	}
	return m, nil
}

func (m Model) handleRenameMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		newName := m.RenameInput.Value()
		if newName != "" && m.RenameID != "" {
			return m, func() tea.Cmd {
				if err := m.DB.RenameSession(m.RenameID, newName); err != nil {
					log.Printf("rename error: %v", err)
					return func() tea.Msg { return statusMsg(fmt.Sprintf("Rename failed: %v", err)) }
				}
				return func() tea.Msg { return statusMsg("Renamed") }
			}()
		}
		m.Mode = ModeBrowse
		return m, nil
	case "esc":
		m.Mode = ModeBrowse
		return m, nil
	}

var cmd tea.Cmd
	m.RenameInput, cmd = m.RenameInput.Update(msg)
	return m, cmd
}

