package services

import (
	"testing"

	"github.com/goravel/framework/facades"
)

func TestFlexibleScheduleQueueName(t *testing.T) {
	prev := facades.Config().GetString("tenancy.flex_schedule_queue", "schedule")
	t.Cleanup(func() {
		facades.Config().Add("tenancy.flex_schedule_queue", prev)
	})
	facades.Config().Add("tenancy.flex_schedule_queue", "schedule")
	if got := FlexibleScheduleQueueName(); got != "schedule" {
		t.Fatalf("got %q", got)
	}
	facades.Config().Add("tenancy.flex_schedule_queue", "")
	if got := FlexibleScheduleQueueName(); got != "schedule" {
		t.Fatalf("empty fallback got %q", got)
	}
}
