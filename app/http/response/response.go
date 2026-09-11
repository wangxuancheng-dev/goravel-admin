package response

import (
	"reflect"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/http/helpers"
	"goravel/app/http/trans"
	"goravel/app/services"
	"goravel/app/utils/errorlog"
	"goravel/app/utils/traceid"
)

// 错误日志信息的 context key
const (
	errorLogModuleKey     = "error_log_module"
	errorLogMessageKey    = "error_log_message"
	errorLogAttributesKey = "error_log_attributes"
)

// Success 成功响应（支持多语言，自动包含 trace_id）
// messageKey 可选，如果不传则使用默认值 "success"
// 使用方式：
//   - response.Success(ctx)
//   - response.Success(ctx, data)
//   - response.Success(ctx, "custom_message", data)
func Success(ctx http.Context, args ...any) http.Response {
	var messageKey string
	var data any

	// 智能识别参数：如果第一个参数是 string 且长度合理（<=50），则认为是 messageKey
	if len(args) > 0 {
		if msgKey, ok := args[0].(string); ok && len(msgKey) <= 50 {
			// 方式1：传了 messageKey
			messageKey = msgKey
			if len(args) > 1 {
				data = args[1]
			}
		} else {
			// 方式2：没传 messageKey，第一个参数就是 data
			messageKey = "success" // 默认值
			data = args[0]
		}
	} else {
		// 没有参数，使用默认值
		messageKey = "success"
	}

	message := trans.Get(ctx, messageKey)

	response := http.Json{
		"code":    200,
		"message": message,
	}

	// 自动包含 trace_id，方便前端追踪
	if traceID := traceid.FromHTTPContext(ctx); traceID != "" {
		response["trace_id"] = traceID
	}

	// 如果有数据，添加到响应中
	if data != nil {
		// 转换时间字段到对应时区
		convertedData := helpers.ConvertTimesInData(ctx, data)
		response["data"] = convertedData
	}
	return ctx.Response().Success().Json(response)
}

// SuccessWithHeader 成功响应（支持多语言和自定义Header，自动包含 trace_id）
func SuccessWithHeader(ctx http.Context, messageKey string, headerKey, headerValue string, data ...http.Json) http.Response {
	message := trans.Get(ctx, messageKey)

	response := http.Json{
		"code":    200,
		"message": message,
	}

	// 自动包含 trace_id，方便前端追踪
	if traceID := traceid.FromHTTPContext(ctx); traceID != "" {
		response["trace_id"] = traceID
	}

	if len(data) > 0 {
		// 转换时间字段到对应时区
		convertedData := helpers.ConvertTimesInData(ctx, data[0])
		if convertedMap, ok := convertedData.(map[string]any); ok {
			response["data"] = http.Json(convertedMap)
		} else {
			response["data"] = convertedData
		}
	}
	return ctx.Response().Header(headerKey, headerValue).Success().Json(response)
}

// Error 错误响应（支持多语言，自动包含 trace_id 和 error_code）
// 支持两种调用方式：
//  1. response.Error(ctx, code, messageKey) - 使用翻译键
//  2. response.Error(ctx, code, err) - 自动检测 BusinessError 并处理占位符替换
//
// 当 code >= 500 时，如果 context 中包含错误日志信息，会自动记录日志
func Error(ctx http.Context, code int, messageOrErr any) http.Response {
	var message string
	var messageKey string

	// 判断第二个参数是 error 还是 string
	switch v := messageOrErr.(type) {
	case error:
		// 如果是 error，检查是否是 BusinessError
		if businessErr, ok := apperrors.GetBusinessError(v); ok {
			// 使用 GetFormattedMessage 自动处理翻译和占位符替换
			message = businessErr.GetFormattedMessage(ctx)
			messageKey = businessErr.Code
		} else {
			// 普通错误，直接使用错误消息
			message = v.Error()
			messageKey = "operation_failed"
		}
	case string:
		// 如果是 string，当作翻译键处理
		messageKey = v
		message = trans.Get(ctx, messageKey)
	default:
		// 其他类型，转换为字符串
		messageKey = "operation_failed"
		message = trans.Get(ctx, messageKey)
	}

	response := http.Json{
		"code":       code,
		"message":    message,
		"error_code": messageKey, // 添加错误码字段，方便前端判断
	}

	// 自动包含 trace_id，方便前端显示和用户报告错误
	if traceID := traceid.FromHTTPContext(ctx); traceID != "" {
		response["trace_id"] = traceID
	}

	// 系统级错误（500+）自动记录日志（如果 context 中有日志信息）
	if code >= 500 {
		if module, ok := ctx.Value(errorLogModuleKey).(string); ok && module != "" {
			logMessage := ctx.Value(errorLogMessageKey)
			if logMsg, ok := logMessage.(string); ok && logMsg != "" {
				attributes := ctx.Value(errorLogAttributesKey)
				var attrs map[string]any
				if attributes != nil {
					if attrsMap, ok := attributes.(map[string]any); ok {
						attrs = attrsMap
					}
				}
				errorlog.RecordHTTP(ctx, module, logMsg, attrs, "%s: %s", module, logMsg)
			}
		}
	}

	return ctx.Response().Json(code, response)
}

