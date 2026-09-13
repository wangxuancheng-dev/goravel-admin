package commands

import (
	"fmt"
	"strings"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/tenancy"
)

type PlatformAdminCreate struct{}

func (r *PlatformAdminCreate) Signature() string {
	return "platform:admin"
}

func (r *PlatformAdminCreate) Description() string {
	return "Create or reset a platform console admin (platform DB)"
}

func (r *PlatformAdminCreate) Extend() command.Extend {
	return command.Extend{
		Category: "platform",
		Flags: []command.Flag{
			&command.StringFlag{Name: "name", Usage: "Display name"},
			&command.StringFlag{Name: "role", Value: models.PlatformAdminRoleOwner, Usage: "owner|viewer (default owner)"},
		},
	}
}

func (r *PlatformAdminCreate) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Error("TENANCY_DRIVER=database required")
		return nil
	}
	username := strings.TrimSpace(ctx.Argument(0))
	password := ctx.Argument(1)
	if username == "" || password == "" {
		ctx.Error("usage: platform:admin {username} {password} [--name=...] [--role=owner|viewer]")
		return nil
	}
	role := models.NormalizePlatformAdminRole(ctx.Option("role"))
	admin, err := services.UpsertPlatformAdmin(username, password, ctx.Option("name"), role)
	if err != nil {
		ctx.Error(err.Error())
		return err
	}
	ctx.Success(fmt.Sprintf("platform admin ready username=%s id=%d role=%s", admin.Username, admin.ID, models.NormalizePlatformAdminRole(admin.Role)))
	return nil
}
