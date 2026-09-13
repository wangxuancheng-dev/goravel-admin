package middleware

import (
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
	"goravel/app/utils/logger"
)

// PlatformJwt authenticates platform console users against the platform DB.
// Does not run Tenant middleware — never switches to a tenant connection.
func PlatformJwt() http.Middleware {
	return newMiddleware("platform_jwt", func(ctx http.Context) {
		if !tenancy.Enabled() {
			response.Abort(ctx, http.StatusBadRequest, apperrors.ErrTenancyDisabled.Code)
			return
		}

		token := ctx.Request().Header("Authorization", "")
		if str.Of(token).IsEmpty() {
			token = ctx.Request().Query("_token", "")
		}
		if str.Of(token).IsEmpty() {
			response.Abort(ctx, http.StatusUnauthorized, "not_logged_in")
			return
		}
		token = str.Of(token).ChopStart("Bearer ").Trim().String()
		if token == "" {
			response.Abort(ctx, http.StatusUnauthorized, "not_logged_in")
			return
		}

		tokenService := services.NewPlatformTokenService(ctx)
		accessToken, err := tokenService.FindToken(token)
		if err != nil {
			tokenPrefix := token[:min(20, len(token))]
			if isExpectedUnauthorizedErr(err) {
				logger.WarnfHTTP(ctx, "Platform JWT: unauthorized token: %v, prefix: %s", err, tokenPrefix)
			} else {
				logger.ErrorfHTTP(ctx, "Platform JWT: FindToken error: %v, prefix: %s", err, tokenPrefix)
			}
			response.Abort(ctx, http.StatusUnauthorized, "invalid_token")
			return
		}
		if accessToken == nil || accessToken.TokenableType != models.TokenableTypePlatformAdmin {
			response.Abort(ctx, http.StatusUnauthorized, "invalid_token")
			return
		}

		var admin models.PlatformAdmin
		if err := appfacades.PlatformOrmQuery(ctx).Where("id", accessToken.TokenableID).First(&admin); err != nil {
			response.Abort(ctx, http.StatusUnauthorized, "user_not_found")
			return
		}
		if admin.Status != models.PlatformAdminStatusActive {
			response.Abort(ctx, http.StatusForbidden, apperrors.ErrAccountDisabled.Code)
			return
		}

		_ = tokenService.UpdateLastUsedAt(token)
		if accessToken.ExpiresAt != nil {
			ttl := facades.Config().GetInt("jwt.ttl", 60)
			if ttl > 0 {
				newExpiresAt := time.Now().Add(time.Duration(ttl) * time.Minute)
				_, _ = appfacades.PlatformOrmQuery(ctx).
					Model(&models.PersonalAccessToken{}).
					Where("id", accessToken.ID).
					Update("expires_at", newExpiresAt)
			}
		}

		ctx.WithValue("platform_admin", admin)
		ctx.WithValue("platform_token", accessToken)
		ctx.Request().Next()
	})
}

// PlatformOwner requires an owner-role platform admin (blocks viewers from write ops).
func PlatformOwner() http.Middleware {
	return newMiddleware("platform_owner", func(ctx http.Context) {
		admin, ok := ctx.Value("platform_admin").(models.PlatformAdmin)
		if !ok || admin.ID == 0 {
			response.Abort(ctx, http.StatusUnauthorized, "not_logged_in")
			return
		}
		if !admin.IsOwner() {
			response.Abort(ctx, http.StatusForbidden, apperrors.ErrPlatformReadonly.Code)
			return
		}
		ctx.Request().Next()
	})
}

// RequireTenancy aborts when TENANCY_DRIVER is not database (for public platform login probe).
func RequireTenancy() http.Middleware {
	return newMiddleware("require_tenancy", func(ctx http.Context) {
		if !tenancy.Enabled() {
			response.Abort(ctx, http.StatusBadRequest, apperrors.ErrTenancyDisabled.Code)
			return
		}
		ctx.Request().Next()
	})
}
