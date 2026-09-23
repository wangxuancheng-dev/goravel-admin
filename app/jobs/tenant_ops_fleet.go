package jobs

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/services"
)

// TenantOpsFleet runs many migrate/seed ops in one long-running job using
// TENANCY_MIGRATE_CONCURRENCY (same path as tenant:migrate-all).
type TenantOpsFleet struct{}

func (r *TenantOpsFleet) Signature() string {
	return "tenant_ops_fleet"
}

func (r *TenantOpsFleet) Handle(args ...any) (retErr error) {
	fleet, parseErr := parseTenantOpsFleetArgs(args...)
	defer func() {
		if rec := recover(); rec != nil {
			facades.Log().Errorf("TenantOpsFleet Job panic: %v", rec)
			retErr = fmt.Errorf("panic: %v", rec)
		}
	}()
	if parseErr != nil {
		return parseErr
	}
	// Never bubble item failures: retrying the whole fleet would re-run successes.
	_ = services.RunTenantOpsFleet(fleet)
	return nil
}

// ShouldRetry disables queue retries for fleet jobs (per-tenant state is durable).
func (r *TenantOpsFleet) ShouldRetry(err error, attempt int) (bool, time.Duration) {
	return false, 0
}

func parseTenantOpsFleetArgs(args ...any) (services.TenantOpsFleetArgs, error) {
	var out services.TenantOpsFleetArgs
	if len(args) < 1 {
		return out, fmt.Errorf("missing tenant ops fleet arguments")
	}
	switch v := args[0].(type) {
	case services.TenantOpsFleetArgs:
		return v, nil
	case string:
		if err := json.Unmarshal([]byte(v), &out); err != nil {
			return out, err
		}
		return out, nil
	case map[string]any:
		out.Concurrency = cast.ToInt(v["concurrency"])
		rawItems, ok := v["items"]
		if !ok {
			return out, fmt.Errorf("fleet args missing items")
		}
		b, err := json.Marshal(rawItems)
		if err != nil {
			return out, err
		}
		if err := json.Unmarshal(b, &out.Items); err != nil {
			return out, err
		}
		return out, nil
	default:
		return out, fmt.Errorf("invalid tenant ops fleet args type: %T", args[0])
	}
}