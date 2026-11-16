package providers

import (
	"context"
	"os"
	"time"

	"github.com/goravel/framework/contracts/foundation"
	frameworkfacades "github.com/goravel/framework/facades"
	"gorm.io/gorm"

	"goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenantcache"
	"goravel/app/tenantdb"
)

type DatabaseServiceProvider struct {
}

func (receiver *DatabaseServiceProvider) Register(app foundation.Application) {

}

func (receiver *DatabaseServiceProvider) Boot(app foundation.Application) {
	// Migrations and seeders are now registered in bootstrap/app.go via WithMigrations and WithSeeders
	ensureDefaultDatabaseReachable()
	// Wrap Schema/Orm so tenant connection names resolve through the app-owned registry
	// (tenantdb) and migrate/seed can bind a tenant Schema per goroutine.
	facades.InstallTenantAwareSchema(app)
	facades.InstallTenantAwareOrm(app)
	// Tenancy pool limits + rebuilding evicted tenant connections from the tenants table.
	services.InstallTenantConnectionResolver()
	// Platform-only tables (models.PlatformModel) are refused on tenant connections, so
	// OrmQuery(tenantCtx).Model(&models.Tenant{}) fails loudly instead of misrouting.
	tenantdb.ForbidTables(models.PlatformTables())
	// Every write to the tenants table invalidates the cached rows automatically.
	installTenantCacheHooks()
	// Orm().Fresh() replaces the gorm instance the hooks live on: re-attach them.
	facades.OnOrmFresh(func() {
		tenantcache.ResetHooks()
		installTenantCacheHooks()
	})
}

func installTenantCacheHooks() {
	q, ok := facades.PlatformQuery().(interface{ Instance() *gorm.DB })
	if !ok || q.Instance() == nil {
		// Without the hooks, writes would not invalidate: tenantcache stays disabled.
		frameworkfacades.Log().Warning("tenantcache: platform gorm instance unavailable; tenant row cache disabled")
		return
	}
	tenantcache.InstallHooks(q.Instance())
}

// ensureDefaultDatabaseReachable 在应用启动阶段检测默认数据库是否可连，避免 ORM Query 为 nil 或后续才出现隐蔽错误。
// 设置环境变量 SKIP_DB_STARTUP_PING=1 可跳过（例如仅跑不依赖库的单元测试时）。
func ensureDefaultDatabaseReachable() {
	if os.Getenv("SKIP_DB_STARTUP_PING") == "1" {
		return
	}

	orm := facades.Orm()
	if orm == nil {
		facades.Log().Fatalf("database: ORM facade is nil; check that framework database providers are registered before app providers")
	}

	sqlDB, err := orm.DB()
	if err != nil {
		facades.Log().Fatalf("database: failed to obtain sql.DB for default connection: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		name := facades.Config().GetString("database.default")
		facades.Log().Fatalf(
			"database: cannot connect to default connection %q (ping failed): %v — ensure MySQL/PostgreSQL is running and DB_HOST, DB_PORT, DB_DATABASE, DB_USERNAME in .env are correct",
			name,
			err,
		)
	}
}
