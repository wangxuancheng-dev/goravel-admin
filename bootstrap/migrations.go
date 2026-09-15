package bootstrap

import (
	"goravel/database/migrations"

	"github.com/goravel/framework/contracts/database/schema"
)

func Migrations() []schema.Migration {
	return migrations.All()
}
