package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	contractsorm "github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/entitlement"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/tenancy"
)

const (
	defaultPlanCode  = "free"
	snapshotCacheTTL = 30 * time.Second
)

// EntitlementService is the landlord-side facade for feature catalog, plans,
// subscriptions, overrides, snapshots, and runtime checks.
type EntitlementService struct {
	ctx context.Context

	cacheMu sync.RWMutex
	cache   map[uint]cachedEntitlementSnapshot
}

type cachedEntitlementSnapshot struct {
	set       *entitlement.EffectiveSet
	expiresAt time.Time
}

func NewEntitlementService(ctx context.Context) *EntitlementService {
	if ctx == nil {
		ctx = context.Background()
	}
	return &EntitlementService{ctx: ctx, cache: map[uint]cachedEntitlementSnapshot{}}
}

func NewEntitlementServiceFromHTTP(ctx http.Context) *EntitlementService {
	if ctx == nil {
		return NewEntitlementService(context.Background())
	}
	return NewEntitlementService(ctx.Context())
}

func (s *EntitlementService) q() contractsorm.Query {
	return appfacades.PlatformOrmQuery(s.ctx)
}

// EnsureBootstrap seeds the default free plan when missing (idempotent).
func (s *EntitlementService) EnsureBootstrap() error {
	count, err := s.q().Model(&models.PlatformPlan{}).Where("code", defaultPlanCode).Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	plan := models.PlatformPlan{
		Code:        defaultPlanCode,
		Name:        "Free",
		Description: "Default plan for entitlement-managed modules (no features on by default)",
		IsDefault:   true,
		IsPublic:    true,
		Status:      1,
		Sort:        0,
		Currency:    "CNY",
	}
	return s.q().Create(&plan)
}

// ---------- Catalog ----------

type UpsertFeatureInput struct {
	Key             string
	Name            string
	Description     string
	Type            string
	DefaultValue    string
	MenuSlugs    []string
	Channels     []string
	AlwaysOn     *bool
	Status       *uint8
	Sort         *int
}

// RegisterModuleFeature upserts a boolean module.* feature (codegen helper).
func (s *EntitlementService) RegisterModuleFeature(moduleName, displayName, menuSlug string, _ []string) (*models.PlatformFeature, error) {
	res, err := s.RegisterModule(RegisterModuleSpec{
		ModuleName:  moduleName,
		DisplayName: displayName,
		MenuSlug:    menuSlug,
	})
	if err != nil {
		return nil, err
	}
	return res.Feature, nil
}

func (s *EntitlementService) UpsertFeature(in UpsertFeatureInput) (*models.PlatformFeature, error) {
	if err := s.EnsureBootstrap(); err != nil {
		return nil, err
	}
	key := strings.ToLower(strings.TrimSpace(in.Key))
	if err := entitlement.ValidateFeatureKey(key); err != nil {
		return nil, apperrors.ErrInvalidArgument
	}
	featType := strings.ToLower(strings.TrimSpace(in.Type))
	if featType == "" {
		featType = entitlement.TypeBoolean
	}
	switch featType {
	case entitlement.TypeBoolean, entitlement.TypeLimit, entitlement.TypeConfig:
	default:
		return nil, apperrors.ErrInvalidArgument
	}

	var existing models.PlatformFeature
	err := s.q().Where("key", key).First(&existing)
	nowStatus := uint8(1)
	if in.Status != nil {
		nowStatus = *in.Status
	}
	sort := 0
	if in.Sort != nil {
		sort = *in.Sort
	}
	def := in.DefaultValue
	if def == "" {
		if featType == entitlement.TypeBoolean {
			def = "false"
		} else if featType == entitlement.TypeLimit {
			def = "0"
		}
	}

	if err != nil || existing.ID == 0 {
		f := models.PlatformFeature{
			Key:          key,
			Name:         strings.TrimSpace(in.Name),
			Description:  strings.TrimSpace(in.Description),
			Type:         featType,
			DefaultValue: def,
			Status:       nowStatus,
			Sort:         sort,
		}
		if in.AlwaysOn != nil {
			f.AlwaysOn = *in.AlwaysOn
		}
		if f.Name == "" {
			f.Name = key
		}
		f.SetMenuSlugs(in.MenuSlugs)
		f.SetChannels(in.Channels)
		if err := s.q().Create(&f); err != nil {
			return nil, err
		}
		s.afterCatalogChange()
		return &f, nil
	}

	existing.Name = entitlementFirstNonEmpty(strings.TrimSpace(in.Name), existing.Name)
	if strings.TrimSpace(in.Description) != "" {
		existing.Description = strings.TrimSpace(in.Description)
	}
	existing.Type = featType
	existing.DefaultValue = def
	if in.AlwaysOn != nil {
		existing.AlwaysOn = *in.AlwaysOn
	}
	if in.Status != nil {
		existing.Status = *in.Status
	}
	if in.Sort != nil {
		existing.Sort = *in.Sort
	}
	if in.MenuSlugs != nil {
		existing.SetMenuSlugs(in.MenuSlugs)
	}
	if in.Channels != nil {
		existing.SetChannels(in.Channels)
	}
	if err := s.q().Save(&existing); err != nil {
		return nil, err
	}
	s.afterCatalogChange()
	return &existing, nil
}

