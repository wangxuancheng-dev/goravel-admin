package middleware

import (
	"bytes"
	"strings"
	"testing"
)

func TestOperationErrorMessage(t *testing.T) {
	cases := []struct {
		name string
		body *bytes.Buffer
		want string
	}{
		{"nil", nil, ""},
		{"empty", &bytes.Buffer{}, ""},
		{"not json", bytes.NewBufferString("oops"), ""},
		{"code and message", bytes.NewBufferString(`{"code":400,"error_code":"password_missing_chars","message":"密码还需包含：特殊符号"}`), "password_missing_chars: 密码还需包含：特殊符号"},
		{"message only", bytes.NewBufferString(`{"code":500,"message":"boom"}`), "boom"},
	}
	for _, c := range cases {
		if got := operationErrorMessage(c.body); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}

	long := bytes.NewBufferString(`{"message":"` + strings.Repeat("错", 600) + `"}`)
	if got := []rune(operationErrorMessage(long)); len(got) != 500 {
		t.Errorf("long message should be truncated to 500 runes, got %d", len(got))
	}
}
