# myweb Architecture Design

## Overview

**myweb** is a browser-based MySQL client that provides the same core experience as [mysh](https://github.com/eyjian/mysh) — SQL editing with syntax highlighting, auto-completion, and result display — through a web interface instead of a terminal.

Key principles:
- **Single binary** — Go backend with embedded SPA frontend, same philosophy as mysh
- **Local-first** — listens on `127.0.0.1` by default, no authentication required
- **Code reuse** — shares `connection`, `metadata`, `executor`, `output`, `config` packages from mysh

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Browser (SPA)                        │
│  ┌──────────┐ ┌──────────┐ ┌────────┐ ┌─────────────┐ │
│  │ CodeMirror│ │ Result   │ │Sidebar │ │ History     │ │
│  │ SQL Editor│ │ Table    │ │ DB/Browse│ │ Panel       │ │
│  └─────┬─────┘ └────┬─────┘ └───┬────┘ └──────┬──────┘ │
│        └────────┬────┘───────────┘─────────────┘        │
│            HTTP REST + WebSocket                        │
├─────────────────────────────────────────────────────────┤
│                  Go HTTP Server                         │
│  ┌──────────┐ ┌───────────┐ ┌───────────────────────┐  │
│  │ REST API │ │ WebSocket │ │ Static (embed.FS)     │  │
│  │ Handler  │ │ Handler   │ │ ui/dist/              │  │
│  └────┬─────┘ └─────┬─────┘ └───────────────────────┘  │
├───────┴──────────────┴──────────────────────────────────┤
│                Shared Packages (from mysh)              │
│  ┌──────────┐ ┌──────────┐ ┌────────┐ ┌────────────┐  │
│  │connection│ │ metadata │ │executor│ │  output     │  │
│  │  pool    │ │  cache   │ │        │ │ formatter   │  │
│  └──────────┘ └──────────┘ └────────┘ └────────────┘  │
│  ┌──────────┐                                           │
│  │  config  │                                           │
│  └──────────┘                                           │
├─────────────────────────────────────────────────────────┤
│                    MySQL Server                         │
└─────────────────────────────────────────────────────────┘
```

## Project Structure

```
myweb/
├── cmd/
│   └── myweb/            # Entry point
│       └── main.go
├── server/
│   ├── server.go         # HTTP server setup, embed.FS, routes
│   ├── api.go            # REST API handlers
│   ├── ws.go             # WebSocket handler
│   └── middleware.go     # CORS, logging, recovery
├── ui/
│   ├── src/              # React frontend source
│   │   ├── App.tsx
│   │   ├── components/
│   │   │   ├── SqlEditor.tsx      # CodeMirror 6 wrapper
│   │   │   ├── ResultTable.tsx    # Virtual-scroll result grid
│   │   │   ├── Sidebar.tsx        # DB/table/column browser
│   │   │   ├── ConnectDialog.tsx  # Connection modal
│   │   │   ├── HistoryPanel.tsx   # Query history
│   │   │   └── StatusBar.tsx      # Connection info + format toggle
│   │   ├── hooks/
│   │   │   ├── useWebSocket.ts    # WebSocket connection manager
│   │   │   └── useApi.ts          # REST API helpers
│   │   └── index.tsx
│   ├── dist/             # Build output (go:embed target)
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── docs/
│   └── design/
│       └── ARCHITECTURE.md  # This file
├── Makefile
├── install.sh
├── go.mod
└── go.sum
```

## Go Backend

### Entry Point

`cmd/myweb/main.go`:
- Parse CLI flags: `-addr` (default `127.0.0.1:8080`), `-open` (auto-open browser), `-dsn` (initial connection)
- Load config from `~/.myweb.yaml` (reuse mysh `config` package)
- Create connection pool, metadata cache, executor (from mysh packages)
- Start HTTP server with embedded SPA
- Optionally open browser

### REST API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/status` | Connection status (host, db, user, server version, uptime) |
| `POST` | `/api/connect` | Connect to database `{"dsn":"user:pass@tcp(host:port)/db"}` |
| `POST` | `/api/reconnect` | Reconnect to current server |
| `GET` | `/api/databases` | List all databases |
| `GET` | `/api/tables?db=xxx` | List tables in a database |
| `GET` | `/api/columns?table=xxx&db=xxx` | List columns for a table |
| `POST` | `/api/query` | Execute a short query synchronously `{"sql":"SELECT ..."}` |
| `POST` | `/api/export` | Export last result `{"format":"csv","path":"~/out.csv"}` |
| `GET` | `/api/history` | Get query history (paginated) |
| `GET` | `/api/aliases` | Get configured aliases |

### WebSocket Protocol

Endpoint: `ws://localhost:8080/ws`

Used for long-running queries, streaming results, and watch mode.

**Client → Server messages:**

```json
{"type": "query", "sql": "SELECT * FROM users"}
{"type": "cancel"}
{"type": "watch", "interval": 5, "sql": "SELECT COUNT(*) FROM processes"}
{"type": "watch_stop"}
```

**Server → Client messages:**

```json
{"type": "result_start", "columns": ["id","name","email"], "format": "table"}
{"type": "result_row", "row": [1, "Alice", "alice@example.com"]}
{"type": "result_row", "row": [2, "Bob", "bob@example.com"]}
{"type": "result_end", "rows": 2, "time": "0.03s", "affected": 0}
{"type": "error", "message": "Table 'foo' doesn't exist"}
{"type": "watch_tick", "seq": 3, "columns": [...], "rows": [...], "time": "0.01s"}
{"type": "connected", "host": "localhost", "db": "mydb", "user": "root"}
{"type": "disconnected"}
```

Streaming `result_row` messages allow large result sets to render incrementally without waiting for the full query to complete.

### Static File Serving

Frontend build output is embedded using `go:embed`:

```go
//go:embed ui/dist/*
var staticFiles embed.FS
```

Served at `/` with SPA fallback (unknown paths return `index.html` for client-side routing).

## Frontend (React SPA)

### Tech Stack

| Library | Purpose |
|---------|---------|
| React 19 | UI framework |
| TypeScript | Type safety |
| Vite | Build tool (fast HMR in dev) |
| CodeMirror 6 | SQL editor with `@codemirror/lang-sql` |
| @tanstack/react-table | Virtual-scroll result table |
| Tailwind CSS | Styling |

### Page Layout

```
┌──────────────────────────────────────────────────────┐
│  🔵 myweb    mydb@localhost:3306       [Connect]     │  ← Top bar
├──────────┬───────────────────────────────────────────┤
│          │  ┌─────────────────────────────────────┐  │
│ Tables   │  │  SQL Editor (CodeMirror 6)         │  │
│──────────│  │  SELECT * FROM users WHERE ...       │  │
│ ▸ users  │  │                                     │  │
│ ▸ orders │  └─────────────────────────────────────┘  │
│ ▸ products│  [▶ Run]  [⏹ Cancel]  [📥 Export]        │  ← Action bar
│          │  ┌─────────────────────────────────────┐  │
│ Columns  │  │  Results Table                      │  │
│──────────│  │  id | name  | email                 │  │
│ id  INT  │  │  1  | Alice | alice@...             │  │
│ name VARC│  │  2  | Bob   | bob@...               │  │
│ email VAR│  └─────────────────────────────────────┘  │
│          │  2 rows in 0.03s  [Table|JSON|Markdown]   │  ← Status bar
└──────────┴───────────────────────────────────────────┘
```

### Components

#### SqlEditor (CodeMirror 6)
- SQL syntax highlighting via `@codemirror/lang-sql`
- Auto-completion: on `Ctrl+Space`, queries `/api/tables`, `/api/columns` for suggestions
- Multi-line editing, bracket matching, auto-close brackets
- Execute on `Ctrl+Enter` or `Cmd+Enter`
- `\G`, `\j`, `\m` suffix support (same as mysh)

#### ResultTable
- Virtual scrolling for large result sets (only renders visible rows)
- Column header click to sort (client-side)
- Column width auto-fit with manual resize
- NULL values displayed as dim italic `NULL` (consistent with mysh)
- Format toggle: Table / JSON / Markdown tabs

#### Sidebar
- Accordion sections: Databases → Tables → Columns
- Single click on table/column name → insert into editor at cursor
- Double click on table → auto-fill `SELECT * FROM table LIMIT 100`
- Refresh button to reload metadata

#### ConnectDialog
- Modal with fields: Host, Port, User, Password, Database
- Or paste a full DSN string
- Save connection as session (stored in config)

#### HistoryPanel
- Slide-out panel showing query history
- Search/filter by keyword
- Click to refill editor

#### StatusBar
- Connection indicator (green dot = connected, red = disconnected)
- Current database and host
- Last query row count and execution time
- Output format toggle buttons

### WebSocket Connection Management

```
useWebSocket hook:
  - Auto-connect on mount
  - Auto-reconnect with exponential backoff
  - Message queue for offline period
  - Parse streaming result_row → incremental table update
```

## Security

- **Default bind**: `127.0.0.1:8080` — localhost only
- **No authentication** by default — local tool, same as mysh
- **Optional token**: `--token` flag sets a Bearer token for API/WebSocket auth
- **No CORS headers** for external origins (localhost only)
- SQL injection: not applicable — user writes SQL directly, same as any DB client

## Shared Code from mysh

| Package | Usage in myweb |
|---------|---------------|
| `connection` | MySQL connection pool, reconnect, DB/table/column queries |
| `metadata` | Schema cache (lazy refresh, dirty marking) |
| `executor` | SQL execution, auto-reconnect retry, cancel support |
| `output` | Result formatting (table/vertical/JSON/Markdown) |
| `config` | Config file loading/saving, aliases, sessions |
| `history` | Query history persistence |

Packages **not** shared:
- `tui` — terminal-specific Bubble Tea model
- `editor` — terminal line editor
- `highlight` — ANSI terminal highlighting (frontend uses CodeMirror)
- `completer` — terminal completion menu (frontend uses CodeMirror completion)

## Development Workflow

### Local Development

```bash
# Terminal 1: Start frontend dev server with HMR
cd ui && npm run dev

# Terminal 2: Start Go backend with proxy to frontend
go run ./cmd/myweb --dev
```

In `--dev` mode, the Go server proxies `/` requests to Vite dev server (default `http://localhost:5173`) instead of using embedded files.

### Production Build

```bash
# Build frontend
cd ui && npm run build   # → ui/dist/

# Build Go binary (embeds ui/dist/)
go build -o myweb ./cmd/myweb

# Or via Makefile
make build
```

### Cross-compile

```bash
make cross-compile
# Produces: myweb-linux-amd64, myweb-darwin-arm64, etc.
```

## CLI Flags

```
Usage: myweb [options] [dsn]

Options:
  -addr string    Listen address (default "127.0.0.1:8080")
  -open           Open browser automatically (default true)
  -dev            Development mode (proxy to Vite)
  -token string   Bearer token for API authentication
  -config string  Config file path (default "~/.myweb.yaml")
  -version        Show version

Arguments:
  dsn             MySQL DSN to connect on startup (optional)
                  user:pass@tcp(host:port)/dbname
```

## Configuration

Config file: `~/.myweb.yaml` (compatible with mysh's `~/.mysh.yaml` format)

```yaml
# Connection defaults
host: "127.0.0.1"
port: 3306
user: "root"
password: ""

# Server settings
addr: "127.0.0.1:8080"
open_browser: true

# Display
theme: "dark"              # dark / light
page_size: 100             # Default result limit
auto_vertical: false       # Auto-switch to vertical for wide results

# SQL aliases (shared with mysh)
aliases:
  top10: "SELECT * FROM users ORDER BY score DESC LIMIT 10"

# Saved sessions (shared with mysh)
sessions:
  prod:
    host: "db.prod.example.com"
    port: 3306
    user: "admin"
    database: "myapp"
```

## Phased Implementation Plan

### Phase 1: Skeleton + Basic Query
- Go HTTP server with embed.FS
- WebSocket handler (query/cancel)
- React scaffold + CodeMirror SQL editor
- Result table (basic, no virtual scroll)
- Connect dialog

### Phase 2: Schema Browser + Completion
- Sidebar with DB/table/column tree
- CodeMirror auto-completion from metadata API
- Double-click table → SELECT query
- History panel

### Phase 3: Advanced Features
- Virtual scroll result table
- Export to CSV/JSON/Markdown
- Watch mode (periodic re-execute)
- Format toggle (Table/JSON/Markdown)
- Session management

### Phase 4: Polish
- Dark/light theme
- Keyboard shortcuts panel
- install.sh + cross-compile
- Config file sharing with mysh
