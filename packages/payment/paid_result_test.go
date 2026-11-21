package payment

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyPaidResultStatusGate(t *testing.T) {
	skip, err := ApplyPaidResultStatusGate(StatusPaid)
	require.NoError(t, err)
	assert.True(t, skip)

	skip, err = ApplyPaidResultStatusGate(StatusPending)
	require.NoError(t, err)
	assert.False(t, skip)

	skip, err = ApplyPaidResultStatusGate(StatusFailed)
	require.Error(t, err)
	assert.False(t, skip)
	assert.True(t, errors.Is(err, ErrStatusInvalid))
}
