package services

import (
	"encoding/json"
	"testing"
)

func TestFinalizeAIGeneratedConfigMasterDetail(t *testing.T) {
	s := &CodeGeneratorServiceImpl{}
	raw := `{
		"module_name": "quote",
		"table_name": "quotes",
		"is_master_detail": true,
		"detail_table_name": "quote_items",
		"fields": [
			{"name": "quote_no", "label": "Quote No", "db_type": "string"}
		],
		"detail_fields": [
			{"name": "sku", "label": "SKU", "db_type": "string"},
			{"name": "qty", "label": "Qty", "db_type": "integer"}
		]
	}`
	var config AIGeneratedConfig
	if err := json.Unmarshal([]byte(raw), &config); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := s.finalizeAIGeneratedConfig(&config); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if !config.IsMasterDetail {
		t.Fatal("expected is_master_detail")
	}
	if config.DetailTableName != "quote_items" {
		t.Fatalf("detail table: %s", config.DetailTableName)
	}
	if len(config.DetailFields) < 2 {
		t.Fatalf("detail fields too few: %d", len(config.DetailFields))
	}
	hasCreatedAt := false
	for _, f := range config.DetailFields {
		if f.Name == "created_at" {
			hasCreatedAt = true
		}
	}
	if !hasCreatedAt {
		t.Fatal("detail fields should get system timestamps")
	}
}

func TestFinalizeAIGeneratedConfigInfersMasterDetail(t *testing.T) {
	s := &CodeGeneratorServiceImpl{}
	config := &AIGeneratedConfig{
		ModuleName:      "order",
		TableName:       "orders",
		Fields:          []FieldConfig{{Name: "order_no", DBType: "string"}},
		DetailTableName: "order_items",
		DetailFields:    []FieldConfig{{Name: "sku", DBType: "string"}},
	}
	if err := s.finalizeAIGeneratedConfig(config); err != nil {
		t.Fatalf("finalize: %v", err)
	}
	if !config.IsMasterDetail {
		t.Fatal("should infer is_master_detail from detail payload")
	}
}

func TestFinalizeAIGeneratedConfigRejectsIncompleteMasterDetail(t *testing.T) {
	s := &CodeGeneratorServiceImpl{}
	config := &AIGeneratedConfig{
		ModuleName:     "order",
		TableName:      "orders",
		Fields:         []FieldConfig{{Name: "order_no", DBType: "string"}},
		IsMasterDetail: true,
	}
	if err := s.finalizeAIGeneratedConfig(config); err == nil {
		t.Fatal("expected detail_table_name_required")
	}
}
