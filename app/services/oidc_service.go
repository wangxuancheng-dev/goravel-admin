package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"golang.org/x/oauth2"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/utils"
)

// oidcStateCachePrefix is intentionally global (not tenancy.CacheKey).
// The IdP callback hits a shared API host with no tenant hint; the state payload
// carries tenant_id so the callback can BindHTTP before admin lookup.
const oidcStateCachePrefix = "oidc:state:"

type oidcStatePayload struct {
	TenantID   uint   `json:"tenant_id,omitempty"`
	TenantCode string `json:"tenant_code,omitempty"`
}

// oidcRuntimeSettings merges process env (oidc.*) with tenant configs group "oidc".
// redirect_url / frontend_redirect stay global (single registered callback URL).
type oidcRuntimeSettings struct {
	Enabled       bool
	Issuer        string
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	Scopes        string
	ButtonLabel   string
	AutoProvision bool
}

type OIDCService interface {
	Enabled(ctx context.Context) bool
	PublicInfo(ctx context.Context) map[string]any
	AuthURL(ctx http.Context) (string, error)
	HandleCallback(ctx http.Context, code, state string) (plainToken string, admin *models.Admin, tenantCode string, err error)
}

type OIDCServiceImpl struct {
	authService AuthService
}

func NewOIDCService(authService AuthService) OIDCService {
	return &OIDCServiceImpl{authService: authService}
}

func (s *OIDCServiceImpl) resolveSettings(ctx context.Context) oidcRuntimeSettings {
	cfg := oidcRuntimeSettings{
		Enabled:       facades.Config().GetBool("oidc.enabled", false),
		Issuer:        strings.TrimSpace(facades.Config().GetString("oidc.issuer", "")),
		ClientID:      strings.TrimSpace(facades.Config().GetString("oidc.client_id", "")),
		ClientSecret:  strings.TrimSpace(facades.Config().GetString("oidc.client_secret", "")),
		RedirectURL:   strings.TrimSpace(facades.Config().GetString("oidc.redirect_url", "")),
		Scopes:        strings.TrimSpace(facades.Config().GetString("oidc.scopes", "openid profile email")),
		ButtonLabel:   strings.TrimSpace(facades.Config().GetString("oidc.button_label", "Enterprise SSO")),
		AutoProvision: facades.Config().GetBool("oidc.auto_provision", false),
	}
	m := utils.GetConfigGroupMap(ctx, "oidc")
	if v, ok := m["enabled"]; ok && strings.TrimSpace(v) != "" {
		cfg.Enabled = utils.ParseConfigBool(v)
	}
	if v := strings.TrimSpace(m["issuer"]); v != "" {
		cfg.Issuer = v
	}
	if v := strings.TrimSpace(m["client_id"]); v != "" {
		cfg.ClientID = v
	}
	if v := strings.TrimSpace(m["client_secret"]); v != "" {
		cfg.ClientSecret = v
	}
	if v := strings.TrimSpace(m["scopes"]); v != "" {
		cfg.Scopes = v
	}
	if v := strings.TrimSpace(m["button_label"]); v != "" {
		cfg.ButtonLabel = v
	}
	if v, ok := m["auto_provision"]; ok && strings.TrimSpace(v) != "" {
		cfg.AutoProvision = utils.ParseConfigBool(v)
	}
	if cfg.ButtonLabel == "" {
		cfg.ButtonLabel = "Enterprise SSO"
	}
	if cfg.Scopes == "" {
		cfg.Scopes = "openid profile email"
	}
	return cfg
}

func (s *OIDCServiceImpl) settingsReady(cfg oidcRuntimeSettings) bool {
	return cfg.Enabled && cfg.Issuer != "" && cfg.ClientID != "" && cfg.RedirectURL != ""
}

func (s *OIDCServiceImpl) Enabled(ctx context.Context) bool {
	return s.settingsReady(s.resolveSettings(ctx))
}

func (s *OIDCServiceImpl) PublicInfo(ctx context.Context) map[string]any {
	cfg := s.resolveSettings(ctx)
	enabled := s.settingsReady(cfg)
	info := map[string]any{
		"enabled": enabled,
	}
	if enabled {
		info["button_label"] = cfg.ButtonLabel
		info["redirect_path"] = "/api/admin/auth/oidc/redirect"
	}
	return info
}

