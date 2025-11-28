package api

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins in development
		// In production, check origin against allowed domains
		return true
	},
}

// Client represents a WebSocket client
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	id   string
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WebSocket] Client connected: %s (total: %d)", client.id, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("[WebSocket] Client disconnected: %s (total: %d)", client.id, len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(messageType string, event string, data interface{}) {
	message := map[string]interface{}{
		"type":      messageType,
		"event":     event,
		"data":      data,
		"timestamp": time.Now().Unix(),
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WebSocket] Error marshaling message: %v", err)
		return
	}

	h.broadcast <- jsonMessage
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Error: %v", err)
			}
			break
		}

		// Handle incoming messages (e.g., ping)
		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err == nil {
			if msgType, ok := msg["type"].(string); ok && msgType == "ping" {
				// Respond with pong
				pong := map[string]string{"type": "pong"}
				if pongJSON, err := json.Marshal(pong); err == nil {
					c.send <- pongJSON
				}
			}
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current WebSocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("[WebSocket] Upgrade error: %v", err)
		return
	}

	clientID := c.Request.RemoteAddr
	client := &Client{
		hub:  s.wsHub,
		conn: conn,
		send: make(chan []byte, 256),
		id:   clientID,
	}

	client.hub.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// handleSSE handles Server-Sent Events connections
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create a channel for this SSE connection
	eventChan := make(chan []byte, 10)

	// Create a temporary client to receive broadcasts
	client := &Client{
		hub:  s.wsHub,
		send: eventChan,
		id:   r.RemoteAddr,
	}

	s.wsHub.register <- client
	defer func() {
		s.wsHub.unregister <- client
	}()

	// Keep connection alive and send events
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case message := <-eventChan:
			// Send SSE formatted message
			w.Write([]byte("data: "))
			w.Write(message)
			w.Write([]byte("\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		case <-ticker.C:
			// Send keep-alive comment
			w.Write([]byte(": keepalive\n\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}
}

// BroadcastArtifactUpdate broadcasts artifact updates to all connected clients
func (s *Server) BroadcastArtifactUpdate(event string, data interface{}) {
	if s.wsHub != nil {
		s.wsHub.Broadcast("artifact", event, data)
	}
}

// BroadcastRepositoryUpdate broadcasts repository updates to all connected clients
func (s *Server) BroadcastRepositoryUpdate(event string, data interface{}) {
	if s.wsHub != nil {
		s.wsHub.Broadcast("repository", event, data)
	}
}

// BroadcastScanUpdate broadcasts scan updates to all connected clients
func (s *Server) BroadcastScanUpdate(event string, data interface{}) {
	if s.wsHub != nil {
		s.wsHub.Broadcast("scan", event, data)
	}
}

// BroadcastComplianceUpdate broadcasts compliance updates to all connected clients
func (s *Server) BroadcastComplianceUpdate(event string, data interface{}) {
	if s.wsHub != nil {
		s.wsHub.Broadcast("compliance", event, data)
	}
}

// BroadcastUserUpdate broadcasts user updates to all connected clients
func (s *Server) BroadcastUserUpdate(event string, data interface{}) {
	if s.wsHub != nil {
		s.wsHub.Broadcast("user", event, data)
	}
}
