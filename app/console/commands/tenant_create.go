package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

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
			&command.StringFlag{Name: "host", Usage: "远程库 Host（空则用平台 DB_HOST）"},
			&command.StringFlag{Name: "port", Usage: "远程库 Port（0/空则用平台 DB_PORT）"},
			&command.StringFlag{Name: "username", Usage: "远程库用户（空则用平台用户）"},
			&command.StringFlag{Name: "password", Usage: "远程库密码（空则用平台密码；落库前 APP_KEY 加密）"},
			&command.BoolFlag{Name: "skip-create", Usage: "跳过 CREATE DATABASE/SCHEMA（远程库已存在）"},
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

	port := 0
	if p := strings.TrimSpace(ctx.Option("port")); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}

	tenant, err := services.NewTenantAdminService().Create(services.TenantCreateInput{
		Code:      code,
		Name:      name,
		Driver:    strings.TrimSpace(ctx.Option("driver")),
		Isolation: strings.TrimSpace(ctx.Option("isolation")),
		Database:  strings.TrimSpace(ctx.Option("database")),
		Schema:    strings.TrimSpace(ctx.Option("schema")),
		Host:      strings.TrimSpace(ctx.Option("host")),
		Port:      port,
		Username:  strings.TrimSpace(ctx.Option("username")),
		Password:   ctx.Option("password"),
		Migrate:    ctx.OptionBool("migrate"),
		SkipCreate: ctx.OptionBool("skip-create"),
	})
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Success(fmt.Sprintf("已创建租户 id=%d code=%s connection=%s db=%s schema=%s host=%s",
		tenant.ID, tenant.Code, tenant.ConnectionName, tenant.Database, tenant.Schema, tenant.Host))
	return nil
}
