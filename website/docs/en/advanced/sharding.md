# Sharding guide

> This page mirrors the Chinese documentation for accuracy. Switch language to **简体中文**, or open the [Chinese version](/advanced/sharding).

---

分表查询/写入请使用 `appfacades.OrmQuery(ctx)`；服务层用 `NewXxxService(ctx)` + `OrmQuery(s.ctx)`。

项目支持两种分表策略：**时间分表**（按月分表）和**哈希分表**（按ID哈希分表）。

## 分表策略概述

### 已实现的分表

#### 已接入：时间分表
- `orders` - 订单主表
- `order_details` - 订单详情表
- `payments` - 支付记录表

#### 已接入：哈希分表
- `user_balance_logs` - 用户余额变动记录表（按 `user_id` 哈希分表，默认 4 个分表；**上线后分片数视为冻结，变更需数据迁移**）

### 分表策略对比

| 特性 | 时间分表 | 哈希分表 |
|------|---------|---------|
| **分表键** | `created_at` (时间) | 业务ID（如 `user_id`） |
| **分表数量** | 动态增长（按月） | 固定数量（可配置） |
| **适用场景** | 数据按时间分布，查询通常有时间范围 | 数据按业务ID分布，查询通常按ID |
| **查询特点** | 可能需要跨多个分表查询 | 通常只查询单个分表 |
| **分表命名** | `{table}_YYYYMM` | `{table}_{shard_index}` |
| **示例** | `orders_202501` | `user_balance_logs_0` |

## 时间分表（按月分表） {#时间分表按月分表}

### 分表策略

- **分表方式**：按月分表
- **分表键字段**：使用 `created_at`（由 `orm.Model` 自动提供），类型为 `time.Time`
- **分表名称格式**：`{base_table_name}_{YYYYMM}`，例如 `orders_202501`
- **表结构定义**：统一在 migrations 中，便于维护和版本控制
- **查询跨月数据**：需要查询多个分表并合并结果（已在 `OrderService` 中实现）

### 创建时间分表

#### 1. 在 Migration 中定义表创建函数

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

#### 2. 在 ShardingService 中注册表创建函数

在 `app/services/sharding_service.go` 中注册表创建函数：

```go
// registerOrderTables 注册订单表的创建函数
func (s *ShardingServiceImpl) registerOrderTables() {
	// 注册订单主表（调用 migrations 中的函数）
	s.RegisterTableCreator("orders", migrations.CreateOrdersShardingTable)
	
	// 注册订单详情表（调用 migrations 中的函数）
	s.RegisterTableCreator("order_details", migrations.CreateOrderDetailsShardingTable)
}

// 同时注册支付分表：
// s.RegisterTableCreator("payments", migrations.CreatePaymentsShardingTable)
```

#### 3. 创建分表命令（可选）

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

#### 4. 定时任务（可选）

在 `app/console/kernel.go` 中添加定时任务，自动创建未来的分表：

```go
// 每月1号凌晨1点执行（UTC时间），创建下个月的订单分表
facades.Schedule().Command("order:create-sharding-tables").Monthly().OnOneServer()
```

### 使用时间分表

#### 在服务层使用分表

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

#### 查询跨月数据

查询跨月数据时，需要使用 `utils.GetShardingTableNames()` 获取所有相关的分表，然后分别查询并合并结果：

```go
// 获取时间范围内的所有分表
tableNames := utils.GetShardingTableNames("orders", startTime, endTime)

// 分别查询每个分表
for _, tableName := range tableNames {
	// 查询逻辑
}
```

## 哈希分表（按ID哈希分表） {#哈希分表按id哈希分表}

### 分表策略

- **分表方式**：按业务ID哈希分表
- **分表键字段**：业务ID（如 `user_id`），类型为 `uint`
- **分表名称格式**：`{base_table_name}_{shard_index}`，例如 `user_balance_logs_0`
- **分表数量**：固定数量，建议为 2 的幂次（如 4, 8, 16, 32, 64 等）。**写入生产数据后视为冻结**，修改 `sharding.user_balance_logs_shards` 必须伴随数据再平衡，否则历史记录路由失效。
- **分表逻辑**：`shardingKey % numberOfShards`
- **查询特点**：通常只查询单个分表，不支持跨分表查询

