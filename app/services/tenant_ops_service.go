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
	"goravel/app/utils"
)

const tenantOpLockTTL = 30 * time.Minute

// TenantOpActor is the platform admin who triggered an op.
type TenantOpActor struct {
	ID   uint
	Name string
}

// TenantOpsArgs is the queue payload for platform tenant maintenance jobs.
type TenantOpsArgs struct {
	TenantID     uint   `json:"tenant_id"`
	Op           string `json:"op"` // migrate|seed|backup|restore|purge
	WithSeed     bool   `json:"with_seed,omitempty"`
	BackupName   string `json:"backup_name,omitempty"`
	Keep         int    `json:"keep,omitempty"`
	OpLogID      uint   `json:"op_log_id,omitempty"`
	BatchID      string `json:"batch_id,omitempty"`
	OperatorID   uint   `json:"operator_id,omitempty"`
	OperatorName string `json:"operator_name,omitempty"`
	// Purge (async after soft-delete): Code is required; flags select what to remove.
	Code         string `json:"code,omitempty"`
	PurgeObjects bool   `json:"purge_objects,omitempty"`
	PurgeBackups bool   `json:"purge_backups,omitempty"`
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

// NewTenantOpsBatchID returns a correlation id for multi-tenant enqueue.
func NewTenantOpsBatchID() string {
	return utils.GenerateULID()
}

// BeginQueuedOp marks the tenant as queued for op.
func (s *TenantOpsService) BeginQueuedOp(id uint, op string, withSeed bool, actor TenantOpActor, batchID string) (*models.Tenant, TenantOpsArgs, error) {
	return s.BeginQueuedOpWithBackup(id, op, withSeed, "", actor, batchID)
}

// BeginQueuedOpWithBackup is BeginQueuedOp plus optional backup file name (restore).
func (s *TenantOpsService) BeginQueuedOpWithBackup(id uint, op string, withSeed bool, backupName string, actor TenantOpActor, batchID string) (*models.Tenant, TenantOpsArgs, error) {
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

	logID := CreateTenantOpLog(tenant, op, models.TenantOpStatusQueued, "", batchID, actor, &now, nil)
	return tenant, TenantOpsArgs{
		TenantID:     id,
		Op:           op,
		WithSeed:     withSeed,
		BackupName:   strings.TrimSpace(backupName),
		OpLogID:      logID,
		BatchID:      strings.TrimSpace(batchID),
		OperatorID:   actor.ID,
		OperatorName: actor.Name,
	}, nil
}

// BeginQueuedBackup enqueues backup with optional keep override (negative = config default).
func (s *TenantOpsService) BeginQueuedBackup(id uint, keep int, actor TenantOpActor, batchID string) (*models.Tenant, TenantOpsArgs, error) {
	tenant, args, err := s.BeginQueuedOpWithBackup(id, models.TenantOpBackup, false, "", actor, batchID)
	if err != nil {
		return nil, TenantOpsArgs{}, err
	}
	args.Keep = keep
	return tenant, args, nil
}

func (s *TenantOpsService) MarkOpFailed(tenant *models.Tenant, msg string, opLogID uint) error {
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
		UpdateTenantOpLog(opLogID, tenant, tenant.LastOp, models.TenantOpStatusFailed, msg, &now)
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

// RunTenantOp executes migrate/seed/backup/restore/purge for a queued platform job.
func RunTenantOp(args TenantOpsArgs) error {
	op := strings.TrimSpace(args.Op)
	if op == models.TenantOpPurge {
		return runTenantPurgeOp(args)
	}
	if args.TenantID == 0 {
		return apperrors.ErrInvalidArgument.WithMessage("tenant_id required")
	}
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
	UpdateTenantOpLog(args.OpLogID, tenant, op, models.TenantOpStatusRunning, "", nil)

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
		UpdateTenantOpLog(args.OpLogID, tenant, op, models.TenantOpStatusFailed, msg, &failAt)
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
	UpdateTenantOpLog(args.OpLogID, tenant, op, models.TenantOpStatusSuccess, successMsg, &okAt)
	return nil
}

// runTenantPurgeOp clears object storage and/or local backups by tenant code (works after soft-delete).
func runTenantPurgeOp(args TenantOpsArgs) error {
	code := strings.TrimSpace(args.Code)
	tenant := &models.Tenant{Code: code}
	if args.TenantID > 0 {
		tenant.ID = args.TenantID
		var row models.Tenant
		if err := appfacades.PlatformOrmQuery(nil).WithTrashed().Where("id", args.TenantID).First(&row); err == nil {
			tenant = &row
			if code == "" {
				code = row.Code
			}
		}
	}
	if code == "" {
		return apperrors.ErrInvalidArgument.WithMessage("tenant code required for purge")
	}
	if !args.PurgeObjects && !args.PurgeBackups {
		return apperrors.ErrInvalidArgument.WithMessage("purge_objects or purge_backups required")
	}

	lockKey := fmt.Sprintf("platform:tenant_purge:%s", code)
	if args.TenantID > 0 {
		lockKey = fmt.Sprintf("platform:tenant_op:%d", args.TenantID)
	}
	lock := facades.Cache().Lock(lockKey, tenantOpLockTTL)
	if !lock.Get() {
		facades.Log().Warningf("tenant_ops purge skip: lock held code=%s", code)
		return nil
	}
	defer lock.Release()

	UpdateTenantOpLog(args.OpLogID, tenant, models.TenantOpPurge, models.TenantOpStatusRunning, "", nil)

	var parts []string
	var runErr error
	if args.PurgeObjects {
		if err := PurgeTenantObjectStorage(code); err != nil {
			runErr = err
		} else {
			parts = append(parts, "objects")
		}
	}
	if runErr == nil && args.PurgeBackups {
		if err := PurgeTenantLocalBackups(code); err != nil {
			runErr = err
		} else {
			parts = append(parts, "backups")
		}
	}

	if runErr != nil {
		msg := runErr.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		failAt := time.Now()
		UpdateTenantOpLog(args.OpLogID, tenant, models.TenantOpPurge, models.TenantOpStatusFailed, msg, &failAt)
		if tenant.ID > 0 {
			_, _ = appfacades.PlatformOrmQuery(nil).WithTrashed().Model(&models.Tenant{}).Where("id", tenant.ID).Update(map[string]any{
				"last_op":         models.TenantOpPurge,
				"last_op_status":  models.TenantOpStatusFailed,
				"last_op_message": msg,
				"last_op_at":      failAt,
			})
		}
		return runErr
	}

	successMsg := "purge ok: " + strings.Join(parts, "+")
	okAt := time.Now()
	UpdateTenantOpLog(args.OpLogID, tenant, models.TenantOpPurge, models.TenantOpStatusSuccess, successMsg, &okAt)
	if tenant.ID > 0 {
		_, _ = appfacades.PlatformOrmQuery(nil).WithTrashed().Model(&models.Tenant{}).Where("id", tenant.ID).Update(map[string]any{
			"last_op":         models.TenantOpPurge,
			"last_op_status":  models.TenantOpStatusSuccess,
			"last_op_message": successMsg,
			"last_op_at":      okAt,
		})
	}
	return nil
}

// BeginQueuedPurge records an op-log row for async file cleanup after tenant soft-delete.
func (s *TenantOpsService) BeginQueuedPurge(tenantID uint, code string, purgeObjects, purgeBackups bool, actor TenantOpActor) (TenantOpsArgs, error) {
	if err := s.requireEnabled(); err != nil {
		return TenantOpsArgs{}, err
	}
	code = strings.TrimSpace(code)
	if code == "" || tenantID == 0 {
		return TenantOpsArgs{}, apperrors.ErrInvalidArgument.WithMessage("tenant id and code required for purge")
	}
	if !purgeObjects && !purgeBackups {
		return TenantOpsArgs{}, apperrors.ErrInvalidArgument.WithMessage("purge_objects or purge_backups required")
	}
	tenant := &models.Tenant{Code: code}
	tenant.ID = tenantID
	var row models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).WithTrashed().Where("id", tenantID).First(&row); err == nil {
		tenant = &row
	}
	now := time.Now()
	logID := CreateTenantOpLog(tenant, models.TenantOpPurge, models.TenantOpStatusQueued, "", "", actor, &now, nil)
	return TenantOpsArgs{
		TenantID:     tenantID,
		Op:           models.TenantOpPurge,
		Code:         code,
		PurgeObjects: purgeObjects,
		PurgeBackups: purgeBackups,
		OpLogID:      logID,
		OperatorID:   actor.ID,
		OperatorName: actor.Name,
	}, nil
}

// CreateTenantOpLog inserts one landlord history row; returns id (0 on failure).
func CreateTenantOpLog(tenant *models.Tenant, op, status, message, batchID string, actor TenantOpActor, started, finished *time.Time) uint {
	if tenant == nil || tenant.ID == 0 || op == "" {
		return 0
	}
	if len(message) > 2000 {
		message = message[:2000]
	}
	row := models.TenantOpLog{
		TenantID:     tenant.ID,
		Code:         tenant.Code,
		Op:           op,
		Status:       status,
		Message:      message,
		BatchID:      strings.TrimSpace(batchID),
		OperatorID:   actor.ID,
		OperatorName: strings.TrimSpace(actor.Name),
		StartedAt:    started,
		FinishedAt:   finished,
	}
	if err := appfacades.PlatformOrmQuery(nil).Create(&row); err != nil {
		return 0
	}
	return row.ID
}

// UpdateTenantOpLog updates the same job row; falls back to latest open row when id is 0.
func UpdateTenantOpLog(id uint, tenant *models.Tenant, op, status, message string, finished *time.Time) {
	if len(message) > 2000 {
		message = message[:2000]
	}
	updates := map[string]any{
		"status":  status,
		"message": message,
	}
	if finished != nil {
		updates["finished_at"] = finished
	}
	q := appfacades.PlatformOrmQuery(nil).Model(&models.TenantOpLog{})
	if id > 0 {
		_, _ = q.Where("id", id).Update(updates)
		return
	}
	if tenant == nil || tenant.ID == 0 || op == "" {
		return
	}
	var row models.TenantOpLog
	err := appfacades.PlatformOrmQuery(nil).Model(&models.TenantOpLog{}).
		Where("tenant_id", tenant.ID).
		Where("op", op).
		Where("status IN ?", []string{models.TenantOpStatusQueued, models.TenantOpStatusRunning}).
		Order("id desc").
		First(&row)
	if err != nil || row.ID == 0 {
		_ = CreateTenantOpLog(tenant, op, status, message, "", TenantOpActor{}, nil, finished)
		return
	}
	_, _ = appfacades.PlatformOrmQuery(nil).Model(&models.TenantOpLog{}).Where("id", row.ID).Update(updates)
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

// TenantOpLogFilters for the global platform ops log list.
type TenantOpLogFilters struct {
	Code     string
	Op       string
	Status   string
	BatchID  string
	Operator string
}

// ListTenantOpLogsPaged returns a filtered platform-wide ops log page.
func ListTenantOpLogsPaged(filters TenantOpLogFilters, page, pageSize int) ([]models.TenantOpLog, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 15
	}
	if pageSize > 100 {
		pageSize = 100
	}
	query := appfacades.PlatformOrmQuery(nil).Model(&models.TenantOpLog{})
	if code := strings.TrimSpace(filters.Code); code != "" {
		query = query.Where("code LIKE ?", "%"+code+"%")
	}
	if op := strings.TrimSpace(filters.Op); op != "" {
		query = query.Where("op", op)
	}
	if st := strings.TrimSpace(filters.Status); st != "" {
		query = query.Where("status", st)
	}
	if batch := strings.TrimSpace(filters.BatchID); batch != "" {
		query = query.Where("batch_id", batch)
	}
	if opName := strings.TrimSpace(filters.Operator); opName != "" {
		query = query.Where("operator_name LIKE ?", "%"+opName+"%")
	}
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var rows []models.TenantOpLog
	err = query.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows)
	return rows, total, err
}

// TenantOpLogToJSON serializes a log row for API responses.
func TenantOpLogToJSON(row *models.TenantOpLog) map[string]any {
	if row == nil {
		return nil
	}
	return map[string]any{
		"id":            row.ID,
		"tenant_id":     row.TenantID,
		"code":          row.Code,
		"op":            row.Op,
		"status":        row.Status,
		"message":       row.Message,
		"batch_id":      row.BatchID,
		"operator_id":   row.OperatorID,
		"operator_name": row.OperatorName,
		"started_at":    row.StartedAt,
		"finished_at":   row.FinishedAt,
		"created_at":    row.CreatedAt,
		"updated_at":    row.UpdatedAt,
	}
}
