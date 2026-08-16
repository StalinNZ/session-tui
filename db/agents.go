package db

import (
	"os"
	"path/filepath"
	"strings"
)

func ListOpenCodeSessions(home string) ([]Session, error) {
	paths := []string{
		filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
		filepath.Join(home, ".local", "share", "opencode", "opencode-dev.db"),
	}
	var all []Session
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			db, err := Open(p)
			if err != nil {
				continue
			}
			sessions, err := db.ListSessions(0, 0)
			if err == nil {
				all = append(all, sessions...)
			}
			db.Close()
		}
	}
	seen := make(map[string]bool)
	var unique []Session
	for _, s := range all {
		if !seen[s.ID] {
			seen[s.ID] = true
			unique = append(unique, s)
		}
	}
	return unique, nil
}

func ListClaudeCodeSessions(home string) ([]Session, error) {
	root := filepath.Join(home, ".claude", "projects")
	return ListClaudeJSONLSessions(root)
}

func ListCursorSessions(home string) ([]Session, error) {
	root := filepath.Join(home, "AppData", "Roaming", "Cursor", "User", "workspaceStorage")
	return ListCursorJSONLSessions(root)
}

func ListHermesSessions(home string) ([]Session, error) {
	return []Session{}, nil
}

func ListKimiSessions(home string) ([]Session, error) {
	root := filepath.Join(home, ".kimi-code", "sessions")
	return ListKimiJSONLSessions(root)
}

func ListKiroSessions(home string) ([]Session, error) {
	root := filepath.Join(home, "AppData", "Local", "Kiro", "sessions")
	return ListKiroJSONLSessions(root)
}

func ListAntigravitySessions(home string) ([]Session, error) {
	return []Session{}, nil
}

func ListCursorJSONLSessions(root string) ([]Session, error) {
	var sessions []Session
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return sessions, nil
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		sessionsFromFile, _ := parseClaudeJSONL(path)
		sessions = append(sessions, sessionsFromFile...)
		return nil
	})
	return sessions, err
}

func ListKimiJSONLSessions(root string) ([]Session, error) {
	return []Session{}, nil
}

func ListKiroJSONLSessions(root string) ([]Session, error) {
	return []Session{}, nil
}


