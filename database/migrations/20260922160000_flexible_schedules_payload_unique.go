package migrations

import (
	"strconv"

	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

// M20260922160000FlexibleSchedulesPayloadUnique adds payload JSON and unique(handler, tenant_id).
type M20260922160000FlexibleSchedulesPayloadUnique struct{}

func (m *M20260922160000FlexibleSchedulesPayloadUnique) Signature() string {
	return "20260922160000_flexible_schedules_payload_unique"
}

func (m *M20260922160000FlexibleSchedulesPayloadUnique) Up() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("flexible_schedules") {
		return nil
	}
	if !facades.Schema().HasColumn("flexible_schedules", "payload") {
		if err := facades.Schema().Table("flexible_schedules", func(table schema.Blueprint) {
			table.Text("payload").Nullable().Comment("handler options JSON object")
		}); err != nil {
			return err
		}
	}
	if err := dedupeFlexibleSchedulesByHandlerTenant(); err != nil {
		return err
	}
	return facades.Schema().Table("flexible_schedules", func(table schema.Blueprint) {
		table.Unique("handler", "tenant_id")
	})
}

func dedupeFlexibleSchedulesByHandlerTenant() error {
	type row struct {
		ID       uint   `gorm:"column:id"`
		Handler  string `gorm:"column:handler"`
		TenantID uint   `gorm:"column:tenant_id"`
	}
	var rows []row
	if err := facades.Orm().Query().Table("flexible_schedules").Order("id asc").Get(&rows); err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, r := range rows {
		key := r.Handler + "\x00" + strconv.FormatUint(uint64(r.TenantID), 10)
		if _, ok := seen[key]; ok {
			if _, err := facades.Orm().Query().Table("flexible_schedules").Where("id", r.ID).Delete(); err != nil {
				return err
			}
			continue
		}
		seen[key] = struct{}{}
	}
	return nil
}

func (m *M20260922160000FlexibleSchedulesPayloadUnique) Down() error {
	if SkipOnTenantConnection() {
		return nil
	}
	if !facades.Schema().HasTable("flexible_schedules") {
		return nil
	}
	_ = facades.Schema().Table("flexible_schedules", func(table schema.Blueprint) {
		table.DropUnique("handler", "tenant_id")
	})
	if facades.Schema().HasColumn("flexible_schedules", "payload") {
		return facades.Schema().Table("flexible_schedules", func(table schema.Blueprint) {
			table.DropColumn("payload")
		})
	}
	return nil
}
