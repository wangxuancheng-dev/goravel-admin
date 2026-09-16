package services

import (
	"strings"

	"goravel/app/models"
)

// TenantOnboardInput drives the platform create→migrate→seed→optional domain wizard.
type TenantOnboardInput struct {
	Create      TenantCreateInput
	WithMigrate bool
	WithSeed    bool
	DomainHost  string
	DomainSSL   string
	Actor       TenantOpActor
}

// TenantOnboardResult is returned after create (+ optional queue begin / domain bind).
// Caller must Dispatch the queue job when QueuedArgs is set.
type TenantOnboardResult struct {
	Tenant     *models.Tenant
	Queued     bool
	Op         string
	OpLogID    uint
	QueuedArgs *TenantOpsArgs
	Domain     *models.TenantDomain
	LoginLinks TenantLoginLinks
	Steps      []string
}

// PrepareTenantOnboard creates a tenant, optionally begins migrate(+seed), optionally binds a vanity domain.
func PrepareTenantOnboard(in TenantOnboardInput) (*TenantOnboardResult, error) {
	admin := NewTenantAdminService()
	createIn := in.Create
	createIn.Migrate = false
	tenant, err := admin.Create(createIn)
	if err != nil {
		return nil, err
	}

	result := &TenantOnboardResult{
		Tenant: tenant,
		Steps:  []string{"created"},
	}

	host := strings.TrimSpace(in.DomainHost)
	if host != "" {
		dom, derr := NewTenantDomainService().Create(tenant.ID, TenantDomainCreateInput{
			Host:      host,
			SSLMode:   in.DomainSSL,
			IsPrimary: true,
		})
		if derr != nil {
			result.Steps = append(result.Steps, "domain_failed")
			result.LoginLinks = BuildTenantLoginLinks(tenant)
			return result, derr
		}
		result.Domain = dom
		result.Steps = append(result.Steps, "domain_pending")
	}

	ops := NewTenantOpsService()
	if in.WithMigrate {
		t2, args, qerr := ops.BeginQueuedOp(tenant.ID, models.TenantOpMigrate, in.WithSeed, in.Actor, "")
		if qerr != nil {
			result.LoginLinks = BuildTenantLoginLinks(tenant)
			return result, qerr
		}
		result.Tenant = t2
		result.Queued = true
		result.Op = models.TenantOpMigrate
		result.OpLogID = args.OpLogID
		argsCopy := args
		result.QueuedArgs = &argsCopy
		if in.WithSeed {
			result.Steps = append(result.Steps, "migrate_seed_queued")
		} else {
			result.Steps = append(result.Steps, "migrate_queued")
		}
	} else if in.WithSeed {
		t2, args, qerr := ops.BeginQueuedOp(tenant.ID, models.TenantOpSeed, false, in.Actor, "")
		if qerr != nil {
			result.LoginLinks = BuildTenantLoginLinks(tenant)
			return result, qerr
		}
		result.Tenant = t2
		result.Queued = true
		result.Op = models.TenantOpSeed
		result.OpLogID = args.OpLogID
		argsCopy := args
		result.QueuedArgs = &argsCopy
		result.Steps = append(result.Steps, "seed_queued")
	}

	result.LoginLinks = BuildTenantLoginLinks(result.Tenant)
	return result, nil
}
