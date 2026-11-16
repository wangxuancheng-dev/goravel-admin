package feature_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"
	_ "github.com/jackc/pgx/v5/stdlib" // "pgx" database/sql driver for the other-node session
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancyctx"
	"goravel/app/utils"
)

// pgTenantEndpoint reads the PostgreSQL server used for postgres tenant tests:
//
//	TENANCY_TEST_PG_HOST=127.0.0.1 TENANCY_TEST_PG_PORT=5432
//	TENANCY_TEST_PG_USER=postgres TENANCY_TEST_PG_PASSWORD=secret
//
// The tests skip when TENANCY_TEST_PG_HOST is unset. The platform database stays
// whatever DB_CONNECTION is (MySQL in CI), so this covers mixed-driver fleets.
func pgTenantEndpoint(t *testing.T) services.TenantCreateInput {
	t.Helper()
	host := os.Getenv("TENANCY_TEST_PG_HOST")
	if host == "" {
		t.Skip("skip: TENANCY_TEST_PG_HOST not set")
	}
	port, _ := strconv.Atoi(os.Getenv("TENANCY_TEST_PG_PORT"))
	if port == 0 {
		port = 5432
	}
	user := os.Getenv("TENANCY_TEST_PG_USER")
	if user == "" {
		user = "postgres"
	}
	pw := os.Getenv("TENANCY_TEST_PG_PASSWORD")
	if pw == "" {
		pw = "postgres"
	}
	return services.TenantCreateInput{Driver: "postgres", Host: host, Port: port, Username: user, Password: pw}
}

