package platform

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/queue"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	"goravel/app/health"
	"goravel/app/http/controllers/admin"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/jobs"
	"goravel/app/models"
	"goravel/app/services"
)

type TenantController struct{}

func NewTenantController() *TenantController {
	return &TenantController{}
}

func (c *TenantController) service() *services.TenantAdminService {
	return services.NewTenantAdminService()
}

func (c *TenantController) Index(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.BuildTenantAdminFiltersFromHTTP(ctx)
	list, total, err := c.service().GetList(filters, page, pageSize)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	return response.Paginate(ctx, services.TenantsToJSONList(list), total, page, pageSize)
}

func (c *TenantController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByIDIncludingTrashed(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	meta := services.LoadTenantDomainListMeta([]uint{tenant.ID})
	return response.Success(ctx, map[string]any{"tenant": services.TenantToJSONWithDomain(tenant, meta[tenant.ID])})
}

type tenantStoreBody struct {
	Code              string `json:"code" form:"code"`
	Name              string `json:"name" form:"name"`
	Driver            string `json:"driver" form:"driver"`
	Isolation         string `json:"isolation" form:"isolation"`
	Database          string `json:"database" form:"database"`
	Schema            string `json:"schema" form:"schema"`
	Host              string `json:"host" form:"host"`
	Port              int    `json:"port" form:"port"`
	Username          string `json:"username" form:"username"`
	Password          string `json:"password" form:"password"`
	Migrate           *bool  `json:"migrate" form:"migrate"` // 若传 true 则拒绝，引导 CLI
	SkipCreate        bool   `json:"skip_create" form:"skip_create"`
	StorageLimitBytes *int64 `json:"storage_limit_bytes" form:"storage_limit_bytes"`
	StorageMode       string `json:"storage_mode" form:"storage_mode"`
	StorageDriver     string `json:"storage_driver" form:"storage_driver"`
	StorageKey        string `json:"storage_key" form:"storage_key"`
	StorageSecret     string `json:"storage_secret" form:"storage_secret"`
	StorageRegion     string `json:"storage_region" form:"storage_region"`
	StorageBucket     string `json:"storage_bucket" form:"storage_bucket"`
	StorageURL        string `json:"storage_url" form:"storage_url"`
	StorageEndpoint   string `json:"storage_endpoint" form:"storage_endpoint"`
	StorageUsePathStyle *bool `json:"storage_use_path_style" form:"storage_use_path_style"`
	StorageSSL        *bool  `json:"storage_ssl" form:"storage_ssl"`
}

func (c *TenantController) Store(ctx http.Context) http.Response {
	var body tenantStoreBody
	_ = ctx.Request().Bind(&body)
	if strings.TrimSpace(body.Code) == "" || strings.TrimSpace(body.Name) == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	if body.Migrate != nil && *body.Migrate {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrTenantMigrateViaCLI.Code)
	}
	tenant, err := c.service().Create(services.TenantCreateInput{
		Code:                body.Code,
		Name:                body.Name,
		Driver:              body.Driver,
		Isolation:           body.Isolation,
		Database:            body.Database,
		Schema:              body.Schema,
		Host:                body.Host,
		Port:                body.Port,
		Username:            body.Username,
		Password:            body.Password,
		Migrate:             false,
		SkipCreate:          body.SkipCreate,
		StorageLimitBytes:   body.StorageLimitBytes,
		StorageMode:         body.StorageMode,
		StorageDriver:       body.StorageDriver,
		StorageKey:          body.StorageKey,
		StorageSecret:       body.StorageSecret,
		StorageRegion:       body.StorageRegion,
		StorageBucket:       body.StorageBucket,
		StorageURL:          body.StorageURL,
		StorageEndpoint:     body.StorageEndpoint,
		StorageUsePathStyle: body.StorageUsePathStyle,
		StorageSSL:          body.StorageSSL,
	})
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{"tenant": services.TenantToJSON(tenant)})
}

