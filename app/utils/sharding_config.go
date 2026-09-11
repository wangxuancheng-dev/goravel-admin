package utils

import (
	"fmt"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"
)

// DefaultMaxTimeRangeMonths 默认最大查询时间跨度（月），与 config sharding.max_time_range_months 一致。
const DefaultMaxTimeRangeMonths = 3

// GetTimeShardingSuffixLayout 时间分表后缀格式，默认 200601（按月）。
func GetTimeShardingSuffixLayout() string {
	layout := facades.Config().GetString("sharding.time_suffix_layout", "200601")
	if layout == "" {
		return "200601"
	}
	return layout
}

// GetMaxTimeRangeMonths 列表/导出允许的最大时间跨度（月）。
func GetMaxTimeRangeMonths() int {
	months := cast.ToInt(facades.Config().Get("sharding.max_time_range_months", DefaultMaxTimeRangeMonths))
	if months <= 0 {
		return DefaultMaxTimeRangeMonths
	}
	return months
}

// GetIDLookupScanMonths 仅按 ID 反查分表时向前扫描的月数。
func GetIDLookupScanMonths() int {
	months := cast.ToInt(facades.Config().Get("sharding.id_lookup_scan_months", 6))
	if months <= 0 {
		return 6
	}
	return months
}

// GetUserBalanceLogsShards 用户余额变动记录哈希分表数量。
// 上线后应视为冻结配置；勿热改，否则历史数据路由失效。
func GetUserBalanceLogsShards() int {
	shards := cast.ToInt(facades.Config().Get("sharding.user_balance_logs_shards", 4))
	if shards <= 0 {
		return 4
	}
	return shards
}

// DefaultMaxUnionLimitPerTable 跨分表 UNION 单表拉取上限默认值。
const DefaultMaxUnionLimitPerTable = 10000

// GetMaxUnionLimitPerTable 跨分表 UNION 分页时每个分表最多拉取的行数。
func GetMaxUnionLimitPerTable() int {
	n := cast.ToInt(facades.Config().Get("sharding.max_union_limit_per_table", DefaultMaxUnionLimitPerTable))
	if n <= 0 {
		return DefaultMaxUnionLimitPerTable
	}
	return n
}

// ValidateShardingDeepPagination 校验跨分表深分页：offset+pageSize 不得超过单表拉取上限。
func ValidateShardingDeepPagination(page, pageSize int) error {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	offset := (page - 1) * pageSize
	maxRows := GetMaxUnionLimitPerTable()
	if offset+pageSize > maxRows {
		return fmt.Errorf("deep_pagination_exceeded:%d", maxRows)
	}
	return nil
}

// DeepPaginationMaxFromError 从 ValidateShardingDeepPagination 错误中解析 max（供服务层转 BusinessError）。
func DeepPaginationMaxFromError(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	var max int
	if _, scanErr := fmt.Sscanf(err.Error(), "deep_pagination_exceeded:%d", &max); scanErr != nil {
		return 0, false
	}
	return max, true
}
