package config

import (
	"github.com/goravel/framework/facades"
)

func init() {
	config := facades.Config()
	config.Add("sharding", map[string]any{
		// 时间分表后缀格式（Go time.Format 布局），默认按月 orders_202501
		"time_suffix_layout": config.Env("SHARDING_TIME_SUFFIX_LAYOUT", "200601"),
		// 列表/导出查询允许的最大时间跨度（月），避免跨过多分表
		"max_time_range_months": config.Env("SHARDING_MAX_TIME_RANGE_MONTHS", 3),
		// 仅 ID 反查分表时向前扫描的月数（无 order_no 提示时）
		"id_lookup_scan_months": config.Env("SHARDING_ID_LOOKUP_SCAN_MONTHS", 6),
		// 用户余额变动记录哈希分表数量（建议 2 的幂）。
	// 警告：上线写入数据后视为冻结；变更分片数必须做数据迁移/再平衡，否则路由错乱。
		"user_balance_logs_shards": config.Env("SHARDING_USER_BALANCE_LOGS_SHARDS", 4),
		// 跨分表 UNION 分页时，单表最多拉取行数；offset+page_size 超过此值将拒绝深分页
		"max_union_limit_per_table": config.Env("SHARDING_MAX_UNION_LIMIT_PER_TABLE", 10000),
	})
}
