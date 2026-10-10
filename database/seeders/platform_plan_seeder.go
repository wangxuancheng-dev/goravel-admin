package seeders

import (
	"context"
	"strings"

	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// PlatformPlanSeeder seeds the landlord reference plans (free, pro).
//
// Notes:
//   - Landlord data only: skipped on tenant databases and when platform_plans is missing.
//   - Create-if-missing: existing plans are never overwritten, so manual edits survive re-seeding.
//   - Plans carry no feature grants; the feature catalog is registered by the code generator
//     (module.* keys) and granted per plan in the platform admin UI.
type PlatformPlanSeeder struct{}

func (s *PlatformPlanSeeder) Signature() string {
	return "PlatformPlanSeeder"
}

func (s *PlatformPlanSeeder) Run() error {
	if platformPlanSeedSkipped() {
		return nil
	}
	q := appfacades.PlatformOrmQuery(context.Background())
	if q == nil {
		return nil
	}

	plans := []models.PlatformPlan{
		{
			Code:         "free",
			Name:         "Free",
			Description:  "Default plan for new tenants. Grant features here to make them available to everyone.",
			IsDefault:    true,
			IsPublic:     true,
			PriceMonthly: 0,
			PriceYearly:  0,
			Currency:     "CNY",
			Status:       1,
			Sort:         0,
		},
		{
			Code:         "pro",
			Name:         "Pro",
			Description:  "Reference paid plan (prices in minor units: 9900 = 99.00 per month). Edit freely.",
			IsDefault:    false,
			IsPublic:     true,
			PriceMonthly: 9900,
			PriceYearly:  99000,
			Currency:     "CNY",
			Status:       1,
			Sort:         10,
		},
	}

	for i := range plans {
		plan := plans[i]
		count, err := q.Model(&models.PlatformPlan{}).Where("code", plan.Code).Count()
		if err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if plan.IsDefault {
			// never create a second default plan
			defaults, err := q.Model(&models.PlatformPlan{}).Where("is_default", true).Count()
			if err != nil {
				return err
			}
			if defaults > 0 {
				plan.IsDefault = false
			}
		}
		if err := q.Create(&plan); err != nil {
			return err
		}
	}
	return nil
}

// platformPlanSeedSkipped reports true when this seed run targets a tenant database
// or the landlord tables are not migrated yet.
func platformPlanSeedSkipped() bool {
	if appfacades.BoundTenantSchema() != nil {
		return true
	}
	conn := strings.TrimSpace(facades.Schema().GetConnection())
	if conn == "" {
		conn = strings.TrimSpace(facades.Config().GetString("database.default", ""))
	}
	if strings.HasPrefix(conn, "tenant_") {
		return true
	}
	return !facades.Schema().HasTable("platform_plans")
}
