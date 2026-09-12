package platform

import (
	"encoding/json"
	"strings"

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

// Ping checks that the tenant database connection is reachable.
func (c *TenantController) Ping(ctx http.Context) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, err := c.service().GetByID(id)
	if err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusNotFound, err, map[string]any{"id": id})
	}
	conn := services.NewTenantConnectionService()
	if err := conn.Ping(tenant, 0); err != nil {
		return admin.HandleGeneratedServiceError(ctx, "tenant", http.StatusBadGateway, err, map[string]any{
			"id":   id,
			"code": tenant.Code,
		})
	}
	return response.Success(ctx, map[string]any{
		"ok":               true,
		"tenant":           services.TenantToJSON(tenant),
		"provision_status": tenant.ProvisionStatus,
	})
}

type tenantMigrateBody struct {
	WithSeed bool `json:"with_seed" form:"with_seed"`
}

func (c *TenantController) ops() *services.TenantOpsService {
	return services.NewTenantOpsService()
}

func (c *TenantController) enqueueOp(ctx http.Context, op string, withSeed bool) http.Response {
	id := helpers.GetUintRoute(ctx, "id")
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}
	tenant, args, err := c.ops().BeginQueuedOp(id, op, withSeed)
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
	return c.enqueueOp(ctx, models.TenantOpBackup, false)
}
