package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meettoy2004/lnmonja/internal/models"
	"github.com/meettoy2004/lnmonja/internal/storage"
	"go.uber.org/zap"
)

// WebSocketServer handles WebSocket connections for real-time updates
type WebSocketServer struct {
	upgrader  websocket.Upgrader
	clients   map[*WebSocketClient]bool
	clientsMu sync.RWMutex
	broadcast chan *WSMessage
	store     storage.Storage
	logger    *zap.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
}

// WebSocketClient represents a connected WebSocket client
type WebSocketClient struct {
	conn       *websocket.Conn
	send       chan []byte
	server     *WebSocketServer
	subscriptions map[string]bool
	subsMu     sync.RWMutex
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
	NodeID    string      `json:"node_id,omitempty"`
}

// NewWebSocketServer creates a new WebSocket server
func NewWebSocketServer(store storage.Storage, logger *zap.Logger) *WebSocketServer {
	ctx, cancel := context.WithCancel(context.Background())

	ws := &WebSocketServer{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		clients:   make(map[*WebSocketClient]bool),
		broadcast: make(chan *WSMessage, 1000),
		store:     store,
		logger:    logger,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start broadcast handler
	ws.wg.Add(1)
	go ws.handleBroadcasts()

	return ws
}

// ServeHTTP handles WebSocket upgrade requests
func (ws *WebSocketServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	client := &WebSocketClient{
		conn:          conn,
		send:          make(chan []byte, 256),
		server:        ws,
		subscriptions: make(map[string]bool),
	}

	ws.clientsMu.Lock()
	ws.clients[client] = true
	ws.clientsMu.Unlock()

	ws.logger.Info("New WebSocket client connected",
		zap.String("remote_addr", r.RemoteAddr),
	)

	// Start client goroutines
	go client.writePump()
	go client.readPump()
}

// handleBroadcasts handles broadcasting messages to all clients
func (ws *WebSocketServer) handleBroadcasts() {
	defer ws.wg.Done()

	for {
		select {
		case <-ws.ctx.Done():
			return
		case message := <-ws.broadcast:
			ws.clientsMu.RLock()
			for client := range ws.clients {
				// Check if client is subscribed to this message type
				if !client.isSubscribed(message.Type) && !client.isSubscribed("all") {
					continue
				}

				data, err := json.Marshal(message)
				if err != nil {
					ws.logger.Error("Failed to marshal message", zap.Error(err))
					continue
				}

				select {
				case client.send <- data:
				default:
					// Client send buffer is full, close connection
					ws.removeClient(client)
				}
			}
			ws.clientsMu.RUnlock()
		}
	}
}

// BroadcastMetrics broadcasts metrics to subscribed clients
func (ws *WebSocketServer) BroadcastMetrics(metrics []*models.Metric) {
	if len(metrics) == 0 {
		return
	}

	message := &WSMessage{
		Type:      "metrics",
		Timestamp: time.Now(),
		Data:      metrics,
	}

	select {
	case ws.broadcast <- message:
	default:
		ws.logger.Warn("Broadcast channel full, dropping metrics update")
	}
}

// BroadcastAlert broadcasts an alert to all clients
func (ws *WebSocketServer) BroadcastAlert(alert *models.Alert) {
	message := &WSMessage{
		Type:      "alert",
		Timestamp: time.Now(),
		Data:      alert,
		NodeID:    alert.Labels["node"],
	}

	select {
	case ws.broadcast <- message:
	default:
		ws.logger.Warn("Broadcast channel full, dropping alert")
	}
}

// BroadcastNodeStatus broadcasts node status changes
func (ws *WebSocketServer) BroadcastNodeStatus(node *models.Node) {
	message := &WSMessage{
		Type:      "node_status",
		Timestamp: time.Now(),
		Data:      node,
		NodeID:    node.ID,
	}

	select {
	case ws.broadcast <- message:
	default:
		ws.logger.Warn("Broadcast channel full, dropping node status")
	}
}

// removeClient removes a client from the server
func (ws *WebSocketServer) removeClient(client *WebSocketClient) {
	ws.clientsMu.Lock()
	defer ws.clientsMu.Unlock()

	if _, ok := ws.clients[client]; ok {
		delete(ws.clients, client)
		close(client.send)
		client.conn.Close()
	}
}

// Close closes the WebSocket server
func (ws *WebSocketServer) Close() error {
	ws.cancel()
	ws.wg.Wait()

	ws.clientsMu.Lock()
	defer ws.clientsMu.Unlock()

	for client := range ws.clients {
		client.conn.Close()
		close(client.send)
	}

	return nil
}

// Client methods

// readPump reads messages from the WebSocket connection
func (c *WebSocketClient) readPump() {
	defer func() {
		c.server.removeClient(c)
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
				c.server.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		// Handle client messages (subscriptions, etc.)
		c.handleMessage(message)
	}
}

// writePump writes messages to the WebSocket connection
func (c *WebSocketClient) writePump() {
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

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to current websocket message
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

// handleMessage handles messages from the client
func (c *WebSocketClient) handleMessage(data []byte) {
	var msg struct {
		Type   string   `json:"type"`
		Topics []string `json:"topics"`
	}

	if err := json.Unmarshal(data, &msg); err != nil {
		c.server.logger.Error("Failed to unmarshal client message", zap.Error(err))
		return
	}

	switch msg.Type {
	case "subscribe":
		c.subscribe(msg.Topics)
	case "unsubscribe":
		c.unsubscribe(msg.Topics)
	case "ping":
		c.sendPong()
	default:
		c.server.logger.Warn("Unknown message type", zap.String("type", msg.Type))
	}
}

// subscribe subscribes the client to topics
func (c *WebSocketClient) subscribe(topics []string) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()

	for _, topic := range topics {
		c.subscriptions[topic] = true
	}

	c.server.logger.Debug("Client subscribed", zap.Strings("topics", topics))
}

// unsubscribe unsubscribes the client from topics
func (c *WebSocketClient) unsubscribe(topics []string) {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()

	for _, topic := range topics {
		delete(c.subscriptions, topic)
	}

	c.server.logger.Debug("Client unsubscribed", zap.Strings("topics", topics))
}

// isSubscribed checks if the client is subscribed to a topic
func (c *WebSocketClient) isSubscribed(topic string) bool {
	c.subsMu.RLock()
	defer c.subsMu.RUnlock()

	return c.subscriptions[topic]
}

// sendPong sends a pong response
func (c *WebSocketClient) sendPong() {
	response := map[string]string{"type": "pong"}
	data, err := json.Marshal(response)
	if err != nil {
		return
	}

	select {
	case c.send <- data:
	default:
	}
}

// GetConnectedClients returns the number of connected clients
func (ws *WebSocketServer) GetConnectedClients() int {
	ws.clientsMu.RLock()
	defer ws.clientsMu.RUnlock()
	return len(ws.clients)
}
