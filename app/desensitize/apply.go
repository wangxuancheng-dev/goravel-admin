package desensitize

import (
	"context"

	"goravel/app/models"
)

// Field is a convenience wrapper around LoadPolicy().ApplyString.
// Call BindViewer(ctx) once before using Field in a loop so bypass is cached.
func Field(ctx context.Context, module, field, mode, value string) string {
	return LoadPolicy(ctx).ApplyString(ctx, module, field, mode, value)
}

// ApplyUserList masks phone/email on user rows for list responses.
func ApplyUserList(ctx context.Context, users []models.User) {
	ctx = BindViewer(ctx)
	p := LoadPolicy(ctx)
	if !p.ShouldMask(ctx, ModeList) {
		return
	}
	for i := range users {
		applyUserFields(p, &users[i])
	}
}

// ApplyUserExport masks phone/email for a single export row.
// Call BindViewer(ctx) once before a multi-row loop and pass that ctx here.
func ApplyUserExport(ctx context.Context, user *models.User) {
	if user == nil {
		return
	}
	p := LoadPolicy(ctx)
	if !p.ShouldMask(ctx, ModeExport) {
		return
	}
	applyUserFields(p, user)
}

func applyUserFields(p Policy, user *models.User) {
	if strategy := p.Strategy("user", "phone"); strategy != "" {
		user.Phone = Mask(user.Phone, strategy)
	}
	if strategy := p.Strategy("user", "email"); strategy != "" {
		user.Email = Mask(user.Email, strategy)
	}
}

// ApplyAdminListMap masks phone/email keys on one admin list map.
func ApplyAdminListMap(ctx context.Context, data map[string]any) {
	LoadPolicy(ctx).ApplyMap(ctx, "admin", ModeList, data)
}

// ApplyAdminListMaps masks phone/email on many admin list maps with one bypass resolve.
func ApplyAdminListMaps(ctx context.Context, items []map[string]any) {
	ctx = BindViewer(ctx)
	p := LoadPolicy(ctx)
	if !p.ShouldMask(ctx, ModeList) {
		return
	}
	for _, item := range items {
		p.ApplyMap(ctx, "admin", ModeList, item)
	}
}
