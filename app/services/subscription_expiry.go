package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/translation"
	"github.com/goravel/framework/facades"

	"goravel/app/entitlement"
	apperrors "goravel/app/errors"
	appfacades "goravel/app/facades"
	"goravel/app/models"
)

// Subscription expiry: the hourly tenant:subscription-expiry job reminds tenants (and
// the platform webhook) before a plan ends, then applies the platform's policy once
// ends_at (or trial_ends_at when there is no ends_at) plus the grace period has passed.
// Each notice is sent once per subscription (tenant_subscriptions.last_remind_days).

const (
	PlatformSettingSubExpiryAction  = "subscription_expiry_action"
	PlatformSettingSubGraceDays     = "subscription_grace_days"
	PlatformSettingSubRemindDays    = "subscription_remind_days"
	PlatformSettingSubNoticeLocale  = "subscription_notice_locale"
	SubExpiryActionDowngrade        = "downgrade"   // expire: tenant falls back to the default plan
	SubExpiryActionSuspend          = "suspend"     // expire and disable the tenant
	SubExpiryActionNotifyOnly       = "notify_only" // only notices; the plan keeps working
	subExpiryMaxGraceDays           = 90
	subExpiryMaxRemindDay           = 90
	subExpiryMaxRemindCount         = 5
	subExpiryStageEnded             = 0
	subExpiryNotificationType       = "notice" // a type the tenant notification bell shows (系统通知)
	defaultSubExpiryRemindDaysValue = "7,3,1"
)

// SubscriptionExpiryPolicy is the platform-wide expiry behaviour (platform_settings).
type SubscriptionExpiryPolicy struct {
	Action       string `json:"action"`
	GraceDays    int    `json:"grace_days"`
	RemindDays   []int  `json:"remind_days"`
	NoticeLocale string `json:"notice_locale"`
}

// GetSubscriptionExpiryPolicy reads the policy, falling back to defaults per field.
func GetSubscriptionExpiryPolicy(ctx context.Context) SubscriptionExpiryPolicy {
	p := SubscriptionExpiryPolicy{
		Action:       strings.TrimSpace(GetPlatformSetting(ctx, PlatformSettingSubExpiryAction, SubExpiryActionDowngrade)),
		NoticeLocale: strings.TrimSpace(GetPlatformSetting(ctx, PlatformSettingSubNoticeLocale, "cn")),
	}
	if !validSubExpiryAction(p.Action) {
		p.Action = SubExpiryActionDowngrade
	}
	if p.NoticeLocale != "cn" && p.NoticeLocale != "en" {
		p.NoticeLocale = "cn"
	}
	if n, err := strconv.Atoi(strings.TrimSpace(GetPlatformSetting(ctx, PlatformSettingSubGraceDays, "0"))); err == nil && n >= 0 && n <= subExpiryMaxGraceDays {
		p.GraceDays = n
	}
	days, err := ParseSubscriptionRemindDays(GetPlatformSetting(ctx, PlatformSettingSubRemindDays, defaultSubExpiryRemindDaysValue))
	if err != nil {
		days, _ = ParseSubscriptionRemindDays(defaultSubExpiryRemindDaysValue)
	}
	p.RemindDays = days
	return p
}

// SetSubscriptionExpiryPolicy validates and stores the policy.
func SetSubscriptionExpiryPolicy(ctx context.Context, p SubscriptionExpiryPolicy) error {
	p.Action = strings.TrimSpace(p.Action)
	if !validSubExpiryAction(p.Action) || p.GraceDays < 0 || p.GraceDays > subExpiryMaxGraceDays ||
		(p.NoticeLocale != "cn" && p.NoticeLocale != "en") {
		return apperrors.ErrInvalidArgument
	}
	days, err := normalizeRemindDays(p.RemindDays)
	if err != nil {
		return err
	}
	for key, value := range map[string]string{
		PlatformSettingSubExpiryAction: p.Action,
		PlatformSettingSubGraceDays:    strconv.Itoa(p.GraceDays),
		PlatformSettingSubRemindDays:   formatRemindDays(days),
		PlatformSettingSubNoticeLocale: p.NoticeLocale,
	} {
		if err := SetPlatformSetting(ctx, key, value); err != nil {
			return err
		}
	}
	return nil
}

