package server

import (
	"mysh/config"
	"mysh/connection"
	"mysh/executor"
	"mysh/metadata"
)

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
	cfg *Config
}

// New creates a new Server.
func New(cfg *Config) *Server {
	return &Server{cfg: cfg}
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return nil
}
