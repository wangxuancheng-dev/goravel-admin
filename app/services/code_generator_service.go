package services

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"text/template"
	"time"

	"gorm.io/gorm"

	appfacades "goravel/app/facades"
)

type FieldConfig struct {
	Name         string          `json:"name"`
	Label        string          `json:"label"`
	GoType       string          `json:"go_type"`
	DBType       string          `json:"db_type"`
	Validators   []string        `json:"validators"`
	Required     bool            `json:"required"`
	Searchable   bool            `json:"searchable"`
	Sortable     bool            `json:"sortable"`
	SearchType   string          `json:"search_type"`
	SearchUIType string          `json:"search_ui_type"`
	Dictionary   string          `json:"dictionary"`
	ApiUrl       string          `json:"api_url"`
	Relation     *RelationConfig `json:"relation"`
	ShowInList   bool            `json:"show_in_list"`
	ShowInForm   bool            `json:"show_in_form"`
	ShowInDetail bool            `json:"show_in_detail"`
	Precision    int             `json:"precision"` // 精度 (总位数)
	Scale        int             `json:"scale"`     // 标度 (小数位数)
	FormType     string          `json:"form_type"`
}

type RelationConfig struct {
	Table        string `json:"table"`         // 关联表名
	ForeignKey   string `json:"foreign_key"`   // 外键字段
	DisplayField string `json:"display_field"` // 显示字段
	RelationType string `json:"relation_type"` // 关联类型：hasOne, belongsTo, hasMany
	Alias        string `json:"alias"`         // 别名
	IsTree       bool   `json:"is_tree"`       // 是否为树形数据
}

type FieldType struct {
	Label      string   `json:"label"`
	Value      string   `json:"value"`
	GoType     string   `json:"go_type"`
	DBType     string   `json:"db_type"`
	Validators []string `json:"validators"`
}

type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type FilesExistError struct {
	Files []string
}

func (e *FilesExistError) Error() string {
	return fmt.Sprintf("files already exist: %v", e.Files)
}

type CodeGeneratorService interface {
	Preview(moduleName, tableName string, fields []FieldConfig, fileType string, options map[string]bool) (string, error)
	Save(moduleName, tableName string, fields []FieldConfig, selectedFiles []string, options map[string]bool) ([]string, error)
	ForceSave(moduleName, tableName string, fields []FieldConfig, selectedFiles []string, options map[string]bool) ([]string, error)
	GetFieldTypes() []FieldType
	GetTables() ([]string, error)
	GetTableColumns(tableName string) ([]FieldConfig, error)
	Generate(moduleName, tableName string, fields []FieldConfig, selectedFiles []string, options map[string]bool) ([]GeneratedFile, error)
	GenerateWithAI(ctx context.Context, userDescription string) (*AIGeneratedConfig, error)
	InstallModule(moduleName, tableName string, options map[string]bool, install *ModuleInstallConfig) (*ModuleInstallResult, error)
}

type AIGeneratedConfig struct {
	ModuleName string        `json:"module_name"`
	TableName  string        `json:"table_name"`
	Fields     []FieldConfig `json:"fields"`
}

func buildAISystemTimestampField(name, label string) FieldConfig {
	return FieldConfig{
		Name:         name,
		Label:        label,
		GoType:       "time.Time",
		DBType:       "datetime",
		Required:     false,
		Searchable:   true,
		Sortable:     false,
		SearchType:   "=",
		SearchUIType: "datetimerange",
		ShowInList:   true,
		ShowInForm:   false,
		ShowInDetail: true,
		FormType:     "datetime-picker",
	}
}

func ensureAISystemFields(fields []FieldConfig) []FieldConfig {
	hasCreatedAt := false
	hasUpdatedAt := false
	for _, field := range fields {
		if field.Name == "created_at" {
			hasCreatedAt = true
		}
		if field.Name == "updated_at" {
			hasUpdatedAt = true
		}
	}

	if !hasCreatedAt {
		fields = append(fields, buildAISystemTimestampField("created_at", "创建时间"))
	}
	if !hasUpdatedAt {
		fields = append(fields, buildAISystemTimestampField("updated_at", "更新时间"))
	}

	return fields
}

type CodeGeneratorServiceImpl struct {
	ctx context.Context
}

//go:embed templates/*
var templates embed.FS

func NewCodeGeneratorService(ctx context.Context) CodeGeneratorService {
	return &CodeGeneratorServiceImpl{ctx: ctx}
}

func normalizeGeneratedContent(content string) string {
	lines := strings.Split(content, "\n")
	normalized := make([]string, 0, len(lines))
	blankCount := 0

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			blankCount++
			if blankCount > 1 {
				continue
			}
		} else {
			blankCount = 0
		}

		normalized = append(normalized, line)
	}

	return strings.Join(normalized, "\n")
}

func hasExportEnabled(options map[string]bool) bool {
	if options == nil {
		return false
	}
	return options["has_export"]
}

func isAsyncExportEnabled(options map[string]bool) bool {
	if !hasExportEnabled(options) {
		return false
	}
	return options["export_async"]
}

func hasImportEnabled(options map[string]bool) bool {
	if options == nil {
		return false
	}
	return options["has_import"]
}

func isAsyncImportEnabled(options map[string]bool) bool {
	if !hasImportEnabled(options) {
		return false
	}
	return options["import_async"]
}

func normalizeFrontendWhitespace(content string) string {
	content = normalizeGeneratedContent(content)

	// Remove template-artifact blank lines commonly left inside objects/arrays.
	replacements := []struct {
		pattern string
		replace string
	}{
		{`(\{\n)([ \t]*\n)+`, `$1`},            // blank lines after "{"
		{`(\[\n)([ \t]*\n)+`, `$1`},            // blank lines after "["
		{`([ \t]*,\n)([ \t]*\n)+`, `$1`},       // blank lines after commas
		{`\n([ \t]*\n)+([ \t]*[}\]])`, `\n$2`}, // blank lines before "}" or "]"
	}

	for _, item := range replacements {
		re := regexp.MustCompile(item.pattern)
		content = re.ReplaceAllString(content, item.replace)
	}

	return content
}

func isFrontendGeneratedFile(path string) bool {
	slashPath := filepath.ToSlash(path)
	if !strings.HasPrefix(slashPath, "html/") && !strings.HasPrefix(slashPath, "html-react/") {
		return false
	}

	switch strings.ToLower(filepath.Ext(path)) {
	case ".vue", ".js", ".ts", ".tsx", ".json", ".css", ".scss", ".less", ".md":
		return true
	default:
		return false
	}
}

func prettierExecutablePath(forPath string) string {
	executable := "prettier"
	if runtime.GOOS == "windows" {
		executable = "prettier.cmd"
	}

	projectDir := "html"
	if strings.HasPrefix(filepath.ToSlash(forPath), "html-react/") {
		projectDir = "html-react"
	}

	return filepath.Join(projectDir, "node_modules", ".bin", executable)
}

func formatFrontendContentWithPrettier(path, content string) string {
	if !isFrontendGeneratedFile(path) {
		return content
	}

	content = normalizeFrontendWhitespace(content)

	prettierPath := prettierExecutablePath(path)
	if _, err := os.Stat(prettierPath); err != nil {
		return content
	}

	cmd := exec.Command(prettierPath, "--stdin-filepath", path)
	cmd.Stdin = strings.NewReader(content)
	cmd.Dir = "."
	output, err := cmd.Output()
	if err != nil {
		return normalizeFrontendWhitespace(content)
	}

	return normalizeFrontendWhitespace(string(output))
}

func (s *CodeGeneratorServiceImpl) Generate(moduleName, tableName string, fields []FieldConfig, selectedFiles []string, options map[string]bool) ([]GeneratedFile, error) {
	var files []GeneratedFile

	generators := []struct {
		fileType string
		generate func(string, string, []FieldConfig, map[string]bool) (GeneratedFile, error)
		enabled  func(map[string]bool) bool
		skip     func([]FieldConfig, map[string]bool) bool
	}{
		{"model", s.generateModel, nil, nil},
		{"controller", s.generateController, nil, nil},
		{"service", s.generateService, nil, nil},
		{"request_create", s.generateRequestCreate, nil, nil},
		{"request_update", s.generateRequestUpdate, nil, nil},
		{"migration", s.generateMigration, nil, nil},
		{"export_job", s.generateExportJob, isAsyncExportEnabled, nil},
		{"import_job", s.generateImportJob, isAsyncImportEnabled, nil},
		{"api", s.generateFrontendAPI, nil, func([]FieldConfig, map[string]bool) bool { return !vueGeneratorEnabled() }},
		{"list_page_config", s.generateFrontendListPageConfig, nil, func([]FieldConfig, map[string]bool) bool { return !vueGeneratorEnabled() }},
		{"list_page", s.generateFrontendListPage, nil, func([]FieldConfig, map[string]bool) bool { return !vueGeneratorEnabled() }},
		{"form_page", s.generateFrontendFormPage, nil, func([]FieldConfig, map[string]bool) bool { return !vueGeneratorEnabled() }},
		{"react_api", s.generateReactAPI, nil, func([]FieldConfig, map[string]bool) bool { return !reactGeneratorEnabled() }},
		{"react_list_page", s.generateReactListPage, nil, func([]FieldConfig, map[string]bool) bool { return !reactGeneratorEnabled() }},
		{"react_list_page_config", s.generateReactListPageConfig, nil, func(fields []FieldConfig, options map[string]bool) bool {
			return !reactGeneratorEnabled() || isSimpleReactModule(fields, options)
		}},
		{"react_form_modal", s.generateReactFormModal, nil, func(fields []FieldConfig, options map[string]bool) bool {
			return !reactGeneratorEnabled() || isSimpleReactModule(fields, options)
		}},
	}

	// Create a map for faster lookup of selected files
	selectedMap := make(map[string]bool)
	hasSelection := len(selectedFiles) > 0
	if hasSelection {
		for _, f := range selectedFiles {
			selectedMap[f] = true
		}
		if isAsyncExportEnabled(options) {
			selectedMap["export_job"] = true
		}
		if isAsyncImportEnabled(options) {
			selectedMap["import_job"] = true
		}
		// list page always pairs with its config module
		if selectedMap["list_page"] {
			selectedMap["list_page_config"] = true
		}
		expandSelectedReactFiles(selectedMap, fields, options)
	}

	for _, gen := range generators {
		if hasSelection && !selectedMap[gen.fileType] {
			continue
		}
		if gen.enabled != nil && !gen.enabled(options) {
			continue
		}
		if gen.skip != nil && gen.skip(fields, options) {
			continue
		}

		file, err := gen.generate(moduleName, tableName, fields, options)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", gen.fileType, err)
		}

		if strings.HasSuffix(file.Path, ".go") {
			formatted, err := format.Source([]byte(file.Content))
			if err == nil {
				file.Content = normalizeGeneratedContent(string(formatted))
			} else {
				file.Content = normalizeGeneratedContent(file.Content)
			}
		} else {
			file.Content = normalizeGeneratedContent(file.Content)
		}
		file.Content = formatFrontendContentWithPrettier(file.Path, file.Content)

		files = append(files, file)
	}

	return files, nil
}