func pgCreateTenant(t *testing.T, base services.TenantCreateInput, code, isolation, database string) *models.Tenant {
	t.Helper()
	in := base
	in.Code, in.Name, in.Isolation, in.Database = code, "PG "+code, isolation, database
	tn, err := services.NewTenantAdminService().Create(in)
	require.NoError(t, err, "create postgres tenant %s", code)
	conn := services.NewTenantConnectionService()
	t.Cleanup(func() {
		assert.NoError(t, conn.DropStorage(tn), "drop storage %s", code)
		_, _ = appfacades.PlatformQuery().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	return tn
}

func pgWithTenant(t *testing.T, tn *models.Tenant, fn func(q orm.Query) error) {
	t.Helper()
	err := withTenantSchema(services.NewTenantConnectionService(), tn, func() error {
		return fn(appfacades.OrmQuery(tenancyctx.WithTenant(context.Background(), tn.ID, tn.ConnectionName, tn.Code)))
	})
	require.NoError(t, err, "tenant %s", tn.Code)
}

// TestPostgresPlatformSchemaScope runs when the platform itself is on PostgreSQL
// (DB_CONNECTION=postgres): a schema-isolated tenant then lives in the platform
// database by default, and the platform Schema must not report the tenant's tables as
// its own (the framework's postgres HasTable scans every schema).
func TestPostgresPlatformSchemaScope(t *testing.T) {
	if !isPostgres(appfacades.PlatformQuery()) {
		t.Skip("skip: platform database is not PostgreSQL")
	}
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)

	code := fmt.Sprintf("pgp%d", time.Now().UnixNano()%1_000_000_000)
	tn, err := services.NewTenantAdminService().Create(services.TenantCreateInput{
		Code: code, Name: "PG platform schema", Driver: "postgres", Isolation: models.TenantIsolationSchema,
	})
	require.NoError(t, err)
	conn := services.NewTenantConnectionService()
	t.Cleanup(func() {
		assert.NoError(t, conn.DropStorage(tn))
		_, _ = appfacades.PlatformQuery().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	require.Equal(t, facades.Config().GetString("database.connections.postgres.database"), tn.Database,
		"precondition: the schema tenant lives in the platform database")

	const table = "only_in_tenant_schema"
	pgWithTenant(t, tn, func(q orm.Query) error {
		_, err := q.Exec("CREATE TABLE " + table + " (id int)")
		return err
	})
	require.True(t, appfacades.SchemaHasTable(tenantCtx(tn), table), "the tenant sees its table")

	platform := appfacades.PlatformConnectionName()
	assert.False(t, facades.Schema().HasTable(table), "facades.Schema() (platform) must not see a tenant schema's table")
	assert.False(t, facades.Schema().Connection(platform).HasTable(table), "Schema().Connection(platform) must not see it either")
	assert.NotContains(t, facades.Schema().GetTableListing(), table)
	assert.True(t, facades.Schema().HasTable("tenants"), "the platform still sees its own tables")
}

// TestPostgresPlatformTenantOwnAccount runs when the platform is on PostgreSQL: a
// tenant with its own account but no host is created by the platform account, and
// must still be able to migrate (create tables) with its own account: PostgreSQL 15+
// only lets the database owner create in public, and a schema only lets its owner.
func TestPostgresPlatformTenantOwnAccount(t *testing.T) {
	q := appfacades.PlatformQuery()
	if !isPostgres(q) {
		t.Skip("skip: platform database is not PostgreSQL")
	}
	withTenancyDriver(t, "database")

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	role, password := "pgown"+suffix, "pw-"+suffix
	_, err := q.Exec(fmt.Sprintf(`CREATE ROLE %q LOGIN PASSWORD '%s'`, role, password))
	if err != nil {
		t.Skipf("skip: platform account cannot create roles: %v", err)
	}
	conn := services.NewTenantConnectionService()
	var created []*models.Tenant
	t.Cleanup(func() {
		for _, tn := range created {
			assert.NoError(t, conn.DropStorage(tn))
			_, _ = appfacades.PlatformQuery().Where("id", tn.ID).ForceDelete(&models.Tenant{})
		}
		_, _ = appfacades.PlatformQuery().Exec(fmt.Sprintf(`DROP ROLE IF EXISTS %q`, role))
	})

	for _, isolation := range []string{models.TenantIsolationDatabase, models.TenantIsolationSchema} {
		tn, err := services.NewTenantAdminService().Create(services.TenantCreateInput{
			Code: "pgo" + suffix + isolation[:1], Name: "own account " + isolation,
			Driver: "postgres", Isolation: isolation, Username: role, Password: password,
		})
		require.NoError(t, err, "create %s tenant", isolation)
		created = append(created, tn)
		require.NoError(t, conn.MigrateTenant(tn), "%s tenant migrates with its own account", isolation)
		pgWithTenant(t, tn, func(q orm.Query) error {
			var user string
			if err := q.Raw("SELECT current_user").Scan(&user); err != nil {
				return err
			}
			if user != role {
				return fmt.Errorf("tenant connected as %q, want its own account %q", user, role)
			}
			return nil
		})
	}
}

// pgBackupRestore backs tn up with pg_dump, changes it, restores it over the live
// tenant with psql and checks the data is back as backed up, while bystander (a schema
// tenant in the same database) is untouched. Needs pg_dump / psql on PATH (required in
// CI, skipped locally without them).
func pgBackupRestore(t *testing.T, tn, bystander *models.Tenant) {
	t.Helper()
	for _, tool := range []string{"pg_dump", "psql"} {
		if _, err := exec.LookPath(tool); err != nil {
			if os.Getenv("CI") != "" {
				t.Fatalf("%s not on PATH in CI", tool)
			}
			t.Logf("skip backup/restore of %s: %s not on PATH", tn.Code, tool)
			return
		}
	}

	path, backupErr := services.BackupTenant(tn, 0)
	require.NoError(t, backupErr, "backup %s", tn.Code)
	t.Cleanup(func() { _ = os.RemoveAll(services.TenantBackupDir(tn.Code)) })

	// After the backup: change rows, add a table (like a new monthly shard), and
	// write to the bystander, which the restore must leave alone.
	ownMarker := "pg_iso_" + tn.Code
	bystanderKey := "bystander_after_" + tn.Code
	pgWithTenant(t, tn, func(q orm.Query) error {
		if _, err := q.Table("configs").Where("key", ownMarker).Delete(); err != nil {
			return err
		}
		if _, err := q.Exec("CREATE TABLE extra_after_backup (id int)"); err != nil {
			return err
		}
		return q.Table("configs").Create(map[string]any{
			"key": "after_backup_" + tn.Code, "value": "x", "group": "test", "type": "string", "remark": "after backup",
		})
	})
	pgWithTenant(t, bystander, func(q orm.Query) error {
		return q.Table("configs").Create(map[string]any{
			"key": bystanderKey, "value": "x", "group": "test", "type": "string", "remark": "bystander",
		})
	})

	abs, absErr := filepath.Abs(path)
	require.NoError(t, absErr)
	require.NoError(t, services.RestoreTenantFromFile(tn, abs), "restore %s over the live tenant", tn.Code)

	pgWithTenant(t, tn, func(q orm.Query) error {
		restored, err := q.Table("configs").Where("key", ownMarker).Count()
		if err != nil {
			return err
		}
		later, err := q.Table("configs").Where("key", "after_backup_"+tn.Code).Count()
		if err != nil {
			return err
		}
		if restored != 1 || later != 0 {
			return fmt.Errorf("%s after restore: backed-up row %d (want 1), later row %d (want 0)", tn.Code, restored, later)
		}
		if appfacades.SchemaHasTable(tenantCtx(tn), "extra_after_backup") {
			return fmt.Errorf("%s after restore: table created after the backup still exists", tn.Code)
		}
		return nil
	})
	pgWithTenant(t, bystander, func(q orm.Query) error {
		n, err := q.Table("configs").Where("key", bystanderKey).Count()
		if err == nil && n != 1 {
			err = fmt.Errorf("restoring %s rolled back %s (its later row is gone)", tn.Code, bystander.Code)
		}
		return err
	})
}

// TestPostgresTenantLifecycle provisions postgres tenants (database and schema
// isolation) from a platform on another driver, migrates and seeds them in
// parallel, checks data isolation and transactions, rolls back, and drops them.
func TestPostgresTenantLifecycle(t *testing.T) {
	base := pgTenantEndpoint(t)
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	appfacades.InstallTenantAwareSchema(facades.App())
	appfacades.InstallTenantAwareOrm(facades.App())

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	conn := services.NewTenantConnectionService()

	// Without a host the tenant would go on the platform server, which runs another
	// driver here (MySQL): rejected up front instead of failing half-way.
	if !isPostgres(appfacades.PlatformQuery()) {
		_, err := services.NewTenantAdminService().Create(services.TenantCreateInput{
			Code: "pgn" + suffix, Name: "no host", Driver: "postgres",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "host and port are required")
	}

	// A session of "another app instance" on the tenant database stays open until
	// after the tenant is dropped (cleanups run last-registered first): dropping must
	// still succeed, as it does on MySQL.
	var otherNode *sql.DB
	t.Cleanup(func() {
		if otherNode != nil {
			_ = otherNode.Close()
		}
	})

	// Schema-isolated tenants share one database; a database-isolated tenant
	// creates it, then two schema tenants live inside it.
	host := pgCreateTenant(t, base, "pgd"+suffix, models.TenantIsolationDatabase, "")
	shared := host.Database

	dsn := (&url.URL{
		Scheme: "postgres", User: url.UserPassword(base.Username, base.Password),
		Host: fmt.Sprintf("%s:%d", base.Host, base.Port), Path: "/" + shared, RawQuery: "sslmode=disable",
	}).String()
	var openErr error
	otherNode, openErr = sql.Open("pgx", dsn)
	require.NoError(t, openErr)
	require.NoError(t, otherNode.Ping(), "open a session on %s", shared)
	tenants := []*models.Tenant{
		host,
		pgCreateTenant(t, base, "pgs"+suffix+"a", models.TenantIsolationSchema, shared),
		pgCreateTenant(t, base, "pgs"+suffix+"b", models.TenantIsolationSchema, shared),
	}

	errs := make([]error, len(tenants))
	var wg sync.WaitGroup
	for i := range tenants {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if errs[i] = conn.MigrateTenant(tenants[i]); errs[i] == nil {
				errs[i] = conn.SeedTenant(tenants[i], "AdminSeeder")
			}
		}(i)
	}
	wg.Wait()
	for i, e := range errs {
		require.NoError(t, e, "migrate+seed %s", tenants[i].Code)
		assert.Equal(t, models.TenantProvisionReady, tenants[i].ProvisionStatus)
	}

	for _, tn := range tenants {
		pgWithTenant(t, tn, func(q orm.Query) error {
			ok, err := q.Table("admins").Where("username", "admin").Exists()
			if err == nil && !ok {
				err = fmt.Errorf("admin not seeded")
			}
			if err != nil {
				return err
			}
			return q.Table("configs").Create(map[string]any{
				"key": "pg_iso_" + tn.Code, "value": tn.Code, "group": "test", "type": "string", "remark": "pg isolation",
			})
		})
	}
	for _, tn := range tenants {
		for _, other := range tenants {
			pgWithTenant(t, tn, func(q orm.Query) error {
				n, err := q.Table("configs").Where("key", "pg_iso_"+other.Code).Count()
				want := int64(0)
				if tn.ID == other.ID {
					want = 1
				}
				if err == nil && n != want {
					err = fmt.Errorf("%s sees %d markers of %s, want %d", tn.Code, n, other.Code, want)
				}
				return err
			})
		}
	}

	// Sharded orders: the shard tables are found (information_schema on postgres is
	// per schema, not per database), the UNION query runs, and the tenant sharing the
	// database (pgd) does not see the schema tenant's orders.
	for _, tn := range tenants[:2] {
		bound := tenantCtx(tn)
		order, _, err := services.NewOrderService(bound).CreateOrder(1, 9.5, []services.OrderProduct{
			{ProductID: 1, ProductName: "sku", Price: 9.5, Quantity: 1},
		}, "req-pg-"+tn.Code, "pg order "+tn.Code)
		require.NoError(t, err, "create order on %s", tn.Code)

		got, _, err := services.NewOrderService(bound).GetOrderByOrderNo(order.OrderNo)
		require.NoError(t, err, "get order on %s", tn.Code)
		assert.Equal(t, order.OrderNo, got.OrderNo)

		// Shard-altering migrations list existing shards this way.
		shards, err := utils.GetAllExistingShardingTables(bound, "orders")
		require.NoError(t, err)
		assert.Len(t, shards, 1, "%s should list its one order shard", tn.Code)
	}
	for _, tn := range tenants[:2] {
		// The range spans two months, so the list goes through the multi-shard UNION
		// query (identifier quoting differs between MySQL and PostgreSQL).
		list, total, err := services.NewOrderService(tenantCtx(tn)).GetOrders(services.OrderFilters{
			StartTime: time.Now().AddDate(0, -1, 0),
			EndTime:   time.Now().Add(time.Hour),
		}, 1, 20)
		require.NoError(t, err, "list orders on %s", tn.Code)
		require.Equal(t, int64(1), total, "%s order total", tn.Code)
		require.Len(t, list, 1)
		assert.Equal(t, "pg order "+tn.Code, list[0].Remark, "%s must list only its own order", tn.Code)
	}

	// The order-number retry on a postgres tenant (the platform may be MySQL): the
	// duplicate is recognised and the transaction survives the failed INSERT.
	createOrderWithCollision(t, tenantCtx(tenants[0]), "pg tenant")

	// Rollback inside a tenant transaction leaves nothing behind.
	a := tenants[1]
	err := appfacades.OrmTransaction(tenantCtx(a), func(tx orm.Query) error {
		if err := tx.Table("configs").Create(map[string]any{
			"key": "pg_tx_" + suffix, "value": "x", "group": "test", "type": "string", "remark": "rollback",
		}); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	require.Error(t, err)
	pgWithTenant(t, a, func(q orm.Query) error {
		n, err := q.Table("configs").Where("key", "pg_tx_"+suffix).Count()
		if err == nil && n != 0 {
			err = fmt.Errorf("rolled back row persisted")
		}
		return err
	})

	pgBackupRestore(t, tenants[0], tenants[2])
	pgBackupRestore(t, tenants[1], tenants[2])

	// Rolling back the last batch (all migrations: they ran as one) empties the tenant
	// schema, then migrate works again. Landlord-only migrations must stay off the
	// landlord: rolling a tenant back must not drop the platform's goralens tables.
	platformSchema := facades.Schema().Connection(appfacades.PlatformConnectionName())
	hadGoralens := platformSchema.HasTable("goralens_entries")
	require.NoError(t, conn.RollbackTenant(tenants[2], 0, 0))
	if hadGoralens {
		assert.True(t, platformSchema.HasTable("goralens_entries"), "tenant rollback dropped the platform goralens table")
	}
	pgWithTenant(t, tenants[2], func(q orm.Query) error {
		if appfacades.SchemaHasTable(tenantCtx(tenants[2]), "admins") {
			return fmt.Errorf("admins still exists after full rollback")
		}
		return nil
	})
	require.NoError(t, conn.MigrateTenant(tenants[2]))
}
