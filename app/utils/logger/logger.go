package logger

import (
	"context"
	"fmt"

	"github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"

	"goravel/app/tenancyctx"
	"goravel/app/utils/traceid"
)

// DebugfHTTP logs a debug message and automatically attaches trace_id from the http context.
// Debug messages are only shown when APP_DEBUG=true
func DebugfHTTP(ctx http.Context, format string, args ...any) {
	if ctx == nil {
		facades.Log().Debugf(format, args...)
		return
	}

	facades.Log().Debugf(prependHTTP(ctx, format), args...)
}

// Debugf logs a debug message without any context.
// Debug messages are only shown when APP_DEBUG=true
func Debugf(format string, args ...any) {
	facades.Log().Debugf(format, args...)
}

// InfofHTTP logs an info message and automatically attaches trace_id from the http context.
func InfofHTTP(ctx http.Context, format string, args ...any) {
	if ctx == nil {
		facades.Log().Infof(format, args...)
		return
	}

	facades.Log().Infof(prependHTTP(ctx, format), args...)
}

// WarnfHTTP logs a warning and automatically attaches trace_id from the http context.
func WarnfHTTP(ctx http.Context, format string, args ...any) {
	if ctx == nil {
		facades.Log().Warningf(format, args...)
		return
	}

	facades.Log().Warningf(prependHTTP(ctx, format), args...)
}

// ErrorfHTTP logs an error and automatically attaches trace_id from the http context.
func ErrorfHTTP(ctx http.Context, format string, args ...any) {
	if ctx == nil {
		facades.Log().Errorf(format, args...)
		return
	}

	facades.Log().Errorf(prependHTTP(ctx, format), args...)
}

// ErrorfContext logs an error with a standard context's trace id (if available).
func ErrorfContext(ctx context.Context, format string, args ...any) {
	if ctx == nil {
		facades.Log().Errorf(format, args...)
		return
	}

	facades.Log().Errorf(prependContext(ctx, format), args...)
}

// Errorf logs an error without any context (fallback).
func Errorf(format string, args ...any) {
	facades.Log().Errorf(format, args...)
}

func prependHTTP(ctx http.Context, format string) string {
	if ctx == nil {
		return format
	}
	return prependContext(ctx, format)
}

func prependContext(ctx context.Context, format string) string {
	trace := ""
	if httpCtx, ok := ctx.(http.Context); ok {
		trace = traceid.FromHTTPContext(httpCtx)
	} else {
		trace = traceid.FromContext(ctx)
	}
	prefix := ""
	if trace != "" {
		prefix += fmt.Sprintf("[trace_id=%s] ", trace)
	}
	if code, ok := tenancyctx.CodeFrom(ctx); ok && code != "" {
		prefix += fmt.Sprintf("[tenant_code=%s] ", code)
	} else if id, ok := tenancyctx.IDFrom(ctx); ok && id > 0 {
		prefix += fmt.Sprintf("[tenant_id=%d] ", id)
	}
	if prefix == "" {
		return format
	}
	return prefix + format
}

func prependTrace(traceID, format string) string {
	if traceID == "" {
		return format
	}
	return fmt.Sprintf("[trace_id=%s] %s", traceID, format)
}
