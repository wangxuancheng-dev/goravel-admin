package search

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/goravel/framework/facades"
)

// ResourceOrders is the built-in searchable resource key.
const ResourceOrders = "orders"

// IndexDefinition describes a searchable resource in a driver-agnostic way.
// Register extra resources with RegisterDefinition from your module's init/provider
// (do not hard-code optional business modules here).
type IndexDefinition struct {
	Key          string
	PrimaryKey   string   // document id field (Meili primary; ES _id uses this value)
	Searchable   []string // full-text fields
	Filterable   []string // term/range filters
	Sortable     []string
	DefaultBoost map[string]float64
}

var (
	indexDefsMu sync.RWMutex
	indexDefs   = map[string]IndexDefinition{
		ResourceOrders: {
			Key:        ResourceOrders,
			PrimaryKey: "order_no",
			Searchable: []string{"order_no", "product_names", "remark"},
			Filterable: []string{"id", "order_no", "user_id", "status", "amount", "created_at", "updated_at"},
			Sortable:   []string{"id", "order_no", "user_id", "status", "amount", "created_at", "updated_at"},
			DefaultBoost: map[string]float64{
				"order_no": 2,
			},
		},
	}
)

// RegisterDefinition adds or replaces a searchable resource definition (call from module init).
func RegisterDefinition(def IndexDefinition) {
	key := strings.ToLower(strings.TrimSpace(def.Key))
	if key == "" {
		return
	}
	def.Key = key
	indexDefsMu.Lock()
	defer indexDefsMu.Unlock()
	indexDefs[key] = def
}

// Definition returns a registered index definition.
func Definition(resource string) (IndexDefinition, bool) {
	indexDefsMu.RLock()
	defer indexDefsMu.RUnlock()
	d, ok := indexDefs[strings.ToLower(strings.TrimSpace(resource))]
	return d, ok
}

// RegisteredResources returns registered resource keys (snapshot).
func RegisteredResources() []string {
	indexDefsMu.RLock()
	defer indexDefsMu.RUnlock()
	out := make([]string, 0, len(indexDefs))
	for k := range indexDefs {
		out = append(out, k)
	}
	return out
}

// ResourceSyncEnabled reports whether sync is on for a resource key.
func ResourceSyncEnabled(resource string) bool {
	if !Enabled() {
		return false
	}
	resource = strings.ToLower(strings.TrimSpace(resource))
	return facades.Config().GetBool("search.indexes."+resource+".sync_enabled", false)
}

// ResourceIndexShortName returns configured short name (without tenant prefix).
func ResourceIndexShortName(resource string) string {
	resource = strings.ToLower(strings.TrimSpace(resource))
	fallback := resource
	if fallback == "" {
		fallback = "index"
	}
	n := strings.TrimSpace(facades.Config().GetString("search.indexes."+resource+".name", fallback))
	if n == "" {
		return fallback
	}
	return n
}

// ResourceIndexShortNameFor returns tenant-scoped short name (fail-closed empty when unbound).
func ResourceIndexShortNameFor(ctx context.Context, resource string) string {
	return PrefixedIndexShortName(ctx, ResourceIndexShortName(resource))
}

// IsResourceIndexShortName reports whether shortName belongs to resource (incl. tenant prefix).
func IsResourceIndexShortName(resource, shortName string) bool {
	shortName = strings.TrimSpace(shortName)
	if shortName == "" {
		return false
	}
	base := ResourceIndexShortName(resource)
	if shortName == base || shortName == resource {
		return true
	}
	return strings.HasSuffix(shortName, "_"+base) || strings.HasSuffix(shortName, "_"+resource)
}

// MatchResource returns the resource key for a short index name, if known.
func MatchResource(shortName string) (string, bool) {
	indexDefsMu.RLock()
	defer indexDefsMu.RUnlock()
	for key := range indexDefs {
		if IsResourceIndexShortName(key, shortName) {
			return key, true
		}
	}
	return "", false
}

// AnyResourceSyncEnabled is true when any registered resource has sync enabled.
func AnyResourceSyncEnabled() bool {
	if !Enabled() {
		return false
	}
	for _, key := range RegisteredResources() {
		if ResourceSyncEnabled(key) {
			return true
		}
	}
	return false
}

// DocumentID extracts primary key string for indexing/delete.
func DocumentID(def IndexDefinition, doc map[string]any) (string, error) {
	if def.PrimaryKey == "" {
		return "", fmt.Errorf("search index %s missing primary key", def.Key)
	}
	raw, ok := doc[def.PrimaryKey]
	if !ok || raw == nil {
		return "", fmt.Errorf("document missing primary key %s", def.PrimaryKey)
	}
	switch v := raw.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return "", fmt.Errorf("empty primary key %s", def.PrimaryKey)
		}
		return v, nil
	case float64:
		return fmt.Sprintf("%.0f", v), nil
	case float32:
		return fmt.Sprintf("%.0f", v), nil
	case int:
		return fmt.Sprintf("%d", v), nil
	case int64:
		return fmt.Sprintf("%d", v), nil
	case uint:
		return fmt.Sprintf("%d", v), nil
	case uint64:
		return fmt.Sprintf("%d", v), nil
	default:
		s := strings.TrimSpace(fmt.Sprint(v))
		if s == "" || s == "<nil>" {
			return "", fmt.Errorf("invalid primary key %s", def.PrimaryKey)
		}
		return s, nil
	}
}
