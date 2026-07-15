package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"session-tui/config"
	"session-tui/db"
	srvmcp "session-tui/mcp"
	"session-tui/tui"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Config error: %v\n", err)
		os.Exit(1)
	}

	// Open database
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "DB error: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	log.Printf("[session-tui] Using database: %s", database.Path())
	if cfg.Mem0Enabled() {
		log.Printf("[session-tui] Mem0 integration enabled: %s", cfg.Mem0URL)
	} else {
		log.Printf("[session-tui] Mem0 integration disabled")
	}

	if cfg.Headless {
		runHeadless(database, cfg)
	} else {
		runWithTUI(database, cfg)
	}
}

func runHeadless(database *db.DB, cfg *config.Config) {
	// Create a minimal model (MCP server needs it for cursor state)
	model, err := tui.NewModel(database, cfg.Mem0Enabled())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Model error: %v\n", err)
		os.Exit(1)
	}

	mcpServer := srvmcp.New(database, model, cfg.Port)

	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[session-tui] Starting MCP SSE server on port %d (headless)", cfg.Port)
		if err := mcpServer.Start(); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	}()

	<-sigCh
	log.Println("[session-tui] Shutting down...")
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	mcpServer.Stop(shutdownCtx)
}

func runWithTUI(database *db.DB, cfg *config.Config) {
	// Create TUI model
	model, err := tui.NewModel(database, cfg.Mem0Enabled())
	if err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	// Create MCP server (before starting TUI so port is ready)
	mcpServer := srvmcp.New(database, model, cfg.Port)

	// Handle graceful shutdown
	_, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start MCP server in background
	go func() {
		log.Printf("[session-tui] Starting MCP SSE server on port %d", cfg.Port)
		if err := mcpServer.Start(); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	}()

	// Give MCP server a moment to bind
	time.Sleep(100 * time.Millisecond)

	// Notify MCP when TUI cursor changes
	go func() {
		prevIdx := -1
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()

		for range ticker.C {
			idx := model.SelectedIndex()
			if idx != prevIdx {
				prevIdx = idx
				mcpServer.NotifySelectedChanged(idx)
			}
		}
	}()

	// Start TUI
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseAllMotion())
	go func() {
		<-sigCh
		cancel()
		p.Quit()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		mcpServer.Stop(shutdownCtx)
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	mcpServer.Stop(shutdownCtx)
}