func validSubExpiryAction(a string) bool {
	return a == SubExpiryActionDowngrade || a == SubExpiryActionSuspend || a == SubExpiryActionNotifyOnly
}

// ParseSubscriptionRemindDays parses "7,3,1" (empty = no reminders before the end).
func ParseSubscriptionRemindDays(raw string) ([]int, error) {
	var days []int
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, apperrors.ErrInvalidArgument
		}
		days = append(days, n)
	}
	return normalizeRemindDays(days)
}

// normalizeRemindDays dedupes and sorts descending (7,3,1); each day is 1..90.
func normalizeRemindDays(in []int) ([]int, error) {
	seen := map[int]bool{}
	out := []int{}
	for _, d := range in {
		if d < 1 || d > subExpiryMaxRemindDay {
			return nil, apperrors.ErrInvalidArgument
		}
		if !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	if len(out) > subExpiryMaxRemindCount {
		return nil, apperrors.ErrInvalidArgument
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out, nil
}

func formatRemindDays(days []int) string {
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = strconv.Itoa(d)
	}
	return strings.Join(parts, ",")
}

// subscriptionEnd is when the plan stops: ends_at, else the trial end; nil = open-ended.
func subscriptionEnd(sub *models.TenantSubscription) *time.Time {
	if sub.EndsAt != nil && !sub.EndsAt.IsZero() {
		return sub.EndsAt
	}
	if sub.TrialEndsAt != nil && !sub.TrialEndsAt.IsZero() {
		return sub.TrialEndsAt
	}
	return nil
}

type subExpiryDecision struct {
	// RemindStage >= 0: send that notice (days-before threshold, 0 = ended / in grace).
	RemindStage int
	Remind      bool
	Expire      bool
}

// decideSubscriptionExpiry is the pure rule set (unit-tested).
func decideSubscriptionExpiry(end, now time.Time, p SubscriptionExpiryPolicy, lastStage *int) subExpiryDecision {
	sentBefore := func(stage int) bool { return lastStage != nil && *lastStage <= stage }

	if !now.Before(end) {
		if p.Action == SubExpiryActionNotifyOnly {
			// Nothing changes; tell the tenant once that the plan has ended.
			return subExpiryDecision{Remind: !sentBefore(subExpiryStageEnded), RemindStage: subExpiryStageEnded}
		}
		if !now.Before(end.Add(time.Duration(p.GraceDays) * 24 * time.Hour)) {
			return subExpiryDecision{Expire: true}
		}
		// In the grace period.
		return subExpiryDecision{Remind: !sentBefore(subExpiryStageEnded), RemindStage: subExpiryStageEnded}
	}

	// Before the end: the tightest threshold the remaining time falls under.
	daysLeft := int((end.Sub(now) + 24*time.Hour - time.Nanosecond) / (24 * time.Hour))
	stage := -1
	for _, d := range p.RemindDays { // descending
		if daysLeft <= d {
			stage = d
		}
	}
	if stage < 0 || sentBefore(stage) {
		return subExpiryDecision{}
	}
	return subExpiryDecision{Remind: true, RemindStage: stage}
}

// SubscriptionExpiryReport summarises one run.
type SubscriptionExpiryReport struct {
	Checked   int      `json:"checked"`
	Reminded  int      `json:"reminded"`
	Expired   int      `json:"expired"`
	Suspended int      `json:"suspended"`
	Errors    []string `json:"errors,omitempty"`
}

// RunSubscriptionExpiry sends due notices and expires ended subscriptions, for all
// tenants or only onlyTenantIDs when given.
func RunSubscriptionExpiry(now time.Time, onlyTenantIDs ...uint) (*SubscriptionExpiryReport, error) {
	if now.IsZero() {
		now = time.Now()
	}
	report := &SubscriptionExpiryReport{}
	q := appfacades.PlatformQuery()
	if down, _ := appfacades.QueryUnavailable(q); down {
		return report, nil
	}
	policy := GetSubscriptionExpiryPolicy(context.Background())

	var subs []models.TenantSubscription
	query := q.Model(&models.TenantSubscription{}).
		Where("status", entitlement.SubActive).
		Where("(ends_at IS NOT NULL OR trial_ends_at IS NOT NULL)")
	if len(onlyTenantIDs) > 0 {
		query = query.Where("tenant_id IN ?", onlyTenantIDs)
	}
	if err := query.Order("id asc").Find(&subs); err != nil {
		return report, err
	}

	for i := range subs {
		sub := &subs[i]
		end := subscriptionEnd(sub)
		if end == nil {
			continue
		}
		report.Checked++
		d := decideSubscriptionExpiry(*end, now, policy, sub.LastRemindDays)
		if !d.Remind && !d.Expire {
			continue
		}
		var tenant models.Tenant
		if err := appfacades.PlatformQuery().Where("id", sub.TenantID).First(&tenant); err != nil || tenant.ID == 0 {
			continue // tenant gone (soft-deleted rows are not loaded)
		}
		planName := subscriptionPlanName(sub.PlanID)
		if d.Expire {
			expired, suspended, err := expireSubscription(sub, &tenant, planName, *end, policy)
			if expired {
				report.Expired++
			}
			if suspended {
				report.Suspended++
			}
			if err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("tenant %s: %v", tenant.Code, err))
			}
			continue
		}
		// Claim the stage first (conditional update) so overlapping runs send once.
		if !claimRemindStage(sub, d.RemindStage) {
			continue
		}
		sendSubscriptionNotice(&tenant, sub, planName, *end, d.RemindStage, now, policy)
		report.Reminded++
	}
	return report, nil
}

