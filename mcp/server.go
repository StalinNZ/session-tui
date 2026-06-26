package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"

	"session-tui/db"
	"session-tui/tui"
)

type SessionMCPServer struct {
	mcpServer   *server.MCPServer
	sseServer   *server.SSEServer
	db          *db.DB
	model       *tui.Model
	mu          sync.RWMutex
	selectedIdx int
	port        int
	notifyCh    chan struct{}
}

func New(database *db.DB, tuiModel *tui.Model, port int) *SessionMCPServer {
	s := &SessionMCPServer{
		db:        database,
		model:     tuiModel,
		port:      port,
		selectedIdx: 0,
		notifyCh:  make(chan struct{}, 100),
	}

	s.mcpServer = server.NewMCPServer(
		"session-tui",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	s.registerResources()
	s.registerTools()

	return s
}

func (s *SessionMCPServer) Start() error {
	sseServer := server.NewSSEServer(s.mcpServer,
		server.WithBaseURL(fmt.Sprintf("http://127.0.0.1:%d", s.port)),
		server.WithBasePath("/"),
	)
	s.sseServer = sseServer

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	log.Printf("[session-tui] MCP SSE server listening on %s", addr)

	// Background goroutine: flush notification channel periodically
	go s.notificationFlusher()

	return sseServer.Start(addr)
}

func (s *SessionMCPServer) Stop(ctx context.Context) error {
	if s.sseServer != nil {
		return s.sseServer.Shutdown(ctx)
	}
	return nil
}

// NotifyClients signals that TUI state has changed
func (s *SessionMCPServer) NotifySelectedChanged(idx int) {
	s.mu.Lock()
	s.selectedIdx = idx
	s.mu.Unlock()

	select {
	case s.notifyCh <- struct{}{}:
	default:
	}
}

func (s *SessionMCPServer) NotifyListChanged() {
	select {
	case s.notifyCh <- struct{}{}:
	default:
	}
}

func (s *SessionMCPServer) notificationFlusher() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		select {
		case <-s.notifyCh:
			s.mcpServer.SendNotificationToAllClients(
				mcp.MethodNotificationResourcesListChanged,
				nil,
			)
		default:
		}
	}
}

func (s *SessionMCPServer) registerResources() {
	// List resource
	listRes := mcp.NewResource(
		"session-tui://list",
		"Session List",
		mcp.WithResourceDescription("Full list of all sessions with metadata"),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(listRes, s.handleListResource)

	// Selected resource
	selRes := mcp.NewResource(
		"session-tui://selected",
		"Selected Session",
		mcp.WithResourceDescription("Currently highlighted session in the TUI"),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(selRes, s.handleSelectedResource)

	// Stats resource
	statsRes := mcp.NewResource(
		"session-tui://stats",
		"Session Stats",
		mcp.WithResourceDescription("Aggregate session statistics"),
		mcp.WithMIMEType("application/json"),
	)
	s.mcpServer.AddResource(statsRes, s.handleStatsResource)
}

func (s *SessionMCPServer) handleListResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	sessions, err := s.db.ListSessions(0, 0)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	if sessions == nil {
		sessions = []db.Session{}
	}

	data, err := json.Marshal(sessions)
	if err != nil {
		return nil, fmt.Errorf("marshal sessions: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "session-tui://list",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *SessionMCPServer) handleSelectedResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	s.mu.RLock()
	idx := s.selectedIdx
	s.mu.RUnlock()

	sel := s.model.SelectedSession()
	if sel == nil {
		return []mcp.ResourceContents{
			mcp.TextResourceContents{
				URI:      "session-tui://selected",
				MIMEType: "application/json",
				Text:     `{"selected": null, "cursor": 0}`,
			},
		}, nil
	}

	resp := map[string]any{
		"selected": sel,
		"cursor":   idx,
	}

	data, _ := json.Marshal(resp)
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "session-tui://selected",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *SessionMCPServer) handleStatsResource(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	stats, err := s.db.GetStats()
	if err != nil {
		return nil, fmt.Errorf("get stats: %w", err)
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return nil, fmt.Errorf("marshal stats: %w", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "session-tui://stats",
			MIMEType: "application/json",
			Text:     string(data),
		},
	}, nil
}

func (s *SessionMCPServer) registerTools() {
	// Delete session tool
	deleteTool := mcp.NewTool("delete_session",
		mcp.WithDescription("Delete a session by ID"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session ID to delete (e.g. ses_100cb0375ffeTbt6PthH9gHBzo)"),
		),
	)
	s.mcpServer.AddTool(deleteTool, s.handleDeleteTool)

	// Rename session tool
	renameTool := mcp.NewTool("rename_session",
		mcp.WithDescription("Rename a session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session ID to rename"),
		),
		mcp.WithString("new_title",
			mcp.Required(),
			mcp.Description("New title for the session"),
		),
	)
	s.mcpServer.AddTool(renameTool, s.handleRenameTool)

	// Focus session tool
	focusTool := mcp.NewTool("focus_session",
		mcp.WithDescription("Move TUI cursor to a specific session"),
		mcp.WithString("session_id",
			mcp.Required(),
			mcp.Description("Session ID to focus on"),
		),
	)
	s.mcpServer.AddTool(focusTool, s.handleFocusTool)

	// Refresh tool
	refreshTool := mcp.NewTool("refresh",
		mcp.WithDescription("Reload sessions from database"),
	)
	s.mcpServer.AddTool(refreshTool, s.handleRefreshTool)
}

func (s *SessionMCPServer) handleDeleteTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")

	session, err := s.db.GetSession(sessionID)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Session not found: %s", sessionID)), nil
	}

	if err := s.db.DeleteSession(sessionID); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Delete failed: %v", err)), nil
	}

	s.NotifyListChanged()

	text := fmt.Sprintf("Deleted: %s (%s, %d msgs)", session.Title, sessionID, session.MsgCount)
	return mcp.NewToolResultText(text), nil
}

func (s *SessionMCPServer) handleRenameTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")
	newTitle := req.GetString("new_title", "")

	if newTitle == "" {
		return mcp.NewToolResultError("new_title is required"), nil
	}

	if err := s.db.RenameSession(sessionID, newTitle); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Rename failed: %v", err)), nil
	}

	s.NotifyListChanged()

	return mcp.NewToolResultText(fmt.Sprintf("Renamed %s → %s", sessionID, newTitle)), nil
}

func (s *SessionMCPServer) handleFocusTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessionID := req.GetString("session_id", "")

	sessions, err := s.db.ListSessions(0, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("List failed: %v", err)), nil
	}

	for i, sess := range sessions {
		if sess.ID == sessionID {
			s.mu.Lock()
			s.selectedIdx = i
			s.mu.Unlock()
			s.NotifySelectedChanged(i)
			return mcp.NewToolResultText(fmt.Sprintf("Focused on session %d: %s", i, sess.Title)), nil
		}
	}
	return mcp.NewToolResultError(fmt.Sprintf("Session not found: %s", sessionID)), nil
}

func (s *SessionMCPServer) handleRefreshTool(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	sessions, err := s.db.ListSessions(0, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Refresh failed: %v", err)), nil
	}

	s.mu.Lock()
	s.selectedIdx = 0
	s.mu.Unlock()
	s.NotifyListChanged()

	return mcp.NewToolResultText(fmt.Sprintf("Refreshed: %d sessions loaded", len(sessions))), nil
}