// Abort writes a canonical error JSON (with error_code / trace_id) and stops the middleware chain.
// Prefer this over hand-rolled ctx.Response().Json(...).Abort() in middleware.
func Abort(ctx http.Context, code int, messageOrErr any) {
	resp := Error(ctx, code, messageOrErr)
	if abortable, ok := resp.(interface{ Abort() error }); ok {
		_ = abortable.Abort()
		return
	}
	ctx.Request().Abort(code)
}

// SetErrorLog 设置错误日志信息到 context（用于系统级错误）
// 在调用 response.Error 之前调用此函数，Error 函数会自动记录日志
func SetErrorLog(ctx http.Context, module, logMessage string, attributes map[string]any) {
	ctx.WithValue(errorLogModuleKey, module)
	ctx.WithValue(errorLogMessageKey, logMessage)
	if attributes != nil {
		ctx.WithValue(errorLogAttributesKey, attributes)
	}
}

// ErrorWithLog 错误响应并自动记录日志（用于系统级错误）
// 支持多种调用方式，自动推断参数：
//
// 最简洁方式（推荐）：
//   - response.ErrorWithLog(ctx, "module", err)
//   - response.ErrorWithLog(ctx, "module", err, map[string]any{"extra": "value"})
//
// 完整方式：
//   - response.ErrorWithLog(ctx, code, messageKey, module, logMessage, err)
//   - response.ErrorWithLog(ctx, code, messageKey, module, logMessage, err, map[string]any{...})
func ErrorWithLog(ctx http.Context, args ...any) http.Response {
	// 智能识别调用方式
	if len(args) == 0 {
		return Error(ctx, http.StatusInternalServerError, "operation_failed")
	}

	// 方式1：最简洁方式 - ErrorWithLog(ctx, "module", err) 或 ErrorWithLog(ctx, "module", err, attrs)
	if len(args) >= 2 {
		if module, ok := args[0].(string); ok {
			var err error
			var attributes map[string]any

			// 查找 error 和 attributes
			for i := 1; i < len(args); i++ {
				switch v := args[i].(type) {
				case error:
					if err == nil {
						err = v
					}
				case map[string]any:
					if attributes == nil {
						attributes = v
					}
				}
			}

			if err == nil {
				return Error(ctx, http.StatusInternalServerError, "operation_failed")
			}

			// 自动生成日志消息
			logMessage := err.Error()
			if len(logMessage) > 100 {
				logMessage = logMessage[:100] + "..."
			}

			// 设置属性
			if attributes == nil {
				attributes = make(map[string]any)
			}
			if _, exists := attributes["error"]; !exists {
				attributes["error"] = err.Error()
			}

			// 自动记录日志
			SetErrorLog(ctx, module, logMessage, attributes)
			return Error(ctx, http.StatusInternalServerError, "operation_failed")
		}
	}

	// 方式2：完整方式 - ErrorWithLog(ctx, code, messageKey, module, logMessage, ...)
	if len(args) >= 4 {
		code, codeOk := args[0].(int)
		messageKey, msgOk := args[1].(string)
		module, modOk := args[2].(string)
		logMessage, logOk := args[3].(string)

		if codeOk && msgOk && modOk && logOk {
			var attributes map[string]any
			var err error

			// 解析剩余参数
			for i := 4; i < len(args); i++ {
				switch v := args[i].(type) {
				case error:
					if err == nil {
						err = v
					}
				case map[string]any:
					if attributes == nil {
						attributes = v
					}
				}
			}

			// 设置属性
			if attributes == nil {
				attributes = make(map[string]any)
			}
			if err != nil {
				if _, exists := attributes["error"]; !exists {
					attributes["error"] = err.Error()
				}
			}

			// 系统级错误（500+）设置日志信息，Error 函数会自动记录
			if code >= 500 {
				SetErrorLog(ctx, module, logMessage, attributes)
			}
			return Error(ctx, code, messageKey)
		}
	}

	// 无法识别参数格式，返回通用错误
	return Error(ctx, http.StatusInternalServerError, "operation_failed")
}

