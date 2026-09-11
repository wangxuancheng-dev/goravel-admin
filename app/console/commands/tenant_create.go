package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/services"
)

type TenantCreate struct{}

func (r *TenantCreate) Signature() string {
	return "tenant:create"
}

func (r *TenantCreate) Description() string {
	return "创建租户元数据并 CREATE DATABASE/SCHEMA（一户一库骨架）"
}

func (r *TenantCreate) Extend() command.Extend {
	return command.Extend{
		Category: "tenant",
		Flags: []command.Flag{
			&command.StringFlag{Name: "driver", Aliases: []string{"d"}, Usage: "mysql|postgres（默认跟随 DB_CONNECTION）"},
			&command.StringFlag{Name: "isolation", Aliases: []string{"i"}, Usage: "database|schema（MySQL 仅 database）"},
			&command.StringFlag{Name: "database", Usage: "目标 database 名（默认 prefix+code）"},
			&command.StringFlag{Name: "schema", Usage: "PG schema 名（isolation=schema 时默认 prefix+code）"},
			&command.BoolFlag{Name: "migrate", Aliases: []string{"m"}, Usage: "创建后立即对该连接执行 migrate"},
		},
	}
}

func (r *TenantCreate) Handle(ctx console.Context) error {
	codeRaw := ctx.Argument(0)
	name := ctx.Argument(1)
	if codeRaw == "" || name == "" {
		ctx.Error("用法: tenant:create {code} {name} [--driver=mysql|postgres] [--isolation=database|schema] [--migrate]")
		return nil
	}

	code, err := services.NormalizeTenantCode(codeRaw)
	if err != nil {
		ctx.Error(err.Error())
		return nil
	}

	driverName := strings.TrimSpace(ctx.Option("driver"))
	if driverName == "" {
		driverName = facades.Config().GetString("database.default", "mysql")
	}
	isolation := strings.TrimSpace(ctx.Option("isolation"))
	isolation, err = services.ResolveTenantIsolation(driverName, isolation)
	if err != nil {
		ctx.Error(err.Error())
		return nil
	}

	database := strings.TrimSpace(ctx.Option("database"))
	if database == "" {
		if isolation == models.TenantIsolationSchema {
			database = facades.Config().GetString("database.connections."+driverName+".database", "goravel")
		} else {
			database = services.DefaultTenantDatabaseName(code)
		}
	}
	schemaName := strings.TrimSpace(ctx.Option("schema"))
	if isolation == models.TenantIsolationSchema && schemaName == "" {
		schemaName = services.DefaultTenantSchemaName(code)
	}

	var existing models.Tenant
	if err := appfacades.PlatformOrmQuery(nil).Where("code", code).First(&existing); err == nil && existing.ID > 0 {
		ctx.Error(fmt.Sprintf("租户 code=%s 已存在 (id=%d)", code, existing.ID))
		return nil
	}

	tenant := models.Tenant{
		Code:      code,
		Name:      name,
		Status:    models.TenantStatusActive,
		Driver:    driverName,
		Isolation: isolation,
		Database:  database,
		Schema:    schemaName,
		// connection_name 先占位，入库后按 id 更新
		ConnectionName: fmt.Sprintf("tenant_pending_%s", code),
	}

	if err := appfacades.PlatformOrmQuery(nil).Create(&tenant); err != nil {
		ctx.Error("创建租户记录失败: " + err.Error())
		return err
	}

	tenant.ConnectionName = services.TenantConnectionName(tenant.ID)
	if _, err := appfacades.PlatformOrmQuery(nil).Model(&tenant).Update(map[string]any{
		"connection_name": tenant.ConnectionName,
	}); err != nil {
		ctx.Error("更新 connection_name 失败: " + err.Error())
		return err
	}

	svc := services.NewTenantConnectionService()
	if err := svc.CreateStorage(&tenant); err != nil {
		ctx.Error("创建数据库/Schema 失败: " + err.Error())
		return err
	}
	ctx.Success(fmt.Sprintf("已创建租户 id=%d code=%s connection=%s db=%s schema=%s",
		tenant.ID, tenant.Code, tenant.ConnectionName, tenant.Database, tenant.Schema))

	if ctx.OptionBool("migrate") {
		if err := svc.MigrateTenant(&tenant); err != nil {
			ctx.Error("migrate 失败: " + err.Error())
			return err
		}
		ctx.Success("租户 migrate 完成")
		if err := svc.SeedTenant(&tenant); err != nil {
			ctx.Error("seed 失败: " + err.Error())
			return err
		}
		ctx.Success("租户 seed 完成")
	}

	return nil
}
