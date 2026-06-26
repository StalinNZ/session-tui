package config

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	Port     int
	DBPath   string
	Mem0URL  string // empty = disabled
	Headless bool   // skip TUI, only run MCP server
}

func Load() (*Config, error) {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", 8300, "MCP SSE server port")
	flag.StringVar(&cfg.DBPath, "db", "", "Path to opencode-dev.db (auto-detects if empty)")
	flag.StringVar(&cfg.Mem0URL, "mem0-url", "", "Mem0 MCP server URL (optional, e.g. http://127.0.0.1:8301/sse)")
	flag.BoolVar(&cfg.Headless, "headless", false, "Run MCP server only (no TUI)")
	flag.Parse()

	// Env overrides
	if v := os.Getenv("SESSION_TUI_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &cfg.Port)
	}
	if v := os.Getenv("SESSION_TUI_DB"); v != "" {
		cfg.DBPath = v
	}
	if v := os.Getenv("MEM0_URL"); v != "" {
		cfg.Mem0URL = v
	}
	if v := os.Getenv("SESSION_TUI_HEADLESS"); v == "1" || v == "true" {
		cfg.Headless = true
	}

	// Auto-detect DB path
	if cfg.DBPath == "" {
		home, _ := os.UserHomeDir()
		candidates := []string{
			filepath.Join(home, ".local", "share", "opencode", "opencode-dev.db"),
			filepath.Join(home, ".local", "share", "opencode", "opencode.db"),
		}
		for _, p := range candidates {
			if _, err := os.Stat(p); err == nil {
				cfg.DBPath = p
				break
			}
		}
		if cfg.DBPath == "" {
			return nil, fmt.Errorf("could not find opencode database; set --db or SESSION_TUI_DB")
		}
	}

	return cfg, nil
}

func (c *Config) Mem0Enabled() bool {
	return c.Mem0URL != ""
}
