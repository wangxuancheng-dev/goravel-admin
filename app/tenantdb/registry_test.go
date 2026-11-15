package tenantdb

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeBuilds struct {
	calls  atomic.Int32
	closed sync.Map // *Conn -> struct{}
	gate   map[string]chan struct{}
	fail   atomic.Bool
}

func (f *fakeBuilds) build(_ context.Context, name string, spec Spec) (*Conn, error) {
	f.calls.Add(1)
	if g := f.gate[name]; g != nil {
		<-g
	}
	if f.fail.Load() {
		return nil, errors.New("boom")
	}
	c := &Conn{name: name, fingerprint: spec.Fingerprint(), database: spec.Database}
	c.closeFn = func() error { f.closed.Store(c, struct{}{}); return nil }
	c.touch()
	return c, nil
}

func (f *fakeBuilds) isClosed(c *Conn) bool { _, ok := f.closed.Load(c); return ok }

type delayed struct {
	mu  sync.Mutex
	fns []func()
}

func (d *delayed) later(_ time.Duration, f func()) {
	d.mu.Lock()
	d.fns = append(d.fns, f)
	d.mu.Unlock()
}
func (d *delayed) fire() {
	d.mu.Lock()
	fns := d.fns
	d.fns = nil
	d.mu.Unlock()
	for _, f := range fns {
		f()
	}
}

func newTestRegistry(opts Options) (*Registry, *fakeBuilds, *delayed) {
	r := NewRegistry(opts)
	fb := &fakeBuilds{gate: map[string]chan struct{}{}}
	d := &delayed{}
	r.build = fb.build
	r.closeLater = d.later
	return r, fb, d
}

func spec(db string) Spec { return Spec{Driver: DriverMySQL, Database: db} }

func TestEnsureBuildsOncePerNameUnderConcurrency(t *testing.T) {
	r, fb, _ := newTestRegistry(Options{})
	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := r.Ensure("tenant_1", spec("t1")); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if n := fb.calls.Load(); n != 1 {
		t.Fatalf("build called %d times, want 1", n)
	}
}

// A slow (or unreachable) tenant being built must not block lookups of others.
func TestBuildingOneTenantDoesNotBlockOthers(t *testing.T) {
	r, fb, _ := newTestRegistry(Options{})
	if _, err := r.Ensure("tenant_1", spec("t1")); err != nil {
		t.Fatal(err)
	}
	fb.gate["tenant_2"] = make(chan struct{})
	go func() { _, _ = r.Ensure("tenant_2", spec("t2")) }()
	time.Sleep(20 * time.Millisecond) // tenant_2 is now blocked inside build

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			if _, err := r.Conn("tenant_1"); err != nil {
				t.Error(err)
			}
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("lookups of tenant_1 blocked while tenant_2 was building")
	}
	close(fb.gate["tenant_2"])
}

func TestSpecChangeReplacesAndRetiresOldPool(t *testing.T) {
	r, fb, d := newTestRegistry(Options{CloseGrace: time.Minute})
	old, _ := r.Ensure("tenant_1", spec("t1"))
	same, _ := r.Ensure("tenant_1", spec("t1"))
	if same != old {
		t.Fatal("same spec must reuse the connection")
	}
	repl, err := r.Ensure("tenant_1", spec("t1_new"))
	if err != nil || repl == old || repl.DatabaseName() != "t1_new" {
		t.Fatalf("spec change must rebuild: %v", err)
	}
	if fb.isClosed(old) {
		t.Fatal("old pool must stay open during the grace period")
	}
	d.fire()
	if !fb.isClosed(old) || fb.isClosed(repl) {
		t.Fatal("old pool must close after the grace period, new one must stay open")
	}
}

func TestForgetClosesImmediately(t *testing.T) {
	r, fb, _ := newTestRegistry(Options{CloseGrace: time.Minute})
	c, _ := r.Ensure("tenant_1", spec("t1"))
	r.Forget("tenant_1")
	if !fb.isClosed(c) {
		t.Fatal("Forget must close the pool")
	}
	if _, ok := r.Get("tenant_1"); ok {
		t.Fatal("Forget must unregister")
	}
}

