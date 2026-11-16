package tenancyctx

import (
	"context"
	"runtime/debug"

	"github.com/goravel/framework/facades"
)

// Go runs fn on a new goroutine — the one way to start background work from a request
// or job (tenantlint rejects a `go` statement that captures a ctx).
//
// fn receives detach(ctx): the same tenant routing (OrmQuery / cache keys / storage
// prefix still hit ctx's tenant) but not ctx's cancellation, which ends with the HTTP
// request. Values fn needs from an http.Context (admin id, headers, ...) must be read
// before calling Go: Goravel recycles the http.Context once the request returns.
//
// A panic in fn is recovered and logged with name, so it cannot take the process down.
func Go(ctx context.Context, name string, fn func(ctx context.Context)) {
	if fn == nil {
		return
	}
	detached := detach(ctx)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logPanic(name, r)
			}
		}()
		fn(detached)
	}()
}

func logPanic(name string, r any) {
	defer func() { _ = recover() }() // no logger (bare unit test): drop it
	if l := facades.Log(); l != nil {
		l.Errorf("tenancyctx.Go %s: panic: %v\n%s", name, r, debug.Stack())
	}
}
