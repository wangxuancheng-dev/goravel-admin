package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"goravel/app/models"
)

func TestResolveAdminDataScopeWidestAndSuperAdmin(t *testing.T) {
	admin := models.Admin{
		DepartmentID: 3,
		Roles: []models.Role{
			{Slug: "ops", Status: 1, DataScope: models.DataScopeSelf},
			{Slug: "lead", Status: 1, DataScope: models.DataScopeDept},
		},
	}
	admin.ID = 42
	ctx := context.WithValue(context.Background(), "admin", admin)

	resolved := ResolveAdminDataScope(ctx)
	assert.Equal(t, models.DataScopeDept, resolved.Scope)
	assert.Equal(t, uint(42), resolved.AdminID)
	assert.Equal(t, []uint{3}, resolved.DeptIDs)

	super := models.Admin{
		Roles: []models.Role{
			{Slug: "super-admin", Status: 1, DataScope: models.DataScopeSelf},
		},
	}
	super.ID = 1
	ctx2 := context.WithValue(context.Background(), "admin", super)
	resolved2 := ResolveAdminDataScope(ctx2)
	assert.Equal(t, models.DataScopeAll, resolved2.Scope)
}

func TestResolveAdminDataScopeDeptWithoutDepartmentFallsBackToSelf(t *testing.T) {
	admin := models.Admin{
		DepartmentID: 0,
		Roles: []models.Role{
			{Slug: "ops", Status: 1, DataScope: models.DataScopeDeptAndChild},
		},
	}
	admin.ID = 9
	ctx := context.WithValue(context.Background(), "admin", admin)
	resolved := ResolveAdminDataScope(ctx)
	assert.Equal(t, models.DataScopeSelf, resolved.Scope)
}

func TestResolveAdminDataScopeNoAdminIsAll(t *testing.T) {
	resolved := ResolveAdminDataScope(context.Background())
	assert.Equal(t, models.DataScopeAll, resolved.Scope)
}

func TestCanAccessOwnedBySelf(t *testing.T) {
	admin := models.Admin{
		Roles: []models.Role{{Slug: "ops", Status: 1, DataScope: models.DataScopeSelf}},
	}
	admin.ID = 7
	ctx := context.WithValue(context.Background(), "admin", admin)
	assert.True(t, CanAccessOwnedBy(ctx, 7))
	assert.False(t, CanAccessOwnedBy(ctx, 8))
}
