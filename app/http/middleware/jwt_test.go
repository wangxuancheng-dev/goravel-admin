package middleware

import (
	"errors"
	"fmt"
	"testing"

	frameworkerrors "github.com/goravel/framework/errors"

	apperrors "goravel/app/errors"
)

// Only a missing or expired token is the client's fault; anything else from the lookup is
// a server failure and must not turn into a 401 that logs the user out.
func TestIsExpectedUnauthorizedErr(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"not found", frameworkerrors.OrmRecordNotFound, true},
		{"wrapped not found", fmt.Errorf("find token: %w", frameworkerrors.OrmRecordNotFound), true},
		{"expired", apperrors.ErrInvalidArgument.WithMessage("token expired"), true},
		{"connection refused", errors.New("dial tcp 127.0.0.1:3306: connect: connection refused"), false},
		{"too many connections", errors.New("Error 1040: Too many connections"), false},
		{"nil", nil, false},
	}
	for _, c := range cases {
		if got := isExpectedUnauthorizedErr(c.err); got != c.want {
			t.Errorf("%s: isExpectedUnauthorizedErr(%v) = %v, want %v", c.name, c.err, got, c.want)
		}
	}
}