func (s *OIDCServiceImpl) provider(ctx context.Context, cfg oidcRuntimeSettings) (*oidc.Provider, *oauth2.Config, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc provider: %w", err)
	}
	var scopes []string
	for _, part := range strings.Fields(cfg.Scopes) {
		if part != "" {
			scopes = append(scopes, part)
		}
	}
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       scopes,
	}
	return provider, oauthCfg, nil
}

func (s *OIDCServiceImpl) AuthURL(ctx http.Context) (string, error) {
	reqCtx := context.Background()
	if ctx != nil && ctx.Context() != nil {
		reqCtx = ctx.Context()
	}
	// Tenant ORM routing keys live on http.Context (same as ConfigService).
	cfg := s.resolveSettings(ctx)
	if !s.settingsReady(cfg) {
		return "", apperrors.NewBusinessError("oidc_disabled", "OIDC SSO is disabled")
	}
	_, oauthCfg, err := s.provider(reqCtx, cfg)
	if err != nil {
		return "", err
	}
	state, err := randomState(24)
	if err != nil {
		return "", err
	}

	payload := oidcStatePayload{}
	if tenancy.Enabled() {
		id, ok := helpers.GetTenantIDFromContext(ctx)
		if !ok || id == 0 {
			return "", apperrors.ErrTenantRequired
		}
		payload.TenantID = id
		if code, ok := helpers.GetTenantCodeFromContext(ctx); ok {
			payload.TenantCode = code
		}
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode oidc state: %w", err)
	}
	if err := facades.Cache().Put(oidcStateCachePrefix+state, string(raw), 10*time.Minute); err != nil {
		return "", fmt.Errorf("store oidc state: %w", err)
	}
	return oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOffline), nil
}

