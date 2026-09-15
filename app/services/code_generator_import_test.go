package services

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateImportJobAndController(t *testing.T) {
	s := &CodeGeneratorServiceImpl{}
	fields := []FieldConfig{
		{Name: "name", GoType: "string", DBType: "string", Label: "Name", ShowInList: true, ShowInForm: true, FormType: "input"},
		{Name: "status", GoType: "bool", DBType: "boolean", Label: "Status", ShowInList: true, ShowInForm: true, FormType: "select"},
	}
	opts := map[string]bool{
		"has_import":   true,
		"import_async": true,
		"has_create":   true,
		"has_edit":     true,
		"has_delete":   true,
	}

	files, err := s.Generate("demo", "demos", fields, []string{"controller", "service", "import_job"}, opts)
	require.NoError(t, err)

	byPath := map[string]string{}
	for _, f := range files {
		byPath[f.Path] = f.Content
		assert.NotContains(t, f.Content, "<<.")
	}

	job, ok := byPath["app/jobs/import_demos.go"]
	require.True(t, ok)
	assert.Contains(t, job, "type ImportDemos struct{}")
	assert.Contains(t, job, "ImportFromCSV")

	ctrl, ok := byPath["app/http/controllers/admin/demo_controller.go"]
	require.True(t, ok)
	assert.Contains(t, ctrl, "func (c *DemoController) Import(")
	assert.Contains(t, ctrl, "enqueueDemoImport")
	assert.Contains(t, ctrl, "&jobs.ImportDemos{}")

	svc, ok := byPath["app/services/demo_service.go"]
	require.True(t, ok)
	assert.Contains(t, svc, "ImportFromCSV(csvContent string)")
	assert.True(t, strings.Contains(svc, "headerMap"))
}

func TestGenerateImportBelongsToUsesUint(t *testing.T) {
	s := &CodeGeneratorServiceImpl{}
	fields := []FieldConfig{
		{
			Name: "admin_id", Label: "Admin", DBType: "bigInteger", GoType: "int64",
			ShowInList: true, ShowInForm: true, FormType: "select",
			Relation: &RelationConfig{
				Table: "admins", ForeignKey: "admin_id", DisplayField: "name", RelationType: "belongsTo",
			},
		},
		{Name: "title", GoType: "string", DBType: "string", Label: "Title", ShowInList: true, ShowInForm: true, FormType: "input"},
	}
	opts := map[string]bool{"has_import": true, "has_create": true}

	files, err := s.Generate("article", "articles", fields, []string{"service"}, opts)
	require.NoError(t, err)

	var svc string
	for _, f := range files {
		if strings.HasSuffix(f.Path, "article_service.go") {
			svc = f.Content
			break
		}
	}
	require.NotEmpty(t, svc)
	assert.Contains(t, svc, "item.AdminId = cast.ToUint(val)")
	assert.NotContains(t, svc, "item.AdminId = cast.ToInt64(val)")
}

func TestGenerateImportSyncSkipsJob(t *testing.T) {
	s := &CodeGeneratorServiceImpl{}
	fields := []FieldConfig{
		{Name: "name", GoType: "string", DBType: "string", Label: "Name", ShowInList: true, ShowInForm: true, FormType: "input"},
	}
	opts := map[string]bool{
		"has_import":   true,
		"import_async": false,
	}
	files, err := s.Generate("demo", "demos", fields, []string{"controller", "import_job"}, opts)
	require.NoError(t, err)
	for _, f := range files {
		assert.False(t, strings.HasPrefix(f.Path, "app/jobs/import_"))
	}
}
