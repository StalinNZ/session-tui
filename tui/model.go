package tui

import (
	"fmt"
	"os"
	"path/filepath"
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

type TabMode int

const (
	TabOpenCode TabMode = iota
	TabAGY
	TabOMP
	TabClaude
	TabHermes
	TabWorktree // first dynamic worktree tab
)

func (t TabMode) String() string {
	switch t {
	case TabOpenCode:
		return "OpenCode"
	case TabAGY:
		return "AGY"
	case TabOMP:
		return "OMP"
	case TabClaude:
		return "Claude"
	case TabHermes:
		return "Hermes"
	}
	return "?"
}

func (t TabMode) Glyph() string {
	switch t {
	case TabOpenCode:
		return "󰈙"
	case TabAGY:
		return "󰛖"
	case TabOMP:
		return "󰛖"
	case TabClaude:
		return "󰚩"
	case TabHermes:
		return "󰗃"
	}
	return "?"
}

// WorktreeAgent holds sessions for a dynamic Orca worktree tab
type WorktreeAgent struct {
	Name     string // display name (e.g., "browsermcp-auth-claude")
	Glyph    string
	Sessions []db.Session
}

type Model struct {
	DB *db.DB

	// Fixed tabs
	OpenCodeSessions []db.Session
	AGYSessions      []db.Session
	OMPSessions      []db.Session
	ClaudeSessions   []db.Session
	HermesSessions   []db.Session

	// Dynamic worktree tabs
	WorktreeAgents []WorktreeAgent
	WorktreeIdx    int // current worktree tab index (0-based into WorktreeAgents)

	Cursor        int
	Mode          Mode
	ActiveTab     TabMode
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

	// AGY data source paths
	AGYDBPath     string
	AGYHistoryPath string
}

// FirstWorktreeTab returns the TabMode for the first worktree agent
func (m *Model) FirstWorktreeTab() TabMode {
	return TabWorktree
}

// CurrentSessions returns the session slice for the active tab
func (m *Model) CurrentSessions() []db.Session {
	switch m.ActiveTab {
	case TabAGY:
		return m.AGYSessions
	case TabOMP:
		return m.OMPSessions
	case TabClaude:
		return m.ClaudeSessions
	case TabHermes:
		return m.HermesSessions
	case TabWorktree:
		idx := int(m.ActiveTab) - int(TabWorktree)
		if idx >= 0 && idx < len(m.WorktreeAgents) {
			return m.WorktreeAgents[idx].Sessions
		}
		return nil
	default:
		return m.OpenCodeSessions
	}
}

func (m *Model) SetCurrentSessions(sessions []db.Session) {
	switch m.ActiveTab {
	case TabAGY:
		m.AGYSessions = sessions
	case TabOMP:
		m.OMPSessions = sessions
	case TabClaude:
		m.ClaudeSessions = sessions
	case TabHermes:
		m.HermesSessions = sessions
	case TabWorktree:
		idx := int(m.ActiveTab) - int(TabWorktree)
		if idx >= 0 && idx < len(m.WorktreeAgents) {
			m.WorktreeAgents[idx].Sessions = sessions
		}
	default:
		m.OpenCodeSessions = sessions
	}
}

func (m *Model) SortSessions() {
	sessions := m.CurrentSessions()
	if sessions == nil {
		return
	}
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
	m.SetCurrentSessions(sessions)
}

func NewModel(database *db.DB, mem0Enabled bool) (*Model, error) {
	sessions, err := database.ListSessions(0, 0)
	if err != nil {
		return nil, err
	}

	home, _ := os.UserHomeDir()
	agyDBPath := filepath.Join(home, ".gemini", "antigravity-cli", "conversation_summaries.db")
	agyHistoryPath := filepath.Join(home, ".gemini", "antigravity-cli", "history.jsonl")

	// Load AGY conversations (best-effort, non-fatal)
	agySessions, _ := db.ListAGYConversations(agyDBPath, agyHistoryPath)
	if agySessions == nil {
		agySessions = []db.Session{}
	}

	// Load OMP sessions (OmniRoute)
	ompSessions, _ := db.ListOMPSessions()
	if ompSessions == nil {
		ompSessions = []db.Session{}
	}

	// Load Claude Desktop sessions (non-worktree)
	claudeSessions, _ := db.ListClaudeSessions()
	if claudeSessions == nil {
		claudeSessions = []db.Session{}
	}

	// Load Hermes (placeholder)
	hermesSessions, _ := db.ListHermesSessions()
	if hermesSessions == nil {
		hermesSessions = []db.Session{}
	}

	// Load Orca worktree sessions
	worktreeMap, _ := db.ListWorktreeSessions()
	var worktreeAgents []WorktreeAgent
	for name, sess := range worktreeMap {
		// Sort by time desc by default
		sort.Slice(sess, func(i, j int) bool {
			return sess[i].TimeUpdated > sess[j].TimeUpdated
		})
		worktreeAgents = append(worktreeAgents, WorktreeAgent{
			Name:     name,
			Glyph:    "󰄱",
			Sessions: sess,
		})
	}
	// Deterministic order
	sort.Slice(worktreeAgents, func(i, j int) bool {
		return worktreeAgents[i].Name < worktreeAgents[j].Name
	})

	fi := textinput.New()
	fi.Placeholder = "filter sessions..."
	fi.CharLimit = 100
	fi.Width = 50

	ri := textinput.New()
	ri.Placeholder = "new session name..."
	ri.CharLimit = 200
	ri.Width = 60

	m := &Model{
		DB:               database,
		OpenCodeSessions: sessions,
		AGYSessions:      agySessions,
		OMPSessions:      ompSessions,
		ClaudeSessions:   claudeSessions,
		HermesSessions:   hermesSessions,
		WorktreeAgents:   worktreeAgents,
		Cursor:           0,
		ActiveTab:        TabOpenCode,
		Mode:             ModeBrowse,
		SortMode:         SortTimeDesc,
		FilterInput:      fi,
		RenameInput:      ri,
		Mem0Enabled:      mem0Enabled,
		AGYDBPath:        agyDBPath,
		AGYHistoryPath:   agyHistoryPath,
	}
	m.SortSessions()
	return m, nil
}

func (m *Model) FilteredSessions() []db.Session {
	all := m.CurrentSessions()
	if m.FilterText == "" {
		return all
	}
	lower := strings.ToLower(m.FilterText)
	var filtered []db.Session
	for _, s := range all {
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

func (m *Model) DeleteAGY(sessionID string) error {
	return db.DeleteAGYConversation(m.AGYDBPath, sessionID)
}

func (m *Model) ReloadAGY() {
	sessions, err := db.ListAGYConversations(m.AGYDBPath, m.AGYHistoryPath)
	if err != nil {
		m.StatusMsg = "AGY reload error"
		return
	}
	m.AGYSessions = sessions
	if m.ActiveTab == TabAGY {
		m.SortSessions()
	}
}

func (m *Model) SwitchTab(tab TabMode) {
	if tab == m.ActiveTab {
		return
	}
	m.ActiveTab = tab
	m.Cursor = 0
	m.FilterText = ""
	m.FilterInput.SetValue("")
	m.SortSessions()
}

func (m *Model) SwitchWorktreeTab(idx int) {
	if idx < 0 || idx >= len(m.WorktreeAgents) {
		return
	}
	m.ActiveTab = TabWorktree + TabMode(idx)
	m.WorktreeIdx = idx
	m.Cursor = 0
	m.FilterText = ""
	m.FilterInput.SetValue("")
	m.SortSessions()
}

// cycleTab moves forward/backward through all tabs (5 fixed + N worktree tabs)
func (m *Model) cycleTab(forward bool) {
	total := 5 + len(m.WorktreeAgents)
	if total == 0 {
		return
	}
	cur := int(m.ActiveTab)
	if forward {
		cur = (cur + 1) % total
	} else {
		cur = (cur - 1 + total) % total
	}
	switch cur {
	case 0:
		m.SwitchTab(TabOpenCode)
	case 1:
		m.ReloadAGY()
		m.SwitchTab(TabAGY)
	case 2:
		m.SwitchTab(TabOMP)
	case 3:
		m.SwitchTab(TabClaude)
	case 4:
		m.SwitchTab(TabHermes)
	default:
		m.SwitchWorktreeTab(cur - 5)
	}
	m.StatusMsg = fmt.Sprintf("Tab: %s (%d sessions)", m.ActiveTab, len(m.CurrentSessions()))
}

func (m *Model) refresh() {
	switch m.ActiveTab {
	case TabOpenCode:
		sessions, err := m.DB.ListSessions(0, 0)
		if err != nil {
			m.StatusMsg = fmt.Sprintf("Refresh error: %v", err)
			return
		}
		m.OpenCodeSessions = sessions
		m.SortSessions()
		m.StatusMsg = fmt.Sprintf("Refreshed: %d OpenCode sessions", len(sessions))
	case TabAGY:
		m.ReloadAGY()
		m.StatusMsg = fmt.Sprintf("Refreshed: %d AGY conversations", len(m.AGYSessions))
	case TabOMP:
		sessions, _ := db.ListOMPSessions()
		m.OMPSessions = sessions
		m.SortSessions()
		m.StatusMsg = fmt.Sprintf("Refreshed: %d OMP sessions", len(sessions))
	case TabClaude:
		sessions, _ := db.ListClaudeSessions()
		m.ClaudeSessions = sessions
		m.SortSessions()
		m.StatusMsg = fmt.Sprintf("Refreshed: %d Claude sessions", len(sessions))
	case TabHermes:
		sessions, _ := db.ListHermesSessions()
		m.HermesSessions = sessions
		m.SortSessions()
		m.StatusMsg = fmt.Sprintf("Refreshed: %d Hermes sessions", len(sessions))
	case TabWorktree:
		idx := int(m.ActiveTab) - int(TabWorktree)
		if idx >= 0 && idx < len(m.WorktreeAgents) {
			name := m.WorktreeAgents[idx].Name
			worktreeMap, _ := db.ListWorktreeSessions()
			if sessions, ok := worktreeMap[name]; ok {
				sort.Slice(sessions, func(i, j int) bool {
					return sessions[i].TimeUpdated > sessions[j].TimeUpdated
				})
				m.WorktreeAgents[idx].Sessions = sessions
				m.SortSessions()
				m.StatusMsg = fmt.Sprintf("Refreshed: %d %s sessions", len(sessions), name)
			}
		}
	}
	m.Cursor = 0
}