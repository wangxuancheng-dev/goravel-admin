package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

type M20260716000001CreateAttachmentCategoriesTable struct{}

func (r *M20260716000001CreateAttachmentCategoriesTable) Signature() string {
	return "20260716000001_create_attachment_categories_table"
}

func (r *M20260716000001CreateAttachmentCategoriesTable) Up() error {
	if !facades.Schema().HasTable("attachment_categories") {
		if err := facades.Schema().Create("attachment_categories", func(table schema.Blueprint) {
			table.ID()
			table.String("name", 50).Comment("分类名称")
			table.String("slug", 50).Comment("分类标识")
			table.UnsignedTinyInteger("status").Default(1).Comment("状态 1:启用 0:禁用")
			table.UnsignedTinyInteger("is_system").Default(0).Comment("是否系统分类 1:是 0:否")
			table.Integer("sort").Default(0).Comment("排序")
			table.String("remark", 500).Nullable().Comment("备注")
			table.Timestamps()
			table.SoftDeletes()
			table.Unique("slug")
			table.Comment("附件分类表")
		}); err != nil {
			return err
		}
	}

	var existing models.AttachmentCategory
	err := migrationQuery().Where("slug", models.AttachmentCategorySlugUncategorized).First(&existing)
	if err != nil || existing.ID == 0 {
		uncategorized := models.AttachmentCategory{
			Name:     "未分类",
			Slug:     models.AttachmentCategorySlugUncategorized,
			Status:   1,
			IsSystem: 1,
			Sort:     0,
			Remark:   "系统默认分类，不可删除",
		}
		if createErr := migrationQuery().Create(&uncategorized); createErr != nil {
			return createErr
		}
		existing = uncategorized
	}

	if existing.ID == 0 {
		return nil
	}

	if !facades.Schema().HasTable("attachments") {
		return nil
	}
	if !facades.Schema().HasColumn("attachments", "category_id") {
		if err := facades.Schema().Table("attachments", func(table schema.Blueprint) {
			table.UnsignedBigInteger("category_id").Default(existing.ID).Comment("分类ID").After("admin_id")
			table.Index("category_id")
		}); err != nil {
			return err
		}
	}

	_, err = migrationQuery().Model(&models.Attachment{}).Where("category_id", 0).OrWhereNull("category_id").Update("category_id", existing.ID)
	return err
}

func (r *M20260716000001CreateAttachmentCategoriesTable) Down() error {
	if facades.Schema().HasTable("attachments") && facades.Schema().HasColumn("attachments", "category_id") {
		_ = facades.Schema().Table("attachments", func(table schema.Blueprint) {
			table.DropIndex("category_id")
			table.DropColumn("category_id")
		})
	}
	return facades.Schema().DropIfExists("attachment_categories")
}
