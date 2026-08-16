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

// All Orca-supported agents with their session loaders
type AgentType int

const (
	// Installed on your system
	AgentOpenCode AgentType = iota
	AgentClaudeCode
	AgentCursor
	AgentHermes
	AgentKimi
	AgentKiro
	AgentAntigravity

	// Not installed but supported
	AgentCodex
	AgentGrok
	AgentCopilot
	AgentMiMoCode
	AgentAmp
	AgentOpenClaude
	AgentPi
	AgentOhMyPi
	AgentHermesAgent
	AgentDevin
	AgentGoose
	AgentAuggie
	AgentAutohandCode
	AgentCharm
	AgentCline
	AgentCodebuff
	AgentCommandCode
	AgentContinue
	AgentDroid
	AgentKilocode
	AgentKimiCoding
	AgentMistralVibe
	AgentQwenCode
	AgentRovoDev
)

var AllAgents = []struct {
	Type  AgentType
	Name  string
	Glyph string
	Load  func(home string) ([]db.Session, error)
}{
	// Your installed agents (first, higher priority)
	{AgentOpenCode, "OpenCode", "󰈙", db.ListOpenCodeSessions},
	{AgentClaudeCode, "Claude Code", "󰚩", db.ListClaudeCodeSessions},
	{AgentCursor, "Cursor", "󰓬", db.ListCursorSessions},
	{AgentHermes, "Hermes", "󰗃", db.ListHermesSessions},
	{AgentKimi, "Kimi", "󰷚", db.ListKimiSessions},
	{AgentKiro, "Kiro", "󰀄", db.ListKiroSessions},
	{AgentAntigravity, "Antigravity", "󰛖", db.ListAntigravitySessions},

	// Not installed but supported
	{AgentCodex, "Codex", "󱃔", nil},
	{AgentGrok, "Grok", "󰩓", nil},
	{AgentCopilot, "GitHub Copilot", "", nil},
	{AgentMiMoCode, "MiMo Code", "󰒊", nil},
	{AgentAmp, "Amp", "󰒼", nil},
	{AgentOpenClaude, "OpenClaude", "󰚩", nil},
	{AgentPi, "Pi", "󰗇", nil},
	{AgentOhMyPi, "oh-my-pi", "󰍡", nil},
	{AgentHermesAgent, "Hermes Agent", "󰗃", nil},
	{AgentDevin, "Devin", "󰌠", nil},
	{AgentGoose, "Goose", "󰭹", nil},
	{AgentAuggie, "Auggie", "󰰹", nil},
	{AgentAutohandCode, "Autohand Code", "󰗧", nil},
	{AgentCharm, "Charm", "󰴹", nil},
	{AgentCline, "Cline", "󰄱", nil},
	{AgentCodebuff, "Codebuff", "󰲂", nil},
	{AgentCommandCode, "Command Code", "󰗧", nil},
	{AgentContinue, "Continue", "󰒝", nil},
	{AgentDroid, "Droid", "󰀐", nil},
	{AgentKilocode, "Kilocode", "󰥔", nil},
	{AgentKimiCoding, "Kimi Coding", "󰷚", nil},
	{AgentKiro, "Kiro", "󰀄", nil},
	{AgentMistralVibe, "Mistral Vibe", "󰾀", nil},
	{AgentQwenCode, "Qwen Code", "󰛔", nil},
	{AgentRovoDev, "Rovo Dev", "󰐱", nil},
}

// DynamicAgent holds sessions for a tab (either fixed agent or worktree)
type DynamicAgent struct {
	Type      AgentType
	Name      string
	Glyph     string
	Sessions  []db.Session
	Worktree  bool // true if this is a worktree tab
}

