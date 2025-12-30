package inapp

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/channels"
	"github.com/MuhibNayem/connectify-v2/notification-service/pkg/models"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

// clientConn wraps a websocket connection with a write channel
// This implements the "single writer per connection" pattern
type clientConn struct {
	conn   *websocket.Conn
	userID string
	send   chan []byte
}

// WebSocketChannel delivers notifications via WebSocket connections
// Production features:
// - Redis Pub/Sub for multi-instance broadcasting
// - Connection limits (backpressure)
// - Per-user connection caps
// - Single writer per connection (no concurrent writes)
type WebSocketChannel struct {
	clients     map[string][]*clientConn
	mu          sync.RWMutex
	connCount   int
	redisClient *redis.Client

	// Backpressure Config
	maxTotalConnections int
	maxPerUserConns     int
}

// WebSocketConfig for customization
type WebSocketConfig struct {
	RedisClient         *redis.Client
	MaxTotalConnections int
	MaxPerUserConns     int
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
	sendBufferSize = 256
)

func NewWebSocketChannel(cfg WebSocketConfig) *WebSocketChannel {
	if cfg.MaxTotalConnections == 0 {
		cfg.MaxTotalConnections = 10000
	}
	if cfg.MaxPerUserConns == 0 {
		cfg.MaxPerUserConns = 5
	}

	wsc := &WebSocketChannel{
		clients:             make(map[string][]*clientConn),
		redisClient:         cfg.RedisClient,
		maxTotalConnections: cfg.MaxTotalConnections,
		maxPerUserConns:     cfg.MaxPerUserConns,
	}

	// Subscribe to Redis for cross-pod broadcasting
	if cfg.RedisClient != nil {
		go wsc.subscribeToRedis()
	}

	return wsc
}

func (w *WebSocketChannel) Name() string {
	return "inapp"
}

// Send publishes notification to Redis for cross-pod delivery
// Uses consistent channel format: notifications:broadcast with wrapper
func (w *WebSocketChannel) Send(ctx context.Context, notification *models.Notification, recipient *channels.ChannelRecipient) error {
	if w.redisClient == nil {
		// Fallback: local delivery only
		return w.deliverLocal(recipient.UserID, notification)
	}

	// Wrap with userID for filtering on subscriber side
	wrapper := struct {
		UserID       string               `json:"user_id"`
		Notification *models.Notification `json:"notification"`
	}{
		UserID:       recipient.UserID,
		Notification: notification,
	}

	payload, err := json.Marshal(wrapper)
	if err != nil {
		return err
	}

	// Publish to global broadcast channel (all pods subscribe to this)
	return w.redisClient.Publish(ctx, "notifications:broadcast", payload).Err()
}

// deliverLocal sends to connections on this instance only
func (w *WebSocketChannel) deliverLocal(userID string, notification *models.Notification) error {
	w.mu.RLock()
	conns := w.clients[userID]
	w.mu.RUnlock()

	if len(conns) == 0 {
		return nil
	}

	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	// Non-blocking send to each connection's write pump
	for _, cc := range conns {
		select {
		case cc.send <- data:
			// Message queued
		default:
			// Buffer full, drop message for this connection
		}
	}
	return nil
}

func (w *WebSocketChannel) SendBatch(ctx context.Context, notifications []*models.Notification, recipients []*channels.ChannelRecipient) error {
	for i, notif := range notifications {
		w.Send(ctx, notif, recipients[i])
	}
	return nil
}

func (w *WebSocketChannel) IsEnabled() bool {
	return true
}

func (w *WebSocketChannel) HealthCheck(ctx context.Context) error {
	if w.redisClient != nil {
		return w.redisClient.Ping(ctx).Err()
	}
	return nil
}

// RegisterClient registers a WebSocket connection with a dedicated write pump
func (w *WebSocketChannel) RegisterClient(userID string, conn *websocket.Conn) error {
	w.mu.Lock()

	// Backpressure checks
	if w.connCount >= w.maxTotalConnections {
		w.mu.Unlock()
		return ErrTooManyConnections
	}
	if len(w.clients[userID]) >= w.maxPerUserConns {
		w.mu.Unlock()
		return ErrTooManyUserConnections
	}

	cc := &clientConn{
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, sendBufferSize),
	}

	w.clients[userID] = append(w.clients[userID], cc)
	w.connCount++
	w.mu.Unlock()

	// Start both pumps
	go w.writePump(cc)
	go w.readPump(cc)

	return nil
}

// readPump pumps messages from the websocket connection to the hub.
// The application ensures that there is at most one reader on a connection per execution.
func (w *WebSocketChannel) readPump(cc *clientConn) {
	defer func() {
		cc.conn.Close()
		w.UnregisterClient(cc.userID, cc.conn)
	}()

	cc.conn.SetReadLimit(maxMessageSize)
	cc.conn.SetReadDeadline(time.Now().Add(pongWait))
	cc.conn.SetPongHandler(func(string) error {
		cc.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, _, err := cc.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error? For now just exit
			}
			break
		}
		// We don't expect clients to send messages, just Pongs and maybe Acks.
		// Ignoring received text/binary for now.
	}
}

// writePump handles all writes to a single connection
// This is the ONLY goroutine that writes to this connection
func (w *WebSocketChannel) writePump(cc *clientConn) {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		cc.conn.Close()
		// We don't unregister here to avoid double-unregister race, readPump handles it
	}()

	for {
		select {
		case message, ok := <-cc.send:
			cc.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed
				cc.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := cc.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			cc.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := cc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Backpressure errors
var (
	ErrTooManyConnections     = &BackpressureError{Message: "server at capacity, try again later"}
	ErrTooManyUserConnections = &BackpressureError{Message: "too many connections for this user"}
)

type BackpressureError struct {
	Message string
}

func (e *BackpressureError) Error() string {
	return e.Message
}

func (w *WebSocketChannel) UnregisterClient(userID string, conn *websocket.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()

	conns := w.clients[userID]
	for i, cc := range conns {
		if cc.conn == conn {
			close(cc.send) // Signal write pump to stop
			w.clients[userID] = append(conns[:i], conns[i+1:]...)
			w.connCount--
			break
		}
	}

	if len(w.clients[userID]) == 0 {
		delete(w.clients, userID)
	}
}

func (w *WebSocketChannel) GetConnectionCount() int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.connCount
}

// subscribeToRedis listens to the global broadcast channel
// and routes messages to local connections
func (w *WebSocketChannel) subscribeToRedis() {
	ctx := context.Background()
	pubsub := w.redisClient.Subscribe(ctx, "notifications:broadcast")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for msg := range ch {
		// Parse wrapper
		var wrapper struct {
			UserID       string               `json:"user_id"`
			Notification *models.Notification `json:"notification"`
		}
		if err := json.Unmarshal([]byte(msg.Payload), &wrapper); err != nil {
			continue
		}

		// Route to local connections (non-blocking)
		w.deliverLocal(wrapper.UserID, wrapper.Notification)
	}
}
