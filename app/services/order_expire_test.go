package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"goravel/app/models"
)

func TestOrderIsExpired(t *testing.T) {
	past := time.Now().UTC().Add(-time.Minute)
	future := time.Now().UTC().Add(time.Minute)
	assert.False(t, OrderIsExpired(nil, time.Now()))
	assert.False(t, OrderIsExpired(&models.Order{}, time.Now()))
	assert.True(t, OrderIsExpired(&models.Order{ExpireAt: &past}, time.Now().UTC()))
	assert.False(t, OrderIsExpired(&models.Order{ExpireAt: &future}, time.Now().UTC()))
}
