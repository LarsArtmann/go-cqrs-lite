# storage/view — Queryable SQL Read Model Store

The view store persists projection state in SQL tables with **queryable columns** —
unlike blind KV stores, you can filter, order, and paginate by column values directly
in SQL. Designed for read models materialized by CQRS projections.

## Quick Start (AutoMapper — recommended)

`AutoMapper` generates the full column mapping from struct tags, eliminating
boilerplate for tables with many columns:

```go
type UserView struct {
    Name       string    `view:"name"`
    Email      string    `view:"email"`
    Age        int       `view:"age"`
    CreatedAt  time.Time `view:"created_at"`
    Tombstoned bool      `view:"tombstoned"`
}

store, err := view.NewSQLiteViewStore[UserView, UserID](
    db,
    view.AutoMapperWithTombstone[UserView]("users_view", "tombstoned"),
)
```

The key column (`TEXT PRIMARY KEY` named `key`) is added automatically — do NOT
tag the key field. Fields without a `view:"..."` tag are skipped.

### SQL type inference

| Go type                    | SQL type |
| -------------------------- | -------- |
| `string`, `*string`        | TEXT     |
| `int`, `int32`, `int64`    | INTEGER  |
| `uint`, `uint32`, `uint64` | INTEGER  |
| `float32`, `float64`       | REAL     |
| `bool`                     | INTEGER  |
| `time.Time`                | TEXT     |
| `[]byte`                   | BLOB     |

Use `view:"-"` to explicitly skip a field.

## Manual ViewMapper (escape hatch)

For custom SQL types, computed columns, or scan logic that reflection can't handle:

```go
mapper := view.ViewMapper[UserView]{
    Table: "users_view",
    Columns: []view.ViewColumn[UserView]{
        {Name: "name", Type: "TEXT", Extract: func(v *UserView) any { return v.Name }},
        {Name: "email", Type: "TEXT", Extract: func(v *UserView) any { return v.Email }},
        {Name: "age", Type: "INTEGER", Extract: func(v *UserView) any { return v.Age }},
    },
    ScanRow: func(scan func(dest ...any) error) (*UserView, error) {
        var v UserView
        if err := scan(&v.Name, &v.Email, &v.Age); err != nil {
            return nil, err
        }
        return &v, nil
    },
}

store, err := view.NewSQLiteViewStore[UserView, UserID](db, mapper)
```

## Constructors

```go
// SQLite
store, err := view.NewSQLiteViewStore[V, K](db, mapper, opts...)

// PostgreSQL
store, err := view.NewSQLViewStore[V, K](db, mapper, opts...)

// Explicit dialect
store, err := view.NewViewStoreWithDialect[V, K](db, dialect, mapper, opts...)
```

All constructors auto-create the table and indexes unless `WithoutViewAutoMigrate()`
is passed. Use this when you manage schema manually (e.g., shared `RelationalSchema`
owning all DDL):

```go
store, err := view.NewSQLiteViewStore[V, K](db, mapper, view.WithoutViewAutoMigrate())
```

## CRUD

```go
store.Set(ctx, key, &view)           // upsert
val, err := store.Get(ctx, key)      // returns kv.ErrNotFound if missing
store.Delete(ctx, key)               // no-op if missing
items, err := store.Scan(ctx, nil)   // all records
items, err := store.Scan(ctx, []byte("user:")) // key prefix scan (LIKE 'user:%')
```

## Querying with Conditions, Ordering, and Keyset Pagination

```go
results, err := store.Query(ctx, kv.ViewQuery{
    Conditions: []kv.Condition{
        {Column: "age", Op: kv.OpGt, Value: 18},
    },
    Order: []kv.OrderClause{
        {Column: "name", Desc: false},
    },
    Limit: 20,
})
```

### Keyset pagination (high-performance seek pagination)

For large result sets, keyset pagination avoids the O(n) cost of `OFFSET`:

```go
page1, _ := store.Query(ctx, kv.ViewQuery{
    Order: []kv.OrderClause{{Column: "created_at", Desc: true}},
    Keyset: &kv.Keyset{
        Columns: []string{"created_at"},
        Values:  []any{lastSeenCreatedAt}, // from previous page's last row
    },
    Limit: 50,
})
```

The key column is always appended as a tiebreaker. Set `Keyset` to enable seek
mode; leave it nil for offset/limit mode.

## Tombstone Filtering

If your mapper has a `TombstoneColumn`, you can query excluding or only returning
tombstoned rows:

```go
active, _ := store.QueryByTombstone(ctx, true, false)   // exclude tombstoned
deleted, _ := store.QueryByTombstone(ctx, false, true)   // only tombstoned
```

## Batch Operations

```go
err := store.BatchSet(ctx, []kv.ViewItem[V, K]{
    {Key: key1, Value: &val1},
    {Key: key2, Value: &val2},
    // ...up to hundreds; chunked to respect SQLite's 999-param limit
})
```

## Transactions

```go
err := db.RunInTx(ctx, func(tx *sql.Tx) error {
    txStore := store.InTx(tx)
    return txStore.Set(ctx, key, &val)
})
```

## Secondary Indexes

Add indexes via the mapper:

```go
mapper := view.AutoMapper[UserView]("users_view")
mapper.Indexes = []view.IndexSpec{
    {Name: "idx_users_email", Columns: []string{"email"}},
    {Name: "idx_users_age_created", Columns: []string{"age", "created_at"},
     Where: "tombstoned = 0"}, // partial index
}
```

## Re-exports from `storage`

All types and constructors are re-exported from the top-level `storage` package,
so you can import `storage` instead of `storage/view`:

```go
import "github.com/larsartmann/go-cqrs-lite/storage/v4"

store, _ := storage.NewSQLiteViewStore[V, K](db, storage.AutoMapper[V]("my_view"))
```
