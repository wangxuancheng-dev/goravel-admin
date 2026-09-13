package services

import (
	"strings"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"

	"github.com/goravel/framework/facades"
)

// UpsertPlatformAdmin creates or resets a platform console admin on the platform DB.
// role is owner|viewer (empty => owner).
func UpsertPlatformAdmin(username, password, name, role string) (*models.PlatformAdmin, error) {
	if !tenancy.Enabled() {
		return nil, apperrors.ErrTenancyDisabled
	}
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, apperrors.ErrInvalidArgument.WithMessage("username and password are required")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = username
	}
	role = models.NormalizePlatformAdminRole(role)
	hashed, err := facades.Hash().Make(password)
	if err != nil {
		return nil, apperrors.ErrPasswordEncryptFailed.WithError(err)
	}

	q := appfacades.PlatformOrmQuery(nil)
	var existing models.PlatformAdmin
	if err := q.Where("username", username).First(&existing); err == nil && existing.ID > 0 {
		if _, err := q.Model(&existing).Update(map[string]any{
			"password": hashed,
			"name":     name,
			"role":     role,
			"status":   models.PlatformAdminStatusActive,
		}); err != nil {
			return nil, err
		}
		existing.Password = hashed
		existing.Name = name
		existing.Role = role
		existing.Status = models.PlatformAdminStatusActive
		return &existing, nil
	}

	admin := models.PlatformAdmin{
		Username: username,
		Password: hashed,
		Name:     name,
		Role:     role,
		Status:   models.PlatformAdminStatusActive,
	}
	if err := q.Create(&admin); err != nil {
		return nil, err
	}
	return &admin, nil
}

// ChangePlatformAdminPassword verifies the old password then sets a new hash.
func ChangePlatformAdminPassword(adminID uint, oldPassword, newPassword string) error {
	if !tenancy.Enabled() {
		return apperrors.ErrTenancyDisabled
	}
	if adminID == 0 || oldPassword == "" || newPassword == "" {
		return apperrors.ErrInvalidArgument
	}
	q := appfacades.PlatformOrmQuery(nil)
	var admin models.PlatformAdmin
	if err := q.Where("id", adminID).First(&admin); err != nil || admin.ID == 0 {
		return apperrors.ErrUserNotFound
	}
	if !facades.Hash().Check(oldPassword, admin.Password) {
		return apperrors.ErrOldPasswordError
	}
	hashed, err := facades.Hash().Make(newPassword)
	if err != nil {
		return apperrors.ErrPasswordEncryptFailed.WithError(err)
	}
	if _, err := q.Model(&admin).Update(map[string]any{"password": hashed}); err != nil {
		return err
	}
	return nil
}

// PlatformAdminToJSON hides password.
func PlatformAdminToJSON(a *models.PlatformAdmin) map[string]any {
	if a == nil {
		return nil
	}
	role := models.NormalizePlatformAdminRole(a.Role)
	return map[string]any{
		"id":       a.ID,
		"username": a.Username,
		"name":     a.Name,
		"role":     role,
		"status":   a.Status,
	}
}
