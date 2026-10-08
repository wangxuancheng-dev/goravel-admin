package config

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisTLSConfigDisabled(t *testing.T) {
	assert.Nil(t, redisTLSConfig(false, "redis.example.com", "", false))
}

func TestRedisTLSConfigDefaultsServerNameToHost(t *testing.T) {
	cfg := redisTLSConfig(true, "hook-redis.example.cache.amazonaws.com", "", false)
	require.NotNil(t, cfg)
	assert.Equal(t, uint16(tls.VersionTLS12), cfg.MinVersion)
	assert.Equal(t, "hook-redis.example.cache.amazonaws.com", cfg.ServerName)
	assert.False(t, cfg.InsecureSkipVerify)
}

func TestRedisTLSConfigExplicitServerNameAndInsecure(t *testing.T) {
	cfg := redisTLSConfig(true, "127.0.0.1", "redis.internal", true)
	require.NotNil(t, cfg)
	assert.Equal(t, "redis.internal", cfg.ServerName)
	assert.True(t, cfg.InsecureSkipVerify)
}
