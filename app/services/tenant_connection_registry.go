package services

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/goravel/framework/facades"

	"goravel/app/tenantdb"
)

// Tenant connections live in tenantdb.Default (app-owned pools, lock-free lookups).
// This file wires it to the tenancy config and the platform tenants table.

var configureTenantDBOnce sync.Once

// configureTenantDB applies TENANCY_REGISTERED_* limits and installs the resolver
// that rebuilds an evicted connection from the tenants table on demand.
func configureTenantDB() {
	configureTenantDBOnce.Do(func() {
		tenantdb.Default.SetOptions(tenantdb.Options{
			MaxConns:    registeredMax,
			IdleTTL:     registeredIdleTTL,
			CloseGrace:  2 * time.Minute,
			PingTimeout: 5 * time.Second,
		})
		tenantdb.Default.SetResolver(resolveTenantConnection)
	})
}

// InstallTenantConnectionResolver configures the registry (called at boot).
func InstallTenantConnectionResolver() { configureTenantDB() }

func registeredMax() int {
	return facades.Config().GetInt("tenancy.registered_max", 0)
}

func registeredIdleTTL() time.Duration {
	sec := facades.Config().GetInt("tenancy.registered_idle_ttl", 900)
	if sec <= 0 {
		return 0
	}
	return time.Duration(sec) * time.Second
}

// tenantIDFromConnectionName parses tenant_<id> (TenantConnectionName).
func tenantIDFromConnectionName(name string) (uint, bool) {
	rest, ok := strings.CutPrefix(name, "tenant_")
	if !ok || rest == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(rest, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return uint(id), true
}

// resolveTenantConnection rebuilds the Spec of tenant_<id> from the tenants table.
// Status is not checked here: binding (BindHTTP / BindBackground) gates access; this
// only keeps an already-bound ctx working after its connection was evicted.
func resolveTenantConnection(name string) (tenantdb.Spec, error) {
	id, ok := tenantIDFromConnectionName(name)
	if !ok {
		return tenantdb.Spec{}, tenantdb.ErrUnknownConnection
	}
	s := NewTenantConnectionService()
	tenant, err := s.FindTenantByIDOrCode(strconv.FormatUint(uint64(id), 10))
	if err != nil {
		return tenantdb.Spec{}, err
	}
	if tenant.ConnectionName == "" {
		tenant.ConnectionName = TenantConnectionName(tenant.ID)
	}
	if tenant.ConnectionName != name {
		return tenantdb.Spec{}, tenantdb.ErrUnknownConnection
	}
	return s.buildSpec(tenant)
}