// ErrorWithLogAuto 超简洁版本：自动推断所有参数
// 只需要传入 module 和 err，其他参数自动推断
// 默认使用 HTTP 500 状态码和通用的错误消息
//
// 使用方式：
//   - response.ErrorWithLogAuto(ctx, "operation-log", err)
//   - response.ErrorWithLogAuto(ctx, "operation-log", err, map[string]any{"extra": "value"})
func ErrorWithLogAuto(ctx http.Context, module string, args ...any) http.Response {
	var err error
	var attributes map[string]any

	// 解析参数
	for i := len(args) - 1; i >= 0; i-- {
		switch v := args[i].(type) {
		case error:
			if err == nil {
				err = v
			}
		case map[string]any:
			if attributes == nil {
				attributes = v
			}
		}
	}

	// 如果没有 error，返回通用错误
	if err == nil {
		return Error(ctx, http.StatusInternalServerError, "operation_failed")
	}

	// 如果没有提供 attributes，创建一个
	if attributes == nil {
		attributes = make(map[string]any)
	}

	// 自动添加 error 字段
	if _, exists := attributes["error"]; !exists {
		attributes["error"] = err.Error()
	}

	// 自动生成日志消息：使用 err.Error() 作为日志消息
	logMessage := err.Error()
	// 如果错误信息太长，截取前100个字符
	if len(logMessage) > 100 {
		logMessage = logMessage[:100] + "..."
	}

	// 自动推断 messageKey：根据 module 生成通用的错误消息键
	messageKey := "operation_failed"
	if module != "" {
		// 尝试使用 module 相关的错误消息键
		// 例如 "operation-log" -> "operation_failed"
		// 或者保持通用
		messageKey = "operation_failed"
	}

	// 系统级错误（500）自动记录日志
	SetErrorLog(ctx, module, logMessage, attributes)
	return Error(ctx, http.StatusInternalServerError, messageKey)
}

// ValidationError 验证错误响应（支持多语言，自动包含 trace_id 和 error_code）
// 自动提取第一个错误信息并添加到 message 中，方便前端直接显示
func ValidationError(ctx http.Context, code int, messageKey string, errors map[string]map[string]string) http.Response {
	baseMessage := trans.Get(ctx, messageKey)

	// 提取第一个错误信息
	var firstError string
	for _, fieldErrors := range errors {
		for _, errorMsg := range fieldErrors {
			firstError = errorMsg
			break
		}
		if firstError != "" {
			break
		}
	}

	// 如果有具体错误信息，将其添加到 message 中
	var message string
	if firstError != "" {
		message = firstError
	} else {
		message = baseMessage
	}

	response := http.Json{
		"code":       code,
		"message":    message,
		"error_code": messageKey, // 添加错误码字段，方便前端判断
		"errors":     errors,
	}

	// 自动包含 trace_id，方便前端显示和用户报告错误
	if traceID := traceid.FromHTTPContext(ctx); traceID != "" {
		response["trace_id"] = traceID
	}

	return ctx.Response().Json(code, response)
}

