package clients

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/redis/go-redis/v9"
)

var (
	redisClients sync.Map
	redisMutex   sync.Mutex
)

// GetRedisClient 获取 Redis 客户端（使用连接池，支持多连接）
// connectionName: Redis 连接名称，默认为 "default"
func GetRedisClient(connectionName string) (*redis.Client, error) {
	if connectionName == "" {
		connectionName = "default"
	}

	if client, ok := redisClients.Load(connectionName); ok {
		redisClient := client.(*redis.Client)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		err := redisClient.Ping(ctx).Err()
		cancel()
		if err == nil {
			return redisClient, nil
		}
		redisClients.Delete(connectionName)
	}

	redisMutex.Lock()
	defer redisMutex.Unlock()

	if client, ok := redisClients.Load(connectionName); ok {
		return client.(*redis.Client), nil
	}

	client, err := createRedisClient(connectionName)
	if err != nil {
		return nil, err
	}
	redisClients.Store(connectionName, client)
	return client, nil
}

func createRedisClient(connectionName string) (*redis.Client, error) {
	host := facades.Config().GetString(fmt.Sprintf("database.redis.%s.host", connectionName), "")
	if host == "" {
		host = facades.Config().GetString("database.redis.default.host", "127.0.0.1")
	}

	port := facades.Config().GetInt(fmt.Sprintf("database.redis.%s.port", connectionName), 0)
	if port == 0 {
		port = facades.Config().GetInt("database.redis.default.port", 6379)
	}

	password := facades.Config().GetString(fmt.Sprintf("database.redis.%s.password", connectionName), "")
	if password == "" {
		password = facades.Config().GetString("database.redis.default.password", "")
	}

	db := facades.Config().GetInt(fmt.Sprintf("database.redis.%s.database", connectionName), -1)
	if db == -1 {
		db = facades.Config().GetInt("database.redis.default.database", 0)
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", host, port),
		Password:     password,
		DB:           db,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("Redis 连接失败 [%s]: %v", connectionName, err)
	}
	return client, nil
}

// CloseRedisClient 关闭指定的 Redis 客户端并从缓存中移除
func CloseRedisClient(connectionName string) error {
	if connectionName == "" {
		connectionName = "default"
	}
	if client, ok := redisClients.LoadAndDelete(connectionName); ok {
		return client.(*redis.Client).Close()
	}
	return nil
}

// CloseAllRedisClients 关闭所有缓存的 Redis 客户端
func CloseAllRedisClients() error {
	var errs []error
	redisClients.Range(func(key, value any) bool {
		if err := value.(*redis.Client).Close(); err != nil {
			errs = append(errs, err)
		}
		redisClients.Delete(key)
		return true
	})
	if len(errs) > 0 {
		return fmt.Errorf("关闭 Redis 客户端时发生错误: %v", errs)
	}
	return nil
}
