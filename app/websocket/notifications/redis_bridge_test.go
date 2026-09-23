package notifications

import (
	"encoding/json"
	"testing"
	"time"
)

func TestBridgeMsgNotificationRoundTrip(t *testing.T) {
	receiver := uint(9)
	sender := uint(3)
	readAt := time.Date(2026, 9, 23, 4, 0, 0, 0, time.UTC)
	created := time.Date(2026, 9, 23, 3, 0, 0, 0, time.UTC)

	msg := bridgeMsg{
		Kind:     bridgeKindNotification,
		Origin:   "abc",
		TenantID: 42,
		Notification: &notifWire{
			ID:         7,
			Title:      "hello",
			Content:    "world",
			Type:       "announcement",
			SenderID:   &sender,
			ReceiverID: &receiver,
			IsRead:     true,
			ReadAt:     &readAt,
			CreatedAt:  created,
		},
	}

	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}

	var decoded bridgeMsg
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Kind != bridgeKindNotification || decoded.TenantID != 42 {
		t.Fatalf("unexpected envelope: %+v", decoded)
	}
	n := wireToNotification(decoded.Notification)
	if n == nil || n.ID != 7 || n.Title != "hello" || n.ReceiverID == nil || *n.ReceiverID != 9 {
		t.Fatalf("unexpected notification: %+v", n)
	}
}

func TestBridgeMsgAdminPayloadRoundTrip(t *testing.T) {
	msg := bridgeMsg{
		Kind:     bridgeKindAdminPayload,
		Origin:   "x",
		TenantID: 1,
		AdminID:  5,
		Payload: map[string]any{
			"type":        "read_all",
			"receiver_id": float64(5),
		},
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded bridgeMsg
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Kind != bridgeKindAdminPayload || decoded.AdminID != 5 {
		t.Fatalf("unexpected: %+v", decoded)
	}
	if decoded.Payload["type"] != "read_all" {
		t.Fatalf("payload: %+v", decoded.Payload)
	}
}
