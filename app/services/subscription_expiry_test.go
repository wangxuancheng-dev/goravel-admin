package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSubscriptionRemindDays(t *testing.T) {
	days, err := ParseSubscriptionRemindDays(" 1, 7,3 ,7")
	require.NoError(t, err)
	assert.Equal(t, []int{7, 3, 1}, days)

	days, err = ParseSubscriptionRemindDays("")
	require.NoError(t, err)
	assert.Empty(t, days)

	for _, bad := range []string{"0", "91", "x", "1,2,3,4,5,6"} {
		_, err = ParseSubscriptionRemindDays(bad)
		assert.Error(t, err, bad)
	}
}

func TestDecideSubscriptionExpiry(t *testing.T) {
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	p := SubscriptionExpiryPolicy{Action: SubExpiryActionDowngrade, GraceDays: 3, RemindDays: []int{7, 3, 1}}
	stage := func(n int) *int { return &n }

	// Far away: nothing.
	d := decideSubscriptionExpiry(now.Add(10*day), now, p, nil)
	assert.False(t, d.Remind || d.Expire)

	// 5 days left -> the 7-day notice, once.
	d = decideSubscriptionExpiry(now.Add(5*day), now, p, nil)
	assert.True(t, d.Remind)
	assert.Equal(t, 7, d.RemindStage)
	d = decideSubscriptionExpiry(now.Add(5*day), now, p, stage(7))
	assert.False(t, d.Remind, "7-day notice already sent")

	// 2.5 days left -> 3-day notice after the 7-day one; 12h left -> 1-day notice.
	d = decideSubscriptionExpiry(now.Add(60*time.Hour), now, p, stage(7))
	assert.True(t, d.Remind)
	assert.Equal(t, 3, d.RemindStage)
	d = decideSubscriptionExpiry(now.Add(12*time.Hour), now, p, stage(3))
	assert.Equal(t, 1, d.RemindStage)
	assert.True(t, d.Remind)

	// The job was down for a while: only the tightest notice is sent.
	d = decideSubscriptionExpiry(now.Add(12*time.Hour), now, p, nil)
	assert.Equal(t, 1, d.RemindStage)

	// Ended, inside the grace period: the "ended" notice once, no expiry yet.
	d = decideSubscriptionExpiry(now.Add(-day), now, p, stage(1))
	assert.True(t, d.Remind)
	assert.Equal(t, subExpiryStageEnded, d.RemindStage)
	assert.False(t, d.Expire)
	d = decideSubscriptionExpiry(now.Add(-day), now, p, stage(subExpiryStageEnded))
	assert.False(t, d.Remind || d.Expire)

	// Grace over: expire.
	d = decideSubscriptionExpiry(now.Add(-3*day), now, p, stage(subExpiryStageEnded))
	assert.True(t, d.Expire)

	// No grace: expire right at the end.
	p0 := p
	p0.GraceDays = 0
	assert.True(t, decideSubscriptionExpiry(now, now, p0, nil).Expire)

	// notify_only never expires; tells the tenant once that the plan ended.
	pn := p
	pn.Action = SubExpiryActionNotifyOnly
	d = decideSubscriptionExpiry(now.Add(-30*day), now, pn, nil)
	assert.False(t, d.Expire)
	assert.True(t, d.Remind)
	d = decideSubscriptionExpiry(now.Add(-30*day), now, pn, stage(subExpiryStageEnded))
	assert.False(t, d.Remind || d.Expire)

	// No reminder days configured: still expires.
	pr := p0
	pr.RemindDays = nil
	assert.False(t, decideSubscriptionExpiry(now.Add(12*time.Hour), now, pr, nil).Remind)
	assert.True(t, decideSubscriptionExpiry(now.Add(-time.Minute), now, pr, nil).Expire)
}
