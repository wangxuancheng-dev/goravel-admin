package models

import (
	"strings"
	"time"

	"github.com/goravel/framework/database/orm"
)

const (
	TenantStatusActive   uint8 = 1
	TenantStatusDisabled uint8 = 0

	TenantIsolationDatabase = "database"
	TenantIsolationSchema   = "schema"

	TenantDriverMySQL    = "mysql"
	TenantDriverPostgres = "postgres"

	// Provision lifecycle (platform tenants table).
	TenantProvisionPending   = "pending"
	TenantProvisionMigrating = "migrating"
	TenantProvisionReady     = "ready"
	TenantProvisionFailed    = "failed"

	TenantOpMigrate = "migrate"
	TenantOpSeed    = "seed"
	TenantOpBackup  = "backup"
	TenantOpRestore = "restore"
	TenantOpPurge   = "purge"

	TenantOpStatusIdle    = "idle"
	TenantOpStatusQueued  = "queued"
	TenantOpStatusRunning = "running"
	TenantOpStatusSuccess = "success"
	TenantOpStatusFailed  = "failed"
)

// Tenant 平台库中的租户元数据（一户一库 / 一 schema）
type Tenant struct {
	orm.Model
	Code             string     `gorm:"uniqueIndex;size:64;not null;comment:租户短码" json:"code"`
	Name             string     `gorm:"size:100;not null;comment:显示名称" json:"name"`
	Status           uint8      `gorm:"default:1;index;comment:1启用 0禁用" json:"status"`
	ProvisionStatus  string     `gorm:"size:32;default:pending;index;comment:pending|migrating|ready|failed" json:"provision_status"`
	Driver           string     `gorm:"size:20;not null;comment:mysql|postgres" json:"driver"`
	Isolation        string     `gorm:"size:20;not null;comment:database|schema" json:"isolation"`
	Host             string     `gorm:"size:255;comment:空则回落平台 DB_HOST" json:"host"`
	Port             int        `gorm:"comment:0 则回落平台 DB_PORT" json:"port"`
	Database         string     `gorm:"size:128;not null;comment:目标 database 名" json:"database"`
	Schema           string     `gorm:"size:128;comment:PG schema 名；database 隔离时可空" json:"schema"`
	Username         string     `gorm:"size:128;comment:空则回落平台用户" json:"username"`
	Password         string     `gorm:"type:text;comment:空则回落平台密码；APP_KEY 加密" json:"-"`
	ConnectionName   string     `gorm:"size:64;uniqueIndex;not null;comment:运行时 connection 名" json:"connection_name"`
	LastMigrateError       string     `gorm:"type:text;comment:最近一次 migrate 错误" json:"last_migrate_error"`
	MigratedAt             *time.Time `gorm:"comment:最近一次 migrate 成功时间" json:"migrated_at"`
	SchemaMigrationCount   int64      `gorm:"default:0;comment:租户库 migrations 表行数(最近成功 migrate)" json:"schema_migration_count"`
	Maintenance            bool       `gorm:"default:false;index;comment:维护模式 挡业务流量" json:"maintenance"`
	MaintenanceMessage     string     `gorm:"size:500;comment:维护提示文案" json:"maintenance_message"`
	LastOp                 string     `gorm:"size:32;comment:最近平台运维操作" json:"last_op"`
	LastOpStatus     string     `gorm:"size:32;comment:idle|queued|running|success|failed" json:"last_op_status"`
	LastOpMessage    string     `gorm:"type:text;comment:最近运维结果" json:"last_op_message"`
	LastOpAt         *time.Time `gorm:"comment:最近运维时间" json:"last_op_at"`
	LastBackupPath    string     `gorm:"size:512;comment:最近备份路径" json:"last_backup_path"`
	StorageLimitBytes int64      `gorm:"default:0;comment:对象存储配额字节 0不限" json:"storage_limit_bytes"`
	HealthStatus      string     `gorm:"size:32;default:unknown;index;comment:ok|warn|fail|unknown" json:"health_status"`
	HealthCheckedAt   *time.Time `gorm:"comment:last health inspect" json:"health_checked_at"`
	HealthIssues      string     `gorm:"type:text;comment:json issue codes" json:"health_issues"`
	LastPingOK        bool       `gorm:"default:false;comment:last ping ok" json:"last_ping_ok"`
	LastPingMs        int64      `gorm:"default:0;comment:last ping latency ms" json:"last_ping_ms"`
	orm.SoftDeletes
}

// IsProvisionReady reports whether the tenant DB has been migrated and may accept traffic.
func (t *Tenant) IsProvisionReady() bool {
	if t == nil {
		return false
	}
	status := strings.TrimSpace(t.ProvisionStatus)
	if status == "" {
		// Pre-column rows / unset: treat as ready only when explicitly empty after legacy installs.
		return true
	}
	return status == TenantProvisionReady
}
