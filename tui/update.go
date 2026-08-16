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
		m.AgentSessions[AgentOpenCode] = sessions
		if m.ActiveTab >= 0 {
			vis := m.VisibleAgents()
			if m.ActiveTab < len(vis) && vis[m.ActiveTab].Type == AgentOpenCode {
				m.SortSessions()
			}
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
			vis := m.VisibleAgents()
			tabs := make([]struct {
				name  string
				width int
			}, len(vis))
			for i, a := range vis {
				lbl := fmt.Sprintf(" %d %s %s (%d) ", i+1, a.Glyph, a.Name, len(a.Sessions))
				tabs[i] = struct {
					name  string
					width int
				}{a.Name, displayWidth(lbl) + 4}
			}
			cx := 1
			for i, tab := range tabs {
				if x >= cx && x < cx+tab.width {
					m.SwitchTab(i)
					m.StatusMsg = fmt.Sprintf("Tab: %s (%d sessions)", tab.name, len(m.VisibleAgents()[i].Sessions))
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
					if m.isAGY() {
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
				if m.isAGY() {
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

func (m *Model) isAGY() bool {
	if m.ActiveTab >= 0 && m.ActiveTab < len(m.VisibleAgents()) {
		return m.VisibleAgents()[m.ActiveTab].Type == AgentAntigravity
	}
	return false
}

func (m *Model) handleBrowseMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		if m.isAGY() {
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
		if m.isAGY() {
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
	}

	// Number keys 1-9 switch to visible tabs by index
	vis := m.VisibleAgents()
	if len(msg.String()) == 1 && msg.String()[0] >= '1' && msg.String()[0] <= '9' {
		idx := int(msg.String()[0] - '1')
		if idx < len(vis) {
			m.SwitchTab(idx)
			m.StatusMsg = fmt.Sprintf("Tab: %s (%d sessions)", vis[idx].Name, len(vis[idx].Sessions))
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