type Model struct {
	DB *db.DB

	// Agent sessions (populated on demand)
	AgentSessions map[AgentType][]db.Session

	// Dynamic worktree tabs
	WorktreeAgents []DynamicAgent
	WorktreeIdx    int // index into WorktreeAgents for the active worktree tab

	Cursor        int
	Mode          Mode
	ActiveTab     int // index into VisibleAgents()
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

// VisibleAgents returns only agents that have sessions or are worktrees
func (m *Model) VisibleAgents() []DynamicAgent {
	var visible []DynamicAgent
	for _, a := range AllAgents {
		sessions := m.AgentSessions[a.Type]
		if len(sessions) > 0 {
			visible = append(visible, DynamicAgent{
				Type:     a.Type,
				Name:     a.Name,
				Glyph:    a.Glyph,
				Sessions: sessions,
				Worktree: false,
			})
		}
	}
	// Append worktree agents
	for _, wa := range m.WorktreeAgents {
		if len(wa.Sessions) > 0 {
			visible = append(visible, wa)
		}
	}
	return visible
}

func (m *Model) CurrentSessions() []db.Session {
	vis := m.VisibleAgents()
	if m.ActiveTab >= 0 && m.ActiveTab < len(vis) {
		return vis[m.ActiveTab].Sessions
	}
	return nil
}

func (m *Model) ActiveAgent() *DynamicAgent {
	vis := m.VisibleAgents()
	if m.ActiveTab >= 0 && m.ActiveTab < len(vis) {
		return &vis[m.ActiveTab]
	}
	return nil
}

func (m *Model) SetCurrentSessions(sessions []db.Session) {
	vis := m.VisibleAgents()
	if m.ActiveTab < 0 || m.ActiveTab >= len(vis) {
		return
	}
	a := vis[m.ActiveTab]
	if a.Worktree {
		// Find the matching worktree agent and update its sessions
		for i := range m.WorktreeAgents {
			if m.WorktreeAgents[i].Name == a.Name {
				m.WorktreeAgents[i].Sessions = sessions
				break
			}
		}
	} else {
		m.AgentSessions[a.Type] = sessions
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
	home, _ := os.UserHomeDir()

	// Load AGY conversations (best-effort, non-fatal)
	agyDBPath := filepath.Join(home, ".gemini", "antigravity-cli", "conversation_summaries.db")
	agyHistoryPath := filepath.Join(home, ".gemini", "antigravity-cli", "history.jsonl")
	agySessions, _ := db.ListAGYConversations(agyDBPath, agyHistoryPath)
	if agySessions == nil {
		agySessions = []db.Session{}
	}

	// Load Orca worktree sessions
	worktreeMap, _ := db.ListWorktreeSessions()
	var worktreeAgents []DynamicAgent
	for name, sess := range worktreeMap {
		sort.Slice(sess, func(i, j int) bool {
			return sess[i].TimeUpdated > sess[j].TimeUpdated
		})
		worktreeAgents = append(worktreeAgents, DynamicAgent{
			Type:      AgentOpenCode, // placeholder type
			Name:      name,
			Glyph:     "󰄱",
			Sessions:  sess,
			Worktree:  true,
		})
	}
	sort.Slice(worktreeAgents, func(i, j int) bool {
		return worktreeAgents[i].Name < worktreeAgents[j].Name
	})

	// Initialize agent sessions map
	agentSessions := make(map[AgentType][]db.Session)
	for _, a := range AllAgents {
		if a.Load != nil {
			sessions, _ := a.Load(home)
			if sessions == nil {
				sessions = []db.Session{}
			}
			agentSessions[a.Type] = sessions
		} else {
			agentSessions[a.Type] = []db.Session{}
		}
	}
	// Antigravity uses AGY sessions
	agentSessions[AgentAntigravity] = agySessions

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
		AgentSessions:    agentSessions,
		WorktreeAgents:   worktreeAgents,
		Cursor:           0,
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
	m.AgentSessions[AgentAntigravity] = sessions
	if m.ActiveTab >= 0 {
		vis := m.VisibleAgents()
		if m.ActiveTab < len(vis) && vis[m.ActiveTab].Type == AgentAntigravity {
			m.SortSessions()
		}
	}
}

// Switch to an agent by its index in visible agents
func (m *Model) SwitchTab(agentIdx int) {
	if agentIdx >= 0 && agentIdx < len(m.VisibleAgents()) {
		m.ActiveTab = agentIdx
		m.Cursor = 0
		m.FilterText = ""
		m.FilterInput.SetValue("")
		m.SortSessions()
	}
}

func (m *Model) SwitchWorktreeTab(idx int) {
	if idx >= 0 && idx < len(m.WorktreeAgents) {
		m.ActiveTab = -1 // determined by worktree selection
		m.WorktreeIdx = idx
		m.Cursor = 0
		m.FilterText = ""
		m.FilterInput.SetValue("")
		m.ActiveTab = m.worktreeTabIndex(idx)
		m.SortSessions()
	}
}

// worktreeTabIndex returns the index in VisibleAgents() for worktree idx
func (m *Model) worktreeTabIndex(idx int) int {
	vis := m.VisibleAgents()
	start := 0
	for i, v := range vis {
		if v.Worktree {
			start = i
			break
		}
	}
	return start + idx
}

// cycleTab moves forward/backward through all VISIBLE tabs
func (m *Model) cycleTab(forward bool) {
	vis := m.VisibleAgents()
	if len(vis) == 0 {
		return
	}
	cur := m.ActiveTab
	if forward {
		cur = (cur + 1) % len(vis)
	} else {
		cur = (cur - 1 + len(vis)) % len(vis)
	}
	m.SwitchTab(cur)
	m.StatusMsg = fmt.Sprintf("Tab: %s (%d sessions)", vis[cur].Name, len(vis[cur].Sessions))
}

func (m *Model) refresh() {
	home, _ := os.UserHomeDir()
	for _, a := range AllAgents {
		if a.Load != nil {
			sessions, _ := a.Load(home)
			if sessions == nil {
				sessions = []db.Session{}
			}
			sort.Slice(sessions, func(i, j int) bool {
				return sessions[i].TimeUpdated > sessions[j].TimeUpdated
			})
			m.AgentSessions[a.Type] = sessions
		}
	}
	// Refresh worktrees
	worktreeMap, _ := db.ListWorktreeSessions()
	m.WorktreeAgents = nil
	for name, sess := range worktreeMap {
		sort.Slice(sess, func(i, j int) bool {
			return sess[i].TimeUpdated > sess[j].TimeUpdated
		})
		m.WorktreeAgents = append(m.WorktreeAgents, DynamicAgent{
			Type:      AgentOpenCode,
			Name:      name,
			Glyph:     "󰄱",
			Sessions:  sess,
			Worktree:  true,
		})
	}
	sort.Slice(m.WorktreeAgents, func(i, j int) bool {
		return m.WorktreeAgents[i].Name < m.WorktreeAgents[j].Name
	})
	m.SortSessions()
	m.Cursor = 0
}