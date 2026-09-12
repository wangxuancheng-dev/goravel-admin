package search

import (
	"testing"
)

func TestDefinitionOrders(t *testing.T) {
	orders, ok := Definition(ResourceOrders)
	if !ok || orders.PrimaryKey != "order_no" {
		t.Fatalf("orders def: %+v ok=%v", orders, ok)
	}
}

func TestRegisterDefinitionExtendsMatch(t *testing.T) {
	RegisterDefinition(IndexDefinition{
		Key:        "products",
		PrimaryKey: "id",
		Searchable: []string{"name"},
		Filterable: []string{"id", "status"},
		Sortable:   []string{"id"},
	})
	t.Cleanup(func() {
		indexDefsMu.Lock()
		delete(indexDefs, "products")
		indexDefsMu.Unlock()
	})

	if !IsResourceIndexShortName("products", "acme_products") {
		t.Fatal("tenant products should match")
	}
	key, ok := MatchResource("t1_products")
	if !ok || key != "products" {
		t.Fatalf("got %q ok=%v", key, ok)
	}
	def, ok := Definition("products")
	if !ok || def.PrimaryKey != "id" {
		t.Fatalf("def=%+v", def)
	}
}

func TestDocumentID(t *testing.T) {
	def, _ := Definition(ResourceOrders)
	id, err := DocumentID(def, map[string]any{"order_no": "ORD-1"})
	if err != nil || id != "ORD-1" {
		t.Fatalf("got %q err=%v", id, err)
	}
}

func TestMatchResourceOrders(t *testing.T) {
	key, ok := MatchResource("acme_orders")
	if !ok || key != ResourceOrders {
		t.Fatalf("got %q ok=%v", key, ok)
	}
}
