# Development guide

> Prefer the [Chinese development guide](/guide/development): it now leads with the **code-generator** path (Notice walkthrough), then keeps the hand-written guestbook as reference.

::: tip
Secondary development default = code generator into this repo + checklist. Do not add a separate docs section for it.
:::
以留言板为例，说明完整 CRUD（后端接口 + 前端页面）。

## 目录

- [概述](#概述)
- [新增模块硬编码检查清单](#新增模块硬编码检查清单)
- [后端开发](#后端开发)
  - [1. 数据库迁移](#_1-数据库迁移)
  - [2. 创建模型](#_2-创建模型)
  - [3. 创建服务层](#_3-创建服务层)
  - [4. 创建请求验证](#_4-创建请求验证)
  - [5. 创建控制器](#_5-创建控制器)
  - [6. 注册路由](#_6-注册路由)
  - [7. 运行迁移](#_7-运行迁移)
- [前端开发](#前端开发)
  - [1. 创建 API 客户端](#_1-创建-api-客户端)
  - [2. 创建列表页面](#_2-创建列表页面)
  - [3. 创建表单组件](#_3-创建表单组件)
  - [4. 注册路由](#_4-注册路由)
  - [5. 添加国际化文本](#_5-添加国际化文本)
- [测试验证](#测试验证)
- [总结](#总结)

---

## 概述

留言板模块功能需求：
- 留言列表（支持分页、搜索、筛选）
- 创建留言
- 编辑留言
- 删除留言
- 回复留言（可选）

数据库表结构：
- `guestbooks` 表：留言主表
- 字段：id, name, email, content, reply, status, ip, user_agent, created_at, updated_at, deleted_at

---

## 新增模块硬编码检查清单

新增后台模块前后，请按 [硬编码检查清单](/guide/hardcoded-checklist) 逐项检查，重点关注：

- 权限种子与 slug 一致性
- 操作日志标题映射与下拉可检索
- 中英文翻译完整性
- 模型 `json` 标签与前端字段风格一致（推荐 snake_case）
- 观测页/系统日志页等存在写死映射的页面配置

---

## 后端开发

### 1. 数据库迁移

创建迁移文件：`database/migrations/20250101000029_create_guestbooks_table.go`

```go
package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20250101000029CreateGuestbooksTable struct{}

func (r *M20250101000029CreateGuestbooksTable) Signature() string {
	return "20250101000029_create_guestbooks_table"
}

func (r *M20250101000029CreateGuestbooksTable) Up() error {
	if !facades.Schema().HasTable("guestbooks") {
		return facades.Schema().Create("guestbooks", func(table schema.Blueprint) {
			table.BigIncrements("id")
			table.String("name", 50).Comment("留言人姓名")
			table.String("email", 100).Comment("留言人邮箱")
			table.Text("content").Comment("留言内容")
			table.Text("reply").Nullable().Comment("回复内容")
			table.UnsignedTinyInteger("status").Default(1).Comment("状态：1-已审核，0-待审核")
			table.String("ip", 50).Nullable().Comment("IP地址")
			table.String("user_agent", 500).Nullable().Comment("用户代理")
			table.Timestamps()
			table.SoftDeletes()
			table.Comment("留言板表")
			
			// 索引
			table.Index("status")
			table.Index("created_at")
		})
	}
	return nil
}

func (r *M20250101000029CreateGuestbooksTable) Down() error {
	return facades.Schema().DropIfExists("guestbooks")
}
```

### 2. 创建模型

创建模型文件：`app/models/guestbook.go`

```go
package models

import (
	"github.com/goravel/framework/database/orm"
)

type Guestbook struct {
	orm.Model
	Name      string `gorm:"size:50;comment:留言人姓名" json:"name"`
	Email     string `gorm:"size:100;comment:留言人邮箱" json:"email"`
	Content   string `gorm:"type:text;comment:留言内容" json:"content"`
	Reply     string `gorm:"type:text;comment:回复内容" json:"reply"`
	Status    uint8  `gorm:"default:1;comment:状态 1:已审核 0:待审核" json:"status"`
	IP        string `gorm:"size:50;comment:IP地址" json:"ip"`
	UserAgent string `gorm:"size:500;comment:用户代理" json:"user_agent"`
	orm.SoftDeletes
}
```

### 3. 创建服务层

Goravel v1.18：`NewXxxService(ctx)` + `appfacades.OrmQuery(s.ctx)`；Controller 用 getter 传请求 `ctx`。

创建服务接口和实现：`app/services/guestbook_service.go`

```go
package services

import (
	"context"

	"github.com/goravel/framework/contracts/database/orm"

	appfacades "goravel/app/facades"
	"github.com/spf13/cast"

	apperrors "goravel/app/errors"
	"goravel/app/http/helpers"
	"goravel/app/models"
)

type GuestbookService interface {
	GetByID(id uint) (*models.Guestbook, error)
	GetList(filters GuestbookFilters, page, pageSize int) ([]models.Guestbook, int64, error)
	Create(data map[string]any) (*models.Guestbook, error)
	Update(id uint, data map[string]any) error
	Delete(id uint) error
	Reply(id uint, replyContent string) error
}

type GuestbookFilters struct {
	Name      string
	Email     string
	Status    string
	StartTime string
	EndTime   string
	OrderBy   string
}

type GuestbookServiceImpl struct {
	ctx context.Context
}

func NewGuestbookServiceImpl(ctx context.Context) *GuestbookServiceImpl {
	return &GuestbookServiceImpl{ctx: ctx}
}

func (s *GuestbookServiceImpl) GetByID(id uint) (*models.Guestbook, error) {
	var guestbook models.Guestbook
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).First(&guestbook); err != nil {
		return nil, apperrors.ErrNotFound.WithError(err)
	}
	return &guestbook, nil
}

func (s *GuestbookServiceImpl) buildQuery(filters GuestbookFilters) orm.Query {
	query := appfacades.OrmQuery(s.ctx).Model(&models.Guestbook{})

	if filters.Name != "" {
		query = query.Where("name LIKE ?", "%"+filters.Name+"%")
	}
	if filters.Email != "" {
		query = query.Where("email LIKE ?", "%"+filters.Email+"%")
	}
	if filters.Status != "" {
		query = query.Where("status", filters.Status)
	}
	if filters.StartTime != "" {
		query = query.Where("created_at >= ?", filters.StartTime)
	}
	if filters.EndTime != "" {
		query = query.Where("created_at <= ?", filters.EndTime)
	}

	return query
}

func (s *GuestbookServiceImpl) GetList(filters GuestbookFilters, page, pageSize int) ([]models.Guestbook, int64, error) {
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

	var guestbooks []models.Guestbook
	err = query.Offset((page-1)*pageSize).Limit(pageSize).Find(&guestbooks)
	if err != nil {
		return nil, 0, err
	}

	return guestbooks, total, nil
}

func (s *GuestbookServiceImpl) Create(data map[string]any) (*models.Guestbook, error) {
	var guestbook models.Guestbook
	if err := appfacades.OrmQuery(s.ctx).Create(&guestbook, data); err != nil {
		return nil, err
	}
	return &guestbook, nil
}

func (s *GuestbookServiceImpl) Update(id uint, data map[string]any) error {
	var guestbook models.Guestbook
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).First(&guestbook); err != nil {
		return apperrors.ErrNotFound.WithError(err)
	}
	return appfacades.OrmQuery(s.ctx).Save(&guestbook, data)
}

func (s *GuestbookServiceImpl) Delete(id uint) error {
	var guestbook models.Guestbook
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).First(&guestbook); err != nil {
		return apperrors.ErrNotFound.WithError(err)
	}
	_, err := appfacades.OrmQuery(s.ctx).Delete(&guestbook)
	return err
}

func (s *GuestbookServiceImpl) Reply(id uint, replyContent string) error {
	var guestbook models.Guestbook
	if err := appfacades.OrmQuery(s.ctx).Where("id", id).First(&guestbook); err != nil {
		return apperrors.ErrNotFound.WithError(err)
	}
	guestbook.Reply = replyContent
	guestbook.Status = 1 // 回复后自动审核通过
	return appfacades.OrmQuery(s.ctx).Save(&guestbook)
}
```

### 4. 创建请求验证

创建创建请求验证：`app/http/requests/admin/guestbook_create.go`

```go
package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type GuestbookCreate struct {
	Name    string `form:"name" json:"name"`
	Email   string `form:"email" json:"email"`
	Content string `form:"content" json:"content"`
	Status  uint8  `form:"status" json:"status"`
}

func (r *GuestbookCreate) Authorize(ctx http.Context) error {
	return nil
}

func (r *GuestbookCreate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"name":    "required|max:50",
		"email":   "required|email|max:100",
		"content": "required|min:5|max:2000",
		"status":  "in:0,1",
	}
}

func (r *GuestbookCreate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"name":    trans.Get(ctx, "validation.attributes.name"),
		"email":   trans.Get(ctx, "validation.attributes.email"),
		"content": trans.Get(ctx, "validation.attributes.content"),
		"status":  trans.Get(ctx, "validation.attributes.status"),
	}
}
```

校验文案统一维护在 `lang/cn/validation.json` 与 `lang/en/validation.json`，无需再实现 `Messages()`。

创建更新请求验证：`app/http/requests/admin/guestbook_update.go`

```go
package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type GuestbookUpdate struct {
	Name    string `form:"name" json:"name"`
	Email   string `form:"email" json:"email"`
	Content string `form:"content" json:"content"`
	Reply   string `form:"reply" json:"reply"`
	Status  uint8  `form:"status" json:"status"`
}

func (r *GuestbookUpdate) Authorize(ctx http.Context) error {
	return nil
}

func (r *GuestbookUpdate) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"name":    "max:50",
		"email":   "email|max:100",
		"content": "min:5|max:2000",
		"reply":   "max:2000",
		"status":  "in:0,1",
	}
}

func (r *GuestbookUpdate) Messages(ctx http.Context) map[string]any {
	return map[string]any{
		"name.max":     trans.Get(ctx, "validation_name_max"),
		"email.email":      trans.Get(ctx, "validation_email_format"),
		"content.min":  trans.Get(ctx, "validation_content_min"),
		"content.max":  trans.Get(ctx, "validation_content_max"),
		"reply.max":    trans.Get(ctx, "validation_reply_max"),
		"status.in":         trans.Get(ctx, "validation_status_in"),
	}
}

func (r *GuestbookUpdate) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"name":    trans.Get(ctx, "validation_name"),
		"email":   trans.Get(ctx, "validation_email"),
		"content": trans.Get(ctx, "validation_content"),
		"reply":   trans.Get(ctx, "validation_reply"),
		"status":  trans.Get(ctx, "validation_status"),
	}
}
```

创建回复请求验证：`app/http/requests/admin/guestbook_reply.go`

```go
package admin

import (
	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

type GuestbookReply struct {
	Reply string `form:"reply" json:"reply"`
}

func (r *GuestbookReply) Authorize(ctx http.Context) error {
	return nil
}

func (r *GuestbookReply) Rules(ctx http.Context) map[string]any {
	return map[string]any{
		"reply": "required|min:5|max:2000",
	}
}

func (r *GuestbookReply) Messages(ctx http.Context) map[string]any {
	return map[string]any{
		"reply.required": trans.Get(ctx, "validation_reply_required"),
		"reply.min":  trans.Get(ctx, "validation_reply_min"),
		"reply.max":  trans.Get(ctx, "validation_reply_max"),
	}
}

func (r *GuestbookReply) Attributes(ctx http.Context) map[string]any {
	return map[string]any{
		"reply": trans.Get(ctx, "validation_reply"),
	}
}
```

### 5. 创建控制器

创建控制器文件：`app/http/controllers/admin/guestbook_controller.go`

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

type GuestbookController struct{}

func NewGuestbookController() *GuestbookController {
	return &GuestbookController{}
}

func (r *GuestbookController) guestbookService(ctx http.Context) services.GuestbookService {
	return services.NewGuestbookServiceImpl(ctx)
}

// Index 留言列表
// @Summary      获取留言列表
// @Description  分页获取留言列表，支持按姓名、邮箱、状态等条件筛选
// @Tags         留言板管理
// @Accept       json
// @Produce      json
// @Param        page       query     int     false  "页码" default(1)
// @Param        page_size  query     int     false  "每页数量" default(20)
// @Param        name       query     string  false  "姓名（模糊搜索）"
// @Param        email      query     string  false  "邮箱（模糊搜索）"
// @Param        status     query     string  false  "状态：1-已审核，0-待审核"
// @Param        start_time query     string  false  "开始时间（格式：YYYY-MM-DD HH:mm:ss）"
// @Param        end_time   query     string  false  "结束时间（格式：YYYY-MM-DD HH:mm:ss）"
// @Param        order_by   query     string  false  "排序（格式：字段:asc/desc，如：created_at:desc）"
// @Success      200        {object}  map[string]any
// @Failure      400        {object}  map[string]any "参数错误"
// @Failure      401        {object}  map[string]any "未登录"
// @Failure      403        {object}  map[string]any "无权限"
// @Failure      500        {object}  map[string]any "服务器错误"
// @Router       /api/admin/guestbooks [get]
// @Security     BearerAuth
func (r *GuestbookController) Index(ctx http.Context) http.Response {
	page := cast.ToInt(ctx.Request().Query("page", "1"))
	pageSize := cast.ToInt(ctx.Request().Query("page_size", "20"))

	startTimeStr := ctx.Request().Query("start_time", "")
	endTimeStr := ctx.Request().Query("end_time", "")
	startTime := ""
	endTime := ""
	if startTimeStr != "" {
		startTime = helpers.ConvertTimeToUTC(ctx, startTimeStr)
	}
	if endTimeStr != "" {
		endTime = helpers.ConvertTimeToUTC(ctx, endTimeStr)
	}

	filters := services.GuestbookFilters{
		Name:      ctx.Request().Query("name", ""),
		Email:     ctx.Request().Query("email", ""),
		Status:    ctx.Request().Query("status", ""),
		StartTime: startTime,
		EndTime:   endTime,
		OrderBy:   ctx.Request().Query("order_by", ""),
	}

	guestbooks, total, err := r.guestbookService(ctx).GetList(filters, page, pageSize)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	return response.Success(ctx, http.Json{
		"list":      guestbooks,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Show 留言详情
// @Summary      获取留言详情
// @Description  根据ID获取留言详细信息
// @Tags         留言板管理
// @Accept       json
// @Produce      json
// @Param        id   path     int  true  "留言ID"
// @Success      200  {object} map[string]any
// @Failure      404  {object} map[string]any "留言不存在"
// @Router       /api/admin/guestbooks/{id} [get]
// @Security     BearerAuth
func (r *GuestbookController) Show(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))
	guestbook, err := r.guestbookService(ctx).GetByID(id)
	if err != nil {
		return response.Error(ctx, http.StatusNotFound, apperrors.ErrNotFound.Code)
	}

	return response.Success(ctx, http.Json{
		"guestbook": guestbook,
	})
}

// Store 创建留言
// @Summary      创建留言
// @Description  创建新的留言
// @Tags         留言板管理
// @Accept       json
// @Produce      json
// @Param        name     body     string  true  "留言人姓名"
// @Param        email    body     string  true  "留言人邮箱"
// @Param        content  body     string  true  "留言内容"
// @Param        status   body     int     false "状态：1-已审核，0-待审核" default(0)
// @Success      200      {object} map[string]any
// @Failure      400      {object} map[string]any "参数错误"
// @Router       /api/admin/guestbooks [post]
// @Security     BearerAuth
func (r *GuestbookController) Store(ctx http.Context) http.Response {
	var req adminrequests.GuestbookCreate
	errors, err := ctx.Request().ValidateRequest(&req)
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if errors != nil {
		return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", errors.All())
	}

	// 获取客户端IP和User-Agent
	ip := ctx.Request().Ip()
	userAgent := ctx.Request().Header("User-Agent", "")

	data := map[string]any{
		"name":       req.Name,
		"email":      req.Email,
		"content":    req.Content,
		"status":     req.Status,
		"ip":         ip,
		"user_agent": userAgent,
	}

	guestbook, err := r.guestbookService(ctx).Create(data)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	return response.Success(ctx, http.Json{
		"guestbook": guestbook,
	})
}

// Update 更新留言
// @Summary      更新留言
// @Description  更新留言信息，包括回复内容
// @Tags         留言板管理
// @Accept       json
// @Produce      json
// @Param        id       path     int     true  "留言ID"
// @Param        name     body     string  false "留言人姓名"
// @Param        email    body     string  false "留言人邮箱"
// @Param        content  body     string  false "留言内容"
// @Param        reply    body     string  false "回复内容"
// @Param        status   body     int     false "状态：1-已审核，0-待审核"
// @Success      200      {object} map[string]any
// @Failure      404      {object} map[string]any "留言不存在"
// @Router       /api/admin/guestbooks/{id} [put]
// @Security     BearerAuth
func (r *GuestbookController) Update(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))

	var req adminrequests.GuestbookUpdate
	errors, err := ctx.Request().ValidateRequest(&req)
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if errors != nil {
		return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", errors.All())
	}

	data := map[string]any{}
	if req.Name != "" {
		data["name"] = req.Name
	}
	if req.Email != "" {
		data["email"] = req.Email
	}
	if req.Content != "" {
		data["content"] = req.Content
	}
	if req.Reply != "" {
		data["reply"] = req.Reply
	}
	if ctx.Request().Input("status") != "" {
		data["status"] = req.Status
	}

	if err := r.guestbookService(ctx).Update(id, data); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	guestbook, _ := r.guestbookService(ctx).GetByID(id)
	return response.Success(ctx, http.Json{
		"guestbook": guestbook,
	})
}

// Destroy 删除留言
// @Summary      删除留言
// @Description  删除指定的留言
// @Tags         留言板管理
// @Accept       json
// @Produce      json
// @Param        id   path     int  true  "留言ID"
// @Success      200  {object} map[string]any
// @Failure      404  {object} map[string]any "留言不存在"
// @Router       /api/admin/guestbooks/{id} [delete]
// @Security     BearerAuth
func (r *GuestbookController) Destroy(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))

	if err := r.guestbookService(ctx).Delete(id); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	return response.Success(ctx)
}

// Reply 回复留言
// @Summary      回复留言
// @Description  对留言进行回复，回复后自动审核通过
// @Tags         留言板管理
// @Accept       json
// @Produce      json
// @Param        id     path     string  true  "留言ID"
// @Param        reply  body     string  true  "回复内容"
// @Success      200    {object} map[string]any
// @Failure      404    {object} map[string]any "留言不存在"
// @Router       /api/admin/guestbooks/{id}/reply [post]
// @Security     BearerAuth
func (r *GuestbookController) Reply(ctx http.Context) http.Response {
	id := cast.ToUint(ctx.Request().Route("id"))

	var req adminrequests.GuestbookReply
	errors, err := ctx.Request().ValidateRequest(&req)
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if errors != nil {
		return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", errors.All())
	}

	if err := r.guestbookService(ctx).Reply(id, req.Reply); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, err.Error())
	}

	guestbook, _ := r.guestbookService(ctx).GetByID(id)
	return response.Success(ctx, http.Json{
		"guestbook": guestbook,
	})
}
```

### 6. 注册路由

在 `routes/admin.go` 文件中添加路由：

```go
// 在 Admin() 函数中添加控制器实例
guestbookController := admin.NewGuestbookController()

// 在需要认证、权限验证的路由组中添加（约第73行）
router.Resource("guestbooks", guestbookController)
router.Post("guestbooks/{id}/reply", guestbookController.Reply) // 回复留言接口
```

### 7. 运行迁移

```bash
# 运行迁移
go run . artisan migrate

# linux使用
./main artisan migrate
```

---

## 前端开发

### 1. 创建 API 客户端

创建 API 文件：`html/src/api/guestbook.js`

```javascript
import request from '../utils/request'
import { createCRUDApi, extendApi } from '../utils/apiFactory'

// 创建基础 CRUD API
const baseGuestbookApi = createCRUDApi('guestbooks')

// 扩展 API，添加自定义方法
const guestbookApi = extendApi(baseGuestbookApi, {
  // 回复留言
  reply: (id, data) => {
    return request({
      url: `/guestbooks/${id}/reply`,
      method: 'post',
      data
    })
  }
})

// 导出所有方法
export const {
  list: getGuestbookList,
  detail: getGuestbookDetail,
  create: createGuestbook,
  update: updateGuestbook,
  delete: deleteGuestbook,
  reply: replyGuestbook
} = guestbookApi
```

### 2. 创建列表页面

创建列表页面：`html/src/views/guestbook/GuestbookList.vue`

```vue
<template>
  <div class="list-page">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ $t('menu.guestbook') }}</span>
          <el-button 
            type="primary" 
            :disabled="getButtonState('guestbook.store').disabled"
            @click="handleAdd"
          >
            <el-icon><PlusIcon /></el-icon>
            {{ $t('guestbook.add_guestbook') }}
          </el-button>
        </div>
      </template>

      <!-- 搜索表单 -->
      <SearchForm
        :model="searchForm"
        :fields="searchFields"
        :initial-values="initialSearchForm"
        i18n-prefix="guestbook"
        @search="handleSearch"
        @reset="handleReset"
      />

      <!-- 数据表格 -->
      <VxeTable
        ref="tableRef"
        :data="tableData"
        :loading="loading"
        :columns="tableColumns"
        :height="600"
        @sort-change="handleSortChange"
      >
        <template #status="{ row }">
          <el-tag :type="row.status === 1 ? 'success' : 'warning'">
            {{ row.status === 1 ? $t('guestbook.status_approved') : $t('guestbook.status_pending') }}
          </el-tag>
        </template>

        <template #content="{ row }">
          <div class="content-preview">
            {{ row.content && row.content.length > 50 ? row.content.substring(0, 50) + '...' : row.content }}
          </div>
        </template>

        <template #reply="{ row }">
          <div v-if="row.reply" class="reply-preview">
            {{ row.reply.length > 50 ? row.reply.substring(0, 50) + '...' : row.reply }}
          </div>
          <span v-else class="text-gray-400">-</span>
        </template>

        <template #operation="{ row }">
          <TableActionButtons
            :row="row"
            :primary-actions="getPrimaryActions(row)"
            :more-actions="getMoreActions(row)"
            :get-button-state="getButtonState"
            @action="handleAction"
          />
        </template>
      </VxeTable>

      <!-- 分页 -->
      <Pagination
        v-model="pagination"
        :auto-load="true"
        :on-page-change="loadData"
      />
    </el-card>

    <!-- 添加/编辑对话框 -->
    <GuestbookForm
      ref="guestbookFormRef"
      v-model="dialogVisible"
      :edit-id="editId"
      @success="handleFormSuccess"
    />
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, markRaw } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import SearchForm from '../../components/SearchForm.vue'
import Pagination from '../../components/Pagination.vue'
import VxeTable from '../../components/VxeTable.vue'
import TableActionButtons from '../../components/TableActionButtons.vue'
import GuestbookForm from './GuestbookForm.vue'
import { useListPage } from '../../composables/useListPage'
import { usePermission } from '../../composables/usePermission'
import { useCrud } from '../../composables/useCrud'
import {
  getGuestbookList,
  deleteGuestbook,
  updateGuestbook,
  replyGuestbook
} from '../../api/guestbook'
import logger from '../../utils/logger'
import ErrorHandler from '../../utils/errorHandler'

const PlusIcon = markRaw(Plus)

// 权限控制
const { getButtonState } = usePermission()

const { t } = useI18n()
const tableRef = ref(null)
const guestbookFormRef = ref(null)

// 使用组合式函数
const {
  dialogVisible,
  editId,
  handleAdd,
  handleEdit,
  handleDelete,
  handleFormSuccess
} = useCrud(guestbookFormRef)

const {
  loading,
  tableData,
  pagination,
  searchForm,
  initialSearchForm,
  searchFields,
  tableColumns,
  loadData,
  handleSearch,
  handleReset,
  handleSortChange
} = useListPage({
  api: getGuestbookList,
  searchFields: [
    {
      key: 'name',
      label: 'guestbook.name',
      type: 'input',
      placeholder: 'guestbook.name_placeholder'
    },
    {
      key: 'email',
      label: 'guestbook.email',
      type: 'input',
      placeholder: 'guestbook.email_placeholder'
    },
    {
      key: 'status',
      label: 'guestbook.status',
      type: 'select',
      options: [
        { label: 'guestbook.status_all', value: '' },
        { label: 'guestbook.status_approved', value: '1' },
        { label: 'guestbook.status_pending', value: '0' }
      ]
    },
    {
      key: 'start_time',
      label: 'common.start_time',
      type: 'datetime'
    },
    {
      key: 'end_time',
      label: 'common.end_time',
      type: 'datetime'
    }
  ],
  tableColumns: [
    { field: 'id', title: 'ID', width: 80 },
    { field: 'name', title: 'guestbook.name', width: 120 },
    { field: 'email', title: 'guestbook.email', width: 180 },
    { 
      field: 'content', 
      title: 'guestbook.content', 
      width: 300,
      slots: { default: 'content' }
    },
    { 
      field: 'reply', 
      title: 'guestbook.reply', 
      width: 200,
      slots: { default: 'reply' }
    },
    { 
      field: 'status', 
      title: 'guestbook.status', 
      width: 100,
      slots: { default: 'status' }
    },
    { field: 'ip', title: 'guestbook.ip', width: 130 },
    { field: 'created_at', title: 'common.created_at', width: 180 },
    { 
      field: 'operation', 
      title: 'common.operation', 
      width: 200,
      fixed: 'right',
      slots: { default: 'operation' }
    }
  ]
})

// 主要操作按钮
const getPrimaryActions = (row) => {
  return [
    {
      label: t('common.edit'),
      action: 'edit',
      permission: 'guestbook.update',
      icon: 'Edit'
    },
    {
      label: t('common.delete'),
      action: 'delete',
      permission: 'guestbook.destroy',
      icon: 'Delete',
      type: 'danger'
    }
  ]
}

// 更多操作按钮
const getMoreActions = (row) => {
  return [
    {
      label: t('guestbook.reply'),
      action: 'reply',
      permission: 'guestbook.reply',
      icon: 'ChatLineRound',
      show: !row.reply // 已有回复则不显示
    }
  ]
}

// 处理操作
const handleAction = async (action, row) => {
  switch (action) {
    case 'edit':
      handleEdit(row.id)
      break
    case 'delete':
      await handleDelete(row.id, deleteGuestbook, t('guestbook.guestbook'))
      break
    case 'reply':
      handleReply(row)
      break
  }
}

// 处理回复
const handleReply = async (row) => {
  try {
    const { value } = await ElMessageBox.prompt(
      t('guestbook.reply_content'),
      t('guestbook.reply'),
      {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        inputType: 'textarea',
        inputPlaceholder: t('guestbook.reply_placeholder'),
        inputValidator: (value) => {
          if (!value || value.trim().length < 5) {
            return t('validation_reply_min')
          }
          if (value.length > 2000) {
            return t('validation_reply_max')
          }
          return true
        }
      }
    )

    await replyGuestbook(row.id, { reply: value })
    ElMessage.success(t('guestbook.reply_success'))
    loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ErrorHandler.handle(error)
    }
  }
}

onMounted(() => {
  loadData()
})
</script>

<style scoped>
.content-preview,
.reply-preview {
  word-break: break-word;
  line-height: 1.5;
}
</style>
```

### 3. 创建表单组件

创建表单组件：`html/src/views/guestbook/GuestbookForm.vue`

```vue
<template>
  <el-dialog
    v-model="dialogVisible"
    :title="editId ? $t('guestbook.edit_guestbook') : $t('guestbook.add_guestbook')"
    width="800px"
    @close="handleClose"
  >
    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="100px"
    >
      <el-form-item :label="$t('guestbook.name')" prop="name">
        <el-input
          v-model="formData.name"
          :placeholder="$t('guestbook.name_placeholder')"
          maxlength="50"
          show-word-limit
        />
      </el-form-item>

      <el-form-item :label="$t('guestbook.email')" prop="email">
        <el-input
          v-model="formData.email"
          :placeholder="$t('guestbook.email_placeholder')"
          maxlength="100"
        />
      </el-form-item>

      <el-form-item :label="$t('guestbook.content')" prop="content">
        <el-input
          v-model="formData.content"
          type="textarea"
          :rows="6"
          :placeholder="$t('guestbook.content_placeholder')"
          maxlength="2000"
          show-word-limit
        />
      </el-form-item>

      <el-form-item v-if="editId" :label="$t('guestbook.reply')" prop="reply">
        <el-input
          v-model="formData.reply"
          type="textarea"
          :rows="4"
          :placeholder="$t('guestbook.reply_placeholder')"
          maxlength="2000"
          show-word-limit
        />
      </el-form-item>

      <el-form-item :label="$t('guestbook.status')" prop="status">
        <el-radio-group v-model="formData.status">
          <el-radio :label="1">{{ $t('guestbook.status_approved') }}</el-radio>
          <el-radio :label="0">{{ $t('guestbook.status_pending') }}</el-radio>
        </el-radio-group>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="handleClose">{{ $t('common.cancel') }}</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">
        {{ $t('common.confirm') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, reactive, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { getGuestbookDetail, createGuestbook, updateGuestbook } from '../../api/guestbook'
import ErrorHandler from '../../utils/errorHandler'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  editId: {
    type: [Number, String],
    default: null
  }
})

const emit = defineEmits(['update:modelValue', 'success'])

const { t } = useI18n()
const formRef = ref(null)
const submitting = ref(false)

const dialogVisible = computed({
  get: () => props.modelValue,
  set: (val) => emit('update:modelValue', val)
})

const formData = reactive({
  name: '',
  email: '',
  content: '',
  reply: '',
  status: 0
})

const formRules = {
  name: [
    { required: true, message: t('validation_name_required'), trigger: 'blur' },
    { max: 50, message: t('validation_name_max'), trigger: 'blur' }
  ],
  email: [
    { required: true, message: t('validation_email_required'), trigger: 'blur' },
    { type: 'email', message: t('validation_email_format'), trigger: 'blur' },
    { max: 100, message: t('validation_email_max'), trigger: 'blur' }
  ],
  content: [
    { required: true, message: t('validation_content_required'), trigger: 'blur' },
    { min: 5, message: t('validation_content_min'), trigger: 'blur' },
    { max: 2000, message: t('validation_content_max'), trigger: 'blur' }
  ],
  reply: [
    { max: 2000, message: t('validation_reply_max'), trigger: 'blur' }
  ]
}

// 加载详情数据
const loadDetail = async () => {
  if (!props.editId) {
    resetForm()
    return
  }

  try {
    const res = await getGuestbookDetail(props.editId)
    if (res.data && res.data.guestbook) {
      Object.assign(formData, {
        name: res.data.guestbook.name || '',
        email: res.data.guestbook.email || '',
        content: res.data.guestbook.content || '',
        reply: res.data.guestbook.reply || '',
        status: res.data.guestbook.status ?? 0
      })
    }
  } catch (error) {
    ErrorHandler.handle(error)
  }
}

// 重置表单
const resetForm = () => {
  Object.assign(formData, {
    name: '',
    email: '',
    content: '',
    reply: '',
    status: 0
  })
  formRef.value?.clearValidate()
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      const data = { ...formData }
      if (!props.editId) {
        // 创建时不需要回复字段
        delete data.reply
        await createGuestbook(data)
        ElMessage.success(t('guestbook.create_success'))
      } else {
        await updateGuestbook(props.editId, data)
        ElMessage.success(t('guestbook.update_success'))
      }
      emit('success')
      handleClose()
    } catch (error) {
      ErrorHandler.handle(error)
    } finally {
      submitting.value = false
    }
  })
}

// 关闭对话框
const handleClose = () => {
  dialogVisible.value = false
  resetForm()
}

// 监听 editId 变化
watch(() => props.editId, () => {
  if (dialogVisible.value) {
    loadDetail()
  }
})

// 监听对话框显示
watch(dialogVisible, (val) => {
  if (val) {
    loadDetail()
  }
})
</script>
```

### 4. 注册路由

在 `html/src/router/index.js` 中添加路由：

```javascript
{
  path: '/guestbook',
  name: 'GuestbookList',
  component: () => import('../views/guestbook/GuestbookList.vue'),
  meta: {
    title: 'menu.guestbook',
    permission: 'guestbook.index'
  }
}
```

### 5. 添加国际化文本

在 `lang/cn.json` 和 `lang/en.json` 中添加相关翻译：

**cn.json:**
```json
{
  "menu": {
    "guestbook": "留言板"
  },
  "guestbook": {
    "add_guestbook": "添加留言",
    "edit_guestbook": "编辑留言",
    "name": "姓名",
    "name_placeholder": "请输入姓名",
    "email": "邮箱",
    "email_placeholder": "请输入邮箱",
    "content": "留言内容",
    "content_placeholder": "请输入留言内容（5-2000字符）",
    "reply": "回复",
    "reply_placeholder": "请输入回复内容（5-2000字符）",
    "status": "状态",
    "status_all": "全部",
    "status_approved": "已审核",
    "status_pending": "待审核",
    "ip": "IP地址",
    "create_success": "留言创建成功",
    "update_success": "留言更新成功",
    "reply_success": "回复成功"
  }
}
```

---

## 测试验证

### 自动化集成测试（`tests/feature`）

项目提供 HTTP 层 smoke test，覆盖健康检查、公开 API 校验与 Admin JWT 保护：

```bash
# 本地需配置 .env 并 migrate 后执行
go test -v ./tests/feature/...
```

CI 中 **Backend** job 跑快速单测（不含 `tests/feature`），**Backend Integration** job 启动 MySQL 后单独跑集成测试。

### 后端接口测试（手动）

1. **测试列表接口**
```bash
curl -X GET "http://localhost:3000/api/admin/guestbooks?page=1&page_size=20" \
  -H "Authorization: Bearer YOUR_TOKEN"
```

2. **测试创建接口**
```bash
curl -X POST "http://localhost:3000/api/admin/guestbooks" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "张三",
    "email": "zhangsan@example.com",
    "content": "这是一条测试留言",
    "status": 0
  }'
```

3. **测试回复接口**
```bash
curl -X POST "http://localhost:3000/api/admin/guestbooks/1/reply" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "reply": "感谢您的留言，我们会尽快处理。"
  }'
```

### 前端功能测试

1. 访问留言列表页面，验证数据加载
2. 测试搜索和筛选功能
3. 测试添加留言功能
4. 测试编辑留言功能
5. 测试删除留言功能
6. 测试回复留言功能

---

## 总结

通过以上步骤，我们完成了一个完整的留言板模块开发，包括：

### 后端部分
- ✅ 数据库迁移文件
- ✅ 模型定义
- ✅ 服务层业务逻辑
- ✅ 请求验证
- ✅ 控制器处理
- ✅ 路由注册

### 前端部分
- ✅ API 客户端封装
- ✅ 列表页面（搜索、分页、操作）
- ✅ 表单组件（创建/编辑）
- ✅ 路由配置
- ✅ 国际化文本

### 开发要点

1. **遵循项目规范**：代码风格、目录结构、命名规范
2. **统一响应格式**：使用 `response.Success()` 和 `response.Error()`
3. **权限控制**：使用中间件和前端权限检查
4. **错误处理**：统一的错误处理和提示
5. **代码复用**：使用组合式函数和工具函数
6. **用户体验**：加载状态、表单验证、操作确认


按照这个文档的步骤，你可以快速开发其他类似的 CRUD 模块。

---

## 新增模块时：操作标题维护清单（操作日志）

当你新增后台模块或接口时，如果希望在“操作日志”里看到正确的操作标题（可检索、可翻译），请同步维护下面几个位置：

1. **权限种子（首选来源）**
   - 文件：`database/seeders/permission_seeder.go`
   - 要求：为接口添加 `Slug + Method + Path`，例如 `xxx.update`。
   - 说明：操作日志标题优先使用权限标识（slug）。

2. **后端标题兜底映射**
   - 文件：`app/utils/operation_title.go`
   - 函数：`generateDefaultTitle(method, path)`
   - 适用：当权限中间件未命中 slug 或历史数据场景，需按路径生成默认标题时。

3. **前端多语言（操作标题显示）**
   - 文件：
     - `html/src/i18n/locales/zh-CN.json`
     - `html/src/i18n/locales/en-US.json`
   - 位置：`permission.*` 命名空间
   - 要求：为新增 slug 增加翻译，例如 `permission.xxx.update`。

4. **操作日志标题搜索下拉（推荐）**
   - 文件：`html/src/views/log/OperationLogList.vue`
   - 位置：
     - `defaultTitleSlugs`（预置可选项）

### 建议规则

- 优先保证标题来自**权限 slug**，`operation_title.go` 仅作为兜底。
- 新增接口后，至少验证三处：
  1) 操作日志列表标题显示正确；
  2) 标题下拉可搜到并已翻译；
  3) 使用标题筛选可命中对应日志。