type tenantUpdateBody struct {
	Name                *string `json:"name" form:"name"`
	Host                *string `json:"host" form:"host"`
	Port                *int    `json:"port" form:"port"`
	Username            *string `json:"username" form:"username"`
	Password            *string `json:"password" form:"password"`
	Database            *string `json:"database" form:"database"`
	Schema              *string `json:"schema" form:"schema"`
	StorageLimitBytes   *int64  `json:"storage_limit_bytes" form:"storage_limit_bytes"`
	StorageMode         *string `json:"storage_mode" form:"storage_mode"`
	StorageDriver       *string `json:"storage_driver" form:"storage_driver"`
	StorageKey          *string `json:"storage_key" form:"storage_key"`
	StorageSecret       *string `json:"storage_secret" form:"storage_secret"`
	StorageRegion       *string `json:"storage_region" form:"storage_region"`
	StorageBucket       *string `json:"storage_bucket" form:"storage_bucket"`
	StorageURL          *string `json:"storage_url" form:"storage_url"`
	StorageEndpoint     *string `json:"storage_endpoint" form:"storage_endpoint"`
	StorageUsePathStyle *bool   `json:"storage_use_path_style" form:"storage_use_path_style"`
	StorageSSL          *bool   `json:"storage_ssl" form:"storage_ssl"`
	ClearStorageSecret  *bool   `json:"clear_storage_secret" form:"clear_storage_secret"`
}

func (c *TenantController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantUpdateBody
	_ = ctx.Request().Bind(&body)
	tenant, err := c.service().UpdateConnection(id, services.TenantUpdateInput{
		Name:                body.Name,
		Host:                body.Host,
		Port:                body.Port,
		Username:            body.Username,
		Password:            body.Password,
		Database:            body.Database,
		Schema:              body.Schema,
		StorageLimitBytes:   body.StorageLimitBytes,
		StorageMode:         body.StorageMode,
		StorageDriver:       body.StorageDriver,
		StorageKey:          body.StorageKey,
		StorageSecret:       body.StorageSecret,
		StorageRegion:       body.StorageRegion,
		StorageBucket:       body.StorageBucket,
		StorageURL:          body.StorageURL,
		StorageEndpoint:     body.StorageEndpoint,
		StorageUsePathStyle: body.StorageUsePathStyle,
		StorageSSL:          body.StorageSSL,
		ClearStorageSecret:  body.ClearStorageSecret,
	})
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"tenant": services.TenantToJSON(tenant)})
}

type tenantStatusBody struct {
	Status *uint8 `json:"status" form:"status"`
}

func (c *TenantController) UpdateStatus(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantStatusBody
	_ = ctx.Request().Bind(&body)
	if body.Status == nil || (*body.Status != models.TenantStatusActive && *body.Status != models.TenantStatusDisabled) {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	tenant, err := c.service().SetStatus(id, *body.Status)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"tenant": services.TenantToJSON(tenant)})
}

type tenantMaintenanceBody struct {
	Maintenance        *bool  `json:"maintenance" form:"maintenance"`
	MaintenanceMessage string `json:"maintenance_message" form:"maintenance_message"`
}

// UpdateMaintenance toggles per-tenant maintenance mode.
func (c *TenantController) UpdateMaintenance(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantMaintenanceBody
	_ = ctx.Request().Bind(&body)
	if body.Maintenance == nil {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	tenant, err := c.service().SetMaintenance(id, *body.Maintenance, body.MaintenanceMessage)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"tenant": services.TenantToJSON(tenant)})
}

// OpsSummary returns provision/op status counts for the platform tenant console.
func (c *TenantController) OpsSummary(ctx http.Context) http.Response {
	sum, err := c.service().OpsSummary()
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{"summary": sum})
}

// OpsOverview aggregates health-style fields + ops summary for the ops home page.
func (c *TenantController) OpsOverview(ctx http.Context) http.Response {
	sum, err := c.service().OpsSummary()
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{
		"app_version":              facades.Config().GetString("app.version", ""),
		"expected_migration_count": services.ExpectedSchemaMigrationCount(),
		"summary":                  sum,
		"queue":                    services.BuildPlatformQueueStatus(),
		"backup_keep":              facades.Config().GetInt("tenancy.backup_keep", 10),
		"alerts": map[string]any{
			"tenant_ops":    services.TenantOpsAlertConfigStatus(),
			"tenant_health": services.TenantHealthAlertConfigStatus(),
			"queue":         health.QueueAlertConfigStatus(),
		},
		"deploy_tips": []string{
			"Release cutover: run NEW image CLI migrate + tenant:migrate-all before switching traffic",
			"Platform UI migrate uses the currently running binary only",
		},
	})
}

