package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"goravel/app/models"
)

func TestDemoActivityDesiredStatusOnce(t *testing.T) {
	start := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	a := &models.DemoActivity{
		ScheduleType: models.DemoActivityScheduleOnce,
		StartAt:      &start,
		EndAt:        &end,
		Enabled:      true,
		Timezone:     "Asia/Shanghai",
	}

	assert.Equal(t, models.DemoActivityStatusPending, DemoActivityDesiredStatus(a, start.Add(-time.Second)))
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, start))
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, start.Add(time.Hour)))
	assert.Equal(t, models.DemoActivityStatusEnded, DemoActivityDesiredStatus(a, end))
	assert.False(t, DemoActivityIsActive(a, end))
	assert.True(t, DemoActivityIsActive(a, start.Add(time.Minute)))
}

func TestDemoActivityDesiredStatusDaily(t *testing.T) {
	a := &models.DemoActivity{
		ScheduleType: models.DemoActivityScheduleDaily,
		DailyStart:   "09:00",
		DailyEnd:     "18:00",
		Enabled:      true,
		Timezone:     "UTC",
	}
	inWindow := time.Date(2026, 9, 20, 10, 30, 0, 0, time.UTC)
	outWindow := time.Date(2026, 9, 20, 8, 59, 0, 0, time.UTC)
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, inWindow))
	assert.Equal(t, models.DemoActivityStatusPending, DemoActivityDesiredStatus(a, outWindow))
}

func TestDemoActivityDesiredStatusDailyOvernight(t *testing.T) {
	a := &models.DemoActivity{
		ScheduleType: models.DemoActivityScheduleDaily,
		DailyStart:   "22:00",
		DailyEnd:     "02:00",
		Enabled:      true,
		Timezone:     "UTC",
	}
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, time.Date(2026, 9, 20, 23, 0, 0, 0, time.UTC)))
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, time.Date(2026, 9, 21, 1, 0, 0, 0, time.UTC)))
	assert.Equal(t, models.DemoActivityStatusPending, DemoActivityDesiredStatus(a, time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)))
}

func TestDemoActivityDesiredStatusWeekly(t *testing.T) {
	// 2026-09-20 is Sunday => iso 7
	a := &models.DemoActivity{
		ScheduleType: models.DemoActivityScheduleWeekly,
		Weekdays:     "1,2,3,4,5",
		DailyStart:   "09:00",
		DailyEnd:     "18:00",
		Enabled:      true,
		Timezone:     "UTC",
	}
	sunday := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	monday := time.Date(2026, 9, 21, 10, 0, 0, 0, time.UTC)
	assert.Equal(t, models.DemoActivityStatusPending, DemoActivityDesiredStatus(a, sunday))
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, monday))
}

func TestDemoActivityDesiredStatusMonthlyRange(t *testing.T) {
	a := &models.DemoActivity{
		ScheduleType:  models.DemoActivityScheduleMonthly,
		MonthDayStart: 1,
		MonthDayEnd:   5,
		Enabled:       true,
		Timezone:      "UTC",
	}
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)))
	assert.Equal(t, models.DemoActivityStatusPending, DemoActivityDesiredStatus(a, time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC)))
}

func TestDemoActivityDesiredStatusYearly(t *testing.T) {
	a := &models.DemoActivity{
		ScheduleType: models.DemoActivityScheduleYearly,
		YearStart:    "03-01",
		YearEnd:      "03-15",
		Enabled:      true,
		Timezone:     "UTC",
	}
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, time.Date(2026, 3, 10, 8, 0, 0, 0, time.UTC)))
	assert.Equal(t, models.DemoActivityStatusPending, DemoActivityDesiredStatus(a, time.Date(2026, 4, 1, 8, 0, 0, 0, time.UTC)))
}

func TestDemoActivityDesiredStatusYearlyWrap(t *testing.T) {
	a := &models.DemoActivity{
		ScheduleType: models.DemoActivityScheduleYearly,
		YearStart:    "12-20",
		YearEnd:      "01-10",
		Enabled:      true,
		Timezone:     "UTC",
	}
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, time.Date(2026, 12, 25, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, models.DemoActivityStatusRunning, DemoActivityDesiredStatus(a, time.Date(2027, 1, 5, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, models.DemoActivityStatusPending, DemoActivityDesiredStatus(a, time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)))
}

func TestDemoActivityDisabled(t *testing.T) {
	start := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)
	a := &models.DemoActivity{
		ScheduleType: models.DemoActivityScheduleOnce,
		StartAt:      &start,
		EndAt:        &end,
		Enabled:      false,
	}
	assert.Equal(t, models.DemoActivityStatusEnded, DemoActivityDesiredStatus(a, start.Add(time.Minute)))
}
