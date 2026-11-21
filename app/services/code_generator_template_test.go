package services

import (
	"context"
	"strings"
	"testing"
)

func TestReactListPageTemplatePreview(t *testing.T) {
	s := NewCodeGeneratorService(context.Background())
	fields := []FieldConfig{
		{
			Name: "admin_id", Label: "Admin", DBType: "bigInteger", GoType: "int64",
			ShowInList: true, ShowInForm: true, FormType: "select",
			Relation: &RelationConfig{
				Table: "admins", ForeignKey: "admin_id", DisplayField: "name", RelationType: "belongsTo",
			},
		},
		{Name: "title", Label: "Title", ShowInList: true, ShowInForm: true, FormType: "input"},
		{Name: "content", Label: "Content", ShowInList: true, ShowInForm: true, FormType: "editor"},
		{Name: "status", Label: "Status", ShowInList: true, ShowInForm: true, FormType: "switch"},
	}

	code, err := s.Preview("article", "articles", fields, "react_list_page", map[string]bool{
		"has_create": true, "has_edit": true, "has_delete": true, "show_toolbar": true,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if !strings.Contains(code, "useListPage<ArticleRow>(") {
		t.Fatalf("expected typed useListPage call, got:\n%s", code[:min(500, len(code))])
	}
	if !strings.Contains(code, "getButtonState('article.update')") {
		t.Fatal("status switch must use root ModuleName via $.ModuleName")
	}
	if strings.Contains(code, "normalizeRows") {
		t.Fatal("react list page must not set normalizeRows; API normalizeListResponse is the boundary")
	}
	if strings.Contains(code, "<<") || strings.Contains(code, ">>>") {
		t.Fatalf("template delimiters leaked into output")
	}
}

func TestReactTreeListPageTemplatePreview_StatusSwitch(t *testing.T) {
	s := NewCodeGeneratorService(context.Background())
	fields := []FieldConfig{
		{Name: "parent_id", Label: "Parent", ShowInList: true, ShowInForm: true, FormType: "number"},
		{Name: "name", Label: "Name", ShowInList: true, ShowInForm: true, FormType: "input"},
		{Name: "status", Label: "Status", ShowInList: true, ShowInForm: true, FormType: "switch"},
	}

	code, err := s.Preview("category", "categories", fields, "react_list_page", map[string]bool{
		"has_create": true, "has_edit": true, "has_delete": true, "is_tree_list": true, "show_toolbar": true,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if !strings.Contains(code, "getButtonState('category.update')") {
		t.Fatal("tree status switch must use root ModuleName via $.ModuleName")
	}
}

func TestVueListPageTemplatePreview_EditorColumn(t *testing.T) {
	s := NewCodeGeneratorService(context.Background())
	fields := []FieldConfig{
		{Name: "title", Label: "Title", ShowInList: true, ShowInForm: true, FormType: "input"},
		{Name: "content", Label: "Content", ShowInList: true, ShowInForm: true, FormType: "editor"},
	}

	listPage, err := s.Preview("article", "articles", fields, "list_page", map[string]bool{
		"has_create": true, "has_edit": true, "has_delete": true, "show_toolbar": true,
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if !strings.Contains(listPage, "extractTextFromMarkdown") {
		t.Fatal("vue list page must strip HTML for editor columns")
	}
	if !strings.Contains(listPage, "#content=") {
		t.Fatal("vue list page must define content slot for editor field")
	}

	config, err := s.Preview("article", "articles", fields, "list_page_config", map[string]bool{
		"has_create": true, "has_edit": true, "has_delete": true, "show_toolbar": true,
	})
	if err != nil {
		t.Fatalf("preview config failed: %v", err)
	}
	if !strings.Contains(config, "slot: 'content'") {
		t.Fatal("vue list config must use slot for editor column")
	}
}

func TestRequestUpdateTemplatePreview_RichTextPrepare(t *testing.T) {
	s := NewCodeGeneratorService(context.Background())
	fields := []FieldConfig{
		{Name: "title", Label: "Title", ShowInList: true, ShowInForm: true, FormType: "input"},
		{Name: "content", Label: "Content", ShowInList: true, ShowInForm: true, FormType: "editor"},
	}

	createCode, err := s.Preview("article", "articles", fields, "request_create", nil)
	if err != nil {
		t.Fatalf("preview request_create failed: %v", err)
	}
	if !strings.Contains(createCode, `PrepareRichTextFieldForValidation(data, "content")`) {
		t.Fatal("request_create must prepare rich text content")
	}

	updateCode, err := s.Preview("article", "articles", fields, "request_update", nil)
	if err != nil {
		t.Fatalf("preview request_update failed: %v", err)
	}
	if !strings.Contains(updateCode, `PrepareRichTextFieldForValidation(data, "content")`) {
		t.Fatal("request_update must prepare rich text content")
	}
	if !strings.Contains(updateCode, "func (r *ArticleUpdate) PrepareForValidation") {
		t.Fatal("request_update must define PrepareForValidation")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Every file type the generator writes must also preview, and a backend preview must be
// byte-for-byte the file Generate produces. The service and migration previews used to
// fail ("can't evaluate field EntitlementRowQuota/Timestamp") because Preview built its
// own template data that had drifted from the generators.
func TestPreviewMatchesGeneratedFiles(t *testing.T) {
	impl := NewCodeGeneratorService(context.Background()).(*CodeGeneratorServiceImpl)
	fields := []FieldConfig{
		{Name: "parent_id", Label: "Parent", GoType: "uint", DBType: "unsignedBigInteger", ShowInList: true, ShowInForm: true, FormType: "number"},
		{Name: "name", Label: "Name", GoType: "string", DBType: "string", ShowInList: true, ShowInForm: true, FormType: "input", Required: true, Searchable: true},
		{Name: "status", Label: "Status", GoType: "uint8", DBType: "unsignedTinyInteger", ShowInList: true, ShowInForm: true, FormType: "switch"},
		{Name: "is_public", Label: "Public", GoType: "bool", DBType: "boolean", ShowInList: true, ShowInForm: true, FormType: "switch"},
	}
	optionSets := map[string]map[string]bool{
		"plain": {"has_create": true, "has_edit": true, "has_delete": true},
		"tree":  {"has_create": true, "has_edit": true, "has_delete": true, "is_tree_list": true},
	}
	backend := map[string]bool{"model": true, "controller": true, "service": true, "request_create": true, "request_update": true, "migration": true}

	for setName, opts := range optionSets {
		for _, gen := range impl.fileGenerators() {
			t.Run(setName+"/"+gen.fileType, func(t *testing.T) {
				preview, err := impl.Preview("probe", "probes", fields, gen.fileType, opts)
				if err != nil {
					t.Fatalf("preview failed: %v", err)
				}
				if strings.TrimSpace(preview) == "" {
					t.Fatal("empty preview")
				}
				if note := previewNotGeneratedNote(gen.fileType, fields, opts); note != "" {
					if preview != note {
						t.Fatalf("a file the generator skips must preview as the note, got:\n%s", preview)
					}
					files, err := impl.Generate("probe", "probes", fields, []string{gen.fileType}, opts)
					if err != nil {
						t.Fatalf("generate failed: %v", err)
					}
					for _, f := range files {
						if strings.HasSuffix(f.Path, "Config.ts") || strings.HasSuffix(f.Path, "FormModal.tsx") {
							t.Fatalf("note says %s is not generated, but Generate wrote %s", gen.fileType, f.Path)
						}
					}
					return
				}
				if !backend[gen.fileType] {
					return
				}
				files, err := impl.Generate("probe", "probes", fields, []string{gen.fileType}, opts)
				if err != nil {
					t.Fatalf("generate failed: %v", err)
				}
				if len(files) != 1 {
					t.Fatalf("expected 1 generated file, got %d", len(files))
				}
				if files[0].Content != preview {
					t.Fatalf("preview differs from the generated %s", files[0].Path)
				}
			})
		}
	}

	service, err := impl.Preview("probe", "probes", fields, "service", optionSets["plain"])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(service, "Status:   req.Status,") {
		t.Error("generated service must copy status from the request so a disabled row stays disabled")
	}
}

func TestPreviewRejectsUnknownFileType(t *testing.T) {
	_, err := NewCodeGeneratorService(context.Background()).Preview("probe", "probes", nil, "nope", nil)
	if err == nil || !strings.Contains(err.Error(), "unknown file type") {
		t.Fatalf("want unknown file type error, got %v", err)
	}
}