type tenantOnboardBody struct {
	Code                string `json:"code" form:"code"`
	Name                string `json:"name" form:"name"`
	Driver              string `json:"driver" form:"driver"`
	Isolation           string `json:"isolation" form:"isolation"`
	Database            string `json:"database" form:"database"`
	Schema              string `json:"schema" form:"schema"`
	Host                string `json:"host" form:"host"`
	Port                int    `json:"port" form:"port"`
	Username            string `json:"username" form:"username"`
	Password            string `json:"password" form:"password"`
	SkipCreate          bool   `json:"skip_create" form:"skip_create"`
	StorageLimitBytes   *int64 `json:"storage_limit_bytes" form:"storage_limit_bytes"`
	StorageMode         string `json:"storage_mode" form:"storage_mode"`
	StorageDriver       string `json:"storage_driver" form:"storage_driver"`
	StorageKey          string `json:"storage_key" form:"storage_key"`
	StorageSecret       string `json:"storage_secret" form:"storage_secret"`
	StorageRegion       string `json:"storage_region" form:"storage_region"`
	StorageBucket       string `json:"storage_bucket" form:"storage_bucket"`
	StorageURL          string `json:"storage_url" form:"storage_url"`
	StorageEndpoint     string `json:"storage_endpoint" form:"storage_endpoint"`
	StorageUsePathStyle *bool  `json:"storage_use_path_style" form:"storage_use_path_style"`
	StorageSSL          *bool  `json:"storage_ssl" form:"storage_ssl"`
	WithMigrate         *bool  `json:"with_migrate" form:"with_migrate"`
	WithSeed            *bool  `json:"with_seed" form:"with_seed"`
	DomainHost          string `json:"domain_host" form:"domain_host"`
	DomainSSL           string `json:"domain_ssl_mode" form:"domain_ssl_mode"`
}

// Onboard creates a tenant, optionally queues migrate(+seed), optionally binds a domain, returns login links.
func (c *TenantController) Onboard(ctx http.Context) http.Response {
	var body tenantOnboardBody
	_ = ctx.Request().Bind(&body)
	if strings.TrimSpace(body.Code) == "" || strings.TrimSpace(body.Name) == "" {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	withMigrate := true
	if body.WithMigrate != nil {
		withMigrate = *body.WithMigrate
	}
	withSeed := true
	if body.WithSeed != nil {
		withSeed = *body.WithSeed
	}
	result, err := services.PrepareTenantOnboard(services.TenantOnboardInput{
		Create: services.TenantCreateInput{
			Code:                body.Code,
			Name:                body.Name,
			Driver:              body.Driver,
			Isolation:           body.Isolation,
			Database:            body.Database,
			Schema:              body.Schema,
			Host:                body.Host,
			Port:                body.Port,
			Username:            body.Username,
			Password:            body.Password,
			SkipCreate:          body.SkipCreate,
			StorageLimitBytes:   body.StorageLimitBytes,
			StorageMode:         body.StorageMode,
			StorageDriver:       body.StorageDriver,
			StorageKey:          body.StorageKey,
			StorageSecret:       body.StorageSecret,
			StorageRegion:       body.StorageRegion,
			StorageBucket:       body.StorageBucket,
			StorageURL:          body.StorageURL,
			StorageEndpoint:     body.StorageEndpoint,
			StorageUsePathStyle: body.StorageUsePathStyle,
			StorageSSL:          body.StorageSSL,
		},
		WithMigrate: withMigrate,
		WithSeed:    withSeed,
		DomainHost:  body.DomainHost,
		DomainSSL:   body.DomainSSL,
		Actor:       platformActor(ctx),
	})
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	if result.QueuedArgs != nil {
		payload, merr := json.Marshal(result.QueuedArgs)
		if merr != nil {
			_ = c.ops().MarkOpFailed(result.Tenant, merr.Error(), result.OpLogID)
			return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, merr, nil)
		}
		if qerr := facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{{
			Type:  "string",
			Value: string(payload),
		}}).OnQueue("long-running").Dispatch(); qerr != nil {
			_ = c.ops().MarkOpFailed(result.Tenant, qerr.Error(), result.OpLogID)
			return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, apperrors.ErrTenantOpQueueFailed.WithError(qerr), nil)
		}
	}
	out := map[string]any{
		"tenant":      services.TenantToJSON(result.Tenant),
		"queued":      result.Queued,
		"op":          result.Op,
		"op_log_id":   result.OpLogID,
		"steps":       result.Steps,
		"login_links": result.LoginLinks,
	}
	if result.Domain != nil {
		out["domain"] = services.TenantDomainToJSON(result.Domain)
	}
	if result.DomainError != nil {
		out["domain_error"] = result.DomainError.Error()
	}
	return response.Success(ctx, out)
}