func claimRemindStage(sub *models.TenantSubscription, stage int) bool {
	q := appfacades.PlatformQuery().Model(&models.TenantSubscription{}).
		Where("id", sub.ID).Where("status", entitlement.SubActive)
	if sub.LastRemindDays == nil {
		q = q.Where("last_remind_days IS NULL")
	} else {
		q = q.Where("last_remind_days", *sub.LastRemindDays)
	}
	res, err := q.Update("last_remind_days", stage)
	if err != nil || res == nil || res.RowsAffected == 0 {
		return false
	}
	sub.LastRemindDays = &stage
	return true
}

// expireSubscription marks the subscription expired (once), recomputes entitlements,
// notifies, and disables the tenant under the suspend policy.
func expireSubscription(sub *models.TenantSubscription, tenant *models.Tenant, planName string, end time.Time, p SubscriptionExpiryPolicy) (expired, suspended bool, err error) {
	res, err := appfacades.PlatformQuery().Model(&models.TenantSubscription{}).
		Where("id", sub.ID).Where("status", entitlement.SubActive).
		Update("status", entitlement.SubExpired)
	if err != nil {
		return false, false, err
	}
	if res == nil || res.RowsAffected == 0 {
		return false, false, nil // another run got there first
	}
	ent := NewEntitlementService(context.Background())
	_ = ent.audit(0, tenant.ID, "subscription.expire", fmt.Sprintf("sub=%d plan=%s action=%s", sub.ID, planName, p.Action))
	if _, err := ent.Recompute(tenant.ID); err != nil {
		facades.Log().Warningf("subscription expiry: recompute tenant %s: %v", tenant.Code, err)
	}

	// Notify before suspending: a disabled tenant can no longer be reached.
	sendSubscriptionExpiredNotice(tenant, sub, planName, end, p)

	if p.Action != SubExpiryActionSuspend || tenant.Status != models.TenantStatusActive {
		return true, false, nil
	}
	if _, err := NewTenantAdminService().SetStatus(tenant.ID, models.TenantStatusDisabled); err != nil {
		return true, false, err
	}
	_ = ent.audit(0, tenant.ID, "tenant.suspend_on_expiry", fmt.Sprintf("sub=%d", sub.ID))
	return true, true, nil
}

func subscriptionPlanName(planID uint) string {
	var plan models.PlatformPlan
	if err := appfacades.PlatformQuery().Where("id", planID).First(&plan); err != nil || plan.ID == 0 {
		return ""
	}
	if strings.TrimSpace(plan.Name) != "" {
		return plan.Name
	}
	return plan.Code
}

