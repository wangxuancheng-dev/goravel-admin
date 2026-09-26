package middleware

import (
	"context"
	"encoding/json"
	appfacades "goravel/app/facades"
	"slices"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancyctx"
	"goravel/app/utils"
	"goravel/app/utils/logger"
	"goravel/app/utils/traceid"
)

// configStrings 从 Goravel 配置中读取字符串切片，兼容 []string 和 []any 两种底层类型。
func configStrings(key string) []string {
	val := facades.Config().Get(key, []string{})
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// shouldPersistOperationLog 根据配置判断本次请求是否需要写入操作日志。
// 规则来源：config/operation_log.go，新增排除项只需修改配置。
func shouldPersistOperationLog(method, path string, ctx http.Context) bool {
	if !slices.Contains(configStrings("operation_log.allowed_methods"), method) {
		return false
	}
	if slices.Contains(configStrings("operation_log.excluded_paths"), path) {
		return false
	}
	for _, prefix := range configStrings("operation_log.excluded_path_prefixes") {
		if strings.HasPrefix(path, prefix) {
			return false
		}
	}
	// 分片上传仅 merge 操作记录日志
	chunkPath := "/api/admin/attachments/chunk"
	if chunkPath != "" && path == chunkPath {
		action := ctx.Request().Input("action", "")
		if action == "" {
			action = ctx.Request().Query("action", "")
		}
		return action == "merge"
	}
	return true
}

// OperationLog 操作日志中间件
func OperationLog() http.Middleware {
	return newMiddleware("operation_log", func(ctx http.Context) {
		systemLogService := services.NewSystemLogService(ctx)
		startTime := time.Now()

		// 获取请求信息
		method := ctx.Request().Method()
		path := ctx.Request().Path()
		ip := ctx.Request().Ip()
		userAgent := ctx.Request().Header("User-Agent", "")

		// 获取请求参数（排除敏感信息）
		var requestBody string
		if method == "POST" || method == "PUT" || method == "PATCH" {
			// 获取所有输入参数
			inputs := make(map[string]any)
			// 记录所有非敏感参数
			allInputs := ctx.Request().All()
			for key, value := range allInputs {
				// 使用工具函数检查是否是敏感字段
				if utils.IsSensitiveField(key) {
					inputs[key] = "***"
				} else {
					inputs[key] = value
				}
			}
			if data, err := json.Marshal(inputs); err == nil {
				requestBody = string(data)
			}
		}

		// 获取管理员ID（从JWT中间件设置的context中获取）
		var adminID uint
		if admin, err := helpers.GetAdminFromContext(ctx); err == nil {
			adminID = admin.ID
		}

		// 审计变更：在 Next 前加载快照，闭包在 Next 后计算 changes（规则见 utils.RegisterAuditHandler / audit_handlers.go）
		computeAuditChanges := utils.PrepareAuditChanges(ctx, method, path, requestBody, adminID)

		// 继续处理请求
		ctx.Request().Next()

		// 计算耗时
		duration := int(time.Since(startTime).Milliseconds())

		if shouldPersistOperationLog(method, path, ctx) {
			// 在请求处理后再获取一次管理员ID（确保JWT中间件已执行）
			// 如果之前没有获取到，再次尝试从context获取
			if adminID == 0 {
				if admin, err := helpers.GetAdminFromContext(ctx); err == nil {
					adminID = admin.ID
				}
			}

			// 默认状态为成功
			status := uint8(1)
			var errorMsg string

			// 在goroutine之前保存所有需要的数据，避免context问题
			savedAdminID := adminID
			savedMethod := method
			savedPath := path
			savedIP := ip
			savedUserAgent := userAgent
			savedRequestBody := requestBody
			savedDuration := duration

			// 提前获取 traceCtx，用于日志记录
			traceCtx := traceid.DeriveContextFromHTTP(ctx)

			// 生成操作标题（只使用权限标识）
			title := utils.GetOperationTitleFromContext(ctx)
			if title == "operation.unknown" {
				// 如果无法生成标题，记录调试日志
				logger.ErrorfContext(traceCtx, "Failed to generate operation title, method: %s, path: %s", savedMethod, savedPath)
			}

			var changes string
			if computeAuditChanges != nil {
				changes = computeAuditChanges()
			}
			traceID := traceid.FromHTTPContext(ctx)

			operationLog := models.OperationLog{
				AdminID:   savedAdminID,
				TraceID:   traceID,
				Method:    savedMethod,
				Path:      savedPath,
				Title:     title,
				IP:        savedIP,
				UserAgent: savedUserAgent,
				Request:   savedRequestBody,
				Changes:   changes,
				Status:    status,
				ErrorMsg:  errorMsg,
				Duration:  savedDuration,
			}

			// 异步落库：请求结束后 HTTP ctx 会取消，需 Detach 保留租户连接键
			persistCtx := tenancyctx.Detach(ctx)
			go func(dbCtx context.Context) {
				defer func() {
					if r := recover(); r != nil {
						logger.ErrorfContext(context.Background(), "Panic in operation log goroutine: %v", r)
					}
				}()
				if err := appfacades.OrmQuery(dbCtx).Create(&operationLog); err != nil {
					_ = systemLogService.Record(dbCtx, "error", "operation-log", "failed to persist operation log", map[string]any{
						"error": err.Error(),
						"path":  savedPath,
					})
					logger.ErrorfContext(dbCtx, "Failed to create operation log: %v", err)
				} else {
					services.DispatchAuditWebhook(dbCtx, "operation_log", map[string]any{
						"id":        operationLog.ID,
						"admin_id":  operationLog.AdminID,
						"method":    operationLog.Method,
						"path":      operationLog.Path,
						"title":     operationLog.Title,
						"status":    operationLog.Status,
						"ip":        operationLog.IP,
						"duration":  operationLog.Duration,
						"trace_id":  operationLog.TraceID,
					})
				}
			}(persistCtx)
		}
	})
}
