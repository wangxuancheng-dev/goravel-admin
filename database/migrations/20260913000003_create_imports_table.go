package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20260913000003CreateImportsTable struct{}

func (r *M20260913000003CreateImportsTable) Signature() string {
	return "20260913000003_create_imports_table"
}

func (r *M20260913000003CreateImportsTable) Up() error {
	if facades.Schema().HasTable("imports") {
		return nil
	}
	return facades.Schema().Create("imports", func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.UnsignedBigInteger("admin_id").Nullable().Comment("管理员ID")
		table.String("type", 50).Nullable().Comment("导入类型")
		table.UnsignedTinyInteger("status").Default(0).Comment("状态 0:处理中 1:成功 2:失败")
		table.String("disk", 50).Nullable().Comment("存储驱动")
		table.String("path", 255).Nullable().Comment("源文件路径")
		table.Integer("total_rows").Default(0).Comment("总行数")
		table.Integer("success_rows").Default(0).Comment("成功行数")
		table.Integer("failed_rows").Default(0).Comment("失败行数")
		table.String("error_file_path", 255).Nullable().Comment("失败行 CSV 路径")
		table.Text("error_msg").Nullable().Comment("错误信息")
		table.Timestamps()

		table.Index("admin_id")
		table.Index("type")
		table.Comment("导入记录表")
	})
}

func (r *M20260913000003CreateImportsTable) Down() error {
	return facades.Schema().DropIfExists("imports")
}