type tenantHealthInspectBody struct {
	Limit int  `json:"limit" form:"limit"`
	Alert *bool `json:"alert" form:"alert"`
}

// HealthInspect runs a one-shot tenant health scan (owner only).
func (c *TenantController) HealthInspect(ctx http.Context) http.Response {
	var body tenantHealthInspectBody
	_ = ctx.Request().Bind(&body)
	alert := true
	if body.Alert != nil {
		alert = *body.Alert
	}
	report, err := services.RunTenantHealthInspect(body.Limit, alert)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{"report": report})
}

type tenantMigrateBatchBody struct {
	IDs             []uint `json:"ids" form:"ids"`
	ProvisionStatus string `json:"provision_status" form:"provision_status"`
	WithSeed        bool   `json:"with_seed" form:"with_seed"`
	Limit           int    `json:"limit" form:"limit"`
}

// MigrateBatch enqueues migrate for multiple tenants (e.g. all failed).
func (c *TenantController) MigrateBatch(ctx http.Context) http.Response {
	var body tenantMigrateBatchBody
	_ = ctx.Request().Bind(&body)
	if len(body.IDs) == 0 && strings.TrimSpace(body.ProvisionStatus) == "" {
		body.ProvisionStatus = models.TenantProvisionFailed
	}
	tenants, err := c.service().ListForBatchMigrate(body.IDs, body.ProvisionStatus, body.Limit)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	actor := platformActor(ctx)
	batchID := services.NewTenantOpsBatchID()
	queued := make([]map[string]any, 0)
	skipped := make([]map[string]any, 0)
	failed := make([]map[string]any, 0)
	for i := range tenants {
		tenant := &tenants[i]
		_, args, err := c.ops().BeginQueuedOp(tenant.ID, models.TenantOpMigrate, body.WithSeed, actor, batchID)
		if err != nil {
			skipped = append(skipped, map[string]any{
				"id":    tenant.ID,
				"code":  tenant.Code,
				"error": err.Error(),
			})
			continue
		}
		payload, err := json.Marshal(args)
		if err != nil {
			_ = c.ops().MarkOpFailed(tenant, err.Error(), args.OpLogID)
			failed = append(failed, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": err.Error()})
			continue
		}
		if err := facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{{
			Type:  "string",
			Value: string(payload),
		}}).OnQueue("long-running").Dispatch(); err != nil {
			_ = c.ops().MarkOpFailed(tenant, err.Error(), args.OpLogID)
			failed = append(failed, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": err.Error()})
			continue
		}
		queued = append(queued, map[string]any{"id": tenant.ID, "code": tenant.Code})
	}
	return response.Success(ctx, map[string]any{
		"batch_id":      batchID,
		"queued_count":  len(queued),
		"skipped_count": len(skipped),
		"failed_count":  len(failed),
		"queued":        queued,
		"skipped":       skipped,
		"failed":        failed,
	})
}

// Ping checks that the tenant database connection is reachable (with latency).
func (c *TenantController) Ping(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	detail := services.NewTenantConnectionService().BuildPingDetail(tenant)
	payload := map[string]any{
		"ok":               detail.OK,
		"ping":             detail,
		"tenant":           services.TenantToJSON(tenant),
		"provision_status": tenant.ProvisionStatus,
	}
	if !detail.OK {
		return response.Success(ctx, payload) // still return detail for UI
	}
	return response.Success(ctx, payload)
}

type tenantMigrateBody struct {
	WithSeed bool `json:"with_seed" form:"with_seed"`
}

type tenantBackupBody struct {
	Keep *int `json:"keep" form:"keep"` // nil => config default (-1)
}

type tenantRestoreBody struct {
	BackupName string `json:"backup_name" form:"backup_name"`
	Name       string `json:"name" form:"name"`
}

type tenantDeleteBody struct {
	ConfirmCode  string `json:"confirm_code" form:"confirm_code"`
	DropDatabase bool   `json:"drop_database" form:"drop_database"`
	PurgeObjects bool   `json:"purge_objects" form:"purge_objects"`
	PurgeBackups bool   `json:"purge_backups" form:"purge_backups"`
	// PurgeFiles is a legacy alias: when true, enables both purge_objects and purge_backups.
	PurgeFiles bool `json:"purge_files" form:"purge_files"`
}

type tenantForceDeleteBody struct {
	ConfirmCode  string `json:"confirm_code" form:"confirm_code"`
	PurgeObjects *bool  `json:"purge_objects" form:"purge_objects"` // nil => true
	PurgeBackups *bool  `json:"purge_backups" form:"purge_backups"` // nil => true
	PurgeFiles   bool   `json:"purge_files" form:"purge_files"`
}

type tenantPurgeBody struct {
	PurgeObjects bool `json:"purge_objects" form:"purge_objects"`
	PurgeBackups bool `json:"purge_backups" form:"purge_backups"`
	PurgeFiles   bool `json:"purge_files" form:"purge_files"`
}

type tenantOpsBatchBody struct {
	Op              string `json:"op" form:"op"` // migrate|seed|backup
	IDs             []uint `json:"ids" form:"ids"`
	ProvisionStatus string `json:"provision_status" form:"provision_status"`
	Status          *uint8 `json:"status" form:"status"`
	WithSeed        bool   `json:"with_seed" form:"with_seed"`
	Limit           int    `json:"limit" form:"limit"`
	Keep            *int   `json:"keep" form:"keep"`
}

type tenantPruneBody struct {
	Keep *int `json:"keep" form:"keep"`
}

func (c *TenantController) ops() *services.TenantOpsService {
	return services.NewTenantOpsService()
}

func platformActor(ctx http.Context) services.TenantOpActor {
	adminUser, _ := ctx.Value("platform_admin").(models.PlatformAdmin)
	name := strings.TrimSpace(adminUser.Name)
	if name == "" {
		name = strings.TrimSpace(adminUser.Username)
	}
	return services.TenantOpActor{ID: adminUser.ID, Name: name}
}

func (c *TenantController) enqueueOp(ctx http.Context, op string, withSeed bool) http.Response {
	return c.enqueueOpWithOpts(ctx, op, withSeed, "", -1)
}

func (c *TenantController) enqueueOpWithOpts(ctx http.Context, op string, withSeed bool, backupName string, keep int) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	actor := platformActor(ctx)
	var tenant *models.Tenant
	var args services.TenantOpsArgs
	var err error
	if op == models.TenantOpBackup {
		tenant, args, err = c.ops().BeginQueuedBackup(id, keep, actor, "")
	} else {
		tenant, args, err = c.ops().BeginQueuedOpWithBackup(id, op, withSeed, backupName, actor, "")
	}
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	payload, err := json.Marshal(args)
	if err != nil {
		_ = c.ops().MarkOpFailed(tenant, err.Error(), args.OpLogID)
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	if err := facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{{
		Type:  "string",
		Value: string(payload),
	}}).OnQueue("long-running").Dispatch(); err != nil {
		_ = c.ops().MarkOpFailed(tenant, err.Error(), args.OpLogID)
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, apperrors.ErrTenantOpQueueFailed.WithError(err), map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{
		"queued":    true,
		"op":        op,
		"op_log_id": args.OpLogID,
		"tenant":    services.TenantToJSON(tenant),
	})
}

