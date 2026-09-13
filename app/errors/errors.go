package errors

import (
	stderrors "errors"
	"fmt"
	"strings"

	"goravel/app/http/trans"

	"github.com/goravel/framework/contracts/http"
)

// 定义业务错误类型
var (
	// 认证相关错误
	ErrAccountDisabled       = NewBusinessError("account_disabled", "账号已被禁用")
	ErrPasswordError         = NewBusinessError("password_error", "密码错误")
	ErrNotLoggedIn           = NewBusinessError("not_logged_in", "未登录")
	ErrUsernameOrPasswordErr = NewBusinessError("username_or_password_error", "用户名或密码错误")
	ErrLoginFailed           = NewBusinessError("login_failed", "登录失败")
	ErrLoginLocked           = NewBusinessError("login_locked", "登录失败次数过多，账号已被临时锁定 {minutes} 分钟，请稍后再试")

	// 验证相关错误
	ErrValidationFailed = NewBusinessError("validation_failed", "验证失败")
	ErrInvalidArgument  = NewBusinessError("invalid_argument", "无效的参数")

	// 资源相关错误
	ErrRecordNotFound       = NewBusinessError("record_not_found", "记录不存在")
	ErrBlacklistNotFound    = NewBusinessError("blacklist_not_found", "黑名单不存在")
	ErrNotificationNotFound = NewBusinessError("notification_not_found", "通知不存在")
	ErrAdminNotFound        = NewBusinessError("admin_not_found", "管理员不存在")
	ErrRoleNotFound         = NewBusinessError("role_not_found", "角色不存在")
	ErrMenuNotFound         = NewBusinessError("menu_not_found", "菜单不存在")
	ErrPermissionNotFound   = NewBusinessError("permission_not_found", "权限不存在")
	ErrDepartmentNotFound   = NewBusinessError("department_not_found", "部门不存在")
	ErrPositionNotFound     = NewBusinessError("position_not_found", "岗位不存在")
	ErrDictionaryNotFound   = NewBusinessError("dictionary_not_found", "字典不存在")
	ErrLogNotFound          = NewBusinessError("log_not_found", "日志不存在")

	// IP 相关错误
	ErrIPAddressRequired    = NewBusinessError("ip_address_required", "IP地址不能为空")
	ErrInvalidCIDRFormat    = NewBusinessError("invalid_cidr_format", "CIDR格式错误")
	ErrInvalidIPRangeFormat = NewBusinessError("invalid_ip_range_format", "IP范围格式错误")
	ErrInvalidIPFormat      = NewBusinessError("invalid_ip_format", "IP格式错误")
	ErrInvalidIPRangeOrder  = NewBusinessError("invalid_ip_range_order", "IP范围顺序错误")

	// 参数相关错误
	ErrIDRequired       = NewBusinessError("id_required", "ID不能为空")
	ErrIDsRequired      = NewBusinessError("ids_required", "IDs不能为空")
	ErrParamsError      = NewBusinessError("params_error", "参数错误")
	ErrParamsRequired   = NewBusinessError("params_required", "参数不能为空")
	ErrFileRequired     = NewBusinessError("file_required", "文件不能为空")
	ErrFilePathRequired = NewBusinessError("file_path_required", "文件路径不能为空")
	ErrCodeRequired     = NewBusinessError("code_required", "验证码不能为空")
	ErrTokenIDRequired  = NewBusinessError("token_id_required", "Token ID不能为空")
	ErrTokenIDsRequired = NewBusinessError("token_ids_required", "Token IDs不能为空")
	ErrUserIDRequired   = NewBusinessError("user_id_required", "用户ID不能为空")
	ErrChunkIDRequired  = NewBusinessError("chunk_id_required", "分片ID不能为空")
	ErrFilenameRequired = NewBusinessError("filename_required", "文件名不能为空")

	// 附件相关错误
	ErrChunkUploadOnlyLocalStorage             = NewBusinessError("chunk_upload_only_local_storage", "大文件分片上传仅支持本地存储")
	ErrInvalidChunkIndex                       = NewBusinessError("invalid_chunk_index", "分片索引无效")
	ErrInvalidTotalChunks                      = NewBusinessError("invalid_total_chunks", "总分片数无效")
	ErrInvalidTotalSize                        = NewBusinessError("invalid_total_size", "总大小无效")
	ErrInvalidChunkSize                        = NewBusinessError("invalid_chunk_size", "分片大小无效")
	ErrChunkFileRequired                       = NewBusinessError("chunk_file_required", "分片文件不能为空")
	ErrInvalidAction                           = NewBusinessError("invalid_action", "无效的操作")
	ErrAttachmentNotFound                      = NewBusinessError("attachment_not_found", "附件不存在")
	ErrAttachmentCategoryNotFound              = NewBusinessError("attachment_category_not_found", "附件分类不存在")
	ErrAttachmentCategoryProtectedCannotDelete = NewBusinessError("attachment_category_protected_cannot_delete", "系统分类不可删除")
	ErrAttachmentNotPublic                     = NewBusinessError("attachment_not_public", "附件未公开，无法匿名访问")
	ErrAttachmentFileTypeNotAllowed            = NewBusinessError("attachment_file_type_not_allowed", "不允许上传该类型的文件")
	ErrAttachmentFileContentMismatch           = NewBusinessError("attachment_file_content_mismatch", "文件内容与扩展名不匹配")
	ErrChunkNotFound                           = NewBusinessError("chunk_not_found", "分片 {chunk_index} 不存在")
	ErrChunkMissing                            = NewBusinessError("chunk_missing", "分片缺失: {missing_chunks} (共 {count} 个分片缺失)")
	ErrNoChunkDataToMerge                      = NewBusinessError("no_chunk_data_to_merge", "没有可合并的分片数据")
	ErrSaveChunkFailed                         = NewBusinessError("save_chunk_failed", "保存分片失败")
	ErrCreateDirectoryFailed                   = NewBusinessError("create_directory_failed", "创建目标目录失败")
	ErrCreateFileFailed                        = NewBusinessError("create_file_failed", "创建目标文件失败")
	ErrWriteChunkFailed                        = NewBusinessError("write_chunk_failed", "写入分片 {chunk_index} 失败")
	ErrCloseFileFailed                         = NewBusinessError("close_file_failed", "关闭目标文件失败")
	ErrSaveFileFailed                          = NewBusinessError("save_file_failed", "保存文件失败")
	ErrDeleteFileFailed                        = NewBusinessError("delete_file_failed", "删除文件失败")
	ErrStorageDiskNotConfigured                = NewBusinessError("storage_disk_not_configured", "文件存储驱动 {disk} 未配置，请在 .env 填写对应密钥，或将后台文件存储改回 local")

	// 数据存在性错误
	ErrUsernameExists                = NewBusinessError("username_exists", "用户名已存在")
	ErrUsernameUnavailable           = NewBusinessError("username_unavailable", "username unavailable")
	ErrMenuSlugExists                = NewBusinessError("menu_slug_exists", "菜单标识已存在")
	ErrRoleNameExists                = NewBusinessError("role_name_exists", "角色名称已存在")
	ErrRoleSlugExists                = NewBusinessError("role_slug_exists", "角色标识已存在")
	ErrPermissionNameExists          = NewBusinessError("permission_name_exists", "权限名称已存在")
	ErrPermissionSlugExists          = NewBusinessError("permission_slug_exists", "权限标识已存在")
	ErrPermissionNameOrSlugExists    = NewBusinessError("permission_name_or_slug_exists", "权限名称或标识已存在")
	ErrPermissionNameAndSlugRequired = NewBusinessError("permission_name_and_slug_required", "权限名称和标识不能为空")

	// 业务逻辑错误
	ErrForbidden                       = NewBusinessError("forbidden", "无权限访问该资源")
	ErrMenuHasChildren                 = NewBusinessError("menu_has_children", "菜单存在子菜单，无法删除")
	ErrDepartmentHasChildren           = NewBusinessError("department_has_children", "部门存在子部门，无法删除")
	ErrDepartmentHasAdmins             = NewBusinessError("department_has_admins", "部门存在管理员，无法删除")
	ErrPositionHasAdmins               = NewBusinessError("position_has_admins", "岗位存在管理员，无法删除")
	ErrGoogleAuthenticatorNotBound     = NewBusinessError("google_authenticator_not_bound", "未绑定谷歌验证器")
	ErrGoogleAuthenticatorAlreadyBound = NewBusinessError("google_authenticator_already_bound", "已绑定谷歌验证器")
	ErrGoogleCodeInvalid               = NewBusinessError("google_code_invalid", "谷歌验证码无效")
	ErrGoogleCodeRequired              = NewBusinessError("google_code_required", "谷歌验证码不能为空")
	ErrSecretAndCodeRequired           = NewBusinessError("secret_and_code_required", "密钥和验证码不能为空")
	ErrScheduleCommandRequired         = NewBusinessError("schedule_command_required", "请选择要执行的定时任务")
	ErrScheduleCommandNotAllowed       = NewBusinessError("schedule_command_not_allowed", "不允许执行该命令")
	ErrScheduleBusy                    = NewBusinessError("schedule_busy", "该定时任务正在执行中，请稍后再试")
	ErrScheduleRunFailed               = NewBusinessError("schedule_run_failed", "定时任务执行失败")
	ErrOldPasswordError                = NewBusinessError("old_password_error", "旧密码错误")
	ErrPasswordTooWeak                 = NewBusinessError("password_too_weak", "密码强度不足")
	ErrMustChangePassword              = NewBusinessError("must_change_password", "请先修改密码后再继续操作")
	ErrSensitiveConfirmRequired        = NewBusinessError("sensitive_confirm_required", "敏感操作需要二次确认")
	ErrSensitiveConfirmInvalid         = NewBusinessError("sensitive_confirm_invalid", "二次确认失败")
	ErrLoginAnomaly                    = NewBusinessError("login_anomaly", "检测到异常登录")
	ErrInvalidTokenID                  = NewBusinessError("invalid_token_id", "无效的Token ID")
	ErrInvalidTokenIDs                 = NewBusinessError("invalid_token_ids", "无效的Token IDs")
	ErrInvalidUserID                   = NewBusinessError("invalid_user_id", "无效的用户ID")
	ErrDictionaryTypeRequired          = NewBusinessError("dictionary_type_required", "字典类型不能为空")
	ErrConfigGroupRequired             = NewBusinessError("config_group_required", "配置组不能为空")
	ErrConfigsRequired                 = NewBusinessError("configs_required", "配置不能为空")
	ErrCaptchaExpireMin                = NewBusinessError("captcha_expire_min", "验证码有效期不能小于 {seconds} 秒")
	ErrEmailConfigRequired             = NewBusinessError("email_config_required", "邮箱配置不能为空")
	ErrOptionTypeRequired              = NewBusinessError("option_type_required", "选项类型不能为空")
	ErrInvalidOptionType               = NewBusinessError("invalid_option_type", "无效的选项类型")

	// 余额相关错误
	ErrInsufficientBalance = NewBusinessError("insufficient_balance", "余额不足，当前余额: {balance}")
	ErrInvalidBalanceType  = NewBusinessError("invalid_balance_type", "无效的变动类型: {type}")

	// 订单相关错误
	ErrOrderNotFound           = NewBusinessError("order_not_found", "订单不存在")
	ErrOrderIDRequired         = NewBusinessError("order_id_required", "订单ID不能为空")
	ErrGetLockFailed           = NewBusinessError("get_lock_failed", "获取锁失败")
	ErrGenerateOrderNoFailed   = NewBusinessError("generate_order_no_failed", "生成唯一订单号失败，请重试")
	ErrCreateOrderFailed       = NewBusinessError("create_order_failed", "创建订单失败")
	ErrCreateOrderDetailFailed = NewBusinessError("create_order_detail_failed", "创建订单详情失败")
	ErrQueryOrderDetailFailed  = NewBusinessError("query_order_detail_failed", "查询订单详情失败")
	ErrDeleteOrderDetailFailed = NewBusinessError("delete_order_detail_failed", "删除订单详情失败")

	// 支付相关错误
	ErrPaymentMethodNotFound        = NewBusinessError("payment_method_not_found", "支付方式不存在")
	ErrPaymentNotFound              = NewBusinessError("payment_not_found", "支付记录不存在")
	ErrPaymentMethodDisabled        = NewBusinessError("payment_method_disabled", "支付方式已禁用")
	ErrPaymentMethodCodeExists      = NewBusinessError("payment_method_code_exists", "支付方式代码已存在")
	ErrInvalidPaymentType           = NewBusinessError("invalid_payment_type", "无效的支付类型")
	ErrPaymentGatewayDisabled       = NewBusinessError("payment_gateway_disabled", "该支付渠道未在服务端启用")
	ErrPaymentConfigRequired        = NewBusinessError("payment_config_required", "支付配置不能为空")
	ErrCreatePaymentFailed          = NewBusinessError("create_payment_failed", "创建支付记录失败")
	ErrPaymentAmountInvalid         = NewBusinessError("payment_amount_invalid", "支付金额无效")
	ErrPaymentAmountMismatch        = NewBusinessError("payment_amount_mismatch", "支付金额与订单金额不一致")
	ErrPaymentStatusInvalid         = NewBusinessError("payment_status_invalid", "支付状态无效")
	ErrPaymentNotifyInvalid         = NewBusinessError("payment_notify_invalid", "支付回调数据无效")
	ErrPaymentGatewayNotImplemented = NewBusinessError("payment_gateway_not_implemented", "支付网关该能力尚未实现，详见开源文档支付边界说明")
	ErrOrderNotPayable              = NewBusinessError("order_not_payable", "订单当前状态不可支付")

	// 导出相关错误
	ErrExportRecordNotFound    = NewBusinessError("record_not_found", "导出记录不存在")
	ErrWriteCSVHeaderFailed    = NewBusinessError("write_csv_header_failed", "写入CSV表头失败")
	ErrWriteCSVDataFailed      = NewBusinessError("write_csv_data_failed", "写入CSV数据失败")
	ErrCSVWriteFailed          = NewBusinessError("csv_write_failed", "CSV写入失败")
	ErrBatchDeleteExportFailed = NewBusinessError("batch_delete_failed", "批量删除导出记录失败")

	// 导入相关错误
	ErrInvalidCSVFormat = NewBusinessError("invalid_csv_format", "CSV格式无效")
	ErrInvalidFileType  = NewBusinessError("invalid_file_type", "文件类型无效")

	// 分表相关错误
	ErrBaseTableNotRegistered    = NewBusinessError("base_table_not_registered", "未注册的基础表名: {base_table_name}")
	ErrCreateShardingTableFailed = NewBusinessError("create_sharding_table_failed", "创建分表 {table_name} 失败")
	ErrDeepPaginationExceeded    = NewBusinessError("deep_pagination_exceeded", "分页过深，请缩小时间范围或增加筛选条件（offset+page_size 不能超过 {max}）")
	ErrOrderNoRequired           = NewBusinessError("order_no_required", "订单号不能为空")
	ErrPaymentNoRequired         = NewBusinessError("payment_no_required", "支付单号不能为空")

	// 操作相关错误
	ErrCreateFailed          = NewBusinessError("create_failed", "创建失败")
	ErrUpdateFailed          = NewBusinessError("update_failed", "更新失败")
	ErrQueryFailed           = NewBusinessError("query_failed", "查询失败")
	ErrDeleteFailed          = NewBusinessError("delete_failed", "删除失败")
	ErrPasswordEncryptFailed = NewBusinessError("password_encrypt_failed", "密码加密失败")

	// 资源不存在错误（其他资源错误已在资源相关错误部分定义）
	ErrTokenNotFound             = NewBusinessError("token_not_found", "Token不存在")
	ErrUnauthorized              = NewBusinessError("unauthorized", "未授权")
	ErrTenantRequired            = NewBusinessError("tenant_required", "请指定租户")
	ErrTenantNotFound            = NewBusinessError("tenant_not_found", "租户不存在")
	ErrTenantDisabled            = NewBusinessError("tenant_disabled", "租户已禁用")
	ErrTenantConnectionFailed    = NewBusinessError("tenant_connection_failed", "租户数据库连接失败")
	ErrTenancyDisabled           = NewBusinessError("tenancy_disabled", "未开启多租户（TENANCY_DRIVER=database）")
	ErrTenantExists              = NewBusinessError("tenant_exists", "租户编码已存在")
	ErrTenantMigrateViaCLI       = NewBusinessError("tenant_migrate_via_cli", "开户请勿同步 migrate；请在平台控制台触发异步迁移，或使用 CLI：go run . artisan tenant:migrate {code}")
	ErrTenantPasswordCorrupt     = NewBusinessError("tenant_password_corrupt", "租户库密码密文无效，请重新设置")
	ErrTenantNotReady            = NewBusinessError("tenant_not_ready", "租户尚未完成 migrate，请先在平台控制台触发迁移或执行 tenant:migrate")
	ErrTenantCredentialsRequired = NewBusinessError("tenant_credentials_required", "远程租户库必须提供独立 username/password；同机共用平台账号请设置 TENANCY_ALLOW_PLATFORM_DB_CREDENTIALS=true")
	ErrTenantHintConflict        = NewBusinessError("tenant_hint_conflict", "子域名与请求中的租户标识不一致")
	ErrTenantOpInProgress        = NewBusinessError("tenant_op_in_progress", "该租户已有运维任务进行中，请稍后再试")
	ErrTenantOpQueueFailed       = NewBusinessError("tenant_op_queue_failed", "租户运维任务入队失败")
	ErrTenantStorageQuotaExceeded = NewBusinessError("tenant_storage_quota_exceeded", "租户对象存储配额已用尽")
	ErrTenantTrafficQuotaExceeded = NewBusinessError("tenant_traffic_quota_exceeded", "租户本月流量配额已用尽")
	ErrTokenRefreshFailed        = NewBusinessError("token_refresh_failed", "Token刷新失败")
	ErrUserNotFound              = NewBusinessError("user_not_found", "用户不存在")

	// 权限保护错误
	ErrProtectedAdmin                = NewBusinessError("protected_admin", "受保护的管理员不能进行此操作")
	ErrAdminProtectedCannotDisable   = NewBusinessError("admin_protected_cannot_disable", "受保护的管理员不能禁用")
	ErrAdminCannotModifyRoles        = NewBusinessError("admin_cannot_modify_roles", "不能修改管理员角色")
	ErrAdminProtectedCannotDelete    = NewBusinessError("admin_protected_cannot_delete", "受保护的管理员不能删除")
	ErrAdminCannotDeleteSelf         = NewBusinessError("admin_cannot_delete_self", "不能删除自己")
	ErrRoleProtectedCannotModifySlug = NewBusinessError("role_protected_cannot_modify_slug", "受保护的角色不能修改标识")
	ErrRoleProtectedCannotDisable    = NewBusinessError("role_protected_cannot_disable", "受保护的角色不能禁用")
	ErrRoleProtectedCannotDelete     = NewBusinessError("role_protected_cannot_delete", "受保护的角色不能删除")
)