func (s *EntitlementService) ListFeatures() ([]models.PlatformFeature, error) {
	var list []models.PlatformFeature
	err := s.q().Model(&models.PlatformFeature{}).Order("sort asc").Order("id asc").Find(&list)
	return list, err
}

func (s *EntitlementService) GetFeature(key string) (*models.PlatformFeature, error) {
	var f models.PlatformFeature
	if err := s.q().Where("key", strings.ToLower(strings.TrimSpace(key))).First(&f); err != nil || f.ID == 0 {
		return nil, apperrors.ErrEntitlementFeatureNotFound
	}
	return &f, nil
}

// ---------- Plans ----------

type UpsertPlanInput struct {
	Code         string
	Name         string
	Description  string
	IsDefault    bool
	IsPublic     bool
	PriceMonthly int64
	PriceYearly  int64
	Currency     string
	Status       uint8
	Sort         int
	Entitlements map[string]string // feature_key -> value
}

func (s *EntitlementService) ListPlans() ([]models.PlatformPlan, error) {
	var list []models.PlatformPlan
	err := s.q().Model(&models.PlatformPlan{}).Order("sort asc").Order("id asc").Find(&list)
	return list, err
}

func (s *EntitlementService) UpsertPlan(in UpsertPlanInput) (*models.PlatformPlan, error) {
	if err := s.EnsureBootstrap(); err != nil {
		return nil, err
	}
	code := strings.ToLower(strings.TrimSpace(in.Code))
	if code == "" {
		return nil, apperrors.ErrInvalidArgument
	}
	var plan models.PlatformPlan
	err := s.q().Where("code", code).First(&plan)
	currency := strings.TrimSpace(in.Currency)
	if currency == "" {
		currency = "CNY"
	}
	status := in.Status
	if status == 0 && err != nil {
		status = 1
	}

	if err != nil || plan.ID == 0 {
		plan = models.PlatformPlan{
			Code:         code,
			Name:         entitlementFirstNonEmpty(strings.TrimSpace(in.Name), code),
			Description:  strings.TrimSpace(in.Description),
			IsDefault:    in.IsDefault,
			IsPublic:     in.IsPublic,
			PriceMonthly: in.PriceMonthly,
			PriceYearly:  in.PriceYearly,
			Currency:     currency,
			Status:       status,
			Sort:         in.Sort,
		}
		if plan.Status == 0 {
			plan.Status = 1
		}
		if err := s.q().Create(&plan); err != nil {
			return nil, err
		}
	} else {
		plan.Name = entitlementFirstNonEmpty(strings.TrimSpace(in.Name), plan.Name)
		plan.Description = strings.TrimSpace(in.Description)
		plan.IsDefault = in.IsDefault
		plan.IsPublic = in.IsPublic
		plan.PriceMonthly = in.PriceMonthly
		plan.PriceYearly = in.PriceYearly
		plan.Currency = currency
		if in.Status != 0 {
			plan.Status = in.Status
		}
		plan.Sort = in.Sort
		if err := s.q().Save(&plan); err != nil {
			return nil, err
		}
	}

	if in.IsDefault {
		_, _ = s.q().Model(&models.PlatformPlan{}).Where("id", "!=", plan.ID).Update("is_default", false)
	}

	if in.Entitlements != nil {
		if err := s.replacePlanEntitlements(plan.ID, in.Entitlements); err != nil {
			return nil, err
		}
		_ = s.RecomputeAllForPlan(plan.ID)
	}
	return &plan, nil
}

