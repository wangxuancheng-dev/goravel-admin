package helpers

import (
	"regexp"
	"strings"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/support/str"
)

// sortFieldPattern allows only simple SQL identifiers (blocks ORDER BY injection).
var sortFieldPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// ApplySort applies ORDER BY from "field:direction" or "field1:dir1,field2:dir2".
// Field names must match [a-zA-Z_][a-zA-Z0-9_]*. When allowedFields is non-empty,
// only those columns are accepted; unknown or unsafe fields are skipped.
// If nothing valid remains, defaultSort is applied (same rules).
func ApplySort(query orm.Query, orderBy string, defaultSort string, allowedFields ...string) orm.Query {
	sortStr := orderBy
	if sortStr == "" {
		sortStr = defaultSort
	}
	if sortStr == "" {
		return query
	}

	allowed := make(map[string]struct{}, len(allowedFields))
	for _, f := range allowedFields {
		f = strings.TrimSpace(f)
		if f != "" {
			allowed[f] = struct{}{}
		}
	}

	orderClauses := parseSortClauses(sortStr, allowed)
	if len(orderClauses) == 0 && orderBy != "" && defaultSort != "" && orderBy != defaultSort {
		orderClauses = parseSortClauses(defaultSort, allowed)
	}
	if len(orderClauses) == 0 {
		return query
	}

	orderStr := orderClauses[0]
	for i := 1; i < len(orderClauses); i++ {
		orderStr = str.Of(orderStr).Append(", ").Append(orderClauses[i]).String()
	}
	return query.Order(orderStr)
}

func parseSortClauses(sortStr string, allowed map[string]struct{}) []string {
	sortFields := str.Of(sortStr).Split(",")
	var orderClauses []string

	for _, field := range sortFields {
		field = str.Of(field).Trim().String()
		if str.Of(field).IsEmpty() {
			continue
		}

		parts := str.Of(field).Split(":")
		fieldName := str.Of(parts[0]).Trim().String()
		direction := "asc"
		if len(parts) > 1 {
			direction = str.Of(parts[1]).Trim().Lower().String()
		}
		if direction != "asc" && direction != "desc" {
			direction = "asc"
		}

		if !sortFieldPattern.MatchString(fieldName) {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[fieldName]; !ok {
				continue
			}
		}

		orderClauses = append(orderClauses, fieldName+" "+direction)
	}
	return orderClauses
}

// ParseSort parses sort params into field -> direction (asc/desc).
// Unsafe field names are omitted.
func ParseSort(orderBy string) map[string]string {
	result := make(map[string]string)
	if orderBy == "" {
		return result
	}

	for _, field := range str.Of(orderBy).Split(",") {
		field = str.Of(field).Trim().String()
		if str.Of(field).IsEmpty() {
			continue
		}
		parts := str.Of(field).Split(":")
		fieldName := str.Of(parts[0]).Trim().String()
		if !sortFieldPattern.MatchString(fieldName) {
			continue
		}
		direction := "asc"
		if len(parts) > 1 {
			direction = str.Of(parts[1]).Trim().Lower().String()
		}
		if direction != "asc" && direction != "desc" {
			direction = "asc"
		}
		result[fieldName] = direction
	}
	return result
}
