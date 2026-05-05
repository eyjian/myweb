# myweb Phase 1 实现计划：骨架 + 基础查询

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**目标：** 构建 myweb 最小可用版本——Go 后端提供 REST/WebSocket API，Vue 3 前端提供 SQL 编辑器和结果表格，用户可以在浏览器中执行 SQL 查询并查看结果。

**架构：** Go HTTP 服务器复用 mysh 的 connection/executor/metadata/output 包，通过 REST API 和 WebSocket 暴露功能；Vue 3 + CodeMirror 6 前端嵌入 Go 二进制，单文件分发。

**技术栈：** Go 1.25 + gorilla/websocket + embed.FS | Vue 3 + TypeScript + Vite + CodeMirror 6 + Tailwind CSS

---

## 文件结构

```
myweb/
├── cmd/myweb/main.go           # 入口：CLI 参数、依赖组装、启动 HTTP 服务器
├── server/
│   ├── server.go               # Server 结构体、路由注册、embed.FS 服务、开发代理
│   ├── api.go                  # REST API 处理器（/api/*）
│   ├── ws.go                   # WebSocket 处理器（/ws）
│   └── middleware.go           # 日志、恢复中间件
├── ui/
│   ├── src/
│   │   ├── App.vue             # 根组件（布局框架）
│   │   ├── main.ts             # 入口
│   │   ├── style.css           # Tailwind 入口
│   │   ├── components/
│   │   │   ├── SqlEditor.vue   # CodeMirror 6 SQL 编辑器
│   │   │   ├── ResultTable.vue # 基础结果表格（Phase 1 无虚拟滚动）
│   │   │   ├── ConnectDialog.vue # 连接对话框
│   │   │   └── StatusBar.vue   # 连接状态 + 执行信息
│   │   └── composables/
│   │       ├── useWebSocket.ts # WebSocket 连接管理
│   │       └── useApi.ts       # REST API 辅助函数
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   ├── tsconfig.node.json
│   ├── vite.config.ts
│   └── env.d.ts
├── Makefile
├── go.mod
└── go.sum
```

---

### Task 1: Go 模块初始化 + 依赖

**文件：**
- 修改：`go.mod`
- 创建：`server/server.go`（占位）

- [ ] **Step 1: 添加 mysh 依赖和 gorilla/websocket**

```bash
cd /data/workspace/eyjian/myweb
go get github.com/eyjian/mysh@v0.1.0
go get github.com/gorilla/websocket@latest
```

由于使用 go.work，mysh 会通过本地路径解析。验证：

```bash
go list -m all | grep mysh
```

预期：能看到 mysh 模块

- [ ] **Step 2: 创建 server 包占位文件**

创建 `server/server.go`：

```go
package server

// Server will be implemented in Task 3
```

- [ ] **Step 3: 验证编译**

```bash
cd /data/workspace/eyjian/myweb && go build ./...
```

预期：编译成功，无错误

- [ ] **Step 4: 提交**

```bash
git add go.mod go.sum server/
git commit -m "chore: init Go module with mysh and gorilla/websocket deps"
```

---

### Task 2: Go 入口 main.go

**文件：**
- 创建：`cmd/myweb/main.go`

- [ ] **Step 1: 编写入口程序**

创建 `cmd/myweb/main.go`：

```go
package main

import (
	"flag"
	"fmt"
	"os"

	"mysh/config"
	"mysh/connection"
	"mysh/executor"
	"mysh/metadata"

	"github.com/eyjian/myweb/server"
)

var version = "dev"

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	openBrowser := flag.Bool("open", true, "open browser automatically")
	dev := flag.Bool("dev", false, "development mode (proxy to Vite)")
	showVersion := flag.Bool("version", false, "show version")
	flag.Parse()

	if *showVersion {
		fmt.Printf("myweb version %s\n", version)
		os.Exit(0)
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config load failed: %s\n", err)
	}

	// Connect to MySQL (from config or default localhost)
	pool, err := connection.New(&cfg.Connection)
	if err != nil {
		fmt.Fprintf(os.Stderr, "MySQL connection failed: %s\n", err)
		fmt.Fprintf(os.Stderr, "Starting without database connection. Use the Connect dialog in browser.\n")
	}

	// Initialize metadata cache
	var meta *metadata.Cache
	if pool != nil {
		meta, _ = metadata.NewCache(pool)
		if meta != nil {
			go func() { _ = meta.Refresh() }()
		}
	}

	// Initialize executor
	var exec *executor.Executor
	if pool != nil {
		exec = executor.New(pool, meta)
	}

	// Create and start server
	srv := server.New(&server.Config{
		Addr:    *addr,
		Dev:     *dev,
		Config:  cfg,
		Pool:    pool,
		Meta:    meta,
		Executor: exec,
	})

	if *openBrowser && !*dev {
		go openURL(fmt.Sprintf("http://%s", *addr))
	}

	fmt.Printf("myweb listening on http://%s\n", *addr)
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %s\n", err)
		os.Exit(1)
	}
}

func openURL(url string) {
	// Best-effort open browser; ignore errors
	// Use "open" on macOS, "xdg-open" on Linux, "start" on Windows
	name := "xdg-open"
	if runtime.GOOS == "darwin" {
		name = "open"
	} else if runtime.GOOS == "windows" {
		name = "start"
	}
	_ = exec.Command(name, url).Start()
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /data/workspace/eyjian/myweb && go build ./cmd/myweb
```