func (s *EntitlementService) replacePlanEntitlements(planID uint, grants map[string]string) error {
	_, _ = s.q().Model(&models.PlatformPlanEntitlement{}).Where("plan_id", planID).Delete()
	for key, value := range grants {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			continue
		}
		row := models.PlatformPlanEntitlement{
			PlanID:     planID,
			FeatureKey: key,
			Value:      strings.TrimSpace(value),
		}
		if err := s.q().Create(&row); err != nil {
			return err
		}
	}
	return nil
}

func (s *EntitlementService) ListPlanEntitlements(planID uint) ([]models.PlatformPlanEntitlement, error) {
	var list []models.PlatformPlanEntitlement
	err := s.q().Model(&models.PlatformPlanEntitlement{}).Where("plan_id", planID).Find(&list)
	return list, err
}

// ---------- Subscriptions / overrides ----------

type AssignSubscriptionInput struct {
	PlanCode     string
	BillingCycle string
	EndsAt       *time.Time
	TrialEndsAt  *time.Time
	Note         string
	AdminID      uint
}

func (s *EntitlementService) AssignSubscription(tenantID uint, in AssignSubscriptionInput) (*models.TenantSubscription, error) {
	if tenantID == 0 {
		return nil, apperrors.ErrTenantNotFound
	}
	if err := s.EnsureBootstrap(); err != nil {
		return nil, err
	}
	if _, err := s.requireTenant(tenantID); err != nil {
		return nil, err
	}
	code := strings.ToLower(strings.TrimSpace(in.PlanCode))
	if code == "" {
		code = defaultPlanCode
	}
	var plan models.PlatformPlan
	if err := s.q().Where("code", code).Where("status", 1).First(&plan); err != nil || plan.ID == 0 {
		return nil, apperrors.ErrEntitlementPlanNotFound
	}

	// cancel previous active
	_, _ = s.q().Model(&models.TenantSubscription{}).
		Where("tenant_id", tenantID).
		Where("status", entitlement.SubActive).
		Update("status", entitlement.SubCanceled)

	now := time.Now()
	cycle := strings.TrimSpace(in.BillingCycle)
	if cycle == "" {
		cycle = "manual"
	}
	sub := models.TenantSubscription{
		TenantID:     tenantID,
		PlanID:       plan.ID,
		Status:       entitlement.SubActive,
		BillingCycle: cycle,
		StartsAt:     &now,
		EndsAt:       in.EndsAt,
		TrialEndsAt:  in.TrialEndsAt,
		Note:         strings.TrimSpace(in.Note),
	}
	if err := s.q().Create(&sub); err != nil {
		return nil, err
	}
	_ = s.audit(in.AdminID, tenantID, "subscription.assign", fmt.Sprintf("plan=%s sub=%d", plan.Code, sub.ID))
	if _, err := s.Recompute(tenantID); err != nil {
		return nil, err
	}
	return &sub, nil
}

type SetOverrideInput struct {
	FeatureKey string
	Enabled    *bool
	Value      string
	Pinned     *bool
	ExpiresAt  *time.Time
	Reason     string
	AdminID    uint
}

