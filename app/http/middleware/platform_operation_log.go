package middleware

import (
	"encoding/json"
	"slices"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/http/helpers"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
)

func shouldPersistPlatformOperationLog(method, path string) bool {
	allowed := []string{"POST", "PUT", "PATCH", "DELETE"}
	if !slices.Contains(allowed, method) {
		return false
	}
	excluded := []string{
		"/api/platform/login",
		"/api/platform/login/captcha",
		"/api/platform/info",
		"/api/platform/logout",
	}
	if slices.Contains(excluded, path) {
		return false
	}
	return strings.HasPrefix(path, "/api/platform/")
}

// PlatformOperationLog records platform console write operations to landlord DB.
func PlatformOperationLog() http.Middleware {
	return newMiddleware("platform_operation_log", func(ctx http.Context) {
		startTime := time.Now()
		method := ctx.Request().Method()
		path := ctx.Request().Path()
		ip := helpers.GetRealIP(ctx)
		userAgent := ctx.Request().Header("User-Agent", "")

		var requestBody string
		if method == "POST" || method == "PUT" || method == "PATCH" || method == "DELETE" {
			inputs := make(map[string]any)
			for key, value := range ctx.Request().All() {
				if utils.IsSensitiveField(key) {
					inputs[key] = "***"
				} else {
					inputs[key] = value
				}
			}
			if data, err := json.Marshal(inputs); err == nil {
				requestBody = string(data)
			}
		}

		var adminID uint
		var username string
		if admin, ok := ctx.Value("platform_admin").(models.PlatformAdmin); ok {
			adminID = admin.ID
			username = admin.Username
		}

		ctx.Request().Next()

		if !shouldPersistPlatformOperationLog(method, path) {
			return
		}

		if adminID == 0 {
			if admin, ok := ctx.Value("platform_admin").(models.PlatformAdmin); ok {
				adminID = admin.ID
				username = admin.Username
			}
		}

		duration := int(time.Since(startTime).Milliseconds())
		row := &models.PlatformOperationLog{
			AdminID:   adminID,
			Username:  username,
			Method:    method,
			Path:      path,
			Title:     services.PlatformOperationTitle(method, path),
			IP:        ip,
			UserAgent: userAgent,
			Request:   requestBody,
			Status:    1,
			Duration:  duration,
		}
		if err := services.CreatePlatformOperationLog(row); err != nil {
			facades.Log().Errorf("Failed to create platform operation log: %v", err)
		}
	})
}
