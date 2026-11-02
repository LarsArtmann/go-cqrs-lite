# Evidence Pack: CUE `Subsume` as a schema-compatibility oracle

> **Date:** 2026-11-02 · **Status:** FINAL — input to [ADR-0163](../adr/0163-schema-compatibility-structural-differ-over-cue.md)
> **Gate:** G0 of [`docs/planning/2026-11-02_05-31_SUPERB-catalog-cue-compat-and-unification-pareto-plan.md`](../planning/2026-11-02_05-31_SUPERB-catalog-cue-compat-and-unification-pareto-plan.md)
> **Verdict:** **NO-GO** — every residual miss is a false *compatible*, the unsafe direction.
> **Consumed by:** `catalog/compat/doc.go` (why the structural walk exists), ADR-0163.

## 1. The question

`docs/SCHEMA-COMPATIBILITY.md` states a policy in prose: some payload-shape changes are additive
(old events still decode), others owe an explicit op (`RenameField`, `RemoveField`, `Transform`).
The repo had three partial answers — the prose policy, static ladder checks (cqrs-lint E021/E022),
and `schema.EventSchema.Fingerprint` — and none can say whether a *specific* shape delta was
additive or breaking. A fingerprint reports **that** a shape moved, never **how**.

The plan's hypothesis was that CUE already answers this. `cue.Value.Subsume` documents exactly the
question we want to ask:

> `// Subsume reports nil when w is an instance of v or an error otherwise.`
> `// Without options, the entire value is considered for assumption, which means Subsume tests`
> `// whether v is a backwards compatible (newer) API version of w.`
> — `cuelang.org/go/cue/types.go:1793-1797`

G0 asked: does `encoding/jsonschema.Extract` + `Value.Subsume` classify the policy's own rows
correctly? If it does uniformly, CUE becomes the oracle. If it does not, the fallback (a structural
differ over `catalog/schema`) is the only honest option, because a gate that lies in the unsafe
direction is worse than no gate.

## 2. Harness

Scratch module outside the repo, pinned exactly:

```
cuelang.org/go v0.17.1
```

Pipeline per fixture — JSON Schema document → CUE data value → `jsonschema.Extract` → built CUE
value → `new.Subsume(old, opts...)`:

```go
func extract(ctx *cue.Context, doc string, cfg *jsonschema.Config) (cue.Value, error) {
	v := ctx.CompileString(doc)
	if err := v.Err(); err != nil {
		return cue.Value{}, err
	}
	file, err := jsonschema.Extract(v, cfg) // cue.InstanceOrValue -> *ast.File
	if err != nil {
		return cue.Value{}, err
	}
	out := ctx.BuildFile(file)
	return out, out.Err()
}

// nil means "old is an instance of new", i.e. the change is additive.
err := newV.Subsume(oldV, opts...)
```

Six configurations were measured: `jsonschema.Config{}` ("open") and `Config{Strict: true}`
("strict"), each crossed with four `Value.Subsume` option sets — none, `cue.Raw()`, `cue.Final()`,
`cue.Raw()+cue.Final()`.

`Strict` is not a cosmetic knob: it makes `Extract` reject JSON Schema formats it does not model, so
`"format":"int32"` fails extraction outright (§4, H12).

## 3. Fixtures (24)

Twelve come straight from the policy's two tables. Twelve more stress the shapes the policy treats
as breaking but that a naive subtype check can miss: schema-closing, format changes, newly
constrained array items, restructuring, and constraint additions.