func (s *EntitlementService) SetOverride(tenantID uint, in SetOverrideInput) (*models.TenantEntitlementOverride, error) {
	if tenantID == 0 {
		return nil, apperrors.ErrTenantNotFound
	}
	key := strings.ToLower(strings.TrimSpace(in.FeatureKey))
	feat, err := s.GetFeature(key)
	if err != nil {
		return nil, err
	}
	value := strings.TrimSpace(in.Value)
	if in.Enabled != nil {
		value = entitlement.FormatBoolValue(*in.Enabled)
	}
	if value == "" {
		return nil, apperrors.ErrInvalidArgument
	}

	var row models.TenantEntitlementOverride
	findErr := s.q().Where("tenant_id", tenantID).Where("feature_key", key).First(&row)
	pinned := true
	if in.Pinned != nil {
		pinned = *in.Pinned
	}
	if findErr != nil || row.ID == 0 {
		row = models.TenantEntitlementOverride{
			TenantID:   tenantID,
			FeatureKey: key,
			Value:      value,
			Pinned:     pinned,
			ExpiresAt:  in.ExpiresAt,
			Reason:     strings.TrimSpace(in.Reason),
			CreatedBy:  in.AdminID,
		}
		if err := s.q().Create(&row); err != nil {
			return nil, err
		}
	} else {
		row.Value = value
		row.Pinned = pinned
		row.ExpiresAt = in.ExpiresAt
		if strings.TrimSpace(in.Reason) != "" {
			row.Reason = strings.TrimSpace(in.Reason)
		}
		if err := s.q().Save(&row); err != nil {
			return nil, err
		}
	}
	_ = feat
	_ = s.audit(in.AdminID, tenantID, "override.set", fmt.Sprintf("%s=%s", key, value))
	if _, err := s.Recompute(tenantID); err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *EntitlementService) DeleteOverride(tenantID uint, featureKey string, adminID uint) error {
	key := strings.ToLower(strings.TrimSpace(featureKey))
	res, err := s.q().Model(&models.TenantEntitlementOverride{}).
		Where("tenant_id", tenantID).
		Where("feature_key", key).
		Delete()
	if err != nil {
		return err
	}
	if res == nil || res.RowsAffected == 0 {
		return apperrors.ErrEntitlementOverrideNotFound
	}
	_ = s.audit(adminID, tenantID, "override.delete", key)
	_, err = s.Recompute(tenantID)
	return err
}

// ---------- Snapshot / runtime ----------

