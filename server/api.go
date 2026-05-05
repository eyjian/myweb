package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/eyjian/mysh/config"
	"github.com/eyjian/mysh/connection"
	"github.com/eyjian/mysh/executor"
	"github.com/eyjian/mysh/metadata"
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
		resp.Host = s.cfg.Config.Connection.Host
		resp.Database = s.cfg.Pool.CurrentDB()
		resp.User = s.cfg.Config.Connection.User
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
		_ = s.cfg.Pool.Close()
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
		db = s.cfg.Pool.CurrentDB()
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
		db = s.cfg.Pool.CurrentDB()
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
	cfg := &config.ConnectionConfig{}
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
	closingParen := strings.Index(rest, ")")
	if closingParen >= 0 {
		rest = rest[:closingParen] + rest[closingParen+1:]
	}
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
