package migrations

import (
	"fmt"

	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20250128000001CreateOrdersTable struct {
}

func (r *M20250128000001CreateOrdersTable) Signature() string {
	return "20250128000001_create_orders_table"
}

func (r *M20250128000001CreateOrdersTable) Up() error {
	// 订单表使用按月分表策略，不创建基础表
	// 分表通过命令 order:create-sharding-tables 创建，格式为 orders_YYYYMM 和 order_details_YYYYMM
	// 也可以在创建订单时自动创建（如果分表不存在）
	// 表结构定义在 CreateOrdersShardingTable 和 CreateOrderDetailsShardingTable 函数中
	return nil
}

func (r *M20250128000001CreateOrdersTable) Down() error {
	// 订单表使用分表，不删除基础表（因为基础表不存在）
	// 如需删除分表，请手动执行 DROP TABLE 语句
	return nil
}

// CreateOrdersShardingTable 创建订单主表分表（供服务层和命令层调用）
func CreateOrdersShardingTable(tableName string) error {
	return facades.Schema().Create(tableName, func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.String("order_no", 50).Comment("订单号")
		table.UnsignedBigInteger("user_id").Comment("用户ID")
		table.Decimal("amount").Comment("订单金额(10,2)")
		table.String("status", 20).Default("pending").Comment("订单状态 pending:待支付 paid:已支付 cancelled:已取消")
		table.Text("remark").Nullable().Comment("备注")
		table.Timestamp("expire_at").Nullable().Comment("pending order auto-cancel deadline (UTC)")
		table.Timestamps()
		table.SoftDeletes()
		table.Unique("order_no") // 唯一索引，防止并发下订单号重复

		// 复合索引优化原则：等值查询字段在前，范围查询字段在后
		// created_at 是范围查询(>=, <=)，必须放在复合索引最后
		// 1. 状态筛选 + 时间范围（按状态筛选订单）
		table.Index("status", "created_at")
		// 2. 用户ID + 时间范围（查询特定用户的订单）
		table.Index("user_id", "created_at")
		// 3. 用户ID + 状态 + 时间范围（同时按用户和状态筛选）
		table.Index("user_id", "status", "created_at")
		// 4. 仅时间范围查询（无其他筛选条件时使用）
		table.Index("created_at")
		// 5. 按金额排序优化（ORDER BY amount + WHERE created_at 范围）
		table.Index("amount", "created_at")
		// 6. unpaid auto-cancel sweep
		table.Index("status", "expire_at")

		table.Comment(fmt.Sprintf("订单主表 - %s", tableName))
	})
}

// CreateOrderDetailsShardingTable 创建订单详情表分表（供服务层和命令层调用）
func CreateOrderDetailsShardingTable(tableName string) error {
	return facades.Schema().Create(tableName, func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.UnsignedBigInteger("order_id").Comment("订单ID")
		table.UnsignedBigInteger("product_id").Comment("商品ID")
		table.String("product_name", 200).Comment("商品名称")
		table.Decimal("price").Comment("单价(10,2)")
		table.Integer("quantity").Comment("数量")
		table.Decimal("subtotal").Comment("小计(10,2)")
		table.Timestamps()
		table.SoftDeletes()
		table.Index("order_id")
		table.Index("product_id")
		table.Index("product_name")
		table.Index("created_at")
		table.Comment(fmt.Sprintf("订单详情表 - %s", tableName))
	})
}
