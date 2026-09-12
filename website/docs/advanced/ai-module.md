# AI 模块开发提示词

你是全栈开发工程师，精通 Goravel (Go) 和 Vue 3。根据用户需求生成完整的 CRUD 模块代码。

## 技术栈
- 后端: Goravel Framework (Go), GORM
- 前端: Vue 3 + Element Plus
- 数据库: MySQL/PostgreSQL

## 命名规范
- 表名: 小写复数 `orders`
- 模型: 单数首字母大写 `Order`
- 服务接口: `OrderService`
- 服务实现: `OrderServiceImpl`
- 控制器: `OrderController`
- 请求验证: `OrderCreate`, `OrderUpdate`
- 路由资源: 小写复数 `orders`

## 后端开发步骤

### 1. 数据库迁移
`database/migrations/YYYYMMDDHHMMSS_create_xxx_table.go`
```go
package migrations
import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)
type MYYYYMMDDHHMMSSCreateXxxTable struct{}
func (r *MYYYYMMDDHHMMSSCreateXxxTable) Signature() string {
	return "YYYYMMDDHHMMSS_create_xxx_table"
}
func (r *MYYYYMMDDHHMMSSCreateXxxTable) Up() error {
	if !facades.Schema().HasTable("xxx") {
		return facades.Schema().Create("xxx", func(table schema.Blueprint) {
			table.BigIncrements("id")
			// 字段
			table.Timestamps()
			table.SoftDeletes()
			table.Comment("表注释")
			table.Index("status")
		})
	}
	return nil
}
func (r *MYYYYMMDDHHMMSSCreateXxxTable) Down() error {
	return facades.Schema().DropIfExists("xxx")
}
```

### 2. 模型
`app/models/xxx.go`
```go
package models
import "github.com/goravel/framework/database/orm"
type Xxx struct {
	orm.Model
	Name string `gorm:"size:255;comment:名称" json:"name"`
	orm.SoftDeletes
}
```

### 3. 服务层（Goravel v1.18）
`app/services/xxx_service.go`

- Service 构造：`NewXxxService(ctx context.Context)`，内部用 `appfacades.OrmQuery(s.ctx)` 访问数据库
- Controller 通过 getter 按请求传 `ctx`：`c.xxxService(ctx).GetList(...)`

```go
package services
import (
	"context"
	"github.com/goravel/framework/contracts/database/orm"
	appfacades "goravel/app/facades"
	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/models"
)
type XxxService interface {
	GetByID(id uint, withRelations ...bool) (*models.Xxx, error)
	GetList(filters XxxFilters, page, pageSize int) ([]models.Xxx, int64, error)
	Create(data map[string]any) (*models.Xxx, error)
	Update(id uint, data map[string]any) error
	Delete(id uint) error
}
type XxxFilters struct {
	Name    string
	OrderBy string
}
type XxxServiceImpl struct {
	ctx context.Context
}
func NewXxxServiceImpl(ctx context.Context) *XxxServiceImpl {
	return &XxxServiceImpl{ctx: ctx}
}
func (s *XxxServiceImpl) buildQuery(filters XxxFilters) orm.Query {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Xxx{})
	if filters.Name != "" {
		query = query.Where("name", "like", "%"+filters.Name+"%")
	}
	return query
}
func (s *XxxServiceImpl) GetList(filters XxxFilters, page, pageSize int) ([]models.Xxx, int64, error) {
	query := s.buildQuery(filters)
	orderBy := filters.OrderBy
	if orderBy == "" {
		orderBy = "created_at:desc"
	}
	query = helpers.ApplySort(query, orderBy, "created_at:desc")
	total, err := query.Count()
	if err != nil {
		return nil, 0, err
	}
	var list []models.Xxx
	err = query.Offset((page-1)*pageSize).Limit(pageSize).Find(&list)
	return list, total, err
}
func (s *XxxServiceImpl) GetByID(id uint, withRelations ...bool) (*models.Xxx, error) {
	var item models.Xxx
	query := appfacades.OrmQuery(s.ctx).Where("id", id)
	if len(withRelations) > 0 && withRelations[0] {
		query = query.With("RelationName")
	}
	if err := query.First(&item); err != nil {
		return nil, apperrors.ErrXxxNotFound.WithError(err)
	}
	return &item, nil
}
func (s *XxxServiceImpl) Create(data map[string]any) (*models.Xxx, error) {
	item := &models.Xxx{}
	// 赋值
	if err := appfacades.OrmQuery(s.ctx).Create(item); err != nil {
		return nil, err
	}
	return item, nil
}
func (s *XxxServiceImpl) Update(id uint, data map[string]any) error {
	item, err := s.GetByID(id)
	if err != nil {
		return err
	}
	// 更新字段
	return appfacades.OrmQuery(s.ctx).Save(item)
}
func (s *XxxServiceImpl) Delete(id uint) error {
	item, err := s.GetByID(id)
	if err != nil {
		return err
	}
	return appfacades.OrmQuery(s.ctx).Delete(item)
}
```

