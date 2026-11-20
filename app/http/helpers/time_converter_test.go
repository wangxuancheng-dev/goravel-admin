package helpers

import "testing"

func TestIsTimeField(t *testing.T) {
	whitelist := map[string]struct{}{
		"created_at": {},
		"updatedat":  {},
	}

	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		{name: "snake case", field: "created_at", expected: true},
		{name: "camel case", field: "UpdatedAt", expected: true},
		{name: "non time field", field: "username", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if actual := isTimeField(tt.field, whitelist); actual != tt.expected {
				t.Fatalf("isTimeField(%q) = %v, expected %v", tt.field, actual, tt.expected)
			}
		})
	}
}

func TestIsTimeFieldSuffixAt(t *testing.T) {
	plain := map[string]struct{}{"created_at": {}}
	if isTimeField("last_op_at", plain) {
		t.Fatal("last_op_at must not match without the *_at rule")
	}
	withSuffix := map[string]struct{}{"created_at": {}, timeFieldSuffixAt: {}}
	for _, f := range []string{"last_op_at", "Health_Checked_At", "subscription_ends_at", "created_at"} {
		if !isTimeField(f, withSuffix) {
			t.Fatalf("%s must match the *_at rule", f)
		}
	}
	for _, f := range []string{"status", "format", "chat"} {
		if isTimeField(f, withSuffix) {
			t.Fatalf("%s must not match", f)
		}
	}
}

func TestConvertTimesInMap_IgnoreNonTimeFields(t *testing.T) {
	data := map[string]any{
		"id":   1,
		"name": "admin",
		"meta": map[string]any{
			"status": "active",
		},
	}

	whitelist := map[string]struct{}{
		"created_at": {},
	}

	converted, ok := convertTimesInMap(data, "UTC", whitelist).(map[string]any)
	if !ok {
		t.Fatalf("expected map result, got %T", converted)
	}

	if converted["name"] != "admin" {
		t.Fatalf("expected unchanged name field, got %#v", converted["name"])
	}

	meta, ok := converted["meta"].(map[string]any)
	if !ok {
		t.Fatalf("expected nested map result, got %T", converted["meta"])
	}

	if meta["status"] != "active" {
		t.Fatalf("expected unchanged nested field, got %#v", meta["status"])
	}
}
