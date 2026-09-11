package notifications

import (
	"encoding/json"
	"sync"
	"time"

	"goravel/app/models"
)

type adminKey struct {
	tenantID uint
	adminID  uint
}

type NotificationHub struct {
	clients    map[adminKey]map[*notificationClient]bool
	register   chan *notificationClient
	unregister chan *notificationClient
	broadcast  chan broadcastMsg
	stop       chan struct{} // 停止信号
	mu         sync.RWMutex
}

type broadcastMsg struct {
	tenantID     uint
	notification *models.Notification
}

var hubInstance = newNotificationHub()

func init() {
	go hubInstance.run()
}

func Hub() *NotificationHub {
	return hubInstance
}

func newNotificationHub() *NotificationHub {
	return &NotificationHub{
		clients:    make(map[adminKey]map[*notificationClient]bool),
		register:   make(chan *notificationClient),
		unregister: make(chan *notificationClient),
		broadcast:  make(chan broadcastMsg, 100),
		stop:       make(chan struct{}),
	}
}

func (h *NotificationHub) run() {
	for {
		select {
		case <-h.stop:
			// 收到停止信号，关闭所有客户端连接并退出
			h.mu.Lock()
			for _, adminClients := range h.clients {
				for client := range adminClients {
					close(client.send)
				}
			}
			h.clients = make(map[adminKey]map[*notificationClient]bool)
			h.mu.Unlock()
			return
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		case msg := <-h.broadcast:
			h.dispatch(msg.tenantID, msg.notification)
		}
	}
}

func (h *NotificationHub) addClient(client *notificationClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := adminKey{tenantID: client.tenantID, adminID: client.adminID}
	if _, ok := h.clients[key]; !ok {
		h.clients[key] = make(map[*notificationClient]bool)
	}
	h.clients[key][client] = true
}

func (h *NotificationHub) removeClient(client *notificationClient) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := adminKey{tenantID: client.tenantID, adminID: client.adminID}
	if adminClients, ok := h.clients[key]; ok {
		if _, exists := adminClients[client]; exists {
			delete(adminClients, client)
			close(client.send)
		}
		if len(adminClients) == 0 {
			delete(h.clients, key)
		}
	}
}

func (h *NotificationHub) dispatch(tenantID uint, notification *models.Notification) {
	payload := h.payload(notification)
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	sendAll := notification.ReceiverID == nil
	for key, adminClients := range h.clients {
		if key.tenantID != tenantID {
			continue
		}
		if !sendAll && *notification.ReceiverID != key.adminID {
			continue
		}
		for client := range adminClients {
			select {
			case client.send <- data:
			default:
				// Slow or stuck client: drop this frame rather than blocking the hub.
			}
		}
	}
}

func (h *NotificationHub) payload(notification *models.Notification) map[string]any {
	var readAt *string
	if notification.ReadAt != nil {
		formatted := notification.ReadAt.Format(time.RFC3339)
		readAt = &formatted
	}

	var receiver uint
	if notification.ReceiverID != nil {
		receiver = *notification.ReceiverID
	}

	var sender uint
	if notification.SenderID != nil {
		sender = *notification.SenderID
	}

	return map[string]any{
		"id":          notification.ID,
		"title":       notification.Title,
		"content":     notification.Content,
		"type":        notification.Type,
		"sender_id":   sender,
		"receiver_id": receiver,
		"is_read":     notification.IsRead,
		"read_at":     readAt,
		"created_at":  notification.CreatedAt.Format(time.RFC3339),
	}
}

func (h *NotificationHub) Broadcast(tenantID uint, notification *models.Notification) {
	select {
	case h.broadcast <- broadcastMsg{tenantID: tenantID, notification: notification}:
		// 成功发送
	case <-h.stop:
		// Hub 已停止，忽略广播
	}
}

// SendToAdmin pushes an arbitrary JSON payload to one admin's connections (all tabs/windows).
func (h *NotificationHub) SendToAdmin(tenantID, adminID uint, payload map[string]any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	adminClients, ok := h.clients[adminKey{tenantID: tenantID, adminID: adminID}]
	if !ok {
		return
	}
	for client := range adminClients {
		select {
		case client.send <- data:
		default:
		}
	}
}

// Stop 停止 NotificationHub，关闭所有连接并退出 goroutine
func (h *NotificationHub) Stop() {
	close(h.stop)
}

func (h *NotificationHub) Stats() (int, int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	admins := len(h.clients)
	connections := 0
	for _, adminClients := range h.clients {
		connections += len(adminClients)
	}

	return admins, connections
}