// Migrate enqueues async migrate (optional seed) for one tenant.
func (c *TenantController) Migrate(ctx http.Context) http.Response {
	var body tenantMigrateBody
	_ = ctx.Request().Bind(&body)
	return c.enqueueOp(ctx, models.TenantOpMigrate, body.WithSeed)
}

// Seed enqueues async db:seed for one tenant.
func (c *TenantController) Seed(ctx http.Context) http.Response {
	return c.enqueueOp(ctx, models.TenantOpSeed, false)
}

// Backup enqueues async mysqldump/pg_dump for one tenant.
func (c *TenantController) Backup(ctx http.Context) http.Response {
	var body tenantBackupBody
	_ = ctx.Request().Bind(&body)
	keep := -1
	if body.Keep != nil {
		keep = *body.Keep
	}
	return c.enqueueOpWithOpts(ctx, models.TenantOpBackup, false, "", keep)
}

// Restore enqueues async restore from a backup file name under the tenant backup dir.
func (c *TenantController) Restore(ctx http.Context) http.Response {
	var body tenantRestoreBody
	_ = ctx.Request().Bind(&body)
	name := strings.TrimSpace(body.BackupName)
	if name == "" {
		name = strings.TrimSpace(body.Name)
	}
	return c.enqueueOpWithOpts(ctx, models.TenantOpRestore, false, name, -1)
}

// Destroy soft-deletes tenant metadata; optional DROP DB. Object storage is NOT purged here
// (purge happens on force-delete / recycle-bin purge).
func (c *TenantController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantDeleteBody
	_ = ctx.Request().Bind(&body)
	result, err := c.service().DeleteTenant(id, body.ConfirmCode, services.TenantDeleteOptions{
		DropDatabase: body.DropDatabase,
	})
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}

	return response.Success(ctx, map[string]any{
		"deleted":       true,
		"id":            result.ID,
		"code":          result.Code,
		"drop_database": result.DropDatabase,
	})
}

