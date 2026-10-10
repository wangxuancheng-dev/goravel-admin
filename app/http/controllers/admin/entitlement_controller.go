package admin

import (
	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// EntitlementController exposes the tenant-facing, read-only plan summary.
type EntitlementController struct{}

func NewEntitlementController() *EntitlementController {
	return &EntitlementController{}
}

// Me returns the current tenant's plan, subscription window, features and quotas.
// When tenancy is off it returns enabled=false so the UI can hide the page.
func (c *EntitlementController) Me(ctx http.Context) http.Response {
	if !tenancy.Enabled() {
		return response.Success(ctx, map[string]any{"enabled": false})
	}
	tenantID, ok := helpers.GetTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return response.Success(ctx, map[string]any{"enabled": false})
	}
	summary, err := services.NewEntitlementServiceFromHTTP(ctx).TenantPlanSummaryFor(tenantID)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}
	return response.Success(ctx, map[string]any{
		"enabled":      true,
		"version":      summary.Version,
		"plan":         summary.Plan,
		"subscription": summary.Subscription,
		"features":     summary.Features,
		"limits":       summary.Limits,
	})
}
