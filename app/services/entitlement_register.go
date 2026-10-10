package services

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/entitlement"
	"goravel/app/models"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// RegisterModuleSpec describes a generated or hand-written module for entitlements.
//
// Modes:
//   - AlwaysOn=true: available to every tenant; plan/override cannot turn it off (no quota unless RowQuota).
//   - RowQuota=true: also registers quota.module.{name}.rows; Create should call CheckAndConsumeModuleRows.
//   - neither: boolean feature gated by plan/override only.
type RegisterModuleSpec struct {
	ModuleName  string
	DisplayName string
	MenuSlug    string
	AlwaysOn    bool
	RowQuota    bool
	AdminID     uint
}

// RegisterModuleResult is returned after catalog upsert.
type RegisterModuleResult struct {
	FeatureKey   string                  `json:"feature_key"`
	LimitKey     string                  `json:"limit_key,omitempty"`
	Feature      *models.PlatformFeature `json:"feature"`
	LimitFeature *models.PlatformFeature `json:"limit_feature,omitempty"`
}

// RegisterCapabilitySpec registers a boolean capability under an existing module
// (e.g. module.member.export). menu_slugs stay empty so menus are not affected.
type RegisterCapabilitySpec struct {
	ModuleName  string
	Capability  string
	DisplayName string
	AlwaysOn    bool
	AdminID     uint
}

// RegisterModule upserts catalog entries for hand-written or generated modules.
// It does not enable the module for any tenant (unless AlwaysOn).
func (s *EntitlementService) RegisterModule(spec RegisterModuleSpec) (*RegisterModuleResult, error) {
	name := entitlement.NormalizeModuleName(spec.ModuleName)
	if !entitlement.ValidModuleSegment(name) {
		return nil, apperrors.ErrInvalidArgument
	}
	display := strings.TrimSpace(spec.DisplayName)
	if display == "" {
		display = name
	}
	menuSlug := strings.TrimSpace(spec.MenuSlug)
	if menuSlug == "" {
		menuSlug = name // default: menu slug == module name (codegen convention)
	}
	slugs := []string{menuSlug}

	def := "false"
	if spec.AlwaysOn {
		def = "true"
	}
	feat, err := s.UpsertFeature(UpsertFeatureInput{
		Key:          entitlement.ModuleFeatureKey(name),
		Name:         display,
		Description:  "Module feature (generated or hand-written)",
		Type:         entitlement.TypeBoolean,
		DefaultValue: def,
		MenuSlugs:    slugs,
		AlwaysOn:     &spec.AlwaysOn,
	})
	if err != nil {
		return nil, err
	}

	out := &RegisterModuleResult{
		FeatureKey: feat.Key,
		Feature:    feat,
	}

	if spec.RowQuota {
		limitKey := entitlement.ModuleRowsLimitKey(name)
		limitFeat, err := s.UpsertFeature(UpsertFeatureInput{
			Key:          limitKey,
			Name:         display + " row quota",
			Description:  "Max rows for module " + name + "; set via plan entitlements (0=deny, -1/unlimited=unlimited)",
			Type:         entitlement.TypeLimit,
			DefaultValue: "0",
		})
		if err != nil {
			return nil, err
		}
		out.LimitKey = limitKey
		out.LimitFeature = limitFeat
	}

	_ = s.audit(spec.AdminID, 0, "module.register", fmt.Sprintf("%s always_on=%v row_quota=%v", feat.Key, spec.AlwaysOn, spec.RowQuota))
	return out, nil
}

// RegisterCapability upserts module.{module}.{capability} with no menu binding.
func (s *EntitlementService) RegisterCapability(spec RegisterCapabilitySpec) (*models.PlatformFeature, error) {
	key := entitlement.ModuleCapabilityKey(spec.ModuleName, spec.Capability)
	if key == "" {
		return nil, apperrors.ErrInvalidArgument
	}
	display := strings.TrimSpace(spec.DisplayName)
	if display == "" {
		display = entitlement.NormalizeModuleName(spec.ModuleName) + " / " + entitlement.NormalizeModuleName(spec.Capability)
	}
	def := "false"
	if spec.AlwaysOn {
		def = "true"
	}
	feat, err := s.UpsertFeature(UpsertFeatureInput{
		Key:          key,
		Name:         display,
		Description:  "In-module capability (no menu)",
		Type:         entitlement.TypeBoolean,
		DefaultValue: def,
		MenuSlugs:    []string{},
		AlwaysOn:     &spec.AlwaysOn,
	})
	if err != nil {
		return nil, err
	}
	_ = s.audit(spec.AdminID, 0, "capability.register", feat.Key)
	return feat, nil
}

