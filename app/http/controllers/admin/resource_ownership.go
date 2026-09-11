package admin

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/models"
)

// canAccessOwnedResource 所有者或配置的超级管理员可访问。
func canAccessOwnedResource(actorID, ownerID, superAdminID uint) bool {
	if actorID == 0 {
		return false
	}
	if actorID == ownerID {
		return true
	}
	return superAdminID > 0 && actorID == superAdminID
}

func configuredSuperAdminID() uint {
	return cast.ToUint(facades.Config().GetInt("admin.super_admin_id", 1))
}

// ForbidUnlessOwnerOrSuper 校验资源归属；返回非 nil 时应直接作为 HTTP 响应返回。
func ForbidUnlessOwnerOrSuper(ctx http.Context, ownerAdminID uint) http.Response {
	adminID, err := helpers.GetAdminIDFromContext(ctx)
	if err != nil {
		return response.Error(ctx, http.StatusUnauthorized, "unauthorized")
	}
	if canAccessOwnedResource(adminID, ownerAdminID, configuredSuperAdminID()) {
		return nil
	}
	return response.Error(ctx, http.StatusForbidden, "forbidden")
}

// ForbidUnlessAttachmentReadable 公开附件任意已登录管理员可读；私有附件仅所有者或超管。
func ForbidUnlessAttachmentReadable(ctx http.Context, attachment *models.Attachment) http.Response {
	if attachment == nil {
		return response.Error(ctx, http.StatusNotFound, "attachment_not_found")
	}
	if attachment.IsPublic == 1 {
		return nil
	}
	return ForbidUnlessOwnerOrSuper(ctx, attachment.AdminID)
}

// ForbidUnlessAttachmentMutable 修改/删除附件：仅所有者或超管（公开附件亦同）。
func ForbidUnlessAttachmentMutable(ctx http.Context, attachment *models.Attachment) http.Response {
	if attachment == nil {
		return response.Error(ctx, http.StatusNotFound, "attachment_not_found")
	}
	return ForbidUnlessOwnerOrSuper(ctx, attachment.AdminID)
}
