# 系统日志记录指南

## 日志级别记录策略

### ✅ 需要记录到系统日志的级别

#### 1. **error（错误）级别** - **必须记录**
- **数据库操作失败** - 创建、更新、删除数据失败
- **文件操作失败** - 保存、删除文件失败
- **系统服务异常** - 缓存、队列、分片表等基础设施错误
- **关键业务逻辑错误** - 订单创建失败、支付失败等影响业务流程的错误
- **外部服务调用失败** - 第三方API调用失败

#### 2. **warning（警告）级别** - **应该记录**
- **非关键操作失败但可继续** - 如分片删除失败、分片读取失败（不影响主流程）
- **资源清理失败** - 临时文件清理失败、缓存清理失败等
- **性能警告** - 操作耗时过长、资源使用率高等
- **配置问题** - 配置缺失或无效，但系统仍可运行

### ❌ 不需要记录到系统日志的级别

#### 3. **info（信息）级别** - **不记录到数据库**
- **正常业务流程** - 订单创建成功、支付成功等
- **操作统计** - 用户登录、数据导出等
- **系统状态** - 服务启动、配置加载等
- **说明**：这些信息只在文件日志中记录，不写入数据库，避免产生大量日志

#### 4. **debug（调试）级别** - **不记录到数据库**
- **调试信息** - 变量值、函数调用栈等
- **详细执行流程** - 仅在开发环境需要
- **说明**：只在开发环境的文件日志中记录，生产环境通常不记录

## 原则总结

**需要记录到系统日志的错误：**
1. **数据库操作失败** - 创建、更新、删除数据失败
2. **文件操作失败** - 保存、删除文件失败
3. **系统服务异常** - 缓存、队列、分片表等基础设施错误
4. **关键业务逻辑错误** - 订单创建失败、支付失败等影响业务流程的错误
5. **外部服务调用失败** - 第三方API调用失败

**需要记录到系统日志的警告：**
1. **非关键操作失败** - 不影响主流程但需要关注的问题
2. **资源清理失败** - 临时文件、缓存清理失败等
3. **性能警告** - 需要监控的性能问题

**不需要记录到系统日志的错误：**
1. **参数验证错误** - 如"订单ID不能为空"、"user_id不能为空"
2. **业务逻辑的正常错误** - 如"订单不存在"、"用户不存在"、"记录不存在"
3. **查询不存在的记录** - 这些是正常的业务场景，不是系统错误

## 需要记录的错误清单

### 1. order_service.go

#### ✅ 需要记录
- `fmt.Errorf("获取锁失败: %v", err)` - 系统服务异常
- `fmt.Errorf("生成唯一订单号失败，请重试")` - 关键业务逻辑错误
- `fmt.Errorf("创建订单失败: %v", err)` - 数据库操作失败
- `fmt.Errorf("创建订单详情失败: %v", err)` - 数据库操作失败
- `fmt.Errorf("删除订单详情失败: %v", err)` - 数据库操作失败

#### ❌ 不需要记录
- `fmt.Errorf("订单ID不能为空")` - 参数验证错误
- `fmt.Errorf("订单不存在")` - 业务逻辑的正常错误（多次出现）

### 2. attachment_service.go

#### ✅ 需要记录
- `fmt.Errorf("保存分片失败: %w", err)` - 文件操作失败
- `fmt.Errorf("创建目标目录失败: %w", err)` - 文件操作失败
- `fmt.Errorf("创建目标文件失败: %w", err)` - 文件操作失败
- `fmt.Errorf("写入分片 %d 失败: %w", i, err)` - 文件操作失败
- `fmt.Errorf("关闭目标文件失败: %w", err)` - 文件操作失败
- `fmt.Errorf("创建附件记录失败: %w", err)` - 数据库操作失败
- `fmt.Errorf("保存文件失败: %w", err)` - 文件操作失败
- `fmt.Errorf("更新附件显示名称失败: %v", err)` - 数据库操作失败
- `fmt.Errorf("删除文件失败: %w", err)` - 文件操作失败
- `fmt.Errorf("删除附件记录失败: %w", err)` - 数据库操作失败

#### ❌ 不需要记录
- `fmt.Errorf("分片索引无效")` - 参数验证错误
- `fmt.Errorf("分片 %d 不存在", i)` - 业务逻辑的正常错误
- `fmt.Errorf("附件不存在: %v", err)` - 业务逻辑的正常错误（多次出现）
- `fmt.Errorf("查询附件失败: %v", err)` - 查询错误，但可能是正常的"不存在"情况

### 3. user_service.go

#### ✅ 需要记录
- `fmt.Errorf("更新用户余额失败: %v", err)` - 数据库操作失败
- `fmt.Errorf("创建余额变动记录失败: %v", err)` - 数据库操作失败

#### ❌ 不需要记录
- `fmt.Errorf("用户不存在: %v", err)` - 业务逻辑的正常错误
- `fmt.Errorf("余额不足，当前余额: %.2f", user.Balance)` - 业务逻辑的正常错误
- `fmt.Errorf("无效的变动类型: %s", logType)` - 参数验证错误

### 4. user_balance_log_service.go

#### ✅ 需要记录
- `fmt.Errorf("创建余额变动记录失败: %v", err)` - 数据库操作失败

