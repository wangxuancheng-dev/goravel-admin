package models

import (
	"reflect"
	"sort"

	gormschema "gorm.io/gorm/schema"
)

// PlatformModel is implemented by the models stored in the platform (landlord)
// database. With TENANCY_DRIVER=database their tables exist only there, and a query
// on a tenant connection that names one fails at once (the tenantdb table guard)
// instead of reading an empty or stale copy: read and write them with
// appfacades.PlatformQuery() / PlatformOrmQuery(ctx), never OrmQuery(ctx).
//
// A new platform model implements this and is listed in platformModels; its
// migration creates the table only on the platform (SkipOnTenantConnection), which
// a test in database/migrations checks against this list.
type PlatformModel interface {
	PlatformModel()
}

func (Tenant) PlatformModel()                      {}
func (TenantDomain) PlatformModel()                {}
func (TenantOpLog) PlatformModel()                 {}
func (TenantSubscription) PlatformModel()          {}
func (TenantEntitlementOverride) PlatformModel()   {}
func (TenantEntitlementSnapshot) PlatformModel()   {}
func (TenantUsageCounter) PlatformModel()          {}
func (PlatformAdmin) PlatformModel()               {}
func (PlatformAlertDelivery) PlatformModel()       {}
func (PlatformEntitlementAuditLog) PlatformModel() {}
func (PlatformFeature) PlatformModel()             {}
func (PlatformLoginLog) PlatformModel()            {}
func (PlatformOperationLog) PlatformModel()        {}
func (PlatformPlan) PlatformModel()                {}
func (PlatformPlanEntitlement) PlatformModel()     {}
func (PlatformSetting) PlatformModel()             {}
func (FlexibleSchedule) PlatformModel()            {}

var platformModels = []PlatformModel{
	Tenant{}, TenantDomain{}, TenantOpLog{}, TenantSubscription{},
	TenantEntitlementOverride{}, TenantEntitlementSnapshot{}, TenantUsageCounter{},
	PlatformAdmin{}, PlatformAlertDelivery{}, PlatformEntitlementAuditLog{},
	PlatformFeature{}, PlatformLoginLog{}, PlatformOperationLog{}, PlatformPlan{},
	PlatformPlanEntitlement{}, PlatformSetting{}, FlexibleSchedule{},
}

// Platform-only tables without a model of their own: the database queue's.
var platformTablesWithoutModel = []string{"jobs", "failed_jobs"}

// PlatformTables returns the platform-only table names, sorted.
func PlatformTables() []string {
	naming := gormschema.NamingStrategy{}
	out := append([]string(nil), platformTablesWithoutModel...)
	for _, m := range platformModels {
		if t, ok := m.(interface{ TableName() string }); ok {
			out = append(out, t.TableName())
			continue
		}
		out = append(out, naming.TableName(reflect.TypeOf(m).Name()))
	}
	sort.Strings(out)
	return out
}
