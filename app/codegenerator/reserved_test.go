package codegenerator

import "testing"

func TestIsReservedTable(t *testing.T) {
	if !IsReservedTable("admins") {
		t.Fatal("admins should be reserved")
	}
	if !IsReservedTable("platform_foo") {
		t.Fatal("platform_ prefix should be reserved")
	}
	if !IsReservedTable("orders_202501") {
		t.Fatal("numeric shard suffix should be reserved")
	}
	if IsReservedTable("articles") {
		t.Fatal("articles should be allowed")
	}
	if IsReservedTable("quotes") || IsReservedTable("quote_items") {
		t.Fatal("master-detail demo tables should be allowed")
	}
}

func TestSplitCSV(t *testing.T) {
	got := SplitCSV(" a, b ,,c ")
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("unexpected SplitCSV result: %#v", got)
	}
}