func (s *EntitlementService) Recompute(tenantID uint) (*entitlement.EffectiveSet, error) {
	if tenantID == 0 {
		return nil, apperrors.ErrTenantNotFound
	}
	if err := s.EnsureBootstrap(); err != nil {
		return nil, err
	}

	catalogRows, err := s.ListFeatures()
	if err != nil {
		return nil, err
	}
	catalog := make([]entitlement.FeatureMeta, 0, len(catalogRows))
	for _, f := range catalogRows {
		if f.Status == 0 {
			continue
		}
		catalog = append(catalog, entitlement.FeatureMeta{
			Key:          f.Key,
			Type:         f.Type,
			DefaultValue: f.DefaultValue,
			MenuSlugs:    f.MenuSlugs(),
			Channels:     f.Channels(),
			AlwaysOn:     f.AlwaysOn,
			Status:       f.Status,
		})
	}

	planCode := ""
	planGrants := map[string]entitlement.Grant{}
	var sub models.TenantSubscription
	if err := s.q().Where("tenant_id", tenantID).Where("status", entitlement.SubActive).
		Order("id desc").First(&sub); err == nil && sub.ID > 0 {
		var plan models.PlatformPlan
		if err := s.q().Where("id", sub.PlanID).First(&plan); err == nil && plan.ID > 0 {
			planCode = plan.Code
			var rows []models.PlatformPlanEntitlement
			_ = s.q().Where("plan_id", plan.ID).Find(&rows)
			for _, r := range rows {
				planGrants[r.FeatureKey] = entitlement.Grant{Key: r.FeatureKey, Value: r.Value, Source: entitlement.SourcePlan}
			}
		}
	} else {
		// no subscription: use default plan grants if any
		var plan models.PlatformPlan
		if err := s.q().Where("is_default", true).Where("status", 1).First(&plan); err == nil && plan.ID > 0 {
			planCode = plan.Code
			var rows []models.PlatformPlanEntitlement
			_ = s.q().Where("plan_id", plan.ID).Find(&rows)
			for _, r := range rows {
				planGrants[r.FeatureKey] = entitlement.Grant{Key: r.FeatureKey, Value: r.Value, Source: entitlement.SourcePlan}
			}
		}
	}

	overrides := map[string]entitlement.Grant{}
	var ovs []models.TenantEntitlementOverride
	_ = s.q().Where("tenant_id", tenantID).Find(&ovs)
	for _, o := range ovs {
		overrides[o.FeatureKey] = entitlement.Grant{
			Key:       o.FeatureKey,
			Value:     o.Value,
			Source:    entitlement.SourceOverride,
			ExpiresAt: o.ExpiresAt,
		}
	}

	set := entitlement.Resolve(entitlement.MergeInput{
		TenantID:  tenantID,
		PlanCode:  planCode,
		Catalog:   catalog,
		Plan:      planGrants,
		Overrides: overrides,
		Now:       time.Now(),
	})

	var prev models.TenantEntitlementSnapshot
	version := int64(1)
	if err := s.q().Where("tenant_id", tenantID).First(&prev); err == nil && prev.ID > 0 {
		version = prev.Version + 1
	}
	set.Version = version

	payload, err := json.Marshal(set)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if prev.ID > 0 {
		prev.Version = version
		prev.PlanCode = planCode
		prev.Payload = string(payload)
		prev.ComputedAt = now
		if err := s.q().Save(&prev); err != nil {
			return nil, err
		}
	} else {
		snap := models.TenantEntitlementSnapshot{
			TenantID:   tenantID,
			Version:    version,
			PlanCode:   planCode,
			Payload:    string(payload),
			ComputedAt: now,
		}
		if err := s.q().Create(&snap); err != nil {
			return nil, err
		}
	}
	s.enrichEffectiveSet(set)
	s.putCache(tenantID, set)
	return set, nil
}

func (s *EntitlementService) RecomputeAllForPlan(planID uint) error {
	var subs []models.TenantSubscription
	_ = s.q().Where("plan_id", planID).Where("status", entitlement.SubActive).Find(&subs)
	for _, sub := range subs {
		if _, err := s.Recompute(sub.TenantID); err != nil {
			return err
		}
	}
	return nil
}

func (s *EntitlementService) LoadEffective(tenantID uint) (*entitlement.EffectiveSet, error) {
	if tenantID == 0 {
		return entitlement.NewEffectiveSet(0), nil
	}
	if set := s.getCache(tenantID); set != nil {
		s.enrichEffectiveSet(set)
		return set, nil
	}
	var snap models.TenantEntitlementSnapshot
	if err := s.q().Where("tenant_id", tenantID).First(&snap); err == nil && snap.ID > 0 && strings.TrimSpace(snap.Payload) != "" {
		var set entitlement.EffectiveSet
		if err := json.Unmarshal([]byte(snap.Payload), &set); err == nil {
			set.Version = snap.Version
			s.enrichEffectiveSet(&set)
			s.putCache(tenantID, &set)
			return &set, nil
		}
	}
	return s.Recompute(tenantID)
}

