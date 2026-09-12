# 分表查询服务使用指南

## 概述

`ShardingQueryService` 是一个通用的分表查询服务，用于简化多分表查询的实现。它封装了 UNION ALL 查询、分页、排序等通用逻辑，让开发者只需要关注业务特定的筛选条件和列定义。

## 核心优势

1. **代码复用**：避免为每个分表重复编写 UNION ALL 查询逻辑
2. **统一优化**：所有分表查询都使用相同的优化策略（表存在性检查、列名明确指定等）
3. **易于维护**：通用逻辑集中管理，修改一处即可影响所有使用该服务的表
4. **类型安全**：通过接口和回调函数保证类型安全

## 使用步骤

### 1. 定义筛选条件结构体

```go
// 例如：日志表的筛选条件
type LogFilters struct {
    UserID    uint      // 用户ID
    Level     string    // 日志级别
    Keyword   string    // 关键词搜索
    StartTime time.Time // 开始时间
    EndTime   time.Time // 结束时间
    OrderBy   string    // 排序字段（格式：字段:asc/desc）
}
```

### 2. 实现 WHERE 条件构建函数

```go
// buildLogWhereClause 构建日志查询的 WHERE 条件
func (s *LogServiceImpl) buildLogWhereClause(filters any) (string, []any) {
    logFilters, ok := filters.(LogFilters)
    if !ok {
        return "", nil
    }

    var conditions []string
    var args []any

    // 时间范围
    if !logFilters.StartTime.IsZero() {
        conditions = append(conditions, "created_at >= ?")
        args = append(args, logFilters.StartTime)
    }
    if !logFilters.EndTime.IsZero() {
        conditions = append(conditions, "created_at <= ?")
        args = append(args, logFilters.EndTime)
    }

    // 用户ID筛选
    if logFilters.UserID > 0 {
        conditions = append(conditions, "user_id = ?")
        args = append(args, logFilters.UserID)
    }

    // 日志级别筛选
    if logFilters.Level != "" {
        conditions = append(conditions, "level = ?")
        args = append(args, logFilters.Level)
    }

    // 关键词搜索
    if logFilters.Keyword != "" {
        conditions = append(conditions, "(message LIKE ? OR context LIKE ?)")
        keyword := "%" + logFilters.Keyword + "%"
        args = append(args, keyword, keyword)
    }

    return strings.Join(conditions, " AND "), args
}
```

### 3. 实现列名获取函数

```go
// getLogTableColumns 获取日志表的所有列名
func (s *LogServiceImpl) getLogTableColumns() string {
    // 只包含模型中定义的字段，确保所有分表都有这些字段
    columns := []string{
        "id",
        "user_id",
        "level",
        "message",
        "context",
        "created_at",
        "updated_at",
        "deleted_at",
    }
    return strings.Join(columns, ", ")
}
```

### 4. 初始化分表查询服务

```go
type LogServiceImpl struct {
    shardingService      ShardingService
    shardingQueryService ShardingQueryService
}

func NewLogService() *LogServiceImpl {
    service := &LogServiceImpl{
        shardingService: NewShardingService(),
    }

    // 初始化分表查询服务
    service.shardingQueryService = NewShardingQueryService(ShardingQueryConfig{
        BaseTableName: "logs",
        GetColumns: func() string {
            return service.getLogTableColumns()
        },
        BuildWhereClause: func(filters any) (string, []any) {
            return service.buildLogWhereClause(filters)
        },
        GetAllowedOrderFields: func() map[string]bool {
            return map[string]bool{
                "id":         true,
                "user_id":    true,
                "level":       true,
                "created_at": true,
                "updated_at": true,
            }
        },
        DefaultOrderBy: "created_at:desc",
        ModuleName:     "log",
    })

    return service
}
```

### 5. 使用分表查询服务

