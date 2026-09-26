package seeders

import (
	"github.com/goravel/framework/facades"

	"goravel/app/models"
)

type DictionarySeeder struct {
}

func (s *DictionarySeeder) Signature() string {
	return "DictionarySeeder"
}

func (s *DictionarySeeder) Run() error {
	hasTranslationKey := facades.Schema().HasColumn("dictionaries", "translation_key")
	hasIsSystem := facades.Schema().HasColumn("dictionaries", "is_system")

	dictionaryToMap := func(dict models.Dictionary) map[string]any {
		data := map[string]any{
			"type":        dict.Type,
			"label":       dict.Label,
			"value":       dict.Value,
			"description": dict.Description,
			"status":      dict.Status,
			"is_system":   dict.IsSystem,
			"sort":        dict.Sort,
			"remark":      dict.Remark,
		}
		if hasTranslationKey {
			data["translation_key"] = dict.TranslationKey
		}
		if !hasIsSystem {
			delete(data, "is_system")
		}
		return data
	}

	// 创建字典数据
	dictionaries := []models.Dictionary{
		{Type: "status", Label: "启用", Value: "1", Description: "启用状态", Status: 1, IsSystem: 1, Sort: 1, TranslationKey: "enabled"},
		{Type: "status", Label: "禁用", Value: "0", Description: "禁用状态", Status: 1, IsSystem: 1, Sort: 2, TranslationKey: "disabled"},
		{Type: "menu_type", Label: "目录", Value: "1", Description: "目录类型", Status: 1, IsSystem: 1, Sort: 1, TranslationKey: "directory"},
		{Type: "menu_type", Label: "菜单", Value: "2", Description: "菜单类型", Status: 1, IsSystem: 1, Sort: 2, TranslationKey: "menu"},
		{Type: "menu_type", Label: "按钮", Value: "3", Description: "按钮类型", Status: 1, IsSystem: 1, Sort: 3, TranslationKey: "button"},
	}

	for _, dict := range dictionaries {
		// 使用 type + label 作为幂等键：存在则更新，不存在才创建
		query := facades.Orm().Query().Model(&models.Dictionary{}).Where("type", dict.Type).Where("label", dict.Label)
		count, err := query.Count()
		if err != nil {
			return err
		}

		if count == 0 {
			if err := facades.Orm().Query().Table("dictionaries").Create(dictionaryToMap(dict)); err != nil {
				return err
			}
			continue
		}

		var existing models.Dictionary
		if err := facades.Orm().Query().Model(&models.Dictionary{}).Where("type", dict.Type).Where("label", dict.Label).First(&existing); err != nil {
			return err
		}

		updateData := map[string]any{
			"value":       dict.Value,
			"description": dict.Description,
			"status":      dict.Status,
			"is_system":   dict.IsSystem,
			"sort":        dict.Sort,
		}
		if hasTranslationKey {
			updateData["translation_key"] = dict.TranslationKey
		}
		if !hasIsSystem {
			delete(updateData, "is_system")
		}
		if _, err := facades.Orm().Query().Model(&models.Dictionary{}).Where("id", existing.ID).Update(updateData); err != nil {
			return err
		}
	}

	return nil
}
