package desensitize

import (
	"context"
	"strings"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/rbac"
	"goravel/app/utils"
)

// View modes. Detail/show is intentionally not masked by default.
const (
	ModeList   = "list"
	ModeExport = "export"
)

type viewerAdminIDKey struct{}
type viewerBypassCacheKey struct{}

// WithViewerAdminID attaches the acting admin id (export jobs, background work).
func WithViewerAdminID(ctx context.Context, adminID uint) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, viewerAdminIDKey{}, adminID)
}

// BindViewer resolves bypass once and caches it on ctx.
// JWT stores admin by value, so AdminFromContext returns a fresh copy each time;
// without this cache, list/export loops would re-query roles per row.
func BindViewer(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Value(viewerBypassCacheKey{}).(bool); ok {
		return ctx
	}
	bypass := LoadPolicy(ctx).computeViewerBypasses(ctx)
	return context.WithValue(ctx, viewerBypassCacheKey{}, bypass)
}

// ViewerAdminID returns the id set by WithViewerAdminID.
func ViewerAdminID(ctx context.Context) uint {
	if ctx == nil {
		return 0
	}
	v := ctx.Value(viewerAdminIDKey{})
	if v == nil {
		return 0
	}
	return cast.ToUint(v)
}

// Policy is the resolved desensitize configuration.
type Policy struct {
	Enabled         bool
	BypassRoleSlugs []string
	Modes           map[string]bool
	Modules         map[string]map[string]string // module -> field -> strategy
}

// LoadPolicy reads process defaults from config/desensitize, then overlays
// per-tenant (or single-site) rows from the configs table when present.
// With TENANCY_DRIVER=database each tenant DB has its own configs → independent switch.
func LoadPolicy(ctx context.Context) Policy {
	envEnabled := parseEnabled(facades.Config().Get("desensitize.enabled", true))
	envBypass := facades.Config().GetString("desensitize.bypass_role_slugs", "super-admin")

	p := Policy{
		Enabled:         utils.GetConfigValueBool(ctx, "desensitize", "enabled", envEnabled),
		BypassRoleSlugs: splitCSV(utils.GetConfigValue(ctx, "desensitize", "bypass_role_slugs", envBypass)),
		Modes:           map[string]bool{},
		Modules:         map[string]map[string]string{},
	}

	modesRaw := facades.Config().Get("desensitize.modes", []string{ModeList, ModeExport})
	for _, m := range toStringSlice(modesRaw) {
		m = strings.ToLower(strings.TrimSpace(m))
		if m != "" {
			p.Modes[m] = true
		}
	}
	if len(p.Modes) == 0 {
		p.Modes[ModeList] = true
		p.Modes[ModeExport] = true
	}

	modulesRaw := facades.Config().Get("desensitize.modules")
	if modules, ok := modulesRaw.(map[string]any); ok {
		for module, fieldsRaw := range modules {
			fields := map[string]string{}
			switch typed := fieldsRaw.(type) {
			case map[string]any:
				for field, strategy := range typed {
					fields[strings.ToLower(field)] = strings.ToLower(cast.ToString(strategy))
				}
			case map[string]string:
				for field, strategy := range typed {
					fields[strings.ToLower(field)] = strings.ToLower(strategy)
				}
			}
			if len(fields) > 0 {
				p.Modules[strings.ToLower(module)] = fields
			}
		}
	}
	return p
}

// ModeEnabled reports whether masking applies for the mode.
func (p Policy) ModeEnabled(mode string) bool {
	if !p.Enabled {
		return false
	}
	return p.Modes[strings.ToLower(strings.TrimSpace(mode))]
}

// Strategy returns the strategy for module.field, or empty if unset.
func (p Policy) Strategy(module, field string) string {
	fields, ok := p.Modules[strings.ToLower(module)]
	if !ok {
		return ""
	}
	return fields[strings.ToLower(field)]
}

