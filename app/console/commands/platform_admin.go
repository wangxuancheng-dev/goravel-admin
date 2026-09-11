package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	appfacades "goravel/app/facades"
	"goravel/app/models"
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
	name := strings.TrimSpace(ctx.Option("name"))
	if name == "" {
		name = username
	}
	hashed, err := facades.Hash().Make(password)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}

	var existing models.PlatformAdmin
	q := appfacades.PlatformOrmQuery(nil)
	if err := q.Where("username", username).First(&existing); err == nil && existing.ID > 0 {
		if _, err := q.Model(&existing).Update(map[string]any{
			"password": hashed,
			"name":     name,
			"status":   models.PlatformAdminStatusActive,
		}); err != nil {
			ctx.Error(err.Error())
			return err
		}
		ctx.Success(fmt.Sprintf("已更新平台管理员 username=%s id=%d", username, existing.ID))
		return nil
	}

	admin := models.PlatformAdmin{
		Username: username,
		Password: hashed,
		Name:     name,
		Status:   models.PlatformAdminStatusActive,
	}
	if err := q.Create(&admin); err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Success(fmt.Sprintf("已创建平台管理员 username=%s id=%d", username, admin.ID))
	return nil
}
