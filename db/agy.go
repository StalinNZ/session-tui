package db

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type historyEntry struct {
	Display        string `json:"display"`
	Timestamp      int64  `json:"timestamp"`
	Workspace      string `json:"workspace"`
	ConversationID string `json:"conversationId"`
}

// loadHistoryTitles reads history.jsonl and builds a map of
// conversationId → first user message (display text).
func loadHistoryTitles(historyPath string) map[string]string {
	titles := make(map[string]string)

	f, err := os.Open(historyPath)
	if err != nil {
		return titles
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB lines

	for scanner.Scan() {
		var entry historyEntry
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.ConversationID == "" {
			continue
		}
		// Only record the first occurrence (oldest = first message in conversation)
		if _, exists := titles[entry.ConversationID]; !exists {
			titles[entry.ConversationID] = entry.Display
		}
	}

	return titles
}

// ListAGYConversations reads the Antigravity CLI conversation_summaries.db
// and returns sessions mapped to OpenCode's Session struct for the TUI.
func ListAGYConversations(agyDBPath, historyPath string) ([]Session, error) {
	// Check file exists
	if _, err := os.Stat(agyDBPath); os.IsNotExist(err) {
		return []Session{}, nil
	}

	conn, err := sql.Open("sqlite", agyDBPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open agy db: %w", err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	// Load fallback titles from history
	historyTitles := loadHistoryTitles(historyPath)

	rows, err := conn.Query(`
		SELECT conversation_id, title, step_count, last_modified_time, agent_name, preview
		FROM conversation_summaries
		ORDER BY last_modified_time DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query agy conversations: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var convID, title, preview, agentName, modTime string
		var stepCount int

		if err := rows.Scan(&convID, &title, &stepCount, &modTime, &agentName, &preview); err != nil {
			return nil, fmt.Errorf("scan agy conversation: %w", err)
		}

		// Parse timestamp (AGY uses ISO 8601 with timezone)
		parsed, err := time.Parse("2006-01-02 15:04:05.9999999-07:00", modTime)
		if err != nil {
			// Try without fractional seconds
			parsed, err = time.Parse("2006-01-02 15:04:05-07:00", modTime)
			if err != nil {
				parsed = time.Now()
			}
		}
		unixMs := parsed.UnixMilli()

		// Build title: use AGY title if available, else history fallback, else preview, else short ID
		displayTitle := strings.TrimSpace(title)
		if displayTitle == "" {
			if ht, ok := historyTitles[convID]; ok {
				displayTitle = ht
			}
		}
		if displayTitle == "" {
			displayTitle = strings.TrimSpace(preview)
		}
		if displayTitle == "" {
			if len(convID) > 8 {
				displayTitle = convID[:8] + "…"
			} else {
				displayTitle = convID
			}
		}
		// Truncate long titles
		runes := []rune(displayTitle)
		if len(runes) > 120 {
			displayTitle = string(runes[:117]) + "…"
		}

		if agentName == "" {
			agentName = "agy"
		}

		sessions = append(sessions, Session{
			ID:          "agy_" + convID,
			Title:       displayTitle,
			Slug:        convID,
			Agent:       agentName,
			MsgCount:    stepCount,
			TotalTokens: 0, // AGY doesn't expose token counts in summaries
			TimeCreated: unixMs,
			TimeUpdated: unixMs,
			CreatedAt:   parsed.Format("01/02 15:04"),
			UpdatedAt:   parsed.Format("01/02 15:04"),
		})
	}

	return sessions, nil
}

// AGYConversationsDir returns the path to the conversations directory
// (same parent as the summaries DB, subdirectory "conversations").
func AGYConversationsDir(agyDBPath string) string {
	return filepath.Join(filepath.Dir(agyDBPath), "conversations")
}

// DeleteAGYConversation removes an AGY conversation from both the
// summaries database and the conversations directory.
// sessionID should be in "agy_{uuid}" format.
func DeleteAGYConversation(agyDBPath, sessionID string) error {
	// Strip "agy_" prefix to get the UUID
	uuid := strings.TrimPrefix(sessionID, "agy_")
	if uuid == sessionID {
		return fmt.Errorf("not an AGY session: %s", sessionID)
	}

	// Delete from summaries DB
	conn, err := sql.Open("sqlite", agyDBPath+"?_journal_mode=WAL")
	if err != nil {
		return fmt.Errorf("open agy db: %w", err)
	}
	defer conn.Close()
	conn.SetMaxOpenConns(1)

	result, err := conn.Exec("DELETE FROM conversation_summaries WHERE conversation_id = ?", uuid)
	if err != nil {
		return fmt.Errorf("delete from summaries: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("conversation not found in summaries: %s", uuid)
	}

	// Delete the .db file from conversations directory
	convDir := AGYConversationsDir(agyDBPath)
	dbPath := filepath.Join(convDir, uuid+".db")
	if _, err := os.Stat(dbPath); err == nil {
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("delete conversation file: %w", err)
		}
	}

	return nil
}
