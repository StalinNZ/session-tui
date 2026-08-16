package db

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// JSONL message types
type claudeMsg struct {
	Type      string      `json:"type"`
	Message   interface{} `json:"message"`
	Timestamp string      `json:"timestamp"`
	SessionID string      `json:"sessionId"`
	Version   string      `json:"version"`
	GitBranch string      `json:"gitBranch"`
	Cwd       string      `json:"cwd"`
}

func parseTimestamp(ts string) int64 {
	if ts == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		t2, err2 := time.Parse(time.RFC3339, ts)
		if err2 != nil {
			return 0
		}
		return t2.UnixMilli()
	}
	return t.UnixMilli()
}

// ListClaudeJSONLSessions reads all .jsonl files under root and returns Session list
func ListClaudeJSONLSessions(root string) ([]Session, error) {
	var sessions []Session
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // continue
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		// Extract session ID from filename
		sessionID := strings.TrimSuffix(d.Name(), ".jsonl")

		// Parse first user message as title, count total messages
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		dec := json.NewDecoder(f)
		var title string
		var msgCount int
		var firstTime, lastTime int64

		for dec.More() {
			var msg claudeMsg
			if err := dec.Decode(&msg); err != nil {
				continue
			}
			msgCount++
			ts := parseTimestamp(msg.Timestamp)
			if firstTime == 0 {
				firstTime = ts
			}
			lastTime = ts

			// First user message -> title
			if title == "" && msg.Type == "user" {
				if m, ok := msg.Message.(map[string]interface{}); ok {
					if content, ok := m["content"].(string); ok && content != "" {
						content = cleanTitle(content)
						if content != "" {
							if len(content) > 60 {
								title = content[:60] + "…"
							} else {
								title = content
							}
						}
					}
				}
			}
		}

		if title == "" {
			title = "(no user message)"
		}
		if lastTime == 0 {
			lastTime = firstTime
		}
		if lastTime == 0 {
			lastTime = time.Now().UnixMilli()
		}

		sessions = append(sessions, Session{
			ID:          sessionID,
			Title:       title,
			Slug:        sessionID,
			Agent:       "Claude",
			MsgCount:    msgCount,
			TimeCreated: firstTime,
			TimeUpdated: lastTime,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk jsonl: %w", err)
	}
	return sessions, nil
}

// ListOMPSessions - OmniRoute sessions (OmniRoute worktree .claude/projects)
func ListOMPSessions() ([]Session, error) {
	home, _ := os.UserHomeDir()
	ompRoot := filepath.Join(home, ".claude", "projects", "C---A-UTU-LLMS-300000--A-LLM-MCP-set-OmniRoute")
	return ListClaudeJSONLSessions(ompRoot)
}

// ListClaudeSessions - Claude Desktop sessions (non-worktree dirs)
func ListClaudeSessions() ([]Session, error) {
	home, _ := os.UserHomeDir()
	claudeRoot := filepath.Join(home, ".claude", "projects")
	var sessions []Session

	entries, err := os.ReadDir(claudeRoot)
	if err != nil {
		return sessions, nil
	}

	for _, d := range entries {
		if !d.IsDir() {
			continue
		}
		// Skip worktree dirs (they have encoded paths like C--Users-Stalin-orca-workspaces-...)
		name := d.Name()
		if strings.Contains(name, "orca-workspaces") || strings.Contains(name, "A-UTU-LLMS") {
			continue
		}
		// Parse sessions from this project
		dirSessions, _ := ListClaudeJSONLSessions(filepath.Join(claudeRoot, name))
		sessions = append(sessions, dirSessions...)
	}
	return sessions, nil
}

// ListHermesSessions - placeholder for Hermes
func ListHermesSessions() ([]Session, error) {
	return []Session{}, nil
}

// ListWorktreeSessions - all Orca worktree sessions as agent tabs
func ListWorktreeSessions() (map[string][]Session, error) {
	home, _ := os.UserHomeDir()
	root := filepath.Join(home, ".claude", "projects")
	result := make(map[string][]Session)

	entries, err := os.ReadDir(root)
	if err != nil {
		return result, err
	}

	for _, d := range entries {
		if !d.IsDir() {
			continue
		}
		name := d.Name()
		// Only worktree dirs (have orca-workspaces or A-UTU in encoded name)
		if !strings.Contains(name, "orca-workspaces") && !strings.Contains(name, "A-UTU-LLMS") {
			continue
		}
		// Decode name for display
		display := decodeDirName(name)
		sessions, _ := ListClaudeJSONLSessions(filepath.Join(root, name))
		if len(sessions) > 0 {
			// Set agent name
			for i := range sessions {
				sessions[i].Agent = display
			}
			result[display] = sessions
		}
	}
	return result, nil
}

func decodeDirName(encoded string) string {
	// C--Users-Stalin-orca-workspaces-OmniRoute-browsermcp-auth-claude -> browsermcp-auth-claude
	// C---A-UTU-LLMS-300000--A-LLM-MCP-set-OmniRoute -> OmniRoute
	if strings.Contains(encoded, "orca-workspaces") {
		idx := strings.Index(encoded, "orca-workspaces")
		if idx >= 0 {
			return encoded[idx+len("orca-workspaces-"):]
		}
	}
	if strings.Contains(encoded, "A-UTU-LLMS") {
		idx := strings.Index(encoded, "A-UTU-LLMS")
		if idx >= 0 {
			// Find the last segment after the encoded path
			suffix := encoded[idx:]
			parts := strings.Split(suffix, "-")
			if len(parts) > 2 {
				return strings.Join(parts[2:], "-")
			}
			return suffix
		}
	}
	// Fallback: take last meaningful segment
	parts := strings.Split(encoded, "-")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "Users" && parts[i] != "Stalin" && parts[i] != "C" && parts[i] != "" {
			return strings.Join(parts[i:], "-")
		}
	}
	return encoded
}

// cleanTitle strips Orca dispatch preambles and returns first meaningful line
func cleanTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Skip Orca dispatch preamble
		if strings.HasPrefix(line, "You are working inside Orca") {
			continue
		}
		if strings.HasPrefix(line, "Your coordinator's terminal handle is:") {
			continue
		}
		if strings.HasPrefix(line, "Your task ID is:") {
			continue
		}
		if strings.HasPrefix(line, "=== CLI COMMANDS ===") {
			continue
		}
		if strings.HasPrefix(line, "# ") {
			continue
		}
		// Skip very long lines (likely dump)
		if len(line) > 200 {
			continue
		}
		return line
	}
	return ""
}