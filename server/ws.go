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

// wsMessage represents a client-to-server WebSocket message.
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
		s.sendWS(client, wsOutMessage{
			Type:     "connected",
			Host:     s.cfg.Config.Connection.Host,
			Database: s.cfg.Pool.CurrentDB(),
			User:     s.cfg.Config.Connection.User,
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
	if s.cfg.Executor != nil {
		s.cfg.Executor.Cancel()
	}
	s.sendWS(c, wsOutMessage{Type: "info", Message: "query cancelled"})
}

func (s *Server) handleWSWatch(c *wsClient, msg wsMessage) {
	// Watch mode will be implemented in Phase 3
	s.sendWS(c, wsOutMessage{Type: "error", Message: "watch mode not yet implemented"})
}

func (s *Server) handleWSWatchStop(c *wsClient) {
	// Watch mode will be implemented in Phase 3
}