// ShouldMask is true when enabled, mode is active, and viewer does not bypass.
func (p Policy) ShouldMask(ctx context.Context, mode string) bool {
	if !p.ModeEnabled(mode) {
		return false
	}
	return !p.ViewerBypasses(ctx)
}

// ViewerBypasses is true when the acting admin has a bypass role slug.
func (p Policy) ViewerBypasses(ctx context.Context) bool {
	if ctx != nil {
		if cached, ok := ctx.Value(viewerBypassCacheKey{}).(bool); ok {
			return cached
		}
	}
	return p.computeViewerBypasses(ctx)
}

func (p Policy) computeViewerBypasses(ctx context.Context) bool {
	if len(p.BypassRoleSlugs) == 0 {
		return false
	}
	admin := rbac.AdminFromContext(ctx)
	if admin != nil && admin.ID > 0 {
		return adminHasBypassRole(ctx, admin, p.BypassRoleSlugs)
	}
	if id := ViewerAdminID(ctx); id > 0 {
		return adminIDHasBypassRole(ctx, id, p.BypassRoleSlugs)
	}
	return false
}

// ApplyString masks value for module.field when policy says so.
func (p Policy) ApplyString(ctx context.Context, module, field, mode, value string) string {
	if !p.ShouldMask(ctx, mode) {
		return value
	}
	strategy := p.Strategy(module, field)
	if strategy == "" {
		return value
	}
	return Mask(value, strategy)
}

// ApplyMap masks configured keys in-place for the module/mode.
func (p Policy) ApplyMap(ctx context.Context, module, mode string, data map[string]any) {
	if data == nil || !p.ShouldMask(ctx, mode) {
		return
	}
	fields, ok := p.Modules[strings.ToLower(module)]
	if !ok {
		return
	}
	for field, strategy := range fields {
		raw, exists := data[field]
		if !exists {
			continue
		}
		s, ok := raw.(string)
		if !ok {
			s = cast.ToString(raw)
		}
		data[field] = Mask(s, strategy)
	}
}

func adminHasBypassRole(ctx context.Context, admin *models.Admin, slugs []string) bool {
	if admin == nil {
		return false
	}
	if len(admin.Roles) == 0 {
		_ = loadAdminRoles(ctx, admin)
	}
	slugSet := toLowerSet(slugs)
	for _, role := range admin.Roles {
		if role.Status != 1 {
			continue
		}
		if slugSet[strings.ToLower(role.Slug)] {
			return true
		}
	}
	return false
}

func adminIDHasBypassRole(ctx context.Context, adminID uint, slugs []string) bool {
	if adminID == 0 {
		return false
	}
	admin := &models.Admin{}
	admin.ID = adminID
	return adminHasBypassRole(ctx, admin, slugs)
}

func loadAdminRoles(ctx context.Context, admin *models.Admin) error {
	if admin == nil || admin.ID == 0 {
		return nil
	}
	if len(admin.Roles) > 0 {
		return nil
	}
	type adminRoleRow struct {
		RoleID uint `gorm:"column:role_id"`
	}
	var rows []adminRoleRow
	if err := appfacades.OrmQuery(ctx).Table("admin_role").Where("admin_id", admin.ID).Find(&rows); err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	roleIDs := make([]uint, 0, len(rows))
	for _, row := range rows {
		roleIDs = append(roleIDs, row.RoleID)
	}
	var roles []models.Role
	if err := appfacades.OrmQuery(ctx).Where("id IN ?", roleIDs).Where("status", 1).Find(&roles); err != nil {
		return err
	}
	admin.Roles = roles
	return nil
}

func parseEnabled(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		s := strings.ToLower(strings.TrimSpace(t))
		return s == "1" || s == "true" || s == "on" || s == "yes"
	default:
		return cast.ToBool(v)
	}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func toStringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			s := strings.TrimSpace(cast.ToString(item))
			if s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		return splitCSV(t)
	default:
		return nil
	}
}

func toLowerSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, item := range items {
		out[strings.ToLower(strings.TrimSpace(item))] = true
	}
	return out
}
