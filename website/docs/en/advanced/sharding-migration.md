# Sharding migration

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/advanced/sharding-migration).

---

> **Goravel v1.18**：分表查询请使用 `appfacades.OrmQuery(ctx)`。详见 [分表指南 (SHARDING_GUIDE.md)](/advanced/sharding)。

> **注意**：本文档主要介绍分表字段修改的详细步骤。如需了解完整的分表功能说明，请参考 [分表指南 (SHARDING_GUIDE.md)](/advanced/sharding)。

项目支持两种分表策略：**时间分表**（按月分表）和**哈希分表**（按ID哈希分表）。本文档主要介绍如何修改分表字段。

## 分表功能概述

### 分表策略

- **分表方式**：按月分表
- **分表键字段**：使用 `created_at`（由 `orm.Model` 自动提供），类型为 `time.Time`
- **分表名称格式**：`{base_table_name}_{YYYYMM}`，例如 `orders_202501`
- **表结构定义**：统一在 migrations 中，便于维护和版本控制
- **查询跨月数据**：需要查询多个分表并合并结果（已在 `OrderService` 中实现）

### 已实现的分表

- `orders` - 订单主表
- `order_details` - 订单详情表
- `payments` - 支付记录表
- `user_balance_logs` - 用户余额流水（哈希分表）

## 创建分表

### 1. 在 Migration 中定义表创建函数

在对应的 migration 文件中添加创建分表的函数，例如 `database/migrations/20250128000001_create_orders_table.go`：

```go
// CreateOrdersShardingTable 创建订单主表分表（供服务层和命令层调用）
func CreateOrdersShardingTable(tableName string) error {
	return facades.Schema().Create(tableName, func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.String("order_no", 50).Comment("订单号")
		// ... 其他字段定义
		table.Index("order_no")
		table.Comment(fmt.Sprintf("订单主表 - %s", tableName))
	})
}
```

### 2. 在 ShardingService 中注册表创建函数

在 `app/services/sharding_service.go` 的 `registerOrderTables` 方法中（或创建新的注册方法）注册表创建函数：

```go
// registerOrderTables 注册订单表的创建函数
func (s *ShardingServiceImpl) registerOrderTables() {
	// 注册订单主表（调用 migrations 中的函数）
	s.RegisterTableCreator("orders", migrations.CreateOrdersShardingTable)
	
	// 注册订单详情表（调用 migrations 中的函数）
	s.RegisterTableCreator("order_details", migrations.CreateOrderDetailsShardingTable)
}
```

### 3. 创建分表命令（可选）

如需手动创建分表，可以参考 `app/console/commands/create_order_sharding_tables.go` 创建类似的命令。

**命令使用示例：**

```bash
# 创建分表（默认创建上个月、当前月份及未来2个月，共4个月）
go run . artisan order:create-sharding-tables

# 创建指定月份的分表
go run . artisan order:create-sharding-tables --month=202512

# 创建上个月、当前月份及未来N个月的分表（--months 参数包括上个月和当前月）
# 例如：--months=6 会创建上个月、当前月及未来4个月，共6个月
go run . artisan order:create-sharding-tables --months=6
```

### 4. 定时任务（可选）

在 `app/console/kernel.go` 中添加定时任务，自动创建未来的分表：

```go
// 每月1号凌晨1点执行（UTC时间），创建下个月的订单分表
facades.Schedule().Command("order:create-sharding-tables").Monthly().OnOneServer()
```

## 使用分表

### 在服务层使用分表

在服务层使用 `ShardingService` 确保分表存在：

```go
// 确保分表存在（使用订单的创建时间）
now := time.Now().UTC()
tableName := utils.GetShardingTableName("orders", now)
if err := s.shardingService.EnsureShardingTable(tableName, "orders"); err != nil {
	return err
}

// 使用分表进行查询（v1.18：传递 Service 持有的 ctx）
appfacades.OrmQuery(s.ctx).Table(tableName).Where("id", orderID).First(&order)
```

