package notifications

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/carbon"

	"goravel/app/clients"
	"goravel/app/models"
)

const (
	bridgeKindNotification = "notification"
	bridgeKindAdminPayload = "admin_payload"
)

// bridgeMsg is the Redis Pub/Sub envelope for cross-process WS fan-out.
type bridgeMsg struct {
	Kind         string         `json:"kind"`
	Origin       string         `json:"origin"`
	TenantID     uint           `json:"tenant_id"`
	AdminID      uint           `json:"admin_id,omitempty"`
	Notification *notifWire     `json:"notification,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
}

// notifWire is a JSON-safe subset of models.Notification for the wire.
type notifWire struct {
	ID         uint       `json:"id"`
	Title      string     `json:"title"`
	Content    string     `json:"content"`
	Type       string     `json:"type"`
	SenderID   *uint      `json:"sender_id,omitempty"`
	ReceiverID *uint      `json:"receiver_id,omitempty"`
	IsRead     bool       `json:"is_read"`
	ReadAt     *time.Time `json:"read_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

var (
	bridgeInstanceID = newBridgeInstanceID()
	bridgeStarted    atomic.Bool
	bridgeStopOnce   sync.Once
	bridgeCancel     context.CancelFunc
)

func newBridgeInstanceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte(time.Now().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(b)
}

func bridgeEnabled() bool {
	return facades.Config().GetBool("websocket.redis_bridge", true)
}

func bridgeChannel() string {
	ch := facades.Config().GetString("websocket.redis_channel", "goravel:ws:notifications")
	if ch == "" {
		return "goravel:ws:notifications"
	}
	return ch
}

func bridgeConnection() string {
	name := facades.Config().GetString("websocket.redis_connection", "default")
	if name == "" {
		return "default"
	}
	return name
}

func notificationToWire(n *models.Notification) *notifWire {
	if n == nil {
		return nil
	}
	w := &notifWire{
		ID:         n.ID,
		Title:      n.Title,
		Content:    n.Content,
		Type:       n.Type,
		SenderID:   n.SenderID,
		ReceiverID: n.ReceiverID,
		IsRead:     n.IsRead,
		ReadAt:     n.ReadAt,
	}
	if n.CreatedAt != nil && !n.CreatedAt.IsZero() {
		w.CreatedAt = n.CreatedAt.StdTime()
	}
	return w
}

func wireToNotification(w *notifWire) *models.Notification {
	if w == nil {
		return nil
	}
	n := &models.Notification{
		Title:      w.Title,
		Content:    w.Content,
		Type:       w.Type,
		SenderID:   w.SenderID,
		ReceiverID: w.ReceiverID,
		IsRead:     w.IsRead,
		ReadAt:     w.ReadAt,
	}
	n.ID = w.ID
	if !w.CreatedAt.IsZero() {
		n.CreatedAt = carbon.NewDateTime(carbon.Parse(w.CreatedAt.UTC().Format(time.RFC3339)))
	}
	return n
}

// StartRedisBridge subscribes to the WS fan-out channel. Safe to call once per process.
// Intended to run from AppServiceProvider.Boot on every normal ./main start (API and Worker).
func StartRedisBridge() {
	if !bridgeEnabled() {
		return
	}
	if !bridgeStarted.CompareAndSwap(false, true) {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	bridgeCancel = cancel

	go runRedisSubscriber(ctx)
	facades.Log().Infof("websocket redis bridge started channel=%s", bridgeChannel())
}

// StopRedisBridge cancels the subscriber (tests / graceful shutdown).
func StopRedisBridge() {
	bridgeStopOnce.Do(func() {
		if bridgeCancel != nil {
			bridgeCancel()
		}
	})
}

func runRedisSubscriber(ctx context.Context) {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return
		}
		err := subscribeOnce(ctx)
		if ctx.Err() != nil {
			return
		}
		facades.Log().Warningf("websocket redis bridge subscribe ended: %v; retry in %s", err, backoff)
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		if backoff < 30*time.Second {
			backoff *= 2
		}
	}
}

func subscribeOnce(ctx context.Context) error {
	client, err := clients.GetRedisClient(bridgeConnection())
	if err != nil {
		return err
	}

	pubsub := client.Subscribe(ctx, bridgeChannel())
	defer func() { _ = pubsub.Close() }()

	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case msg, ok := <-ch:
			if !ok {
				return fmt.Errorf("websocket redis pubsub channel closed")
			}
			handleBridgeMessage(msg.Payload)
		}
	}
}

func handleBridgeMessage(raw string) {
	var msg bridgeMsg
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		return
	}
	// Same process already applied locally when publishing.
	if msg.Origin != "" && msg.Origin == bridgeInstanceID {
		return
	}

	switch msg.Kind {
	case bridgeKindNotification:
		n := wireToNotification(msg.Notification)
		if n == nil {
			return
		}
		Hub().deliverLocalNotification(msg.TenantID, n)
	case bridgeKindAdminPayload:
		if msg.Payload == nil {
			return
		}
		Hub().deliverLocalAdminPayload(msg.TenantID, msg.AdminID, msg.Payload)
	}
}

// publishBridge returns true when the message was published to Redis.
func publishBridge(msg bridgeMsg) bool {
	if !bridgeEnabled() || !bridgeStarted.Load() {
		return false
	}
	msg.Origin = bridgeInstanceID
	data, err := json.Marshal(msg)
	if err != nil {
		return false
	}
	client, err := clients.GetRedisClient(bridgeConnection())
	if err != nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Publish(ctx, bridgeChannel(), data).Err(); err != nil {
		facades.Log().Debugf("websocket redis publish failed: %v", err)
		return false
	}
	return true
}

func publishNotification(tenantID uint, notification *models.Notification) bool {
	return publishBridge(bridgeMsg{
		Kind:         bridgeKindNotification,
		TenantID:     tenantID,
		Notification: notificationToWire(notification),
	})
}

func publishAdminPayload(tenantID, adminID uint, payload map[string]any) bool {
	return publishBridge(bridgeMsg{
		Kind:     bridgeKindAdminPayload,
		TenantID: tenantID,
		AdminID:  adminID,
		Payload:  payload,
	})
}
