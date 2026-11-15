package tenantdb

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/goravel/framework/contracts/database"
	"github.com/goravel/framework/contracts/database/driver"
	contractsorm "github.com/goravel/framework/contracts/database/orm"
	contractsschema "github.com/goravel/framework/contracts/database/schema"
	contractslog "github.com/goravel/framework/contracts/log"
	contractstelemetry "github.com/goravel/framework/contracts/telemetry"
	frameworkorm "github.com/goravel/framework/database/orm"
	frameworkschema "github.com/goravel/framework/database/schema"

	"goravel/app/support/configoverlay"
)

// Conn is one tenant database connection owned by the app.
type Conn struct {
	name        string
	fingerprint string
	database    string

	cfg      *configoverlay.Overlay
	log      contractslog.Log
	driver   driver.Driver
	dbConfig database.Config
	query    contractsorm.Query
	queries  *frameworkorm.QueriesCache
	resolver contractstelemetry.Resolver
	orm      contractsorm.Orm
	sqlDB    *sql.DB
	closeFn  func() error

	migrations func() []contractsschema.Migration

	lastUsed  atomic.Int64
	closeOnce sync.Once
}

// Name is the connection name, e.g. tenant_12.
func (c *Conn) Name() string { return c.name }

// DatabaseName is the tenant database this connection points at.
func (c *Conn) DatabaseName() string { return c.database }

// DB is the underlying pool (do not Close it; the registry owns it).
func (c *Conn) DB() *sql.DB { return c.sqlDB }

// Orm is the shared Orm of this connection, for request paths. It must not be
// mutated (SetQuery / SetConnection): use NewOrm / NewSchema for that.
func (c *Conn) Orm() contractsorm.Orm { return c.orm }

// NewOrm returns a fresh Orm over the shared pool. Mutating it (the migrator calls
// SetQuery with its transaction) affects nobody else.
func (c *Conn) NewOrm() contractsorm.Orm {
	return frameworkorm.NewOrm(context.Background(), c.cfg, c.name, c.dbConfig, c.query, c.queries, c.log, nil, noFresh, c.resolver)
}

// NewSchema returns a fresh Schema (with its own Orm) over the shared pool, for
// migrate / seed / DDL. The migrator mutates the Schema it runs on (SetConnection,
// SetQuery per migration), so every run needs its own instance.
func (c *Conn) NewSchema() (contractsschema.Schema, error) {
	var migrations []contractsschema.Migration
	if c.migrations != nil {
		migrations = c.migrations()
	}
	s, err := frameworkschema.NewSchema(c.cfg, c.log, c.NewOrm(), c.driver, migrations)
	if err != nil {
		return nil, fmt.Errorf("tenantdb: schema for %s: %w", c.name, err)
	}
	return s, nil
}

// Ping checks the pool can reach the database.
func (c *Conn) Ping(ctx context.Context) error {
	if c.sqlDB == nil {
		return fmt.Errorf("tenantdb: %s has no pool", c.name)
	}
	return c.sqlDB.PingContext(ctx)
}

func (c *Conn) touch() { c.lastUsed.Store(time.Now().UnixNano()) }

func (c *Conn) idleSince() time.Time { return time.Unix(0, c.lastUsed.Load()) }

// Close releases the pool (idempotent). Only for connections that were not obtained
// from a Registry, e.g. Open for short-lived maintenance connections.
func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		if c.closeFn != nil {
			err = c.closeFn()
		}
	})
	return err
}

func noFresh(...any) {}