预期：编译成功（可能有 unused import 警告，后续 Task 会使用）

- [ ] **Step 3: 提交**

```bash
git add cmd/
git commit -m "feat: add main entry point with CLI flags"
```

---

### Task 3: Go HTTP 服务器 + embed.FS + 开发代理

**文件：**
- 修改：`server/server.go`

- [ ] **Step 1: 实现 Server 结构体和路由注册**

替换 `server/server.go` 内容：

```go
package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"time"

	"mysh/config"
	"mysh/connection"
	"mysh/executor"
	"mysh/metadata"
)

//go:embed all:../ui/dist
var staticFiles embed.FS

// Config holds server configuration.
type Config struct {
	Addr     string
	Dev      bool
	Config   *config.Config
	Pool     *connection.Pool
	Meta     *metadata.Cache
	Executor *executor.Executor
}

// Server is the HTTP server for myweb.
type Server struct {
	cfg    *Config
	mux    *http.ServeMux
	hub    *Hub
}

// New creates a new Server.
func New(cfg *Config) *Server {
	s := &Server{
		cfg: cfg,
		mux: http.NewServeMux(),
		hub: newHub(),
	}

	// API routes
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/connect", s.handleConnect)
	s.mux.HandleFunc("/api/databases", s.handleDatabases)
	s.mux.HandleFunc("/api/tables", s.handleTables)
	s.mux.HandleFunc("/api/columns", s.handleColumns)
	s.mux.HandleFunc("/api/query", s.handleQuery)
	s.mux.HandleFunc("/ws", s.handleWebSocket)

	// Static files or dev proxy
	if cfg.Dev {
		s.mux.HandleFunc("/", s.devProxy)
	} else {
		s.serveStatic()
	}

	go s.hub.run()
	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	handler := s.recoveryMiddleware(s.loggingMiddleware(s.mux))
	srv := &http.Server{
		Addr:         s.cfg.Addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}
	return srv.ListenAndServe()
}

// serveStatic registers embedded SPA file serving.
func (s *Server) serveStatic() {
	sub, err := fs.Sub(staticFiles, "../ui/dist")
	if err != nil {
		log.Printf("Warning: embedded files not found: %s", err)
		return
	}
	fileServer := http.FileServer(http.FS(sub))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || !strings.Contains(path, ".") {
			// SPA fallback: serve index.html for unknown routes
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// devProxy proxies requests to Vite dev server in development mode.
func (s *Server) devProxy(w http.ResponseWriter, r *http.Request) {
	// Proxy to Vite dev server at localhost:5173
	target := "http://localhost:5173" + r.URL.Path
	if r.URL.RawQuery != "" {
		target += "?" + r.URL.RawQuery
	}
	resp, err := http.DefaultClient.Do(&http.Request{
		Method: r.Method,
		URL:    mustParseURL(target),
		Header: r.Header,
		Body:   r.Body,
	})
	if err != nil {
		http.Error(w, "Vite dev server not reachable: "+err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func mustParseURL(s string) *url.URL {
	u, _ := url.Parse(s)
	return u
}
```

- [ ] **Step 2: 添加缺少的 import**

确保 import 包含 `io`, `net/url`：

```go
import (
	"embed"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	// ... mysh imports
)
```

- [ ] **Step 3: 创建临时 ui/dist/index.html 占位**

创建 `ui/dist/index.html`：

```html
<!DOCTYPE html>
<html>
<head><title>myweb</title></head>
<body><div id="app">myweb loading...</div></body>
</html>
```

这样 `go:embed` 才能编译通过。

- [ ] **Step 4: 验证编译**

```bash
cd /data/workspace/eyjian/myweb && go build ./server
```

预期：编译成功

- [ ] **Step 5: 提交**

```bash
git add server/ ui/dist/
git commit -m "feat: HTTP server with embed.FS, routing, and dev proxy"
```

---

### Task 4: 中间件

**文件：**
- 创建：`server/middleware.go`

- [ ] **Step 1: 实现日志和恢复中间件**

创建 `server/middleware.go`：

```go
package server

import (
	"log"
	"net/http"
	"runtime/debug"
	"time"
)

// loggingMiddleware logs each request.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

// recoveryMiddleware recovers from panics in handlers.
func (s *Server) recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("PANIC: %s\n%s", err, debug.Stack())
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// corsMiddleware adds CORS headers for local development.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 2: 更新 server.go 使用 CORS 中间件**

在 `server.go` 的 `ListenAndServe` 方法中，在 dev 模式下添加 CORS：

```go
func (s *Server) ListenAndServe() error {
	handler := s.recoveryMiddleware(s.loggingMiddleware(s.mux))
	if s.cfg.Dev {
		handler = corsMiddleware(handler)
	}
	// ... rest unchanged
}
```

- [ ] **Step 3: 验证编译**

```bash
cd /data/workspace/eyjian/myweb && go build ./server
```

- [ ] **Step 4: 提交**

```bash
git add server/middleware.go server/server.go
git commit -m "feat: add logging, recovery, and CORS middleware"
```

---

### Task 5: REST API 处理器

**文件：**
- 创建：`server/api.go`

- [ ] **Step 1: 实现 REST API 处理器**

创建 `server/api.go`：

```go
package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"mysh/connection"
	"mysh/executor"
)