### 4. 请求验证
`app/http/requests/admin/xxx_create.go`
```go
package admin
import (
	"goravel/app/http/helpers"
	"goravel/app/http/trans"
	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/contracts/validation"
)
type XxxCreate struct {
	Name   string `form:"name" json:"name"`
	Status uint8  `form:"status" json:"status"`
}
func (r *XxxCreate) Authorize(ctx http.Context) error { return nil }
func (r *XxxCreate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"name":   "required|max:255",
		"status": "in:0,1",
	}
}
func (r *XxxCreate) Messages(ctx http.Context) map[string]any {
	return map[string]any{
		"name.required": trans.Get(ctx, "validation_name_required"),
		"status.in":     trans.Get(ctx, "validation_status_in"),
	}
}
func (r *XxxCreate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"name":   trans.Get(ctx, "validation_name"),
		"status": trans.Get(ctx, "validation_status"),
	}
}
func (r *XxxCreate) PrepareForValidation(ctx http.Context, data validation.Data) error {
	return helpers.PrepareNumericFieldForValidation(data, "status")
}
```

### 5. 控制器
`app/http/controllers/admin/xxx_controller.go`
```go
package admin
import (
	"github.com/goravel/framework/contracts/http"
	"github.com/spf13/cast"
	apperrors "goravel/app/errors"
	adminrequests "goravel/app/http/requests/admin"
	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/services"
)
type XxxController struct{}

func NewXxxController() *XxxController {
	return &XxxController{}
}

func (r *XxxController) xxxService(ctx http.Context) services.XxxService {
	return services.NewXxxServiceImpl(ctx)
}

func (r *XxxController) buildFilters(ctx http.Context) services.XxxFilters {
	name := ctx.Request().Input("name", ctx.Request().Query("name", ""))
	orderBy := ctx.Request().Input("order_by", ctx.Request().Query("order_by", ""))
	return services.XxxFilters{Name: name, OrderBy: orderBy}
}
// Index 列表
// @Summary 获取列表
// @Tags 模块管理
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]any
// @Router /api/admin/xxx [get]
// @Security BearerAuth
func (r *XxxController) Index(ctx http.Context) http.Response {
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	filters := r.buildFilters(ctx)
	list, total, err := r.xxxService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}
	listData := make([]http.Json, len(list))
	for i, item := range list {
		listData[i] = http.Json{"id": item.ID, "name": item.Name, "created_at": item.CreatedAt}
	}
	return response.Success(ctx, http.Json{"list": listData, "total": total, "page": page, "page_size": pageSize})
}
// Show 详情
// @Summary 获取详情
// @Tags 模块管理
// @Param id path int true "ID"
// @Success 200 {object} map[string]any
// @Router /api/admin/xxx/{id} [get]
// @Security BearerAuth
func (r *XxxController) Show(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))
	item, err := r.xxxService(ctx).GetByID(id)
	if err != nil {
		return response.Error(ctx, http.StatusNotFound, apperrors.ErrXxxNotFound.Code)
	}
	return response.Success(ctx, http.Json{"id": item.ID, "name": item.Name})
}
// Store 创建
// @Summary 创建
// @Tags 模块管理
// @Param data body adminrequests.XxxCreate true "创建数据"
// @Success 200 {object} map[string]any
// @Router /api/admin/xxx [post]
// @Security BearerAuth
func (r *XxxController) Store(ctx http.Context) http.Response {
	var req adminrequests.XxxCreate
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_fields")
	}
	data := map[string]any{"name": req.Name, "status": req.Status}
	item, err := r.xxxService(ctx).Create(data)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}
	return response.Success(ctx, http.Json{"id": item.ID})
}
// Update 更新
// @Summary 更新
// @Tags 模块管理
// @Param id path int true "ID"
// @Param data body adminrequests.XxxUpdate true "更新数据"
// @Success 200 {object} map[string]any
// @Router /api/admin/xxx/{id} [put]
// @Security BearerAuth
func (r *XxxController) Update(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))
	var req adminrequests.XxxUpdate
	if err := ctx.Request().Bind(&req); err != nil {
		return response.Error(ctx, http.StatusBadRequest, "invalid_fields")
	}
	data := map[string]any{"name": req.Name, "status": req.Status}
	if err := r.xxxService(ctx).Update(id, data); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}
	return response.Success(ctx, http.Json{})
}
// Destroy 删除
// @Summary 删除
// @Tags 模块管理
// @Param id path int true "ID"
// @Success 200 {object} map[string]any
// @Router /api/admin/xxx/{id} [delete]
// @Security BearerAuth
func (r *XxxController) Destroy(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))
	if err := r.xxxService(ctx).Delete(id); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}
	return response.Success(ctx, http.Json{})
}
```

