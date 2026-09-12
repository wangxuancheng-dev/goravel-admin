package services

import (
	"fmt"
	"strings"

	"goravel/app/utils"
)

func isSimpleReactModule(fields []FieldConfig, options map[string]bool) bool {
	if options != nil && options["is_tree_list"] {
		return false
	}

	for _, field := range fields {
		if field.Relation != nil && field.ShowInForm {
			return false
		}
		if !field.ShowInForm {
			continue
		}
		switch field.FormType {
		case "editor", "markdown", "image-upload", "file-upload", "date-picker", "datetime-picker", "select", "radio", "checkbox":
			return false
		}
	}

	return true
}

func (s *CodeGeneratorServiceImpl) getReactListPageTemplateName(fields []FieldConfig, options map[string]bool) string {
	if options != nil && options["is_tree_list"] {
		return "templates/react_tree_list_page.tsx.tpl"
	}
	if isSimpleReactModule(fields, options) {
		return "templates/react_simple_list.tsx.tpl"
	}
	return "templates/react_list_page.tsx.tpl"
}

func (s *CodeGeneratorServiceImpl) getReactListPageConfigTemplateName(options map[string]bool) string {
	if options != nil && options["is_tree_list"] {
		return "templates/react_tree_list.config.ts.tpl"
	}
	return "templates/react_list.config.ts.tpl"
}

func (s *CodeGeneratorServiceImpl) getReactFormModalTemplateName(options map[string]bool) string {
	if options != nil && options["is_tree_list"] {
		return "templates/react_tree_form_modal.tsx.tpl"
	}
	return "templates/react_form_modal.tsx.tpl"
}

func (s *CodeGeneratorServiceImpl) generateReactAPI(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/api.ts.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read react api template: %w", err)
	}

	hasCreate, hasEdit, hasDelete, hasExport, hasImport := frontendCrudOptions(options)
	importAsync := options != nil && options["import_async"]
	templateFields := s.convertFieldsToTemplateFields(fields)
	data := struct {
		ModelName   string
		ModuleName  string
		ModuleNameK string
		FormFields  []TemplateFieldConfig
		HasCreate   bool
		HasEdit     bool
		HasDelete   bool
		HasExport   bool
		HasImport   bool
		ImportAsync bool
	}{
		ModelName:   toPascalCase(moduleName),
		ModuleName:  moduleName,
		ModuleNameK: toKebabCase(moduleName),
		FormFields:  templateFields,
		HasCreate:   hasCreate,
		HasEdit:     hasEdit,
		HasDelete:   hasDelete,
		HasExport:   hasExport,
		HasImport:   hasImport,
		ImportAsync: importAsync,
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("html-react/src/api/%s.ts", toKebabCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateReactListPage(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateName := s.getReactListPageTemplateName(fields, options)

	templateContent, err := templates.ReadFile(templateName)
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read react list template: %w", err)
	}

	hasCreate, hasEdit, hasDelete, hasExport, hasImport := frontendCrudOptions(options)
	exportAsync := options != nil && options["export_async"]
	importAsync := options != nil && options["import_async"]
	enableBatchActions, showToolbar := frontendListOptions(options)
	templateFields := s.convertFieldsToTemplateFields(fields)
	isTreeList := options != nil && options["is_tree_list"]
	data := s.buildListPageTemplateData(moduleName, templateFields, hasCreate, hasEdit, hasDelete, hasExport, exportAsync, hasImport, importAsync, enableBatchActions, showToolbar, isTreeList)

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("html-react/src/pages/%s/%sList.tsx", toKebabCase(moduleName), toPascalCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateReactListPageConfig(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	if isSimpleReactModule(fields, options) {
		return GeneratedFile{}, fmt.Errorf("react_list_page_config skipped for simple module")
	}

	templateName := s.getReactListPageConfigTemplateName(options)
	templateContent, err := templates.ReadFile(templateName)
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read react list config template: %w", err)
	}

	hasCreate, hasEdit, hasDelete, hasExport, hasImport := frontendCrudOptions(options)
	exportAsync := options != nil && options["export_async"]
	importAsync := options != nil && options["import_async"]
	enableBatchActions, showToolbar := frontendListOptions(options)
	templateFields := s.convertFieldsToTemplateFields(fields)
	isTreeList := options != nil && options["is_tree_list"]
	data := s.buildListPageTemplateData(moduleName, templateFields, hasCreate, hasEdit, hasDelete, hasExport, exportAsync, hasImport, importAsync, enableBatchActions, showToolbar, isTreeList)

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("html-react/src/pages/%s/%s.config.ts", toKebabCase(moduleName), toCamelCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateReactFormModal(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	if isSimpleReactModule(fields, options) {
		return GeneratedFile{}, fmt.Errorf("react_form_modal skipped for simple module")
	}

	templateName := s.getReactFormModalTemplateName(options)
	templateContent, err := templates.ReadFile(templateName)
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read react form template: %w", err)
	}

	hasCreate, hasEdit, _, _, _ := frontendCrudOptions(options)
	enableBatchActions, showToolbar := frontendListOptions(options)
	templateFields := s.convertFieldsToTemplateFields(fields)
	isTreeList := options != nil && options["is_tree_list"]
	data := s.buildListPageTemplateData(moduleName, templateFields, hasCreate, hasEdit, false, false, false, false, false, enableBatchActions, showToolbar, isTreeList)

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("html-react/src/pages/%s/%sFormModal.tsx", toKebabCase(moduleName), toPascalCase(moduleName)),
		Content: content,
	}, nil
}

func frontendCrudOptions(options map[string]bool) (hasCreate, hasEdit, hasDelete, hasExport, hasImport bool) {
	hasCreate, hasEdit, hasDelete, hasExport, hasImport = true, true, true, false, false
	if options == nil {
		return
	}
	if val, ok := options["has_create"]; ok {
		hasCreate = val
	}
	if val, ok := options["has_edit"]; ok {
		hasEdit = val
	}
	if val, ok := options["has_delete"]; ok {
		hasDelete = val
	}
	if val, ok := options["has_export"]; ok {
		hasExport = val
	}
	if val, ok := options["has_import"]; ok {
		hasImport = val
	}
	return
}

func frontendListOptions(options map[string]bool) (enableBatchActions, showToolbar bool) {
	enableBatchActions, showToolbar = false, true
	if options == nil {
		return
	}
	if val, ok := options["enable_batch_actions"]; ok {
		enableBatchActions = val
	}
	if val, ok := options["show_toolbar"]; ok {
		showToolbar = val
	}
	return
}

func reactGeneratorEnabled() bool {
	return utils.CodeGeneratorFrontendReact()
}

func vueGeneratorEnabled() bool {
	return utils.CodeGeneratorFrontendVue()
}

func expandSelectedReactFiles(selectedMap map[string]bool, fields []FieldConfig, options map[string]bool) {
	if selectedMap["react_list_page"] && !isSimpleReactModule(fields, options) {
		selectedMap["react_list_page_config"] = true
		selectedMap["react_form_modal"] = true
	}
	if selectedMap["react_list_page_config"] || selectedMap["react_form_modal"] {
		selectedMap["react_list_page"] = true
	}
}

func shouldGenerateFileType(fileType string, selectedMap map[string]bool, hasSelection bool) bool {
	if !hasSelection {
		return true
	}
	return selectedMap[fileType]
}

func isReactFileType(fileType string) bool {
	return strings.HasPrefix(fileType, "react_")
}

func isVueFileType(fileType string) bool {
	switch fileType {
	case "api", "list_page_config", "list_page", "form_page":
		return true
	default:
		return false
	}
}
