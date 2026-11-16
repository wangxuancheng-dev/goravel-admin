package feature_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appfacades "goravel/app/facades"
	"goravel/app/models"
	"goravel/app/search"
	"goravel/app/services"
)

// recordingEngine is a search engine that records dropped indexes.
type recordingEngine struct {
	mu      sync.Mutex
	dropped []string
	dropErr error
}

func (e *recordingEngine) Name() string                              { return search.DriverElasticsearch }
func (e *recordingEngine) Ping(context.Context) error                { return nil }
func (e *recordingEngine) EnsureIndex(context.Context, string) error { return nil }
func (e *recordingEngine) Index(context.Context, string, string, map[string]any) error {
	return nil
}
func (e *recordingEngine) Delete(context.Context, string, string) error { return nil }
func (e *recordingEngine) Search(context.Context, string, search.SearchRequest) (*search.SearchResult, error) {
	return &search.SearchResult{}, nil
}
func (e *recordingEngine) DropIndex(_ context.Context, index string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.dropErr != nil {
		return e.dropErr
	}
	e.dropped = append(e.dropped, index)
	return nil
}
func (e *recordingEngine) drops() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return append([]string(nil), e.dropped...)
}

// TestTenantSearchIndexesFollowTheTenant: a tenant's search indexes are named after
// its code, and codes are reusable once a tenant is deleted for good. The indexes must
// therefore be dropped before the row is removed (a failure keeps the row, so the code
// stays taken) and again when a tenant is created (leftovers from before this rule).
func TestTenantSearchIndexesFollowTheTenant(t *testing.T) {
	withTenancyDriver(t, "database")
	allowSharedTenantDBCredentials(t)
	for key, val := range map[string]any{"search.enabled": true, "search.driver": search.DriverElasticsearch} {
		prev := facades.Config().Get(key)
		facades.Config().Add(key, val)
		t.Cleanup(func() { facades.Config().Add(key, prev) })
	}
	engine := &recordingEngine{}
	search.SetEngineForTest(engine)
	t.Cleanup(func() { search.SetEngineForTest(nil) })

	// A code longer than 48 chars: index names used to be cut at 48, so two tenants
	// sharing a 48-char prefix shared an index. (54 keeps "tenant_"+code a valid
	// MySQL database name.)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	code := "sidx" + strings.Repeat("x", 54-4-len(suffix)) + suffix
	require.Len(t, code, 54)
	want := code + "_" + search.OrdersIndexShortName()

	admin := services.NewTenantAdminService()
	conn := services.NewTenantConnectionService()
	tn, err := admin.Create(services.TenantCreateInput{Code: code, Name: "search index"})
	if err != nil {
		skipWithoutTenantDB(t, err)
	}
	t.Cleanup(func() {
		_ = conn.DropStorage(tn)
		_, _ = appfacades.PlatformQuery().WithTrashed().Where("id", tn.ID).ForceDelete(&models.Tenant{})
	})
	assert.Contains(t, engine.drops(), want, "creating a tenant clears indexes left under its code (full code, not truncated)")

	_, err = admin.DeleteTenant(tn.ID, code, services.TenantDeleteOptions{})
	require.NoError(t, err)

	// The search engine is down: the tenant is not deleted for good, the code stays taken.
	engine.dropErr = errors.New("search engine unreachable")
	require.Error(t, admin.ForceDeleteTenant(tn.ID, code))
	kept, err := admin.GetByIDIncludingTrashed(tn.ID)
	require.NoError(t, err, "the row (and its code) survives a failed index drop")
	require.Equal(t, tn.ID, kept.ID)

	engine.dropErr = nil
	before := len(engine.drops())
	require.NoError(t, admin.ForceDeleteTenant(tn.ID, code))
	assert.Greater(t, len(engine.drops()), before)
	assert.Equal(t, want, engine.drops()[len(engine.drops())-1], "deleting for good drops the tenant's index")
	_, err = admin.GetByIDIncludingTrashed(tn.ID)
	assert.Error(t, err, "the row is gone after its indexes")

	// A missing id is "not found", not an empty tenant that later checks (such as
	// confirm_code == code, both "") would accept.
	_, err = admin.GetByID(tn.ID)
	assert.Error(t, err)
	_, err = admin.DeleteTenant(tn.ID, "", services.TenantDeleteOptions{})
	assert.Error(t, err, "deleting a missing tenant with an empty confirm code is refused")
}
