Fixed on master in f55c9e631 — option (a) from this issue. Not in a tag yet; lands with the next metaengine cut.

Row resolution is json-tag-first with the Go-field-name fallback. One extractor site (`metaengine/reflect.go:330`), so the memory engine's filter, sort, and keyset-cursor comparisons all resolve `started_at` the way SQL engines already did:

```go
// metaengine/reflect.go — pass 1 json tag, pass 2 Go field name
if strings.EqualFold(JSONFieldName(f), name) {
	return v.Field(i).Interface()
}
```

Proof: `metaengine/reflect_row_parity_test.go:22` — json key, case-insensitive key, Go field name, case-insensitive field name, untagged field, missing column → nil. Cross-engine leg in `metaengine/sqliteengine/json_key_parity_test.go:60`: memory vs sqlite must return identical rows for a json-tag-named filter, the exact split brain this issue reported. Both green at `3dbf04192`.