### 列表深分页与单号契约

- 跨分表 UNION 分页：`offset + page_size` 不得超过 `sharding.max_union_limit_per_table`（默认 10000），超限返回 `deep_pagination_exceeded`。
- 订单/支付写路径（更新、删除、状态变更）必须带业务单号（`order_no` / `payment_no`），不要只靠自增 `id`。
- 订单后台列表在搜索同步开启时优先走搜索引擎，失败再回退分表 DB。
- 生产环境建议依赖定时预建表；`EnsureShardingTable` 仅为兜底（带短缓存与同名表创建锁）。

### 创建哈希分表

#### 1. 在 Migration 中定义表创建函数

在对应的 migration 文件中添加创建分表的函数，例如 `database/migrations/20250130000002_create_user_balance_logs_table.go`：

```go
// CreateUserBalanceLogsShardingTable 创建用户余额变动记录分表（供服务层调用）
func CreateUserBalanceLogsShardingTable(tableName string) error {
	return facades.Schema().Create(tableName, func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.UnsignedBigInteger("user_id").Comment("用户ID")
		table.String("type", 20).Comment("变动类型:income收入,expense支出,refund退款")
		// ... 其他字段定义
		table.Index("user_id")
		table.Comment(fmt.Sprintf("用户余额变动记录表 - %s", tableName))
	})
}
```

#### 2. 在 ShardingService 中注册表创建函数

在 `app/services/sharding_service.go` 中注册表创建函数：

```go
// registerUserBalanceLogTables 注册用户余额变动记录表的创建函数
func (s *ShardingServiceImpl) registerUserBalanceLogTables() {
	// 注册用户余额变动记录表（调用 migrations 中的函数）
	s.RegisterTableCreator("user_balance_logs", migrations.CreateUserBalanceLogsShardingTable)
}
```

#### 3. 定义分表数量常量（可选）

在 `app/constants/sharding.go` 中定义分表数量：

```go
// UserBalanceLogsShards 用户余额变动记录表的分表数量
// 建议设置为 2 的幂次（如 4, 8, 16, 32, 64, 128 等）
const UserBalanceLogsShards = 4
```

### 使用哈希分表

#### 在服务层使用分表

```go
// 根据 user_id 计算分表名称
tableName := utils.GetHashShardingTableName("user_balance_logs", userID, constants.UserBalanceLogsShards)

// 确保分表存在
if err := s.shardingService.EnsureShardingTable(tableName, "user_balance_logs"); err != nil {
	return err
}

// 使用分表进行查询（必须包含分表键字段；v1.18 使用 OrmQuery(ctx)）
appfacades.OrmQuery(s.ctx).Table(tableName).Where("user_id", userID).First(&log)
```

#### 注意事项

1. **所有查询必须包含分表键字段**：哈希分表需要分表键来路由到正确的分表
2. **不支持跨分表查询**：如果需要查询多个ID的数据，需要分别查询后合并
3. **分表自动创建**：首次插入数据时，会自动创建对应的分表（通过 `EnsureShardingTable`）

## 为新的表添加分表支持

### 添加时间分表支持

