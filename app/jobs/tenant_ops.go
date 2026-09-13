package jobs

import (
	"encoding/json"
	"fmt"

	"github.com/goravel/framework/facades"
	"github.com/spf13/cast"

	"goravel/app/services"
)

// TenantOps runs platform tenant migrate / seed / backup / restore / purge asynchronously.
type TenantOps struct{}

func (r *TenantOps) Signature() string {
	return "tenant_ops"
}

func (r *TenantOps) Handle(args ...any) (retErr error) {
	opArgs, parseErr := parseTenantOpsArgs(args...)
	defer func() {
		if rec := recover(); rec != nil {
			facades.Log().Errorf("TenantOps Job panic: %v", rec)
			retErr = fmt.Errorf("panic: %v", rec)
			if parseErr == nil && opArgs.TenantID != 0 {
				admin := services.NewTenantAdminService()
				ops := services.NewTenantOpsService()
				if tenant, err := admin.GetByID(opArgs.TenantID); err == nil {
					_ = ops.MarkOpFailed(tenant, retErr.Error(), opArgs.OpLogID)
				}
			}
		}
	}()
	if parseErr != nil {
		return parseErr
	}
	return services.RunTenantOp(opArgs)
}

func parseTenantOpsArgs(args ...any) (services.TenantOpsArgs, error) {
	var out services.TenantOpsArgs
	if len(args) < 1 {
		return out, fmt.Errorf("missing tenant ops arguments")
	}
	switch v := args[0].(type) {
	case services.TenantOpsArgs:
		return v, nil
	case string:
		if err := json.Unmarshal([]byte(v), &out); err != nil {
			return out, err
		}
		return out, nil
	case map[string]any:
		out.TenantID = cast.ToUint(v["tenant_id"])
		out.Op = cast.ToString(v["op"])
		out.WithSeed = cast.ToBool(v["with_seed"])
		out.BackupName = cast.ToString(v["backup_name"])
		out.Keep = cast.ToInt(v["keep"])
		out.OpLogID = cast.ToUint(v["op_log_id"])
		out.BatchID = cast.ToString(v["batch_id"])
		out.OperatorID = cast.ToUint(v["operator_id"])
		out.OperatorName = cast.ToString(v["operator_name"])
		return out, nil
	default:
		return out, fmt.Errorf("invalid tenant ops args type: %T", args[0])
	}
}
