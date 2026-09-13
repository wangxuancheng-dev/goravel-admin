package services

import (
	"testing"
	"time"
)

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
