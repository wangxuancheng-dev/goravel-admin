package gateways

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockNotifySignRoundTrip(t *testing.T) {
	sign := mockNotifySign("PAY1", "SUCCESS", 9.9, "secret")
	assert.True(t, hmacEqual(sign, mockNotifySign("PAY1", "SUCCESS", 9.9, "secret")))
	assert.False(t, hmacEqual(sign, mockNotifySign("PAY1", "SUCCESS", 9.91, "secret")))
}
