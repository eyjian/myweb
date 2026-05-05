package server

import (
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/eyjian/mysh/config"
	"github.com/eyjian/mysh/connection"
	"github.com/eyjian/mysh/executor"
	"github.com/eyjian/mysh/metadata"
)

// Config holds server configuration.
type Config struct {
	Addr     string
	Dev      bool
	Config   *config.Config
	Pool     *connection.Pool
	Meta     *metadata.Cache
	Executor *executor.Executor
	UIFiles  fs.FS // embedded UI files
}

// Server is the HTTP server for myweb.
type Server struct {
	cfg *Config
	mux *http.ServeMux
	hub *Hub
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
	if s.cfg.Dev {
		handler = corsMiddleware(handler)
	}
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
	sub, err := fs.Sub(s.cfg.UIFiles, "ui/dist")
	if err != nil {
		log.Printf("Warning: embedded UI files not found: %s", err)
		return
	}
	fileServer := http.FileServer(http.FS(sub))
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" || !strings.Contains(path, ".") {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}

// devProxy proxies requests to Vite dev server in development mode.
func (s *Server) devProxy(w http.ResponseWriter, r *http.Request) {
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
