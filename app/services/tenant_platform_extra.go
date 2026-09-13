package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/clients"
	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/tenancy"
)

// TenantOverview is a quick health snapshot of a tenant database.
type TenantOverview struct {
	Database       string `json:"database"`
	Driver         string `json:"driver"`
	TableCount     int64  `json:"table_count"`
	DatabaseBytes  int64  `json:"database_bytes"`
	AdminsCount    int64  `json:"admins_count"`
	MigrationsCount int64 `json:"migrations_count"`
	ProvisionStatus string `json:"provision_status"`
	PingOK         bool   `json:"ping_ok"`
	PingMs         int64  `json:"ping_ms"`
	Error          string `json:"error,omitempty"`
}

// TenantPingDetail is an extended connectivity check.
type TenantPingDetail struct {
	OK               bool   `json:"ok"`
	LatencyMs        int64  `json:"latency_ms"`
	Driver           string `json:"driver"`
	Database         string `json:"database"`
	Host             string `json:"host"`
	Port             int    `json:"port"`
	ProvisionStatus  string `json:"provision_status"`
	LastMigrateError string `json:"last_migrate_error"`
	LastOp           string `json:"last_op"`
	LastOpStatus     string `json:"last_op_status"`
	LastOpMessage    string `json:"last_op_message"`
	Error            string `json:"error,omitempty"`
}

// TenantLoginLinks helps open the tenant admin UI.
type TenantLoginLinks struct {
	Resolver   string `json:"resolver"`
	Header     string `json:"header"`
	QueryURL   string `json:"query_url"`
	Hint       string `json:"hint"`
	TenantCode string `json:"tenant_code"`
}

// PlatformQueueStatus reports long-running queue visibility for tenant ops.
type PlatformQueueStatus struct {
	Connection string `json:"connection"`
	Queue      string `json:"queue"`
	Pending    int64  `json:"pending"`
	Available  bool   `json:"available"`
	Message    string `json:"message"`
}

// TenantDeleteOptions controls destructive side effects when removing a tenant.
type TenantDeleteOptions struct {
	DropDatabase bool
	PurgeObjects bool
	PurgeBackups bool
}

// TenantDeleteResult is returned after soft-delete (and optional DROP DATABASE).
type TenantDeleteResult struct {
	ID           uint   `json:"id"`
	Code         string `json:"code"`
	DropDatabase bool   `json:"drop_database"`
	PurgeObjects bool   `json:"purge_objects"`
	PurgeBackups bool   `json:"purge_backups"`
}

// DeleteTenant soft-deletes platform metadata and optionally drops the tenant DB/schema.
// Object storage / local backup cleanup is NOT done here — enqueue TenantOpPurge asynchronously.
func (s *TenantAdminService) DeleteTenant(id uint, confirmCode string, opts TenantDeleteOptions) (*TenantDeleteResult, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	tenant, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(confirmCode) != tenant.Code {
		return nil, apperrors.ErrInvalidArgument.WithMessage("confirm_code must equal tenant code")
	}
	if tenantOpBusy(tenant) {
		return nil, apperrors.ErrTenantOpInProgress
	}
	s.conn.Forget(tenant.ConnectionName)
	if opts.DropDatabase {
		if err := s.conn.DropStorage(tenant); err != nil {
			return nil, apperrors.ErrTenantConnectionFailed.WithError(err)
		}
	}
	if _, err := appfacades.PlatformOrmQuery(nil).Where("id", tenant.ID).Delete(&models.Tenant{}); err != nil {
		return nil, err
	}
	return &TenantDeleteResult{
		ID:           tenant.ID,
		Code:         tenant.Code,
		DropDatabase: opts.DropDatabase,
		PurgeObjects: opts.PurgeObjects,
		PurgeBackups: opts.PurgeBackups,
	}, nil
}

// ListForBatchOp returns tenants for batch seed/backup/migrate.
func (s *TenantAdminService) ListForBatchOp(ids []uint, provisionStatus string, status *uint8, limit int) ([]models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	query := appfacades.PlatformOrmQuery(nil).Model(&models.Tenant{})
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	if ps := strings.TrimSpace(provisionStatus); ps != "" {
		query = query.Where("provision_status", ps)
	}
	if status != nil {
		query = query.Where("status", *status)
	}
	var list []models.Tenant
	if err := query.Order("id asc").Limit(limit).Find(&list); err != nil {
		return nil, err
	}
	return list, nil
}