// Paginate 分页响应（支持多语言，自动包含 trace_id）
// messageKey 可选，如果不传则使用默认值 "get_success"
// 使用方式：
//   - response.Paginate(ctx, list, total, page, pageSize)
//   - response.Paginate(ctx, "custom_message", list, total, page, pageSize)
func Paginate(ctx http.Context, args ...any) http.Response {
	var messageKey string
	var list any
	var total int64
	var page, pageSize int

	// 智能识别参数：如果第一个参数是 string 且长度合理（<=50），则认为是 messageKey
	if len(args) >= 4 {
		if msgKey, ok := args[0].(string); ok && len(msgKey) <= 50 {
			// 方式1：传了 messageKey
			messageKey = msgKey
			list = args[1]
			if t, ok := args[2].(int64); ok {
				total = t
			} else if t, ok := args[2].(int); ok {
				total = int64(t)
			}
			if p, ok := args[3].(int); ok {
				page = p
			}
			if len(args) >= 5 {
				if ps, ok := args[4].(int); ok {
					pageSize = ps
				}
			}
		} else {
			// 方式2：没传 messageKey
			messageKey = "get_success" // 默认值
			list = args[0]
			if t, ok := args[1].(int64); ok {
				total = t
			} else if t, ok := args[1].(int); ok {
				total = int64(t)
			}
			if p, ok := args[2].(int); ok {
				page = p
			}
			if len(args) >= 4 {
				if ps, ok := args[3].(int); ok {
					pageSize = ps
				}
			}
		}
	}

	// 如果 messageKey 为空，使用默认值
	if messageKey == "" {
		messageKey = "get_success"
	}

	message := trans.Get(ctx, messageKey)

	// 转换列表中的时间字段到对应时区
	convertedList := helpers.ConvertTimesInData(ctx, list)

	response := http.Json{
		"code":    200,
		"message": message,
		"data": http.Json{
			"list":        convertedList,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": helpers.TotalPages(total, pageSize),
		},
	}

	// 自动包含 trace_id，方便前端追踪
	if traceID := traceid.FromHTTPContext(ctx); traceID != "" {
		response["trace_id"] = traceID
	}

	return ctx.Response().Success().Json(response)
}

// PaginateQueryOptions 分页查询选项
type PaginateQueryOptions struct {
	// WithRelations 预加载关联，例如 []string{"Department", "Roles"}
	WithRelations []string
	// Transform 数据转换函数，可以对查询结果进行转换
	Transform func(any) any
	// ErrorHandler 自定义错误处理函数，如果为 nil 则使用默认错误处理
	ErrorHandler func(ctx http.Context, err error, module string) http.Response
	// ErrorModule 错误日志模块名，用于 ErrorWithLog
	ErrorModule string
}

