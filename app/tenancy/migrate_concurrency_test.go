package tenancy

import "testing"

func TestClampMigrateConcurrency(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want int
	}{
		{0, DefaultMigrateConcurrency},
		{-1, DefaultMigrateConcurrency},
		{1, 1},
		{2, 2},
		{50, 50},
		{MaxMigrateConcurrency, MaxMigrateConcurrency},
		{MaxMigrateConcurrency + 1, MaxMigrateConcurrency},
	}
	for _, tc := range cases {
		if got := ClampMigrateConcurrency(tc.in); got != tc.want {
			t.Fatalf("ClampMigrateConcurrency(%d)=%d want %d", tc.in, got, tc.want)
		}
	}
}