// Undelete restores soft-deleted tenant metadata from the recycle bin.
func (c *TenantController) Undelete(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().UndeleteTenant(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"tenant": services.TenantToJSON(tenant)})
}

// ForceDestroy permanently removes a soft-deleted tenant. By default it enqueues purge of
// object storage + local backups, then hard-deletes the landlord row after purge succeeds.
func (c *TenantController) ForceDestroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantForceDeleteBody
	_ = ctx.Request().Bind(&body)
	tenant, err := c.service().GetByIDIncludingTrashed(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	if !services.TenantIsTrashed(tenant) {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusBadRequest, apperrors.ErrTenantNotTrashed, map[string]any{"id": id})
	}
	if strings.TrimSpace(body.ConfirmCode) != tenant.Code {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusBadRequest, apperrors.ErrInvalidArgument.WithMessage("confirm_code must equal tenant code"), map[string]any{"id": id})
	}

	purgeObjects := true
	purgeBackups := true
	if body.PurgeObjects != nil {
		purgeObjects = *body.PurgeObjects
	}
	if body.PurgeBackups != nil {
		purgeBackups = *body.PurgeBackups
	}
	if body.PurgeFiles {
		purgeObjects = true
		purgeBackups = true
	}

	if purgeObjects || purgeBackups {
		purgeQueued, resp := c.enqueuePurge(ctx, tenant.ID, tenant.Code, purgeObjects, purgeBackups, true)
		if resp != nil {
			return resp
		}
		return response.Success(ctx, map[string]any{
			"force_delete_queued": true,
			"purge_queued":        purgeQueued,
			"id":                  tenant.ID,
			"code":                tenant.Code,
			"purge_objects":       purgeObjects,
			"purge_backups":       purgeBackups,
		})
	}

	if err := c.service().ForceDeleteTenant(id, body.ConfirmCode); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{
		"force_deleted": true,
		"id":            id,
		"purge_objects": false,
		"purge_backups": false,
	})
}

// Purge enqueues async object/backup cleanup (typically for soft-deleted tenants / retry).
func (c *TenantController) Purge(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByIDIncludingTrashed(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	var body tenantPurgeBody
	_ = ctx.Request().Bind(&body)
	purgeObjects := body.PurgeObjects || body.PurgeFiles
	purgeBackups := body.PurgeBackups || body.PurgeFiles
	if !purgeObjects && !purgeBackups {
		purgeObjects = true
		purgeBackups = true
	}
	purgeQueued, resp := c.enqueuePurge(ctx, tenant.ID, tenant.Code, purgeObjects, purgeBackups, false)
	if resp != nil {
		return resp
	}
	return response.Success(ctx, map[string]any{
		"purge_queued":  purgeQueued,
		"id":            tenant.ID,
		"code":          tenant.Code,
		"purge_objects": purgeObjects,
		"purge_backups": purgeBackups,
	})
}

// enqueuePurge returns (queued, errorResponse). errorResponse is non-nil on failure.
func (c *TenantController) enqueuePurge(ctx http.Context, id uint, code string, purgeObjects, purgeBackups, forceDeleteAfter bool) (bool, http.Response) {
	if !purgeObjects && !purgeBackups {
		return false, nil
	}
	args, beginErr := c.ops().BeginQueuedPurgeEx(id, code, purgeObjects, purgeBackups, forceDeleteAfter, platformActor(ctx))
	if beginErr != nil {
		return false, admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, beginErr, map[string]any{"id": id})
	}
	payload, marshalErr := json.Marshal(args)
	if marshalErr != nil {
		failAt := time.Now()
		stub := &models.Tenant{Code: code}
		stub.ID = id
		services.UpdateTenantOpLog(args.OpLogID, stub, models.TenantOpPurge, models.TenantOpStatusFailed, marshalErr.Error(), &failAt)
		services.MarkTenantOpFailedIncludingTrashed(id, models.TenantOpPurge, marshalErr.Error(), failAt)
		return false, admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, apperrors.ErrTenantOpQueueFailed.WithError(marshalErr), map[string]any{"id": id})
	}
	if err := facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{{
		Type:  "string",
		Value: string(payload),
	}}).OnQueue("long-running").Dispatch(); err != nil {
		failAt := time.Now()
		stub := &models.Tenant{Code: code}
		stub.ID = id
		services.UpdateTenantOpLog(args.OpLogID, stub, models.TenantOpPurge, models.TenantOpStatusFailed, err.Error(), &failAt)
		services.MarkTenantOpFailedIncludingTrashed(id, models.TenantOpPurge, err.Error(), failAt)
		return false, admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, apperrors.ErrTenantOpQueueFailed.WithError(err), map[string]any{"id": id})
	}
	return true, nil
}

