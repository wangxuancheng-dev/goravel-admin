package tenantdb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// ErrUnknownConnection is returned by a Resolver for a name that is not a tenant
// connection, and by Registry.Conn when no Resolver can produce it.
var ErrUnknownConnection = errors.New("tenantdb: unknown tenant connection")

// Resolver rebuilds the Spec of a connection name on demand (e.g. after the
// connection was evicted while a job still carries its name in ctx).
type Resolver func(name string) (Spec, error)

// Options tune a Registry. Funcs are read on every registration, so config changes
// apply without a restart. Zero values disable the corresponding limit.
type Options struct {
	MaxConns    func() int           // max registered connections; oldest idle are evicted
	IdleTTL     func() time.Duration // evict connections unused for this long
	CloseGrace  time.Duration        // delay before closing an evicted / replaced pool
	PingTimeout time.Duration        // timeout of the ping after building a connection
}

// Registry holds the tenant connections of this process.
//
// Get / Conn on a registered name take only a read lock and an atomic store, so the
// request path never waits on a connection being built: building and pinging a new
// connection happen outside any lock, deduplicated per name.
type Registry struct {
	mu    sync.RWMutex
	conns map[string]*Conn

	group singleflight.Group

	optsMu   sync.RWMutex
	opts     Options
	resolver Resolver

	// build creates and pings a connection (swapped in unit tests).
	build func(ctx context.Context, name string, spec Spec) (*Conn, error)
	// closeLater schedules a delayed close (swapped in unit tests).
	closeLater func(d time.Duration, f func())
}

// NewRegistry returns an empty Registry that builds real connections.
func NewRegistry(opts Options) *Registry {
	return &Registry{
		conns:      map[string]*Conn{},
		opts:       opts,
		build:      Open,
		closeLater: func(d time.Duration, f func()) { time.AfterFunc(d, f) },
	}
}

// Default is the process-wide tenant connection registry.
var Default = NewRegistry(Options{CloseGrace: 2 * time.Minute, PingTimeout: 5 * time.Second})

// SetOptions replaces the options.
func (r *Registry) SetOptions(opts Options) {
	r.optsMu.Lock()
	defer r.optsMu.Unlock()
	r.opts = opts
}

// SetResolver installs the Resolver used by Conn on a miss.
func (r *Registry) SetResolver(resolver Resolver) {
	r.optsMu.Lock()
	defer r.optsMu.Unlock()
	r.resolver = resolver
}

func (r *Registry) options() (Options, Resolver) {
	r.optsMu.RLock()
	defer r.optsMu.RUnlock()
	return r.opts, r.resolver
}

// Get returns the registered connection, if any.
func (r *Registry) Get(name string) (*Conn, bool) {
	r.mu.RLock()
	c := r.conns[name]
	r.mu.RUnlock()
	if c == nil {
		return nil, false
	}
	c.touch()
	return c, true
}

// Conn returns the registered connection, rebuilding it through the Resolver when it
// is not registered. Names the Resolver does not know yield ErrUnknownConnection.
func (r *Registry) Conn(name string) (*Conn, error) {
	if c, ok := r.Get(name); ok {
		return c, nil
	}
	_, resolver := r.options()
	if resolver == nil {
		return nil, ErrUnknownConnection
	}
	spec, err := resolver(name)
	if err != nil {
		return nil, err
	}
	return r.Ensure(name, spec)
}

// Ensure returns the connection for name, building it when it is missing or was
// built from a different Spec (another host, database or credentials). A replaced
// connection is closed after the grace period, so in-flight users can finish.
func (r *Registry) Ensure(name string, spec Spec) (*Conn, error) {
	fp := spec.Fingerprint()
	if c, ok := r.Get(name); ok && c.fingerprint == fp {
		return c, nil
	}
	v, err, _ := r.group.Do(name+"\x00"+fp, func() (any, error) {
		if c, ok := r.Get(name); ok && c.fingerprint == fp {
			return c, nil
		}
		opts, _ := r.options()
		timeout := opts.PingTimeout
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		c, err := r.build(ctx, name, spec)
		if err != nil {
			return nil, err
		}

		r.mu.Lock()
		old := r.conns[name]
		r.conns[name] = c
		victims := r.evictLocked(name, opts)
		r.mu.Unlock()

		if old != nil {
			victims = append(victims, old)
		}
		r.retire(opts.CloseGrace, victims...)
		return c, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*Conn), nil
}

// Forget unregisters name and closes its pool now (e.g. before DROP DATABASE).
func (r *Registry) Forget(name string) {
	r.mu.Lock()
	c := r.conns[name]
	delete(r.conns, name)
	r.mu.Unlock()
	if c != nil {
		_ = c.Close()
	}
}

// Count returns how many connections are registered.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.conns)
}

// Names returns the registered connection names, sorted.
func (r *Registry) Names() []string {
	r.mu.RLock()
	names := make([]string, 0, len(r.conns))
	for name := range r.conns {
		names = append(names, name)
	}
	r.mu.RUnlock()
	sort.Strings(names)
	return names
}

// CloseAll unregisters and closes every connection (shutdown / tests).
func (r *Registry) CloseAll() {
	r.mu.Lock()
	all := r.conns
	r.conns = map[string]*Conn{}
	r.mu.Unlock()
	for _, c := range all {
		_ = c.Close()
	}
}

// evictLocked removes idle connections and, over MaxConns, the least recently used
// ones; protect is never evicted. Caller holds r.mu and closes the returned victims.
func (r *Registry) evictLocked(protect string, opts Options) []*Conn {
	var victims []*Conn
	now := time.Now()
	if opts.IdleTTL != nil {
		if ttl := opts.IdleTTL(); ttl > 0 {
			for name, c := range r.conns {
				if name != protect && now.Sub(c.idleSince()) >= ttl {
					victims = append(victims, c)
					delete(r.conns, name)
				}
			}
		}
	}
	if opts.MaxConns != nil {
		if limit := opts.MaxConns(); limit > 0 {
			for len(r.conns) > limit {
				oldest := ""
				var oldestAt time.Time
				for name, c := range r.conns {
					if name == protect {
						continue
					}
					if at := c.idleSince(); oldest == "" || at.Before(oldestAt) {
						oldest, oldestAt = name, at
					}
				}
				if oldest == "" {
					break
				}
				victims = append(victims, r.conns[oldest])
				delete(r.conns, oldest)
			}
		}
	}
	return victims
}

func (r *Registry) retire(grace time.Duration, victims ...*Conn) {
	for _, c := range victims {
		if c == nil {
			continue
		}
		c := c
		if grace <= 0 {
			_ = c.Close()
			continue
		}
		r.closeLater(grace, func() { _ = c.Close() })
	}
}

// String is for debugging.
func (r *Registry) String() string { return fmt.Sprintf("tenantdb.Registry%v", r.Names()) }
