package admin

import (
	"errors"
	"testing"
	"time"

	"goravel/app/models"
)

func TestApplyAsyncExportFailure(t *testing.T) {
	if applyAsyncExportFailure(nil, errors.New("x")) {
		t.Fatal("nil record should return false")
	}
	rec := &models.Export{Status: models.ExportStatusProcessing}
	if applyAsyncExportFailure(rec, nil) {
		t.Fatal("nil error should return false")
	}
	if !applyAsyncExportFailure(rec, errors.New("queue down")) {
		t.Fatal("expected true")
	}
	if rec.Status != models.ExportStatusFailed {
		t.Fatalf("status=%d", rec.Status)
	}
	if rec.ErrorMsg != "queue down" {
		t.Fatalf("error_msg=%q", rec.ErrorMsg)
	}
}

func TestFormatExportTimestamp(t *testing.T) {
	if got := formatExportTimestamp(nil); got != "" {
		t.Fatalf("nil -> %q", got)
	}
	ts := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	got := formatExportTimestamp(ts)
	if got == "" {
		t.Fatal("time.Time should format non-empty")
	}
	gotPtr := formatExportTimestamp(&ts)
	if gotPtr == "" {
		t.Fatal("*time.Time should format non-empty")
	}
}
