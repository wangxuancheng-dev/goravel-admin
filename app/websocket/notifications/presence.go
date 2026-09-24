package notifications

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/redis/go-redis/v9"

	"goravel/app/clients"
)

const (
	presenceTTL               = 45 * time.Second
	presenceHeartbeatInterval = 15 * time.Second
	presenceStatsTimeout      = 2 * time.Second
)

func presenceKey() string {
	key := facades.Config().GetString("websocket.presence_key", "goravel:ws:presence")
	if key == "" {
		return "goravel:ws:presence"
	}
	return key
}

func presenceEnabled() bool {
	return bridgeEnabled()
}

func presenceMember(connID string, tenantID, adminID uint) string {
	return fmt.Sprintf("%s|%d|%d", connID, tenantID, adminID)
}

func parsePresenceMember(member string) (tenantID, adminID uint, ok bool) {
	parts := strings.Split(member, "|")
	if len(parts) != 3 {
		return 0, 0, false
	}
	tid, err1 := strconv.ParseUint(parts[1], 10, 64)
	aid, err2 := strconv.ParseUint(parts[2], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return uint(tid), uint(aid), true
}

func countPresenceMembers(members []string) (admins, connections int) {
	if len(members) == 0 {
		return 0, 0
	}
	seen := make(map[adminKey]struct{}, len(members))
	for _, m := range members {
		tid, aid, ok := parsePresenceMember(m)
		if !ok {
			continue
		}
		connections++
		seen[adminKey{tenantID: tid, adminID: aid}] = struct{}{}
	}
	return len(seen), connections
}

func presenceExpireUnix() float64 {
	return float64(time.Now().Add(presenceTTL).Unix())
}

func presenceTouch(c *notificationClient) {
	if c == nil || c.connID == "" || !presenceEnabled() {
		return
	}
	client, err := clients.GetRedisClient(bridgeConnection())
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), presenceStatsTimeout)
	defer cancel()
	member := presenceMember(c.connID, c.tenantID, c.adminID)
	if err := client.ZAdd(ctx, presenceKey(), redis.Z{
		Score:  presenceExpireUnix(),
		Member: member,
	}).Err(); err != nil {
		facades.Log().Warningf("websocket presence touch failed: %v", err)
	}
}

func presenceRemove(c *notificationClient) {
	if c == nil || c.connID == "" || !presenceEnabled() {
		return
	}
	client, err := clients.GetRedisClient(bridgeConnection())
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), presenceStatsTimeout)
	defer cancel()
	member := presenceMember(c.connID, c.tenantID, c.adminID)
	if err := client.ZRem(ctx, presenceKey(), member).Err(); err != nil {
		facades.Log().Debugf("websocket presence remove failed: %v", err)
	}
}

// presenceClusterStats returns cluster-wide admin/connection counts from Redis.
// ok is false when Redis is unavailable (caller should fall back to local).
func presenceClusterStats() (admins, connections int, ok bool) {
	if !presenceEnabled() {
		return 0, 0, false
	}
	client, err := clients.GetRedisClient(bridgeConnection())
	if err != nil {
		return 0, 0, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), presenceStatsTimeout)
	defer cancel()

	key := presenceKey()
	now := strconv.FormatInt(time.Now().Unix(), 10)
	if err := client.ZRemRangeByScore(ctx, key, "-inf", now).Err(); err != nil {
		facades.Log().Debugf("websocket presence prune failed: %v", err)
		return 0, 0, false
	}
	members, err := client.ZRange(ctx, key, 0, -1).Result()
	if err != nil {
		facades.Log().Debugf("websocket presence stats failed: %v", err)
		return 0, 0, false
	}
	admins, connections = countPresenceMembers(members)
	return admins, connections, true
}