### 查询跨月数据

查询跨月数据时，需要使用 `utils.GetShardingTableNames()` 获取所有相关的分表，然后分别查询并合并结果：

```go
// 获取时间范围内的所有分表
tableNames := utils.GetShardingTableNames("orders", startTime, endTime)

// 分别查询每个分表
for _, tableName := range tableNames {
	// 查询逻辑
}
```

## 分表字段修改

当订单表使用分表策略（按月分表）时，如果需要添加或修改字段，需要同时更新：

1. **表创建函数**（用于新创建的分表）
2. **所有已存在的分表**（通过 migration）

## 步骤说明

### 1. 修改表创建函数

在 `database/migrations/20250128000001_create_orders_table.go` 中修改 `CreateOrdersShardingTable` 和/或 `CreateOrderDetailsShardingTable` 函数，添加新字段：

```go
// CreateOrdersShardingTable 创建订单主表分表（供服务层和命令层调用）
func CreateOrdersShardingTable(tableName string) error {
	return facades.Schema().Create(tableName, func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.String("order_no", 50).Comment("订单号")
		table.UnsignedBigInteger("user_id").Comment("用户ID")
		table.Decimal("amount").Comment("订单金额(10,2)")
		table.String("status", 20).Default("pending").Comment("订单状态 pending:待支付 paid:已支付 cancelled:已取消")
		table.Text("remark").Nullable().Comment("备注")
		
		// 新添加的字段
		table.String("payment_method", 50).Nullable().Comment("支付方式: alipay, wechat, bank")
		
		table.Timestamps()
		table.SoftDeletes()
		table.Unique("order_no")
		table.Index("user_id")
		table.Index("created_at")
		table.Comment(fmt.Sprintf("订单主表 - %s", tableName))
	})
}
```

### 2. 创建 Migration 修改已存在的分表

创建一个新的 migration 文件，根据分表类型选择合适的方法获取所有已存在的分表：

#### 时间分表（按月分表）

使用 `utils.GetAllExistingShardingTables()` 获取所有已存在的时间分表：

```go
package migrations

import (
	"fmt"
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
	"goravel/app/utils"
)

type M20251228000001AddPaymentMethodToOrders struct {
}

func (r *M20251228000001AddPaymentMethodToOrders) Signature() string {
	return "20251228000001_add_payment_method_to_orders"
}

func (r *M20251228000001AddPaymentMethodToOrders) Up() error {
	// 获取所有已存在的订单主表分表（时间分表）
	ordersTables, err := utils.GetAllExistingShardingTables("orders")
	if err != nil {
		return fmt.Errorf("获取订单分表列表失败: %v", err)
	}

	// 遍历所有分表，添加字段
	for _, tableName := range ordersTables {
		if !facades.Schema().HasTable(tableName) {
			continue
		}

		// 检查字段是否已存在（避免重复添加）
		if !facades.Schema().HasColumn(tableName, "payment_method") {
			if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
				table.String("payment_method", 50).Nullable().Comment("支付方式: alipay, wechat, bank").After("status")
			}); err != nil {
				return fmt.Errorf("修改分表 %s 失败: %v", tableName, err)
			}
			facades.Log().Infof("✓ 已为分表 %s 添加字段 payment_method", tableName)
		}
	}

	return nil
}

func (r *M20251228000001AddPaymentMethodToOrders) Down() error {
	// 回滚操作：删除添加的字段
	ordersTables, err := utils.GetAllExistingShardingTables("orders")
	if err != nil {
		return fmt.Errorf("获取订单分表列表失败: %v", err)
	}

	for _, tableName := range ordersTables {
		if !facades.Schema().HasTable(tableName) {
			continue
		}

		if facades.Schema().HasColumn(tableName, "payment_method") {
			if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
				table.DropColumn("payment_method")
			}); err != nil {
				return fmt.Errorf("回滚分表 %s 失败: %v", tableName, err)
			}
		}
	}

	return nil
}
```

