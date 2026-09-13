package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
)

const tenantOpLockTTL = 30 * time.Minute

// TenantOpsArgs is the queue payload for platform tenant maintenance jobs.
type TenantOpsArgs struct {
	TenantID   uint   `json:"tenant_id"`
	Op         string `json:"op"` // migrate|seed|backup|restore
	WithSeed   bool   `json:"with_seed,omitempty"`
	BackupName string `json:"backup_name,omitempty"` // restore only
	Keep       int    `json:"keep,omitempty"`        // backup retention override; <0 = config default
}

type TenantOpsService struct {
	admin *TenantAdminService
	conn  *TenantConnectionService
}

func NewTenantOpsService() *TenantOpsService {
	return &TenantOpsService{
		admin: NewTenantAdminService(),
		conn:  NewTenantConnectionService(),
	}
}

func (s *TenantOpsService) requireEnabled() error {
	if !tenancy.Enabled() {
		return apperrors.ErrTenancyDisabled
	}
	return nil
}

// BeginQueuedOp marks the tenant as queued for op. Caller must Dispatch the job;
// on dispatch failure call MarkOpFailed (and restore provision for migrate).
func (s *TenantOpsService) BeginQueuedOp(id uint, op string, withSeed bool) (*models.Tenant, TenantOpsArgs, error) {
	return s.BeginQueuedOpWithBackup(id, op, withSeed, "")
}

// BeginQueuedOpWithBackup is BeginQueuedOp plus optional backup file name (restore).
func (s *TenantOpsService) BeginQueuedOpWithBackup(id uint, op string, withSeed bool, backupName string) (*models.Tenant, TenantOpsArgs, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, TenantOpsArgs{}, err
	}
	op = strings.TrimSpace(op)
	switch op {
	case models.TenantOpMigrate, models.TenantOpSeed, models.TenantOpBackup, models.TenantOpRestore:
	default:
		return nil, TenantOpsArgs{}, apperrors.ErrInvalidArgument.WithMessage("unknown tenant op")
	}
	if op == models.TenantOpRestore && strings.TrimSpace(backupName) == "" {
		return nil, TenantOpsArgs{}, apperrors.ErrInvalidArgument.WithMessage("backup_name required for restore")
	}

	tenant, err := s.admin.GetByID(id)
	if err != nil {
		return nil, TenantOpsArgs{}, err
	}
	if tenantOpBusy(tenant) {
		return nil, TenantOpsArgs{}, apperrors.ErrTenantOpInProgress
	}

	lockKey := fmt.Sprintf("platform:tenant_op:%d", id)
	lock := facades.Cache().Lock(lockKey, 15*time.Second)
	if !lock.Get() {
		return nil, TenantOpsArgs{}, apperrors.ErrTenantOpInProgress
	}
	defer lock.Release()

	tenant, err = s.admin.GetByID(id)
	if err != nil {
		return nil, TenantOpsArgs{}, err
	}
	if tenantOpBusy(tenant) {
		return nil, TenantOpsArgs{}, apperrors.ErrTenantOpInProgress
	}

	now := time.Now()
	updates := map[string]any{
		"last_op":         op,
		"last_op_status":  models.TenantOpStatusQueued,
		"last_op_message": "",
		"last_op_at":      now,
	}
	if op == models.TenantOpMigrate {
		updates["provision_status"] = models.TenantProvisionMigrating
		updates["last_migrate_error"] = ""
	}
	if _, err := appfacades.PlatformOrmQuery(nil).Model(tenant).Update(updates); err != nil {
		return nil, TenantOpsArgs{}, err
	}
	tenant.LastOp = op
	tenant.LastOpStatus = models.TenantOpStatusQueued
	tenant.LastOpMessage = ""
	tenant.LastOpAt = &now
	if op == models.TenantOpMigrate {
		tenant.ProvisionStatus = models.TenantProvisionMigrating
		tenant.LastMigrateError = ""
	}
	AppendTenantOpLog(tenant, op, models.TenantOpStatusQueued, "", &now, nil)

	return tenant, TenantOpsArgs{TenantID: id, Op: op, WithSeed: withSeed, BackupName: strings.TrimSpace(backupName)}, nil
}

