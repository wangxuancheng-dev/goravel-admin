package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("websocket", map[string]any{
		// Cross-API notification fan-out via Redis Pub/Sub (auto-started on Boot).
		"redis_bridge": config.Env("WEBSOCKET_REDIS_BRIDGE", true),
		// Redis connection name under database.redis.*
		"redis_connection": config.Env("WEBSOCKET_REDIS_CONNECTION", "default"),
		// Pub/Sub channel (shared by all API/Worker processes).
		"redis_channel": config.Env("WEBSOCKET_REDIS_CHANNEL", "goravel:ws:notifications"),
		// ZSET key for cluster-wide WS presence (monitor online_admins / connections).
		"presence_key": config.Env("WEBSOCKET_REDIS_PRESENCE_KEY", "goravel:ws:presence"),
	})
}
