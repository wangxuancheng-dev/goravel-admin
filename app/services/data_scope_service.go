package services

import (
	"context"

	"github.com/goravel/framework/contracts/database/orm"

	"goravel/app/models"
	"goravel/app/rbac"
)

// DataScopeResolved re-exports the domain type for backward-compatible imports.
type DataScopeResolved = rbac.DataScopeResolved

// DataScopeApplyOpts re-exports the domain type for backward-compatible imports.
type DataScopeApplyOpts = rbac.DataScopeApplyOpts

const (
	DataScopeModeDept  = rbac.DataScopeModeDept
	DataScopeModeAdmin = rbac.DataScopeModeAdmin
)

// GetDepartmentSubtreeIDs returns departmentID and all descendant IDs.
func GetDepartmentSubtreeIDs(ctx context.Context, departmentID uint) []uint {
	return rbac.GetDepartmentSubtreeIDs(ctx, departmentID)
}

// adminFromContext extracts Admin from JWT/http context values.
func adminFromContext(ctx context.Context) *models.Admin {
	return rbac.AdminFromContext(ctx)
}

// ResolveAdminDataScope computes effective scope (widest among roles). No admin => All (non-HTTP callers).
func ResolveAdminDataScope(ctx context.Context) DataScopeResolved {
	return rbac.ResolveAdminDataScope(ctx)
}

// ApplyDataScope applies resolved data scope to a list query. No-op when scope is All or admin missing.
func ApplyDataScope(ctx context.Context, query orm.Query, opts DataScopeApplyOpts) orm.Query {
	return rbac.ApplyDataScope(ctx, query, opts)
}

// CanAccessOwnedBy reports whether current admin may access a resource owned by ownerAdminID.
func CanAccessOwnedBy(ctx context.Context, ownerAdminID uint) bool {
	return rbac.CanAccessOwnedBy(ctx, ownerAdminID)
}
