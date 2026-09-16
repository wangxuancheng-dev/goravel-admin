package routes

import (
	"net/http"
	"strings"

	contractshttp "github.com/goravel/framework/contracts/http"

	"goravel/app/errors"
	"goravel/app/facades"
	"goravel/app/http/middleware"
	"goravel/app/http/response"
)

func registerRouteFallback() {
	facades.Route().Fallback(func(ctx contractshttp.Context) contractshttp.Response {
		// OPTIONS has no registered route; if Cors Abort missed, still return 204 + CORS
		// instead of JSON 404 (browsers treat that as a failed preflight).
		if strings.EqualFold(ctx.Request().Method(), http.MethodOptions) {
			middleware.WriteCorsPreflightResponse(ctx)
			return ctx.Response().NoContent(http.StatusNoContent)
		}

		if strings.HasPrefix(ctx.Request().Path(), "/api/") {
			return respondRouteNotFound(ctx)
		}

		return ctx.Response().String(http.StatusNotFound, "404 page not found")
	})
}

// respondRouteNotFound returns unified JSON 404 payload for unmatched API routes.
func respondRouteNotFound(ctx contractshttp.Context) contractshttp.Response {
	return response.Error(ctx, http.StatusNotFound, errors.ErrRecordNotFound.Code)
}
