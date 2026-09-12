package services

import (
	"fmt"
	"strings"
)

// ModulePermissionDef describes one API permission for a generated module.
type ModulePermissionDef struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Method      string `json:"method"`
	Path        string `json:"path"`
	Description string `json:"description"`
	Sort        int    `json:"sort"`
}

// ModuleInstallConfig controls menu/permission installation for a generated module.
type ModuleInstallConfig struct {
	Enabled        bool   `json:"enabled"`
	MenuTitle      string `json:"menu_title"`
	ParentMenuSlug string `json:"parent_menu_slug"`
	MenuSort       int    `json:"menu_sort"`
	Frontend       string `json:"frontend"` // react | vue
}

// ModuleManifest is the single source of truth for generated module metadata.
type ModuleManifest struct {
	ModuleName     string                `json:"module_name"`
	TableName      string                `json:"table_name"`
	MenuTitle      string                `json:"menu_title"`
	MenuSlug       string                `json:"menu_slug"`
	ParentMenuSlug string                `json:"parent_menu_slug"`
	Path           string                `json:"path"`
	Component      string                `json:"component"`
	Icon           string                `json:"icon"`
	MenuSort       int                   `json:"menu_sort"`
	HasCreate      bool                  `json:"has_create"`
	HasEdit        bool                  `json:"has_edit"`
	HasDelete      bool                  `json:"has_delete"`
	HasExport      bool                  `json:"has_export"`
	HasImport      bool                  `json:"has_import"`
	Permissions    []ModulePermissionDef `json:"permissions"`
}

// BuildModuleManifest derives install metadata from generator inputs.
func BuildModuleManifest(moduleName, tableName string, options map[string]bool, install *ModuleInstallConfig) (*ModuleManifest, error) {
	moduleName = strings.TrimSpace(moduleName)
	tableName = strings.TrimSpace(tableName)
	if moduleName == "" {
		return nil, fmt.Errorf("module_name is required")
	}
	if tableName == "" {
		return nil, fmt.Errorf("table_name is required")
	}

	hasCreate := optionEnabled(options, "has_create", true)
	hasEdit := optionEnabled(options, "has_edit", true)
	hasDelete := optionEnabled(options, "has_delete", true)
	hasExport := optionEnabled(options, "has_export", false)
	hasImport := optionEnabled(options, "has_import", false)

	frontend := "react"
	menuTitle := toPascalCase(moduleName)
	parentSlug := ""
	menuSort := 0
	if install != nil {
		if strings.TrimSpace(install.MenuTitle) != "" {
			menuTitle = strings.TrimSpace(install.MenuTitle)
		}
		parentSlug = strings.TrimSpace(install.ParentMenuSlug)
		menuSort = install.MenuSort
		if strings.TrimSpace(install.Frontend) != "" {
			frontend = strings.TrimSpace(install.Frontend)
		}
	}

	apiBase := "/api/admin/" + tableName
	manifest := &ModuleManifest{
		ModuleName:     moduleName,
		TableName:      tableName,
		MenuTitle:      menuTitle,
		MenuSlug:       moduleName,
		ParentMenuSlug: parentSlug,
		Path:           "/" + tableName,
		Component:      buildGeneratedComponent(moduleName, frontend),
		Icon:           "Document",
		MenuSort:       menuSort,
		HasCreate:      hasCreate,
		HasEdit:        hasEdit,
		HasDelete:      hasDelete,
		HasExport:      hasExport,
		HasImport:      hasImport,
		Permissions: []ModulePermissionDef{
			{Name: menuTitle + "列表", Slug: moduleName + ".index", Method: "GET", Path: apiBase, Description: "查看" + menuTitle + "列表", Sort: 1},
			{Name: menuTitle + "详情", Slug: moduleName + ".show", Method: "GET", Path: apiBase + "/*", Description: "查看" + menuTitle + "详情", Sort: 2},
		},
	}

	if hasCreate {
		manifest.Permissions = append(manifest.Permissions, ModulePermissionDef{
			Name: menuTitle + "创建", Slug: moduleName + ".store", Method: "POST", Path: apiBase,
			Description: "创建" + menuTitle, Sort: 3,
		})
	}
	if hasEdit {
		manifest.Permissions = append(manifest.Permissions, ModulePermissionDef{
			Name: menuTitle + "更新", Slug: moduleName + ".update", Method: "PUT", Path: apiBase + "/*",
			Description: "更新" + menuTitle, Sort: 4,
		})
	}
	if hasDelete {
		manifest.Permissions = append(manifest.Permissions, ModulePermissionDef{
			Name: menuTitle + "删除", Slug: moduleName + ".destroy", Method: "DELETE", Path: apiBase + "/*",
			Description: "删除" + menuTitle, Sort: 5,
		})
	}
	if hasExport {
		manifest.Permissions = append(manifest.Permissions, ModulePermissionDef{
			Name: menuTitle + "导出", Slug: moduleName + ".export", Method: "POST", Path: apiBase + "/export",
			Description: "导出" + menuTitle + "列表", Sort: 6,
		})
	}
	if hasImport {
		manifest.Permissions = append(manifest.Permissions, ModulePermissionDef{
			Name: menuTitle + "导入", Slug: moduleName + ".import", Method: "POST", Path: apiBase + "/import",
			Description: "导入" + menuTitle + " CSV", Sort: 7,
		})
	}

	return manifest, nil
}

func optionEnabled(options map[string]bool, key string, defaultValue bool) bool {
	if options == nil {
		return defaultValue
	}
	val, ok := options[key]
	if !ok {
		return defaultValue
	}
	return val
}

func buildGeneratedComponent(moduleName, frontend string) string {
	_ = frontend
	return fmt.Sprintf("%s/%sList", toKebabCase(moduleName), toPascalCase(moduleName))
}