// enrichEffectiveSet applies live catalog rules on top of a stored snapshot:
// always_on, deny-by-default for newly registered keys, menu slug index, channels.
func (s *EntitlementService) enrichEffectiveSet(set *entitlement.EffectiveSet) {
	if set == nil {
		return
	}
	if set.Features == nil {
		set.Features = map[string]bool{}
	}
	if set.Limits == nil {
		set.Limits = map[string]int64{}
	}
	if set.Configs == nil {
		set.Configs = map[string]string{}
	}
	if set.MenuSlugEnabled == nil {
		set.MenuSlugEnabled = map[string]bool{}
	}
	if set.FeatureChannels == nil {
		set.FeatureChannels = map[string][]string{}
	}

	features, err := s.ListFeatures()
	if err != nil {
		return
	}
	for _, f := range features {
		if f.Status == 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(f.Key))
		set.FeatureChannels[key] = f.Channels()
		switch f.Type {
		case entitlement.TypeBoolean:
			if f.AlwaysOn {
				set.Features[key] = true
			} else if _, ok := set.Features[key]; !ok {
				set.Features[key] = entitlement.ParseBoolValue(f.DefaultValue)
			}
			for _, slug := range f.MenuSlugs() {
				slug = strings.ToLower(strings.TrimSpace(slug))
				if slug != "" {
					set.MenuSlugEnabled[slug] = set.Features[key]
				}
			}
		case entitlement.TypeLimit:
			if _, ok := set.Limits[key]; !ok {
				set.Limits[key] = entitlement.ParseLimitValue(f.DefaultValue)
			}
		case entitlement.TypeConfig:
			if _, ok := set.Configs[key]; !ok {
				set.Configs[key] = f.DefaultValue
			}
		}
	}
}

// afterCatalogChange drops caches and recomputes snapshots after catalog edits.
func (s *EntitlementService) afterCatalogChange() {
	s.invalidateAllCache()
	_ = s.RecomputeAllSnapshots()
}

// RecomputeAllSnapshots refreshes every stored tenant snapshot (and active subscriptions).
func (s *EntitlementService) RecomputeAllSnapshots() error {
	seen := map[uint]bool{}
	var snaps []models.TenantEntitlementSnapshot
	_ = s.q().Model(&models.TenantEntitlementSnapshot{}).Find(&snaps)
	for _, snap := range snaps {
		if snap.TenantID == 0 || seen[snap.TenantID] {
			continue
		}
		seen[snap.TenantID] = true
		if _, err := s.Recompute(snap.TenantID); err != nil {
			return err
		}
	}
	var subs []models.TenantSubscription
	_ = s.q().Where("status", entitlement.SubActive).Find(&subs)
	for _, sub := range subs {
		if sub.TenantID == 0 || seen[sub.TenantID] {
			continue
		}
		seen[sub.TenantID] = true
		if _, err := s.Recompute(sub.TenantID); err != nil {
			return err
		}
	}
	return nil
}

// CanForTenant checks a boolean feature for a tenant id.
func (s *EntitlementService) CanForTenant(tenantID uint, featureKey string) (bool, error) {
	set, err := s.LoadEffective(tenantID)
	if err != nil {
		return false, err
	}
	return set.Can(featureKey), nil
}

// CanOnChannelForTenant checks whether the feature is enabled for the tenant.
func (s *EntitlementService) CanOnChannelForTenant(tenantID uint, featureKey, channel string) (bool, error) {
	set, err := s.LoadEffective(tenantID)
	if err != nil {
		return false, err
	}
	return set.CanOnChannel(featureKey, channel), nil
}

// RuntimeCan resolves tenant from request context.
// When tenancy is off, entitlement gating is skipped (returns true) so local single-DB keeps working.
func (s *EntitlementService) RuntimeCan(ctx http.Context, featureKey string, channel string) (bool, error) {
	if !tenancy.Enabled() {
		return true, nil
	}
	tenantID, ok := helpers.GetTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return false, nil
	}
	if strings.TrimSpace(channel) == "" {
		channel = entitlement.ChannelAdmin
	}
	return s.CanOnChannelForTenant(tenantID, featureKey, channel)
}

// ClientViewForTenant builds the client bootstrap payload.
func (s *EntitlementService) ClientViewForTenant(tenantID uint, channel string) (entitlement.ClientView, error) {
	set, err := s.LoadEffective(tenantID)
	if err != nil {
		return entitlement.ClientView{}, err
	}
	usage, _ := s.loadUsageMap(tenantID)
	return entitlement.Present(set, channel, usage), nil
}

