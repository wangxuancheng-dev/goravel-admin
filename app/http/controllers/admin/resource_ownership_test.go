package admin

import "testing"

func TestCanAccessOwnedResource(t *testing.T) {
	cases := []struct {
		name                  string
		actor, owner, superID uint
		want                  bool
	}{
		{"owner", 5, 5, 1, true},
		{"super", 1, 9, 1, true},
		{"other", 3, 9, 1, false},
		{"zero actor", 0, 9, 1, false},
		{"super disabled", 1, 9, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := canAccessOwnedResource(tc.actor, tc.owner, tc.superID)
			if got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
