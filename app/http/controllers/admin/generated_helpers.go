package admin

import (
	stdhttp "net/http"
	"strings"

	"github.com/goravel/framework/contracts/http"

	apperrors "goravel/app/errors"
	"goravel/app/http/response"
)

// ValidateGeneratedRequest validates a form request and returns a ready-to-send response on failure.
func ValidateGeneratedRequest(ctx http.Context, req http.FormRequest) http.Response {
	validationErrors, err := ctx.Request().ValidateRequest(req)
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, err.Error())
	}
	if validationErrors != nil {
		return response.ValidationError(ctx, http.StatusBadRequest, "validation_failed", validationErrors.All())
	}
	return nil
}

// HandleGeneratedServiceError maps BusinessError to HTTP responses and logs unexpected failures.
func HandleGeneratedServiceError(ctx http.Context, module string, status int, err error, attrs map[string]any) http.Response {
	if err == nil {
		return nil
	}
	if businessErr, ok := apperrors.GetBusinessError(err); ok {
		// Pass the BusinessError itself so response.Error can apply i18n + WithParams placeholders.
		return response.Error(ctx, businessErrorStatus(businessErr.Code, status), businessErr)
	}
	if status >= stdhttp.StatusInternalServerError {
		if attrs == nil {
			attrs = map[string]any{}
		}
		return response.ErrorWithLog(ctx, module, err, attrs)
	}
	return response.Error(ctx, status, err.Error())
}

func businessErrorStatus(code string, fallback int) int {
	switch {
	case code == "deep_pagination_exceeded" || code == "time_range_exceeded" || code == "start_time_after_end_time":
		return http.StatusBadRequest
	case code == "params_error" || code == "invalid_argument" || code == "validation_failed":
		return http.StatusBadRequest
	case strings.HasSuffix(code, "_required"):
		return http.StatusBadRequest
	case code == "record_not_found" || strings.HasSuffix(code, "_not_found"):
		return http.StatusNotFound
	case strings.HasSuffix(code, "_exists") || strings.HasSuffix(code, "_already_exists"):
		return http.StatusBadRequest
	case code == "password_encrypt_failed":
		return http.StatusInternalServerError
	case code == "too_many_requests" || code == "login_locked":
		return http.StatusTooManyRequests
	case code == "schedule_busy":
		return http.StatusConflict
	case code == "old_password_error" ||
		code == "google_code_invalid" ||
		code == "google_code_required" ||
		code == "google_authenticator_not_bound" ||
		code == "google_authenticator_already_bound":
		return http.StatusBadRequest
	case code == "token_refresh_failed":
		return http.StatusUnauthorized
	case strings.HasPrefix(code, "role_protected_") ||
		strings.HasPrefix(code, "admin_protected_") ||
		strings.HasPrefix(code, "admin_cannot_") ||
		code == "protected_admin":
		return http.StatusForbidden
	case strings.Contains(code, "_has_"):
		return http.StatusBadRequest
	case code == "account_disabled" || code == "forbidden":
		return http.StatusForbidden
	case code == "not_logged_in" || code == "username_or_password_error" || code == "unauthorized":
		return http.StatusUnauthorized
	case code == "query_failed" || code == "create_failed" || code == "update_failed" || code == "delete_failed" || code == "operation_failed":
		return http.StatusInternalServerError
	case strings.HasSuffix(code, "_failed"):
		return http.StatusInternalServerError
	default:
		return fallback
	}
}
