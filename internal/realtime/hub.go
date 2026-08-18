package realtime

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const redisChannel = "nandi:events"

// Client is one authenticated agent WebSocket.
type Client struct {
	ID       string
	TenantID uuid.UUID
	UserID   uuid.UUID
	send     chan []byte
	hub      *Hub
}

// Hub fans tenant-scoped events to local sockets and across processes via Redis.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[string]*Client
	rdb     *redis.Client
	log     zerolog.Logger
}

func NewHub(rdb *redis.Client, log zerolog.Logger) *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[string]*Client),
		rdb:     rdb,
		log:     log.With().Str("component", "realtime").Logger(),
	}
}

func presenceKey(tenantID, userID uuid.UUID) string {
	return "nandi:presence:" + tenantID.String() + ":" + userID.String()
}

func (h *Hub) SetPresence(ctx context.Context, tenantID, userID uuid.UUID, online bool) {
	if h.rdb == nil {
		return
	}
	key := presenceKey(tenantID, userID)
	if online {
		_ = h.rdb.Set(ctx, key, "online", 90*time.Second).Err()
		return
	}
	_ = h.rdb.Del(ctx, key).Err()
}

func (h *Hub) IsOnline(ctx context.Context, tenantID, userID uuid.UUID) bool {
	if h.rdb == nil {
		h.mu.RLock()
		defer h.mu.RUnlock()
		for _, c := range h.clients[tenantID] {
			if c.UserID == userID {
				return true
			}
		}
		return false
	}
	n, err := h.rdb.Exists(ctx, presenceKey(tenantID, userID)).Result()
	return err == nil && n > 0
}

func (h *Hub) Register(tenantID, userID uuid.UUID) *Client {
	c := &Client{
		ID:       uuid.NewString(),
		TenantID: tenantID,
		UserID:   userID,
		send:     make(chan []byte, 32),
		hub:      h,
	}
	h.mu.Lock()
	if h.clients[tenantID] == nil {
		h.clients[tenantID] = map[string]*Client{}
	}
	h.clients[tenantID][c.ID] = c
	h.mu.Unlock()
	h.SetPresence(context.Background(), tenantID, userID, true)
	h.log.Info().Str("tenant_id", tenantID.String()).Str("user_id", userID.String()).Msg("ws connected")
	return c
}

func (h *Hub) Unregister(c *Client) {
	if c == nil {
		return
	}
	h.mu.Lock()
	if bucket, ok := h.clients[c.TenantID]; ok {
		delete(bucket, c.ID)
		if len(bucket) == 0 {
			delete(h.clients, c.TenantID)
		}
	}
	h.mu.Unlock()
	close(c.send)
	stillConnected := false
	h.mu.RLock()
	for _, other := range h.clients[c.TenantID] {
		if other.UserID == c.UserID {
			stillConnected = true
			break
		}
	}
	h.mu.RUnlock()
	if !stillConnected {
		h.SetPresence(context.Background(), c.TenantID, c.UserID, false)
	}
	h.log.Info().Str("tenant_id", c.TenantID.String()).Str("user_id", c.UserID.String()).Msg("ws disconnected")
}

func (h *Hub) Publish(ctx context.Context, evt Event) {
	raw, err := json.Marshal(evt)
	if err != nil {
		h.log.Error().Err(err).Msg("marshal event")
		return
	}
	if h.rdb != nil {
		if err := h.rdb.Publish(ctx, redisChannel, raw).Err(); err != nil {
			h.log.Error().Err(err).Msg("redis publish failed; delivering locally")
			h.deliver(evt.TenantID, raw)
		}
		return
	}
	h.deliver(evt.TenantID, raw)
}

func (h *Hub) Run(ctx context.Context) {
	if h.rdb == nil {
		h.log.Warn().Msg("redis unavailable; realtime is local-process only")
		return
	}
	pubsub := h.rdb.Subscribe(ctx, redisChannel)
	defer pubsub.Close()

	ch := pubsub.Channel()
	h.log.Info().Msg("subscribed to redis events")
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var evt Event
			if err := json.Unmarshal([]byte(msg.Payload), &evt); err != nil {
				h.log.Warn().Err(err).Msg("bad redis event")
				continue
			}
			h.deliver(evt.TenantID, []byte(msg.Payload))
		}
	}
}

func (h *Hub) deliver(tenantID string, raw []byte) {
	id, err := uuid.Parse(tenantID)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients[id] {
		select {
		case c.send <- raw:
		default:
			h.log.Warn().Str("client_id", c.ID).Msg("dropping event; client send buffer full")
		}
	}
}

func (h *Hub) PresenceCount(tenantID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[tenantID])
}

func (c *Client) WritePump(conn *websocket.Conn) {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		_ = conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump(conn *websocket.Conn) {
	defer func() {
		c.hub.Unregister(c)
		_ = conn.Close()
	}()
	conn.SetReadLimit(4096)
	_ = conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(70 * time.Second))
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}
