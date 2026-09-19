package utils

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"

	"goravel/app/utils/errorlog"
)

// GetShardingTableName 根据时间获取分表名称
// baseTableName: 基础表名，如 "orders"
// orderTime: 订单时间
// 返回: 分表名称，如 "orders_202501"
func GetShardingTableName(baseTableName string, orderTime time.Time) string {
	return fmt.Sprintf("%s_%s", baseTableName, orderTime.Format(GetTimeShardingSuffixLayout()))
}

// GetUserBalanceLogsShardingTableName 根据 user_id 获取用户余额变动记录分表名称
// userID: 用户ID
// 返回: 分表名称，如 "user_balance_logs_0", "user_balance_logs_1" 等
// 分表逻辑：user_id % UserBalanceLogsShards
// 注意：此函数为特定表实现，如需为其他表实现哈希分表，请使用 GetHashShardingTableName
func GetUserBalanceLogsShardingTableName(userID uint) string {
	return GetHashShardingTableName("user_balance_logs", userID, GetUserBalanceLogsShards())
}

// HashShardingConfig 哈希分表配置
type HashShardingConfig struct {
	BaseTableName  string // 基础表名，如 "user_balance_logs"
	NumberOfShards int    // 分表数量，建议为 2 的幂次（如 4, 8, 16, 32, 64 等）
}

// GetUserBalanceLogsShardingConfig 用户余额变动记录分表配置（运行时读取 config）。
func GetUserBalanceLogsShardingConfig() HashShardingConfig {
	return HashShardingConfig{
		BaseTableName:  "user_balance_logs",
		NumberOfShards: GetUserBalanceLogsShards(),
	}
}

// GetHashShardingTableName 通用的哈希分表名称生成函数
// baseTableName: 基础表名，如 "user_balance_logs"
// shardingKey: 分表键值（uint 类型），如 user_id
// numberOfShards: 分表数量，建议为 2 的幂次（如 4, 8, 16, 32, 64 等）
// 返回: 分表名称，如 "user_balance_logs_0", "user_balance_logs_1" 等
// 分表逻辑：shardingKey % numberOfShards
//
// 使用示例：
//
//	// 为 user_balance_logs 表分表（4个分表）
//	tableName := GetHashShardingTableName("user_balance_logs", userID, 4)
//
//	// 为新的表 example_table 分表（8个分表）
//	tableName := GetHashShardingTableName("example_table", entityID, 8)
func GetHashShardingTableName(baseTableName string, shardingKey uint, numberOfShards int) string {
	if numberOfShards <= 0 {
		// 如果分表数量无效，返回基础表名（不分表）
		return baseTableName
	}
	shardIndex := int(shardingKey) % numberOfShards
	return fmt.Sprintf("%s_%d", baseTableName, shardIndex)
}

// GetHashShardingTableNameByConfig 通过配置获取哈希分表名称
func GetHashShardingTableNameByConfig(config HashShardingConfig, shardingKey uint) string {
	return GetHashShardingTableName(config.BaseTableName, shardingKey, config.NumberOfShards)
}

// GetAllHashShardingTableNames 获取所有哈希分表名称列表
// 用于批量操作所有分表（如迁移、统计等）
func GetAllHashShardingTableNames(config HashShardingConfig) []string {
	tableNames := make([]string, config.NumberOfShards)
	for i := 0; i < config.NumberOfShards; i++ {
		tableNames[i] = fmt.Sprintf("%s_%d", config.BaseTableName, i)
	}
	return tableNames
}

// GetHashShardIndex 获取分表索引
func GetHashShardIndex(shardingKey uint, numberOfShards int) int {
	if numberOfShards <= 0 {
		return 0
	}
	return int(shardingKey) % numberOfShards
}

// GetShardingTableNames 获取时间范围内的所有分表名称
// baseTableName: 基础表名
// startTime: 开始时间
// endTime: 结束时间
// 返回: 分表名称列表
func GetShardingTableNames(baseTableName string, startTime, endTime time.Time) []string {
	var tableNames []string

	// 如果结束时间是零值，使用当前时间
	if endTime.IsZero() {
		endTime = time.Now().UTC()
	}

	// 确保开始时间不晚于结束时间
	if startTime.After(endTime) {
		return tableNames
	}

	// 从开始时间到结束时间，按月遍历
	current := time.Date(startTime.Year(), startTime.Month(), 1, 0, 0, 0, 0, startTime.Location())
	end := time.Date(endTime.Year(), endTime.Month(), 1, 0, 0, 0, 0, endTime.Location())

	for !current.After(end) {
		tableNames = append(tableNames, GetShardingTableName(baseTableName, current))
		current = current.AddDate(0, 1, 0) // 加一个月
	}

	return tableNames
}

// 默认最大时间范围（月），运行时应优先使用 GetMaxTimeRangeMonths()。
const defaultMaxTimeRangeMonths = 3

// TimeRangeError 时间范围验证错误
type TimeRangeError struct {
	Key    string
	Params map[string]any
}

func (e *TimeRangeError) Error() string {
	// 返回翻译键，由调用方进行翻译
	return e.Key
}

