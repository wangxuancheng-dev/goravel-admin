package admin

import (
	"fmt"
	nethttp "net/http"
	"strings"
	"time"

	"github.com/goravel/framework/contracts/http"

	"goravel/app/http/helpers"
	"goravel/app/http/response"
	"goravel/app/http/trans"
	"goravel/app/utils"
)

// parseOptionalTimeFromQuery reads an optional time param (query preferred, input fallback) as UTC.
func parseOptionalTimeFromQuery(ctx http.Context, paramName, invalidKey string) (time.Time, http.Response) {
	timeStr := helpers.GetTimeInputOrQueryParam(ctx, paramName)
	if timeStr == "" {
		return time.Time{}, nil
	}

	parsedTime, err := utils.ParseDateTime(timeStr)
	if err != nil {
		return time.Time{}, response.Error(ctx, nethttp.StatusBadRequest, invalidKey)
	}

	return parsedTime, nil
}

// validateTimeRangeResponse validates a time range and returns a unified error response on failure.
func validateTimeRangeResponse(ctx http.Context, startTime, endTime time.Time, maxMonths ...int) http.Response {
	valid, err := utils.ValidateTimeRange(startTime, endTime, maxMonths...)
	if valid {
		return nil
	}

	if timeRangeErr, ok := err.(*utils.TimeRangeError); ok {
		message := trans.Get(ctx, timeRangeErr.Key)
		if timeRangeErr.Params != nil {
			for key, value := range timeRangeErr.Params {
				placeholder := fmt.Sprintf("{%s}", key)
				message = strings.ReplaceAll(message, placeholder, fmt.Sprintf("%v", value))
			}
		}
		return response.Error(ctx, nethttp.StatusBadRequest, message)
	}

	return response.Error(ctx, nethttp.StatusBadRequest, err.Error())
}
