package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"

	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

func (s *TenantConnectionService) CreateStorageWithOptions(tenant *models.Tenant, skipCreate bool) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if skipCreate {
		return nil
	}
	// A dedicated account (TENANCY_DB_DEDICATED_USER) exists before its database
	// (PostgreSQL creates the database owned by it) and is granted it after (MySQL).
	if err := createDedicatedDBUser(tenant); err != nil {
		return err
	}
	if err := s.createStorage(tenant); err != nil {
		return err
	}
	return grantDedicatedDBUser(tenant)
}

// createStorage creates the tenant's database (or schema). An existing one is never
// adopted: it is usually what a former tenant with the same code left behind, and the
// new tenant would inherit its admins and data. ErrTenantStorageExists says so; a
// database prepared on purpose is taken over with skip_create instead.
func (s *TenantConnectionService) createStorage(tenant *models.Tenant) error {
	err := s.createStorageOnce(tenant)
	if err != nil && isAlreadyExistsErr(err) {
		return apperrors.ErrTenantStorageExists.WithError(err)
	}
	return err
}

// isAlreadyExistsErr matches MySQL 1007 (database exists) and PostgreSQL 42P04
// (duplicate_database) / 42P06 (duplicate_schema).
func isAlreadyExistsErr(err error) bool {
	if errors.Is(err, errStorageExists) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "1007") || strings.Contains(msg, "42P04") || strings.Contains(msg, "42P06")
}

var errStorageExists = errors.New("database already exists")

func (s *TenantConnectionService) createStorageOnce(tenant *models.Tenant) error {
	isolation, err := ResolveTenantIsolation(tenant.Driver, tenant.Isolation)
	if err != nil {
		return err
	}

	usePlatform := !TenantUsesCustomHost(tenant)
	driverName := NormalizeTenantDriver(tenant.Driver)

	switch driverName {
	case models.TenantDriverMySQL:
		if err := validateSQLIdent(tenant.Database); err != nil {
			return err
		}
		sql := fmt.Sprintf(
			"CREATE DATABASE `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci",
			tenant.Database,
		)
		if usePlatform {
			_, err := appfacades.PlatformQuery().Exec(sql)
			return err
		}
		return s.withMaintenanceOrmQuery(tenant, func(q orm.Query) error {
			_, err := q.Exec(sql)
			return err
		})
	case models.TenantDriverPostgres:
		if isolation == models.TenantIsolationDatabase {
			if err := validateSQLIdent(tenant.Database); err != nil {
				return err
			}
			createSQL := fmt.Sprintf("CREATE DATABASE %s", quotePGIdent(tenant.Database))
			existsSQL := "SELECT 1 AS exists FROM pg_database WHERE datname = ?"
			checkAndCreate := func(q orm.Query) error {
				var rows []struct {
					Exists int `gorm:"column:exists"`
				}
				if err := q.Raw(existsSQL, tenant.Database).Scan(&rows); err != nil {
					return err
				}
				if len(rows) > 0 {
					return errStorageExists
				}
				_, err := q.Exec(createSQL)
				return err
			}
			if usePlatform {
				// Created by the platform account but used by the tenant's: make the
				// tenant account the owner, or (PostgreSQL 15+) it cannot create tables
				// in the new database's public schema.
				if owner := pgTenantOwner(tenant); owner != "" {
					createSQL += " OWNER " + quotePGIdent(owner)
				}
				return checkAndCreate(appfacades.PlatformQuery())
			}
			return s.withMaintenanceOrmQuery(tenant, checkAndCreate)
		}
		if err := validateSQLIdent(tenant.Schema); err != nil {
			return err
		}
		schemaSQL := fmt.Sprintf("CREATE SCHEMA %s", quotePGIdent(tenant.Schema))
		platformDB := facades.Config().GetString("database.connections."+tenant.Driver+".database", "")
		if usePlatform && (tenant.Database == "" || tenant.Database == platformDB) {
			// A schema the platform account creates is only writable by its owner:
			// hand it to the tenant's account when that is a different one.
			if owner := pgTenantOwner(tenant); owner != "" {
				schemaSQL += " AUTHORIZATION " + quotePGIdent(owner)
			}
			_, err := appfacades.PlatformQuery().Exec(schemaSQL)
			return err
		}
		// Schema on a dedicated / remote database: connect to that database then create schema.
		q, err := s.tenantQuery(tenant)
		if err != nil {
			return err
		}
		_, err = q.Exec(schemaSQL)
		return err
	default:
		return apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}
}

// DropStorage drops the tenant database or schema created by CreateStorage (best-effort
// cleanup), then the tenant's dedicated DB account when it has one.
func (s *TenantConnectionService) DropStorage(tenant *models.Tenant) error {
	if tenant == nil {
		return apperrors.ErrInvalidArgument.WithMessage("tenant is nil")
	}
	if err := s.dropStorage(tenant); err != nil {
		return err
	}
	return dropDedicatedDBUser(tenant)
}

func (s *TenantConnectionService) dropStorage(tenant *models.Tenant) error {
	isolation, err := ResolveTenantIsolation(tenant.Driver, tenant.Isolation)
	if err != nil {
		return err
	}
	usePlatform := !TenantUsesCustomHost(tenant)
	driverName := NormalizeTenantDriver(tenant.Driver)
	s.Forget(tenant.ConnectionName)

	platformDB := facades.Config().GetString("database.connections."+driverName+".database", "")
	if platformDB == "" {
		platformDB = facades.Config().GetString("database.connections.mysql.database", "")
	}

	switch driverName {
	case models.TenantDriverMySQL:
		if err := validateSQLIdent(tenant.Database); err != nil {
			return err
		}
		if err := rejectDroppingPlatformDatabase(tenant.Database, platformDB); err != nil {
			return err
		}
		sql := fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", tenant.Database)
		if usePlatform {
			_, err := appfacades.PlatformQuery().Exec(sql)
			return err
		}
		return s.withMaintenanceOrmQuery(tenant, func(q orm.Query) error {
			_, err := q.Exec(sql)
			return err
		})
	case models.TenantDriverPostgres:
		if isolation == models.TenantIsolationDatabase {
			if err := validateSQLIdent(tenant.Database); err != nil {
				return err
			}
			if err := rejectDroppingPlatformDatabase(tenant.Database, platformDB); err != nil {
				return err
			}
			// Other app instances (and this one's retired pools) may still hold sessions
			// on the database, which makes a plain DROP DATABASE fail on PostgreSQL.
			// WITH (FORCE) terminates them (PostgreSQL 13+; every supported release).
			sql := fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", quotePGIdent(tenant.Database))
			drop := func(q orm.Query) error {
				_, err := q.Exec(sql)
				return err
			}
			if usePlatform {
				return drop(appfacades.PlatformQuery())
			}
			return s.withMaintenanceOrmQuery(tenant, drop)
		}
		if err := validateSQLIdent(tenant.Schema); err != nil {
			return err
		}
		if err := rejectDroppingPlatformSchema(tenant.Schema); err != nil {
			return err
		}
		schemaSQL := fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", quotePGIdent(tenant.Schema))
		if usePlatform && (tenant.Database == "" || tenant.Database == platformDB) {
			_, err := appfacades.PlatformQuery().Exec(schemaSQL)
			return err
		}
		q, err := s.tenantQuery(tenant)
		if err != nil {
			return err
		}
		_, err = q.Exec(schemaSQL)
		// The connection was only opened to drop the schema; do not keep it registered.
		s.Forget(tenant.ConnectionName)
		return err
	default:
		return apperrors.ErrInvalidArgument.WithMessage("unsupported tenant driver")
	}
}
