package payment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/models"
)

func TestApplyPaidResultStatusGate(t *testing.T) {
	skip, err := ApplyPaidResultStatusGate(models.PaymentStatusPaid)
	require.NoError(t, err)
	assert.True(t, skip)

	skip, err = ApplyPaidResultStatusGate(models.PaymentStatusPending)
	require.NoError(t, err)
	assert.False(t, skip)

	skip, err = ApplyPaidResultStatusGate(models.PaymentStatusFailed)
	require.Error(t, err)
	assert.False(t, skip)
	be, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrPaymentStatusInvalid.Code, be.Code)
}