// Overview returns DB size / table / admin counts for one tenant.
func (c *TenantController) Overview(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	overview := services.NewTenantConnectionService().BuildTenantOverview(tenant)
	quota := services.BuildTenantQuotaSnapshot(tenant)
	return response.Success(ctx, map[string]any{
		"overview": overview,
		"quota":    quota,
		"tenant":   services.TenantToJSON(tenant),
	})
}

// OpLogs returns recent ops timeline for one tenant.
func (c *TenantController) OpLogs(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	if _, err := c.service().GetByIDIncludingTrashed(id); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	limit := helpers.GetIntQuery(ctx, "limit", 30)
	rows, err := services.ListTenantOpLogs(id, limit)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	list := make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, services.TenantOpLogToJSON(&rows[i]))
	}
	return response.Success(ctx, map[string]any{"list": list, "total": len(list)})
}

// OpLogsIndex returns a platform-wide tenant ops execution log.
func (c *TenantController) OpLogsIndex(ctx http.Context) http.Response {
	page, pageSize := helpers.PaginationFromQuery(ctx, helpers.PaginationLimits{})
	filters := services.TenantOpLogFilters{
		Code:     strings.TrimSpace(ctx.Request().Query("code", "")),
		Op:       strings.TrimSpace(ctx.Request().Query("op", "")),
		Status:   strings.TrimSpace(ctx.Request().Query("status", "")),
		BatchID:  strings.TrimSpace(ctx.Request().Query("batch_id", "")),
		Operator: strings.TrimSpace(ctx.Request().Query("operator", "")),
	}
	rows, total, err := services.ListTenantOpLogsPaged(filters, page, pageSize)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	list := make([]map[string]any, 0, len(rows))
	for i := range rows {
		list = append(list, services.TenantOpLogToJSON(&rows[i]))
	}
	return response.Paginate(ctx, list, total, page, pageSize)
}

// LoginLinks returns how to open the tenant admin UI.
func (c *TenantController) LoginLinks(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"links": services.BuildTenantLoginLinks(tenant)})
}

// PruneBackups keeps newest N dumps for one tenant.
func (c *TenantController) PruneBackups(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	var body tenantPruneBody
	_ = ctx.Request().Bind(&body)
	keep := 0
	if body.Keep != nil {
		keep = *body.Keep
	}
	removed, err := services.PruneTenantBackups(tenant, keep)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{
		"removed":     removed,
		"keep":        keep,
		"backup_keep": facades.Config().GetInt("tenancy.backup_keep", 10),
	})
}

// Settings returns platform tenancy ops settings visible in the console.
func (c *TenantController) Settings(ctx http.Context) http.Response {
	queue := services.BuildPlatformQueueStatus()
	return response.Success(ctx, map[string]any{
		"backup_keep":             facades.Config().GetInt("tenancy.backup_keep", 10),
		"deleted_retention_days":  facades.Config().GetInt("tenancy.deleted_retention_days", 30),
		"resolver":                facades.Config().GetString("tenancy.resolver", "header"),
		"header":                  facades.Config().GetString("tenancy.header", "X-Tenant-ID"),
		"queue":                   queue,
	})
}

// QueueStatus reports long-running queue visibility.
func (c *TenantController) QueueStatus(ctx http.Context) http.Response {
	return response.Success(ctx, map[string]any{"queue": services.BuildPlatformQueueStatus()})
}

