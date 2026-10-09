package utils

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	appfacades "goravel/app/facades"
)

// CountOptimizer 分页统计优化器
// 当数据量超过阈值时，使用执行计划估算的行数，否则使用实际的 count(*)
type CountOptimizer struct {
	ctx context.Context
	// Threshold 阈值，超过此值使用估算值（默认 10000）
	Threshold int64
	// ModuleName 模块名称，用于日志记录
	ModuleName string
}

// NewCountOptimizer 创建新的 CountOptimizer
func NewCountOptimizer(ctx context.Context, threshold int64, moduleName string) *CountOptimizer {
	if threshold <= 0 {
		threshold = 10000
	}
	return &CountOptimizer{
		ctx:        ctx,
		Threshold:  threshold,
		ModuleName: moduleName,
	}
}

// extractRowsFromPostgreSQLExplain 从 PostgreSQL EXPLAIN JSON 结果中提取行数
func (co *CountOptimizer) extractRowsFromPostgreSQLExplain(jsonStr string) int64 {
	// 简化实现：使用正则表达式提取 "Plan" -> "Plan Rows" 的值
	// 实际应该使用 JSON 解析，但为了简化，使用正则
	re := regexp.MustCompile(`"Plan Rows":\s*(\d+)`)
	matches := re.FindStringSubmatch(jsonStr)
	if len(matches) > 1 {
		if rows, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			return rows
		}
	}
	return 0
}

// extractRowsFromPostgreSQLExplainText 从 PostgreSQL EXPLAIN 文本结果中提取行数
func (co *CountOptimizer) extractRowsFromPostgreSQLExplainText(result []map[string]any) int64 {
	// PostgreSQL 文本格式的 EXPLAIN 结果中，行数通常在 "rows=" 后面
	for _, row := range result {
		if queryPlan, ok := row["QUERY PLAN"]; ok {
			if planStr, ok := queryPlan.(string); ok {
				// 使用正则表达式提取 rows= 后面的数字
				re := regexp.MustCompile(`rows=(\d+)`)
				matches := re.FindStringSubmatch(planStr)
				if len(matches) > 1 {
					if rows, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
						return rows
					}
				}
			}
		}
	}
	return 0
}

// OptimizedCountWithTable 使用表名和 WHERE 条件进行优化的 count 查询
// tableName: 表名
// whereClause: WHERE 子句（不包含 WHERE 关键字），例如："status = ? AND user_id = ?"
// args: WHERE 条件的参数
// 返回：总数、是否使用估算值、错误
func (co *CountOptimizer) OptimizedCountWithTable(tableName, whereClause string, args ...any) (int64, bool, error) {
	driver := strings.ToLower(appfacades.OrmQuery(co.ctx).Driver())

	// COUNT SQL is the same on MySQL and PostgreSQL for plain table identifiers.
	var countSQL string
	if whereClause != "" {
		countSQL = fmt.Sprintf("SELECT COUNT(*) as cnt FROM %s WHERE %s", tableName, whereClause)
	} else {
		countSQL = fmt.Sprintf("SELECT COUNT(*) as cnt FROM %s", tableName)
	}

	exactCount := func() (int64, bool, error) {
		var result struct {
			Cnt int64
		}
		if err := appfacades.OrmQuery(co.ctx).Raw(countSQL, args...).Scan(&result); err != nil {
			return 0, false, err
		}
		return result.Cnt, false, nil
	}

	// EXPLAIN syntax and result shapes differ by driver; estimate only for mysql/postgresql.
	var explainSQL string
	switch driver {
	case "mysql":
		if whereClause != "" {
			explainSQL = fmt.Sprintf("EXPLAIN SELECT COUNT(*) FROM %s WHERE %s", tableName, whereClause)
		} else {
			explainSQL = fmt.Sprintf("EXPLAIN SELECT COUNT(*) FROM %s", tableName)
		}
	case "postgresql":
		if whereClause != "" {
			explainSQL = fmt.Sprintf("EXPLAIN (FORMAT JSON) SELECT COUNT(*) FROM %s WHERE %s", tableName, whereClause)
		} else {
			explainSQL = fmt.Sprintf("EXPLAIN (FORMAT JSON) SELECT COUNT(*) FROM %s", tableName)
		}
	default:
		return exactCount()
	}

	estimatedCount, err := co.executeExplain(explainSQL, args...)
	if err != nil {
		return exactCount()
	}
	if estimatedCount >= co.Threshold {
		return estimatedCount, true, nil
	}
	return exactCount()
}

// executeExplain 执行 EXPLAIN 查询并提取估算行数
func (co *CountOptimizer) executeExplain(explainSQL string, args ...any) (int64, error) {
	driver := strings.ToLower(appfacades.OrmQuery(co.ctx).Driver())
	switch driver {
	case "mysql":
		// MySQL EXPLAIN 返回表格格式
		var explainResult []map[string]any
		if err := appfacades.OrmQuery(co.ctx).Raw(explainSQL, args...).Get(&explainResult); err != nil {
			return 0, err
		}

		if len(explainResult) == 0 {
			return 0, fmt.Errorf("explain result is empty")
		}

		// MySQL EXPLAIN 结果中，rows 字段包含估算行数
		if rows, ok := explainResult[0]["rows"]; ok {
			switch v := rows.(type) {
			case int64:
				return v, nil
			case int:
				return int64(v), nil
			case float64:
				return int64(v), nil
			case string:
				if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
					return parsed, nil
				}
			}
		}
		return 0, fmt.Errorf("cannot extract rows from explain result")

	case "postgresql":
		// PostgreSQL EXPLAIN (FORMAT JSON) 返回 JSON 格式
		var explainResult []map[string]any
		if err := appfacades.OrmQuery(co.ctx).Raw(explainSQL, args...).Get(&explainResult); err != nil {
			return 0, err
		}

		if len(explainResult) == 0 {
			return 0, fmt.Errorf("explain result is empty")
		}

		// PostgreSQL EXPLAIN (FORMAT JSON) 结果是一个包含 JSON 字符串的数组
		if queryPlan, ok := explainResult[0]["QUERY PLAN"]; ok {
			if planStr, ok := queryPlan.(string); ok {
				// 从 JSON 字符串中提取 "Plan" -> "Plan Rows" 的值
				rows := co.extractRowsFromPostgreSQLExplain(planStr)
				if rows > 0 {
					return rows, nil
				}
			}
		}

		// 如果 JSON 格式解析失败，尝试文本格式
		explainTextSQL := strings.Replace(explainSQL, "EXPLAIN (FORMAT JSON)", "EXPLAIN", 1)
		var explainTextResult []map[string]any
		if err := appfacades.OrmQuery(co.ctx).Raw(explainTextSQL, args...).Get(&explainTextResult); err == nil {
			rows := co.extractRowsFromPostgreSQLExplainText(explainTextResult)
			if rows > 0 {
				return rows, nil
			}
		}

		return 0, fmt.Errorf("cannot extract rows from explain result")
	}

	return 0, fmt.Errorf("unsupported database driver: %v", driver)
}

func parseExplainRowsValue(v any) (int64, bool) {
	switch x := v.(type) {
	case int64:
		if x >= 0 {
			return x, true
		}
	case int:
		if x >= 0 {
			return int64(x), true
		}
	case float64:
		if x >= 0 {
			return int64(x), true
		}
	case string:
		trimmed := strings.TrimSpace(x)
		if trimmed == "" {
			return 0, false
		}
		if parsed, err := strconv.ParseInt(trimmed, 10, 64); err == nil && parsed >= 0 {
			return parsed, true
		}
	}
	return 0, false
}