// BeginQueuedBackup enqueues backup with optional keep override (negative = config default).
func (s *TenantOpsService) BeginQueuedBackup(id uint, keep int) (*models.Tenant, TenantOpsArgs, error) {
	tenant, args, err := s.BeginQueuedOpWithBackup(id, models.TenantOpBackup, false, "")
	if err != nil {
		return nil, TenantOpsArgs{}, err
	}
	args.Keep = keep
	return tenant, args, nil
}

func (s *TenantOpsService) MarkOpFailed(tenant *models.Tenant, msg string) error {
	if tenant == nil || tenant.ID == 0 {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if len(msg) > 2000 {
		msg = msg[:2000]
	}
	now := time.Now()
	updates := map[string]any{
		"last_op_status":  models.TenantOpStatusFailed,
		"last_op_message": msg,
		"last_op_at":      now,
	}
	if tenant.LastOp == models.TenantOpMigrate || tenant.ProvisionStatus == models.TenantProvisionMigrating {
		updates["provision_status"] = models.TenantProvisionFailed
		tenant.ProvisionStatus = models.TenantProvisionFailed
	}
	_, err := appfacades.PlatformOrmQuery(nil).Model(tenant).Update(updates)
	if err == nil {
		tenant.LastOpStatus = models.TenantOpStatusFailed
		tenant.LastOpMessage = msg
		tenant.LastOpAt = &now
		AppendTenantOpLog(tenant, tenant.LastOp, models.TenantOpStatusFailed, msg, nil, &now)
	}
	return err
}

func tenantOpBusy(t *models.Tenant) bool {
	if t == nil {
		return false
	}
	busyState := strings.TrimSpace(t.ProvisionStatus) == models.TenantProvisionMigrating
	switch strings.TrimSpace(t.LastOpStatus) {
	case models.TenantOpStatusQueued, models.TenantOpStatusRunning:
		busyState = true
	}
	if !busyState {
		return false
	}
	if t.LastOpAt == nil || time.Since(*t.LastOpAt) > tenantOpLockTTL {
		return false
	}
	return true
}

// RunTenantOp executes migrate/seed/backup/restore for a queued platform job.
func RunTenantOp(args TenantOpsArgs) error {
	if args.TenantID == 0 {
		return apperrors.ErrInvalidArgument.WithMessage("tenant_id required")
	}
	op := strings.TrimSpace(args.Op)
	svc := NewTenantOpsService()
	tenant, err := svc.admin.GetByID(args.TenantID)
	if err != nil {
		return err
	}

	lockKey := fmt.Sprintf("platform:tenant_op:%d", args.TenantID)
	lock := facades.Cache().Lock(lockKey, tenantOpLockTTL)
	if !lock.Get() {
		facades.Log().Warningf("tenant_ops skip: lock held tenant_id=%d op=%s", args.TenantID, op)
		return nil
	}
	defer lock.Release()

	now := time.Now()
	_, _ = appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
		"last_op":         op,
		"last_op_status":  models.TenantOpStatusRunning,
		"last_op_message": "",
		"last_op_at":      now,
	})
	tenant.LastOp = op
	tenant.LastOpStatus = models.TenantOpStatusRunning
	tenant.LastOpAt = &now
	AppendTenantOpLog(tenant, op, models.TenantOpStatusRunning, "", &now, nil)

	var runErr error
	var successMsg string
	migrateSucceeded := false
	switch op {
	case models.TenantOpMigrate:
		runErr = svc.conn.MigrateTenant(tenant)
		migrateSucceeded = runErr == nil
		if migrateSucceeded && args.WithSeed {
			if seedErr := svc.conn.SeedTenant(tenant); seedErr != nil {
				runErr = seedErr
			} else {
				successMsg = "migrate+seed ok"
			}
		} else if migrateSucceeded {
			successMsg = "migrate ok"
		}
	case models.TenantOpSeed:
		runErr = svc.conn.SeedTenant(tenant)
		if runErr == nil {
			successMsg = "seed ok"
		}
	case models.TenantOpBackup:
		path, err := BackupTenant(tenant, args.Keep)
		if err != nil {
			runErr = err
		} else {
			_ = MarkTenantBackupResult(tenant, path)
			successMsg = "backup ok: " + path
		}
	case models.TenantOpRestore:
		absPath, err := ResolveTenantBackupAbsPath(tenant, args.BackupName)
		if err != nil {
			runErr = err
		} else {
			runErr = RestoreTenantFromFile(tenant, absPath)
			if runErr == nil {
				successMsg = "restore ok: " + args.BackupName
			}
		}
	default:
		runErr = apperrors.ErrInvalidArgument.WithMessage("unknown tenant op: " + op)
	}

	if runErr != nil {
		msg := runErr.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		failAt := time.Now()
		updates := map[string]any{
			"last_op":         op,
			"last_op_status":  models.TenantOpStatusFailed,
			"last_op_message": msg,
			"last_op_at":      failAt,
		}
		if op == models.TenantOpMigrate && !migrateSucceeded &&
			tenant.ProvisionStatus == models.TenantProvisionMigrating {
			updates["provision_status"] = models.TenantProvisionFailed
			tenant.ProvisionStatus = models.TenantProvisionFailed
		}
		_, _ = appfacades.PlatformOrmQuery(nil).Model(tenant).Update(updates)
		tenant.LastOp = op
		tenant.LastOpStatus = models.TenantOpStatusFailed
		tenant.LastOpMessage = msg
		tenant.LastOpAt = &failAt
		AppendTenantOpLog(tenant, op, models.TenantOpStatusFailed, msg, &now, &failAt)
		return runErr
	}

	okAt := time.Now()
	_, _ = appfacades.PlatformOrmQuery(nil).Model(tenant).Update(map[string]any{
		"last_op":         op,
		"last_op_status":  models.TenantOpStatusSuccess,
		"last_op_message": successMsg,
		"last_op_at":      okAt,
	})
	tenant.LastOp = op
	tenant.LastOpStatus = models.TenantOpStatusSuccess
	tenant.LastOpMessage = successMsg
	tenant.LastOpAt = &okAt
	AppendTenantOpLog(tenant, op, models.TenantOpStatusSuccess, successMsg, &now, &okAt)
	return nil
}

// AppendTenantOpLog writes a landlord history row (best-effort).
func AppendTenantOpLog(tenant *models.Tenant, op, status, message string, started, finished *time.Time) {
	if tenant == nil || tenant.ID == 0 || op == "" {
		return
	}
	if len(message) > 2000 {
		message = message[:2000]
	}
	row := models.TenantOpLog{
		TenantID:   tenant.ID,
		Code:       tenant.Code,
		Op:         op,
		Status:     status,
		Message:    message,
		StartedAt:  started,
		FinishedAt: finished,
	}
	_ = appfacades.PlatformOrmQuery(nil).Create(&row)
}

// ListTenantOpLogs returns recent op history for a tenant.
func ListTenantOpLogs(tenantID uint, limit int) ([]models.TenantOpLog, error) {
	if tenantID == 0 {
		return nil, apperrors.ErrInvalidArgument.WithMessage("tenant_id required")
	}
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	var rows []models.TenantOpLog
	err := appfacades.PlatformOrmQuery(nil).Model(&models.TenantOpLog{}).
		Where("tenant_id", tenantID).
		Order("id desc").
		Limit(limit).
		Find(&rows)
	return rows, err
}
