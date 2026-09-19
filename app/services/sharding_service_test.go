package services

import (
	"errors"
	"testing"
)

func TestIsShardingTableAlreadyExistsError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"other", errors.New("connection refused"), false},
		{"mysql 1050", errors.New("failed to create orders_202609 table: Error 1050 (42S01): Table 'orders_202609' already exists"), true},
		{"already exists plain", errors.New("table already exists"), true},
		{"duplicate table", errors.New("Duplicate table 'orders_202609'"), true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isShardingTableAlreadyExistsError(tc.err); got != tc.want {
				t.Fatalf("got %v want %v for %v", got, tc.want, tc.err)
			}
		})
	}
}