参考 [时间分表（按月分表）](#时间分表按月分表) 章节的步骤。

### 添加哈希分表支持

#### 步骤 1: 创建 Migration 文件

创建 migration 文件，定义表创建函数：

```go
package migrations

import (
	"fmt"
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20250101000001CreateExampleTable struct {
}

func (r *M20250101000001CreateExampleTable) Signature() string {
	return "20250101000001_create_example_table"
}

func (r *M20250101000001CreateExampleTable) Up() error {
	// 使用哈希分表，不创建基础表
	// 分表通过 CreateExampleShardingTable 函数创建
	return nil
}

func (r *M20250101000001CreateExampleTable) Down() error {
	// 哈希分表，不删除基础表（因为基础表不存在）
	return nil
}

// CreateExampleShardingTable 创建示例表分表（供服务层调用）
func CreateExampleShardingTable(tableName string) error {
	return facades.Schema().Create(tableName, func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.UnsignedBigInteger("entity_id").Comment("实体ID（分表键）")
		// ... 其他字段定义
		table.Index("entity_id")
		table.Comment(fmt.Sprintf("示例表 - %s", tableName))
	})
}
```

#### 步骤 2: 定义分表数量常量

在 `app/constants/sharding.go` 中添加：

```go
// ExampleShards 示例表的分表数量
// 建议设置为 2 的幂次（如 4, 8, 16, 32, 64, 128 等）
const ExampleShards = 8
```

#### 步骤 3: 在 ShardingService 中注册

在 `app/services/sharding_service.go` 的 `NewShardingService()` 方法中注册：

```go
func NewShardingService() ShardingService {
	service := &ShardingServiceImpl{
		creators: make(map[string]TableCreator),
	}

	// 注册订单相关表的创建函数
	service.registerOrderTables()

	// 注册用户余额变动记录表的创建函数
	service.registerUserBalanceLogTables()

	// 注册示例表的创建函数
	service.RegisterTableCreator("example_table", migrations.CreateExampleShardingTable)

	return service
}
```

#### 步骤 4: 在服务层使用

```go
// 根据 entity_id 计算分表名称
tableName := utils.GetHashShardingTableName("example_table", entityID, constants.ExampleShards)

// 确保分表存在
if err := s.shardingService.EnsureShardingTable(tableName, "example_table"); err != nil {
	return err
}

// 使用分表进行写入（v1.18 使用 OrmQuery(ctx)）
appfacades.OrmQuery(s.ctx).Table(tableName).Where("entity_id", entityID).Create(&example)
```

## 分表字段修改

当表使用分表策略时，如果需要添加或修改字段，需要同时更新：

1. **表创建函数**（用于新创建的分表）
2. **所有已存在的分表**（通过 migration）

### 修改表创建函数

在对应的 migration 文件中修改表创建函数，添加新字段。

### 创建 Migration 修改已存在的分表

创建一个新的 migration 文件，使用 `utils.GetAllExistingShardingTables()` 或 `utils.GetAllExistingShardingTablesByPattern()` 获取所有已存在的分表，然后逐个修改：

```go
package migrations

import (
	"fmt"
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
	"goravel/app/utils"
)

type M20251228000001AddFieldToExampleTable struct {
}

func (r *M20251228000001AddFieldToExampleTable) Signature() string {
	return "20251228000001_add_field_to_example_table"
}

func (r *M20251228000001AddFieldToExampleTable) Up() error {
	// 获取所有已存在的分表（哈希分表使用模式匹配）
	exampleTables, err := utils.GetAllExistingShardingTablesByPattern("example_table_%")
	if err != nil {
		return fmt.Errorf("获取示例表分表列表失败: %v", err)
	}

	// 遍历所有分表，添加字段
	for _, tableName := range exampleTables {
		if !facades.Schema().HasTable(tableName) {
			continue
		}

		// 检查字段是否已存在（避免重复添加）
		if !facades.Schema().HasColumn(tableName, "new_field") {
			if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
				table.String("new_field", 100).Nullable().Comment("新字段").After("existing_field")
			}); err != nil {
				return fmt.Errorf("修改分表 %s 失败: %v", tableName, err)
			}
			facades.Log().Infof("✓ 已为分表 %s 添加字段 new_field", tableName)
		}
	}

	return nil
}

func (r *M20251228000001AddFieldToExampleTable) Down() error {
	// 回滚操作：删除添加的字段
	exampleTables, err := utils.GetAllExistingShardingTablesByPattern("example_table_%")
	if err != nil {
		return fmt.Errorf("获取示例表分表列表失败: %v", err)
	}

	for _, tableName := range exampleTables {
		if !facades.Schema().HasTable(tableName) {
			continue
		}

		if facades.Schema().HasColumn(tableName, "new_field") {
			if err := facades.Schema().Table(tableName, func(table schema.Blueprint) {
				table.DropColumn("new_field")
			}); err != nil {
				return fmt.Errorf("回滚分表 %s 失败: %v", tableName, err)
			}
		}
	}

	return nil
}
```

### 注意事项

1. **字段检查**：在添加字段前，务必检查字段是否已存在，避免重复添加导致错误
2. **表检查**：在操作前检查表是否存在
3. **数据备份**：修改字段类型或删除字段前，请先备份数据
4. **新分表**：修改表创建函数后，新创建的分表会自动包含新字段
5. **已存在分表**：需要通过 migration 手动修改所有已存在的分表
6. **回滚支持**：务必实现 `Down()` 方法，支持 migration 回滚

## 工具函数

`app/utils/sharding_helper.go` 提供了以下工具函数：

### 时间分表工具函数

- `GetShardingTableName(baseTableName string, orderTime time.Time)`: 根据时间获取分表名称
- `GetShardingTableNames(baseTableName string, startTime, endTime time.Time)`: 获取时间范围内的所有分表名称
- `ValidateTimeRange(startTime, endTime time.Time, maxMonths ...int)`: 验证时间范围是否超过指定月数

### 哈希分表工具函数

- `GetHashShardingTableName(baseTableName string, shardingKey uint, numberOfShards int)`: 通用的哈希分表名称生成函数
- `GetUserBalanceLogsShardingTableName(userID uint)`: 用户余额变动记录表的分表名称（特定实现）

### 通用工具函数

- `GetAllExistingShardingTables(baseTableName string)`: 获取所有已存在的时间分表名称（格式：`{table}_YYYYMM`）
- `GetAllExistingShardingTablesByPattern(pattern string)`: 通过表名模式获取所有已存在的分表（适用于哈希分表）

### 使用示例

#### 时间分表

```go
// 根据时间获取分表名称
tableName := utils.GetShardingTableName("orders", time.Now())

// 获取时间范围内的所有分表
tableNames := utils.GetShardingTableNames("orders", startTime, endTime)
```

#### 哈希分表

```go
// 通用的哈希分表名称生成
tableName := utils.GetHashShardingTableName("example_table", entityID, 8)

// 特定表的便捷函数（内部调用通用函数）
tableName := utils.GetUserBalanceLogsShardingTableName(userID)
```

## 完整示例

### 时间分表示例

参考 `database/migrations/20250128000001_create_orders_table.go` 和 `app/services/order_service.go`。

### 哈希分表示例

参考 `database/migrations/20250130000002_create_user_balance_logs_table.go` 和 `app/services/user_balance_log_service.go`。

## 常见问题

### Q: 如何选择分表策略？

**A:** 
- **时间分表**：适用于数据按时间分布，查询通常有时间范围，如订单、日志等
- **哈希分表**：适用于数据按业务ID分布，查询通常按ID，如用户相关数据、余额记录等

### Q: 哈希分表数量如何确定？

**A:** 
- 建议设置为 2 的幂次（如 4, 8, 16, 32, 64, 128 等）
- 根据数据量和查询负载确定，一般 4-16 个分表即可
- 分表数量一旦确定，不建议频繁修改（需要数据迁移）

### Q: 如何查询跨分表数据？

**A:** 
- **时间分表**：使用 `GetShardingTableNames()` 获取所有相关分表，分别查询后合并结果
- **哈希分表**：不支持跨分表查询，需要分别查询每个分表后合并结果

### Q: 分表字段修改后，已存在的分表怎么办？

**A:** 创建 migration，使用 `GetAllExistingShardingTables()` 或 `GetAllExistingShardingTablesByPattern()` 获取所有已存在的分表，然后逐个修改。

