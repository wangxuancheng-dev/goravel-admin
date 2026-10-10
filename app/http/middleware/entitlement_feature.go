package middleware

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/entitlement"
	"goravel/app/http/response"
	"goravel/app/services"
)

// EntitlementFeature gates a platform_features key by landlord entitlements.
//
// When tenancy is off, the check passes so single-DB installs keep working.
func EntitlementFeature(featureKey string, channel ...string) http.Middleware {
	key := strings.ToLower(strings.TrimSpace(featureKey))
	ch := entitlement.ChannelAdmin
	if len(channel) > 0 && strings.TrimSpace(channel[0]) != "" {
		if n := entitlement.NormalizeChannel(channel[0]); n != "" {
			ch = n
		}
	}
	return newMiddleware("entitlement_feature_"+key, func(ctx http.Context) {
		ok, err := services.NewEntitlementServiceFromHTTP(ctx).RuntimeCan(ctx, key, ch)
		if err != nil {
			response.Abort(ctx, http.StatusInternalServerError, err)
			return
		}
		if !ok {
			response.Abort(ctx, http.StatusForbidden, apperrors.ErrTenantFeatureDisabled.Code)
			return
		}
		ctx.Request().Next()
	})
}

// EntitlementModule is sugar for module.{name} feature keys.
func EntitlementModule(moduleName string, channel ...string) http.Middleware {
	return EntitlementFeature(entitlement.ModuleFeatureKey(moduleName), channel...)
}