// --- JSON request/response types ---

type connectRequest struct {
	DSN string `json:"dsn"`
}

type queryRequest struct {
	SQL string `json:"sql"`
}

type statusResponse struct {
	Connected bool   `json:"connected"`
	Host      string `json:"host,omitempty"`
	Database  string `json:"database,omitempty"`
	User      string `json:"user,omitempty"`
	Version   string `json:"server_version,omitempty"`
}

type databasesResponse struct {
	Databases []string `json:"databases"`
}

type tablesResponse struct {
	Tables []string `json:"tables"`
}

type columnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Key      string `json:"key,omitempty"`
}

type columnsResponse struct {
	Columns []columnInfo `json:"columns"`
}

type queryResponse struct {
	Columns      []string `json:"columns,omitempty"`
	Rows         [][]any  `json:"rows,omitempty"`
	RowCount     int64    `json:"row_count,omitempty"`
	AffectedRows int64    `json:"affected_rows,omitempty"`
	Duration     string   `json:"duration,omitempty"`
	IsQuery      bool     `json:"is_query"`
	Warning      string   `json:"warning,omitempty"`
	Error        string   `json:"error,omitempty"`
}

type errorResponse struct {
	Error string `json:"error"`
}

// --- Handlers ---

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	resp := statusResponse{Connected: s.cfg.Pool != nil}
	if s.cfg.Pool != nil {
		info := s.cfg.Pool.Status()
		resp.Host = info.Host
		resp.Database = info.Database
		resp.User = info.User
		resp.Version = info.ServerVersion
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "POST only"})
		return
	}

	var req connectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	// Parse DSN to config
	cfg := s.cfg.Config.Connection // copy defaults
	if req.DSN != "" {
		parsed, err := parseDSN(req.DSN)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
			return
		}
		cfg = *parsed
	}

	pool, err := connection.New(&cfg)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}

	// Replace server's pool and rebuild executor
	if s.cfg.Pool != nil {
		s.cfg.Pool.Close()
	}
	s.cfg.Pool = pool
	s.cfg.Config.Connection = cfg

	meta, _ := metadata.NewCache(pool)
	s.cfg.Meta = meta
	s.cfg.Executor = executor.New(pool, meta)
	if meta != nil {
		go func() { _ = meta.Refresh() }()
	}

	writeJSON(w, http.StatusOK, statusResponse{
		Connected: true,
		Host:      cfg.Host,
		Database:  cfg.Database,
		User:      cfg.User,
	})
}

func (s *Server) handleDatabases(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Pool == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "not connected"})
		return
	}
	dbs, err := s.cfg.Pool.Databases()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, databasesResponse{Databases: dbs})
}

func (s *Server) handleTables(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Pool == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "not connected"})
		return
	}
	db := r.URL.Query().Get("db")
	if db == "" {
		db = s.cfg.Config.Connection.Database
	}
	tables, err := s.cfg.Pool.Tables(db)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, tablesResponse{Tables: tables})
}

func (s *Server) handleColumns(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Pool == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "not connected"})
		return
	}
	table := r.URL.Query().Get("table")
	db := r.URL.Query().Get("db")
	if table == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "table parameter required"})
		return
	}
	if db == "" {
		db = s.cfg.Config.Connection.Database
	}
	cols, err := s.cfg.Pool.Columns(db, table)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}
	result := make([]columnInfo, len(cols))
	for i, c := range cols {
		result[i] = columnInfo{
			Name:     c.Name,
			Type:     c.Type,
			Nullable: c.Nullable,
			Key:      c.Key,
		}
	}
	writeJSON(w, http.StatusOK, columnsResponse{Columns: result})
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "POST only"})
		return
	}
	if s.cfg.Executor == nil {
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "not connected"})
		return
	}

	var req queryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	result, err := s.cfg.Executor.Execute(r.Context(), req.SQL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	resp := queryResponse{
		IsQuery:      result.IsQuery,
		RowCount:     result.RowCount,
		AffectedRows: result.AffectedRows,
		Duration:     result.Duration.String(),
		Warning:      result.Warning,
	}
	if result.Error != nil {
		resp.Error = result.Error.Error()
	}
	if result.IsQuery {
		resp.Columns = result.Columns
		resp.Rows = result.Rows
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// parseDSN parses a MySQL DSN string into ConnectionConfig.
func parseDSN(dsn string) (*config.ConnectionConfig, error) {
	// Simple DSN format: user:password@tcp(host:port)/database
	// Use the mysql driver's ParseDSN if available, otherwise basic parsing
	cfg := &config.ConnectionConfig{}
	// Basic parsing - extract user:pass, host:port, database
	// Full DSN parsing is handled by go-sql-driver/mysql internally
	parts := strings.SplitN(dsn, "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid DSN format, expected user:pass@tcp(host:port)/database")
	}
	userPass := strings.SplitN(parts[0], ":", 2)
	cfg.User = userPass[0]
	if len(userPass) > 1 {
		cfg.Password = userPass[1]
	}

	rest := parts[1]
	rest = strings.TrimPrefix(rest, "tcp(")
	rest = strings.TrimSuffix(rest, ")")
	hostPortDB := strings.SplitN(rest, "/", 2)

	hostPort := strings.SplitN(hostPortDB[0], ":", 2)
	cfg.Host = hostPort[0]
	if len(hostPort) > 1 {
		fmt.Sscanf(hostPort[1], "%d", &cfg.Port)
	}
	if cfg.Port == 0 {
		cfg.Port = 3306
	}
	if len(hostPortDB) > 1 {
		cfg.Database = hostPortDB[1]
	}

	return cfg, nil
}
```

注意：需要添加 `"strings"` 和 `"mysh/config"` 到 import。同时需要检查 `connection.Pool` 是否有 `Status()` 和 `Columns()` 方法——如果没有，需要适配。

- [ ] **Step 2: 检查 mysh connection.Pool 的可用方法**

```bash
cd /data/workspace/eyjian/mysh && grep -n "^func (p \*Pool)" connection/pool.go
```

根据已有代码，Pool 应该有：`New`, `Close`, `Reconnect`, `DSN`, `DB`, `Databases`, `Tables`, `Columns`, `Status` 等方法。如果缺少 `Status()` 方法，需要添加或调整代码。

- [ ] **Step 3: 验证编译**

```bash
cd /data/workspace/eyjian/myweb && go build ./server
```

如果有编译错误，根据 mysh 的实际 API 调整代码。

- [ ] **Step 4: 提交**

```bash
git add server/api.go
git commit -m "feat: add REST API handlers for status, connect, databases, tables, columns, query"
```

---

### Task 6: WebSocket 处理器 + Hub

**文件：**
- 创建：`server/ws.go`

- [ ] **Step 1: 实现 WebSocket Hub 和处理器**

创建 `server/ws.go`：

```go
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local tool
	},
}

