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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
