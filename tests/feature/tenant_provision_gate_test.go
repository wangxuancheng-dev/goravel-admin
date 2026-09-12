package feature_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/tests"
)

func insertPendingTenant(t *testing.T, code string) *models.Tenant {
	t.Helper()
	tenant := models.Tenant{
		Code:            code,
		Name:            "Pending Gate " + code,
		Status:          models.TenantStatusActive,
		ProvisionStatus: models.TenantProvisionPending,
		Driver:          "mysql",
		Isolation:       models.TenantIsolationDatabase,
		Database:        "tenant_" + code,
		ConnectionName:  fmt.Sprintf("tenant_gate_%s", code),
	}
	require.NoError(t, appfacades.PlatformOrmQuery(nil).Create(&tenant))
	t.Cleanup(func() {
		_, _ = appfacades.PlatformOrmQuery(nil).Where("id", tenant.ID).ForceDelete(&models.Tenant{})
	})
	return &tenant
}

func TestPendingTenantRejectedOnAdmin(t *testing.T) {
	withTenancyDriver(t, "database")
	code := fmt.Sprintf("pend%d", time.Now().Unix()%1000000)
	_ = insertPendingTenant(t, code)

	testCase := tests.TestCase{}
	resp, err := testCase.Http(t).
		WithHeader("Authorization", "Bearer unused").
		WithHeader("X-Tenant-ID", code).
		Get("/api/admin/info")
	require.NoError(t, err)
	content, err := resp.Content()
	require.NoError(t, err)
	assert.Contains(t, content, "tenant_not_ready")
}

func TestSetStatusActiveBlockedWhenPending(t *testing.T) {
	withTenancyDriver(t, "database")
	code := fmt.Sprintf("ena%d", time.Now().Unix()%1000000)
	tenant := insertPendingTenant(t, code)
	tenant.Status = models.TenantStatusDisabled
	_, err := appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
		"status": models.TenantStatusDisabled,
	})
	require.NoError(t, err)

	admin := services.NewTenantAdminService()
	_, err = admin.SetStatus(tenant.ID, models.TenantStatusActive)
	require.Error(t, err)
	biz, ok := apperrors.GetBusinessError(err)
	require.True(t, ok)
	assert.Equal(t, apperrors.ErrTenantNotReady.Code, biz.Code)
}
