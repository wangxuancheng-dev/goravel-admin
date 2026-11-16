package facades

import (
	"context"
	"database/sql"
	"errors"

	"github.com/goravel/framework/contracts/database/db"
	"github.com/goravel/framework/contracts/database/orm"
)

// unavailableQuery stands in for the query of a connection that could not be built,
// so OrmQuery / PlatformOrmQuery never return nil: chained calls return the query
// itself and every call that runs SQL returns err. Callers get an error through their
// normal error handling instead of a nil-pointer panic in the middle of a chain
// (OrmQuery(ctx).Where(...).First(...) appears in hundreds of places).
type unavailableQuery struct {
	ctx context.Context
	err error
}

func newUnavailableQuery(ctx context.Context, err error) orm.Query {
	if ctx == nil {
		ctx = context.Background()
	}
	return &unavailableQuery{ctx: ctx, err: err}
}

// QueryUnavailable reports whether q stands in for a connection that could not be
// built (see OrmQuery), with the reason.
func QueryUnavailable(q orm.Query) (bool, error) {
	if q == nil {
		return true, errors.New("orm query unavailable")
	}
	if u, ok := q.(*unavailableQuery); ok {
		return true, u.err
	}
	return false, nil
}

var _ orm.Query = (*unavailableQuery)(nil)

// Calls that run SQL or need the connection: return the error.

func (q *unavailableQuery) Association(string) orm.Association            { return unavailableAssociation{q.err} }
func (q *unavailableQuery) Begin() (orm.Query, error)                     { return nil, q.err }
func (q *unavailableQuery) BeginTransaction() (orm.Query, error)          { return nil, q.err }
func (q *unavailableQuery) Commit() error                                 { return q.err }
func (q *unavailableQuery) Count() (int64, error)                         { return 0, q.err }
func (q *unavailableQuery) Create(any) error                              { return q.err }
func (q *unavailableQuery) DB() (*sql.DB, error)                          { return nil, q.err }
func (q *unavailableQuery) Delete(...any) (*db.Result, error)             { return nil, q.err }
func (q *unavailableQuery) Exec(string, ...any) (*db.Result, error)       { return nil, q.err }
func (q *unavailableQuery) Exists() (bool, error)                         { return false, q.err }
func (q *unavailableQuery) Find(any, ...any) error                        { return q.err }
func (q *unavailableQuery) FindOrFail(any, ...any) error                  { return q.err }
func (q *unavailableQuery) First(any) error                               { return q.err }
func (q *unavailableQuery) FirstOr(any, func() error) error               { return q.err }
func (q *unavailableQuery) FirstOrCreate(any, ...any) error               { return q.err }
func (q *unavailableQuery) FirstOrFail(any) error                         { return q.err }
func (q *unavailableQuery) FirstOrNew(any, any, ...any) error             { return q.err }
func (q *unavailableQuery) ForceDelete(...any) (*db.Result, error)        { return nil, q.err }
func (q *unavailableQuery) Get(any) error                                 { return q.err }
func (q *unavailableQuery) Load(any, string, ...any) error                { return q.err }
func (q *unavailableQuery) LoadMissing(any, string, ...any) error         { return q.err }
func (q *unavailableQuery) Paginate(int, int, any, *int64) error          { return q.err }
func (q *unavailableQuery) Pluck(string, any) error                       { return q.err }
func (q *unavailableQuery) Restore(...any) (*db.Result, error)            { return nil, q.err }
func (q *unavailableQuery) Rollback() error                               { return q.err }
func (q *unavailableQuery) Save(any) error                                { return q.err }
func (q *unavailableQuery) SaveQuietly(any) error                         { return q.err }
func (q *unavailableQuery) Scan(any) error                                { return q.err }
func (q *unavailableQuery) Sum(string, any) error                         { return q.err }
func (q *unavailableQuery) Avg(string, any) error                         { return q.err }
func (q *unavailableQuery) Min(string, any) error                         { return q.err }
func (q *unavailableQuery) Max(string, any) error                         { return q.err }
func (q *unavailableQuery) Update(any, ...any) (*db.Result, error)        { return nil, q.err }
func (q *unavailableQuery) UpdateOrCreate(any, any, any) error            { return q.err }
func (q *unavailableQuery) ToSql() orm.ToSql                              { return unavailableToSql{} }
func (q *unavailableQuery) ToRawSql() orm.ToSql                           { return unavailableToSql{} }
func (q *unavailableQuery) Context() context.Context                      { return q.ctx }
func (q *unavailableQuery) Driver() string                                { return "" }
func (q *unavailableQuery) InTransaction() bool                           { return false }
func (q *unavailableQuery) Scopes(...func(orm.Query) orm.Query) orm.Query { return q }

// Cursor yields nothing (a closed channel); the error surfaces on any other call.
func (q *unavailableQuery) Cursor() chan db.Row {
	ch := make(chan db.Row)
	close(ch)
	return ch
}

// Builder calls: keep the chain going.