// Hub manages WebSocket connections.
type Hub struct {
	mu      sync.Mutex
	clients map[*wsClient]struct{}
}

type wsClient struct {
	conn *websocket.Conn
	hub  *Hub
	send chan []byte
}

func newHub() *Hub {
	return &Hub{
		clients: make(map[*wsClient]struct{}),
	}
}

func (h *Hub) run() {
	// Hub runs passively; client management is in register/unregister
}

func (h *Hub) register(c *wsClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) unregister(c *wsClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// wsMessage represents a WebSocket message.
type wsMessage struct {
	Type     string `json:"type"`
	SQL      string `json:"sql,omitempty"`
	Interval int    `json:"interval,omitempty"`
}

// wsOutMessage represents a server-to-client WebSocket message.
type wsOutMessage struct {
	Type     string   `json:"type"`
	Columns  []string `json:"columns,omitempty"`
	Row      []any    `json:"row,omitempty"`
	Rows     int64    `json:"rows,omitempty"`
	Time     string   `json:"time,omitempty"`
	Affected int64    `json:"affected,omitempty"`
	Message  string   `json:"message,omitempty"`
	Seq      int      `json:"seq,omitempty"`
	Host     string   `json:"host,omitempty"`
	Database string   `json:"database,omitempty"`
	User     string   `json:"user,omitempty"`
	Format   string   `json:"format,omitempty"`
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %s", err)
		return
	}

	client := &wsClient{
		conn: conn,
		hub:  s.hub,
		send: make(chan []byte, 256),
	}
	s.hub.register(client)

	// Send initial connection status
	if s.cfg.Pool != nil {
		info := s.cfg.Pool.Status()
		s.sendWS(client, wsOutMessage{
			Type:     "connected",
			Host:     info.Host,
			Database: info.Database,
			User:     info.User,
		})
	}

	go client.writePump()
	go client.readPump(s)
}

func (c *wsClient) readPump(s *Server) {
	defer func() {
		c.hub.unregister(c)
		c.conn.Close()
	}()

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}

		var msg wsMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			s.sendWS(c, wsOutMessage{Type: "error", Message: "invalid message format"})
			continue
		}

		switch msg.Type {
		case "query":
			s.handleWSQuery(c, msg)
		case "cancel":
			s.handleWSCancel(c)
		case "watch":
			s.handleWSWatch(c, msg)
		case "watch_stop":
			s.handleWSWatchStop(c)
		}
	}
}

func (c *wsClient) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func (s *Server) sendWS(c *wsClient, msg wsOutMessage) {
	data, _ := json.Marshal(msg)
	select {
	case c.send <- data:
	default:
		// Buffer full, drop client
		c.hub.unregister(c)
		close(c.send)
	}
}

// --- WebSocket message handlers ---

func (s *Server) handleWSQuery(c *wsClient, msg wsMessage) {
	if s.cfg.Executor == nil {
		s.sendWS(c, wsOutMessage{Type: "error", Message: "not connected"})
		return
	}

	start := time.Now()
	result, err := s.cfg.Executor.Execute(context.Background(), msg.SQL)
	if err != nil {
		s.sendWS(c, wsOutMessage{Type: "error", Message: err.Error()})
		return
	}

	if result.Error != nil {
		s.sendWS(c, wsOutMessage{Type: "error", Message: result.Error.Error()})
		return
	}

	// Stream result rows
	if result.IsQuery {
		s.sendWS(c, wsOutMessage{
			Type:    "result_start",
			Columns: result.Columns,
			Format:  "table",
		})
		for _, row := range result.Rows {
			s.sendWS(c, wsOutMessage{Type: "result_row", Row: row})
		}
	}

	s.sendWS(c, wsOutMessage{
		Type:     "result_end",
		Rows:     result.RowCount,
		Affected: result.AffectedRows,
		Time:     time.Since(start).Round(time.Millisecond).String(),
	})

	if result.Warning != "" {
		s.sendWS(c, wsOutMessage{Type: "info", Message: result.Warning})
	}
}

