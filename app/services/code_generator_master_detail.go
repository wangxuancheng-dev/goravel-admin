package services

import (
	"fmt"
	"strings"
	"time"
)

// MasterDetailConfig holds sub-table settings for generator (RuoYi/Jeecg-style).
type MasterDetailConfig struct {
	Enabled         bool
	DetailTableName string
	DetailFields    []FieldConfig
}

// MasterDetailTemplateMeta is injected into templates when master-detail is on.
type MasterDetailTemplateMeta struct {
	IsMasterDetail     bool
	DetailTableName    string
	DetailModuleName   string // snake singular, e.g. quote_item
	DetailModelName    string // PascalCase, e.g. QuoteItem
	DetailFKName       string // column, e.g. quote_id
	DetailFKFieldName  string // PascalCase, e.g. QuoteID
	DetailFields       []TemplateFieldConfig
	DetailFormFields   []TemplateFieldConfig // excludes id/timestamps/fk
	DetailJsonKey      string                // "details"
}

func (s *CodeGeneratorServiceImpl) WithMasterDetail(detailTable string, detailFields []FieldConfig) CodeGeneratorService {
	clone := *s
	detailTable = strings.TrimSpace(detailTable)
	clone.masterDetail = MasterDetailConfig{
		Enabled:         detailTable != "" && len(detailFields) > 0,
		DetailTableName: detailTable,
		DetailFields:    detailFields,
	}
	return &clone
}

func (s *CodeGeneratorServiceImpl) masterDetailActive(options map[string]bool) bool {
	return s.masterDetail.Enabled && optionEnabled(options, "is_master_detail", false)
}

func deriveDetailModuleName(detailTable string) string {
	name := strings.TrimSpace(detailTable)
	if name == "" {
		return ""
	}
	switch {
	case strings.HasSuffix(name, "ies"):
		return strings.TrimSuffix(name, "ies") + "y"
	case strings.HasSuffix(name, "ses"):
		return strings.TrimSuffix(name, "es")
	case strings.HasSuffix(name, "s") && !strings.HasSuffix(name, "ss"):
		return strings.TrimSuffix(name, "s")
	default:
		return name
	}
}

func (s *CodeGeneratorServiceImpl) buildMasterDetailMeta(moduleName string, options map[string]bool) MasterDetailTemplateMeta {
	meta := MasterDetailTemplateMeta{DetailJsonKey: "details"}
	if !s.masterDetailActive(options) {
		return meta
	}
	detailTable := s.masterDetail.DetailTableName
	detailModule := deriveDetailModuleName(detailTable)
	fk := toSnakeCase(moduleName) + "_id"
	fields := ensureDetailForeignKey(s.masterDetail.DetailFields, fk, moduleName)
	templateFields := s.convertFieldsToTemplateFields(fields)
	formFields := make([]TemplateFieldConfig, 0, len(templateFields))
	for _, f := range templateFields {
		if f.Name == "id" || f.Name == "created_at" || f.Name == "updated_at" || f.Name == "deleted_at" || f.Name == fk {
			continue
		}
		if !f.ShowInForm {
			continue
		}
		formFields = append(formFields, f)
	}
	meta.IsMasterDetail = true
	meta.DetailTableName = detailTable
	meta.DetailModuleName = detailModule
	meta.DetailModelName = toPascalCase(detailModule)
	meta.DetailFKName = fk
	meta.DetailFKFieldName = toPascalCase(fk)
	meta.DetailFields = templateFields
	meta.DetailFormFields = formFields
	return meta
}

func ensureDetailForeignKey(fields []FieldConfig, fk, masterModule string) []FieldConfig {
	for _, f := range fields {
		if f.Name == fk {
			return fields
		}
	}
	fkField := FieldConfig{
		Name:         fk,
		Label:        toPascalCase(masterModule) + " ID",
		GoType:       "uint",
		DBType:       "bigint",
		Required:     true,
		ShowInList:   false,
		ShowInForm:   false,
		ShowInDetail: false,
		FormType:     "number",
	}
	return append([]FieldConfig{fkField}, fields...)
}

