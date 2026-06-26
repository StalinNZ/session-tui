package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"session-tui/db"
)

type Mode int

const (
	ModeBrowse Mode = iota
	ModeFilter
	ModeConfirmDelete
	ModeRename
)

type Model struct {
	DB            *db.DB
	Sessions      []db.Session
	Cursor        int
	Mode          Mode
	FilterInput   textinput.Model
	FilterText    string
	ConfirmMsg    string
	ConfirmAction func() tea.Cmd
	RenameID      string
	RenameInput   textinput.Model
	StatusMsg     string
	Width         int
	Height        int
	Mem0Enabled   bool
}

func NewModel(database *db.DB, mem0Enabled bool) (*Model, error) {
	sessions, err := database.ListSessions(0, 0)
	if err != nil {
		return nil, err
	}

	fi := textinput.New()
	fi.Placeholder = "filter sessions..."
	fi.CharLimit = 100
	fi.Width = 50

	ri := textinput.New()
	ri.Placeholder = "new session name..."
	ri.CharLimit = 200
	ri.Width = 60

	return &Model{
		DB:          database,
		Sessions:    sessions,
		Cursor:      0,
		Mode:        ModeBrowse,
		FilterInput: fi,
		RenameInput: ri,
		Mem0Enabled: mem0Enabled,
	}, nil
}

func (m *Model) FilteredSessions() []db.Session {
	if m.FilterText == "" {
		return m.Sessions
	}
	lower := strings.ToLower(m.FilterText)
	var filtered []db.Session
	for _, s := range m.Sessions {
		if strings.Contains(strings.ToLower(s.Title), lower) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (m *Model) SelectedSession() *db.Session {
	sessions := m.FilteredSessions()
	if len(sessions) == 0 || m.Cursor < 0 || m.Cursor >= len(sessions) {
		return nil
	}
	return &sessions[m.Cursor]
}

func (m *Model) SelectedIndex() int {
	return m.Cursor
}
