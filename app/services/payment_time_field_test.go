package services

import (
	"testing"
	"time"
)

func TestPaymentTimeColumnAndShardLookback(t *testing.T) {
	if got := paymentTimeColumn(""); got != "created_at" {
		t.Fatalf("default column: %q", got)
	}
	if got := paymentTimeColumn("pay_time"); got != "pay_time" {
		t.Fatalf("pay_time column: %q", got)
	}

	start := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	s, e := paymentShardScanRange(PaymentFilters{StartTime: start, EndTime: end, TimeField: "pay_time"})
	if !s.Equal(start.AddDate(0, -1, 0)) || !e.Equal(end) {
		t.Fatalf("lookback range got %v .. %v", s, e)
	}
	s2, e2 := paymentShardScanRange(PaymentFilters{StartTime: start, EndTime: end, TimeField: "created_at"})
	if !s2.Equal(start) || !e2.Equal(end) {
		t.Fatalf("created_at range got %v .. %v", s2, e2)
	}
}
