# 错误码总览（Admin API）

本文档用于前端联调时快速定位 `error_code`。  
最终以 `app/errors/errors.go` 中定义为准。

## 通用错误响应结构

```json
{
  "code": 404,
  "error_code": "record_not_found",
  "message": "记录不存在",
  "trace_id": "01xxxxxxxxxxxxxxxxxxxxxxx"
}
```

- `code`: HTTP 状态码（同时也是业务返回中的 `code` 字段）
- `error_code`: 前端建议用于分支处理的稳定错误码
- `message`: 可展示给用户的文案（支持 i18n）
- `trace_id`: 链路追踪 ID（排查问题时提供给后端）

## 常见 HTTP 状态码映射

- `400`: 参数错误、校验失败、业务前置条件不满足
- `401`: 未登录、凭证无效
- `403`: 无权限、受保护资源禁止操作
- `404`: 资源不存在
- `429`: 请求过于频繁（如导出防重复提交）
- `500`: 服务器内部错误（可携带 `operation_failed` 等）

## 错误码分组

### 1) 认证与会话

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `not_logged_in` | 未登录 | 401 |
| `unauthorized` | 未授权 | 401 |
| `username_or_password_error` | 用户名或密码错误 | 401 |
| `account_disabled` | 账号已禁用 | 403 |
| `login_failed` | 登录失败 | 400/401 |
| `login_locked` | 登录次数过多被锁定 | 429/403 |
| `token_not_found` | Token 不存在 | 404 |
| `token_refresh_failed` | Token 刷新失败 | 401 |
| `token_id_required` | Token ID 不能为空 | 400 |
| `token_ids_required` | Token IDs 不能为空 | 400 |
| `invalid_token_id` | 无效的 Token ID | 400 |
| `invalid_token_ids` | 无效的 Token IDs | 400 |

### 2) 参数与校验

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `validation_failed` | 请求参数校验失败 | 400 |
| `invalid_argument` | 无效参数 | 400 |
| `params_error` | 参数错误 | 400 |
| `params_required` | 参数不能为空 | 400 |
| `id_required` | ID 不能为空 | 400 |
| `ids_required` | IDs 不能为空 | 400 |
| `file_required` | 文件不能为空 | 400 |
| `file_path_required` | 文件路径不能为空 | 400 |
| `code_required` | 验证码不能为空 | 400 |
| `user_id_required` | 用户 ID 不能为空 | 400 |
| `invalid_user_id` | 无效的用户 ID | 400 |
| `invalid_file_type` | 文件类型无效 | 400 |

### 3) 资源不存在

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `record_not_found` | 通用记录不存在 | 404 |
| `admin_not_found` | 管理员不存在 | 404 |
| `user_not_found` | 用户不存在 | 404 |
| `role_not_found` | 角色不存在 | 404 |
| `menu_not_found` | 菜单不存在 | 404 |
| `permission_not_found` | 权限不存在 | 404 |
| `department_not_found` | 部门不存在 | 404 |
| `position_not_found` | 岗位不存在 | 404 |
| `dictionary_not_found` | 字典不存在 | 404 |
| `blacklist_not_found` | 黑名单不存在 | 404 |
| `notification_not_found` | 通知不存在 | 404 |
| `order_not_found` | 订单不存在 | 404 |
| `payment_not_found` | 支付记录不存在 | 404 |
| `payment_method_not_found` | 支付方式不存在 | 404 |
| `attachment_not_found` | 附件不存在 | 404 |
| `log_not_found` | 日志不存在 | 404 |
| `export_record_not_found` | 导出记录不存在 | 404 |

### 4) 管理员/角色保护规则

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `protected_admin` | 受保护管理员不可操作 | 403 |
| `admin_protected_cannot_disable` | 受保护管理员不能禁用 | 403 |
| `admin_protected_cannot_delete` | 受保护管理员不能删除 | 403 |
| `admin_cannot_delete_self` | 不能删除自己 | 403 |
| `admin_cannot_modify_roles` | 不能修改管理员角色 | 403 |
| `role_protected_cannot_modify_slug` | 受保护角色不能修改标识 | 403 |
| `role_protected_cannot_disable` | 受保护角色不能禁用 | 403 |
| `role_protected_cannot_delete` | 受保护角色不能删除 | 403 |

