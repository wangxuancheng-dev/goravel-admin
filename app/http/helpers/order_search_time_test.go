package helpers

import (
	"testing"
	"time"
)

func TestParseOrderSearchCreatedRange_EmptyUsesDBDefaultWindow(t *testing.T) {
	before := time.Now()
	out, field, key := ParseOrderSearchCreatedRange("", "")
	after := time.Now()

	if field != "" || key != "" {
		t.Fatalf("expected no validation error, got field=%q key=%q", field, key)
	}
	if out.IndexGTE != nil || out.IndexLTE != nil {
		t.Fatal("expected index bounds to be nil when both inputs are empty")
	}
	if out.DBEnd.Before(before.Add(-time.Second)) || out.DBEnd.After(after.Add(time.Second)) {
		t.Fatalf("unexpected DBEnd: %v", out.DBEnd)
	}
	if out.DBStart.After(out.DBEnd) {
		t.Fatal("DBStart must not be after DBEnd")
	}
}

func TestParseOrderSearchCreatedRange_InvalidStart(t *testing.T) {
	_, field, key := ParseOrderSearchCreatedRange("not-a-date", "")
	if field != "created_from" || key != "validation.datetime.start_time_invalid" {
		t.Fatalf("got field=%q key=%q", field, key)
	}
}

func TestParseOrderSearchCreatedRange_InvalidEnd(t *testing.T) {
	_, field, key := ParseOrderSearchCreatedRange("", "bad-end")
	if field != "created_to" || key != "validation.datetime.end_time_invalid" {
		t.Fatalf("got field=%q key=%q", field, key)
	}
}

func TestParseOrderSearchCreatedRange_InvertedRange(t *testing.T) {
	_, field, key := ParseOrderSearchCreatedRange("2026-07-10", "2026-07-01")
	if field != "created_to" || key != "validation.range.time_inverted" {
		t.Fatalf("got field=%q key=%q", field, key)
	}
}

func TestParseOrderSearchCreatedRange_ValidDateOnly(t *testing.T) {
	out, field, key := ParseOrderSearchCreatedRange("2026-07-01", "2026-07-10")
	if field != "" || key != "" {
		t.Fatalf("expected no validation error, got field=%q key=%q", field, key)
	}
	if out.IndexGTE == nil || out.IndexLTE == nil {
		t.Fatal("expected index bounds to be set")
	}
	if *out.IndexGTE != "2026-07-01 00:00:00" {
		t.Fatalf("unexpected IndexGTE: %s", *out.IndexGTE)
	}
	if *out.IndexLTE != "2026-07-10 23:59:59" {
		t.Fatalf("unexpected IndexLTE: %s", *out.IndexLTE)
	}
}
