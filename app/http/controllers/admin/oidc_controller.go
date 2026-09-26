package admin

import (
	nethttp "net/http"
	"net/url"
	"strings"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
	"goravel/app/services"
)

type OIDCController struct{}

func NewOIDCController() *OIDCController {
	return &OIDCController{}
}

func (c *OIDCController) oidcService(ctx http.Context) services.OIDCService {
	adminService := services.NewAdminServiceImpl(ctx)
	tokenService := services.NewTokenServiceImpl(ctx)
	authService := services.NewAuthServiceImpl(ctx, adminService, tokenService)
	return services.NewOIDCService(authService)
}

// Config returns public OIDC SSO flags for the login page.
func (c *OIDCController) Config(ctx http.Context) http.Response {
	return response.Success(ctx, http.Json{
		"oidc": c.oidcService(ctx).PublicInfo(),
	})
}

// Redirect starts the OIDC authorization code flow.
// Requires Tenant middleware when tenancy is on so state can embed tenant_id.
func (c *OIDCController) Redirect(ctx http.Context) http.Response {
	authURL, err := c.oidcService(ctx).AuthURL(ctx)
	if err != nil {
		return HandleGeneratedServiceError(ctx, "oidc", nethttp.StatusBadRequest, err, nil)
	}
	return ctx.Response().Redirect(nethttp.StatusFound, authURL)
}

// Callback finishes OIDC login and redirects to the SPA with a token query.
// Registered without Tenant middleware: tenant is restored from OIDC state.
func (c *OIDCController) Callback(ctx http.Context) http.Response {
	code := strings.TrimSpace(ctx.Request().Query("code", ""))
	state := strings.TrimSpace(ctx.Request().Query("state", ""))
	if errMsg := strings.TrimSpace(ctx.Request().Query("error", "")); errMsg != "" {
		return c.redirectFrontend(ctx, "", "", errMsg)
	}

	token, _, tenantCode, err := c.oidcService(ctx).HandleCallback(ctx, code, state)
	if err != nil {
		codeKey := apperrors.ErrParamsError.Code
		if be, ok := err.(*apperrors.BusinessError); ok {
			codeKey = be.Code
		}
		return c.redirectFrontend(ctx, "", tenantCode, codeKey)
	}
	return c.redirectFrontend(ctx, token, tenantCode, "")
}

func (c *OIDCController) redirectFrontend(ctx http.Context, token, tenantCode, errCode string) http.Response {
	base := strings.TrimSpace(facades.Config().GetString("oidc.frontend_redirect", ""))
	if base == "" {
		base = "/login/oidc-callback"
	}
	u, err := url.Parse(base)
	if err != nil {
		u = &url.URL{Path: "/login/oidc-callback"}
	}
	q := u.Query()
	if token != "" {
		q.Set("token", token)
	}
	if errCode != "" {
		q.Set("error", errCode)
	}
	if code := strings.TrimSpace(tenantCode); code != "" {
		q.Set("tenant_code", code)
	}
	u.RawQuery = q.Encode()
	return ctx.Response().Redirect(nethttp.StatusFound, u.String())
}