func (s *CodeGeneratorServiceImpl) getListPageTemplateName(options map[string]bool) string {
	if options != nil && options["is_tree_list"] {
		return "templates/tree_list_page.vue.tpl"
	}
	return "templates/list_page.vue.tpl"
}

func (s *CodeGeneratorServiceImpl) getListPageConfigTemplateName(options map[string]bool) string {
	if options != nil && options["is_tree_list"] {
		return "templates/tree_list_page.config.js.tpl"
	}
	return "templates/list_page.config.js.tpl"
}

func (s *CodeGeneratorServiceImpl) Preview(moduleName, tableName string, fields []FieldConfig, fileType string, options map[string]bool) (string, error) {
	templateName, err := s.getTemplateName(fileType, fields, options)
	if err != nil {
		return "", err
	}

	templateContent, err := templates.ReadFile(templateName)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %w", err)
	}

	tmpl, err := template.New("preview").Delims("<<", ">>").Parse(string(templateContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	data := s.buildTemplateData(moduleName, tableName, fields, fileType, options)
	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	content := builder.String()
	if strings.Contains(templateName, ".go") || fileType == "model" || fileType == "controller" || fileType == "service" || fileType == "request_create" || fileType == "request_update" || fileType == "migration" {
		formatted, err := format.Source([]byte(content))
		if err == nil {
			return normalizeGeneratedContent(string(formatted)), nil
		}
		return normalizeGeneratedContent(content), nil
	}

	return normalizeGeneratedContent(content), nil
}

func (s *CodeGeneratorServiceImpl) Save(moduleName, tableName string, fields []FieldConfig, selectedFiles []string, options map[string]bool) ([]string, error) {
	files, err := s.Generate(moduleName, tableName, fields, selectedFiles, options)
	if err != nil {
		return nil, err
	}

	var existingFiles []string
	for _, file := range files {
		if _, err := os.Stat(file.Path); err == nil {
			existingFiles = append(existingFiles, file.Path)
		}
	}

	if len(existingFiles) > 0 {
		return nil, &FilesExistError{
			Files: existingFiles,
		}
	}

	var savedFiles []string
	for _, file := range files {
		dir := filepath.Dir(file.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := os.WriteFile(file.Path, []byte(file.Content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}

		savedFiles = append(savedFiles, file.Path)
	}

	if s.containsGeneratedAdminController(files) {
		updated, err := s.syncAdminRoute(moduleName, options)
		if err != nil {
			return nil, err
		}
		if updated {
			savedFiles = append(savedFiles, "routes/admin.go")
		}
	}

	if s.containsGeneratedExportJob(files) {
		updated, err := s.syncQueueJobs(moduleName)
		if err != nil {
			return nil, err
		}
		if updated {
			savedFiles = append(savedFiles, "app/providers/queue_service_provider.go")
		}
	}

	if s.containsGeneratedImportJob(files) {
		updated, err := s.syncImportQueueJobs(moduleName)
		if err != nil {
			return nil, err
		}
		if updated {
			if !containsString(savedFiles, "app/providers/queue_service_provider.go") {
				savedFiles = append(savedFiles, "app/providers/queue_service_provider.go")
			}
		}
	}

	return savedFiles, nil
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func (s *CodeGeneratorServiceImpl) ForceSave(moduleName, tableName string, fields []FieldConfig, selectedFiles []string, options map[string]bool) ([]string, error) {
	files, err := s.Generate(moduleName, tableName, fields, selectedFiles, options)
	if err != nil {
		return nil, err
	}

	var savedFiles []string
	for _, file := range files {
		dir := filepath.Dir(file.Path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		if err := os.WriteFile(file.Path, []byte(file.Content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", file.Path, err)
		}

		savedFiles = append(savedFiles, file.Path)
	}

	if s.containsGeneratedAdminController(files) {
		updated, err := s.syncAdminRoute(moduleName, options)
		if err != nil {
			return nil, err
		}
		if updated {
			savedFiles = append(savedFiles, "routes/admin.go")
		}
	}

	if s.containsGeneratedExportJob(files) {
		updated, err := s.syncQueueJobs(moduleName)
		if err != nil {
			return nil, err
		}
		if updated {
			savedFiles = append(savedFiles, "app/providers/queue_service_provider.go")
		}
	}

	if s.containsGeneratedImportJob(files) {
		updated, err := s.syncImportQueueJobs(moduleName)
		if err != nil {
			return nil, err
		}
		if updated {
			if !containsString(savedFiles, "app/providers/queue_service_provider.go") {
				savedFiles = append(savedFiles, "app/providers/queue_service_provider.go")
			}
		}
	}

	return savedFiles, nil
}

func (s *CodeGeneratorServiceImpl) containsGeneratedAdminController(files []GeneratedFile) bool {
	for _, file := range files {
		if strings.HasPrefix(filepath.ToSlash(file.Path), "app/http/controllers/admin/") {
			return true
		}
	}
	return false
}

func (s *CodeGeneratorServiceImpl) containsGeneratedExportJob(files []GeneratedFile) bool {
	for _, file := range files {
		base := filepath.Base(filepath.ToSlash(file.Path))
		if strings.HasPrefix(filepath.ToSlash(file.Path), "app/jobs/") && strings.HasPrefix(base, "export_") && strings.HasSuffix(base, ".go") {
			return true
		}
	}
	return false
}

func (s *CodeGeneratorServiceImpl) containsGeneratedImportJob(files []GeneratedFile) bool {
	for _, file := range files {
		base := filepath.Base(filepath.ToSlash(file.Path))
		if strings.HasPrefix(filepath.ToSlash(file.Path), "app/jobs/") && strings.HasPrefix(base, "import_") && strings.HasSuffix(base, ".go") {
			return true
		}
	}
	return false
}

func (s *CodeGeneratorServiceImpl) syncAdminRoute(moduleName string, options map[string]bool) (bool, error) {
	const adminRoutePath = "routes/admin.go"
	content, err := os.ReadFile(adminRoutePath)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", adminRoutePath, err)
	}

	routeContent := string(content)
	modelName := toPascalCase(moduleName)
	controllerVar := strings.ToLower(modelName[:1]) + modelName[1:] + "Controller"
	controllerDecl := fmt.Sprintf("\t%s := admin.New%sController()", controllerVar, modelName)
	resourceRoute := fmt.Sprintf("\t\t\trouter.Resource(\"%ss\", %s)", moduleName, controllerVar)

	hasExport := false
	hasImport := false
	if options != nil {
		if val, ok := options["has_export"]; ok {
			hasExport = val
		}
		if val, ok := options["has_import"]; ok {
			hasImport = val
		}
	}
	exportRoute := fmt.Sprintf("\t\t\trouter.Post(\"%ss/export\", %s.Export)", moduleName, controllerVar)
	importRoute := fmt.Sprintf("\t\t\trouter.Post(\"%ss/import\", %s.Import)", moduleName, controllerVar)

	updated := false

	// 注入控制器实例声明
	if !strings.Contains(routeContent, controllerDecl) {
		marker := "\n\n\t// Admin 路由组"
		if strings.Contains(routeContent, marker) {
			routeContent = strings.Replace(routeContent, marker, "\n"+controllerDecl+marker, 1)
		} else {
			routeContent += "\n" + controllerDecl + "\n"
		}
		updated = true
	}

	// 注入 Resource 路由
	if !strings.Contains(routeContent, resourceRoute) {
		marker := "\n\t\t\t// 代码生成器（仅在开发环境可用）"
		insertBlock := "\n" + resourceRoute
		if hasExport {
			insertBlock += "\n" + exportRoute
		}
		if hasImport {
			insertBlock += "\n" + importRoute
		}

		if strings.Contains(routeContent, marker) {
			routeContent = strings.Replace(routeContent, marker, insertBlock+marker, 1)
		} else {
			routeContent += insertBlock + "\n"
		}
		updated = true
	} else {
		if hasExport && !strings.Contains(routeContent, exportRoute) {
			routeContent = strings.Replace(routeContent, resourceRoute, resourceRoute+"\n"+exportRoute, 1)
			updated = true
		}
		if hasImport && !strings.Contains(routeContent, importRoute) {
			anchor := resourceRoute
			if hasExport && strings.Contains(routeContent, exportRoute) {
				anchor = exportRoute
			}
			routeContent = strings.Replace(routeContent, anchor, anchor+"\n"+importRoute, 1)
			updated = true
		}
	}

	// Ensure shared import status / error-file routes exist (mirrors orders wiring).
	if hasImport {
		if !strings.Contains(routeContent, "importController := admin.NewImportController()") {
			marker := "\n\n\t// Admin 路由组"
			decl := "\n\timportController := admin.NewImportController()"
			if strings.Contains(routeContent, marker) {
				routeContent = strings.Replace(routeContent, marker, decl+marker, 1)
			} else {
				routeContent += decl + "\n"
			}
			updated = true
		}
		importShowRoute := "\t\t\trouter.Get(\"imports/{id}\", importController.Show)"
		importErrorRoute := "\t\t\trouter.Get(\"imports/{id}/error-file\", importController.DownloadErrorFile)"
		if !strings.Contains(routeContent, importShowRoute) {
			marker := "\n\t\t\t// 代码生成器（仅在开发环境可用）"
			block := "\n" + importShowRoute + "\n" + importErrorRoute
			if strings.Contains(routeContent, marker) {
				routeContent = strings.Replace(routeContent, marker, block+marker, 1)
			} else {
				routeContent += block + "\n"
			}
			updated = true
		} else if !strings.Contains(routeContent, importErrorRoute) {
			routeContent = strings.Replace(routeContent, importShowRoute, importShowRoute+"\n"+importErrorRoute, 1)
			updated = true
		}
	}

	if !updated {
		return false, nil
	}

	if err := os.WriteFile(adminRoutePath, []byte(routeContent), 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", adminRoutePath, err)
	}

	return true, nil
}

// syncQueueJobs injects &jobs.Export{Model}s{} into QueueServiceProvider.Jobs().
func (s *CodeGeneratorServiceImpl) syncQueueJobs(moduleName string) (bool, error) {
	const queueProviderPath = "app/providers/queue_service_provider.go"
	content, err := os.ReadFile(queueProviderPath)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", queueProviderPath, err)
	}

	updatedContent, updated := injectExportJobRegistration(string(content), toPascalCase(moduleName))
	if !updated {
		return false, nil
	}

	if err := os.WriteFile(queueProviderPath, []byte(updatedContent), 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", queueProviderPath, err)
	}
	return true, nil
}

// injectExportJobRegistration inserts an Export{Model}s job line before the search-sync marker.
func injectExportJobRegistration(content, modelName string) (string, bool) {
	return injectQueueJobRegistration(content, fmt.Sprintf("\t\t&jobs.Export%ss{},", modelName))
}

// syncImportQueueJobs injects &jobs.Import{Model}s{} into QueueServiceProvider.Jobs().
func (s *CodeGeneratorServiceImpl) syncImportQueueJobs(moduleName string) (bool, error) {
	const queueProviderPath = "app/providers/queue_service_provider.go"
	content, err := os.ReadFile(queueProviderPath)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", queueProviderPath, err)
	}

	updatedContent, updated := injectImportJobRegistration(string(content), toPascalCase(moduleName))
	if !updated {
		return false, nil
	}

	if err := os.WriteFile(queueProviderPath, []byte(updatedContent), 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", queueProviderPath, err)
	}
	return true, nil
}

// injectImportJobRegistration inserts an Import{Model}s job line before the search-sync marker.
func injectImportJobRegistration(content, modelName string) (string, bool) {
	return injectQueueJobRegistration(content, fmt.Sprintf("\t\t&jobs.Import%ss{},", modelName))
}

func injectQueueJobRegistration(content, jobLine string) (string, bool) {
	if strings.Contains(content, jobLine) {
		return content, false
	}

	marker := "\t\t// 搜索引擎同步任务"
	if strings.Contains(content, marker) {
		return strings.Replace(content, marker, jobLine+"\n"+marker, 1), true
	}

	idx := strings.LastIndex(content, "return []queue.Job{")
	if idx >= 0 {
		rest := content[idx:]
		closeIdx := strings.Index(rest, "\n\t}")
		if closeIdx >= 0 {
			insertAt := idx + closeIdx
			return content[:insertAt] + "\n" + jobLine + content[insertAt:], true
		}
	}
	return content + "\n" + jobLine + "\n", true
}

func (s *CodeGeneratorServiceImpl) InstallModule(moduleName, tableName string, options map[string]bool, install *ModuleInstallConfig) (*ModuleInstallResult, error) {
	manifest, err := BuildModuleManifest(moduleName, tableName, options, install)
	if err != nil {
		return nil, err
	}

	return NewModuleInstaller(s.ctx).Install(manifest)
}

func (s *CodeGeneratorServiceImpl) getGormDB() (*gorm.DB, error) {
	// Prefer tenant connection from request/job ctx (OrmQuery), not platform default.
	query := appfacades.OrmQuery(s.ctx)
	if query == nil {
		return nil, fmt.Errorf("Query() returned nil")
	}

	// Use reflection to call Instance() on Query object
	// Because Query interface in Goravel framework might not expose Instance() method directly
	// but the implementation (gorm.Query) does have it.
	queryValue := reflect.ValueOf(query)
	instanceMethod := queryValue.MethodByName("Instance")
	if !instanceMethod.IsValid() {
		// Try to see if it's wrapped
		if queryValue.Kind() == reflect.Interface {
			queryValue = queryValue.Elem()
			instanceMethod = queryValue.MethodByName("Instance")
		}
	}

	if instanceMethod.IsValid() {
		results := instanceMethod.Call(nil)
		if len(results) > 0 {
			if db, ok := results[0].Interface().(*gorm.DB); ok {
				return db, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to get *gorm.DB from ORM")
}

func (s *CodeGeneratorServiceImpl) GetTables() ([]string, error) {
	db, err := s.getGormDB()
	if err != nil {
		return nil, err
	}

	var tables []string
	if err := db.Raw("SHOW TABLES").Scan(&tables).Error; err != nil {
		return nil, err
	}

	// 过滤掉系统默认表
	ignoreTables := map[string]bool{
		"users":                  true,
		"roles":                  true,
		"permissions":            true,
		"menus":                  true,
		"role_users":             true,
		"role_permissions":       true,
		"admin_role":             true,
		"role_menu":              true,
		"role_permission":        true,
		"migrations":             true,
		"personal_access_tokens": true,
		"settings":               true,
		"dict_types":             true,
		"dict_data":              true,
		"admins":                 true,
		"failed_jobs":            true,
		"jobs":                   true,
		"attachments":            true,
		"blacklists":             true,
		"configs":                true,
		"currencies":             true,
		"departments":            true,
		"positions":              true,
		"dictionaries":           true,
		"exports":                true,
		"login_logs":             true,
		"notifications":          true,
		"operation_logs":         true,
		"system_logs":            true,
		"slow_query_logs":        true,
		// "payment_methods":        true,
		// "payments":               true,
	}

	var filteredTables []string
	for _, table := range tables {
		if ignoreTables[table] {
			continue
		}
		// 过滤分表逻辑：
		// 1. 过滤掉包含年份月份后缀的表 (例如 _202512, _202601)
		// 2. 过滤掉 Hash 分表 (例如 _1, _2, _100)
		isShardedTable := false

		// 查找最后一个下划线
		lastUnderscoreIndex := strings.LastIndex(table, "_")
		if lastUnderscoreIndex != -1 && lastUnderscoreIndex < len(table)-1 {
			suffix := table[lastUnderscoreIndex+1:]

			// 检查后缀是否纯数字
			isNumeric := true
			for _, ch := range suffix {
				if ch < '0' || ch > '9' {
					isNumeric = false
					break
				}
			}

			if isNumeric {
				isShardedTable = true
			}
		}

		if !isShardedTable {
			filteredTables = append(filteredTables, table)
		}
	}

	return filteredTables, nil
}

func (s *CodeGeneratorServiceImpl) GetTableColumns(tableName string) ([]FieldConfig, error) {
	db, err := s.getGormDB()
	if err != nil {
		return nil, err
	}

	columnTypes, err := db.Migrator().ColumnTypes(tableName)
	if err != nil {
		return nil, err
	}

	var fields []FieldConfig
	for _, ct := range columnTypes {
		name := ct.Name()
		if name == "deleted_at" || name == "id" {
			continue
		}
		dbType := ct.DatabaseTypeName()

		// Skip internal fields if needed, but we usually want them
		// Mapping logic
		goType := "string"
		jsonDBType := "string"

		lowerDBType := strings.ToLower(dbType)
		if strings.Contains(lowerDBType, "int") {
			if strings.Contains(lowerDBType, "bigint") {
				goType = "int64"
				jsonDBType = "bigInteger"
			} else if strings.Contains(lowerDBType, "tinyint") {
				goType = "int"
				jsonDBType = "unsignedTinyInteger"
			} else {
				goType = "int"
				jsonDBType = "integer"
			}
		} else if strings.Contains(lowerDBType, "char") {
			goType = "string"
			jsonDBType = "string"
		} else if strings.Contains(lowerDBType, "text") {
			goType = "string"
			jsonDBType = "text"
		} else if strings.Contains(lowerDBType, "datetime") || strings.Contains(lowerDBType, "timestamp") {
			goType = "time.Time"
			jsonDBType = "datetime"
		} else if strings.Contains(lowerDBType, "date") {
			goType = "time.Time"
			jsonDBType = "date"
		} else if strings.Contains(lowerDBType, "decimal") || strings.Contains(lowerDBType, "double") || strings.Contains(lowerDBType, "float") {
			goType = "float64"
			jsonDBType = "decimal"
		} else if strings.Contains(lowerDBType, "bool") {
			goType = "bool"
			jsonDBType = "boolean"
		} else if strings.Contains(lowerDBType, "json") {
			goType = "string"
			jsonDBType = "json"
		}

		// Refine based on column name
		if name == "id" || name == "created_at" || name == "updated_at" {
			jsonDBType = "datetime"
			if name == "id" {
				jsonDBType = "bigInteger"
				goType = "int64"
			} else {
				goType = "time.Time"
			}
		}

		// Skip internal fields if needed, but we usually want them
		if name == "deleted_at" || name == "id" {
			continue
		}

		// SearchType default
		searchType := "="
		if goType == "string" {
			searchType = "like"
		}

		// Label defaults to name
		label := name
		comment, _ := ct.Comment()
		if comment != "" {
			label = comment
		}

		nullable, _ := ct.Nullable()

		// Precision/Scale
		precision, scale, _ := ct.DecimalSize()

		// 系统字段默认不在表单中显示
		showInForm := true
		if name == "created_at" || name == "updated_at" || name == "deleted_at" {
			showInForm = false
		}

		fields = append(fields, FieldConfig{
			Name:         name,
			Label:        label,
			GoType:       goType,
			DBType:       jsonDBType, // 这里是后端推断的类型，前端会根据这个去匹配 fieldTypes
			Required:     !nullable,
			Searchable:   true,
			Sortable:     false,
			ShowInList:   true,
			ShowInForm:   showInForm,
			ShowInDetail: true,
			SearchType:   searchType,
			SearchUIType: getSearchUIType("", jsonDBType, name),
			Dictionary:   "",
			ApiUrl:       "",
			Precision:    int(precision),
			Scale:        int(scale),
		})

		// 智能推断 FormType 和 Relation
		field := &fields[len(fields)-1]

		// 1. 推断 FormType
		if field.Name == "id" || field.Name == "created_at" || field.Name == "updated_at" || field.Name == "deleted_at" {
			// 系统字段，通常不需要在表单中显示，或者只读
			field.FormType = "input"
		} else if strings.HasSuffix(field.Name, "_id") {
			field.FormType = "select"
			// 推断关联
			refTable := strings.TrimSuffix(field.Name, "_id") + "s" // 简单复数化
			// 树形数据只有在自定义API时才可能，默认关联不使用树形
			field.Relation = &RelationConfig{
				Table:        refTable,
				ForeignKey:   field.Name,
				DisplayField: "name", // 默认猜测 name
				RelationType: "belongsTo",
				Alias:        "",    // 默认使用表名转 PascalCase
				IsTree:       false, // 默认不是树形，只有自定义API时才可能是树形
			}
		} else if strings.Contains(field.Name, "image") || strings.Contains(field.Name, "avatar") || strings.Contains(field.Name, "photo") || strings.Contains(field.Name, "pic") {
			field.FormType = "image-upload"
		} else if strings.Contains(field.Name, "file") {
			field.FormType = "file-upload"
		} else if (strings.Contains(field.Name, "content") || strings.Contains(field.Name, "description") || strings.Contains(field.Name, "detail")) && field.DBType == "text" {
			field.FormType = "editor"
		} else {
			field.FormType = getFormType(field.DBType)
		}
	}
	return fields, nil
}

func (s *CodeGeneratorServiceImpl) GetFieldTypes() []FieldType {
	return []FieldType{
		{
			Label:      "字符串",
			Value:      "string",
			GoType:     "string",
			DBType:     "string",
			Validators: []string{"string", "max:255"},
		},
		{
			Label:      "文本",
			Value:      "text",
			GoType:     "string",
			DBType:     "text",
			Validators: []string{"string"},
		},
		{
			Label:      "整数",
			Value:      "integer",
			GoType:     "int",
			DBType:     "integer",
			Validators: []string{"integer"},
		},
		{
			Label:      "大整数",
			Value:      "bigInteger",
			GoType:     "int64",
			DBType:     "bigInteger",
			Validators: []string{"integer"},
		},
		{
			Label:      "无符号大整数",
			Value:      "unsignedBigInteger",
			GoType:     "uint64",
			DBType:     "unsignedBigInteger",
			Validators: []string{"integer", "min:0"},
		},
		{
			Label:      "无符号小整数",
			Value:      "unsignedTinyInteger",
			GoType:     "uint8",
			DBType:     "unsignedTinyInteger",
			Validators: []string{"integer", "min:0", "max:255"},
		},
		{
			Label:      "小数",
			Value:      "decimal",
			GoType:     "float64",
			DBType:     "decimal",
			Validators: []string{"numeric"},
		},
		{
			Label:      "布尔值",
			Value:      "boolean",
			GoType:     "bool",
			DBType:     "boolean",
			Validators: []string{"boolean"},
		},
		{
			Label:      "日期",
			Value:      "date",
			GoType:     "time.Time",
			DBType:     "date",
			Validators: []string{"date"},
		},
		{
			Label:      "日期时间",
			Value:      "datetime",
			GoType:     "time.Time",
			DBType:     "datetime",
			Validators: []string{"date"},
		},
		{
			Label:      "时间戳",
			Value:      "timestamp",
			GoType:     "int64",
			DBType:     "timestamp",
			Validators: []string{"integer"},
		},
		{
			Label:      "JSON",
			Value:      "json",
			GoType:     "string",
			DBType:     "json",
			Validators: []string{"json"},
		},
	}
}

func (s *CodeGeneratorServiceImpl) getTemplateName(fileType string, fields []FieldConfig, options map[string]bool) (string, error) {
	switch fileType {
	case "model":
		return "templates/model.tpl", nil
	case "controller":
		return "templates/controller.tpl", nil
	case "service":
		return "templates/service.tpl", nil
	case "request_create":
		return "templates/request_create.tpl", nil
	case "request_update":
		return "templates/request_update.tpl", nil
	case "migration":
		return "templates/migration.tpl", nil
	case "export_job":
		return "templates/export_job.tpl", nil
	case "import_job":
		return "templates/import_job.tpl", nil
	case "api":
		return "templates/api.js.tpl", nil
	case "list_page_config":
		return s.getListPageConfigTemplateName(options), nil
	case "list_page":
		return s.getListPageTemplateName(options), nil
	case "form_page":
		return "templates/form_page.vue.tpl", nil
	case "react_api":
		return "templates/api.ts.tpl", nil
	case "react_list_page":
		return s.getReactListPageTemplateName(fields, options), nil
	case "react_list_page_config":
		return s.getReactListPageConfigTemplateName(options), nil
	case "react_form_modal":
		return s.getReactFormModalTemplateName(options), nil
	default:
		return "", fmt.Errorf("unknown file type: %s", fileType)
	}
}

func (s *CodeGeneratorServiceImpl) buildTemplateData(moduleName, tableName string, fields []FieldConfig, fileType string, options map[string]bool) any {
	templateFields := s.convertFieldsToTemplateFields(fields)

	// Default options
	hasCreate := true
	hasEdit := true
	hasDelete := true
	hasExport := false
	exportAsync := false
	hasImport := false
	importAsync := false
	enableBatchActions := false
	showToolbar := true

	if options != nil {
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
		if val, ok := options["export_async"]; ok {
			exportAsync = val
		}
		if val, ok := options["has_import"]; ok {
			hasImport = val
		}
		if val, ok := options["import_async"]; ok {
			importAsync = val
		}
		if val, ok := options["enable_batch_actions"]; ok {
			enableBatchActions = val
		}
		if val, ok := options["show_toolbar"]; ok {
			showToolbar = val
		}
	}

	switch fileType {
	case "model":
		return struct {
			ModelName string
			TableName string
			Fields    []TemplateFieldConfig
			IsTreeList bool
		}{
			ModelName:  toPascalCase(moduleName),
			TableName:  tableName,
			Fields:     templateFields,
			IsTreeList: optionEnabled(options, "is_tree_list", false),
		}
	case "controller":
		var searchableFields []TemplateFieldConfig
		for _, field := range templateFields {
			if field.Searchable {
				searchableFields = append(searchableFields, field)
			}
		}
		return struct {
			ControllerName    string
			ServiceName       string
			ModelName         string
			ModuleName        string
			ListFields        []TemplateFieldConfig
			SearchableFields  []TemplateFieldConfig
			RequestCreateName string
			RequestUpdateName string
			HasCreate         bool
			HasEdit           bool
			HasDelete         bool
			HasExport         bool
			ExportAsync       bool
			HasImport         bool
			ImportAsync       bool
			IsTreeList        bool
		}{
			ControllerName:    toPascalCase(moduleName) + "Controller",
			ServiceName:       toPascalCase(moduleName) + "Service",
			ModelName:         toPascalCase(moduleName),
			ModuleName:        moduleName,
			ListFields:        templateFields,
			SearchableFields:  searchableFields,
			RequestCreateName: toPascalCase(moduleName) + "Create",
			RequestUpdateName: toPascalCase(moduleName) + "Update",
			HasCreate:         hasCreate,
			HasEdit:           hasEdit,
			HasDelete:         hasDelete,
			HasExport:         hasExport,
			ExportAsync:       exportAsync,
			HasImport:         hasImport,
			ImportAsync:       importAsync,
			IsTreeList:        optionEnabled(options, "is_tree_list", false),
		}
	case "service":
		var searchableFields []TemplateFieldConfig
		for _, field := range templateFields {
			if field.Searchable {
				searchableFields = append(searchableFields, field)
			}
		}
		return struct {
			ServiceName       string
			ModelName         string
			ModuleName        string
			SearchableFields  []TemplateFieldConfig
			RequestCreateName string
			RequestUpdateName string
			FormFields        []TemplateFieldConfig
			HasCreate         bool
			HasEdit           bool
			HasDelete         bool
			HasExport         bool
			ExportAsync       bool
			HasImport         bool
			ImportAsync       bool
			IsTreeList        bool
			ParentIDFieldName string
		}{
			ServiceName:       toPascalCase(moduleName) + "Service",
			ModelName:         toPascalCase(moduleName),
			ModuleName:        moduleName,
			SearchableFields:  searchableFields,
			RequestCreateName: toPascalCase(moduleName) + "Create",
			RequestUpdateName: toPascalCase(moduleName) + "Update",
			FormFields:        templateFields,
			HasCreate:         hasCreate,
			HasEdit:           hasEdit,
			HasDelete:         hasDelete,
			HasExport:         hasExport,
			ExportAsync:       exportAsync,
			HasImport:         hasImport,
			ImportAsync:       importAsync,
			IsTreeList:        optionEnabled(options, "is_tree_list", false),
			ParentIDFieldName: resolveParentIDFieldName(templateFields),
		}
	case "request_create":
		return struct {
			ModuleName        string
			TableName         string
			FormFields        []TemplateFieldConfig
			RequestCreateName string
		}{
			ModuleName:        moduleName,
			TableName:         tableName,
			FormFields:        templateFields,
			RequestCreateName: toPascalCase(moduleName) + "Create",
		}
	case "request_update":
		return struct {
			ModuleName        string
			TableName         string
			FormFields        []TemplateFieldConfig
			RequestUpdateName string
		}{
			ModuleName:        moduleName,
			TableName:         tableName,
			FormFields:        templateFields,
			RequestUpdateName: toPascalCase(moduleName) + "Update",
		}
	case "migration":
		for i := range templateFields {
			templateFields[i].MigrationMethod = getMigrationMethod(templateFields[i].DBType)
		}
		return struct {
			ModelName string
			TableName string
			Fields    []TemplateFieldConfig
		}{
			ModelName: toPascalCase(moduleName),
			TableName: tableName,
			Fields:    templateFields,
		}
	case "api":
		return struct {
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
	case "export_job", "import_job":
		return struct {
			ModelName  string
			ModuleName string
			ListFields []TemplateFieldConfig
		}{
			ModelName:  toPascalCase(moduleName),
			ModuleName: moduleName,
			ListFields: templateFields,
		}
	case "list_page", "list_page_config":
		return s.buildListPageTemplateData(moduleName, templateFields, hasCreate, hasEdit, hasDelete, hasExport, exportAsync, hasImport, importAsync, enableBatchActions, showToolbar, optionEnabled(options, "is_tree_list", false))
	case "react_api":
		return struct {
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
	case "react_list_page", "react_list_page_config", "react_form_modal":
		return s.buildListPageTemplateData(moduleName, templateFields, hasCreate, hasEdit, hasDelete, hasExport, exportAsync, hasImport, importAsync, enableBatchActions, showToolbar, optionEnabled(options, "is_tree_list", false))
	case "form_page":
		formFields := applyTreeListFieldFlags(templateFields, optionEnabled(options, "is_tree_list", false))
		// 检查是否有 editor 类型的字段
		hasEditor := false
		for _, field := range formFields {
			if field.FormType == "editor" && field.ShowInForm {
				hasEditor = true
				break
			}
		}

		// 检查是否有 markdown 类型的字段
		hasMarkdown := false
		for _, field := range formFields {
			if field.FormType == "markdown" && field.ShowInForm {
				hasMarkdown = true
				break
			}
		}

		// 检查是否有 image-upload 类型的字段
		hasImageUpload := false
		for _, field := range formFields {
			if field.FormType == "image-upload" && field.ShowInForm {
				hasImageUpload = true
				break
			}
		}

		return struct {
			ModelName      string
			ModuleName     string
			ModuleNameK    string
			FormFields     []TemplateFieldConfig
			HasCreate      bool
			HasEdit        bool
			HasEditor      bool
			HasMarkdown    bool
			HasImageUpload bool
		}{
			ModelName:      toPascalCase(moduleName),
			ModuleName:     moduleName,
			ModuleNameK:    toKebabCase(moduleName),
			FormFields:     formFields,
			HasCreate:      hasCreate,
			HasEdit:        hasEdit,
			HasEditor:      hasEditor,
			HasMarkdown:    hasMarkdown,
			HasImageUpload: hasImageUpload,
		}
	default:
		return struct {
			ModuleName        string
			TableName         string
			FormFields        []TemplateFieldConfig
			RequestCreateName string
			RequestUpdateName string
		}{
			ModuleName:        moduleName,
			TableName:         tableName,
			FormFields:        templateFields,
			RequestCreateName: toPascalCase(moduleName) + "Create",
			RequestUpdateName: toPascalCase(moduleName) + "Update",
		}
	}
}

type TemplateFieldConfig struct {
	Name            string
	Label           string
	FieldName       string
	JsonName        string
	GoType          string
	DBType          string
	Validators      []string
	Required        bool
	MigrationMethod string
	PascalName      string
	Comment         string
	FormType        string
	SearchType      string
	SearchUIType    string
	Dictionary      string
	ApiUrl          string
	Sortable        bool
	Searchable      bool
	ShowInList      bool
	ShowInForm      bool
	ShowInDetail    bool
	Relation        *TemplateRelationConfig
	IsTree          bool // 是否为树形数据（用于自定义API）
	Precision       int
	Scale           int
}

type TemplateRelationConfig struct {
	*RelationConfig
	Name      string // PascalCase 关联名 (如: Admin)
	JsonName  string // camelCase 关联名 (如: admin)
	ModelName string // 关联表对应的模型名称 (如: Admin)
}

func (s *CodeGeneratorServiceImpl) convertFieldsToTemplateFields(fields []FieldConfig) []TemplateFieldConfig {
	fieldTypes := s.GetFieldTypes()
	fieldTypeMap := make(map[string]FieldType)
	for _, ft := range fieldTypes {
		fieldTypeMap[ft.Value] = ft
	}

	templateFields := make([]TemplateFieldConfig, len(fields))
	for i, field := range fields {
		fieldType, exists := fieldTypeMap[field.DBType]
		if !exists {
			fieldType = fieldTypes[0]
		}

		var relation *TemplateRelationConfig
		if field.Relation != nil {
			var relationName string

			// 1. 如果有别名，直接使用别名
			if field.Relation.Alias != "" {
				relationName = toPascalCase(field.Relation.Alias)
			} else {
				// 2. 否则优先尝试从字段名推导 (如 AdminID -> Admin)
				relationName = toPascalCase(field.Name)
				if strings.HasSuffix(relationName, "ID") {
					relationName = strings.TrimSuffix(relationName, "ID")
				} else if strings.HasSuffix(relationName, "Id") {
					relationName = strings.TrimSuffix(relationName, "Id")
				} else {
					// 3. 最后使用表名
					relationName = toPascalCase(field.Relation.Table)
					// 如果是 belongsTo 或 hasOne，尝试转为单数
					if (field.Relation.RelationType == "belongsTo" || field.Relation.RelationType == "hasOne") && strings.HasSuffix(relationName, "s") {
						relationName = relationName[:len(relationName)-1]
					}
				}
			}

			// 计算模型名称 (始终基于表名)
			modelName := toPascalCase(field.Relation.Table)
			// 如果是 belongsTo 或 hasOne，尝试转为单数
			if (field.Relation.RelationType == "belongsTo" || field.Relation.RelationType == "hasOne") && strings.HasSuffix(modelName, "s") {
				modelName = modelName[:len(modelName)-1]
			}

			// 计算 JSON 名称 (首字母小写)
			jsonName := strings.ToLower(relationName[:1]) + relationName[1:]

			relation = &TemplateRelationConfig{
				RelationConfig: field.Relation,
				Name:           relationName,
				JsonName:       jsonName,
				ModelName:      modelName,
			}
		}

		dictionary := resolveFieldDictionary(field.Name, field.Dictionary, field.FormType)

		templateFields[i] = TemplateFieldConfig{
			Name:         field.Name,
			Label:        field.Label,
			FieldName:    toPascalCase(field.Name),
			JsonName:     field.Name,
			GoType:       fieldType.GoType,
			DBType:       field.DBType,
			Validators:   field.Validators,
			Required:     field.Required,
			PascalName:   toPascalCase(field.Name),
			Comment:      field.Label, // Use Label as Comment
			FormType:     field.FormType,
			SearchType:   getSearchType(field.SearchType),
			SearchUIType: getSearchUIType(field.SearchUIType, field.DBType, field.Name),
			Dictionary:   dictionary,
			ApiUrl:       getApiUrl(dictionary, field.ApiUrl),
			Sortable:     field.Sortable,
			Searchable:   field.Searchable,
			ShowInList:   field.ShowInList,
			ShowInForm:   field.ShowInForm,
			ShowInDetail: field.ShowInDetail,
			Relation:     relation,
			IsTree:       field.Relation != nil && field.Relation.IsTree, // 从 relation 中读取 is_tree
			Precision:    field.Precision,
			Scale:        field.Scale,
		}
	}

	return templateFields
}

func (s *CodeGeneratorServiceImpl) generateModel(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/model.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read model template: %w", err)
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	data := struct {
		ModelName  string
		TableName  string
		Fields     []TemplateFieldConfig
		IsTreeList bool
	}{
		ModelName:  toPascalCase(moduleName),
		TableName:  tableName,
		Fields:     templateFields,
		IsTreeList: optionEnabled(options, "is_tree_list", false),
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("app/models/%s.go", toSnakeCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateController(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/controller.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read controller template: %w", err)
	}

	// Default options
	hasCreate := true
	hasEdit := true
	hasDelete := true
	hasExport := false
	exportAsync := false
	hasImport := false
	importAsync := false

	if options != nil {
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
		if val, ok := options["export_async"]; ok {
			exportAsync = val
		}
		if val, ok := options["has_import"]; ok {
			hasImport = val
		}
		if val, ok := options["import_async"]; ok {
			importAsync = val
		}
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	var searchableFields []TemplateFieldConfig
	for _, field := range templateFields {
		if field.Searchable {
			searchableFields = append(searchableFields, field)
		}
	}
	data := struct {
		ControllerName    string
		ServiceName       string
		ModelName         string
		ModuleName        string
		TableName         string
		ListFields        []TemplateFieldConfig
		SearchableFields  []TemplateFieldConfig
		RequestCreateName string
		RequestUpdateName string
		HasCreate         bool
		HasEdit           bool
		HasDelete         bool
		HasExport         bool
		ExportAsync       bool
		HasImport         bool
		ImportAsync       bool
		IsTreeList        bool
	}{
		ControllerName:    toPascalCase(moduleName) + "Controller",
		ServiceName:       toPascalCase(moduleName) + "Service",
		ModelName:         toPascalCase(moduleName),
		ModuleName:        moduleName,
		TableName:         tableName,
		ListFields:        templateFields,
		SearchableFields:  searchableFields,
		RequestCreateName: toPascalCase(moduleName) + "Create",
		RequestUpdateName: toPascalCase(moduleName) + "Update",
		HasCreate:         hasCreate,
		HasEdit:           hasEdit,
		HasDelete:         hasDelete,
		HasExport:         hasExport,
		ExportAsync:       exportAsync,
		HasImport:         hasImport,
		ImportAsync:       importAsync,
		IsTreeList:        optionEnabled(options, "is_tree_list", false),
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("app/http/controllers/admin/%s_controller.go", toSnakeCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateService(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/service.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read service template: %w", err)
	}

	// Default options
	hasCreate := true
	hasEdit := true
	hasDelete := true
	hasExport := false
	exportAsync := false
	hasImport := false
	importAsync := false

	if options != nil {
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
		if val, ok := options["export_async"]; ok {
			exportAsync = val
		}
		if val, ok := options["has_import"]; ok {
			hasImport = val
		}
		if val, ok := options["import_async"]; ok {
			importAsync = val
		}
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	var searchableFields []TemplateFieldConfig
	for _, field := range templateFields {
		if field.Searchable {
			searchableFields = append(searchableFields, field)
		}
	}
	data := struct {
		ServiceName       string
		ModelName         string
		ModuleName        string
		SearchableFields  []TemplateFieldConfig
		RequestCreateName string
		RequestUpdateName string
		FormFields        []TemplateFieldConfig
		HasCreate         bool
		HasEdit           bool
		HasDelete         bool
		HasExport         bool
		ExportAsync       bool
		HasImport         bool
		ImportAsync       bool
		IsTreeList        bool
		ParentIDFieldName string
	}{
		ServiceName:       toPascalCase(moduleName) + "Service",
		ModelName:         toPascalCase(moduleName),
		ModuleName:        moduleName,
		SearchableFields:  searchableFields,
		RequestCreateName: toPascalCase(moduleName) + "Create",
		RequestUpdateName: toPascalCase(moduleName) + "Update",
		FormFields:        templateFields,
		HasCreate:         hasCreate,
		HasEdit:           hasEdit,
		HasDelete:         hasDelete,
		HasExport:         hasExport,
		ExportAsync:       exportAsync,
		HasImport:         hasImport,
		ImportAsync:       importAsync,
		IsTreeList:        optionEnabled(options, "is_tree_list", false),
		ParentIDFieldName: resolveParentIDFieldName(templateFields),
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("app/services/%s_service.go", toSnakeCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateExportJob(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/export_job.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read export_job template: %w", err)
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	data := struct {
		ModelName  string
		ModuleName string
		ListFields []TemplateFieldConfig
	}{
		ModelName:  toPascalCase(moduleName),
		ModuleName: moduleName,
		ListFields: templateFields,
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("app/jobs/export_%ss.go", toSnakeCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateImportJob(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/import_job.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read import_job template: %w", err)
	}

	data := struct {
		ModelName  string
		ModuleName string
	}{
		ModelName:  toPascalCase(moduleName),
		ModuleName: moduleName,
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("app/jobs/import_%ss.go", toSnakeCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateRequestCreate(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/request_create.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read request_create template: %w", err)
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	data := struct {
		ModuleName        string
		TableName         string
		FormFields        []TemplateFieldConfig
		RequestCreateName string
	}{
		ModuleName:        moduleName,
		TableName:         tableName,
		FormFields:        templateFields,
		RequestCreateName: toPascalCase(moduleName) + "Create",
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("app/http/requests/admin/%s_create.go", toSnakeCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateRequestUpdate(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/request_update.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read request_update template: %w", err)
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	data := struct {
		ModuleName        string
		TableName         string
		FormFields        []TemplateFieldConfig
		RequestUpdateName string
	}{
		ModuleName:        moduleName,
		TableName:         tableName,
		FormFields:        templateFields,
		RequestUpdateName: toPascalCase(moduleName) + "Update",
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("app/http/requests/admin/%s_update.go", toSnakeCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateMigration(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/migration.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read migration template: %w", err)
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	for i := range templateFields {
		templateFields[i].MigrationMethod = getMigrationMethod(templateFields[i].DBType)
	}

	timestamp := time.Now().Format("20060102150405")
	data := struct {
		ModelName string
		TableName string
		Fields    []TemplateFieldConfig
		Timestamp string
	}{
		ModelName: toPascalCase(moduleName),
		TableName: tableName,
		Fields:    templateFields,
		Timestamp: timestamp,
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("database/migrations/%s_create_%s_table.go", timestamp, toSnakeCase(tableName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateFrontendAPI(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/api.js.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read api template: %w", err)
	}

	// Default options
	hasCreate := true
	hasEdit := true
	hasDelete := true
	hasExport := false
	hasImport := false
	importAsync := false

	if options != nil {
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
		if val, ok := options["import_async"]; ok {
			importAsync = val
		}
	}

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
		Path:    fmt.Sprintf("html/src/api/%s.js", toKebabCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) buildListPageTemplateData(
	moduleName string,
	templateFields []TemplateFieldConfig,
	hasCreate, hasEdit, hasDelete, hasExport, exportAsync, hasImport, importAsync, enableBatchActions, showToolbar, isTreeList bool,
) any {
	formFields := applyTreeListFieldFlags(templateFields, isTreeList)

	var searchableFields []TemplateFieldConfig
	var listFields []TemplateFieldConfig
	for _, field := range formFields {
		if field.Searchable {
			searchableFields = append(searchableFields, field)
		}
		listFields = append(listFields, field)
	}

	return struct {
		ModelName          string
		ModuleName         string
		ModuleNameCamel    string
		ModuleNameK        string
		SearchableFields   []TemplateFieldConfig
		ListFields         []TemplateFieldConfig
		FormFields         []TemplateFieldConfig
		HasCreate          bool
		HasEdit            bool
		HasDelete          bool
		HasExport          bool
		ExportAsync        bool
		HasImport          bool
		ImportAsync        bool
		EnableBatchActions bool
		ShowToolbar        bool
		IsTreeList         bool
		TreeLabelField     string
		HasEditor           bool
		HasMarkdown         bool
		HasImageUpload      bool
		HasRichTextInList   bool
		HasListStatusSwitch bool
		HasFormSwitch       bool
		HasFormSelect       bool
		HasFormRadio        bool
		HasFormCheckbox     bool
		HasFormNumber       bool
		HasFormDatePicker   bool
	}{
		ModelName:           toPascalCase(moduleName),
		ModuleName:          moduleName,
		ModuleNameCamel:     toCamelCase(moduleName),
		ModuleNameK:         toKebabCase(moduleName),
		SearchableFields:    searchableFields,
		ListFields:          listFields,
		FormFields:          formFields,
		HasCreate:           hasCreate,
		HasEdit:             hasEdit,
		HasDelete:           hasDelete,
		HasExport:           hasExport,
		ExportAsync:         exportAsync,
		HasImport:           hasImport,
		ImportAsync:         importAsync,
		EnableBatchActions:  enableBatchActions,
		ShowToolbar:         showToolbar,
		IsTreeList:          isTreeList,
		TreeLabelField:      resolveTreeLabelField(listFields),
		HasEditor:           hasFieldFormType(formFields, "editor"),
		HasMarkdown:         hasFieldFormType(formFields, "markdown"),
		HasImageUpload:      hasFieldFormType(formFields, "image-upload"),
		HasRichTextInList:   hasListFieldFormType(listFields, "editor", "markdown"),
		HasListStatusSwitch: hasListStatusSwitch(listFields),
		HasFormSwitch:       hasFieldFormType(formFields, "switch"),
		HasFormSelect:       hasFieldFormType(formFields, "select"),
		HasFormRadio:        hasFieldFormType(formFields, "radio"),
		HasFormCheckbox:     hasFieldFormType(formFields, "checkbox"),
		HasFormNumber:       hasFieldFormType(formFields, "number"),
		HasFormDatePicker:   hasFieldFormType(formFields, "date-picker") || hasFieldFormType(formFields, "datetime-picker"),
	}
}

func hasListStatusSwitch(fields []TemplateFieldConfig) bool {
	for _, field := range fields {
		if field.ShowInList && field.Name == "status" && field.FormType == "switch" {
			return true
		}
	}
	return false
}

func hasFieldFormType(fields []TemplateFieldConfig, formType string) bool {
	for _, field := range fields {
		if field.ShowInForm && field.FormType == formType {
			return true
		}
	}
	return false
}

func hasListFieldFormType(fields []TemplateFieldConfig, formTypes ...string) bool {
	typeSet := map[string]bool{}
	for _, formType := range formTypes {
		typeSet[formType] = true
	}
	for _, field := range fields {
		if field.ShowInList && typeSet[field.FormType] {
			return true
		}
	}
	return false
}

func applyTreeListFieldFlags(fields []TemplateFieldConfig, isTreeList bool) []TemplateFieldConfig {
	if !isTreeList {
		return fields
	}

	result := make([]TemplateFieldConfig, len(fields))
	copy(result, fields)
	for i := range result {
		if result[i].Name != "parent_id" {
			continue
		}
		result[i].IsTree = true
		if result[i].FormType == "input" || result[i].FormType == "number" {
			result[i].FormType = "select"
		}
	}
	return result
}

func resolveTreeLabelField(fields []TemplateFieldConfig) string {
	priority := []string{"name", "title", "label"}
	for _, name := range priority {
		for _, field := range fields {
			if field.ShowInList && field.Name == name {
				return name
			}
		}
	}
	for _, field := range fields {
		if !field.ShowInList {
			continue
		}
		switch field.Name {
		case "id", "parent_id", "status", "sort", "created_at", "updated_at":
			continue
		}
		return field.Name
	}
	return "name"
}

func resolveParentIDFieldName(fields []TemplateFieldConfig) string {
	for _, field := range fields {
		if field.Name == "parent_id" {
			if field.FieldName != "" {
				return field.FieldName
			}
			return "ParentID"
		}
	}
	return "ParentID"
}

func (s *CodeGeneratorServiceImpl) generateFrontendListPageConfig(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateName := s.getListPageConfigTemplateName(options)
	templateContent, err := templates.ReadFile(templateName)
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read list_page_config template: %w", err)
	}

	hasCreate := true
	hasEdit := true
	hasDelete := true
	hasExport := false
	exportAsync := false
	hasImport := false
	importAsync := false
	enableBatchActions := false
	showToolbar := true

	if options != nil {
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
		if val, ok := options["export_async"]; ok {
			exportAsync = val
		}
		if val, ok := options["has_import"]; ok {
			hasImport = val
		}
		if val, ok := options["import_async"]; ok {
			importAsync = val
		}
		if val, ok := options["enable_batch_actions"]; ok {
			enableBatchActions = val
		}
		if val, ok := options["show_toolbar"]; ok {
			showToolbar = val
		}
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	data := s.buildListPageTemplateData(moduleName, templateFields, hasCreate, hasEdit, hasDelete, hasExport, exportAsync, hasImport, importAsync, enableBatchActions, showToolbar, optionEnabled(options, "is_tree_list", false))

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("html/src/views/%s/%s.config.js", toKebabCase(moduleName), toCamelCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateFrontendListPage(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateName := s.getListPageTemplateName(options)
	templateContent, err := templates.ReadFile(templateName)
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read list_page template: %w", err)
	}

	hasCreate := true
	hasEdit := true
	hasDelete := true
	hasExport := false
	exportAsync := false
	hasImport := false
	importAsync := false
	enableBatchActions := false
	showToolbar := true

	if options != nil {
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
		if val, ok := options["export_async"]; ok {
			exportAsync = val
		}
		if val, ok := options["has_import"]; ok {
			hasImport = val
		}
		if val, ok := options["import_async"]; ok {
			importAsync = val
		}
		if val, ok := options["enable_batch_actions"]; ok {
			enableBatchActions = val
		}
		if val, ok := options["show_toolbar"]; ok {
			showToolbar = val
		}
	}

	templateFields := s.convertFieldsToTemplateFields(fields)
	data := s.buildListPageTemplateData(moduleName, templateFields, hasCreate, hasEdit, hasDelete, hasExport, exportAsync, hasImport, importAsync, enableBatchActions, showToolbar, optionEnabled(options, "is_tree_list", false))

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("html/src/views/%s/%sList.vue", toKebabCase(moduleName), toPascalCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) generateFrontendFormPage(moduleName, tableName string, fields []FieldConfig, options map[string]bool) (GeneratedFile, error) {
	templateContent, err := templates.ReadFile("templates/form_page.vue.tpl")
	if err != nil {
		return GeneratedFile{}, fmt.Errorf("failed to read form_page template: %w", err)
	}

	// Default options
	hasCreate := true
	hasEdit := true

	if options != nil {
		if val, ok := options["has_create"]; ok {
			hasCreate = val
		}
		if val, ok := options["has_edit"]; ok {
			hasEdit = val
		}
	}

	templateFields := s.convertFieldsToTemplateFields(fields)

	// 检查是否有 editor 类型的字段
	hasEditor := false
	for _, field := range templateFields {
		if field.FormType == "editor" && field.ShowInForm {
			hasEditor = true
			break
		}
	}

	// 检查是否有 markdown 类型的字段
	hasMarkdown := false
	for _, field := range templateFields {
		if field.FormType == "markdown" && field.ShowInForm {
			hasMarkdown = true
			break
		}
	}

	// 检查是否有 image-upload 类型的字段
	hasImageUpload := false
	for _, field := range templateFields {
		if field.FormType == "image-upload" && field.ShowInForm {
			hasImageUpload = true
			break
		}
	}

	data := struct {
		ModelName      string
		ModuleName     string
		ModuleNameK    string
		FormFields     []TemplateFieldConfig
		HasCreate      bool
		HasEdit        bool
		HasEditor      bool
		HasMarkdown    bool
		HasImageUpload bool
	}{
		ModelName:      toPascalCase(moduleName),
		ModuleName:     moduleName,
		ModuleNameK:    toKebabCase(moduleName),
		FormFields:     templateFields,
		HasCreate:      hasCreate,
		HasEdit:        hasEdit,
		HasEditor:      hasEditor,
		HasMarkdown:    hasMarkdown,
		HasImageUpload: hasImageUpload,
	}

	content, err := s.executeTemplate(string(templateContent), data)
	if err != nil {
		return GeneratedFile{}, err
	}

	return GeneratedFile{
		Path:    fmt.Sprintf("html/src/views/%s/%sForm.vue", toKebabCase(moduleName), toPascalCase(moduleName)),
		Content: content,
	}, nil
}

func (s *CodeGeneratorServiceImpl) executeTemplate(templateContent string, data any) (string, error) {
	tmpl, err := template.New("code").Delims("<<", ">>").Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var builder strings.Builder
	if err := tmpl.Execute(&builder, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return builder.String(), nil
}

func toSnakeCase(s string) string {
	result := ""
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result += "_"
		}
		result += string(r)
	}
	return strings.ToLower(result)
}

func toPascalCase(s string) string {
	s = strings.ReplaceAll(s, "-", "_")
	words := strings.Split(s, "_")
	for i := range words {
		if len(words[i]) > 0 {
			words[i] = strings.ToUpper(words[i][:1]) + strings.ToLower(words[i][1:])
		}
	}
	return strings.Join(words, "")
}

func toCamelCase(s string) string {
	pascal := toPascalCase(s)
	if pascal == "" {
		return ""
	}
	return strings.ToLower(pascal[:1]) + pascal[1:]
}

func toKebabCase(s string) string {
	return strings.ReplaceAll(toSnakeCase(strings.ReplaceAll(s, "-", "_")), "_", "-")
}

func getMigrationMethod(dbType string) string {
	switch dbType {
	case "string":
		return "String"
	case "text":
		return "Text"
	case "integer":
		return "Integer"
	case "bigInteger":
		return "BigInteger"
	case "unsignedBigInteger":
		return "UnsignedBigInteger"
	case "unsignedTinyInteger":
		return "UnsignedTinyInteger"
	case "decimal":
		return "Decimal"
	case "boolean":
		return "Boolean"
	case "date":
		return "Date"
	case "datetime":
		return "DateTime"
	case "timestamp":
		return "Timestamp"
	case "json":
		return "Json"
	default:
		return "String"
	}
}

func getFormType(dbType string) string {
	switch dbType {
	case "string":
		return "input"
	case "text":
		return "textarea"
	case "integer":
		return "input"
	case "decimal":
		return "input"
	case "boolean":
		return "switch"
	case "date":
		return "date-picker"
	case "datetime":
		return "datetime-picker"
	case "timestamp":
		return "datetime-picker"
	case "json":
		return "textarea"
	default:
		return "input"
	}
}

func getSearchType(searchType string) string {
	if searchType == "" {
		return "like"
	}
	return searchType
}

func getSearchUIType(searchUIType string, dbType string, fieldName string) string {
	if searchUIType != "" {
		return searchUIType
	}

	// 时间后缀字段默认走范围搜索（created_at / updated_at / *_at）
	if fieldName == "created_at" || fieldName == "updated_at" || strings.HasSuffix(fieldName, "_at") {
		if dbType == "date" {
			return "daterange"
		}
		return "datetimerange"
	}

	switch dbType {
	case "date":
		return "date"
	case "datetime", "timestamp":
		return "datetime"
	case "boolean":
		return "select"
	default:
		return "input"
	}
}

func getApiUrl(dictionary string, apiUrl string) string {
	if apiUrl != "" {
		return apiUrl
	}
	if dictionary != "" {
		return "/options?type=dictionary&dictionary_type=" + dictionary
	}
	return ""
}

func resolveFieldDictionary(name, dictionary, formType string) string {
	if dictionary != "" {
		return dictionary
	}
	if name == "status" && (formType == "select" || formType == "radio" || formType == "switch") {
		return "status"
	}
	return ""
}

// GenerateWithAI 使用 AI 辅助生成代码配置
func (s *CodeGeneratorServiceImpl) GenerateWithAI(ctx context.Context, userDescription string) (*AIGeneratedConfig, error) {
	// 读取 AI 模块开发提示词（正文在 VitePress website/docs）
	promptFile, err := readAIModulePromptFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read AI module prompt: %w", err)
	}

	// 构建系统提示词
	systemPrompt := `你是一位经验丰富的全栈开发工程师，精通 Go 语言（Goravel 框架）和 Vue 3（Element Plus）开发。
你的任务是根据用户的自然语言描述，生成符合项目规范的模块配置。

请仔细阅读以下项目规范和开发指南，然后根据用户需求生成 JSON 格式的配置。`

	// 构建用户提示词
	userPrompt := fmt.Sprintf(`%s

## 用户需求

%s

## 任务

请根据以上规范和用户需求，生成模块配置。配置必须是有效的 JSON 格式，包含以下字段：
- module_name: 模块名称（小写，如 "guestbook", "order"）
- table_name: 表名（小写复数，如 "guestbooks", "orders"）
- fields: 字段配置数组

每个字段配置应包含：
- name: 字段名（小写，如 "title", "content", "status"）
- label: 字段标签（中文，如 "标题", "内容", "状态"）
- db_type: 数据库类型（如 "string", "text", "integer", "bigInteger", "decimal", "boolean", "date", "datetime", "json"）
- go_type: Go 类型（如 "string", "int", "int64", "float64", "bool", "time.Time"）
- required: 是否必填（布尔值）
- searchable: 是否可搜索（布尔值，默认 true）
- sortable: 是否可排序（布尔值，默认 false）
- search_type: 搜索类型（如 "like", "=", ">", "<", "in"）
- search_ui_type: 搜索 UI 类型（如 "input", "select", "date", "datetime", "daterange"）
- form_type: 表单类型（如 "input", "textarea", "select", "switch", "date-picker", "datetime-picker", "image-upload", "file-upload", "editor"）
- show_in_list: 是否在列表中显示（布尔值，默认 true）
- show_in_form: 是否在表单中显示（布尔值，默认 true）
- show_in_detail: 是否在详情中显示（布尔值，默认 true）

请根据用户描述智能推断字段类型和配置。例如：
- 如果提到"状态"，通常是 integer 或 boolean 类型，form_type 为 "select" 或 "switch"
- 如果提到"图片"、"头像"、"照片"，form_type 应为 "image-upload"
- 如果提到"文件"，form_type 应为 "file-upload"
- 如果提到"内容"、"描述"、"详情"且是长文本，db_type 应为 "text"，form_type 可为 "editor" 或 "textarea"
- 如果提到"时间"、"日期"，db_type 应为 "datetime" 或 "date"，form_type 为 "date-picker" 或 "datetime-picker"
- 如果提到"金额"、"价格"，db_type 应为 "decimal"，go_type 为 "float64"

请只返回 JSON 配置，不要包含其他说明文字。`, string(promptFile), userDescription)

	// 调用 AI Service（失败时自动重试一次）
	aiService := NewAIService(s.ctx)
	var aiResponse string
	for attempt := 1; attempt <= 2; attempt++ {
		aiResponse, err = aiService.Complete(ctx, userPrompt, systemPrompt)
		if err == nil {
			break
		}
		if attempt == 2 {
			return nil, fmt.Errorf("AI completion failed: %w", err)
		}
	}

	// 尝试提取 JSON（AI 可能返回带 markdown 代码块的 JSON）
	jsonStr := strings.TrimSpace(aiResponse)

	// 处理 ```json 代码块
	if strings.Contains(jsonStr, "```json") {
		start := strings.Index(jsonStr, "```json")
		if start >= 0 {
			// 从 ```json 之后开始查找结束的 ```
			codeStart := start + 7 // ```json 的长度是 7
			end := strings.Index(jsonStr[codeStart:], "```")
			if end > 0 {
				jsonStr = strings.TrimSpace(jsonStr[codeStart : codeStart+end])
			}
		}
	} else if strings.Contains(jsonStr, "```") {
		// 处理普通的 ``` 代码块
		start := strings.Index(jsonStr, "```")
		if start >= 0 {
			codeStart := start + 3 // ``` 的长度是 3
			end := strings.Index(jsonStr[codeStart:], "```")
			if end > 0 {
				jsonStr = strings.TrimSpace(jsonStr[codeStart : codeStart+end])
			}
		}
	}

	// 尝试查找 JSON 对象（以 { 开头，以 } 结尾）
	if !strings.HasPrefix(jsonStr, "{") {
		start := strings.Index(jsonStr, "{")
		if start >= 0 {
			// 找到最后一个 }
			lastBrace := strings.LastIndex(jsonStr, "}")
			if lastBrace > start {
				jsonStr = strings.TrimSpace(jsonStr[start : lastBrace+1])
			}
		}
	}

	// 解析 JSON
	var config AIGeneratedConfig
	if err := json.Unmarshal([]byte(jsonStr), &config); err != nil {
		return nil, fmt.Errorf("AI 返回的配置格式不正确，无法解析 JSON。请检查 AI 响应格式或重试。错误详情: %v", err)
	}

	// 验证配置
	if config.ModuleName == "" {
		return nil, fmt.Errorf("module_name is required")
	}
	if config.TableName == "" {
		return nil, fmt.Errorf("table_name is required")
	}
	if len(config.Fields) == 0 {
		return nil, fmt.Errorf("fields cannot be empty")
	}
	config.Fields = ensureAISystemFields(config.Fields)

	// 为字段设置默认值
	fieldTypes := s.GetFieldTypes()
	fieldTypeMap := make(map[string]FieldType)
	for _, ft := range fieldTypes {
		fieldTypeMap[ft.Value] = ft
	}

	for i := range config.Fields {
		field := &config.Fields[i]

		// 设置默认值
		if field.Label == "" {
			field.Label = field.Name
		}
		if field.DBType == "" {
			field.DBType = "string"
		}
		if fieldType, ok := fieldTypeMap[field.DBType]; ok {
			if field.GoType == "" {
				field.GoType = fieldType.GoType
			}
		} else {
			if field.GoType == "" {
				field.GoType = "string"
			}
		}
		if field.SearchType == "" {
			if field.DBType == "string" || field.DBType == "text" {
				field.SearchType = "like"
			} else {
				field.SearchType = "="
			}
		}
		if field.SearchUIType == "" {
			field.SearchUIType = getSearchUIType("", field.DBType, field.Name)
		}
		if field.FormType == "" {
			field.FormType = getFormType(field.DBType)
		}
		// 为 decimal 类型设置默认精度和标度
		if field.DBType == "decimal" {
			if field.Precision == 0 {
				field.Precision = 8 // 默认精度
			}
			if field.Scale == 0 {
				field.Scale = 2 // 默认标度
			}
		}
		if !field.Searchable {
			field.Searchable = true
		}
		if !field.ShowInList {
			field.ShowInList = true
		}
		if !field.ShowInForm {
			field.ShowInForm = true
		}
		if !field.ShowInDetail {
			field.ShowInDetail = true
		}
		// 系统时间字段在表单中默认不可编辑。
		if field.Name == "created_at" || field.Name == "updated_at" {
			field.ShowInForm = false
			if field.SearchUIType == "" {
				field.SearchUIType = "datetimerange"
			}
		}
		// 确保关联表对象结构完整
		if field.Relation != nil {
			if field.Relation.RelationType == "" {
				field.Relation.RelationType = "belongsTo"
			}
			if field.Relation.Table == "" {
				field.Relation = nil // 如果关联表为空，则清除关联
			}
		}
	}

	return &config, nil
}

// readAIModulePromptFile loads the AI codegen system prompt from the VitePress docs tree.
func readAIModulePromptFile() ([]byte, error) {
	candidates := []string{
		"website/docs/advanced/ai-module.md",
		filepath.Join("website", "docs", "advanced", "ai-module.md"),
	}
	var lastErr error
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}
	return nil, lastErr
}
