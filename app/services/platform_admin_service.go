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

// PlatformAdminFilters for platform admin list.
type PlatformAdminFilters struct {
	Username string
	Role     string
	Status   string
	OrderBy  string
}

// ListPlatformAdminsPaged returns filtered platform admins.
func ListPlatformAdminsPaged(filters PlatformAdminFilters, page, pageSize int) ([]models.PlatformAdmin, int64, error) {
	if !tenancy.Enabled() {
		return nil, 0, apperrors.ErrTenancyDisabled
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	query := appfacades.PlatformOrmQuery(nil).Model(&models.PlatformAdmin{})
	if u := strings.TrimSpace(filters.Username); u != "" {
		query = query.Where("username LIKE ? OR name LIKE ?", "%"+u+"%", "%"+u+"%")
	}
	if r := strings.TrimSpace(filters.Role); r != "" {
		query = query.Where("role", models.NormalizePlatformAdminRole(r))
	}
	if s := strings.TrimSpace(filters.Status); s != "" {
		query = query.Where("status", s)
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	order := "id desc"
	if strings.TrimSpace(filters.OrderBy) != "" {
		parts := strings.Split(filters.OrderBy, ":")
		col := "id"
		dir := "desc"
		if len(parts) >= 1 {
			switch parts[0] {
			case "id", "username", "name", "role", "status", "created_at":
				col = parts[0]
			}
		}
		if len(parts) >= 2 && strings.EqualFold(parts[1], "asc") {
			dir = "asc"
		}
		order = col + " " + dir
	}
	var rows []models.PlatformAdmin
	err = query.Order(order).Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows)
	return rows, total, err
}

// GetPlatformAdminByID returns one platform admin.
func GetPlatformAdminByID(id uint) (*models.PlatformAdmin, error) {
	if !tenancy.Enabled() {
		return nil, apperrors.ErrTenancyDisabled
	}
	if id == 0 {
		return nil, apperrors.ErrInvalidArgument
	}
	var admin models.PlatformAdmin
	if err := appfacades.PlatformOrmQuery(nil).Where("id", id).FirstOrFail(&admin); err != nil {
		return nil, apperrors.ErrUserNotFound.WithError(err)
	}
	return &admin, nil
}

// CreatePlatformAdminInput for console create.
type CreatePlatformAdminInput struct {
	Username string
	Password string
	Name     string
	Role     string
}

// CreatePlatformAdmin creates a new platform admin (no upsert).
func CreatePlatformAdmin(in CreatePlatformAdminInput) (*models.PlatformAdmin, error) {
	if !tenancy.Enabled() {
		return nil, apperrors.ErrTenancyDisabled
	}
	username := strings.TrimSpace(in.Username)
	password := in.Password
	if username == "" || password == "" {
		return nil, apperrors.ErrInvalidArgument
	}
	if len(password) < 6 {
		return nil, apperrors.ErrPasswordTooWeak
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = username
	}
	role := models.NormalizePlatformAdminRole(in.Role)

	q := appfacades.PlatformOrmQuery(nil)
	var existing models.PlatformAdmin
	if err := q.WithTrashed().Where("username", username).First(&existing); err == nil && existing.ID > 0 {
		if !existing.DeletedAt.Valid {
			return nil, apperrors.ErrUsernameExists
		}
		hashed, err := facades.Hash().Make(password)
		if err != nil {
			return nil, apperrors.ErrPasswordEncryptFailed.WithError(err)
		}
		if _, err := q.WithTrashed().Where("id", existing.ID).Restore(&models.PlatformAdmin{}); err != nil {
			return nil, err
		}
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
		existing.DeletedAt.Valid = false
		return &existing, nil
	}
	hashed, err := facades.Hash().Make(password)
	if err != nil {
		return nil, apperrors.ErrPasswordEncryptFailed.WithError(err)
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

// UpdatePlatformAdminInput for console update.
type UpdatePlatformAdminInput struct {
	Name       *string
	Role       *string
	Status     *uint8
	AllowedIPs *string
}

// UpdatePlatformAdmin updates name/role/status with last-owner and self-protection.
func UpdatePlatformAdmin(actorID, targetID uint, in UpdatePlatformAdminInput) (*models.PlatformAdmin, error) {
	if !tenancy.Enabled() {
		return nil, apperrors.ErrTenancyDisabled
	}
	admin, err := GetPlatformAdminByID(targetID)
	if err != nil {
		return nil, err
	}

	updates := map[string]any{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			name = admin.Username
		}
		updates["name"] = name
		admin.Name = name
	}

	roleChanging := false
	newRole := models.NormalizePlatformAdminRole(admin.Role)
	if in.Role != nil {
		newRole = models.NormalizePlatformAdminRole(*in.Role)
		if newRole != models.NormalizePlatformAdminRole(admin.Role) {
			roleChanging = true
			updates["role"] = newRole
		}
	}

	statusChanging := false
	newStatus := admin.Status
	if in.Status != nil {
		newStatus = *in.Status
		if newStatus != models.PlatformAdminStatusActive && newStatus != models.PlatformAdminStatusDisabled {
			return nil, apperrors.ErrInvalidArgument
		}
		if newStatus != admin.Status {
			statusChanging = true
			updates["status"] = newStatus
		}
	}

	if in.AllowedIPs != nil {
		ips := strings.TrimSpace(*in.AllowedIPs)
		updates["allowed_ips"] = ips
		admin.AllowedIPs = ips
	}

	if actorID > 0 && actorID == targetID && (roleChanging || statusChanging) {
		return nil, apperrors.ErrPlatformCannotModifySelf
	}

	wasActiveOwner := admin.Status == models.PlatformAdminStatusActive && models.NormalizePlatformAdminRole(admin.Role) == models.PlatformAdminRoleOwner
	willBeActiveOwner := newStatus == models.PlatformAdminStatusActive && newRole == models.PlatformAdminRoleOwner
	if wasActiveOwner && !willBeActiveOwner {
		n, err := countActivePlatformOwners()
		if err != nil {
			return nil, err
		}
		if n <= 1 {
			return nil, apperrors.ErrPlatformLastOwner
		}
	}

	if len(updates) == 0 {
		return admin, nil
	}
	if _, err := appfacades.PlatformOrmQuery(nil).Model(admin).Update(updates); err != nil {
		return nil, err
	}
	admin.Role = newRole
	admin.Status = newStatus

	if statusChanging && newStatus == models.PlatformAdminStatusDisabled {
		_ = NewPlatformTokenService(nil).DeleteTokensByUser(models.TokenableTypePlatformAdmin, targetID)
	}
	return admin, nil
}

// ResetPlatformAdminPassword sets a new password (owner action; no old password).
func ResetPlatformAdminPassword(targetID uint, newPassword string) error {
	if !tenancy.Enabled() {
		return apperrors.ErrTenancyDisabled
	}
	if targetID == 0 || newPassword == "" {
		return apperrors.ErrInvalidArgument
	}
	if len(newPassword) < 6 {
		return apperrors.ErrPasswordTooWeak
	}
	admin, err := GetPlatformAdminByID(targetID)
	if err != nil {
		return err
	}
	hashed, err := facades.Hash().Make(newPassword)
	if err != nil {
		return apperrors.ErrPasswordEncryptFailed.WithError(err)
	}
	if _, err := appfacades.PlatformOrmQuery(nil).Model(admin).Update(map[string]any{"password": hashed}); err != nil {
		return err
	}
	_ = NewPlatformTokenService(nil).DeleteTokensByUser(models.TokenableTypePlatformAdmin, targetID)
	return nil
}

// DeletePlatformAdmin soft-deletes a platform admin with protections.
func DeletePlatformAdmin(actorID, targetID uint) error {
	if !tenancy.Enabled() {
		return apperrors.ErrTenancyDisabled
	}
	if targetID == 0 {
		return apperrors.ErrInvalidArgument
	}
	if actorID > 0 && actorID == targetID {
		return apperrors.ErrAdminCannotDeleteSelf
	}
	admin, err := GetPlatformAdminByID(targetID)
	if err != nil {
		return err
	}
	if admin.Status == models.PlatformAdminStatusActive && models.NormalizePlatformAdminRole(admin.Role) == models.PlatformAdminRoleOwner {
		n, err := countActivePlatformOwners()
		if err != nil {
			return err
		}
		if n <= 1 {
			return apperrors.ErrPlatformLastOwner
		}
	}
	if _, err := appfacades.PlatformOrmQuery(nil).Delete(admin); err != nil {
		return err
	}
	_ = NewPlatformTokenService(nil).DeleteTokensByUser(models.TokenableTypePlatformAdmin, targetID)
	return nil
}

func countActivePlatformOwners() (int64, error) {
	return appfacades.PlatformOrmQuery(nil).Model(&models.PlatformAdmin{}).
		Where("status", models.PlatformAdminStatusActive).
		Where("(role = ? OR role = ? OR role IS NULL)", models.PlatformAdminRoleOwner, "").
		Count()
}

// PlatformAdminToJSON hides password.
func PlatformAdminToJSON(a *models.PlatformAdmin) map[string]any {
	if a == nil {
		return nil
	}
	role := models.NormalizePlatformAdminRole(a.Role)
	return map[string]any{
		"id":           a.ID,
		"username":     a.Username,
		"name":         a.Name,
		"role":         role,
		"status":       a.Status,
		"is_2fa_bound": a.Is2FABound(),
		"allowed_ips":  a.AllowedIPs,
		"created_at":   a.CreatedAt,
		"updated_at":   a.UpdatedAt,
	}
}
