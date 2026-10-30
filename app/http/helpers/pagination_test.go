package helpers

import "testing"

func TestListTooDeep(t *testing.T) {
	cases := []struct {
		page, pageSize int
		want           bool
	}{
		{1, 10, false},
		{1000, 10, false}, // last full page within MaxListRows
		{1001, 10, true},
		{100, 100, false},
		{101, 100, true},
	}
	for _, c := range cases {
		if got := ListTooDeep(c.page, c.pageSize); got != c.want {
			t.Errorf("ListTooDeep(%d, %d) = %v, want %v", c.page, c.pageSize, got, c.want)
		}
	}
}
