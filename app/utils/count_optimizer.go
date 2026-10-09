package utils

import (
	"context"
	"fmt"

	appfacades "goravel/app/facades"
)

// CountWithTable runs an exact COUNT(*) on tableName with an optional WHERE clause
// (without the WHERE keyword). Compatible with MySQL and PostgreSQL for plain identifiers.
func CountWithTable(ctx context.Context, tableName, whereClause string, args ...any) (int64, error) {
	var countSQL string
	if whereClause != "" {
		countSQL = fmt.Sprintf("SELECT COUNT(*) as cnt FROM %s WHERE %s", tableName, whereClause)
	} else {
		countSQL = fmt.Sprintf("SELECT COUNT(*) as cnt FROM %s", tableName)
	}

	var result struct {
		Cnt int64
	}
	if err := appfacades.OrmQuery(ctx).Raw(countSQL, args...).Scan(&result); err != nil {
		return 0, err
	}
	return result.Cnt, nil
}
