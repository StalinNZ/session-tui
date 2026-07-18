package tui

import (
	"sort"
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

type SortMode int

const (
	SortTimeDesc SortMode = iota
	SortTokensDesc
	SortMsgsDesc
	SortAgentAsc
	SortTitleAsc
)

func (s SortMode) String() string {
	switch s {
	case SortTimeDesc:
		return "time"
	case SortTokensDesc:
		return "tokens"
	case SortMsgsDesc:
		return "msgs"
	case SortAgentAsc:
		return "agent"
	case SortTitleAsc:
		return "name"
	}
	return "?"
}

type Model struct {
	DB            *db.DB
	Sessions      []db.Session
	Cursor        int
	Mode          Mode
	SortMode      SortMode
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

// Sort sessions in place according to m.SortMode
func (m *Model) SortSessions() {
	sessions := m.Sessions
	switch m.SortMode {
	case SortTimeDesc:
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].TimeUpdated > sessions[j].TimeUpdated
		})
	case SortTokensDesc:
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].TotalTokens > sessions[j].TotalTokens
		})
	case SortMsgsDesc:
		sort.Slice(sessions, func(i, j int) bool {
			return sessions[i].MsgCount > sessions[j].MsgCount
		})
	case SortAgentAsc:
		sort.Slice(sessions, func(i, j int) bool {
			ai, aj := sessions[i].Agent, sessions[j].Agent
			if ai == aj {
				return sessions[i].TimeUpdated > sessions[j].TimeUpdated
			}
			return ai < aj
		})
	case SortTitleAsc:
		sort.Slice(sessions, func(i, j int) bool {
			return strings.ToLower(sessions[i].Title) < strings.ToLower(sessions[j].Title)
		})
	}
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

	m := &Model{
		DB:          database,
		Sessions:    sessions,
		Cursor:      0,
		Mode:        ModeBrowse,
		SortMode:    SortTimeDesc,
		FilterInput: fi,
		RenameInput: ri,
		Mem0Enabled: mem0Enabled,
	}
	m.SortSessions()
	return m, nil
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
