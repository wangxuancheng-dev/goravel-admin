package services

import (
	"strings"
	"testing"
	"time"
)

func TestParseOrderListTimeRangeDefaults(t *testing.T) {
	start, end, err := ParseOrderListTimeRange("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end.IsZero() == false {
		t.Fatalf("empty end should be zero, got %v", end)
	}
	ago := time.Now().UTC().AddDate(0, 0, -7)
	if start.Before(ago.Add(-2*time.Minute)) || start.After(ago.Add(2*time.Minute)) {
		t.Fatalf("default start should be ~7 days ago, got %v", start)
	}
}

func TestParseOrderListTimeRangeInvalid(t *testing.T) {
	_, _, err := ParseOrderListTimeRange("bad-time", "")
	if err == nil || !strings.Contains(err.Error(), "invalid_start_time") {
		t.Fatalf("want invalid_start_time, got %v", err)
	}

	_, _, err = ParseOrderListTimeRange("2026-01-01 00:00:00", "bad-end")
	if err == nil || !strings.Contains(err.Error(), "invalid_end_time") {
		t.Fatalf("want invalid_end_time, got %v", err)
	}
}

func TestParsePaymentListTimeRangeDefaults(t *testing.T) {
	start, end, err := ParsePaymentListTimeRange("", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if end.IsZero() {
		t.Fatal("payment end default should be now, got zero")
	}
	ago := time.Now().UTC().AddDate(0, 0, -7)
	if start.Before(ago.Add(-2*time.Minute)) || start.After(ago.Add(2*time.Minute)) {
		t.Fatalf("default start should be ~7 days ago, got %v", start)
	}
}
