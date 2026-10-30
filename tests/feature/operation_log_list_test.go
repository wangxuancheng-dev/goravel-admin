package feature_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/services"
)

// Pages beyond helpers.MaxListRows are refused instead of running an ever slower OFFSET scan.
func TestOperationLogListRefusesDeepPages(t *testing.T) {
	svc := services.NewOperationLogService(context.Background())

	_, _, err := svc.GetList(services.OperationLogFilters{}, helpers.MaxListRows/10, 10)
	require.NoError(t, err, "the last page within the cap still works")

	_, _, err = svc.GetList(services.OperationLogFilters{}, helpers.MaxListRows/10+1, 10)
	businessErr, ok := apperrors.GetBusinessError(err)
	require.True(t, ok, "want a business error, got %v", err)
	assert.Equal(t, apperrors.ErrDeepPaginationExceeded.Code, businessErr.Code)
}
