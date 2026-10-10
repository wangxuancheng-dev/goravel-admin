package platform

import (
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/entitlement"
	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type EntitlementController struct{}

func NewEntitlementController() *EntitlementController {
	return &EntitlementController{}
}

func (c *EntitlementController) svc(ctx http.Context) *services.EntitlementService {
	return services.NewEntitlementServiceFromHTTP(ctx)
}

func (c *EntitlementController) requireTenancy(ctx http.Context) http.Response {
	if !tenancy.Enabled() {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrTenancyDisabled.Code)
	}
	return nil
}

func (c *EntitlementController) platformAdminID(ctx http.Context) uint {
	if adminUser, ok := ctx.Value("platform_admin").(models.PlatformAdmin); ok {
		return adminUser.ID
	}
	return 0
}

// ---------- Features ----------

func (c *EntitlementController) FeatureIndex(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	list, err := c.svc(ctx).ListFeatures()
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}
	out := make([]map[string]any, 0, len(list))
	for _, f := range list {
		out = append(out, featureToJSON(f))
	}
	return response.Success(ctx, map[string]any{"list": out})
}

func (c *EntitlementController) FeatureStore(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	var body featureBody
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	f, err := c.svc(ctx).UpsertFeature(body.toInput())
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, map[string]any{"feature": featureToJSON(*f)})
}

// RegisterModule registers a generated or hand-written module into the catalog.
func (c *EntitlementController) RegisterModule(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	var body struct {
		ModuleName  string `json:"module_name"`
		DisplayName string `json:"display_name"`
		MenuSlug    string `json:"menu_slug"`
		AlwaysOn    bool   `json:"always_on"`
		RowQuota    bool   `json:"row_quota"`
	}
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	res, err := c.svc(ctx).RegisterModule(services.RegisterModuleSpec{
		ModuleName:  body.ModuleName,
		DisplayName: body.DisplayName,
		MenuSlug:    body.MenuSlug,
		AlwaysOn:    body.AlwaysOn,
		RowQuota:    body.RowQuota,
		AdminID:     c.platformAdminID(ctx),
	})
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, res)
}

func (c *EntitlementController) FeatureUpdate(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	key := strings.TrimSpace(ctx.Request().Route("key"))
	var body featureBody
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	body.Key = key
	f, err := c.svc(ctx).UpsertFeature(body.toInput())
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, map[string]any{"feature": featureToJSON(*f)})
}

// ---------- Plans ----------

func (c *EntitlementController) PlanIndex(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	list, err := c.svc(ctx).ListPlans()
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}
	return response.Success(ctx, map[string]any{"list": list})
}

func (c *EntitlementController) PlanStore(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	var body planBody
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	plan, err := c.svc(ctx).UpsertPlan(body.toInput())
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, map[string]any{"plan": plan})
}

func (c *EntitlementController) PlanUpdate(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	code := strings.TrimSpace(ctx.Request().Route("code"))
	var body planBody
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	body.Code = code
	plan, err := c.svc(ctx).UpsertPlan(body.toInput())
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, map[string]any{"plan": plan})
}

func (c *EntitlementController) PlanEntitlements(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	code := strings.TrimSpace(ctx.Request().Route("code"))
	plans, err := c.svc(ctx).ListPlans()
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}
	var planID uint
	for _, p := range plans {
		if p.Code == code {
			planID = p.ID
			break
		}
	}
	if planID == 0 {
		return response.Error(ctx, http.StatusNotFound, apperrors.ErrEntitlementPlanNotFound.Code)
	}
	rows, err := c.svc(ctx).ListPlanEntitlements(planID)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err)
	}
	return response.Success(ctx, map[string]any{"list": rows})
}

// ---------- Tenant entitlements ----------

func (c *EntitlementController) TenantIndex(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	id := ctx.Request().Route("id")
	tenantID := parseUintParam(id)
	if tenantID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	dto, err := c.svc(ctx).GetTenantEntitlements(tenantID)
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, dto)
}

func (c *EntitlementController) TenantAssignPlan(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	tenantID := parseUintParam(ctx.Request().Route("id"))
	if tenantID == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	var body struct {
		PlanCode     string  `json:"plan_code"`
		BillingCycle string  `json:"billing_cycle"`
		EndsAt       *string `json:"ends_at"`
		Note         string  `json:"note"`
	}
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	var endsAt *time.Time
	if body.EndsAt != nil && strings.TrimSpace(*body.EndsAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.EndsAt))
		if err != nil {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
		}
		endsAt = &t
	}
	sub, err := c.svc(ctx).AssignSubscription(tenantID, services.AssignSubscriptionInput{
		PlanCode:     body.PlanCode,
		BillingCycle: body.BillingCycle,
		EndsAt:       endsAt,
		Note:         body.Note,
		AdminID:      c.platformAdminID(ctx),
	})
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, map[string]any{"subscription": sub})
}

