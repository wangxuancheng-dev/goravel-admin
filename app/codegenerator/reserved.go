package codegenerator

import "strings"

// ReservedTables are built-in tables hidden from the code-generator table picker.
// Append new system / scaffold tables here when the project gains modules.
var ReservedTables = []string{
	// auth / RBAC
	"admins",
	"users",
	"roles",
	"permissions",
	"menus",
	"admin_role",
	"role_menu",
	"role_permission",
	"role_department",
	"role_users",
	"role_permissions",
	"personal_access_tokens",
	// org / dictionary / config
	"departments",
	"positions",
	"dictionaries",
	"dict_types",
	"dict_data",
	"configs",
	"settings",
	"currencies",
	// logs / observability
	"login_logs",
	"operation_logs",
	"system_logs",
	"slow_query_logs",
	"api_endpoint_metrics",
	"notifications",
	// files / tasks
	"attachments",
	"attachment_categories",
	"exports",
	"imports",
	"jobs",
	"failed_jobs",
	"migrations",
	// security / demo / schedule / search
	"blacklists",
	"allowlists",
	"demo_activities",
	"flexible_schedules",
	"search_sync_outbox",
	// built-in business modules (articles / quotes demo tables stay selectable for codegen)
	"orders",
	"order_details",
	"payments",
	"payment_methods",
	"user_balance_logs",
	// tenancy / platform (landlord + tenant meta)
	"tenants",
	"tenant_domains",
	"tenant_op_logs",
	"platform_admins",
	"platform_login_logs",
	"platform_operation_logs",
	"platform_alert_deliveries",
}

// ReservedTablePrefixes hide whole families of system tables (e.g. platform_*).
// Append new prefixes here when needed.
var ReservedTablePrefixes = []string{
	"platform_",
	"tenant_",
}

// IsReservedTable reports whether tableName is a built-in/system table
// (exact match, reserved prefix, or numeric sharding suffix).
func IsReservedTable(tableName string) bool {
	return IsReservedTableWith(tableName, ReservedTables, ReservedTablePrefixes)
}

// IsReservedTableWith is the pure checker (tables/prefixes injectable for tests / config merge).
func IsReservedTableWith(tableName string, tables, prefixes []string) bool {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return true
	}
	for _, t := range tables {
		if tableName == strings.TrimSpace(t) {
			return true
		}
	}
	for _, prefix := range prefixes {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" && strings.HasPrefix(tableName, prefix) {
			return true
		}
	}
	// Sharding suffixes: orders_202501, user_balance_logs_0, ...
	if last := strings.LastIndex(tableName, "_"); last != -1 && last < len(tableName)-1 {
		suffix := tableName[last+1:]
		if suffix != "" {
			numeric := true
			for _, ch := range suffix {
				if ch < '0' || ch > '9' {
					numeric = false
					break
				}
			}
			if numeric {
				return true
			}
		}
	}
	return false
}

// MergeExtras appends extra table names / prefixes (e.g. from config env) onto defaults.
func MergeExtras(base, extras []string) []string {
	if len(extras) == 0 {
		return base
	}
	out := make([]string, 0, len(base)+len(extras))
	out = append(out, base...)
	seen := map[string]bool{}
	for _, t := range base {
		seen[strings.TrimSpace(t)] = true
	}
	for _, t := range extras {
		t = strings.TrimSpace(t)
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// SplitCSV splits a comma-separated env value into trimmed non-empty parts.
func SplitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
