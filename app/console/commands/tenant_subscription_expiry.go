package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"

	"goravel/app/services"
	"goravel/app/tenancy"
)

// TenantSubscriptionExpiry sends plan expiry notices and applies the expiry policy.
type TenantSubscriptionExpiry struct{}

func (r *TenantSubscriptionExpiry) Signature() string {
	return "tenant:subscription-expiry"
}

func (r *TenantSubscriptionExpiry) Description() string {
	return "Remind tenants before their plan ends and expire / downgrade / suspend per platform settings"
}

func (r *TenantSubscriptionExpiry) Extend() command.Extend {
	return command.Extend{Category: "tenant"}
}

func (r *TenantSubscriptionExpiry) Handle(ctx console.Context) error {
	if !tenancy.Enabled() {
		ctx.Warning("tenancy disabled; skip")
		return nil
	}
	report, err := services.RunSubscriptionExpiry(time.Now())
	if err != nil {
		return err
	}
	ctx.Info(fmt.Sprintf("subscription-expiry checked=%d reminded=%d expired=%d suspended=%d",
		report.Checked, report.Reminded, report.Expired, report.Suspended))
	if len(report.Errors) > 0 {
		ctx.Warning(strings.Join(report.Errors, "; "))
	}
	return nil
}
