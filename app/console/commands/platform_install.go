package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/services"
	"goravel/app/tenancy"
)

type PlatformInstall struct{}

func (r *PlatformInstall) Signature() string {
	return "platform:install"
}

func (r *PlatformInstall) Description() string {
	return "多租户首启：平台库 migrate + 创建平台管理员"
}

func (r *PlatformInstall) Extend() command.Extend {
	return command.Extend{
		Category: "platform",
		Flags: []command.Flag{
			&command.StringFlag{Name: "username", Aliases: []string{"u"}, Usage: "平台管理员用户名（或 PLATFORM_ADMIN_USERNAME）"},
			&command.StringFlag{Name: "password", Aliases: []string{"p"}, Usage: "平台管理员密码（或 PLATFORM_ADMIN_PASSWORD）"},
			&command.StringFlag{Name: "name", Usage: "显示名（或 PLATFORM_ADMIN_NAME）"},
		},
	}
}

func (r *PlatformInstall) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("需要 TENANCY_DRIVER=database（见 .env / website/docs/advanced/tenancy.md）")
		return nil
	}

	ctx.Info("正在执行平台库 migrate...")
	if err := facades.Artisan().Call("migrate"); err != nil {
		ctx.Error("migrate 失败: " + err.Error())
		return err
	}
	ctx.Success("平台库 migrate 完成")

	username := strings.TrimSpace(ctx.Option("username"))
	if username == "" {
		username = strings.TrimSpace(facades.Config().GetString("tenancy.platform_admin_username", ""))
	}
	password := ctx.Option("password")
	if password == "" {
		password = facades.Config().GetString("tenancy.platform_admin_password", "")
	}
	name := strings.TrimSpace(ctx.Option("name"))
	if name == "" {
		name = facades.Config().GetString("tenancy.platform_admin_name", "")
	}

	if username == "" || password == "" {
		ctx.Error("请提供管理员：--username --password，或在 .env 设置 PLATFORM_ADMIN_USERNAME / PLATFORM_ADMIN_PASSWORD")
		ctx.Info("示例: go run . artisan platform:install -u admin -p 'secret'")
		return nil
	}

	admin, err := services.UpsertPlatformAdmin(username, password, name)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Success(fmt.Sprintf("平台管理员就绪 username=%s id=%d", admin.Username, admin.ID))
	ctx.Info("下一步:")
	ctx.Info("  1) 前端打开 /platform/login")
	ctx.Info("  2) 开户: go run . artisan tenant:create {code} {name} --migrate")
	ctx.Info("  3) 远程已有库加 --skip-create；HTTP 控制台只登记/启停，migrate 请用 CLI")
	return nil
}