// OpsBatch enqueues migrate|seed|backup for multiple tenants.
func (c *TenantController) OpsBatch(ctx http.Context) http.Response {
	var body tenantOpsBatchBody
	_ = ctx.Request().Bind(&body)
	op := strings.TrimSpace(body.Op)
	if op == "" {
		op = models.TenantOpMigrate
	}
	switch op {
	case models.TenantOpMigrate, models.TenantOpSeed, models.TenantOpBackup:
	default:
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrInvalidArgument.Code)
	}
	if len(body.IDs) == 0 && strings.TrimSpace(body.ProvisionStatus) == "" && body.Status == nil {
		if op == models.TenantOpMigrate {
			body.ProvisionStatus = models.TenantProvisionFailed
		} else {
			active := models.TenantStatusActive
			body.Status = &active
		}
	}
	tenants, err := c.service().ListForBatchOp(body.IDs, body.ProvisionStatus, body.Status, body.Limit)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	keep := -1
	if body.Keep != nil {
		keep = *body.Keep
	}
	actor := platformActor(ctx)
	batchID := services.NewTenantOpsBatchID()
	queued := make([]map[string]any, 0)
	skipped := make([]map[string]any, 0)
	failed := make([]map[string]any, 0)
	for i := range tenants {
		tenant := &tenants[i]
		var args services.TenantOpsArgs
		var beginErr error
		if op == models.TenantOpBackup {
			_, args, beginErr = c.ops().BeginQueuedBackup(tenant.ID, keep, actor, batchID)
		} else {
			_, args, beginErr = c.ops().BeginQueuedOp(tenant.ID, op, body.WithSeed && op == models.TenantOpMigrate, actor, batchID)
		}
		if beginErr != nil {
			skipped = append(skipped, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": beginErr.Error()})
			continue
		}
		payload, err := json.Marshal(args)
		if err != nil {
			_ = c.ops().MarkOpFailed(tenant, err.Error(), args.OpLogID)
			failed = append(failed, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": err.Error()})
			continue
		}
		if err := facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{{
			Type:  "string",
			Value: string(payload),
		}}).OnQueue("long-running").Dispatch(); err != nil {
			_ = c.ops().MarkOpFailed(tenant, err.Error(), args.OpLogID)
			failed = append(failed, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": err.Error()})
			continue
		}
		queued = append(queued, map[string]any{"id": tenant.ID, "code": tenant.Code, "op_log_id": args.OpLogID})
	}
	return response.Success(ctx, map[string]any{
		"op":            op,
		"batch_id":      batchID,
		"queued_count":  len(queued),
		"skipped_count": len(skipped),
		"failed_count":  len(failed),
		"queued":        queued,
		"skipped":       skipped,
		"failed":        failed,
	})
}

// Export downloads tenants matching current filters as CSV.
func (c *TenantController) Export(ctx http.Context) http.Response {
	filters := services.BuildTenantAdminFiltersFromHTTP(ctx)
	list, _, err := c.service().GetList(filters, 1, 5000)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{
		"id", "code", "name", "status", "provision_status", "schema_status", "schema_migration_count",
		"driver", "isolation",
		"host", "port", "database", "schema", "last_op", "last_op_status", "last_backup_path",
	})
	expected := services.ExpectedSchemaMigrationCount()
	for i := range list {
		t := &list[i]
		_ = w.Write([]string{
			strconv.FormatUint(uint64(t.ID), 10),
			t.Code,
			t.Name,
			strconv.FormatUint(uint64(t.Status), 10),
			t.ProvisionStatus,
			services.ResolveTenantSchemaStatus(t, expected),
			strconv.FormatInt(t.SchemaMigrationCount, 10),
			t.Driver,
			t.Isolation,
			t.Host,
			strconv.Itoa(t.Port),
			t.Database,
			t.Schema,
			t.LastOp,
			t.LastOpStatus,
			t.LastBackupPath,
		})
	}
	w.Flush()
	filename := "tenants_" + time.Now().Format("20060102_150405") + ".csv"
	return ctx.Response().
		Header("Content-Type", "text/csv; charset=utf-8").
		Header("Content-Disposition", "attachment; filename="+filename).
		String(http.StatusOK, buf.String())
}

// ListBackups lists SQL dump files under storage/backups/tenants/{code}/.
func (c *TenantController) ListBackups(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	files, dir, err := services.ListTenantBackups(tenant)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{
		"backup_dir":       dir,
		"last_backup_path": tenant.LastBackupPath,
		"backup_keep":      facades.Config().GetInt("tenancy.backup_keep", 10),
		"list":             files,
		"total":            len(files),
	})
}

// DownloadBackup streams a backup .sql file for the tenant.
func (c *TenantController) DownloadBackup(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	name := strings.TrimSpace(ctx.Request().Query("name", ""))
	if name == "" {
		name = strings.TrimSpace(ctx.Request().Input("name", ""))
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	absPath, err := services.ResolveTenantBackupAbsPath(tenant, name)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusBadRequest, err, map[string]any{"id": id, "name": name})
	}
	return ctx.Response().Download(absPath, filepath.Base(absPath))
}