### 5) 2FA（谷歌验证器）

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `google_authenticator_not_bound` | 未绑定谷歌验证器 | 400/403 |
| `google_authenticator_already_bound` | 已绑定谷歌验证器 | 400 |
| `google_code_required` | 谷歌验证码不能为空 | 400 |
| `google_code_invalid` | 谷歌验证码无效 | 400 |
| `secret_and_code_required` | 密钥和验证码不能为空 | 400 |

### 6) 业务冲突与依赖关系

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `username_exists` | 用户名已存在 | 400 |
| `menu_slug_exists` | 菜单标识已存在 | 400 |
| `role_name_exists` | 角色名称已存在 | 400 |
| `role_slug_exists` | 角色标识已存在 | 400 |
| `permission_name_exists` | 权限名称已存在 | 400 |
| `permission_slug_exists` | 权限标识已存在 | 400 |
| `permission_name_or_slug_exists` | 权限名称或标识已存在 | 400 |
| `department_has_children` | 部门存在子部门，无法删除 | 400 |
| `department_has_admins` | 部门存在管理员，无法删除 | 400 |
| `position_has_admins` | 岗位存在管理员，无法删除 | 400 |
| `menu_has_children` | 菜单存在子菜单，无法删除 | 400 |

### 7) 订单与支付

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `order_id_required` | 订单 ID 不能为空 | 400 |
| `generate_order_no_failed` | 生成订单号失败 | 500 |
| `create_order_failed` | 创建订单失败 | 500 |
| `create_order_detail_failed` | 创建订单详情失败 | 500 |
| `query_order_detail_failed` | 查询订单详情失败 | 500 |
| `delete_order_detail_failed` | 删除订单详情失败 | 500 |
| `payment_method_disabled` | 支付方式已禁用 | 400/403 |
| `payment_method_code_exists` | 支付方式代码已存在 | 400 |
| `invalid_payment_type` | 无效支付类型 | 400 |
| `payment_config_required` | 支付配置不能为空 | 400 |
| `create_payment_failed` | 创建支付记录失败 | 500 |
| `payment_amount_invalid` | 支付金额无效 | 400 |
| `payment_status_invalid` | 支付状态无效 | 400 |

### 8) 导入导出

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `export_record_not_found` | 导出记录不存在 | 404 |
| `write_csv_header_failed` | 写入 CSV 表头失败 | 500 |
| `write_csv_data_failed` | 写入 CSV 数据失败 | 500 |
| `csv_write_failed` | CSV 写入失败 | 500 |
| `excel_not_implemented` | （已移除）导出支持 CSV / XLSX，见配置 `storage.export_format` | — |
| `batch_delete_export_failed` | 批量删除导出记录失败 | 500 |
| `invalid_csv_format` | CSV 格式无效 | 400 |

### 9) 附件与分片上传

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `chunk_upload_only_local_storage` | 分片上传仅支持本地存储 | 400 |
| `invalid_chunk_index` | 分片索引无效 | 400 |
| `invalid_total_chunks` | 总分片数无效 | 400 |
| `invalid_total_size` | 总大小无效 | 400 |
| `invalid_chunk_size` | 分片大小无效 | 400 |
| `chunk_file_required` | 分片文件不能为空 | 400 |
| `invalid_action` | 无效操作 | 400 |
| `chunk_not_found` | 分片不存在 | 404 |
| `chunk_missing` | 分片缺失 | 400 |
| `no_chunk_data_to_merge` | 没有可合并的分片 | 400 |
| `save_chunk_failed` | 保存分片失败 | 500 |
| `create_directory_failed` | 创建目录失败 | 500 |
| `create_file_failed` | 创建文件失败 | 500 |
| `write_chunk_failed` | 写入分片失败 | 500 |
| `close_file_failed` | 关闭文件失败 | 500 |
| `save_file_failed` | 保存文件失败 | 500 |
| `delete_file_failed` | 删除文件失败 | 500 |

### 10) 通用操作错误

| error_code | 说明 | 常见 HTTP |
|---|---|---|
| `create_failed` | 创建失败 | 500 |
| `update_failed` | 更新失败 | 500 |
| `delete_failed` | 删除失败 | 500 |
| `query_failed` | 查询失败 | 500 |
| `password_encrypt_failed` | 密码加密失败 | 500 |
| `operation_failed` | 操作失败（兜底错误） | 500 |

## 前端对接建议

- 业务分支判断优先使用 `error_code`，不要仅依赖 `message` 文案。
- 异常排查时同时记录 `trace_id`，便于后端定位日志。
- 若接口文档与实际不一致，以接口实际返回和本文件说明为准，并反馈补充。