func TestEvictionByMaxAndIdle(t *testing.T) {
	maxConns := 2
	ttl := time.Duration(0)
	r, fb, d := newTestRegistry(Options{
		MaxConns:   func() int { return maxConns },
		IdleTTL:    func() time.Duration { return ttl },
		CloseGrace: time.Minute,
	})
	c1, _ := r.Ensure("tenant_1", spec("t1"))
	c1.lastUsed.Store(time.Now().Add(-time.Hour).UnixNano())
	c2, _ := r.Ensure("tenant_2", spec("t2"))
	c3, _ := r.Ensure("tenant_3", spec("t3"))
	if got := r.Names(); len(got) != 2 || got[0] != "tenant_2" || got[1] != "tenant_3" {
		t.Fatalf("max eviction kept %v, want [tenant_2 tenant_3]", got)
	}
	d.fire()
	if !fb.isClosed(c1) || fb.isClosed(c2) || fb.isClosed(c3) {
		t.Fatal("only the evicted least-recently-used pool may close")
	}

	maxConns, ttl = 0, time.Minute
	c2.lastUsed.Store(time.Now().Add(-time.Hour).UnixNano())
	if _, err := r.Ensure("tenant_4", spec("t4")); err != nil {
		t.Fatal(err)
	}
	if got := r.Names(); len(got) != 2 || got[0] != "tenant_3" || got[1] != "tenant_4" {
		t.Fatalf("idle eviction kept %v, want [tenant_3 tenant_4]", got)
	}
}

func TestConnResolvesMissesAndRejectsUnknown(t *testing.T) {
	r, fb, _ := newTestRegistry(Options{})
	if _, err := r.Conn("tenant_9"); !errors.Is(err, ErrUnknownConnection) {
		t.Fatalf("without resolver: %v", err)
	}
	r.SetResolver(func(name string) (Spec, error) {
		if name != "tenant_9" {
			return Spec{}, ErrUnknownConnection
		}
		return spec("t9"), nil
	})
	c, err := r.Conn("tenant_9")
	if err != nil || c.DatabaseName() != "t9" {
		t.Fatalf("resolve: %v", err)
	}
	if _, err := r.Conn("mysql"); !errors.Is(err, ErrUnknownConnection) {
		t.Fatalf("non-tenant name: %v", err)
	}
	if fb.calls.Load() != 1 {
		t.Fatalf("build calls = %d, want 1", fb.calls.Load())
	}
}

func TestFailedBuildIsNotCached(t *testing.T) {
	r, fb, _ := newTestRegistry(Options{})
	fb.fail.Store(true)
	if _, err := r.Ensure("tenant_1", spec("t1")); err == nil {
		t.Fatal("want build error")
	}
	fb.fail.Store(false)
	if _, err := r.Ensure("tenant_1", spec("t1")); err != nil {
		t.Fatalf("retry after failure: %v", err)
	}
}

func TestSpecValidateAndFingerprint(t *testing.T) {
	if err := (Spec{Driver: "sqlite", Database: "x"}).Validate(); err == nil {
		t.Fatal("unsupported driver must fail")
	}
	if err := (Spec{Driver: DriverMySQL}).Validate(); err == nil {
		t.Fatal("empty database must fail")
	}
	a := Spec{Driver: DriverMySQL, Database: "t1", Password: "x", Pool: PoolLimits{MaxOpenConns: 5}}
	b := a
	b.Pool.MaxOpenConns = 50
	if a.Fingerprint() != b.Fingerprint() {
		t.Fatal("pool limits must not change the fingerprint")
	}
	b.Password = "y"
	if a.Fingerprint() == b.Fingerprint() {
		t.Fatal("credentials must change the fingerprint")
	}
}