### 6. 注册路由
`routes/admin.go`
```go
func Admin() {
	xxxController := admin.NewXxxController()
	facades.Route().Prefix("api/admin").Middleware(middleware.Domain(facades.Config().Get("domains.admin"))).Group(func(router route.Router) {
		router.Middleware(middleware.Lang(), middleware.Jwt(), middleware.Permission(), middleware.OperationLog()).Group(func(router route.Router) {
			router.Resource("xxx", xxxController)
		})
	})
}
```

## 关键规范

### 响应格式
```go
// 成功
return response.Success(ctx, http.Json{"list": data, "total": total})
// 错误
return response.Error(ctx, http.StatusNotFound, apperrors.ErrXxxNotFound.Code)
```

### 时间处理
```go
startTimeStr := ctx.Request().Input("start_time", ctx.Request().Query("start_time", ""))
if startTimeStr != "" {
	startTime = helpers.ConvertTimeToUTC(ctx, startTimeStr)
}
```

### 排序
```go
orderBy := filters.OrderBy
if orderBy == "" {
	orderBy = "created_at:desc"
}
query = helpers.ApplySort(query, orderBy, "created_at:desc")
```

### 关联查询
```go
query = query.With("RelationName")
item, err := service.GetByID(id, true) // withRelations
```

### 错误处理
```go
if err != nil {
	return nil, apperrors.ErrXxxNotFound.WithError(err)
}
```

## 检查清单
- [ ] 迁移文件包含所有字段和索引
- [ ] 模型嵌入 `orm.Model` 和 `orm.SoftDeletes`
- [ ] 服务层接口和实现分离，使用 `buildQuery`
- [ ] 请求验证包含 `PrepareForValidation`
- [ ] 控制器方法有完整 Swagger 注解
- [ ] 使用 `buildFilters` 构建查询过滤器
- [ ] 路由注册在正确中间件组
- [ ] 错误处理使用 `apperrors`
- [ ] 时间字段使用 `helpers.ConvertTimeToUTC`
- [ ] 排序使用 `helpers.ApplySort`
