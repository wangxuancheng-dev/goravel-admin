package meilisearch

import (
	"testing"

	"goravel/app/search"
)

func TestBuildMeiliFilter(t *testing.T) {
	got := buildMeiliFilter([]search.Filter{
		{Field: "user_id", Op: "term", Value: 9},
		{Field: "status", Op: "term", Value: "paid"},
		{Field: "amount", Op: "gte", Value: 10.5},
		{Field: "created_at", Op: "lte", Value: "2026-01-01 00:00:00"},
	})
	want := `user_id = 9 AND status = "paid" AND amount >= 10.5 AND created_at <= "2026-01-01 00:00:00"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestMeiliLiteralEscapesQuotes(t *testing.T) {
	if got := meiliLiteral(`a"b`); got != `"a\"b"` {
		t.Fatalf("got %q", got)
	}
}

func TestQueryReadyFalseWithoutClient(t *testing.T) {
	e := &Engine{}
	if e.QueryReady() {
		t.Fatal("nil client should not be query ready")
	}
}
