package tui

import (
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

type (
	sessionsUpdatedMsg struct{}
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

	case sessionsUpdatedMsg:
		sessions, err := m.DB.ListSessions(0, 0)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Error refreshing: %v", err)
			return m, nil
		}
		m.Sessions = sessions
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
		m.ConfirmMsg = fmt.Sprintf("Delete '%s' (%d msgs)? (y/N)", sel.Title, sel.MsgCount)
		m.Mode = ModeConfirmDelete
		m.ConfirmAction = func() tea.Cmd {
			// Run delete, then return refresh command
			return func() tea.Msg {
				if err := m.DB.DeleteSession(sel.ID); err != nil {
					log.Printf("delete error: %v", err)
					return statusMsg(fmt.Sprintf("Delete failed: %v", err))
				}
				return sessionsUpdatedMsg{}
			}
		}
		return m, nil

	case "r":
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
		sessions, err := m.DB.ListSessions(0, 0)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Refresh error: %v", err)
		} else {
			m.Sessions = sessions
			m.StatusMsg = "Refreshed"
		}
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