func (q *unavailableQuery) Distinct(...string) orm.Query                   { return q }
func (q *unavailableQuery) Group(string) orm.Query                         { return q }
func (q *unavailableQuery) GroupBy(...string) orm.Query                    { return q }
func (q *unavailableQuery) Having(any, ...any) orm.Query                   { return q }
func (q *unavailableQuery) InRandomOrder() orm.Query                       { return q }
func (q *unavailableQuery) Join(string, ...any) orm.Query                  { return q }
func (q *unavailableQuery) Limit(int) orm.Query                            { return q }
func (q *unavailableQuery) LockForUpdate() orm.Query                       { return q }
func (q *unavailableQuery) Model(any) orm.Query                            { return q }
func (q *unavailableQuery) Offset(int) orm.Query                           { return q }
func (q *unavailableQuery) Omit(...string) orm.Query                       { return q }
func (q *unavailableQuery) Order(any) orm.Query                            { return q }
func (q *unavailableQuery) OrderBy(string, ...string) orm.Query            { return q }
func (q *unavailableQuery) OrderByDesc(string) orm.Query                   { return q }
func (q *unavailableQuery) OrderByRaw(string) orm.Query                    { return q }
func (q *unavailableQuery) OrWhere(any, ...any) orm.Query                  { return q }
func (q *unavailableQuery) OrWhereBetween(string, any, any) orm.Query      { return q }
func (q *unavailableQuery) OrWhereIn(string, []any) orm.Query              { return q }
func (q *unavailableQuery) OrWhereJsonContains(string, any) orm.Query      { return q }
func (q *unavailableQuery) OrWhereJsonContainsKey(string) orm.Query        { return q }
func (q *unavailableQuery) OrWhereJsonDoesntContain(string, any) orm.Query { return q }
func (q *unavailableQuery) OrWhereJsonDoesntContainKey(string) orm.Query   { return q }
func (q *unavailableQuery) OrWhereJsonLength(string, int) orm.Query        { return q }
func (q *unavailableQuery) OrWhereNotBetween(string, any, any) orm.Query   { return q }
func (q *unavailableQuery) OrWhereNotIn(string, []any) orm.Query           { return q }
func (q *unavailableQuery) OrWhereNull(string) orm.Query                   { return q }
func (q *unavailableQuery) Raw(string, ...any) orm.Query                   { return q }
func (q *unavailableQuery) Select(...string) orm.Query                     { return q }
func (q *unavailableQuery) SelectRaw(any, ...any) orm.Query                { return q }
func (q *unavailableQuery) SharedLock() orm.Query                          { return q }
func (q *unavailableQuery) Table(string, ...any) orm.Query                 { return q }
func (q *unavailableQuery) Where(any, ...any) orm.Query                    { return q }
func (q *unavailableQuery) WhereAll([]string, ...any) orm.Query            { return q }
func (q *unavailableQuery) WhereAny([]string, ...any) orm.Query            { return q }
func (q *unavailableQuery) WhereBetween(string, any, any) orm.Query        { return q }
func (q *unavailableQuery) WhereIn(string, []any) orm.Query                { return q }
func (q *unavailableQuery) WhereJsonContains(string, any) orm.Query        { return q }
func (q *unavailableQuery) WhereJsonContainsKey(string) orm.Query          { return q }
func (q *unavailableQuery) WhereJsonDoesntContain(string, any) orm.Query   { return q }
func (q *unavailableQuery) WhereJsonDoesntContainKey(string) orm.Query     { return q }
func (q *unavailableQuery) WhereJsonLength(string, int) orm.Query          { return q }
func (q *unavailableQuery) WhereNone([]string, ...any) orm.Query           { return q }
func (q *unavailableQuery) WhereNotBetween(string, any, any) orm.Query     { return q }
func (q *unavailableQuery) WhereNotIn(string, []any) orm.Query             { return q }
func (q *unavailableQuery) WhereNotNull(string) orm.Query                  { return q }
func (q *unavailableQuery) WhereNull(string) orm.Query                     { return q }
func (q *unavailableQuery) WithoutEvents() orm.Query                       { return q }
func (q *unavailableQuery) WithoutGlobalScopes(...string) orm.Query        { return q }
func (q *unavailableQuery) WithTrashed() orm.Query                         { return q }
func (q *unavailableQuery) With(string, ...any) orm.Query                  { return q }

type unavailableAssociation struct{ err error }

func (a unavailableAssociation) Find(any, ...any) error { return a.err }
func (a unavailableAssociation) Append(...any) error    { return a.err }
func (a unavailableAssociation) Replace(...any) error   { return a.err }
func (a unavailableAssociation) Delete(...any) error    { return a.err }
func (a unavailableAssociation) Clear() error           { return a.err }
func (a unavailableAssociation) Count() int64           { return 0 }

// unavailableToSql renders nothing: there is no connection to render SQL for.
type unavailableToSql struct{}

func (unavailableToSql) Count() string             { return "" }
func (unavailableToSql) Create(any) string         { return "" }
func (unavailableToSql) Delete(...any) string      { return "" }
func (unavailableToSql) Find(any, ...any) string   { return "" }
func (unavailableToSql) First(any) string          { return "" }
func (unavailableToSql) ForceDelete(...any) string { return "" }
func (unavailableToSql) Get(any) string            { return "" }
func (unavailableToSql) Pluck(string, any) string  { return "" }
func (unavailableToSql) Save(any) string           { return "" }
func (unavailableToSql) Sum(string, any) string    { return "" }
func (unavailableToSql) Update(any, ...any) string { return "" }
