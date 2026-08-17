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

// All 29 Orca-supported agents
type AgentType int

const (
	AgentClaudeCode AgentType = iota
	AgentCodex
	AgentGrok
	AgentCursor
	AgentCopilot
	AgentOpenCode
	AgentMiMoCode
	AgentAmp
	AgentOpenClaude
	AgentAntigravity
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
	AgentKimi
	AgentKiro
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
	{AgentClaudeCode, "Claude Code", "󰚩", db.ListClaudeCodeSessions},
	{AgentCodex, "Codex", "󱃔", db.ListCodexSessions},
	{AgentGrok, "Grok", "󰩓", db.ListGrokSessions},
	{AgentCursor, "Cursor", "󰓬", db.ListCursorSessions},
	{AgentCopilot, "GitHub Copilot", "", db.ListCopilotSessions},
	{AgentOpenCode, "OpenCode", "󰈙", db.ListOpenCodeSessions},
	{AgentMiMoCode, "MiMo Code", "󰒊", db.ListMiMoCodeSessions},
	{AgentAmp, "Amp", "󰒼", db.ListAmpSessions},
	{AgentOpenClaude, "OpenClaude", "󰚩", db.ListOpenClaudeSessions},
	{AgentAntigravity, "Antigravity", "󰛖", db.ListAntigravitySessions},
	{AgentPi, "Pi", "󰗇", db.ListPiSessions},
	{AgentOhMyPi, "oh-my-pi", "󰍡", db.ListOhMyPiSessions},
	{AgentHermesAgent, "Hermes Agent", "󰗃", db.ListHermesSessions},
	{AgentDevin, "Devin", "󰌠", db.ListDevinSessions},
	{AgentGoose, "Goose", "󰭹", db.ListGooseSessions},
	{AgentAuggie, "Auggie", "󰰹", db.ListAuggieSessions},
	{AgentAutohandCode, "Autohand Code", "󰗧", db.ListAutohandCodeSessions},
	{AgentCharm, "Charm", "󰴹", db.ListCharmSessions},
	{AgentCline, "Cline", "󰄱", db.ListClineSessions},
	{AgentCodebuff, "Codebuff", "󰲂", db.ListCodebuffSessions},
	{AgentCommandCode, "Command Code", "󰗧", db.ListCommandCodeSessions},
	{AgentContinue, "Continue", "󰒝", db.ListContinueSessions},
	{AgentDroid, "Droid", "󰀐", db.ListDroidSessions},
	{AgentKilocode, "Kilocode", "󰥔", db.ListKilocodeSessions},
	{AgentKimi, "Kimi", "󰷚", db.ListKimiSessions},
	{AgentKiro, "Kiro", "󰀄", db.ListKiroSessions},
	{AgentMistralVibe, "Mistral Vibe", "󰾀", db.ListMistralVibeSessions},
	{AgentQwenCode, "Qwen Code", "󰛔", db.ListQwenCodeSessions},
	{AgentRovoDev, "Rovo Dev", "󰐱", db.ListRovoDevSessions},
}

type DynamicAgent struct {
	Type     AgentType
	Name     string
	Glyph    string
	Sessions []db.Session
}

type Model struct {
	DB *db.DB

	AgentSessions map[AgentType][]db.Session

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

	AGYDBPath     string
	AGYHistoryPath string
}

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
			})
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

func (m *Model) SetCurrentSessions(sessions []db.Session) {
	vis := m.VisibleAgents()
	if m.ActiveTab < 0 || m.ActiveTab >= len(vis) {
		return
	}
	m.AgentSessions[vis[m.ActiveTab].Type] = sessions
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

	agyDBPath := filepath.Join(home, ".gemini", "antigravity-cli", "conversation_summaries.db")
	agyHistoryPath := filepath.Join(home, ".gemini", "antigravity-cli", "history.jsonl")

	agySessions, _ := db.ListAGYConversations(agyDBPath, agyHistoryPath)
	if agySessions == nil {
		agySessions = []db.Session{}
	}

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
		DB:             database,
		AgentSessions:  agentSessions,
		Cursor:         0,
		Mode:           ModeBrowse,
		SortMode:       SortTimeDesc,
		FilterInput:    fi,
		RenameInput:    ri,
		Mem0Enabled:    mem0Enabled,
		AGYDBPath:      agyDBPath,
		AGYHistoryPath: agyHistoryPath,
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

func (m *Model) SwitchTab(agentIdx int) {
	if agentIdx >= 0 && agentIdx < len(m.VisibleAgents()) {
		m.ActiveTab = agentIdx
		m.Cursor = 0
		m.FilterText = ""
		m.FilterInput.SetValue("")
		m.SortSessions()
	}
}

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
	m.SortSessions()
	m.Cursor = 0
}