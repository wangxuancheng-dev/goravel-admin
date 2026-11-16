package services

import (
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/database/orm"
	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// Dedicated tenant database accounts (TENANCY_DB_DEDICATED_USER=true).
//
// A tenant created on the platform's database server without credentials gets an
// account of its own, created with the platform account and granted its database
// (MySQL) or owning it (PostgreSQL) and nothing else. The tenant connection logs in
// with it, so even a query that reaches the wrong database by mistake is refused by
// the server: the account cannot read the platform database or another tenant's.
// The account is dropped with the tenant's storage.
//
// The platform account needs CREATE USER and GRANT OPTION (MySQL) or CREATEROLE
// (PostgreSQL). Tenants with explicit credentials or on their own host are untouched.

const mysqlMaxUserLen = 32

// DedicatedTenantDBUserEnabled reports whether new tenants get their own DB account.
func DedicatedTenantDBUserEnabled() bool {
	return facades.Config().GetBool("tenancy.db_dedicated_user", false)
}

// DedicatedTenantDBUsername is the account name created for tenant code: t_<code>,
// shortened with a hash suffix to fit MySQL's 32 characters.
func DedicatedTenantDBUsername(code string) string {
	name := "t_" + code
	if len(name) <= mysqlMaxUserLen {
		return name
	}
	sum := sha1.Sum([]byte(code))
	suffix := "_" + hex.EncodeToString(sum[:])[:8]
	return name[:mysqlMaxUserLen-len(suffix)] + suffix
}

// usesDedicatedDBUser reports whether tenant's account is the one this package creates
// (and so may be created and dropped here). Accounts an operator supplied never are.
func usesDedicatedDBUser(tenant *models.Tenant) bool {
	return tenant != nil && !TenantUsesCustomHost(tenant) &&
		strings.TrimSpace(tenant.Username) != "" &&
		tenant.Username == DedicatedTenantDBUsername(tenant.Code)
}

func dedicatedDBUserHost() string {
	h := strings.TrimSpace(facades.Config().GetString("tenancy.db_user_host", "%"))
	if h == "" || strings.ContainsAny(h, "'\\") {
		return "%"
	}
	return h
}

// generateDBPassword returns a random 32-character alphanumeric password (safe to
// inline in CREATE USER / ALTER ROLE, which take no bind parameters).
func generateDBPassword() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789"
	b := make([]byte, 32)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b), nil
}

// createDedicatedDBUser creates (or resets the password of) tenant's account. MySQL:
// before or after the database; PostgreSQL: before it, since the account owns it.
func createDedicatedDBUser(tenant *models.Tenant) error {
	if !usesDedicatedDBUser(tenant) {
		return nil
	}
	if err := validateSQLIdent(tenant.Username); err != nil {
		return err
	}
	password, err := RevealTenantPassword(tenant.Password)
	if err != nil {
		return err
	}
	if !isAlnum(password) {
		return fmt.Errorf("dedicated tenant DB password must be alphanumeric")
	}
	q := appfacades.PlatformQuery()
	user := tenant.Username
	switch NormalizeTenantDriver(tenant.Driver) {
	case models.TenantDriverMySQL:
		account := fmt.Sprintf("'%s'@'%s'", user, dedicatedDBUserHost())
		for _, sql := range []string{
			fmt.Sprintf("CREATE USER IF NOT EXISTS %s IDENTIFIED BY '%s'", account, password),
			fmt.Sprintf("ALTER USER %s IDENTIFIED BY '%s'", account, password),
		} {
			if err := execWithSecret(q, sql); err != nil {
				return fmt.Errorf("create tenant DB user %s: %w", user, err)
			}
		}
		return nil
	case models.TenantDriverPostgres:
		exists, err := pgRoleExists(q, user)
		if err != nil {
			return err
		}
		verb := "CREATE"
		if exists {
			verb = "ALTER"
		}
		if err := execWithSecret(q, fmt.Sprintf("%s ROLE %s WITH LOGIN PASSWORD '%s'", verb, quotePGIdent(user), password)); err != nil {
			return fmt.Errorf("create tenant DB role %s: %w", user, err)
		}
		// CREATE DATABASE ... OWNER / CREATE SCHEMA ... AUTHORIZATION need the platform
		// role to be able to act as the owner. A superuser already can; otherwise this
		// grant (allowed to the role's creator) makes it so.
		_, _ = q.Exec(fmt.Sprintf("GRANT %s TO CURRENT_USER", quotePGIdent(user)))
		return nil
	}
	return nil
}

// grantDedicatedDBUser gives tenant's account its database (MySQL). PostgreSQL needs
// nothing: the account owns the database / schema it is created with.
func grantDedicatedDBUser(tenant *models.Tenant) error {
	if !usesDedicatedDBUser(tenant) || NormalizeTenantDriver(tenant.Driver) != models.TenantDriverMySQL {
		return nil
	}
	if err := validateSQLIdent(tenant.Database); err != nil {
		return err
	}
	sql := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%s'", tenant.Database, tenant.Username, dedicatedDBUserHost())
	if _, err := appfacades.PlatformQuery().Exec(sql); err != nil {
		return fmt.Errorf("grant tenant DB user %s: %w", tenant.Username, err)
	}
	return nil
}

// dropDedicatedDBUser drops tenant's account, after its database / schema.
func dropDedicatedDBUser(tenant *models.Tenant) error {
	if !usesDedicatedDBUser(tenant) {
		return nil
	}
	if err := validateSQLIdent(tenant.Username); err != nil {
		return err
	}
	q := appfacades.PlatformQuery()
	var sql string
	switch NormalizeTenantDriver(tenant.Driver) {
	case models.TenantDriverMySQL:
		sql = fmt.Sprintf("DROP USER IF EXISTS '%s'@'%s'", tenant.Username, dedicatedDBUserHost())
	case models.TenantDriverPostgres:
		sql = fmt.Sprintf("DROP ROLE IF EXISTS %s", quotePGIdent(tenant.Username))
	default:
		return nil
	}
	if _, err := q.Exec(sql); err != nil {
		return fmt.Errorf("drop tenant DB user %s: %w", tenant.Username, err)
	}
	return nil
}

// execWithSecret runs a statement that carries a password on q's connection pool
// directly: through the ORM it would be written, password included, to the SQL log and
// the Goralens debug tables. (Account statements take no bind parameters.) A driver
// error does not echo the statement.
func execWithSecret(q orm.Query, statement string) error {
	db, err := q.DB()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_, err = db.ExecContext(ctx, statement)
	return err
}

func pgRoleExists(q orm.Query, role string) (bool, error) {
	var rows []struct {
		Exists int `gorm:"column:exists"`
	}
	if err := q.Raw("SELECT 1 AS exists FROM pg_roles WHERE rolname = ?", role).Scan(&rows); err != nil {
		return false, err
	}
	return len(rows) > 0, nil
}

func isAlnum(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
