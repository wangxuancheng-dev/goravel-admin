package elasticsearch

import "testing"

func TestTextFieldMappingWithoutAnalyzer(t *testing.T) {
	m := textFieldMapping("", "")
	if m["type"] != "text" {
		t.Fatalf("expected type text, got %v", m["type"])
	}
	if _, ok := m["analyzer"]; ok {
		t.Fatal("expected no analyzer key")
	}
}

func TestTextFieldMappingWithIK(t *testing.T) {
	m := textFieldMapping("ik_max_word", "ik_smart")
	if m["analyzer"] != "ik_max_word" || m["search_analyzer"] != "ik_smart" {
		t.Fatalf("unexpected mapping: %v", m)
	}
}

func TestOrdersIndexBodyTextFields(t *testing.T) {
	body := ordersIndexBody("ik_max_word", "ik_smart")
	props, ok := body["mappings"].(map[string]any)["properties"].(map[string]any)
	if !ok {
		t.Fatal("missing properties")
	}
	remark, ok := props["remark"].(map[string]any)
	if !ok || remark["analyzer"] != "ik_max_word" {
		t.Fatalf("unexpected remark mapping: %v", props["remark"])
	}
}

func TestOrdersIndexBodyStandardFallback(t *testing.T) {
	body := ordersIndexBody("", "")
	props := body["mappings"].(map[string]any)["properties"].(map[string]any)
	remark := props["remark"].(map[string]any)
	if _, ok := remark["analyzer"]; ok {
		t.Fatal("standard mapping should not set analyzer")
	}
}
