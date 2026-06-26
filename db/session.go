package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Session struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	MsgCount     int    `json:"msg_count"`
	TokensInput  int64  `json:"tokens_input"`
	TokensOutput int64  `json:"tokens_output"`
	TotalTokens  int64  `json:"total_tokens"`
	Cost         float64 `json:"cost"`
	TimeCreated  int64  `json:"time_created"`
	TimeUpdated  int64  `json:"time_updated"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type Stats struct {
	TotalSessions int     `json:"total_sessions"`
	TotalMessages int     `json:"total_messages"`
	TotalTokens   int64   `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
	TrashCount    int     `json:"trash_count"`
	AvgMessages   float64 `json:"avg_messages"`
}

type DB struct {
	conn *sql.DB
	path string
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1)
	return &DB{conn: conn, path: path}, nil
}

func (d *DB) Close() error {
	return d.conn.Close()
}

func (d *DB) Path() string { return d.path }

func (d *DB) ListSessions(limit, offset int) ([]Session, error) {
	query := `
		SELECT s.id, s.title, s.slug, s.time_created, s.time_updated,
		       s.tokens_input, s.tokens_output, s.cost,
		       (SELECT COUNT(*) FROM message m WHERE m.session_id = s.id) as msg_count
		FROM session s
		WHERE s.title NOT LIKE '%@general%' AND s.title NOT LIKE '%subagent%'
		ORDER BY s.time_updated DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}

	rows, err := d.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.Title, &s.Slug, &s.TimeCreated, &s.TimeUpdated,
			&s.TokensInput, &s.TokensOutput, &s.Cost, &s.MsgCount); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.TotalTokens = s.TokensInput + s.TokensOutput
		s.CreatedAt = tsToStr(s.TimeCreated)
		s.UpdatedAt = tsToStr(s.TimeUpdated)
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (d *DB) SearchSessions(q string, limit int) ([]Session, error) {
	query := `
		SELECT s.id, s.title, s.slug, s.time_created, s.time_updated,
		       s.tokens_input, s.tokens_output, s.cost,
		       (SELECT COUNT(*) FROM message m WHERE m.session_id = s.id) as msg_count
		FROM session s
		WHERE s.title LIKE ? AND s.title NOT LIKE '%@general%' AND s.title NOT LIKE '%subagent%'
		ORDER BY s.time_updated DESC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	pattern := "%" + strings.ReplaceAll(q, " ", "%") + "%"
	rows, err := d.conn.Query(query, pattern)
	if err != nil {
		return nil, fmt.Errorf("search sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.Title, &s.Slug, &s.TimeCreated, &s.TimeUpdated,
			&s.TokensInput, &s.TokensOutput, &s.Cost, &s.MsgCount); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		s.TotalTokens = s.TokensInput + s.TokensOutput
		s.CreatedAt = tsToStr(s.TimeCreated)
		s.UpdatedAt = tsToStr(s.TimeUpdated)
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (d *DB) GetSession(id string) (*Session, error) {
	query := `
		SELECT s.id, s.title, s.slug, s.time_created, s.time_updated,
		       s.tokens_input, s.tokens_output, s.cost,
		       (SELECT COUNT(*) FROM message m WHERE m.session_id = s.id) as msg_count
		FROM session s WHERE s.id = ?
	`
	var s Session
	err := d.conn.QueryRow(query, id).Scan(&s.ID, &s.Title, &s.Slug, &s.TimeCreated, &s.TimeUpdated,
		&s.TokensInput, &s.TokensOutput, &s.Cost, &s.MsgCount)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	s.TotalTokens = s.TokensInput + s.TokensOutput
	s.CreatedAt = tsToStr(s.TimeCreated)
	s.UpdatedAt = tsToStr(s.TimeUpdated)
	return &s, nil
}

func (d *DB) DeleteSession(id string) error {
	_, err := d.conn.Exec("DELETE FROM message WHERE session_id = ?", id)
	if err != nil {
		return fmt.Errorf("delete messages: %w", err)
	}
	_, err = d.conn.Exec("DELETE FROM session WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (d *DB) RenameSession(id, newTitle string) error {
	_, err := d.conn.Exec("UPDATE session SET title = ? WHERE id = ?", newTitle, id)
	return err
}

func (d *DB) GetStats() (*Stats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(tokens_input + tokens_output), 0) as total_tokens,
			COALESCE(SUM(cost), 0) as total_cost,
			SUM((SELECT COUNT(*) FROM message m WHERE m.session_id = s.id)) as total_msgs
		FROM session s
		WHERE s.title NOT LIKE '%@general%' AND s.title NOT LIKE '%subagent%'
	`
	var stats Stats
	err := d.conn.QueryRow(query).Scan(&stats.TotalSessions, &stats.TotalTokens, &stats.TotalCost, &stats.TotalMessages)
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}
	if stats.TotalSessions > 0 {
		stats.AvgMessages = float64(stats.TotalMessages) / float64(stats.TotalSessions)
	}
	// Count trash (< 5 msgs)
	trashQuery := `
		SELECT COUNT(*) FROM session s
		WHERE (SELECT COUNT(*) FROM message m WHERE m.session_id = s.id) < 5
		AND s.title NOT LIKE '%@general%' AND s.title NOT LIKE '%subagent%'
	`
	d.conn.QueryRow(trashQuery).Scan(&stats.TrashCount)
	return &stats, nil
}

func tsToStr(ms int64) string {
	t := time.UnixMilli(ms)
	return t.Format("01/02 15:04")
}
