package providers

import "testing"

func TestIsBadRequestBodyPanicWithStack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		msg   string
		stack string
		want  bool
	}{
		{
			name: "multipart parse error message",
			msg:  `parse multipart form error: malformed MIME header: missing colon: "------WebKitFormBoundaryx123--"`,
			want: true,
		},
		{
			name: "malformed mime header",
			msg:  "malformed MIME header: missing colon",
			want: true,
		},
		{
			name:  "nil pointer in getHttpBody",
			msg:   "runtime error: invalid memory address or nil pointer dereference",
			stack: "goravel/gin@v1.18.0/context_request.go:660\ngin.getHttpBody\ngin.NewContextRequest",
			want:  true,
		},
		{
			name:  "nil pointer in app code is not client error",
			msg:   "runtime error: invalid memory address or nil pointer dereference",
			stack: "goravel/app/services/foo.go:12\nservices.(*Foo).Bar",
			want:  false,
		},
		{
			name: "unrelated panic message",
			msg:  "something went wrong",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := isBadRequestBodyPanicWithStack(tt.msg, tt.stack)
			if got != tt.want {
				t.Fatalf("isBadRequestBodyPanicWithStack(%q) = %v, want %v", tt.msg, got, tt.want)
			}
		})
	}
}

func TestLoginScopeFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path string
		want string
	}{
		{"/api/platform/login", "platform"},
		{"/api/admin/login", "admin"},
		{"/api/user/login", "user"},
		{"/api/user/register", "user"},
		{"/other", "other"},
		{"", "other"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			t.Parallel()
			if got := loginScopeFromPath(tt.path); got != tt.want {
				t.Fatalf("loginScopeFromPath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestBuildLoginRateLimitKey(t *testing.T) {
	t.Parallel()

	platformKey := buildLoginRateLimitKey("1.2.3.4", "platform", "", "admin")
	tenantKey := buildLoginRateLimitKey("1.2.3.4", "admin", "acme", "admin")
	tenantNoHint := buildLoginRateLimitKey("1.2.3.4", "admin", "", "admin")

	if platformKey != "1.2.3.4:login:platform:admin" {
		t.Fatalf("platform key = %q", platformKey)
	}
	if tenantKey != "1.2.3.4:login:admin:acme:admin" {
		t.Fatalf("tenant key = %q", tenantKey)
	}
	if platformKey == tenantNoHint {
		t.Fatal("platform and tenant-admin login must not share the same rate-limit bucket for the same username")
	}
}
