package platform

import (
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/str"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type AuthController struct{}

func NewAuthController() *AuthController {
	return &AuthController{}
}

type platformLoginBody struct {
	Username string `json:"username" form:"username"`
	Password string `json:"password" form:"password"`
}

// Login authenticates a platform admin on the platform DB.
func (c *AuthController) Login(ctx http.Context) http.Response {
	if !tenancy.Enabled() {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrTenancyDisabled.Code)
	}
	var body platformLoginBody
	_ = ctx.Request().Bind(&body)
	if strings.TrimSpace(body.Username) == "" || body.Password == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}

	var adminUser models.PlatformAdmin
	if err := appfacades.PlatformOrmQuery(ctx).Where("username", strings.TrimSpace(body.Username)).First(&adminUser); err != nil {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUsernameOrPasswordErr.Code)
	}
	if adminUser.Status != models.PlatformAdminStatusActive {
		return response.Error(ctx, http.StatusForbidden, apperrors.ErrAccountDisabled.Code)
	}
	if !facades.Hash().Check(body.Password, adminUser.Password) {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUsernameOrPasswordErr.Code)
	}

	ttl := facades.Config().GetInt("jwt.ttl", 60)
	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(time.Duration(ttl) * time.Minute)
		expiresAt = &t
	}
	ip := helpers.GetRealIP(ctx)
	plainToken, _, err := services.NewPlatformTokenService(ctx).CreateToken(
		models.TokenableTypePlatformAdmin,
		adminUser.ID,
		"platform-token",
		expiresAt,
		"",
		ip,
		"",
		"",
	)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_auth", http.StatusInternalServerError, err, nil)
	}

	return response.Success(ctx, map[string]any{
		"token": plainToken,
		"admin": services.PlatformAdminToJSON(&adminUser),
	})
}

// Info returns the current platform admin.
func (c *AuthController) Info(ctx http.Context) http.Response {
	adminUser, ok := ctx.Value("platform_admin").(models.PlatformAdmin)
	if !ok || adminUser.ID == 0 {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}
	return response.Success(ctx, map[string]any{
		"admin": services.PlatformAdminToJSON(&adminUser),
	})
}

// Logout deletes the current platform token.
func (c *AuthController) Logout(ctx http.Context) http.Response {
	token := ctx.Request().Header("Authorization", "")
	token = str.Of(token).ChopStart("Bearer ").Trim().String()
	if token != "" {
		_ = services.NewPlatformTokenService(ctx).DeleteToken(token)
	}
	return response.Success(ctx)
}