func subExpiryLang(locale, key string, replace map[string]string) string {
	return facades.Lang(context.Background()).Get("subscription."+key, translation.Option{Locale: locale, Replace: replace})
}

func subExpiryTime(t time.Time) string {
	loc, err := time.LoadLocation(facades.Config().GetString("app.timezone", "UTC"))
	if err != nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02 15:04")
}

func sendSubscriptionNotice(tenant *models.Tenant, sub *models.TenantSubscription, planName string, end time.Time, stage int, now time.Time, p SubscriptionExpiryPolicy) {
	replace := map[string]string{"plan": planName, "date": subExpiryTime(end)}
	var title, content, event string
	if stage == subExpiryStageEnded {
		event = "subscription_ended"
		title = subExpiryLang(p.NoticeLocale, "ended_title", nil)
		if p.Action == SubExpiryActionNotifyOnly {
			content = subExpiryLang(p.NoticeLocale, "ended_notify_content", replace)
		} else {
			replace["grace"] = subExpiryTime(end.Add(time.Duration(p.GraceDays) * 24 * time.Hour))
			replace["after"] = subExpiryLang(p.NoticeLocale, "after_"+p.Action, nil)
			content = subExpiryLang(p.NoticeLocale, "grace_content", replace)
		}
	} else {
		event = "subscription_expiring"
		// Shown as "about N days": nearest whole day (at least 1), so an end stored a
		// fraction of a second later than 48h from now still reads "2".
		days := int(math.Round(end.Sub(now).Hours() / 24))
		if days < 1 {
			days = 1
		}
		replace["days"] = strconv.Itoa(days)
		title = subExpiryLang(p.NoticeLocale, "expiring_title", nil)
		content = subExpiryLang(p.NoticeLocale, "expiring_content", replace)
	}
	notifyTenantAdmins(tenant, title, content)
	alertPlatformSubscription(event, tenant, sub, planName, end, p)
}

func sendSubscriptionExpiredNotice(tenant *models.Tenant, sub *models.TenantSubscription, planName string, end time.Time, p SubscriptionExpiryPolicy) {
	replace := map[string]string{"plan": planName, "date": subExpiryTime(end)}
	title := subExpiryLang(p.NoticeLocale, "expired_title", nil)
	content := subExpiryLang(p.NoticeLocale, "expired_content_"+p.Action, replace)
	notifyTenantAdmins(tenant, title, content)
	alertPlatformSubscription("subscription_expired", tenant, sub, planName, end, p)
}

// notifyTenantAdmins posts an in-app notification to every admin of the tenant (best-effort).
func notifyTenantAdmins(tenant *models.Tenant, title, content string) {
	if tenant.Status != models.TenantStatusActive || !tenant.IsProvisionReady() {
		return
	}
	ctx, err := bindTenantSupportCtx(tenant)
	if err != nil {
		facades.Log().Warningf("subscription notice: tenant %s unreachable: %v", tenant.Code, err)
		return
	}
	if _, err := NewNotificationServiceImpl(ctx).Create(title, content, subExpiryNotificationType, nil, nil); err != nil {
		facades.Log().Warningf("subscription notice: tenant %s: %v", tenant.Code, err)
	}
}

// alertPlatformSubscription posts to the platform ops webhook when one is configured
// (recorded under 告警投递 like the other platform alerts).
func alertPlatformSubscription(event string, tenant *models.Tenant, sub *models.TenantSubscription, planName string, end time.Time, p SubscriptionExpiryPolicy) {
	url := ResolveTenantOpsAlertWebhookURL()
	if url == "" {
		return
	}
	payload := map[string]any{
		"event":           event,
		"tenant_id":       tenant.ID,
		"tenant_code":     tenant.Code,
		"subscription_id": sub.ID,
		"plan":            planName,
		"ends_at":         end.Unix(),
		"action":          p.Action,
		"grace_days":      p.GraceDays,
		"timestamp":       time.Now().Unix(),
		"app":             facades.Config().GetString("app.name", ""),
		"env":             facades.Config().GetString("app.env", ""),
	}
	tid := tenant.ID
	deliverPlatformWebhookAlert(models.PlatformAlertChannelWebhook, event, &tid, tenant.Code, "subscription", url, payload)
}
