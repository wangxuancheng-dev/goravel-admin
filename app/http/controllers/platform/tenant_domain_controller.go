package platform

import (
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
	"goravel/app/tenancy"
)

// Domains lists vanity domains for a tenant.
func (c *TenantController) Domains(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	if _, err := c.service().GetByIDIncludingTrashed(id); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	list, err := services.NewTenantDomainService().ListByTenant(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant_domain", http.StatusInternalServerError, err, map[string]any{"tenant_id": id})
	}
	rows := make([]map[string]any, 0, len(list))
	for i := range list {
		rows = append(rows, services.TenantDomainToJSON(&list[i]))
	}
	return response.Success(ctx, map[string]any{"list": rows})
}

type tenantDomainStoreBody struct {
	Host      string `json:"host" form:"host"`
	SSLMode   string `json:"ssl_mode" form:"ssl_mode"`
	IsPrimary bool   `json:"is_primary" form:"is_primary"`
}

// StoreDomain adds a pending vanity domain.
func (c *TenantController) StoreDomain(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantDomainStoreBody
	_ = ctx.Request().Bind(&body)
	row, err := services.NewTenantDomainService().Create(id, services.TenantDomainCreateInput{
		Host:      body.Host,
		SSLMode:   body.SSLMode,
		IsPrimary: body.IsPrimary,
	})
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant_domain", http.StatusBadRequest, err, map[string]any{"tenant_id": id})
	}
	return response.Success(ctx, map[string]any{"domain": services.TenantDomainToJSON(row)})
}

// VerifyDomain runs DNS checks and activates on success.
func (c *TenantController) VerifyDomain(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	domainID := helpers.GetUintRoute(ctx, "domainId")
	if id == 0 || domainID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	row, err := services.NewTenantDomainService().Verify(id, domainID)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant_domain", http.StatusBadRequest, err, map[string]any{
			"tenant_id": id, "domain_id": domainID,
		})
	}
	return response.Success(ctx, map[string]any{"domain": services.TenantDomainToJSON(row)})
}

// SetPrimaryDomain marks an active domain as primary login host.
func (c *TenantController) SetPrimaryDomain(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	domainID := helpers.GetUintRoute(ctx, "domainId")
	if id == 0 || domainID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	row, err := services.NewTenantDomainService().SetPrimary(id, domainID)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant_domain", http.StatusBadRequest, err, map[string]any{
			"tenant_id": id, "domain_id": domainID,
		})
	}
	return response.Success(ctx, map[string]any{"domain": services.TenantDomainToJSON(row)})
}

// DisableDomain stops Host resolution for a vanity domain.
func (c *TenantController) DisableDomain(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	domainID := helpers.GetUintRoute(ctx, "domainId")
	if id == 0 || domainID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	row, err := services.NewTenantDomainService().Disable(id, domainID)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant_domain", http.StatusBadRequest, err, map[string]any{
			"tenant_id": id, "domain_id": domainID,
		})
	}
	return response.Success(ctx, map[string]any{"domain": services.TenantDomainToJSON(row)})
}

// DestroyDomain soft-deletes a vanity domain row.
func (c *TenantController) DestroyDomain(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	domainID := helpers.GetUintRoute(ctx, "domainId")
	if id == 0 || domainID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	if err := services.NewTenantDomainService().Delete(id, domainID); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant_domain", http.StatusBadRequest, err, map[string]any{
			"tenant_id": id, "domain_id": domainID,
		})
	}
	return response.Success(ctx)
}

// TLSAllow is used by edge on-demand TLS (e.g. Caddy ask). No auth.
// GET /api/platform/public/tls-allow?host=crm.example.com
// HTTP 200 + allow=true to issue; HTTP 403 to deny (Caddy ask convention).
func (c *TenantController) TLSAllow(ctx http.Context) http.Response {
	host := strings.TrimSpace(ctx.Request().Query("host", ""))
	if host == "" {
		host = tenancy.RequestHost(ctx.Request().Host(), ctx.Request().Header("X-Forwarded-Host", ""))
	}
	host = tenancy.NormalizeHost(host)
	allow := services.NewTenantDomainService().TLSAllow(host)
	payload := http.Json{
		"allow": allow,
		"host":  host,
	}
	if !allow {
		return ctx.Response().Status(http.StatusForbidden).Json(http.Json{
			"code":    http.StatusForbidden,
			"message": "tls_not_allowed",
			"data":    payload,
		})
	}
	return response.Success(ctx, payload)
}