// BuildTenantOverview inspects the tenant DB for size / table / admin counts.
func (s *TenantConnectionService) BuildTenantOverview(tenant *models.Tenant) *TenantOverview {
	out := &TenantOverview{
		Database:        tenant.Database,
		Driver:          NormalizeTenantDriver(tenant.Driver),
		ProvisionStatus: tenant.ProvisionStatus,
	}
	start := time.Now()
	if err := s.Ping(tenant, 5*time.Second); err != nil {
		out.PingOK = false
		out.Error = err.Error()
		out.PingMs = time.Since(start).Milliseconds()
		return out
	}
	out.PingOK = true
	out.PingMs = time.Since(start).Milliseconds()

	err := s.WithTenantConnection(tenant, func() error {
		dbName := facades.Schema().Orm().DatabaseName()
		if dbName == "" {
			dbName = tenant.Database
		}
		driver := NormalizeTenantDriver(tenant.Driver)
		switch driver {
		case models.TenantDriverPostgres:
			var n int64
			_ = facades.Orm().Query().Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'`).Scan(&n)
			out.TableCount = n
			var bytes int64
			_ = facades.Orm().Query().Raw(`SELECT pg_database_size(current_database())`).Scan(&bytes)
			out.DatabaseBytes = bytes
		default:
			var n int64
			_ = facades.Orm().Query().Raw(
				"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_type IN ('BASE TABLE','SYSTEM VERSIONED')",
				dbName,
			).Scan(&n)
			out.TableCount = n
			var bytes int64
			_ = facades.Orm().Query().Raw(
				"SELECT COALESCE(SUM(data_length+index_length),0) FROM information_schema.tables WHERE table_schema = ?",
				dbName,
			).Scan(&bytes)
			out.DatabaseBytes = bytes
		}
		if facades.Schema().HasTable("admins") {
			n, _ := facades.Orm().Query().Table("admins").Count()
			out.AdminsCount = n
		}
		if facades.Schema().HasTable("migrations") {
			n, _ := facades.Orm().Query().Table("migrations").Count()
			out.MigrationsCount = n
		}
		return nil
	})
	if err != nil && out.Error == "" {
		out.Error = err.Error()
	}
	return out
}

// BuildPingDetail pings and returns latency plus tenant meta.
func (s *TenantConnectionService) BuildPingDetail(tenant *models.Tenant) *TenantPingDetail {
	host, port, _, _, database, _ := ResolveTenantDSN(tenant)
	detail := &TenantPingDetail{
		Driver:           NormalizeTenantDriver(tenant.Driver),
		Database:         database,
		Host:             host,
		Port:             port,
		ProvisionStatus:  tenant.ProvisionStatus,
		LastMigrateError: tenant.LastMigrateError,
		LastOp:           tenant.LastOp,
		LastOpStatus:     tenant.LastOpStatus,
		LastOpMessage:    tenant.LastOpMessage,
	}
	start := time.Now()
	if err := s.Ping(tenant, 5*time.Second); err != nil {
		detail.OK = false
		detail.Error = err.Error()
		detail.LatencyMs = time.Since(start).Milliseconds()
		return detail
	}
	detail.OK = true
	detail.LatencyMs = time.Since(start).Milliseconds()
	return detail
}

// BuildTenantLoginLinks returns how to open the tenant admin UI.
func BuildTenantLoginLinks(tenant *models.Tenant) TenantLoginLinks {
	code := ""
	if tenant != nil {
		code = tenant.Code
	}
	resolver := strings.TrimSpace(facades.Config().GetString("tenancy.resolver", "header"))
	header := strings.TrimSpace(facades.Config().GetString("tenancy.header", "X-Tenant-ID"))
	appURL := strings.TrimRight(strings.TrimSpace(facades.Config().GetString("app.url", "http://localhost")), "/")
	// Frontend is often on another port; query hint works for local SPA.
	queryURL := fmt.Sprintf("/login?tenant_code=%s", code)
	hint := "Open tenant admin with query tenant_code or header " + header
	if resolver == "subdomain" && code != "" {
		hint = "Prefer subdomain: " + code + ".<your-domain>; header fallback may be disabled"
	}
	_ = appURL
	return TenantLoginLinks{
		Resolver:   resolver,
		Header:     header,
		QueryURL:   queryURL,
		Hint:       hint,
		TenantCode: code,
	}
}

// BuildPlatformQueueStatus reports long-running queue depth when Redis is used.
func BuildPlatformQueueStatus() PlatformQueueStatus {
	conn := facades.Config().GetString("queue.default", "sync")
	out := PlatformQueueStatus{
		Connection: conn,
		Queue:      "long-running",
	}
	if conn == "sync" {
		out.Available = true
		out.Message = "QUEUE_CONNECTION=sync: tenant ops run inline in the API process"
		return out
	}
	if conn != "redis" {
		out.Message = "Queue connection is " + conn + "; pending depth not inspected"
		out.Available = true
		return out
	}
	rdb, err := clients.GetRedisClient("default")
	if err != nil {
		out.Message = "redis unavailable: " + err.Error()
		return out
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	// Goravel redis queue often uses stream or list; try common keys.
	keys := []string{
		"goravel_queues:long-running",
		"queues:long-running",
		facades.Config().GetString("queue.connections.redis.queue", "default") + ":long-running",
	}
	var pending int64
	found := false
	for _, key := range keys {
		n, err := rdb.LLen(ctx, key).Result()
		if err == nil {
			pending = n
			found = true
			break
		}
		n, err = rdb.XLen(ctx, key).Result()
		if err == nil {
			pending = n
			found = true
			break
		}
	}
	out.Available = true
	out.Pending = pending
	if !found {
		out.Message = "redis ok; queue depth key not found (worker may still consume long-running)"
	} else {
		out.Message = "long-running pending jobs (approx)"
	}
	_ = tenancy.Enabled()
	return out
}
