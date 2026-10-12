Fixed on master in 1663fa940 — the coerce-at-the-bind-sites option from this issue. Not in a tag yet; lands with the next metaengine cut.

`metaengine.FilterBindValue` (`metaengine/filter_bind.go:23`) renders `time.Time` via RFC3339Nano, byte-identical to how `encoding/json` writes it into the view document, so the bind compares exactly against the stored text:

```go
case time.Time:
	return t.Format(time.RFC3339Nano)
```

Wired at every site:

- planned-table binds: `metaengine/planned_filter.go:77,89`
- sqliteengine standard `json_extract` path plus both raw `buildStreamQuery` sites — the standard branch there was a third raw site this issue didn't name
- `FilterIn` lists coerce element-wise; write-side `metaengine.ExtractFields` coerces the same way so planned TEXT columns match the binds and the JSON doc

Not adopted: the `driver.Valuer` alternative — consumers carry no choice today, the stored-format coercion was sufficient (your workaround proved it).

Proof: `metaengine/sqliteengine/time_filter_test.go:157` (standard json_extract leg) and `:169` (planned-table leg) run this issue's same-day bounds table — Eq/Le/Lt/Ge/Gt per leg, ten assertions. Reverting the coercion fails all ten; green at `3dbf04192`.