#### ❌ 不需要记录
- `fmt.Errorf("user_id 不能为空，GORM Sharding 需要 ShardingKey")` - 参数验证错误（多次出现）
- `fmt.Errorf("用户不存在: %v", err)` - 业务逻辑的正常错误

### 5. export_service.go

#### ✅ 需要记录
- `fmt.Errorf("写入CSV表头失败: %w", err)` - 文件操作失败
- `fmt.Errorf("写入CSV数据失败: %w", err)` - 文件操作失败
- `fmt.Errorf("CSV写入失败: %w", err)` - 文件操作失败
- `fmt.Errorf("保存文件失败: %w", err)` - 文件操作失败

#### ❌ 不需要记录
- `fmt.Errorf("Excel导出功能暂未实现，请使用CSV格式")` - 功能未实现，不是错误

### 6. export_record_service.go

#### ✅ 需要记录
- `fmt.Errorf("删除导出记录失败: %v", err)` - 数据库操作失败
- `fmt.Errorf("批量删除导出记录失败: %v", err)` - 数据库操作失败

#### ❌ 不需要记录
- `fmt.Errorf("导出记录不存在: %v", err)` - 业务逻辑的正常错误（多次出现）
- `fmt.Errorf("查询导出记录失败: %v", err)` - 查询错误

### 7. 各种 service 的 GetByID 方法

#### ❌ 不需要记录（统一处理）
以下错误都是查询不存在的记录，属于正常业务场景：
- `fmt.Errorf("系统日志不存在: %v", err)`
- `fmt.Errorf("操作日志不存在: %v", err)`
- `fmt.Errorf("登录日志不存在: %v", err)`
- `fmt.Errorf("部门不存在: %v", err)`
- `fmt.Errorf("权限不存在: %v", err)`
- `fmt.Errorf("角色不存在: %v", err)`
- `fmt.Errorf("黑名单不存在: %v", err)`
- `fmt.Errorf("字典不存在: %v", err)`
- `fmt.Errorf("管理员不存在: %v", err)`

### 8. 基础设施相关

#### ✅ 需要记录
- `providers/database_service_provider.go`: `fmt.Errorf("获取 GORM DB 实例失败: %v", err)` - 系统服务异常
- `providers/database_service_provider.go`: `fmt.Errorf("注册 GORM Sharding 插件失败: %v", err)` - 系统服务异常
- `utils/gorm_sharding.go`: `fmt.Errorf("ORM 未初始化")` - 系统服务异常
- `utils/gorm_sharding.go`: `fmt.Errorf("无法通过反射获取原生 GORM DB 实例...")` - 系统服务异常
- `utils/sharding_helper.go`: `fmt.Errorf("查询分表失败: %v", err)` - 系统服务异常（多次出现）
- `services/sharding_service.go`: `fmt.Errorf("创建分表 %s 失败: %v", tableName, err)` - 系统服务异常
- `console/commands/create_order_sharding_tables.go`: `fmt.Errorf("创建分表 %s 失败: %v", tableName, err)` - 系统服务异常（多次出现）
- `console/commands/queue_clear.go`: 所有 Redis 相关错误 - 系统服务异常
- `console/commands/queue_stats.go`: 所有 Redis 相关错误 - 系统服务异常

#### ❌ 不需要记录
- `console/commands/create_order_sharding_tables.go`: `fmt.Errorf("月份格式错误...")` - 参数验证错误

### 9. 控制器相关

#### ❌ 不需要记录
- `order_controller.go`: 时间格式验证错误 - 参数验证错误

### 10. 其他服务

#### ✅ 需要记录
- `google_authenticator_service.go`: 所有错误 - 外部服务调用失败

#### ❌ 不需要记录
- `admin_service.go`: `fmt.Errorf("查询管理员失败: %v", err)` - 查询错误，可能是正常的"不存在"

## 实现建议

### 方式1：使用 errorlog.RecordHTTP（推荐）

```go
import "goravel/app/utils/errorlog"

// 在 HTTP 请求处理中
if err != nil {
    errorlog.RecordHTTP(ctx, "order", "创建订单失败", map[string]any{
        "error": err.Error(),
        "user_id": userID,
        "amount": amount,
    }, "创建订单失败: %v", err)
    return nil, nil, fmt.Errorf("创建订单失败: %v", err)
}
```

### 方式2：使用 response.ErrorWithLog

```go
import "goravel/app/http/response"

// 在控制器中
if err != nil {
    return response.ErrorWithLog(ctx, http.StatusInternalServerError, "create_failed", "order", "创建订单失败", err)
}
```

### 方式3：使用 systemLogService.RecordHTTP

```go
import "goravel/app/services"

systemLogService := services.NewSystemLogService()
_ = systemLogService.RecordHTTP(ctx, "error", "order", "创建订单失败", map[string]any{
    "error": err.Error(),
    "user_id": userID,
})
```

## 注意事项

1. **避免重复记录**：如果已经在 controller 层记录了日志，service 层就不需要再记录
2. **记录上下文信息**：尽量记录相关的业务数据（如订单ID、用户ID等），方便排查问题
3. **日志级别**：大部分错误使用 "error" 级别，警告使用 "warning" 级别
4. **性能考虑**：系统日志记录是异步的，但也要避免在高频操作中记录过多日志

