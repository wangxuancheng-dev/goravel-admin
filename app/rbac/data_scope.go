package rbac

import (
	"context"

	"github.com/goravel/framework/contracts/database/orm"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// DataScopeResolved is the effective data scope for the current admin.
type DataScopeResolved struct {
	Scope   uint8
	AdminID uint
	DeptIDs []uint // for dept / deptAndChild / custom
}

// DataScopeApplyOpts controls how ApplyDataScope filters a list query.
type DataScopeApplyOpts struct {
	// ModeDept filters by department column (e.g. admins.department_id).
	// ModeAdmin filters by owner admin column (e.g. articles.admin_id).
	Mode        string
	DeptColumn  string
	AdminColumn string
}

const (
	DataScopeModeDept  = "dept"
	DataScopeModeAdmin = "admin"
)

// GetDepartmentSubtreeIDs returns departmentID and all descendant IDs.
func GetDepartmentSubtreeIDs(ctx context.Context, departmentID uint) []uint {
	if departmentID == 0 {
		return nil
	}
	ids := []uint{departmentID}
	appendChildDepartmentIDs(ctx, departmentID, &ids)
	return ids
}

func appendChildDepartmentIDs(ctx context.Context, parentID uint, departmentIDs *[]uint) {
	var children []models.Department
	if err := appfacades.OrmQuery(ctx).Where("parent_id", parentID).Get(&children); err != nil {
		return
	}
	for _, child := range children {
		*departmentIDs = append(*departmentIDs, child.ID)
		appendChildDepartmentIDs(ctx, child.ID, departmentIDs)
	}
}

// AdminFromContext extracts Admin from JWT/http context values.
func AdminFromContext(ctx context.Context) *models.Admin {
	if ctx == nil {
		return nil
	}
	v := ctx.Value("admin")
	if v == nil {
		return nil
	}
	if admin, ok := v.(models.Admin); ok {
		return &admin
	}
	if admin, ok := v.(*models.Admin); ok {
		return admin
	}
	return nil
}

func loadAdminRoles(ctx context.Context, admin *models.Admin) error {
	if admin == nil {
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

func loadRoleDepartments(ctx context.Context, roleIDs []uint) (map[uint][]uint, error) {
	out := map[uint][]uint{}
	if len(roleIDs) == 0 {
		return out, nil
	}
	type row struct {
		RoleID       uint `gorm:"column:role_id"`
		DepartmentID uint `gorm:"column:department_id"`
	}
	var rows []row
	if err := appfacades.OrmQuery(ctx).Table("role_department").Where("role_id IN ?", roleIDs).Find(&rows); err != nil {
		return out, err
	}
	for _, r := range rows {
		out[r.RoleID] = append(out[r.RoleID], r.DepartmentID)
	}
	return out, nil
}

// ResolveAdminDataScope computes effective scope (widest among roles). No admin => All (non-HTTP callers).
func ResolveAdminDataScope(ctx context.Context) DataScopeResolved {
	admin := AdminFromContext(ctx)
	if admin == nil || admin.ID == 0 {
		return DataScopeResolved{Scope: models.DataScopeAll}
	}
	_ = loadAdminRoles(ctx, admin)

	resolved := DataScopeResolved{
		Scope:   models.DataScopeSelf, // start restrictive; widen via min()
		AdminID: admin.ID,
	}
	if len(admin.Roles) == 0 {
		return resolved
	}

	hasRole := false
	customRoleIDs := make([]uint, 0)
	for _, role := range admin.Roles {
		if role.Status != 1 {
			continue
		}
		if role.Slug == "super-admin" {
			return DataScopeResolved{Scope: models.DataScopeAll, AdminID: admin.ID}
		}
		scope := role.DataScope
		if scope == 0 {
			scope = models.DataScopeAll
		}
		if !hasRole || scope < resolved.Scope {
			resolved.Scope = scope
		}
		hasRole = true
		if scope == models.DataScopeCustom {
			customRoleIDs = append(customRoleIDs, role.ID)
		}
	}
	if !hasRole {
		return resolved
	}

	switch resolved.Scope {
	case models.DataScopeAll:
		return resolved
	case models.DataScopeSelf:
		return resolved
	case models.DataScopeCustom:
		deptMap, err := loadRoleDepartments(ctx, customRoleIDs)
		if err != nil {
			resolved.Scope = models.DataScopeSelf
			return resolved
		}
		seen := map[uint]bool{}
		for _, roleID := range customRoleIDs {
			for _, deptID := range deptMap[roleID] {
				if deptID == 0 || seen[deptID] {
					continue
				}
				seen[deptID] = true
				resolved.DeptIDs = append(resolved.DeptIDs, deptID)
			}
		}
		if len(resolved.DeptIDs) == 0 {
			resolved.Scope = models.DataScopeSelf
		}
		return resolved
	case models.DataScopeDept:
		if admin.DepartmentID == 0 {
			resolved.Scope = models.DataScopeSelf
			return resolved
		}
		resolved.DeptIDs = []uint{admin.DepartmentID}
		return resolved
	case models.DataScopeDeptAndChild:
		if admin.DepartmentID == 0 {
			resolved.Scope = models.DataScopeSelf
			return resolved
		}
		resolved.DeptIDs = GetDepartmentSubtreeIDs(ctx, admin.DepartmentID)
		if len(resolved.DeptIDs) == 0 {
			resolved.Scope = models.DataScopeSelf
		}
		return resolved
	default:
		resolved.Scope = models.DataScopeSelf
		return resolved
	}
}

func adminIDsInDepartments(ctx context.Context, deptIDs []uint) []uint {
	if len(deptIDs) == 0 {
		return nil
	}
	var admins []models.Admin
	if err := appfacades.OrmQuery(ctx).Model(&models.Admin{}).Where("department_id IN ?", deptIDs).Select("id").Find(&admins); err != nil {
		return nil
	}
	ids := make([]uint, 0, len(admins))
	for _, a := range admins {
		ids = append(ids, a.ID)
	}
	return ids
}

func toAnySlice[T any](in []T) []any {
	out := make([]any, len(in))
	for i, v := range in {
		out[i] = v
	}
	return out
}

// ApplyDataScope applies resolved data scope to a list query. No-op when scope is All or admin missing.
func ApplyDataScope(ctx context.Context, query orm.Query, opts DataScopeApplyOpts) orm.Query {
	if query == nil {
		return query
	}
	resolved := ResolveAdminDataScope(ctx)
	if resolved.Scope == models.DataScopeAll {
		return query
	}

	mode := opts.Mode
	if mode == "" {
		mode = DataScopeModeAdmin
	}
	deptCol := opts.DeptColumn
	if deptCol == "" {
		deptCol = "department_id"
	}
	adminCol := opts.AdminColumn
	if adminCol == "" {
		adminCol = "admin_id"
	}

	switch mode {
	case DataScopeModeDept:
		if resolved.Scope == models.DataScopeSelf {
			return query.Where("id = ?", resolved.AdminID)
		}
		if len(resolved.DeptIDs) == 0 {
			return query.Where("1 = 0")
		}
		return query.WhereIn(deptCol, toAnySlice(resolved.DeptIDs))
	default: // admin owner column
		if resolved.Scope == models.DataScopeSelf {
			return query.Where(adminCol+" = ?", resolved.AdminID)
		}
		ids := adminIDsInDepartments(ctx, resolved.DeptIDs)
		if len(ids) == 0 {
			// still allow self rows even if no peers in dept
			return query.Where(adminCol+" = ?", resolved.AdminID)
		}
		// include self in case department_id unset on self
		seen := map[uint]bool{}
		merged := make([]uint, 0, len(ids)+1)
		for _, id := range ids {
			if seen[id] {
				continue
			}
			seen[id] = true
			merged = append(merged, id)
		}
		if !seen[resolved.AdminID] {
			merged = append(merged, resolved.AdminID)
		}
		return query.WhereIn(adminCol, toAnySlice(merged))
	}
}

// CanAccessOwnedBy reports whether current admin may access a resource owned by ownerAdminID.
func CanAccessOwnedBy(ctx context.Context, ownerAdminID uint) bool {
	resolved := ResolveAdminDataScope(ctx)
	if resolved.Scope == models.DataScopeAll {
		return true
	}
	if ownerAdminID == 0 {
		return false
	}
	if resolved.Scope == models.DataScopeSelf {
		return ownerAdminID == resolved.AdminID
	}
	ids := adminIDsInDepartments(ctx, resolved.DeptIDs)
	for _, id := range ids {
		if id == ownerAdminID {
			return true
		}
	}
	return ownerAdminID == resolved.AdminID
}
