package ws

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/gocronx/kubevision/internal/kubernetes/informer"
)

const (
	// writeWait is the time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// pongWait is the time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// pingPeriod sends pings to the peer at this interval. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize is the maximum message size allowed from peer.
	maxMessageSize = 4096

	// sendBufferSize is the buffer size for the client send channel.
	sendBufferSize = 256
)

// allowedOriginHosts is the set of hosts permitted to make cross-origin
// WebSocket connections in addition to the backend's own Host. Populated at
// startup from the KUBEVISION_ALLOWED_ORIGINS environment variable (comma-
// separated list of host[:port] values) so that development front-ends
// (e.g. Vite on localhost:5173) are not blocked without disabling the check
// entirely.
var allowedOriginHosts = parseAllowedOrigins(os.Getenv("KUBEVISION_ALLOWED_ORIGINS"))

// parseAllowedOrigins splits a comma-separated list of host[:port] values into
// a set for O(1) membership testing. Empty entries are silently ignored.
func parseAllowedOrigins(raw string) map[string]bool {
	result := make(map[string]bool)
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry != "" {
			result[entry] = true
		}
	}
	return result
}

// upgrader is the WebSocket upgrader that validates the Origin header against
// the request's Host header to prevent cross-origin WebSocket hijacking.
// In development mode (GIN_MODE != "release") cross-origin connections from
// localhost are also permitted so that the Vite dev-server (default port 5173)
// can reach the backend without disabling origin checks entirely.
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// No Origin header means a same-origin or non-browser request — allow.
		if origin == "" {
			return true
		}
		// Parse the Origin header and compare its host (host[:port]) against
		// the request's Host header so cross-origin browser requests are
		// rejected while same-origin connections continue to work.
		originURL, err := url.Parse(origin)
		if err != nil {
			return false
		}
		// url.Parse puts host[:port] in URL.Host.
		originHost := originURL.Host
		if originHost == r.Host {
			return true
		}
		// Allow explicitly whitelisted origins (set via KUBEVISION_ALLOWED_ORIGINS).
		if allowedOriginHosts[originHost] {
			return true
		}
		// In non-release (development) mode, permit any localhost origin so
		// that a Vite/webpack dev-server on a different port is not blocked.
		// In release mode only the same-origin and whitelisted origins pass.
		if gin.Mode() != gin.ReleaseMode {
			host := originURL.Hostname() // host without port
			return host == "localhost" || host == "127.0.0.1" || host == "::1"
		}
		return false
	},
}

// Client represents a single WebSocket connection.
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	topics map[string]bool // subscribed topics (e.g., "cluster1:pods", "cluster1:deployments")
	mu     sync.RWMutex
}

// subscribeMessage is the JSON payload sent by the client to subscribe/unsubscribe.
type subscribeMessage struct {
	Action string   `json:"action"` // "subscribe" | "unsubscribe"
	Topics []string `json:"topics"` // e.g., ["cluster1:pods", "cluster1:deployments"]
}

// Hub manages WebSocket client connections and broadcasts resource events.
// It uses channel-based communication with a single goroutine event loop to
// avoid concurrent map access.
type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	stopCh     chan struct{}
	logger     *zap.Logger
}

// NewHub creates a new WebSocket Hub.
func NewHub(logger *zap.Logger) *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte, 1024),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stopCh:     make(chan struct{}),
		logger:     logger,
	}
}

// Stop signals the Run goroutine to exit. It is safe to call once during shutdown.
func (h *Hub) Stop() {
	close(h.stopCh)
}

// Run is the main event loop for the Hub. It must be started as a goroutine.
// It processes client registration, unregistration, and message broadcasting
// in a single goroutine to avoid needing locks on the clients map.
// It exits when Stop() is called.
func (h *Hub) Run() {
	for {
		select {
		case <-h.stopCh:
			return

		case client := <-h.register:
			h.clients[client] = true
			h.logger.Debug("websocket client registered",
				zap.Int("total_clients", len(h.clients)),
			)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				h.logger.Debug("websocket client unregistered",
					zap.Int("total_clients", len(h.clients)),
				)
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					// Client send buffer is full; disconnect the slow client.
					close(client.send)
					delete(h.clients, client)
					h.logger.Warn("slow websocket client disconnected")
				}
			}
		}
	}
}

// OnResourceEvent implements the informer.EventListener interface. It serialises
// the event to JSON and sends it to the broadcast channel non-blocking.
func (h *Hub) OnResourceEvent(event informer.ResourceEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		h.logger.Error("failed to marshal resource event", zap.Error(err))
		return
	}

	// Non-blocking send: drop the event if the broadcast channel is full
	// rather than blocking the informer event pipeline.
	select {
	case h.broadcast <- data:
	default:
		h.logger.Warn("broadcast channel full, dropping event",
			zap.String("cluster", event.ClusterID),
			zap.String("resource", event.Resource),
			zap.String("name", event.Name),
		)
	}
}

// HandleWatch is the Gin handler for GET /api/v1/ws/watch.
// It upgrades the HTTP connection to WebSocket and registers the client with
// the Hub.
func (h *Hub) HandleWatch(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("websocket upgrade failed", zap.Error(err))
		return
	}

	client := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan []byte, sendBufferSize),
		topics: make(map[string]bool),
	}

	h.register <- client

	// Start read and write pumps in separate goroutines.
	go client.writePump()
	go client.readPump()
}

// readPump reads messages from the WebSocket connection. It handles subscription
// management and heartbeat (pong) messages. When the connection is closed or an
// error occurs, it unregisters the client from the Hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure,
			) {
				c.hub.logger.Warn("websocket read error", zap.Error(err))
			}
			break
		}

		// Parse subscription messages.
		var msg subscribeMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		c.mu.Lock()
		switch msg.Action {
		case "subscribe":
			for _, topic := range msg.Topics {
				c.topics[topic] = true
			}
		case "unsubscribe":
			for _, topic := range msg.Topics {
				delete(c.topics, topic)
			}
		}
		c.mu.Unlock()
	}
}

// writePump writes messages from the send channel to the WebSocket connection.
// It also sends periodic ping messages to detect dead connections.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel.
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// Check if the client is subscribed to this event's topic.
			if !c.isSubscribed(message) {
				continue
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// Batch pending messages into the same write to reduce syscalls.
			n := len(c.send)
			for i := 0; i < n; i++ {
				pending := <-c.send
				if c.isSubscribed(pending) {
					_, _ = w.Write([]byte{'\n'})
					_, _ = w.Write(pending)
				}
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// isSubscribed checks whether the client has subscribed to the topic matching
// the given event message. If the client has no subscriptions, all events are
// delivered (broadcast mode).
func (c *Client) isSubscribed(message []byte) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// No subscriptions means receive everything.
	if len(c.topics) == 0 {
		return true
	}

	// Quick-parse to extract clusterId and resource for topic matching.
	var partial struct {
		ClusterID string `json:"clusterId"`
		Resource  string `json:"resource"`
	}
	if err := json.Unmarshal(message, &partial); err != nil {
		return true // deliver unparseable messages
	}

	// Match topic format: "clusterID:resource"
	topic := partial.ClusterID + ":" + partial.Resource
	return c.topics[topic]
}