// expandSelectedMasterDetailFiles ensures Detail*Input (in request_create) is generated
// whenever service/update/controller need it.
func expandSelectedMasterDetailFiles(selectedMap map[string]bool, options map[string]bool) {
	if selectedMap == nil || options == nil || !options["is_master_detail"] {
		return
	}
	if selectedMap["service"] || selectedMap["request_update"] || selectedMap["controller"] {
		selectedMap["request_create"] = true
	}
}

// ValidateMasterDetailRequest checks master-detail options before generate/preview/save.
func ValidateMasterDetailRequest(options map[string]bool, detailTable string, detailFields []FieldConfig) error {
	if options == nil || !options["is_master_detail"] {
		return nil
	}
	if options["is_tree_list"] {
		return fmt.Errorf("master_detail_conflicts_tree_list")
	}
	if strings.TrimSpace(detailTable) == "" {
		return fmt.Errorf("detail_table_name_required")
	}
	if isCodeGeneratorReservedTable(detailTable) {
		return fmt.Errorf("system_table_not_allowed")
	}
	if len(detailFields) == 0 {
		return fmt.Errorf("detail_fields_required")
	}
	return nil
}

func (s *CodeGeneratorServiceImpl) generateDetailModel(moduleName string, options map[string]bool) (GeneratedFile, error) {
	meta := s.buildMasterDetailMeta(moduleName, options)
	if !meta.IsMasterDetail {
		return GeneratedFile{}, fmt.Errorf("master detail not enabled")
	}
	templateContent, err := templates.ReadFile("templates/model.tpl")
	if err != nil {
		return GeneratedFile{}, err
	}
	data := struct {
		ModelName         string
		TableName         string
		Fields            []TemplateFieldConfig
		IsTreeList        bool
		IsMasterDetail    bool
		DetailModelName   string
		DetailFKFieldName string
	}{
		ModelName:         meta.DetailModelName,
		TableName:         meta.DetailTableName,
		Fields:            meta.DetailFields,
		IsTreeList:        false,
		IsMasterDetail:    false,
		DetailModelName:   "",
		DetailFKFieldName: "",
	}
	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}
	return GeneratedFile{
		Path:    fmt.Sprintf("app/models/%s.go", toSnakeCase(meta.DetailModuleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateDetailMigration(moduleName string, options map[string]bool) (GeneratedFile, error) {
	meta := s.buildMasterDetailMeta(moduleName, options)
	if !meta.IsMasterDetail {
		return GeneratedFile{}, fmt.Errorf("master detail not enabled")
	}
	templateContent, err := templates.ReadFile("templates/migration.tpl")
	if err != nil {
		return GeneratedFile{}, err
	}
	timestamp := time.Now().Add(time.Second).Format("20060102150405")
	migFields := filterMigrationFields(meta.DetailFields)
	for i := range migFields {
		migFields[i].MigrationMethod = getMigrationMethod(migFields[i].DBType)
	}
	data := struct {
		Timestamp string
		ModelName string
		TableName string
		Fields    []TemplateFieldConfig
	}{
		Timestamp: timestamp,
		ModelName: meta.DetailModelName,
		TableName: meta.DetailTableName,
		Fields:    migFields,
	}
	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}
	return GeneratedFile{
		Path:    fmt.Sprintf("database/migrations/%s_create_%s_table.go", timestamp, meta.DetailTableName),
		Content: content,
	}, nil
}

func filterMigrationFields(fields []TemplateFieldConfig) []TemplateFieldConfig {
	out := make([]TemplateFieldConfig, 0, len(fields))
	for _, f := range fields {
		if f.Name == "id" || f.Name == "created_at" || f.Name == "updated_at" || f.Name == "deleted_at" {
			continue
		}
		out = append(out, f)
	}
	return out
}