func (c *EntitlementController) TenantSetOverride(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	tenantID := parseUintParam(ctx.Request().Route("id"))
	key := strings.TrimSpace(ctx.Request().Route("key"))
	if tenantID == 0 || key == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	var body struct {
		Enabled   *bool   `json:"enabled"`
		Value     string  `json:"value"`
		Pinned    *bool   `json:"pinned"`
		ExpiresAt *string `json:"expires_at"`
		Reason    string  `json:"reason"`
	}
	if err := ctx.Request().Bind(&body); err != nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
	}
	var expires *time.Time
	if body.ExpiresAt != nil && strings.TrimSpace(*body.ExpiresAt) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*body.ExpiresAt))
		if err != nil {
			return response.Error(ctx, http.StatusBadRequest, apperrors.ErrParamsError.Code)
		}
		expires = &t
	}
	row, err := c.svc(ctx).SetOverride(tenantID, services.SetOverrideInput{
		FeatureKey: key,
		Enabled:    body.Enabled,
		Value:      body.Value,
		Pinned:     body.Pinned,
		ExpiresAt:  expires,
		Reason:     body.Reason,
		AdminID:    c.platformAdminID(ctx),
	})
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, map[string]any{"override": row})
}

func (c *EntitlementController) TenantDeleteOverride(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	tenantID := parseUintParam(ctx.Request().Route("id"))
	key := strings.TrimSpace(ctx.Request().Route("key"))
	if err := c.svc(ctx).DeleteOverride(tenantID, key, c.platformAdminID(ctx)); err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, "delete_success", http.Json{})
}

func (c *EntitlementController) TenantRecompute(ctx http.Context) http.Response {
	if resp := c.requireTenancy(ctx); resp != nil {
		return resp
	}
	tenantID := parseUintParam(ctx.Request().Route("id"))
	set, err := c.svc(ctx).Recompute(tenantID)
	if err != nil {
		return mapEntitlementError(ctx, err)
	}
	return response.Success(ctx, map[string]any{
		"effective": entitlement.Present(set, entitlement.ChannelAdmin, nil),
	})
}

// ---------- helpers ----------

type featureBody struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`
	DefaultValue string   `json:"default_value"`
	MenuSlugs    []string `json:"menu_slugs"`
	Channels     []string `json:"channels"`
	AlwaysOn     *bool    `json:"always_on"`
	Status       *uint8   `json:"status"`
	Sort         *int     `json:"sort"`
}

func (b featureBody) toInput() services.UpsertFeatureInput {
	return services.UpsertFeatureInput{
		Key:          b.Key,
		Name:         b.Name,
		Description:  b.Description,
		Type:         b.Type,
		DefaultValue: b.DefaultValue,
		MenuSlugs:    b.MenuSlugs,
		Channels:     b.Channels,
		AlwaysOn:     b.AlwaysOn,
		Status:       b.Status,
		Sort:         b.Sort,
	}
}

type planBody struct {
	Code         string            `json:"code"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	IsDefault    bool              `json:"is_default"`
	IsPublic     bool              `json:"is_public"`
	PriceMonthly int64             `json:"price_monthly"`
	PriceYearly  int64             `json:"price_yearly"`
	Currency     string            `json:"currency"`
	Status       uint8             `json:"status"`
	Sort         int               `json:"sort"`
	Entitlements map[string]string `json:"entitlements"`
}

func (b planBody) toInput() services.UpsertPlanInput {
	return services.UpsertPlanInput{
		Code:         b.Code,
		Name:         b.Name,
		Description:  b.Description,
		IsDefault:    b.IsDefault,
		IsPublic:     b.IsPublic,
		PriceMonthly: b.PriceMonthly,
		PriceYearly:  b.PriceYearly,
		Currency:     b.Currency,
		Status:       b.Status,
		Sort:         b.Sort,
		Entitlements: b.Entitlements,
	}
}

func featureToJSON(f models.PlatformFeature) map[string]any {
	return map[string]any{
		"id":            f.ID,
		"key":           f.Key,
		"name":          f.Name,
		"description":   f.Description,
		"type":          f.Type,
		"default_value": f.DefaultValue,
		"menu_slugs":    f.MenuSlugs(),
		"always_on":     f.AlwaysOn,
		"status":        f.Status,
		"sort":          f.Sort,
	}
}

func mapEntitlementError(ctx http.Context, err error) http.Response {
	if be, ok := err.(*apperrors.BusinessError); ok {
		switch be.Code {
		case apperrors.ErrEntitlementFeatureNotFound.Code,
			apperrors.ErrEntitlementPlanNotFound.Code,
			apperrors.ErrEntitlementOverrideNotFound.Code,
			apperrors.ErrTenantNotFound.Code:
			return response.Error(ctx, http.StatusNotFound, be.Code)
		case apperrors.ErrEntitlementLimitExceeded.Code:
			return response.Error(ctx, http.StatusUnprocessableEntity, be.Code)
		case apperrors.ErrInvalidArgument.Code, apperrors.ErrParamsError.Code:
			return response.Error(ctx, http.StatusBadRequest, be.Code)
		default:
			return response.Error(ctx, http.StatusBadRequest, be.Code)
		}
	}
	return response.Error(ctx, http.StatusInternalServerError, err)
}

func parseUintParam(raw string) uint {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	var n uint64
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + uint64(ch-'0')
	}
	return uint(n)
}