| # | Fixture | Policy class | Rationale |
| --- | --- | --- | --- |
| A1 | add optional field | additive | "Adding a field with a zero value" |
| A2 | add optional nested field | additive | nested variant of A1 |
| A3 | add field inside array items | additive | additive below a list |
| A4 | widen `integer` → `number` | additive | "Widening an int field" |
| A5 | add enum value | additive | widening a domain |
| A6 | required becomes optional | additive | relaxing a requirement |
| B1 | remove field | bump required | `RemoveField` owed |
| B2 | rename field | bump required | `RenameField` owed |
| B3 | change field type | bump required | `Transform` owed |
| B4 | add required field | bump required | old writers cannot satisfy it |
| B5 | narrow enum | bump required | narrowing a domain |
| B6 | become non-nullable | bump required | null payloads now rejected |
| H1 | close `additionalProperties` | bump required | old accepted unknown fields |
| H2 | open `additionalProperties` | additive | new accepts a superset |
| H3 | flatten restructure (`a.b` → `a_b`) | bump required | `Transform` owed |
| H4 | add `pattern` | bump required | previously valid values may now fail |
| H5 | relax `pattern` | additive | every old value still valid |
| H6 | add `maxLength` | bump required | truncation boundary moved |
| H7 | change `format` (`date-time` → `uuid`) | bump required | "Changing a field's type" |
| H8 | become nullable | additive | widening |
| H9 | add array `items` constraint | bump required | old elements may violate the new type |
| H10 | add required next to existing | bump required | old writers may omit it |
| H11 | nested remove field | bump required | nested `RemoveField` owed |
| H12 | `int32` → `int64` | additive | "Widening an int field" |

## 4. Results

```
fixture \ config                 open/bare       open/raw        open/final      open/raw+final  strict/bare     strict/raw
A1 add-optional-field            ok              ok              ok              ok              ok              ok
A2 add-optional-nested           ok              ok              ok              ok              ok              ok
A3 add-field-in-items            ok              ok              ok              ok              ok              ok
A4 widen-int-to-number           ok              ok              ok              ok              ok              ok
A5 add-enum-value                ok              ok              ok              ok              ok              ok
A6 required-becomes-optional     ok              ok              ok              ok              ok              ok
B1 remove-field                  ok              ok              MISS(compatible) MISS(compatible) ok             ok
B2 rename-field                  ok              ok              MISS(compatible) MISS(compatible) ok             ok
B3 change-field-type             ok              ok              ok              ok              ok              ok
B4 add-required-field            ok              ok              MISS(compatible) MISS(compatible) ok             ok
B5 narrow-enum                   ok              ok              ok              ok              ok              ok
B6 become-non-nullable           ok              ok              ok              ok              ok              ok
H1 close additionalProperties    MISS(compatible) ok             MISS(compatible) MISS(compatible) MISS(compatible) ok
H2 open additionalProperties     ok              ok              ok              ok              ok              ok
H3 flatten restructure           ok              ok              MISS(compatible) MISS(compatible) ok             ok
H4 add pattern constraint        ok              ok              ok              ok              ok              ok
H5 relax pattern constraint      ok              ok              ok              ok              ok              ok
H6 add maxLength                 ok              ok              ok              ok              ok              ok
H7 change format                 MISS(compatible) MISS(compatible) MISS(compatible) MISS(compatible) MISS(compatible) MISS(compatible)
H8 become-nullable               ok              ok              ok              ok              ok              ok
H9 add array items constraint    MISS(compatible) MISS(compatible) MISS(compatible) MISS(compatible) MISS(compatible) MISS(compatible)
H10 add required next to existing ok             ok              ok              ok              ok              ok
H11 nested remove field          ok              ok              MISS(compatible) MISS(compatible) ok             ok
H12 int32 to int64 widen         ok              ok              ok              ok              ERROR           ERROR

SCORE                            21/24           22/24           16/24           16/24           20/24           21/24
```

**Best configuration: `Config{}` + `cue.Raw()` → 22/24.**

## 5. Why 22/24 is a NO-GO and not "good enough"

Two facts decide it.

**5.1 Every residual miss is a false `compatible`.** The gate's whole purpose is to refuse a
release that moved a shape without the owed op. A false `breaking` wastes a human's time. A false
`compatible` **licenses shipping a replay-breaking change**. The error is asymmetric, so the
tolerance is zero, not two.

```
UNSAFE MISSES (said compatible, policy says breaking)
  open/bare        3: H1 close additionalProperties, H7 change format, H9 add array items constraint
  open/raw         2: H7 change format, H9 add array items constraint
  open/final       8: B1, B2, B4, H1, H3, H7, H9, H11
  strict/bare      3: H1, H7, H9
  strict/raw       2: H7, H9
```

