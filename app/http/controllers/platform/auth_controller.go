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
	Username      string `json:"username" form:"username"`
	Password      string `json:"password" form:"password"`
	CaptchaID     string `json:"captcha_id" form:"captcha_id"`
	CaptchaAnswer string `json:"captcha_answer" form:"captcha_answer"`
	GoogleCode    string `json:"google_code" form:"google_code"`
}

// Captcha always issues a login captcha for the platform console (forced).
func (c *AuthController) Captcha(ctx http.Context) http.Response {
	captchaID, image, err := services.NewCaptchaServiceImpl(ctx).Generate()
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_captcha", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{
		"captcha": map[string]any{
			"enabled":       true,
			"required":      true,
			"captcha_id":    captchaID,
			"captcha_image": image,
		},
	})
}

// Login authenticates a platform admin on the platform DB.
func (c *AuthController) Login(ctx http.Context) http.Response {
	if !tenancy.Enabled() {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrTenancyDisabled.Code)
	}
	var body platformLoginBody
	_ = ctx.Request().Bind(&body)
	username := strings.TrimSpace(body.Username)
	if username == "" || body.Password == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}

	if ok, messageKey := services.NewCaptchaServiceImpl(ctx).Verify(body.CaptchaID, body.CaptchaAnswer); !ok {
		if messageKey == "" {
			messageKey = "captcha_invalid"
		}
		services.RecordPlatformLoginLog(ctx, 0, username, 0, messageKey)
		return response.Error(ctx, http.StatusBadRequest, messageKey)
	}

	var adminUser models.PlatformAdmin
	if err := appfacades.PlatformOrmQuery(ctx).Where("username", username).First(&adminUser); err != nil {
		services.RecordPlatformLoginLog(ctx, 0, username, 0, "username_not_found")
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUsernameOrPasswordErr.Code)
	}
	if adminUser.Status != models.PlatformAdminStatusActive {
		services.RecordPlatformLoginLog(ctx, adminUser.ID, username, 0, "account_disabled")
		return response.Error(ctx, http.StatusForbidden, apperrors.ErrAccountDisabled.Code)
	}
	if !facades.Hash().Check(body.Password, adminUser.Password) {
		services.RecordPlatformLoginLog(ctx, adminUser.ID, username, 0, "password_error")
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrUsernameOrPasswordErr.Code)
	}

	clientIP := helpers.GetRealIP(ctx)
	if !services.IPInAllowlist(clientIP, adminUser.AllowedIPs) {
		services.RecordPlatformLoginLog(ctx, adminUser.ID, username, 0, "login_ip_not_allowed")
		return response.Error(ctx, http.StatusForbidden, apperrors.ErrLoginIPNotAllowed.Code)
	}

	if adminUser.Is2FABound() {
		code := strings.TrimSpace(body.GoogleCode)
		if code == "" {
			services.RecordPlatformLoginLog(ctx, adminUser.ID, username, 0, "google_code_required")
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeRequired.Code)
		}
		ga := services.NewGoogleAuthenticatorServiceImpl(ctx)
		if !ga.Verify(adminUser.GoogleSecret, code) {
			services.RecordPlatformLoginLog(ctx, adminUser.ID, username, 0, "google_code_invalid")
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeInvalid.Code)
		}
	}

	ttl := facades.Config().GetInt("jwt.ttl", 60)
	var expiresAt *time.Time
	if ttl > 0 {
		t := time.Now().Add(time.Duration(ttl) * time.Minute)
		expiresAt = &t
	}
	ip := clientIP
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

	services.RecordPlatformLoginLog(ctx, adminUser.ID, username, 1, "login_success")

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
	if adminUser, ok := ctx.Value("platform_admin").(models.PlatformAdmin); ok && adminUser.ID > 0 {
		services.RecordPlatformLoginLog(ctx, adminUser.ID, adminUser.Username, 1, "logout_success")
	}
	token := ctx.Request().Header("Authorization", "")
	token = str.Of(token).ChopStart("Bearer ").Trim().String()
	if token != "" {
		_ = services.NewPlatformTokenService(ctx).DeleteToken(token)
	}
	return response.Success(ctx)
}

type platformBind2FABody struct {
	Secret string `json:"secret" form:"secret"`
	Code   string `json:"code" form:"code"`
}

