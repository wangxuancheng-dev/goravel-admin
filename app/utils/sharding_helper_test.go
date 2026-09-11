package utils

import (
	"testing"
	"time"
)

func TestGetShardingTableNameUsesLayout(t *testing.T) {
	tableName := GetShardingTableName("orders", time.Date(2025, 7, 15, 0, 0, 0, 0, time.UTC))
	if tableName != "orders_202507" {
		t.Fatalf("expected orders_202507, got %s", tableName)
	}
}

func TestValidateTimeRangeWithinDefault(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	ok, err := ValidateTimeRange(start, end)
	if !ok || err != nil {
		t.Fatalf("expected valid range, got ok=%v err=%v", ok, err)
	}
}

func TestValidateTimeRangeExceedsLimit(t *testing.T) {
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, GetMaxTimeRangeMonths()+1, 0)
	ok, err := ValidateTimeRange(start, end)
	if ok || err == nil {
		t.Fatalf("expected time range error, got ok=%v err=%v", ok, err)
	}
	if trErr, ok := err.(*TimeRangeError); !ok || trErr.Key != "time_range_exceeded" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetShardingTableNamesMonthly(t *testing.T) {
	start := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 3, 2, 0, 0, 0, 0, time.UTC)
	names := GetShardingTableNames("orders", start, end)
	if len(names) != 3 {
		t.Fatalf("expected 3 tables, got %v", names)
	}
	if names[0] != "orders_202501" || names[2] != "orders_202503" {
		t.Fatalf("unexpected table names: %v", names)
	}
}

func TestValidateShardingDeepPagination(t *testing.T) {
	maxRows := GetMaxUnionLimitPerTable()
	if err := ValidateShardingDeepPagination(1, 10); err != nil {
		t.Fatalf("expected first page ok, got %v", err)
	}
	// page such that offset+pageSize exceeds max
	pageSize := 100
	page := maxRows/pageSize + 2
	if err := ValidateShardingDeepPagination(page, pageSize); err == nil {
		t.Fatalf("expected deep pagination error")
	}
	if max, ok := DeepPaginationMaxFromError(ValidateShardingDeepPagination(page, pageSize)); !ok || max != maxRows {
		t.Fatalf("expected max=%d from error, got %d ok=%v", maxRows, max, ok)
	}
}

func TestGetHashShardingTableName(t *testing.T) {
	name := GetHashShardingTableName("user_balance_logs", 5, 4)
	if name != "user_balance_logs_1" {
		t.Fatalf("expected user_balance_logs_1, got %s", name)
	}
}
