package feature_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
)

// cleanupTenantEntitlementRows removes subscription/override/snapshot rows a test left for tenantID.
func cleanupTenantEntitlementRows(t *testing.T, tenantID uint) {
	t.Helper()
	t.Cleanup(func() {
		for _, m := range []any{
			&models.TenantSubscription{},
			&models.TenantEntitlementOverride{},
			&models.TenantEntitlementSnapshot{},
			&models.TenantUsageCounter{},
			&models.PlatformEntitlementAuditLog{},
		} {
			_, _ = appfacades.PlatformOrmQuery(nil).Where("tenant_id", tenantID).ForceDelete(m)
		}
	})
}

type tenantFeatureRow struct {
	Key         string `json:"key"`
	Effective   bool   `json:"effective"`
	HasOverride bool   `json:"has_override"`
}

type tenantEntitlementsPayload struct {
	Code int `json:"code"`
	Data struct {
		Summary struct {
			PlanCode string `json:"plan_code"`
		} `json:"summary"`
		Catalog []tenantFeatureRow `json:"catalog"`
	} `json:"data"`
}

func getTenantFeature(t *testing.T, token string, tenantID uint, key string) (string, tenantFeatureRow) {
	t.Helper()
	resp, content := platformCall(t, token, "GET", fmt.Sprintf("/api/platform/tenants/%d/entitlements", tenantID), "")
	require.True(t, resp.IsSuccessful(), content)
	var payload tenantEntitlementsPayload
	require.NoError(t, json.Unmarshal([]byte(content), &payload))
	for _, row := range payload.Data.Catalog {
		if row.Key == key {
			return payload.Data.Summary.PlanCode, row
		}
	}
	t.Fatalf("feature %s missing from tenant %d catalog", key, tenantID)
	return "", tenantFeatureRow{}
}

// Plan grant -> override off -> delete override restores the plan value.
func TestPlatformTenantEntitlementLifecycle(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)

	nonce := time.Now().UnixNano() % 1000000
	tenant := insertPendingTenant(t, fmt.Sprintf("ent%d", nonce))
	cleanupTenantEntitlementRows(t, tenant.ID)
	featureKey := fmt.Sprintf("zz_test.flag_%d", nonce)
	planCode := fmt.Sprintf("zz_test_plan_%d", nonce)
	t.Cleanup(func() {
		var plan models.PlatformPlan
		if err := appfacades.PlatformOrmQuery(nil).Where("code", planCode).First(&plan); err == nil && plan.ID > 0 {
			_, _ = appfacades.PlatformOrmQuery(nil).Where("plan_id", plan.ID).ForceDelete(&models.PlatformPlanEntitlement{})
			_, _ = appfacades.PlatformOrmQuery(nil).Where("id", plan.ID).ForceDelete(&models.PlatformPlan{})
		}
		_, _ = appfacades.PlatformOrmQuery(nil).Where("key", featureKey).ForceDelete(&models.PlatformFeature{})
		// Creating the feature refreshed every stored snapshot; refresh again so none keep the temp key.
		_ = services.NewEntitlementService(context.Background()).RecomputeAllSnapshots()
	})

	resp, content := platformCall(t, token, "POST", "/api/platform/features",
		fmt.Sprintf(`{"key":%q,"name":"Test flag","type":"boolean","default_value":"false"}`, featureKey))
	require.True(t, resp.IsSuccessful(), content)
	resp, content = platformCall(t, token, "POST", "/api/platform/plans",
		fmt.Sprintf(`{"code":%q,"name":"Test plan","entitlements":{%q:"true"}}`, planCode, featureKey))
	require.True(t, resp.IsSuccessful(), content)

	_, row := getTenantFeature(t, token, tenant.ID, featureKey)
	assert.False(t, row.Effective, "feature defaults to off before any plan grants it")

	tenantURI := fmt.Sprintf("/api/platform/tenants/%d", tenant.ID)
	resp, content = platformCall(t, token, "POST", tenantURI+"/subscription", fmt.Sprintf(`{"plan_code":%q}`, planCode))
	require.True(t, resp.IsSuccessful(), content)
	plan, row := getTenantFeature(t, token, tenant.ID, featureKey)
	assert.Equal(t, planCode, plan)
	assert.True(t, row.Effective, "plan grants the feature")
	assert.False(t, row.HasOverride)

	resp, content = platformCall(t, token, "PUT", tenantURI+"/entitlements/"+featureKey, `{"enabled":false,"reason":"feature test"}`)
	require.True(t, resp.IsSuccessful(), content)
	_, row = getTenantFeature(t, token, tenant.ID, featureKey)
	assert.False(t, row.Effective, "override beats the plan grant")
	assert.True(t, row.HasOverride)

	resp, content = platformCall(t, token, "DELETE", tenantURI+"/entitlements/"+featureKey, "")
	require.True(t, resp.IsSuccessful(), content)
	_, row = getTenantFeature(t, token, tenant.ID, featureKey)
	assert.True(t, row.Effective, "removing the override restores the plan value")
	assert.False(t, row.HasOverride)

	resp, content = platformCall(t, token, "DELETE", tenantURI+"/entitlements/"+featureKey, "")
	resp.AssertNotFound()
	assert.Contains(t, content, "entitlement_override_not_found")
}

func TestPlatformEntitlementErrors(t *testing.T) {
	withTenancyDriver(t, "database")
	token := loginPlatformSmoke(t)

	tenant := insertPendingTenant(t, fmt.Sprintf("enterr%d", time.Now().UnixNano()%1000000))
	cleanupTenantEntitlementRows(t, tenant.ID)

	resp, content := platformCall(t, token, "POST", fmt.Sprintf("/api/platform/tenants/%d/subscription", tenant.ID), `{"plan_code":"zz_no_such_plan"}`)
	resp.AssertNotFound()
	assert.Contains(t, content, "entitlement_plan_not_found")

	resp, content = platformCall(t, token, "PUT", fmt.Sprintf("/api/platform/tenants/%d/entitlements/zz_no_such.feature", tenant.ID), `{"enabled":true}`)
	resp.AssertNotFound()
	assert.Contains(t, content, "entitlement_feature_not_found")

	resp, content = platformCall(t, token, "GET", "/api/platform/tenants/999999999/entitlements", "")
	resp.AssertNotFound()
	assert.Contains(t, content, "tenant_not_found")

	resp, content = platformCall(t, token, "POST", "/api/platform/features", `{"key":"Bad Key!","type":"boolean"}`)
	resp.AssertStatus(400)
	assert.Contains(t, content, "invalid_argument")
}
