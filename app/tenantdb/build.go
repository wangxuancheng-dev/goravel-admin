package tenantdb

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/database"
	"github.com/goravel/framework/contracts/database/driver"
	contractsorm "github.com/goravel/framework/contracts/database/orm"
	contractsschema "github.com/goravel/framework/contracts/database/schema"
	contractstelemetry "github.com/goravel/framework/contracts/telemetry"
	"github.com/goravel/framework/database/db"
	frameworkgorm "github.com/goravel/framework/database/gorm"
	frameworkorm "github.com/goravel/framework/database/orm"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/carbon"
	instrumentationdatabase "github.com/goravel/framework/telemetry/instrumentation/database"
	"github.com/goravel/mysql"
	"github.com/goravel/postgres"

	"goravel/app/support/configoverlay"
	gormmysql "gorm.io/driver/mysql"
	gormio "gorm.io/gorm"
	gormschema "gorm.io/gorm/schema"
)

// Open builds a connection that is not registered anywhere. The caller owns it and
// must Close it (used for short-lived maintenance connections, e.g. CREATE DATABASE
// on a tenant host's system database).
func Open(ctx context.Context, name string, spec Spec) (*Conn, error) {
	return build(ctx, name, spec)
}

// build mirrors the framework's BuildOrm -> BuildQuery -> BuildGorm for one
// connection, minus every process-wide cache: the gorm DB, Query, Orm and driver
// below belong to this Conn only. The pool is pinged (bounded by ctx) before anything
// else talks to the database: the mysql driver's version probe (Grammar) opens its
// own connection without a timeout, which must not be what discovers a dead host.
func build(ctx context.Context, name string, spec Spec) (*Conn, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	cfg := configoverlay.New(facades.Config(), "database.connections."+name, spec.connectionValues())
	log := facades.Log()

	var drv driver.Driver
	switch spec.Driver {
	case DriverMySQL:
		drv = mysql.NewMysql(cfg, log, nil, name)
	case DriverPostgres:
		drv = postgres.NewPostgres(cfg, log, nil, name)
	}
	// Schema.SetConnection(name) resolves the driver through this key; it must yield
	// this connection's driver, not a facade that would look up the global config.
	cfg.Set("via", func() (driver.Driver, error) { return drv, nil })

	pool := drv.Pool()
	if len(pool.Writers) == 0 {
		return nil, fmt.Errorf("tenantdb: no writer configured for %s", name)
	}
	writer := pool.Writers[0]
	if err := prePing(ctx, writer.Dialector); err != nil {
		return nil, fmt.Errorf("tenantdb: ping %s: %w", name, err)
	}

	gdb, err := gormio.Open(writer.Dialector, &gormio.Config{
		DisableAutomaticPing:                     true,
		DisableForeignKeyConstraintWhenMigrating: true,
		SkipDefaultTransaction:                   true,
		Logger:                                   db.NewLogger(cfg, log).ToGorm(),
		NowFunc: func() time.Time {
			return carbon.Now().StdTime()
		},
		NamingStrategy: gormschema.NamingStrategy{
			TablePrefix:   writer.Prefix,
			SingularTable: writer.Singular,
			NoLowerCase:   writer.NoLowerCase,
			NameReplacer:  writer.NameReplacer,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("tenantdb: open %s: %w", name, err)
	}
	if err = installTableGuard(gdb); err != nil {
		return nil, fmt.Errorf("tenantdb: table guard %s: %w", name, err)
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return nil, fmt.Errorf("tenantdb: pool %s: %w", name, err)
	}
	applyPoolLimits(sqlDB, spec.Pool)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("tenantdb: ping %s: %w", name, err)
	}

	resolver := contractstelemetry.Resolver(func() contractstelemetry.Telemetry {
		return facades.App().MakeTelemetry()
	})
	var instrument *instrumentationdatabase.Instrument
	if instrumentationdatabase.Enabled(cfg) {
		instrument = instrumentationdatabase.NewInstrument(pool, name, resolver)
		if err := gdb.Use(instrumentationdatabase.NewGormPlugin(instrument)); err != nil {
			log.Warningf("tenantdb: database telemetry for %s: %v", name, err)
		}
		instrument.SetDB(sqlDB)
	}

	query := frameworkgorm.NewQuery(context.Background(), cfg, writer, gdb, drv.Grammar(), log, nil, nil, resolver)
	queries := frameworkorm.NewQueriesCache(
		map[string]contractsorm.Query{name: query},
		map[string]database.Config{name: writer},
	)

	c := &Conn{
		name:        name,
		fingerprint: spec.Fingerprint(),
		database:    writer.Database,
		cfg:         cfg,
		log:         log,
		driver:      drv,
		dbConfig:    writer,
		query:       query,
		queries:     queries,
		resolver:    resolver,
		sqlDB:       sqlDB,
		migrations:  registeredMigrations,
		closeFn: func() error {
			err := sqlDB.Close() // waits for queries already running
			if instrument != nil {
				// Not right away: a query that just returned still runs gorm's after-
				// callback, whose span end reads the histogram Shutdown nils (a nil
				// pointer panic in the framework). By then no statement can start.
				time.AfterFunc(instrumentShutdownDelay, instrument.Shutdown)
			}
			return err
		},
	}
	if spec.Driver == DriverPostgres {
		c.pgSchema = writer.Schema // as the framework resolved it for the search_path
	}
	c.orm = frameworkorm.NewOrm(context.Background(), cfg, name, writer, query, queries, log, nil, noFresh, resolver)
	c.touch()
	return c, nil
}

// instrumentShutdownDelay is how long a closed connection's telemetry outlives it.
const instrumentShutdownDelay = 30 * time.Second

// prePing reaches the server under ctx before gorm opens the connection. gorm's mysql
// dialector runs SELECT VERSION() while opening, on a connection with no timeout, so a
// dead host would otherwise hang until the OS TCP timeout. (postgres opens lazily.)
func prePing(ctx context.Context, d gormio.Dialector) error {
	md, ok := d.(*gormmysql.Dialector)
	if !ok || md.Config == nil || md.DSN == "" {
		return nil
	}
	driverName := md.DriverName
	if driverName == "" {
		driverName = gormmysql.DefaultDriverName
	}
	sqlDB, err := sql.Open(driverName, md.DSN)
	if err != nil {
		return err
	}
	defer func() { _ = sqlDB.Close() }()
	return sqlDB.PingContext(ctx)
}

func applyPoolLimits(sqlDB interface {
	SetMaxIdleConns(int)
	SetMaxOpenConns(int)
	SetConnMaxIdleTime(time.Duration)
	SetConnMaxLifetime(time.Duration)
}, p PoolLimits) {
	if p.MaxIdleConns >= 0 {
		sqlDB.SetMaxIdleConns(p.MaxIdleConns)
	}
	if p.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(p.MaxOpenConns)
	}
	if p.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(p.ConnMaxIdleTime)
	}
	if p.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(p.ConnMaxLifetime)
	}
}

// registeredMigrations returns the app's migrations (registered on the framework
// Schema at boot), so a tenant Schema can run them.
func registeredMigrations() []contractsschema.Migration {
	if s := facades.Schema(); s != nil {
		return s.Migrations()
	}
	return nil
}
