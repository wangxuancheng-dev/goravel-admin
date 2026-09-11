package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/services"
	"goravel/app/tenancy"
)

type PlatformAdminCreate struct{}

func (r *PlatformAdminCreate) Signature() string {
	return "platform:admin"
}

func (r *PlatformAdminCreate) Description() string {
	return "创建或重置平台控制台管理员（平台库）"
}

func (r *PlatformAdminCreate) Extend() command.Extend {
	return command.Extend{
		Category: "platform",
		Flags: []command.Flag{
			&command.StringFlag{Name: "name", Usage: "显示名"},
		},
	}
}

func (r *PlatformAdminCreate) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("需要 TENANCY_DRIVER=database")
		return nil
	}
	username := strings.TrimSpace(ctx.Argument(0))
	password := ctx.Argument(1)
	if username == "" || password == "" {
		ctx.Error("用法: platform:admin {username} {password} [--name=...]")
		return nil
	}
	admin, err := services.UpsertPlatformAdmin(username, password, ctx.Option("name"))
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Success(fmt.Sprintf("平台管理员就绪 username=%s id=%d", admin.Username, admin.ID))
	return nil
}