// PaginateQuery 通用的分页查询封装
// query: 构建好的查询对象（已包含所有查询条件）
// list: 用于接收查询结果的切片指针，例如 &[]models.Dictionary{}
// options: 可选配置
//
// 使用示例：
//
//   - 基础用法：
//     return response.PaginateQuery(ctx, query, &dictionaries, nil)
//
//   - 带预加载关联：
//     return response.PaginateQuery(ctx, query, &roles, &response.PaginateQueryOptions{
//     WithRelations: []string{"Permissions", "Menus"},
//     })
//
//   - 带数据转换：
//     return response.PaginateQuery(ctx, query, &admins, &response.PaginateQueryOptions{
//     WithRelations: []string{"Department", "Roles"},
//     Transform: func(data any) any {
//     admins := data.(*[]models.Admin)
//     // 转换逻辑
//     return adminList
//     },
//     })
//
//   - 带错误日志模块：
//     return response.PaginateQuery(ctx, query, &logs, &response.PaginateQueryOptions{
//     WithRelations: []string{"Admin"},
//     ErrorModule:   "login-log",
//     })
func PaginateQuery(ctx http.Context, query orm.Query, list any, options *PaginateQueryOptions) http.Response {
	// 获取分页参数
	page := helpers.GetIntQuery(ctx, "page", 1)
	pageSize := helpers.GetIntQuery(ctx, "page_size", 10)
	page, pageSize = helpers.ValidatePagination(page, pageSize)

	// 获取总数
	total, err := query.Count()
	if err != nil {
		if options != nil && options.ErrorHandler != nil {
			return options.ErrorHandler(ctx, err, options.ErrorModule)
		}
		if options != nil && options.ErrorModule != "" {
			return ErrorWithLog(ctx, options.ErrorModule, err)
		}
		return Error(ctx, http.StatusInternalServerError, "query_failed")
	}

	// 应用预加载关联
	if options != nil && len(options.WithRelations) > 0 {
		for _, relation := range options.WithRelations {
			query = query.With(relation)
		}
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Get(list); err != nil {
		if options != nil && options.ErrorHandler != nil {
			return options.ErrorHandler(ctx, err, options.ErrorModule)
		}
		if options != nil && options.ErrorModule != "" {
			return ErrorWithLog(ctx, options.ErrorModule, err)
		}
		return Error(ctx, http.StatusInternalServerError, "query_failed")
	}

	// 数据转换
	var result any = list
	if options != nil && options.Transform != nil {
		result = options.Transform(list)
	}

	return Paginate(ctx, result, total, page, pageSize)
}

// Export 导出响应（支持多语言）
// headers: CSV表头（可以是翻译键数组或字符串数组）
// data: 数据行，每行是一个字符串切片
// filename: 文件名（不含扩展名）
func Export(ctx http.Context, messageKey string, headers []string, data [][]string, filename string) http.Response {
	message := trans.Get(ctx, messageKey)

	// 翻译表头（如果表头是翻译键，则翻译；如果是普通字符串，则保持原样）
	translatedHeaders := make([]string, len(headers))
	for i, header := range headers {
		// 尝试翻译，如果翻译键不存在则返回原字符串
		translated := trans.Get(ctx, header)
		if translated == header {
			// 如果翻译结果和原字符串相同，说明不是翻译键，直接使用原字符串
			translatedHeaders[i] = header
		} else {
			// 如果翻译成功，使用翻译后的文本
			translatedHeaders[i] = translated
		}
	}

	exportService := services.NewExportService(ctx)
	filePath, err := exportService.ExportToFile(translatedHeaders, data, filename)
	if err != nil {
		return Error(ctx, http.StatusInternalServerError, "output_failed")
	}

	exportURL := exportService.GetExportURL(filePath)

	response := http.Json{
		"code":    200,
		"message": message,
		"data": http.Json{
			"file_path": filePath,
			"file_url":  exportURL,
		},
	}

	// 自动包含 trace_id，方便前端追踪
	if traceID := traceid.FromHTTPContext(ctx); traceID != "" {
		response["trace_id"] = traceID
	}

	return ctx.Response().Success().Json(response)
}

// FindByIDOptions 查找选项
type FindByIDOptions struct {
	// WithRelations 预加载关联，例如 []string{"Department", "Roles"}
	WithRelations []string
	// NotFoundMessageKey 未找到时的错误消息键，例如 "admin_not_found"
	// 如果不提供，默认使用 "record_not_found"
	NotFoundMessageKey string
}

// FindByID 通用的根据ID查找记录函数
// T 必须是嵌入了 orm.Model 的模型类型
// 使用示例：
//
//	// 简单查找
//	admin, resp := response.FindByID[models.Admin](ctx, id, nil)
//	if resp != nil {
//		return resp
//	}
//
//	// 带关联预加载
//	admin, resp := response.FindByID[models.Admin](ctx, id, &response.FindByIDOptions{
//		WithRelations:      []string{"Department", "Roles"},
//		NotFoundMessageKey: "admin_not_found",
//	})
//	if resp != nil {
//		return resp
//	}
func FindByID[T any](ctx http.Context, id uint, options *FindByIDOptions) (*T, http.Response) {
	// 验证ID
	if id == 0 {
		return nil, Error(ctx, http.StatusBadRequest, apperrors.ErrIDRequired.Code)
	}

	// 创建查询
	query := appfacades.OrmQuery(ctx).Where("id", id)

	// 应用关联预加载
	if options != nil && len(options.WithRelations) > 0 {
		for _, relation := range options.WithRelations {
			query = query.With(relation)
		}
	}

	// 查询记录
	var model T
	if err := query.First(&model); err != nil {
		// 确定错误消息键，默认使用 record_not_found
		messageKey := apperrors.ErrRecordNotFound.Code
		if options != nil && options.NotFoundMessageKey != "" {
			messageKey = options.NotFoundMessageKey
		}
		return nil, Error(ctx, http.StatusNotFound, messageKey)
	}

	// 检查记录是否存在（防御性编程）
	// 通过反射检查 ID 字段，如果 ID 为 0，说明记录不存在
	modelPtr := &model
	if !hasValidID(modelPtr) {
		// 确定错误消息键，默认使用 record_not_found
		messageKey := apperrors.ErrRecordNotFound.Code
		if options != nil && options.NotFoundMessageKey != "" {
			messageKey = options.NotFoundMessageKey
		}
		return nil, Error(ctx, http.StatusNotFound, messageKey)
	}

	return modelPtr, nil
}

// hasValidID 检查模型是否有有效的 ID（ID != 0）
// 使用反射来检查模型的 ID 字段
func hasValidID(model any) bool {
	v := reflect.ValueOf(model)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// 查找 ID 字段
	idField := v.FieldByName("ID")
	if !idField.IsValid() {
		// 如果没有找到 ID 字段，假设记录有效（防御性编程）
		return true
	}

	// 检查 ID 是否为 0
	if idField.Kind() == reflect.Uint || idField.Kind() == reflect.Uint32 || idField.Kind() == reflect.Uint64 {
		return idField.Uint() != 0
	}

	return true
}