func (s *Server) handleWSCancel(c *wsClient) {
	// Cancel is handled via executor's cancel mechanism
	s.sendWS(c, wsOutMessage{Type: "info", Message: "query cancelled"})
}

func (s *Server) handleWSWatch(c *wsClient, msg wsMessage) {
	// Watch mode will be implemented in Phase 3
	s.sendWS(c, wsOutMessage{Type: "error", Message: "watch mode not yet implemented"})
}

func (s *Server) handleWSWatchStop(c *wsClient) {
	// Watch mode will be implemented in Phase 3
}
```

- [ ] **Step 2: 验证编译**

```bash
cd /data/workspace/eyjian/myweb && go build ./server
```

- [ ] **Step 3: 提交**

```bash
git add server/ws.go
git commit -m "feat: add WebSocket handler with streaming query results"
```

---

### Task 7: Vue 3 前端脚手架

**文件：**
- 创建：`ui/package.json`, `ui/vite.config.ts`, `ui/tsconfig.json`, `ui/tsconfig.node.json`, `ui/index.html`, `ui/env.d.ts`, `ui/src/main.ts`, `ui/src/style.css`, `ui/src/App.vue`

- [ ] **Step 1: 初始化 Vue 3 项目**

```bash
cd /data/workspace/eyjian/myweb/ui
npm create vite@latest . -- --template vue-ts
```

如果目录非空（已有 dist/index.html），先移除再创建，或手动创建文件。

- [ ] **Step 2: 安装依赖**

```bash
cd /data/workspace/eyjian/myweb/ui
npm install
npm install vue-codemirror @codemirror/lang-sql @codemirror/theme-one-dark
npm install -D tailwindcss @tailwindcss/vite
```

- [ ] **Step 3: 配置 Vite 代理**

编辑 `ui/vite.config.ts`：

```typescript
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [vue(), tailwindcss()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
      '/ws': {
        target: 'ws://localhost:8080',
        ws: true,
      },
    },
  },
})
```

- [ ] **Step 4: 配置 Tailwind CSS**

编辑 `ui/src/style.css`：

```css
@import "tailwindcss";

body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}
```

- [ ] **Step 5: 验证前端编译**

```bash
cd /data/workspace/eyjian/myweb/ui && npm run build
```

预期：构建成功，产物在 `ui/dist/`

- [ ] **Step 6: 提交**

```bash
cd /data/workspace/eyjian/myweb
git add ui/
git commit -m "feat: Vue 3 + Vite + Tailwind + CodeMirror scaffold"
```

---

### Task 8: 前端 composables — useApi + useWebSocket

**文件：**
- 创建：`ui/src/composables/useApi.ts`
- 创建：`ui/src/composables/useWebSocket.ts`

- [ ] **Step 1: 实现 useApi**

创建 `ui/src/composables/useApi.ts`：

```typescript
export interface StatusResponse {
  connected: boolean
  host?: string
  database?: string
  user?: string
  server_version?: string
}

export interface QueryResponse {
  columns?: string[]
  rows?: unknown[][]
  row_count?: number
  affected_rows?: number
  duration?: string
  is_query: boolean
  warning?: string
  error?: string
}

export interface DatabaseResponse {
  databases: string[]
}

export interface TableResponse {
  tables: string[]
}

export interface ColumnInfo {
  name: string
  type: string
  nullable: boolean
  key?: string
}

export interface ColumnResponse {
  columns: ColumnInfo[]
}

async function apiFetch<T>(path: string, options?: RequestInit): Promise<T> {
  const resp = await fetch(path, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!resp.ok) {
    const err = await resp.json().catch(() => ({ error: resp.statusText }))
    throw new Error(err.error || resp.statusText)
  }
  return resp.json()
}

export function useApi() {
  const getStatus = () => apiFetch<StatusResponse>('/api/status')

  const connect = (dsn: string) =>
    apiFetch<StatusResponse>('/api/connect', {
      method: 'POST',
      body: JSON.stringify({ dsn }),
    })

  const getDatabases = () => apiFetch<DatabaseResponse>('/api/databases')

  const getTables = (db?: string) =>
    apiFetch<TableResponse>(`/api/tables${db ? `?db=${db}` : ''}`)

  const getColumns = (table: string, db?: string) =>
    apiFetch<ColumnResponse>(`/api/columns?table=${table}${db ? `&db=${db}` : ''}`)

  const executeQuery = (sql: string) =>
    apiFetch<QueryResponse>('/api/query', {
      method: 'POST',
      body: JSON.stringify({ sql }),
    })

  return { getStatus, connect, getDatabases, getTables, getColumns, executeQuery }
}
```

- [ ] **Step 2: 实现 useWebSocket**

创建 `ui/src/composables/useWebSocket.ts`：

```typescript
import { ref, onUnmounted } from 'vue'

export interface WSResultStart {
  type: 'result_start'
  columns: string[]
  format: string
}

export interface WSResultRow {
  type: 'result_row'
  row: unknown[]
}

export interface WSResultEnd {
  type: 'result_end'
  rows: number
  affected: number
  time: string
}