// CheckAndConsumeModuleRows enforces quota.module.{name}.rows when that limit feature exists.
// No-op when tenancy is off, tenant missing, or the limit feature is not in the catalog.
func (s *EntitlementService) CheckAndConsumeModuleRows(moduleName string, delta int64) error {
	if delta == 0 {
		return nil
	}
	if !tenancy.Enabled() {
		return nil
	}
	tenantID, ok := tenancyctx.IDFrom(s.ctx)
	if !ok || tenantID == 0 {
		return nil
	}
	limitKey := entitlement.ModuleRowsLimitKey(moduleName)
	feat, err := s.GetFeature(limitKey)
	if err != nil || feat == nil || feat.Status == 0 {
		// limit not registered / disabled => module has no row quota
		return nil
	}
	return s.CheckAndConsume(tenantID, limitKey, delta)
}

// ReleaseModuleRows decreases usage after delete (never below zero).
func (s *EntitlementService) ReleaseModuleRows(moduleName string, delta int64) error {
	if delta <= 0 {
		return nil
	}
	if !tenancy.Enabled() {
		return nil
	}
	tenantID, ok := tenancyctx.IDFrom(s.ctx)
	if !ok || tenantID == 0 {
		return nil
	}
	limitKey := entitlement.ModuleRowsLimitKey(moduleName)
	feat, err := s.GetFeature(limitKey)
	if err != nil || feat == nil || feat.Status == 0 {
		return nil
	}
	return s.AdjustUsage(tenantID, limitKey, -delta)
}

// CheckAndConsume increments usage when within limit.
func (s *EntitlementService) CheckAndConsume(tenantID uint, limitKey string, delta int64) error {
	if tenantID == 0 || delta == 0 {
		return nil
	}
	limitKey = strings.ToLower(strings.TrimSpace(limitKey))
	set, err := s.LoadEffective(tenantID)
	if err != nil {
		return err
	}
	limit, unlimited, ok := set.Limit(limitKey)
	if !ok {
		// not in effective set => treat as unrestricted
		return nil
	}
	if unlimited {
		return s.AdjustUsage(tenantID, limitKey, delta)
	}
	used, err := s.getUsage(tenantID, limitKey)
	if err != nil {
		return err
	}
	if delta > 0 && used+delta > limit {
		return apperrors.ErrEntitlementLimitExceeded
	}
	return s.AdjustUsage(tenantID, limitKey, delta)
}

func (s *EntitlementService) getUsage(tenantID uint, featureKey string) (int64, error) {
	var row models.TenantUsageCounter
	err := s.q().Where("tenant_id", tenantID).
		Where("feature_key", featureKey).
		Where("period_key", "lifetime").
		First(&row)
	if err != nil || row.ID == 0 {
		return 0, nil
	}
	return row.Used, nil
}

// AdjustUsage changes the lifetime counter by delta (clamped at 0).
func (s *EntitlementService) AdjustUsage(tenantID uint, featureKey string, delta int64) error {
	featureKey = strings.ToLower(strings.TrimSpace(featureKey))
	var row models.TenantUsageCounter
	err := s.q().Where("tenant_id", tenantID).
		Where("feature_key", featureKey).
		Where("period_key", "lifetime").
		First(&row)
	if err != nil || row.ID == 0 {
		used := delta
		if used < 0 {
			used = 0
		}
		return s.q().Create(&models.TenantUsageCounter{
			TenantID:   tenantID,
			FeatureKey: featureKey,
			PeriodKey:  "lifetime",
			Used:       used,
		})
	}
	next := row.Used + delta
	if next < 0 {
		next = 0
	}
	row.Used = next
	return s.q().Save(&row)
}

// ConsumeModuleRowsFromHTTP is a convenience for controllers/services with http.Context.
func ConsumeModuleRowsFromHTTP(ctx http.Context, moduleName string, delta int64) error {
	return NewEntitlementServiceFromHTTP(ctx).CheckAndConsumeModuleRows(moduleName, delta)
}
