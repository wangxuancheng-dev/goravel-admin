package helpers

import (
	"testing"
)

func TestParseSortRejectsInjection(t *testing.T) {
	got := ParseSort("id;drop table users:desc,created_at:asc")
	if _, ok := got["id;drop table users"]; ok {
		t.Fatal("expected unsafe field to be rejected")
	}
	if got["created_at"] != "asc" {
		t.Fatalf("created_at want asc, got %q", got["created_at"])
	}
}

func TestParseSortAllowsSimpleFields(t *testing.T) {
	got := ParseSort("username:desc,id:asc")
	if got["username"] != "desc" || got["id"] != "asc" {
		t.Fatalf("unexpected parse result: %#v", got)
	}
}
