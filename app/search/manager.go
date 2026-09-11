package search

import (
	"fmt"
	"sync"

	"github.com/goravel/framework/facades"

	"goravel/app/binding"
)

var (
	resolveMu   sync.Mutex
	resolveFn   func() (Engine, error)
	testEngine  Engine
	useTestEng  bool
)

// SetResolver 由 SearchServiceProvider 注册引擎工厂。
func SetResolver(fn func() (Engine, error)) {
	resolveMu.Lock()
	defer resolveMu.Unlock()
	resolveFn = fn
}

// SetEngineForTest 测试注入。
func SetEngineForTest(e Engine) {
	resolveMu.Lock()
	defer resolveMu.Unlock()
	testEngine = e
	useTestEng = e != nil
}

// Resolve 从容器或工厂解析当前 Engine。
func Resolve() (Engine, error) {
	resolveMu.Lock()
	if useTestEng {
		e := testEngine
		resolveMu.Unlock()
		return e, nil
	}
	fn := resolveFn
	resolveMu.Unlock()

	if raw, err := facades.App().Make(binding.SearchEngine); err == nil && raw != nil {
		if e, ok := raw.(Engine); ok && e != nil {
			return e, nil
		}
	}
	if fn != nil {
		return fn()
	}
	return nil, fmt.Errorf("search engine not registered")
}

// MustResolve 解析失败时返回 null 引擎（不 panic，便于降级）。
func MustResolve() Engine {
	e, err := Resolve()
	if err != nil || e == nil {
		return nullEngine{}
	}
	return e
}