### 3. 注册 Migration

在 `database/kernel.go` 的 `Migrations()` 方法中注册新的 migration：

```go
func (kernel Kernel) Migrations() []schema.Migration {
	return []schema.Migration{
		// ... 其他 migrations
		&migrations.M20250128000001CreateOrdersTable{},
		&migrations.M20251228000001AddPaymentMethodToOrders{}, // 新添加的 migration
	}
}
```

### 4. 执行 Migration

运行 migration：

```bash
go run . artisan migrate
```

## 常用操作示例

### 添加字段

```go
if !facades.Schema().HasColumn(tableName, "new_field") {
	facades.Schema().Table(tableName, func(table schema.Blueprint) {
		table.String("new_field", 100).Nullable().Comment("新字段").After("existing_field")
	})
}
```

### 修改字段类型

注意：某些数据库可能不支持直接修改字段类型，需要先删除再添加（会丢失数据，请谨慎操作）：

```go
// 方式1: 直接修改（如果数据库支持）
facades.Schema().Table(tableName, func(table schema.Blueprint) {
	table.String("field_name", 200).Change() // 修改字段长度
})

// 方式2: 删除后重新添加（会丢失数据）
if facades.Schema().HasColumn(tableName, "field_name") {
	facades.Schema().Table(tableName, func(table schema.Blueprint) {
		table.DropColumn("field_name")
	})
	facades.Schema().Table(tableName, func(table schema.Blueprint) {
		table.String("field_name", 200).Comment("新字段")
	})
}
```

### 删除字段

```go
if facades.Schema().HasColumn(tableName, "field_name") {
	facades.Schema().Table(tableName, func(table schema.Blueprint) {
		table.DropColumn("field_name")
	})
}
```

### 添加索引

```go
indexes, _ := facades.Schema().GetIndexes(tableName)
hasIndex := false
for _, index := range indexes {
	if index.Name == "index_name" {
		hasIndex = true
		break
	}
}

if !hasIndex {
	facades.Schema().Table(tableName, func(table schema.Blueprint) {
		table.Index("field_name", "index_name")
	})
}
```

### 删除索引

```go
facades.Schema().Table(tableName, func(table schema.Blueprint) {
	table.DropIndex("index_name")
})
```

## 注意事项

1. **字段检查**：在添加字段前，务必检查字段是否已存在，避免重复添加导致错误
2. **表检查**：在操作前检查表是否存在
3. **数据备份**：修改字段类型或删除字段前，请先备份数据
4. **新分表**：修改表创建函数后，新创建的分表会自动包含新字段
5. **已存在分表**：需要通过 migration 手动修改所有已存在的分表
6. **回滚支持**：务必实现 `Down()` 方法，支持 migration 回滚

## 工具函数

`app/utils/sharding_helper.go` 提供了以下工具函数：

### 时间分表工具函数

- `GetAllExistingShardingTables(baseTableName string)`: 获取所有已存在的时间分表名称（格式：`{table}_YYYYMM`）
- `GetShardingTableName(baseTableName string, orderTime time.Time)`: 根据时间获取分表名称
- `GetShardingTableNames(baseTableName string, startTime, endTime time.Time)`: 获取时间范围内的所有分表名称

### 哈希分表工具函数

- `GetAllExistingShardingTablesByPattern(pattern string)`: 通过表名模式获取所有已存在的分表（适用于哈希分表，如 `"user_balance_logs_%"`）
- `GetHashShardingTableName(baseTableName string, shardingKey uint, numberOfShards int)`: 通用的哈希分表名称生成函数

### 使用建议

- **时间分表**：使用 `GetAllExistingShardingTables("orders")` 获取所有分表
- **哈希分表**：使用 `GetAllExistingShardingTablesByPattern("user_balance_logs_%")` 获取所有分表

## 完整示例

参考 `database/migrations/20251228001858_example_modify_orders_sharding_tables.go` 查看完整示例。