**5.2 The two remaining misses are structural, not tuning.** No option combination reaches them:

- **H7 change `format`.** `format` is not a CUE constraint. `jsonschema.Extract` does not encode
  `format` as a validation the subsume checker can compare; `date-time` and `uuid` are both plain
  `string` in the extracted value, so the pair is indistinguishable *by construction*. The policy is
  explicit — "Changing a field's type" owes `Transform` — so this is a policy row CUE cannot see.
- **H9 add array `items` constraint.** An unconstrained CUE list (`[...]`) subsumes a constrained one
  (`[...string]`) under `Subsume`, so widening *items* is reported as additive. This is CUE's list
  semantics, not a misconfiguration.

Both rows are first-class in this repo's own model: `catalog/schema.Property` carries `Format`,
`Items`, `Pattern`, and `Enum` as inspectable fields. The structural walk sees them directly.

**5.3 The "obvious" option is the worst one.** `cue.Final()` sounds like the right reading — it
treats the old value as closed, which is what a released payload schema *is* — and it collapses to
16/24, turning `remove field`, `rename field`, and `add required field` into **compatible**. That is
precisely backwards: those are the three rows the policy spends an op on.

**5.4 `Strict` is disqualified on extraction, not accuracy.** `Config{Strict: true}` rejects
`"format":"int32"`, so the policy's own "widen int32 → int64" row cannot even be extracted.

## 6. Incidental API findings (worth recording once)

- **Direction.** `v.Subsume(w) == nil` ⟺ **w is an instance of v**. Verified independently: with
  `v = string` and `w = "abc"` it returns nil; reversed (`v = "abc"`, `w = string`) it errors.
  Empirically, `new.Subsume(old)` is the reading that answers the compat question.
- **No CLI escape hatch.** `cue vet`/`cue eval` cannot perform schema-vs-schema subsumption from the
  command line, so the check cannot be pushed into a shell-level gate script: it must be a Go
  program. That is a real integration cost, not just a favourite.
- **Extraction is the fragile half.** The `Config` surface (`Strict`, `StrictFeatures`,
  `StrictKeywords`, `Map`, `MapRef`, `DefaultVersion`, `OpenOnlyWhenExplicit`) is v0.x and partly
  marked experimental, so the bridge — not the lattice — would be the upgrade tax.
- **Binary cost.** A spike binary linking `cuelang.org/go` is 24 MB, and the module pulls eleven
  indirect dependencies (apd, protobuf, yaml, toml, uuid, proto, txtpbfmt, x/net, x/text …).

## 7. Verdict

**NO-GO for CUE as the compatibility oracle.** Not because the primitive is weak — it is
impressive, and it correctly classifies 22 of 24 hand-built policy fixtures on a clean
configuration. It is a NO-GO because:

1. the two rows it cannot see are rows this repo's policy explicitly owes an op on, and
2. its errors point in the unsafe direction, where a gate does real damage.

The shipped fallback is `catalog/compat`: a structural walk over two `catalog/schema.Schema` trees.
It has no dependencies, sees `Format`/`Items`/`Pattern` as first-class fields, and classifies all 24
fixtures plus a 25-case table test in the shipped suite.

## 8. Reopen triggers

Revisit only if **all** hold:

1. `jsonschema` graduates out of v0.x and encodes `format` and constrained lists as comparable
   constraints — the two structural misses close.
2. There is a named consumer ask for CUE-authored payload schemas (ADR-0163 records the rejected
   alternative).
3. `Generate` (schema-from-Go) leaves experimental, so a single CUE source could own both
   directions instead of being a one-way bridge.

Until then, the structural differ is the oracle and this pack is the reason.

## 9. Reproducing

The harness lived in `/tmp/cuespike/final/` (scratch, not committed) with the pin above. It is
self-contained: six JSON Schema pairs per fixture, `CompileString` → `Extract` → `BuildFile` →
`Subsume`. Re-derive it from §2 and the fixture table in §3; the matrix in §4 is the expected output.