// ValidateTimeRange 验证时间范围是否超过指定月数
// startTime: 开始时间
// endTime: 结束时间
// maxMonths: 最大允许的月数，如果为0则使用默认值 DefaultMaxTimeRangeMonths
// 返回: 是否有效，错误信息（错误信息包含翻译键和参数）
func ValidateTimeRange(startTime, endTime time.Time, maxMonths ...int) (bool, error) {
	// 检查开始时间是否晚于结束时间（忽略零值时间）
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		return false, &TimeRangeError{
			Key:    "start_time_after_end_time",
			Params: nil,
		}
	}

	// 确定最大月数
	maxMonthsValue := GetMaxTimeRangeMonths()
	if len(maxMonths) > 0 && maxMonths[0] > 0 {
		maxMonthsValue = maxMonths[0]
	}

	// 检查时间范围是否超过指定月数（忽略零值结束时间）
	if !endTime.IsZero() {
		maxTimeLater := startTime.AddDate(0, maxMonthsValue, 0)
		if endTime.After(maxTimeLater) {
			return false, &TimeRangeError{
				Key: "time_range_exceeded",
				Params: map[string]any{
					"months": maxMonthsValue,
				},
			}
		}
	}

	return true, nil
}

// GetAllExistingShardingTables 获取数据库中所有已存在的分表名称
// baseTableName: 基础表名，如 "orders" 或 "order_details"
// 返回: 已存在的分表名称列表
func GetAllExistingShardingTables(ctx context.Context, baseTableName string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var tableNames []string

	// 构建表名匹配模式：orders_YYYYMM 或 order_details_YYYYMM
	pattern := fmt.Sprintf("%s_%%", baseTableName)

	query, args := buildShardingTableQuery(pattern)

	// 执行查询，使用 Scan 获取结果
	var rows []map[string]any
	if err := appfacades.OrmQuery(ctx).Raw(query, args...).Scan(&rows); err != nil {
		errorlog.Record(ctx, "sharding", "查询分表失败", map[string]any{
			"pattern": pattern,
			"error":   err.Error(),
		}, "查询分表失败: %v", err)
		return nil, fmt.Errorf("查询分表失败: %v", err)
	}

	// 验证表名格式（确保是有效的分表名称，格式为 baseTableName_YYYYMM）
	// 从 pattern 中提取基础表名（移除末尾的 %）
	baseTableNameFromPattern := strings.TrimSuffix(pattern, "_%")
	patternRegex := regexp.MustCompile(fmt.Sprintf("^%s_\\d{6}$", regexp.QuoteMeta(baseTableNameFromPattern)))

	for _, row := range rows {
		// 尝试不同的字段名格式（MySQL 可能返回不同的大小写）
		var tableName string
		var ok bool

		// 尝试 table_name (小写)
		if tableName, ok = row["table_name"].(string); !ok {
			// 尝试 TABLE_NAME (大写)
			if tableName, ok = row["TABLE_NAME"].(string); !ok {
				// 尝试 name / NAME（DM 查询中统一别名为 name）
				if tableName, ok = row["name"].(string); !ok {
					if tableName, ok = row["NAME"].(string); !ok {
						// 尝试遍历所有键
						for key, value := range row {
							if (key == "table_name" || key == "TABLE_NAME" || key == "Table_Name" || key == "name" || key == "NAME") && value != nil {
								if str, ok := value.(string); ok {
									tableName = str
									ok = true
									break
								}
							}
						}
					}
				}
			}
		}

		if ok && tableName != "" {
			// 验证表名格式
			if patternRegex.MatchString(tableName) {
				tableNames = append(tableNames, tableName)
			}
		}
	}

	return tableNames, nil
}

// GetAllExistingShardingTablesByPattern 通过表名模式获取所有已存在的分表
// 这是一个更通用的方法，可以通过自定义模式匹配
// pattern: 表名匹配模式，如 "orders_%" 或 "order_details_%"
func GetAllExistingShardingTablesByPattern(ctx context.Context, pattern string) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	var tableNames []string
	query, args := buildShardingTableQuery(pattern)

	// 执行查询，使用 Scan 获取结果
	var rows []map[string]any
	if err := appfacades.OrmQuery(ctx).Raw(query, args...).Scan(&rows); err != nil {
		errorlog.Record(ctx, "sharding", "查询分表失败", map[string]any{
			"pattern": pattern,
			"error":   err.Error(),
		}, "查询分表失败: %v", err)
		return nil, fmt.Errorf("查询分表失败: %v", err)
	}

	for _, row := range rows {
		if tableName, ok := row["table_name"].(string); ok {
			tableNames = append(tableNames, tableName)
			continue
		}
		if tableName, ok := row["TABLE_NAME"].(string); ok {
			tableNames = append(tableNames, tableName)
			continue
		}
		if tableName, ok := row["name"].(string); ok {
			tableNames = append(tableNames, tableName)
			continue
		}
		if tableName, ok := row["NAME"].(string); ok {
			tableNames = append(tableNames, tableName)
		}
	}

	return tableNames, nil
}

func buildShardingTableQuery(pattern string) (string, []any) {
	if facades.Config().GetString("database.default") == "dm" {
		return `
		SELECT TABLE_NAME AS name
		FROM ALL_TABLES
		WHERE OWNER = SYS_CONTEXT('USERENV', 'CURRENT_SCHEMA')
		AND TABLE_NAME LIKE UPPER(?)
		ORDER BY TABLE_NAME
	`, []any{pattern}
	}

	// Prefer the live Schema/Orm database (tenant bind), not the platform mysql config.
	dbName := ""
	if s := facades.Schema(); s != nil && s.Orm() != nil {
		dbName = s.Orm().DatabaseName()
	}
	if dbName == "" {
		dbName = facades.Config().GetString("database.connections.mysql.database")
	}
	if dbName == "" {
		dbName = facades.Config().GetString("database.connections.postgresql.database")
	}

	return `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = ?
		AND table_name LIKE ?
		ORDER BY table_name
	`, []any{dbName, pattern}
}
