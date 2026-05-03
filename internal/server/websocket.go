package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// WSAuthFunc validates a token string and returns (userID, ok).
type WSAuthFunc func(token string) (userID string, ok bool)

// WSHub manages WebSocket connections and broadcasts events.
type WSHub struct {
	mu         sync.RWMutex
	clients    map[*WSClient]bool
	register   chan *WSClient
	unregister chan *WSClient
	logger     zerolog.Logger
	authFn     WSAuthFunc
}

// WSClient represents a single WebSocket connection.
type WSClient struct {
	hub     *WSHub
	conn    *websocket.Conn
	send    chan []byte
	project string // filter by project, empty = all
}

// WSMessage is the envelope for WebSocket messages.
type WSMessage struct {
	Type      string      `json:"type"`
	ProjectID string      `json:"project_id,omitempty"`
	TaskID    string      `json:"task_id,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewWSHub creates a new WebSocket hub.
func NewWSHub(logger zerolog.Logger) *WSHub {
	return &WSHub{
		clients:    make(map[*WSClient]bool),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		logger:     logger,
	}
}

// SetAuthFunc sets the authentication function for WebSocket connections.
func (h *WSHub) SetAuthFunc(fn WSAuthFunc) {
	h.authFn = fn
}

// Run starts the hub's event loop.
func (h *WSHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			h.logger.Info().Str("project", client.project).Msg("websocket client connected")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
			h.logger.Info().Str("project", client.project).Msg("websocket client disconnected")
		}
	}
}

// Broadcast sends a message to all connected clients (filtered by project).
func (h *WSHub) Broadcast(msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to marshal websocket message")
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for client := range h.clients {
		if client.project != "" && msg.ProjectID != "" && client.project != msg.ProjectID {
			continue
		}
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// ClientCount returns the number of connected clients.
func (h *WSHub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// HandleWebSocket upgrades an HTTP connection to WebSocket.
func (h *WSHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Authentication check
	if h.authFn != nil {
		token := extractWSToken(r)
		if token == "" {
			http.Error(w, "missing authentication token", http.StatusUnauthorized)
			return
		}
		if _, ok := h.authFn(token); !ok {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("websocket upgrade failed")
		return
	}

	project := r.URL.Query().Get("project")
	client := &WSClient{
		hub:     h,
		conn:    conn,
		send:    make(chan []byte, 256),
		project: project,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *WSClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ConnectEventBus subscribes the WebSocket hub to the event bus.
func (h *WSHub) ConnectEventBus(bus *EventBus) {
	bus.Subscribe("*", func(event Event) {
		h.Broadcast(WSMessage{
			Type:      event.Type,
			ProjectID: event.ProjectID,
			TaskID:    event.TaskID,
			Data:      event.Data,
			Timestamp: time.Now().UTC(),
		})
	})
}

// RegisterWSRoutes registers the WebSocket endpoint.
func (s *Server) registerWSRoutes(r chi.Router) {
	r.Get("/ws", s.wsHub.HandleWebSocket)
}

// extractWSToken gets the auth token from query param or Sec-WebSocket-Protocol header.
func extractWSToken(r *http.Request) string {
	if t := r.URL.Query().Get("token"); t != "" {
		return t
	}
	if proto := r.Header.Get("Sec-WebSocket-Protocol"); proto != "" {
		for _, p := range strings.Split(proto, ",") {
			p = strings.TrimSpace(p)
			if strings.HasPrefix(p, "bearer.") {
				return strings.TrimPrefix(p, "bearer.")
			}
		}
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}
