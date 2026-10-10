package services

import (
	"strings"
	"testing"
)

func TestValidateMasterDetailRequest(t *testing.T) {
	if err := ValidateMasterDetailRequest(nil, "", nil); err != nil {
		t.Fatalf("nil options should pass: %v", err)
	}
	opts := map[string]bool{"is_master_detail": true, "is_tree_list": true}
	if err := ValidateMasterDetailRequest(opts, "order_items", []FieldConfig{{Name: "sku"}}); err == nil || err.Error() != "master_detail_conflicts_tree_list" {
		t.Fatalf("expected tree conflict, got %v", err)
	}
	opts = map[string]bool{"is_master_detail": true}
	if err := ValidateMasterDetailRequest(opts, "", []FieldConfig{{Name: "sku"}}); err == nil || err.Error() != "detail_table_name_required" {
		t.Fatalf("expected detail table required, got %v", err)
	}
	if err := ValidateMasterDetailRequest(opts, "order_items", nil); err == nil || err.Error() != "detail_fields_required" {
		t.Fatalf("expected detail fields required, got %v", err)
	}
}

func TestGenerateMasterDetailModule(t *testing.T) {
	s := (&CodeGeneratorServiceImpl{}).WithMasterDetail("order_items", []FieldConfig{
		{Name: "sku", GoType: "string", DBType: "string", Label: "SKU", ShowInForm: true, FormType: "input"},
		{Name: "qty", GoType: "int", DBType: "integer", Label: "Qty", ShowInForm: true, FormType: "number"},
		{Name: "unit_price", GoType: "float64", DBType: "decimal", Label: "Price", ShowInForm: true, FormType: "number", Precision: 10, Scale: 2},
	}).(*CodeGeneratorServiceImpl)

	fields := []FieldConfig{
		{Name: "order_no", GoType: "string", DBType: "string", Label: "Order No", ShowInList: true, ShowInForm: true, FormType: "input"},
	}
	options := map[string]bool{
		"is_master_detail": true,
		"has_create":       true,
		"has_edit":         true,
		"has_delete":       true,
	}

	model, err := s.generateModel("order", "orders", fields, options)
	if err != nil {
		t.Fatalf("generate model: %v", err)
	}
	if !strings.Contains(model.Content, "Details []OrderItem") {
		t.Fatal("master model should embed Details []OrderItem")
	}
	if !strings.Contains(model.Content, `json:"details"`) {
		t.Fatal("master model Details should use details json tag")
	}

	service, err := s.generateService("order", "orders", fields, options)
	if err != nil {
		t.Fatalf("generate service: %v", err)
	}
	if !strings.Contains(service.Content, "With(\"Details\")") {
		t.Fatal("service should preload Details")
	}
	if !strings.Contains(service.Content, "syncOrderDetails") {
		t.Fatal("service should sync detail rows")
	}
	if !strings.Contains(service.Content, "isEmptyOrderItemInput") {
		t.Fatal("service should skip blank detail placeholder rows")
	}
	if !strings.Contains(service.Content, "OrmTransaction(s.ctx,") {
		t.Fatal("master-detail write paths must use OrmTransaction for tenant-aware DB")
	}
	if strings.Contains(service.Content, "OrmQuery(s.ctx).Transaction") {
		t.Fatal("must not use OrmQuery(...).Transaction (not tenant-safe / may not compile)")
	}

	reqCreate, err := s.generateRequestCreate("order", "orders", fields, options)
	if err != nil {
		t.Fatalf("generate request create: %v", err)
	}
	if !strings.Contains(reqCreate.Content, "type OrderItemInput struct") {
		t.Fatal("create request should define OrderItemInput")
	}
	if !strings.Contains(reqCreate.Content, "Details []OrderItemInput") {
		t.Fatal("create request should accept Details")
	}

	formPage, err := s.generateFrontendFormPage("order", "orders", fields, options)
	if err != nil {
		t.Fatalf("generate vue form page: %v", err)
	}
	if !strings.Contains(formPage.Content, "formData.details") {
		t.Fatal("vue form should bind details")
	}
	if !strings.Contains(formPage.Content, "addDetailRow") {
		t.Fatal("vue form should support adding detail rows")
	}

	reactForm, err := s.generateReactFormModal("order", "orders", fields, options)
	if err != nil {
		t.Fatalf("generate react form modal: %v", err)
	}
	if !strings.Contains(reactForm.Content, "Form.List name=\"details\"") {
		t.Fatal("react master-detail form should use Form.List details")
	}
	if !strings.Contains(reactForm.Content, "isBlankDetailRow") {
		t.Fatal("react master-detail form should skip blank detail rows before submit")
	}
	if !strings.Contains(reactForm.Content, "validateFields(masterNames)") {
		t.Fatal("react master-detail form should validate master fields only (not Form.List placeholders)")
	}

	files, err := s.Generate("order", "orders", fields, []string{"model", "service", "request_create"}, options)
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}
	var hasDetailModel, hasDetailMigration bool
	for _, f := range files {
		if strings.Contains(f.Path, "app/models/order_item.go") {
			hasDetailModel = true
			if !strings.Contains(f.Content, "type OrderItem struct") {
				t.Fatal("detail model content missing OrderItem type")
			}
		}
		if strings.Contains(f.Path, "create_order_items_table.go") {
			hasDetailMigration = true
			if !strings.Contains(f.Content, `table.Decimal("unit_price").Total(10).Places(2)`) {
				t.Fatalf("detail migration must use Decimal().Total().Places(), got:\n%s", f.Content)
			}
			if strings.Contains(f.Content, `Decimal("unit_price",`) {
				t.Fatal("detail migration must not use Decimal(name, precision, scale)")
			}
		}
	}
	if !hasDetailModel {
		t.Fatal("generate should append detail model file when model/service selected")
	}
	if !hasDetailMigration {
		t.Fatal("generate should append detail migration when model selected")
	}

	partial, err := s.Generate("order", "orders", fields, []string{"api"}, options)
	if err != nil {
		t.Fatalf("generate api-only: %v", err)
	}
	for _, f := range partial {
		if strings.Contains(f.Path, "order_item.go") || strings.Contains(f.Path, "create_order_items_table.go") {
			t.Fatalf("api-only generate should not append detail artifacts, got %s", f.Path)
		}
	}

	serviceOnly, err := s.Generate("order", "orders", fields, []string{"service"}, options)
	if err != nil {
		t.Fatalf("generate service-only: %v", err)
	}
	var hasCreateReq bool
	for _, f := range serviceOnly {
		if strings.Contains(f.Path, "request") && strings.Contains(f.Path, "create") {
			hasCreateReq = true
			if !strings.Contains(f.Content, "type OrderItemInput struct") {
				t.Fatal("auto-selected request_create must define OrderItemInput for master-detail service")
			}
		}
	}
	if !hasCreateReq {
		t.Fatal("master-detail service generate must also emit request_create (Detail Input type)")
	}
}

func TestExpandSelectedMasterDetailFiles(t *testing.T) {
	selected := map[string]bool{"service": true}
	expandSelectedMasterDetailFiles(selected, map[string]bool{"is_master_detail": true})
	if !selected["request_create"] {
		t.Fatal("service selection should force request_create when master-detail is on")
	}
}

func TestDeriveDetailModuleName(t *testing.T) {
	if got := deriveDetailModuleName("order_items"); got != "order_item" {
		t.Fatalf("order_items -> order_item, got %s", got)
	}
	if got := deriveDetailModuleName("categories"); got != "category" {
		t.Fatalf("categories -> category, got %s", got)
	}
}