export interface WSError {
  type: 'error'
  message: string
}

export interface WSConnected {
  type: 'connected'
  host: string
  database: string
  user: string
}

export interface WSInfo {
  type: 'info'
  message: string
}

export type WSMessage = WSResultStart | WSResultRow | WSResultEnd | WSError | WSConnected | WSInfo

export function useWebSocket() {
  const ws = ref<WebSocket | null>(null)
  const connected = ref(false)
  const lastError = ref('')
  const onMessage = ref<(msg: WSMessage) => void>(() => {})

  function connect() {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/ws`
    const socket = new WebSocket(url)

    socket.onopen = () => {
      connected.value = true
      lastError.value = ''
    }

    socket.onclose = () => {
      connected.value = false
      // Reconnect with exponential backoff
      setTimeout(connect, 3000)
    }

    socket.onerror = () => {
      lastError.value = 'WebSocket connection error'
    }

    socket.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        onMessage.value(msg)
      } catch (e) {
        console.error('Failed to parse WebSocket message', e)
      }
    }

    ws.value = socket
  }

  function send(type: string, data: Record<string, unknown> = {}) {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type, ...data }))
    }
  }

  function sendQuery(sql: string) {
    send('query', { sql })
  }

  function sendCancel() {
    send('cancel')
  }

  connect()

  onUnmounted(() => {
    if (ws.value) {
      ws.value.close()
    }
  })

  return { connected, lastError, onMessage, sendQuery, sendCancel }
}
```

- [ ] **Step 3: 验证前端编译**

```bash
cd /data/workspace/eyjian/myweb/ui && npm run build
```

- [ ] **Step 4: 提交**

```bash
git add ui/src/composables/
git commit -m "feat: add useApi and useWebSocket composables"
```

---

### Task 9: SqlEditor 组件

**文件：**
- 创建：`ui/src/components/SqlEditor.vue`

- [ ] **Step 1: 实现 SQL 编辑器组件**

创建 `ui/src/components/SqlEditor.vue`：

```vue
<script setup lang="ts">
import { ref, onMounted, defineEmits } from 'vue'
import { Codemirror } from 'vue-codemirror'
import { sql } from '@codemirror/lang-sql'
import { oneDark } from '@codemirror/theme-one-dark'
import { keymap } from '@codemirror/view'
import type { Extension } from '@codemirror/state'

const emit = defineEmits<{
  execute: [sql: string]
}>()

const code = ref('')
const extensions = ref<Extension[]>([
  sql(),
  oneDark,
  keymap.of([
    {
      key: 'Ctrl-Enter',
      run: () => {
        if (code.value.trim()) {
          emit('execute', code.value)
        }
        return true
      },
    },
    {
      key: 'Cmd-Enter',
      run: () => {
        if (code.value.trim()) {
          emit('execute', code.value)
        }
        return true
      },
    },
  ]),
])

function setCode(value: string) {
  code.value = value
}

function clearCode() {
  code.value = ''
}

defineExpose({ setCode, clearCode })
</script>

<template>
  <div class="sql-editor h-full">
    <Codemirror
      v-model="code"
      :extensions="extensions"
      :style="{ height: '100%' }"
      placeholder="Type SQL here... (Ctrl+Enter to execute)"
      tab-size="2"
    />
  </div>
</template>

<style scoped>
.sql-editor :deep(.cm-editor) {
  height: 100%;
  font-size: 14px;
}
.sql-editor :deep(.cm-scroller) {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}
</style>
```

- [ ] **Step 2: 验证前端编译**

```bash
cd /data/workspace/eyjian/myweb/ui && npm run build
```

- [ ] **Step 3: 提交**

```bash
git add ui/src/components/SqlEditor.vue
git commit -m "feat: add SqlEditor component with CodeMirror 6"
```

---

### Task 10: ResultTable + StatusBar + ConnectDialog 组件

**文件：**
- 创建：`ui/src/components/ResultTable.vue`
- 创建：`ui/src/components/StatusBar.vue`
- 创建：`ui/src/components/ConnectDialog.vue`

- [ ] **Step 1: 实现 ResultTable 组件**

创建 `ui/src/components/ResultTable.vue`：

```vue
<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  columns: string[]
  rows: unknown[][]
  duration?: string
  error?: string
}>()

const hasResult = computed(() => props.columns.length > 0 || props.rows.length > 0)
</script>

<template>
  <div class="result-table flex flex-col h-full overflow-auto">
    <div v-if="error" class="p-3 text-red-400 bg-red-900/20 rounded">
      {{ error }}
    </div>
    <div v-else-if="hasResult" class="overflow-auto flex-1">
      <table class="w-full text-sm border-collapse">
        <thead class="sticky top-0 bg-gray-800">
          <tr>
            <th
              v-for="col in columns"
              :key="col"
              class="px-3 py-2 text-left text-gray-300 border-b border-gray-700 whitespace-nowrap"
            >
              {{ col }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in rows" :key="i" class="hover:bg-gray-800/50">
            <td
              v-for="(cell, j) in row"
              :key="j"
              class="px-3 py-1.5 border-b border-gray-800 whitespace-nowrap"
              :class="{ 'text-gray-500 italic': cell === null }"
            >
              {{ cell === null ? 'NULL' : cell }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="flex items-center justify-center h-full text-gray-500">
      No results yet. Execute a query to see results.
    </div>
    <div v-if="duration" class="px-3 py-1 text-xs text-gray-400 border-t border-gray-700">
      {{ rows.length }} row(s) in {{ duration }}
    </div>
  </div>
</template>
```

- [ ] **Step 2: 实现 StatusBar 组件**

创建 `ui/src/components/StatusBar.vue`：

```vue
<script setup lang="ts">
defineProps<{
  connected: boolean
  host?: string
  database?: string
  user?: string
}>()
</script>

<template>
  <div class="status-bar flex items-center gap-3 px-4 py-1.5 bg-gray-900 border-t border-gray-700 text-sm">
    <span class="flex items-center gap-1.5">
      <span :class="connected ? 'bg-green-400' : 'bg-red-400'" class="w-2 h-2 rounded-full inline-block"></span>
      <span class="text-gray-300">{{ connected ? 'Connected' : 'Disconnected' }}</span>
    </span>
    <span v-if="host" class="text-gray-400">{{ user }}@{{ host }}</span>
    <span v-if="database" class="text-blue-400">{{ database }}</span>
  </div>
</template>
```

- [ ] **Step 3: 实现 ConnectDialog 组件**

创建 `ui/src/components/ConnectDialog.vue`：

```vue
<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  connect: [dsn: string]
  close: []
}>()

const host = ref('127.0.0.1')
const port = ref(3306)
const user = ref('root')
const password = ref('')
const database = ref('')
const connecting = ref(false)

function onSubmit() {
  connecting.value = true
  const dsn = `${user.value}:${password.value}@tcp(${host.value}:${port.value})/${database.value}`
  emit('connect', dsn)
}

defineExpose({ setConnecting: (v: boolean) => { connecting.value = v } })
</script>

<template>
  <div class="fixed inset-0 bg-black/60 flex items-center justify-center z-50" @click.self="$emit('close')">
    <div class="bg-gray-800 rounded-lg shadow-xl p-6 w-96">
      <h2 class="text-lg font-semibold text-white mb-4">Connect to MySQL</h2>
      <form @submit.prevent="onSubmit" class="space-y-3">
        <div class="flex gap-3">
          <div class="flex-1">
            <label class="text-xs text-gray-400">Host</label>
            <input v-model="host" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
          </div>
          <div class="w-24">
            <label class="text-xs text-gray-400">Port</label>
            <input v-model.number="port" type="number" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
          </div>
        </div>
        <div>
          <label class="text-xs text-gray-400">User</label>
          <input v-model="user" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
        </div>
        <div>
          <label class="text-xs text-gray-400">Password</label>
          <input v-model="password" type="password" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
        </div>
        <div>
          <label class="text-xs text-gray-400">Database</label>
          <input v-model="database" class="w-full px-3 py-1.5 bg-gray-700 text-white rounded border border-gray-600 focus:border-blue-500 focus:outline-none" />
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <button type="button" @click="$emit('close')" class="px-4 py-1.5 text-gray-400 hover:text-white">Cancel</button>
          <button type="submit" :disabled="connecting" class="px-4 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded disabled:opacity-50">
            {{ connecting ? 'Connecting...' : 'Connect' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
```

- [ ] **Step 4: 验证前端编译**

```bash
cd /data/workspace/eyjian/myweb/ui && npm run build
```

- [ ] **Step 5: 提交**

```bash
git add ui/src/components/
git commit -m "feat: add ResultTable, StatusBar, and ConnectDialog components"
```

---

### Task 11: App.vue 主布局 — 组装所有组件

**文件：**
- 修改：`ui/src/App.vue`
- 修改：`ui/src/main.ts`

- [ ] **Step 1: 实现 App.vue**

替换 `ui/src/App.vue`：

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import SqlEditor from './components/SqlEditor.vue'
import ResultTable from './components/ResultTable.vue'
import StatusBar from './components/StatusBar.vue'
import ConnectDialog from './components/ConnectDialog.vue'
import { useApi } from './composables/useApi'
import { useWebSocket } from './composables/useWebSocket'
import type { WSMessage } from './composables/useWebSocket'

const { getStatus, connect } = useApi()
const { connected: wsConnected, onMessage, sendQuery } = useWebSocket()

const connected = ref(false)
const host = ref('')
const database = ref('')
const user = ref('')
const showConnect = ref(false)

// Query result state
const resultColumns = ref<string[]>([])
const resultRows = ref<unknown[][]>([])
const resultDuration = ref('')
const resultError = ref('')

const editorRef = ref<InstanceType<typeof SqlEditor> | null>(null)

// Load initial status
onMounted(async () => {
  try {
    const status = await getStatus()
    connected.value = status.connected
    host.value = status.host || ''
    database.value = status.database || ''
    user.value = status.user || ''
  } catch {
    connected.value = false
  }
})

// Handle WebSocket messages
onMessage.value = (msg: WSMessage) => {
  switch (msg.type) {
    case 'result_start':
      resultColumns.value = msg.columns
      resultRows.value = []
      resultError.value = ''
      break
    case 'result_row':
      resultRows.value = [...resultRows.value, msg.row]
      break
    case 'result_end':
      resultDuration.value = msg.time
      break
    case 'error':
      resultError.value = msg.message
      break
    case 'connected':
      connected.value = true
      host.value = msg.host
      database.value = msg.database
      user.value = msg.user
      break
    case 'info':
      // Show info message (e.g., "Reconnected to server")
      break
  }
}

async function executeQuery(sql: string) {
  resultError.value = ''
  resultColumns.value = []
  resultRows.value = []
  resultDuration.value = ''

  // Use WebSocket for streaming
  sendQuery(sql)
}

async function handleConnect(dsn: string) {
  try {
    const status = await connect(dsn)
    connected.value = status.connected
    host.value = status.host || ''
    database.value = status.database || ''
    user.value = status.user || ''
    showConnect.value = false
  } catch (err: any) {
    resultError.value = err.message
  }
}
</script>

<template>
  <div class="h-screen flex flex-col bg-gray-900 text-gray-100">
    <!-- Top bar -->
    <header class="flex items-center justify-between px-4 py-2 bg-gray-800 border-b border-gray-700">
      <div class="flex items-center gap-2">
        <span class="text-blue-400 font-bold">myweb</span>
        <span v-if="database" class="text-gray-400">{{ database }}@{{ host }}</span>
      </div>
      <button
        @click="showConnect = true"
        class="px-3 py-1 text-sm bg-blue-600 hover:bg-blue-500 rounded"
      >
        {{ connected ? 'Switch Connection' : 'Connect' }}
      </button>
    </header>

    <!-- Main content -->
    <div class="flex-1 flex flex-col overflow-hidden">
      <!-- SQL Editor -->
      <div class="h-48 border-b border-gray-700">
        <SqlEditor ref="editorRef" @execute="executeQuery" />
      </div>

      <!-- Action bar -->
      <div class="flex items-center gap-2 px-4 py-1.5 bg-gray-850 border-b border-gray-700">
        <button @click="editorRef?.getCode && executeQuery(editorRef.getCode())" class="px-3 py-1 text-sm bg-green-600 hover:bg-green-500 rounded flex items-center gap-1">
          ▶ Run
        </button>
        <button class="px-3 py-1 text-sm bg-gray-700 hover:bg-gray-600 rounded">
          ⏹ Cancel
        </button>
      </div>

      <!-- Result area -->
      <div class="flex-1 overflow-hidden">
        <ResultTable
          :columns="resultColumns"
          :rows="resultRows"
          :duration="resultDuration"
          :error="resultError"
        />
      </div>
    </div>

    <!-- Status bar -->
    <StatusBar
      :connected="connected"
      :host="host"
      :database="database"
      :user="user"
    />

    <!-- Connect dialog -->
    <ConnectDialog
      v-if="showConnect"
      @connect="handleConnect"
      @close="showConnect = false"
    />
  </div>
</template>
```

注意：`editorRef?.getCode()` 需要在 SqlEditor 中暴露 `getCode` 方法。更新 SqlEditor 的 defineExpose：

```typescript
defineExpose({ setCode, clearCode, getCode: () => code.value })
```

- [ ] **Step 2: 更新 main.ts**

替换 `ui/src/main.ts`：

```typescript
import { createApp } from 'vue'
import App from './App.vue'
import './style.css'

createApp(App).mount('#app')
```

- [ ] **Step 3: 验证前端编译**

```bash
cd /data/workspace/eyjian/myweb/ui && npm run build
```

- [ ] **Step 4: 提交**

```bash
git add ui/src/
git commit -m "feat: wire up App.vue with all components and WebSocket query execution"
```

---

### Task 12: Makefile + 端到端验证

**文件：**
- 创建：`Makefile`

- [ ] **Step 1: 创建 Makefile**

创建 `Makefile`：

```makefile
.PHONY: build build-ui dev test clean

VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

build: build-ui
	CGO_ENABLED=0 go build $(LDFLAGS) -o myweb ./cmd/myweb

build-ui:
	cd ui && npm run build

dev:
	go run ./cmd/myweb --dev --open=false

test:
	go test ./...

clean:
	rm -f myweb
	rm -rf ui/dist
```

- [ ] **Step 2: 全量构建**

```bash
cd /data/workspace/eyjian/myweb && make build
```

预期：先构建前端，再编译 Go 二进制，最终生成 `myweb` 可执行文件

- [ ] **Step 3: 验证二进制可运行**

```bash
./myweb --version
```

预期：输出 `myweb version dev`

- [ ] **Step 4: 提交**

```bash
git add Makefile
git commit -m "feat: add Makefile for build and dev"
```

---

### Task 13: 清理 + 最终提交

- [ ] **Step 1: 添加 .gitignore**

确保 `ui/.gitignore` 包含 `node_modules` 和 `dist`（Vite 模板自带）。在项目根目录也添加：

创建 `.gitignore`：

```
myweb
ui/node_modules/
ui/dist/
*.exe
```

注意：`ui/dist/` 在开发时需要排除，但 `go:embed` 需要它存在。在 release 构建时手动 `make build` 即可。

- [ ] **Step 2: 删除占位 index.html（如果 Task 3 创建的）**

如果 `ui/dist/index.html` 是占位文件且 `npm run build` 已生成真实版本，无需额外操作。

- [ ] **Step 3: 验证完整构建流程**

```bash
cd /data/workspace/eyjian/myweb && make clean && make build && ./myweb --version
```

- [ ] **Step 4: 最终提交**

```bash
git add -A
git commit -m "chore: add .gitignore and cleanup"
git push origin main
```
