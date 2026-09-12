package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotificationTypeAllowed(t *testing.T) {
	assert.True(t, notificationTypeAllowed("", "announcement"))
	assert.True(t, notificationTypeAllowed("announcement,login_anomaly", "announcement"))
	assert.True(t, notificationTypeAllowed("announcement,login_anomaly", "login_anomaly"))
	assert.False(t, notificationTypeAllowed("announcement,login_anomaly", "message"))
	assert.True(t, notificationTypeAllowed(" Announcement ", "announcement"))
}
