package admin

import (
	"net/http"
	"testing"
)

func TestBusinessErrorStatus(t *testing.T) {
	tests := []struct {
		code     string
		fallback int
		want     int
	}{
		{code: "order_not_found", fallback: 500, want: http.StatusNotFound},
		{code: "record_not_found", fallback: 500, want: http.StatusNotFound},
		{code: "payment_method_code_exists", fallback: 500, want: http.StatusBadRequest},
		{code: "order_no_required", fallback: 500, want: http.StatusBadRequest},
		{code: "time_range_exceeded", fallback: 500, want: http.StatusBadRequest},
		{code: "schedule_busy", fallback: 500, want: http.StatusConflict},
		{code: "too_many_requests", fallback: 500, want: http.StatusTooManyRequests},
		{code: "role_protected_delete", fallback: 500, want: http.StatusForbidden},
		{code: "must_change_password", fallback: 500, want: http.StatusForbidden},
		{code: "password_too_weak", fallback: 500, want: http.StatusBadRequest},
		{code: "sensitive_confirm_required", fallback: 500, want: http.StatusBadRequest},
		{code: "unauthorized", fallback: 500, want: http.StatusUnauthorized},
		{code: "create_failed", fallback: 400, want: http.StatusInternalServerError},
		{code: "export_failed", fallback: 400, want: http.StatusInternalServerError},
		{code: "custom_domain_code", fallback: 418, want: 418},
	}

	for _, tt := range tests {
		if got := businessErrorStatus(tt.code, tt.fallback); got != tt.want {
			t.Fatalf("businessErrorStatus(%q, %d) = %d, want %d", tt.code, tt.fallback, got, tt.want)
		}
	}
}
