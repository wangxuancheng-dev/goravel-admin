package notifications

import "testing"

func TestParsePresenceMember(t *testing.T) {
	tid, aid, ok := parsePresenceMember("abc:def|3|9")
	if !ok || tid != 3 || aid != 9 {
		t.Fatalf("got tid=%d aid=%d ok=%v", tid, aid, ok)
	}
	if _, _, ok := parsePresenceMember("bad"); ok {
		t.Fatal("expected parse failure")
	}
	if _, _, ok := parsePresenceMember("a|x|1"); ok {
		t.Fatal("expected bad tenant")
	}
}

func TestCountPresenceMembers(t *testing.T) {
	admins, conns := countPresenceMembers(nil)
	if admins != 0 || conns != 0 {
		t.Fatalf("empty: %d %d", admins, conns)
	}

	members := []string{
		"inst1:aa|1|10",
		"inst1:bb|1|10", // same admin, second tab
		"inst2:cc|1|11",
		"inst2:dd|2|10", // same admin id, different tenant
		"broken",
	}
	admins, conns = countPresenceMembers(members)
	if conns != 4 {
		t.Fatalf("connections want 4 got %d", conns)
	}
	if admins != 3 {
		t.Fatalf("admins want 3 got %d", admins)
	}
}

func TestPresenceMemberFormat(t *testing.T) {
	m := presenceMember("id1", 7, 8)
	tid, aid, ok := parsePresenceMember(m)
	if !ok || tid != 7 || aid != 8 {
		t.Fatalf("roundtrip failed: %s -> %d %d %v", m, tid, aid, ok)
	}
}
