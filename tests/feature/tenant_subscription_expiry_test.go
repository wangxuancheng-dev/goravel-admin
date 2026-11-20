package feature_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"goravel/app/entitlement"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// Plan expiry end to end: settings API -> reminder notice (once) -> downgrade to the
// default plan when the plan ends -> suspend policy disables the tenant.
func TestTenantSubscriptionExpiry(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	require.True(t, tenancy.Enabled())
	runQueueInline(t)
	platformToken := loginPlatformSmoke(t)

	// Restore the platform-wide policy afterwards.
	bg := context.Background()
	prevPolicy := services.GetSubscriptionExpiryPolicy(bg)
	t.Cleanup(func() { _ = services.SetSubscriptionExpiryPolicy(bg, prevPolicy) })

	suffix := fmt.Sprint(time.Now().UnixNano() % 1000000)
	tenantSvc := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	tn, err := tenantSvc.Create(services.TenantCreateInput{Code: "subx" + suffix, Name: "subx", Migrate: false})
	if err != nil {
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformQuery().Where("tenant_id", tn.ID).ForceDelete(&models.TenantSubscription{})
		_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	resp, content := platformCall(t, platformToken, "POST", fmt.Sprintf("/api/platform/tenants/%d/migrate", tn.ID), `{"with_seed":true}`)
	require.True(t, resp.IsSuccessful(), content)

	ent := services.NewEntitlementService(bg)
	planCode := "subx" + suffix
	plan, err := ent.UpsertPlan(services.UpsertPlanInput{Code: planCode, Name: "Expiry test", Status: 1})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = appfacades.PlatformQuery().Where("plan_id", plan.ID).ForceDelete(&models.PlatformPlanEntitlement{})
		_, _ = appfacades.PlatformQuery().Where("id", plan.ID).ForceDelete(&models.PlatformPlan{})
	})

	// Policy through the settings API: downgrade, no grace, remind 7/3/1 days before.
	resp, content = platformCall(t, platformToken, "PUT", "/api/platform/settings",
		`{"subscription_expiry":{"action":"downgrade","grace_days":0,"remind_days":[1,3,7],"notice_locale":"cn"}}`)
	require.True(t, resp.IsSuccessful(), content)
	assert.Contains(t, content, `"remind_days":[7,3,1]`)
	resp, content = platformCall(t, platformToken, "PUT", "/api/platform/settings",
		`{"subscription_expiry":{"action":"explode","grace_days":0,"remind_days":[],"notice_locale":"cn"}}`)
	assert.False(t, resp.IsSuccessful(), "unknown action must be rejected: %s", content)

	notices := func() int64 {
		n, countErr := appfacades.OrmQuery(tenantCtx(tn)).Model(&models.Notification{}).Where("type", "notice").Count()
		require.NoError(t, countErr)
		return n
	}
	adminCount, err := appfacades.OrmQuery(tenantCtx(tn)).Model(&models.Admin{}).Count()
	require.NoError(t, err)
	require.Positive(t, adminCount)

	// 1) Ends in two days: the 3-day reminder goes to every tenant admin, once.
	in2d := time.Now().Add(48 * time.Hour)
	sub, err := ent.AssignSubscription(tn.ID, services.AssignSubscriptionInput{PlanCode: planCode, EndsAt: &in2d})
	require.NoError(t, err)
	_, err = services.RunSubscriptionExpiry(time.Now(), tn.ID)
	require.NoError(t, err)
	var row models.TenantSubscription
	require.NoError(t, appfacades.PlatformQuery().Where("id", sub.ID).First(&row))
	require.NotNil(t, row.LastRemindDays)
	assert.Equal(t, 3, *row.LastRemindDays)
	assert.Equal(t, entitlement.SubActive, row.Status)
	assert.Equal(t, adminCount, notices(), "one notice per tenant admin")
	var notice models.Notification
	require.NoError(t, appfacades.OrmQuery(tenantCtx(tn)).Where("type", "notice").Order("id desc").First(&notice))
	assert.Equal(t, "套餐即将到期", notice.Title, "notice text comes from lang/cn.json")
	assert.Contains(t, notice.Content, "Expiry test")
	assert.Contains(t, notice.Content, "剩余约 2 天")
	_, err = services.RunSubscriptionExpiry(time.Now(), tn.ID)
	require.NoError(t, err)
	assert.Equal(t, adminCount, notices(), "the same reminder must not repeat")

	// 2) The plan ends: expired, entitlements fall back to the default plan.
	_, err = services.RunSubscriptionExpiry(in2d.Add(time.Minute), tn.ID)
	require.NoError(t, err)
	require.NoError(t, appfacades.PlatformQuery().Where("id", sub.ID).First(&row))
	assert.Equal(t, entitlement.SubExpired, row.Status)
	var snap models.TenantEntitlementSnapshot
	require.NoError(t, appfacades.PlatformQuery().Where("tenant_id", tn.ID).First(&snap))
	assert.NotEqual(t, planCode, snap.PlanCode, "snapshot must leave the expired plan")
	assert.Equal(t, 2*adminCount, notices(), "expiry notice sent")
	var tnRow models.Tenant
	require.NoError(t, appfacades.PlatformQuery().Where("id", tn.ID).First(&tnRow))
	assert.Equal(t, models.TenantStatusActive, tnRow.Status, "downgrade keeps the tenant enabled")

	// 3) Suspend policy: an ended trial disables the tenant.
	require.NoError(t, services.SetSubscriptionExpiryPolicy(bg, services.SubscriptionExpiryPolicy{
		Action: services.SubExpiryActionSuspend, GraceDays: 0, RemindDays: []int{1}, NoticeLocale: "en",
	}))
	trialEnd := time.Now().Add(-time.Hour)
	sub2, err := ent.AssignSubscription(tn.ID, services.AssignSubscriptionInput{PlanCode: planCode, TrialEndsAt: &trialEnd})
	require.NoError(t, err)
	report, err := services.RunSubscriptionExpiry(time.Now(), tn.ID)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, report.Suspended, 1, "report: %+v", report)
	require.NoError(t, appfacades.PlatformQuery().Where("id", sub2.ID).First(&row))
	assert.Equal(t, entitlement.SubExpired, row.Status)
	require.NoError(t, appfacades.PlatformQuery().Where("id", tn.ID).First(&tnRow))
	assert.Equal(t, models.TenantStatusDisabled, tnRow.Status, "suspend policy disables the tenant")
}
