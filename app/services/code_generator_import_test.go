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
