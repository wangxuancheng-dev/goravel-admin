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
	"goravel/app/search"
	"goravel/app/tenancy"
	"goravel/app/tenancyctx"
)

// TenantOverview is a quick health snapshot of a tenant database.
type TenantOverview struct {
	Database        string `json:"database"`
	Driver          string `json:"driver"`
	TableCount      int64  `json:"table_count"`
	DatabaseBytes   int64  `json:"database_bytes"`
	AdminsCount     int64  `json:"admins_count"`
	MigrationsCount int64  `json:"migrations_count"`
	ProvisionStatus string `json:"provision_status"`
	PingOK          bool   `json:"ping_ok"`
	PingMs          int64  `json:"ping_ms"`
	Error           string `json:"error,omitempty"`
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
	Resolver          string   `json:"resolver"`
	Header            string   `json:"header"`
	QueryURL          string   `json:"query_url"`
	Hint              string   `json:"hint"`
	TenantCode        string   `json:"tenant_code"`
	SubdomainURL      string   `json:"subdomain_url,omitempty"`
	PrimaryCustomURL  string   `json:"primary_custom_url,omitempty"`
	ActiveCustomHosts []string `json:"active_custom_hosts,omitempty"`
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
	_ = NewTenantDomainService().DisableAllForTenant(tenant.ID)
	if _, err := appfacades.PlatformQuery().Where("id", tenant.ID).Delete(&models.Tenant{}); err != nil {
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

// UndeleteTenant restores soft-deleted platform metadata (does not recreate a dropped DB).
func (s *TenantAdminService) UndeleteTenant(id uint) (*models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	tenant, err := s.GetByIDIncludingTrashed(id)
	if err != nil {
		return nil, err
	}
	if !TenantIsTrashed(tenant) {
		return nil, apperrors.ErrTenantNotTrashed
	}
	if tenantOpBusy(tenant) {
		return nil, apperrors.ErrTenantOpInProgress
	}
	if _, err = appfacades.PlatformQuery().WithTrashed().Where("id", tenant.ID).Restore(&models.Tenant{}); err != nil {
		return nil, err
	}
	restored, err := s.GetByID(id)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	msg := "undeleted"
	if err := s.conn.Ping(restored, 5*time.Second); err != nil {
		msg = "undeleted; database unreachable (re-migrate or recreate storage if drop_database was used): " + err.Error()
		if len(msg) > 2000 {
			msg = msg[:2000]
		}
		_, _ = appfacades.PlatformQuery().Model(restored).Update(map[string]any{
			"provision_status":   models.TenantProvisionFailed,
			"last_migrate_error": msg,
			"last_op":            "undelete",
			"last_op_status":     models.TenantOpStatusFailed,
			"last_op_message":    msg,
			"last_op_at":         now,
		})
		restored.ProvisionStatus = models.TenantProvisionFailed
		restored.LastMigrateError = msg
		restored.LastOp = "undelete"
		restored.LastOpStatus = models.TenantOpStatusFailed
		restored.LastOpMessage = msg
		restored.LastOpAt = &now
		return restored, nil
	}
	_, _ = appfacades.PlatformQuery().Model(restored).Update(map[string]any{
		"last_op":         "undelete",
		"last_op_status":  models.TenantOpStatusSuccess,
		"last_op_message": msg,
		"last_op_at":      now,
	})
	restored.LastOp = "undelete"
	restored.LastOpStatus = models.TenantOpStatusSuccess
	restored.LastOpMessage = msg
	restored.LastOpAt = &now
	return restored, nil
}

// ForceDeleteTenant permanently removes a soft-deleted tenant: its data first (see
// eraseTenantData), then its row, which frees the code. When any part of the data
// cannot be removed the tenant stays in the recycle bin and the error is returned.
func (s *TenantAdminService) ForceDeleteTenant(id uint, confirmCode string) error {
	if err := s.requireEnabled(); err != nil {
		return err
	}
	tenant, err := s.GetByIDIncludingTrashed(id)
	if err != nil {
		return err
	}
	if !TenantIsTrashed(tenant) {
		return apperrors.ErrTenantNotTrashed
	}
	if strings.TrimSpace(confirmCode) != tenant.Code {
		return apperrors.ErrInvalidArgument.WithMessage("confirm_code must equal tenant code")
	}
	if tenantOpBusy(tenant) {
		return apperrors.ErrTenantOpInProgress
	}
	// The code becomes reusable below: nothing of this tenant may be left under it.
	if err := s.eraseTenantData(tenant); err != nil {
		return err
	}
	_ = NewTenantDomainService().PurgeAllForTenant(tenant.ID)
	if _, err := appfacades.PlatformQuery().WithTrashed().Where("id", tenant.ID).ForceDelete(&models.Tenant{}); err != nil {
		return err
	}
	return nil
}

// eraseTenantData removes everything kept under a tenant's code before the code can be
// reused: a new tenant with the same code gets the same database name, storage prefix,
// backup directory and index names, so anything left over would be handed to it (a
// leftover database is even reused as is). Order: search indexes, object storage and
// backups (their lookup needs the tenant row), then the database / schema and its
// dedicated account. Each step is idempotent, so a failed run can simply be retried.
func (s *TenantAdminService) eraseTenantData(t *models.Tenant) error {
	if err := dropTenantSearchIndexes(t); err != nil {
		return fmt.Errorf("drop search indexes: %w", err)
	}
	if err := PurgeTenantObjectStorage(t.Code); err != nil {
		return fmt.Errorf("purge object storage: %w", err)
	}
	if err := PurgeTenantBackups(t.Code); err != nil {
		return fmt.Errorf("purge backups: %w", err)
	}
	if err := s.conn.DropStorage(t); err != nil {
		return apperrors.ErrTenantConnectionFailed.WithError(fmt.Errorf("drop database: %w", err))
	}
	return nil
}

// dropTenantSearchIndexes deletes the tenant's search indexes before its row (and with
// it the code) is gone for good. Indexes are named after the code, so a later tenant
// reusing the code would otherwise search the old tenant's documents. On failure the
// caller keeps the tenant row, so the code stays taken until a retry succeeds.
func dropTenantSearchIndexes(t *models.Tenant) error {
	ctx := tenancyctx.WithTenant(context.Background(), t.ID, t.ConnectionName, t.Code)
	if err := search.DropTenantIndexes(ctx); err != nil {
		facades.Log().Errorf("drop search indexes of tenant %s: %v", t.Code, err)
		return err
	}
	return nil
}

// CleanupExpiredDeletedTenants permanently deletes tenants soft-deleted more than days
// ago, like ForceDeleteTenant: data first, and a tenant whose data cannot be removed
// is kept (and retried on the next run).
func (s *TenantAdminService) CleanupExpiredDeletedTenants(days int, limit int) (int, error) {
	if err := s.requireEnabled(); err != nil {
		return 0, err
	}
	if days <= 0 {
		return 0, nil
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	var list []models.Tenant
	if err := appfacades.PlatformQuery().Model(&models.Tenant{}).WithTrashed().
		Where("deleted_at IS NOT NULL").
		Where("deleted_at < ?", cutoff).
		Order("deleted_at asc").
		Limit(limit).
		Find(&list); err != nil {
		return 0, err
	}
	removed := 0
	for i := range list {
		t := &list[i]
		if tenantOpBusy(t) {
			continue
		}
		if err := s.eraseTenantData(t); err != nil {
			facades.Log().Warningf("tenant cleanup: keep %s (code not freed): %v", t.Code, err)
			continue // keep the row (and code) until everything is gone
		}
		_ = NewTenantDomainService().PurgeAllForTenant(t.ID)
		if _, err := appfacades.PlatformQuery().WithTrashed().Where("id", t.ID).ForceDelete(&models.Tenant{}); err != nil {
			continue
		}
		removed++
	}
	return removed, nil
}

// ListForBatchOpFiltered returns tenants for batch seed/backup/migrate,
// with optional last_op / last_op_status filters (e.g. last_op=seed, last_op_status=failed).
func (s *TenantAdminService) ListForBatchOpFiltered(ids []uint, provisionStatus string, status *uint8, lastOp, lastOpStatus string, limit int) ([]models.Tenant, error) {
	if err := s.requireEnabled(); err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	query := appfacades.PlatformQuery().Model(&models.Tenant{})
	if len(ids) > 0 {
		query = query.Where("id IN ?", ids)
	}
	if ps := strings.TrimSpace(provisionStatus); ps != "" {
		query = query.Where("provision_status", ps)
	}
	if status != nil {
		query = query.Where("status", *status)
	}
	if op := NormalizeTenantLastOpFilter(lastOp); op != "" {
		query = query.Where("last_op", op)
	}
	if st := NormalizeTenantLastOpStatusFilter(lastOpStatus); st != "" {
		query = query.Where("last_op_status", st)
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

	// Queries go to the tenant's connection explicitly: no goroutine binding, which is
	// reserved for migrate / seed (they take no ctx).
	conn, err := s.ensureConn(tenant)
	if err != nil {
		out.Error = err.Error()
		return out
	}
	q := conn.Orm().Query
	dbName := conn.DatabaseName()
	if dbName == "" {
		dbName = tenant.Database
	}
	switch NormalizeTenantDriver(tenant.Driver) {
	case models.TenantDriverPostgres:
		var n int64
		_ = q().Raw(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = current_schema() AND table_type = 'BASE TABLE'`).Scan(&n)
		out.TableCount = n
		var bytes int64
		_ = q().Raw(`SELECT pg_database_size(current_database())`).Scan(&bytes)
		out.DatabaseBytes = bytes
	default:
		var n int64
		_ = q().Raw(
			"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_type IN ('BASE TABLE','SYSTEM VERSIONED')",
			dbName,
		).Scan(&n)
		out.TableCount = n
		var bytes int64
		_ = q().Raw(
			"SELECT COALESCE(SUM(data_length+index_length),0) FROM information_schema.tables WHERE table_schema = ?",
			dbName,
		).Scan(&bytes)
		out.DatabaseBytes = bytes
	}
	sch, err := conn.NewSchema()
	if err != nil {
		out.Error = err.Error()
		return out
	}
	if sch.HasTable("admins") {
		n, _ := q().Table("admins").Count()
		out.AdminsCount = n
	}
	if sch.HasTable("migrations") {
		n, _ := q().Table("migrations").Count()
		out.MigrationsCount = n
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
	var tenantID uint
	if tenant != nil {
		code = tenant.Code
		tenantID = tenant.ID
	}
	resolver := strings.TrimSpace(facades.Config().GetString("tenancy.resolver", "header"))
	header := strings.TrimSpace(facades.Config().GetString("tenancy.header", "X-Tenant-ID"))
	// Frontend is often on another port; query hint works for local SPA.
	queryURL := fmt.Sprintf("/login?tenant_code=%s", code)
	hint := "Open tenant admin with query tenant_code or header " + header
	base := tenancy.BaseDomain()
	subURL := ""
	if base != "" && code != "" {
		subURL = "https://" + code + "." + base + "/login"
		if resolver == "subdomain" {
			hint = "Prefer subdomain: " + code + "." + base + "; custom domains if bound"
		}
	} else if resolver == "subdomain" && code != "" {
		hint = "Prefer subdomain: " + code + ".<your-domain>; header fallback may be disabled"
	}

	primaryCustom := ""
	var activeHosts []string
	if tenantID > 0 {
		list, _ := NewTenantDomainService().ListByTenant(tenantID)
		for i := range list {
			if list[i].Status != models.TenantDomainStatusActive {
				continue
			}
			activeHosts = append(activeHosts, list[i].Host)
			if list[i].IsPrimary && primaryCustom == "" {
				primaryCustom = "https://" + list[i].Host + "/login"
			}
		}
		if primaryCustom == "" && len(activeHosts) > 0 {
			primaryCustom = "https://" + activeHosts[0] + "/login"
		}
		if primaryCustom != "" {
			hint = "Prefer custom domain login; subdomain still available when TENANCY_BASE_DOMAIN is set"
		}
	}

	return TenantLoginLinks{
		Resolver:          resolver,
		Header:            header,
		QueryURL:          queryURL,
		Hint:              hint,
		TenantCode:        code,
		SubdomainURL:      subURL,
		PrimaryCustomURL:  primaryCustom,
		ActiveCustomHosts: activeHosts,
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
	reader := NewQueueStatsReader(ctx)
	key := reader.RedisQueueKey("redis", "long-running")
	pending, err := rdb.LLen(ctx, key).Result()
	out.Available = true
	if err != nil {
		out.Pending = 0
		out.Message = "redis ok; queue depth key not found (worker may still consume long-running)"
	} else {
		out.Pending = pending
		out.Message = "long-running pending jobs (approx)"
	}
	_ = tenancy.Enabled()
	return out
}