// BusinessError 业务错误类型
type BusinessError struct {
	Code    string
	Message string
	Err     error
	Params  map[string]any // 用于存储动态参数（如余额值）
}

// NewBusinessError 创建新的业务错误
func NewBusinessError(code, message string) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
	}
}

// Error 实现 error 接口
func (e *BusinessError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap 返回包装的错误
func (e *BusinessError) Unwrap() error {
	return e.Err
}

// clone returns a shallow copy so package-level sentinel errors stay immutable.
func (e *BusinessError) clone() *BusinessError {
	if e == nil {
		return nil
	}
	cp := &BusinessError{
		Code:    e.Code,
		Message: e.Message,
		Err:     e.Err,
	}
	if len(e.Params) > 0 {
		cp.Params = make(map[string]any, len(e.Params))
		for k, v := range e.Params {
			cp.Params[k] = v
		}
	}
	return cp
}

// WithError 包装底层错误（返回副本，不修改原错误实例）
func (e *BusinessError) WithError(err error) *BusinessError {
	cp := e.clone()
	cp.Err = err
	return cp
}

// WithMessage 设置自定义消息（返回副本，不修改原错误实例）
func (e *BusinessError) WithMessage(message string) *BusinessError {
	cp := e.clone()
	cp.Message = message
	return cp
}

// WithParams 设置动态参数（返回副本，不修改原错误实例）
// 参数会在控制器中用于替换翻译后消息中的占位符
// 例如：翻译文件中有 "insufficient_balance": "余额不足，当前余额: {balance}"
// 使用 WithParams(map[string]any{"balance": 100}) 后，控制器会替换为 "余额不足，当前余额: 100.00"
func (e *BusinessError) WithParams(params map[string]any) *BusinessError {
	cp := e.clone()
	if cp.Params == nil {
		cp.Params = make(map[string]any, len(params))
	}
	for k, v := range params {
		cp.Params[k] = v
	}
	return cp
}

// Is 检查错误是否匹配
func (e *BusinessError) Is(target error) bool {
	if t, ok := target.(*BusinessError); ok {
		return e.Code == t.Code
	}
	return stderrors.Is(e.Err, target)
}

// WrapError 包装错误并添加上下文信息
func WrapError(err error, code, message string) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// IsBusinessError 检查是否是业务错误（支持 unwrap）
func IsBusinessError(err error) bool {
	var be *BusinessError
	return stderrors.As(err, &be)
}

// GetBusinessError 获取业务错误（支持 unwrap）
func GetBusinessError(err error) (*BusinessError, bool) {
	var be *BusinessError
	if stderrors.As(err, &be) {
		return be, true
	}
	return nil, false
}

// GetFormattedMessage 获取格式化后的消息（支持多语言和占位符替换）
// 自动处理翻译和占位符替换，简化控制器代码
// 使用示例：
//
//	message := businessErr.GetFormattedMessage(ctx)
func (e *BusinessError) GetFormattedMessage(ctx http.Context) string {
	// 1. 获取翻译后的消息
	message := trans.Get(ctx, e.Code)

	// 2. 如果翻译不存在（返回的是 key），使用默认消息
	if message == e.Code {
		message = e.Message
	}

	// 3. 如果有参数，替换占位符
	if len(e.Params) > 0 {
		message = e.replacePlaceholders(message)
	}

	return message
}

// replacePlaceholders 替换消息中的占位符
// 支持 {key} 和 ${key} 格式
func (e *BusinessError) replacePlaceholders(message string) string {
	result := message
	for key, value := range e.Params {
		// 支持 {key} 和 ${key} 格式
		placeholder1 := fmt.Sprintf("{%s}", key)
		placeholder2 := fmt.Sprintf("${%s}", key)

		// 格式化值
		valueStr := e.formatValue(value)

		// 替换占位符
		result = strings.ReplaceAll(result, placeholder1, valueStr)
		result = strings.ReplaceAll(result, placeholder2, valueStr)
	}
	return result
}

// formatValue 格式化参数值
// float64/float32 保留2位小数，整数保持原样，其他类型使用 %v
func (e *BusinessError) formatValue(value any) string {
	switch v := value.(type) {
	case float64:
		return fmt.Sprintf("%.2f", v)
	case float32:
		return fmt.Sprintf("%.2f", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