```go
// GetLogs 查询日志列表（支持多分表）
func (s *LogServiceImpl) GetLogs(filters LogFilters, page, pageSize int) ([]models.Log, int64, error) {
    // 验证时间范围
    valid, err := utils.ValidateTimeRange(filters.StartTime, filters.EndTime)
    if !valid {
        return nil, 0, err
    }

    // 获取需要查询的所有分表
    tableNames := utils.GetShardingTableNames("logs", filters.StartTime, filters.EndTime)
    if len(tableNames) == 0 {
        return []models.Log{}, 0, nil
    }

    // 如果只有一个分表，直接查询（使用 ORM）
    if len(tableNames) == 1 {
        return s.querySingleTable(tableNames[0], filters, page, pageSize)
    }

    // 多个分表：使用通用分表查询服务
    var logs []models.Log
    total, err := s.shardingQueryService.QueryMultipleTables(tableNames, filters, page, pageSize, &logs)
    if err != nil {
        return nil, 0, err
    }
    return logs, total, nil
}

// GetAllLogsForExport 获取所有日志用于导出
func (s *LogServiceImpl) GetAllLogsForExport(filters LogFilters) ([]models.Log, error) {
    // 验证时间范围
    valid, err := utils.ValidateTimeRange(filters.StartTime, filters.EndTime)
    if !valid {
        return nil, err
    }

    // 获取需要查询的所有分表
    tableNames := utils.GetShardingTableNames("logs", filters.StartTime, filters.EndTime)
    if len(tableNames) == 0 {
        return []models.Log{}, nil
    }

    // 如果只有一个分表，直接查询
    if len(tableNames) == 1 {
        query := s.buildShardingQuery(tableNames[0], filters)
        orderBy := filters.OrderBy
        if orderBy == "" {
            orderBy = "created_at:desc"
        }
        query = s.applyOrderBy(query, orderBy)

        var logs []models.Log
        if err := query.Find(&logs); err != nil {
            return nil, apperrors.ErrQueryFailed.WithError(err)
        }
        return logs, nil
    }

    // 多个分表：使用通用分表查询服务
    var logs []models.Log
    err = s.shardingQueryService.QueryMultipleTablesForExport(tableNames, filters, &logs)
    if err != nil {
        return nil, err
    }
    return logs, nil
}
```

## 配置说明

### ShardingQueryConfig 字段说明

| 字段 | 类型 | 说明 | 必填 |
|------|------|------|------|
| `BaseTableName` | `string` | 基础表名，如 "orders" | 是 |
| `GetColumns` | `func() string` | 获取表的所有列名（用于 UNION ALL 查询） | 是 |
| `BuildWhereClause` | `func(any) (string, []any)` | 构建 WHERE 条件，返回条件字符串和参数列表 | 是 |
| `GetAllowedOrderFields` | `func() map[string]bool` | 获取允许排序的字段列表 | 是 |
| `DefaultOrderBy` | `string` | 默认排序，格式：字段:方向，如 "created_at:desc" | 否（默认：created_at:desc） |
| `ModuleName` | `string` | 模块名称，用于日志记录 | 否（默认：sharding） |

## 注意事项

1. **列名必须明确指定**：不要使用 `SELECT *`，必须明确列出所有列名，确保不同分表的列数一致
2. **只包含模型中的字段**：列名列表应该只包含模型中定义的字段，避免查询不存在的字段
3. **WHERE 条件格式**：`BuildWhereClause` 返回的条件字符串不应该包含 `WHERE` 关键字，只返回条件部分（如 "user_id = ? AND status = ?"）
4. **时间范围处理**：如果筛选条件包含时间范围，应该在调用 `QueryMultipleTables` 之前验证时间范围
5. **单表优化**：如果只有一个分表，建议直接使用 ORM 查询，而不是使用 UNION ALL（性能更好）

## 完整示例

参考 `app/services/order_service.go` 中的实现，这是使用通用分表查询服务的完整示例。

## 优势对比

### 使用通用服务前（每个表都需要重复实现）

```go
// 每个表都需要实现这些方法
func (s *OrderServiceImpl) queryMultipleTablesWithUnion(...) { /* 100+ 行代码 */ }
func (s *LogServiceImpl) queryMultipleTablesWithUnion(...) { /* 100+ 行代码 */ }
func (s *PaymentServiceImpl) queryMultipleTablesWithUnion(...) { /* 100+ 行代码 */ }
// ... 每个表都重复
```

### 使用通用服务后（只需配置）

```go
// 每个表只需要配置一次
service.shardingQueryService = NewShardingQueryService(ShardingQueryConfig{
    // ... 配置
})

// 使用时只需一行代码
total, err := s.shardingQueryService.QueryMultipleTables(tableNames, filters, page, pageSize, &results)
```

## 扩展性

如果需要添加新的通用功能（如缓存、性能监控等），只需要在 `ShardingQueryService` 中修改一次，所有使用该服务的表都会自动获得这些功能。