// TenantEntitlementDTO is platform UI aggregate for one tenant.
type TenantEntitlementDTO struct {
	Subscription *models.TenantSubscription         `json:"subscription"`
	Plan         *models.PlatformPlan               `json:"plan"`
	Overrides    []models.TenantEntitlementOverride `json:"overrides"`
	Effective    entitlement.ClientView             `json:"effective"`
	Catalog      []TenantFeatureRow                 `json:"catalog"`
}

type TenantFeatureRow struct {
	Key           string   `json:"key"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	AlwaysOn      bool     `json:"always_on"`
	MenuSlugs     []string `json:"menu_slugs"`
	Effective     bool     `json:"effective"`
	OverrideValue string   `json:"override_value,omitempty"`
	HasOverride   bool     `json:"has_override"`
}

func (s *EntitlementService) GetTenantEntitlements(tenantID uint) (*TenantEntitlementDTO, error) {
	if _, err := s.requireTenant(tenantID); err != nil {
		return nil, err
	}
	set, err := s.LoadEffective(tenantID)
	if err != nil {
		return nil, err
	}
	view := entitlement.Present(set, entitlement.ChannelAdmin, nil)

	dto := &TenantEntitlementDTO{Effective: view}

	var sub models.TenantSubscription
	if err := s.q().Where("tenant_id", tenantID).Where("status", entitlement.SubActive).
		Order("id desc").First(&sub); err == nil && sub.ID > 0 {
		dto.Subscription = &sub
		var plan models.PlatformPlan
		if err := s.q().Where("id", sub.PlanID).First(&plan); err == nil && plan.ID > 0 {
			dto.Plan = &plan
		}
	}

	var ovs []models.TenantEntitlementOverride
	_ = s.q().Where("tenant_id", tenantID).Find(&ovs)
	dto.Overrides = ovs
	ovMap := map[string]models.TenantEntitlementOverride{}
	for _, o := range ovs {
		ovMap[o.FeatureKey] = o
	}

	features, err := s.ListFeatures()
	if err != nil {
		return nil, err
	}
	rows := make([]TenantFeatureRow, 0, len(features))
	for _, f := range features {
		row := TenantFeatureRow{
			Key:       f.Key,
			Name:      f.Name,
			Type:      f.Type,
			AlwaysOn:  f.AlwaysOn,
			MenuSlugs: f.MenuSlugs(),
			Effective: set.Can(f.Key),
		}
		if o, ok := ovMap[f.Key]; ok {
			row.HasOverride = true
			row.OverrideValue = o.Value
		}
		rows = append(rows, row)
	}
	dto.Catalog = rows
	return dto, nil
}

// DisabledMenuSlugsForTenant returns entitlement-managed slugs that should be hidden.
// Uses the live catalog so newly registered modules are denied until explicitly enabled
// (snapshot-only MenuSlugEnabled would miss new keys and incorrectly show menus).
func (s *EntitlementService) DisabledMenuSlugsForTenant(tenantID uint) map[string]bool {
	out := map[string]bool{}
	if !tenancy.Enabled() || tenantID == 0 {
		return out
	}
	set, err := s.LoadEffective(tenantID)
	if err != nil || set == nil {
		return out
	}
	features, err := s.ListFeatures()
	if err != nil {
		return set.DisabledMenuSlugs()
	}
	for _, f := range features {
		if f.Status == 0 || f.Type != entitlement.TypeBoolean {
			continue
		}
		enabled := set.Can(f.Key)
		for _, slug := range f.MenuSlugs() {
			slug = strings.ToLower(strings.TrimSpace(slug))
			if slug != "" && !enabled {
				out[slug] = true
			}
		}
	}
	return out
}

func (s *EntitlementService) loadUsageMap(tenantID uint) (map[string]int64, error) {
	var rows []models.TenantUsageCounter
	if err := s.q().Where("tenant_id", tenantID).Find(&rows); err != nil {
		return nil, err
	}
	out := map[string]int64{}
	for _, r := range rows {
		out[r.FeatureKey] = r.Used
	}
	return out, nil
}

func (s *EntitlementService) requireTenant(tenantID uint) (*models.Tenant, error) {
	var t models.Tenant
	if err := s.q().Where("id", tenantID).First(&t); err != nil || t.ID == 0 {
		return nil, apperrors.ErrTenantNotFound
	}
	return &t, nil
}

func (s *EntitlementService) audit(adminID, tenantID uint, action, detail string) error {
	row := models.PlatformEntitlementAuditLog{
		AdminID:  adminID,
		TenantID: tenantID,
		Action:   action,
		Detail:   detail,
	}
	return s.q().Create(&row)
}

func (s *EntitlementService) getCache(tenantID uint) *entitlement.EffectiveSet {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	item, ok := s.cache[tenantID]
	if !ok || time.Now().After(item.expiresAt) {
		return nil
	}
	return item.set
}

func (s *EntitlementService) putCache(tenantID uint, set *entitlement.EffectiveSet) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache[tenantID] = cachedEntitlementSnapshot{set: set, expiresAt: time.Now().Add(snapshotCacheTTL)}
}

func (s *EntitlementService) invalidateAllCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.cache = map[uint]cachedEntitlementSnapshot{}
}

func entitlementFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// EntitlementDisabledMenuSlugs is a package helper for menu filtering (admin channel).
func EntitlementDisabledMenuSlugs(ctx http.Context) map[string]bool {
	if !tenancy.Enabled() {
		return map[string]bool{}
	}
	tenantID, ok := helpers.GetTenantIDFromContext(ctx)
	if !ok || tenantID == 0 {
		return map[string]bool{}
	}
	return NewEntitlementServiceFromHTTP(ctx).DisabledMenuSlugsForTenant(tenantID)
}

// FilterMenusByEntitlement removes flat menus gated by entitlement-managed features only.
func FilterMenusByEntitlement(ctx http.Context, menus []models.Menu) []models.Menu {
	disabled := EntitlementDisabledMenuSlugs(ctx)
	if len(disabled) == 0 {
		return menus
	}
	hiddenIDs := map[uint]bool{}
	for _, menu := range menus {
		if disabled[strings.ToLower(strings.TrimSpace(menu.Slug))] {
			hiddenIDs[menu.ID] = true
		}
	}
	changed := true
	for changed {
		changed = false
		for _, menu := range menus {
			if hiddenIDs[menu.ParentID] && !hiddenIDs[menu.ID] {
				hiddenIDs[menu.ID] = true
				changed = true
			}
		}
	}
	out := make([]models.Menu, 0, len(menus))
	for _, menu := range menus {
		if !hiddenIDs[menu.ID] {
			out = append(out, menu)
		}
	}
	return out
}

// FilterTreeMenusByEntitlement recursively hides entitlement-managed menus that are not enabled.
// Must be applied on /info menu trees after FilterTreeMenusByModule — rebuilding the tree from DB
// otherwise brings closed modules back via permission MenuIDs.
func FilterTreeMenusByEntitlement(ctx http.Context, menus []models.Menu) []models.Menu {
	disabled := EntitlementDisabledMenuSlugs(ctx)
	if len(disabled) == 0 {
		return menus
	}
	filtered := make([]models.Menu, 0, len(menus))
	for _, menu := range menus {
		if disabled[strings.ToLower(strings.TrimSpace(menu.Slug))] {
			continue
		}
		if len(menu.Children) > 0 {
			menu.Children = FilterTreeMenusByEntitlement(ctx, menu.Children)
		}
		filtered = append(filtered, menu)
	}
	return filtered
}

// EnsureTenantHasDefaultSubscription assigns free plan when tenant has none.
func (s *EntitlementService) EnsureTenantHasDefaultSubscription(tenantID uint) error {
	count, _ := s.q().Model(&models.TenantSubscription{}).
		Where("tenant_id", tenantID).
		Where("status", entitlement.SubActive).
		Count()
	if count > 0 {
		return nil
	}
	_, err := s.AssignSubscription(tenantID, AssignSubscriptionInput{PlanCode: defaultPlanCode})
	return err
}
