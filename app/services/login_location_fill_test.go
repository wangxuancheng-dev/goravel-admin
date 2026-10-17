package services

import "testing"

func TestUniqueIPs(t *testing.T) {
	got := uniqueIPs([]string{" 1.1.1.1 ", "", "1.1.1.1", "2.2.2.2", "3.3.3.3"}, 2)
	if len(got) != 2 || got[0] != "1.1.1.1" || got[1] != "2.2.2.2" {
		t.Fatalf("uniqueIPs = %#v", got)
	}
	if uniqueIPs([]string{"1.1.1.1"}, 0) != nil {
		t.Fatal("limit 0 should return nil")
	}
}
