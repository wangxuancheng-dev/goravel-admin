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
func UpsertPlatformAdmin(username, password, name string) (*models.PlatformAdmin, error) {
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
			"status":   models.PlatformAdminStatusActive,
		}); err != nil {
			return nil, err
		}
		existing.Password = hashed
		existing.Name = name
		existing.Status = models.PlatformAdminStatusActive
		return &existing, nil
	}

	admin := models.PlatformAdmin{
		Username: username,
		Password: hashed,
		Name:     name,
		Status:   models.PlatformAdminStatusActive,
	}
	if err := q.Create(&admin); err != nil {
		return nil, err
	}
	return &admin, nil
}