type platformUnbind2FABody struct {
	Code string `json:"code" form:"code"`
}

type platformAllowedIPsBody struct {
	AllowedIPs string `json:"allowed_ips" form:"allowed_ips"`
}

func currentPlatformAdmin(ctx http.Context) (models.PlatformAdmin, bool) {
	adminUser, ok := ctx.Value("platform_admin").(models.PlatformAdmin)
	return adminUser, ok && adminUser.ID > 0
}

// GoogleAuthenticatorStatus returns whether current platform admin has 2FA bound.
func (c *AuthController) GoogleAuthenticatorStatus(ctx http.Context) http.Response {
	adminUser, ok := currentPlatformAdmin(ctx)
	if !ok {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}
	return response.Success(ctx, map[string]any{
		"bound":       adminUser.Is2FABound(),
		"allowed_ips": adminUser.AllowedIPs,
	})
}

// GoogleAuthenticatorQRCode generates a TOTP secret + QR for binding.
func (c *AuthController) GoogleAuthenticatorQRCode(ctx http.Context) http.Response {
	adminUser, ok := currentPlatformAdmin(ctx)
	if !ok {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}
	if adminUser.Is2FABound() {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleAuthenticatorAlreadyBound.Code)
	}
	ga := services.NewGoogleAuthenticatorServiceImpl(ctx)
	secret, _, err := ga.GenerateSecret("platform:" + adminUser.Username)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_2fa", http.StatusInternalServerError, err, nil)
	}
	qr, err := ga.GenerateQRCodeImage("platform:"+adminUser.Username, secret)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_2fa", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{
		"secret":  secret,
		"qr_code": qr,
	})
}

// BindGoogleAuthenticator binds TOTP for the current platform admin.
func (c *AuthController) BindGoogleAuthenticator(ctx http.Context) http.Response {
	adminUser, ok := currentPlatformAdmin(ctx)
	if !ok {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}
	if adminUser.Is2FABound() {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleAuthenticatorAlreadyBound.Code)
	}
	var body platformBind2FABody
	_ = ctx.Request().Bind(&body)
	secret := strings.TrimSpace(body.Secret)
	code := strings.TrimSpace(body.Code)
	if secret == "" || code == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	ga := services.NewGoogleAuthenticatorServiceImpl(ctx)
	if !ga.Verify(secret, code) {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeInvalid.Code)
	}
	_, err := appfacades.PlatformOrmQuery(ctx).Model(&adminUser).Update("google_secret", secret)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_2fa", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, "bind_success")
}

// UnbindGoogleAuthenticator removes TOTP after verifying a code.
func (c *AuthController) UnbindGoogleAuthenticator(ctx http.Context) http.Response {
	adminUser, ok := currentPlatformAdmin(ctx)
	if !ok {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}
	if !adminUser.Is2FABound() {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleAuthenticatorNotBound.Code)
	}
	var body platformUnbind2FABody
	_ = ctx.Request().Bind(&body)
	code := strings.TrimSpace(body.Code)
	if code == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeRequired.Code)
	}
	ga := services.NewGoogleAuthenticatorServiceImpl(ctx)
	if !ga.Verify(adminUser.GoogleSecret, code) {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrGoogleCodeInvalid.Code)
	}
	_, err := appfacades.PlatformOrmQuery(ctx).Model(&adminUser).Update("google_secret", nil)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_2fa", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, "unbind_success")
}

// UpdateAllowedIPs updates the current platform admin login IP allowlist.
func (c *AuthController) UpdateAllowedIPs(ctx http.Context) http.Response {
	adminUser, ok := currentPlatformAdmin(ctx)
	if !ok {
		return response.Error(ctx, http.StatusUnauthorized, apperrors.ErrNotLoggedIn.Code)
	}
	var body platformAllowedIPsBody
	_ = ctx.Request().Bind(&body)
	ips := strings.TrimSpace(body.AllowedIPs)
	_, err := appfacades.PlatformOrmQuery(ctx).Model(&adminUser).Update("allowed_ips", ips)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "platform_security", http.StatusInternalServerError, err, nil)
	}
	adminUser.AllowedIPs = ips
	return response.Success(ctx, map[string]any{"admin": services.PlatformAdminToJSON(&adminUser)})
}
