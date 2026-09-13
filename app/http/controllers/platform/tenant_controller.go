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
	rows := make([]map[string]any, 0, len(list))
	for i := range list {
		rows = append(rows, services.TenantToJSON(&list[i]))
	}
	return response.Paginate(ctx, rows, total, page, pageSize)
}

func (c *TenantController) Show(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"tenant": services.TenantToJSON(tenant)})
}

type tenantStoreBody struct {
	Code      string `json:"code" form:"code"`
	Name      string `json:"name" form:"name"`
	Driver    string `json:"driver" form:"driver"`
	Isolation string `json:"isolation" form:"isolation"`
	Database  string `json:"database" form:"database"`
	Schema    string `json:"schema" form:"schema"`
	Host      string `json:"host" form:"host"`
	Port      int    `json:"port" form:"port"`
	Username   string `json:"username" form:"username"`
	Password   string `json:"password" form:"password"`
	Migrate    *bool  `json:"migrate" form:"migrate"` // 若传 true 则拒绝，引导 CLI
	SkipCreate bool   `json:"skip_create" form:"skip_create"`
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
		Code:       body.Code,
		Name:       body.Name,
		Driver:     body.Driver,
		Isolation:  body.Isolation,
		Database:   body.Database,
		Schema:     body.Schema,
		Host:       body.Host,
		Port:       body.Port,
		Username:   body.Username,
		Password:   body.Password,
		Migrate:    false,
		SkipCreate: body.SkipCreate,
	})
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{"tenant": services.TenantToJSON(tenant)})
}

type tenantUpdateBody struct {
	Name     *string `json:"name" form:"name"`
	Host     *string `json:"host" form:"host"`
	Port     *int    `json:"port" form:"port"`
	Username *string `json:"username" form:"username"`
	Password *string `json:"password" form:"password"`
	Database *string `json:"database" form:"database"`
	Schema   *string `json:"schema" form:"schema"`
}

func (c *TenantController) Update(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantUpdateBody
	_ = ctx.Request().Bind(&body)
	tenant, err := c.service().UpdateConnection(id, services.TenantUpdateInput{
		Name:     body.Name,
		Host:     body.Host,
		Port:     body.Port,
		Username: body.Username,
		Password: body.Password,
		Database: body.Database,
		Schema:   body.Schema,
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

// OpsSummary returns provision/op status counts for the platform tenant console.
func (c *TenantController) OpsSummary(ctx http.Context) http.Response {
	sum, err := c.service().OpsSummary()
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, nil)
	}
	return response.Success(ctx, map[string]any{"summary": sum})
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
	queued := make([]map[string]any, 0)
	skipped := make([]map[string]any, 0)
	failed := make([]map[string]any, 0)
	for i := range tenants {
		tenant := &tenants[i]
		_, args, err := c.ops().BeginQueuedOp(tenant.ID, models.TenantOpMigrate, body.WithSeed)
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
			_ = c.ops().MarkOpFailed(tenant, err.Error())
			failed = append(failed, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": err.Error()})
			continue
		}
		if err := facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{{
			Type:  "string",
			Value: string(payload),
		}}).OnQueue("long-running").Dispatch(); err != nil {
			_ = c.ops().MarkOpFailed(tenant, err.Error())
			failed = append(failed, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": err.Error()})
			continue
		}
		queued = append(queued, map[string]any{"id": tenant.ID, "code": tenant.Code})
	}
	return response.Success(ctx, map[string]any{
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

func (c *TenantController) enqueueOp(ctx http.Context, op string, withSeed bool) http.Response {
	return c.enqueueOpWithOpts(ctx, op, withSeed, "", -1)
}

func (c *TenantController) enqueueOpWithOpts(ctx http.Context, op string, withSeed bool, backupName string, keep int) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var tenant *models.Tenant
	var args services.TenantOpsArgs
	var err error
	if op == models.TenantOpBackup {
		tenant, args, err = c.ops().BeginQueuedBackup(id, keep)
	} else {
		tenant, args, err = c.ops().BeginQueuedOpWithBackup(id, op, withSeed, backupName)
	}
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	payload, err := json.Marshal(args)
	if err != nil {
		_ = c.ops().MarkOpFailed(tenant, err.Error())
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	if err := facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{{
		Type:  "string",
		Value: string(payload),
	}}).OnQueue("long-running").Dispatch(); err != nil {
		_ = c.ops().MarkOpFailed(tenant, err.Error())
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, apperrors.ErrTenantOpQueueFailed.WithError(err), map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{
		"queued": true,
		"op":     op,
		"tenant": services.TenantToJSON(tenant),
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

// Destroy deletes tenant metadata; optional drop of tenant database/schema.
func (c *TenantController) Destroy(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	var body tenantDeleteBody
	_ = ctx.Request().Bind(&body)
	if err := c.service().DeleteTenant(id, body.ConfirmCode, body.DropDatabase); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"deleted": true, "id": id})
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
	return response.Success(ctx, map[string]any{
		"overview": overview,
		"tenant":   services.TenantToJSON(tenant),
	})
}

// OpLogs returns recent ops timeline for one tenant.
func (c *TenantController) OpLogs(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	if _, err := c.service().GetByID(id); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	limit := helpers.GetIntQuery(ctx, "limit", 30)
	rows, err := services.ListTenantOpLogs(id, limit)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusInternalServerError, err, map[string]any{"id": id})
	}
	return response.Success(ctx, map[string]any{"list": rows, "total": len(rows)})
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
		"backup_keep": facades.Config().GetInt("tenancy.backup_keep", 10),
		"resolver":    facades.Config().GetString("tenancy.resolver", "header"),
		"header":      facades.Config().GetString("tenancy.header", "X-Tenant-ID"),
		"queue":       queue,
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
	queued := make([]map[string]any, 0)
	skipped := make([]map[string]any, 0)
	failed := make([]map[string]any, 0)
	for i := range tenants {
		tenant := &tenants[i]
		var args services.TenantOpsArgs
		var beginErr error
		if op == models.TenantOpBackup {
			_, args, beginErr = c.ops().BeginQueuedBackup(tenant.ID, keep)
		} else {
			_, args, beginErr = c.ops().BeginQueuedOp(tenant.ID, op, body.WithSeed && op == models.TenantOpMigrate)
		}
		if beginErr != nil {
			skipped = append(skipped, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": beginErr.Error()})
			continue
		}
		payload, err := json.Marshal(args)
		if err != nil {
			_ = c.ops().MarkOpFailed(tenant, err.Error())
			failed = append(failed, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": err.Error()})
			continue
		}
		if err := facades.Queue().Job(&jobs.TenantOps{}, []queue.Arg{{
			Type:  "string",
			Value: string(payload),
		}}).OnQueue("long-running").Dispatch(); err != nil {
			_ = c.ops().MarkOpFailed(tenant, err.Error())
			failed = append(failed, map[string]any{"id": tenant.ID, "code": tenant.Code, "error": err.Error()})
			continue
		}
		queued = append(queued, map[string]any{"id": tenant.ID, "code": tenant.Code})
	}
	return response.Success(ctx, map[string]any{
		"op":            op,
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
		"id", "code", "name", "status", "provision_status", "driver", "isolation",
		"host", "port", "database", "schema", "last_op", "last_op_status", "last_backup_path",
	})
	for i := range list {
		t := &list[i]
		_ = w.Write([]string{
			strconv.FormatUint(uint64(t.ID), 10),
			t.Code,
			t.Name,
			strconv.FormatUint(uint64(t.Status), 10),
			t.ProvisionStatus,
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

