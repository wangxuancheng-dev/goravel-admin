package elasticsearch

import (
	"reflect"
	"testing"

	"goravel/app/search"
)

func TestBuildElasticsearchSearchFieldsAppliesBoosts(t *testing.T) {
	fields := buildElasticsearchSearchFields(
		[]string{"order_no", "product_names", "remark"},
		map[string]float64{"order_no": 2},
	)
	want := []string{"order_no^2", "product_names", "remark"}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("got %v want %v", fields, want)
	}
}

func TestBuildElasticsearchSearchFieldsStripsInlineBoost(t *testing.T) {
	fields := buildElasticsearchSearchFields(
		[]string{"order_no^5", "remark"},
		map[string]float64{"order_no": 2},
	)
	want := []string{"order_no^2", "remark"}
	if !reflect.DeepEqual(fields, want) {
		t.Fatalf("got %v want %v", fields, want)
	}
}

func TestEngineQueryReady(t *testing.T) {
	if search.EngineQueryReady(nil) {
		t.Fatal("nil should not be ready")
	}
	if search.EngineQueryReady(search.NewNullEngine()) {
		t.Fatal("null engine should not be ready")
	}
}
