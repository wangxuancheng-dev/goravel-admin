package notifications

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"github.com/coder/websocket"
)

type notificationClient struct {
	hub      *NotificationHub
	conn     *websocket.Conn
	send     chan []byte
	connID   string
	tenantID uint
	adminID  uint
}

func newConnID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return bridgeInstanceID + ":" + hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return bridgeInstanceID + ":" + hex.EncodeToString(b)
}

func newNotificationClient(hub *NotificationHub, conn *websocket.Conn, tenantID, adminID uint) *notificationClient {
	return &notificationClient{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		connID:   newConnID(),
		tenantID: tenantID,
		adminID:  adminID,
	}
}

// serve pushes notifications until the peer disconnects or the send channel closes.
// CloseRead handles control frames (ping/pong/close); do not call Read/Ping concurrently.
func (c *notificationClient) serve() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.CloseNow()
	}()

	ctx := c.conn.CloseRead(context.Background())
	ticker := time.NewTicker(presenceHeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				_ = c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := c.conn.Write(wctx, websocket.MessageText, message)
			cancel()
			if err != nil {
				return
			}
		case <-ticker.C:
			presenceTouch(c)
		case <-ctx.Done():
			return
		}
	}
}

func (h *NotificationHub) RegisterConnection(conn *websocket.Conn, tenantID, adminID uint) {
	client := newNotificationClient(h, conn, tenantID, adminID)
	h.register <- client
	go client.serve()
}
