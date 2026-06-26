# session-tui

Interactive TUI + MCP server for browsing and managing OpenCode sessions.

```
┌─────────────────────────────────────────────────────────────┐
│ 󰃤 SESSION BROWSER                                           │
│  47 sessions                                               │
├─────────────────────────────────────────────────────────────┤
│ ▶ ▓ UFI003 Modem Handoff & Setup (408 msgs)    1.3M │
│   Scrapegraph-AI Rollout with 9router (195 msgs)  607K │
│   Shogun 2 Crashes & Mod Manager (113 msgs)      477K │
│   Torreting Stack Follow-up (245 msgs)          1.2M │
│   …                                                      │
╰─────────────────────────────────────────────────────────────╯
  j/k=nav  /=filter  d=delete  r=rename  R=refresh  q=quit
```

## Architecture

Single binary. Two faces. Zero agent lock-in.

```
┌─────────────────┐     MCP (SSE)      ┌─────────────────┐
│ OpenCode / any  │◄──────────────────►│ session-tui     │
│ MCP client      │  resources/tools   │                 │
│                  │                    │ ┌─────────────┐ │
│ Subscribes to:   │                    │ │ TUI          │ │
│  session-tui://list  │                    │ (bubbletea)  │ │
│  session-tui://selected               │ └─────────────┘ │
│  session-tui://stats                  │         │        │
│                  │                    │    ┌────┴────┐   │
│ Calls tools:     │                    │    │ SQLite  │   │
│  delete_session  │                    │    │ (direct)│   │
│  rename_session  │                    │    └─────────┘   │
│  focus_session   │                    └─────────────────┘
│  refresh         │
└─────────────────┘
```

## Usage

```bash
# Build
go build -o session-tui.exe .

# Run (auto-detects opencode-dev.db)
session-tui.exe

# Custom port
session-tui.exe --port 8300

# Custom DB path
session-tui.exe --db "C:\Users\me\.local\share\opencode\opencode-dev.db"

# Optional: enable mem0 integration
session-tui.exe --mem0-url http://127.0.0.1:8301/sse
```

## Register in OpenCode

Add to your `opencode.json`:

```json
{
  "mcp": {
    "session-tui": {
      "type": "remote",
      "url": "http://127.0.0.1:8300/sse",
      "enabled": true
    }
  }
}
```

## MCP Protocol

### Resources (with subscription support)

| Resource | Description |
|----------|-------------|
| `session-tui://list` | Full session list as JSON array |
| `session-tui://selected` | Currently highlighted session in TUI |
| `session-tui://stats` | Aggregate session statistics |

### Tools

| Tool | Description |
|------|-------------|
| `delete_session` | Delete a session by ID |
| `rename_session` | Rename a session |
| `focus_session` | Move TUI cursor to a specific session |
| `refresh` | Reload sessions from database |

## Keybindings

| Key | Action |
|-----|--------|
| `j` / `down` | Move cursor down |
| `k` / `up` | Move cursor up |
| `g` | Go to top |
| `G` | Go to bottom |
| `/` | Filter sessions (fuzzy search by title) |
| `d` | Delete selected session (with confirmation) |
| `r` | Rename selected session |
| `R` | Refresh from database |
| `q` / `Ctrl+C` | Quit |

## Optional: Mem0

When `--mem0-url` is provided, the server will embed mem0 memories into the `selected` resource for the currently highlighted session. This lets any agent see what mem0 knows about the session without calling mem0 separately.