func (s *OIDCServiceImpl) HandleCallback(ctx http.Context, code, state string) (string, *models.Admin, string, error) {
	code = strings.TrimSpace(code)
	state = strings.TrimSpace(state)
	if code == "" || state == "" {
		return "", nil, "", apperrors.ErrParamsError
	}
	var cached string
	if err := facades.Cache().Get(oidcStateCachePrefix+state, &cached); err != nil || cached == "" {
		return "", nil, "", apperrors.NewBusinessError("oidc_invalid_state", "invalid or expired OIDC state")
	}
	_ = facades.Cache().Forget(oidcStateCachePrefix + state)

	payload, err := parseOIDCStatePayload(cached)
	if err != nil {
		return "", nil, "", apperrors.NewBusinessError("oidc_invalid_state", "invalid or expired OIDC state")
	}
	tenantCode := strings.TrimSpace(payload.TenantCode)

	if tenancy.Enabled() {
		if payload.TenantID == 0 {
			return "", nil, tenantCode, apperrors.ErrTenantRequired
		}
		if err := NewTenantConnectionService().BindHTTPByTenantID(ctx, payload.TenantID); err != nil {
			return "", nil, tenantCode, err
		}
	}
	if tenantCode == "" {
		if code, ok := helpers.GetTenantCodeFromContext(ctx); ok {
			tenantCode = code
		}
	}

	reqCtx := ctx.Context()
	if reqCtx == nil {
		reqCtx = context.Background()
	}
	if err := EnsureClientIPAllowed(ctx, helpers.GetRealIP(ctx)); err != nil {
		return "", nil, tenantCode, err
	}

	cfg := s.resolveSettings(ctx)
	if !s.settingsReady(cfg) {
		return "", nil, tenantCode, apperrors.NewBusinessError("oidc_disabled", "OIDC SSO is disabled")
	}
	provider, oauthCfg, err := s.provider(reqCtx, cfg)
	if err != nil {
		return "", nil, tenantCode, err
	}
	token, err := oauthCfg.Exchange(reqCtx, code)
	if err != nil {
		return "", nil, tenantCode, apperrors.NewBusinessError("oidc_exchange_failed", "OIDC code exchange failed").WithError(err)
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok || rawIDToken == "" {
		return "", nil, tenantCode, apperrors.NewBusinessError("oidc_id_token_missing", "OIDC id_token missing")
	}
	verifier := provider.Verifier(&oidc.Config{ClientID: oauthCfg.ClientID})
	idToken, err := verifier.Verify(reqCtx, rawIDToken)
	if err != nil {
		return "", nil, tenantCode, apperrors.NewBusinessError("oidc_id_token_invalid", "OIDC id_token invalid").WithError(err)
	}

	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		PreferredUsername string `json:"preferred_username"`
		Name              string `json:"name"`
		Sub               string `json:"sub"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return "", nil, tenantCode, apperrors.NewBusinessError("oidc_claims_invalid", "OIDC claims invalid").WithError(err)
	}
	email := strings.TrimSpace(strings.ToLower(claims.Email))
	if email == "" {
		return "", nil, tenantCode, apperrors.NewBusinessError("oidc_email_required", "OIDC email claim required")
	}

	admin, err := s.findOrProvisionAdmin(ctx, cfg, email, claims.PreferredUsername, claims.Name, claims.Sub)
	if err != nil {
		return "", nil, tenantCode, err
	}
	if admin.Status != 1 {
		return "", nil, tenantCode, apperrors.ErrAccountDisabled
	}

	plainToken, err := s.authService.IssueAdminToken(ctx, admin.ID)
	if err != nil {
		return "", nil, tenantCode, err
	}

	reqDump, _ := json.Marshal(map[string]any{
		"provider": "oidc",
		"email":    email,
		"sub":      claims.Sub,
	})
	_ = s.authService.RecordLoginLog(ctx, admin.ID, admin.Username, 1, "login_success_oidc", string(reqDump))

	return plainToken, admin, tenantCode, nil
}

func parseOIDCStatePayload(cached string) (oidcStatePayload, error) {
	cached = strings.TrimSpace(cached)
	var payload oidcStatePayload
	if cached == "" || cached == "1" {
		return payload, nil
	}
	if err := json.Unmarshal([]byte(cached), &payload); err != nil {
		return payload, err
	}
	return payload, nil
}

func (s *OIDCServiceImpl) findOrProvisionAdmin(ctx context.Context, cfg oidcRuntimeSettings, email, preferredUsername, name, sub string) (*models.Admin, error) {
	var admin models.Admin
	err := appfacades.OrmQuery(ctx).Where("email", email).First(&admin)
	if err == nil && admin.ID > 0 {
		return &admin, nil
	}

	if !cfg.AutoProvision {
		return nil, apperrors.NewBusinessError("oidc_user_not_linked", "no admin linked to this OIDC email")
	}

	username := strings.TrimSpace(preferredUsername)
	if username == "" {
		username = strings.Split(email, "@")[0]
	}
	username = sanitizeUsername(username)
	if username == "" {
		username = "oidc_" + sanitizeUsername(sub)
	}

	var existing models.Admin
	if err := appfacades.OrmQuery(ctx).Where("username", username).First(&existing); err == nil && existing.ID > 0 {
		username = fmt.Sprintf("%s_%s", username, randomShort())
	}

	nickname := strings.TrimSpace(name)
	if nickname == "" {
		nickname = username
	}
	passwordHash, hashErr := facades.Hash().Make(randomStateOr("oidc-disabled-password"))
	if hashErr != nil {
		return nil, apperrors.ErrCreateFailed.WithError(hashErr)
	}
	admin = models.Admin{
		Username: username,
		Password: passwordHash,
		Nickname: nickname,
		Email:    email,
		Status:   0,
	}
	if err := appfacades.OrmQuery(ctx).Create(&admin); err != nil {
		return nil, apperrors.ErrCreateFailed.WithError(err)
	}
	return &admin, nil
}

func sanitizeUsername(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}

func randomState(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func randomStateOr(fallback string) string {
	s, err := randomState(16)
	if err != nil {
		return fallback
	}
	return s
}

func randomShort() string {
	s, err := randomState(4)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().Unix()%10000)
	}
	if len(s) > 6 {
		return s[:6]
	}
	return s
}
