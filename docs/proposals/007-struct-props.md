# RFC 007: Struct Values as Component Props

- **Status**: Draft
- **Date**: 2026-03-16
- **Author**: TBD

---

## 1. Motivation

`htmlc` renders Go data by passing a `map[string]any` scope to each component.
When a caller wants to forward all fields of a Go struct to a child component, the
natural Vue idiom is `v-bind="user"`. Today this fails at runtime:

```text
v-bind on component "user-card": expected map or struct, got main.User
```

The failure is silent until the page is rendered — the author receives no
compile-time warning and no helpful error location.

### The failure in practice

```
components/
  UserCard.vue     ← expects props: name, email, role
pages/
  Profile.vue      ← receives a User struct from the server handler
```

`Profile.vue` (caller):

```html
<template>
  <!-- Works today -->
  <UserCard :name="user.Name" :email="user.Email" :role="user.Role" />

  <!-- Fails today with "expected map or struct, got main.User" -->
  <UserCard v-bind="user" />
</template>
```

The manual enumeration of every field is fragile: adding a field to `User` requires
updating every call site. The spread idiom exists precisely to avoid this coupling.

### Why the current code cannot handle it

`toStringMap` in `renderer.go` (lines 1938–1943) is a one-liner type assertion:

```go
// current — renderer.go
func toStringMap(val any) (map[string]any, bool) {
    if val == nil {
        return nil, true
    }
    m, ok := val.(map[string]any)
    return m, ok
}
```

It handles only `map[string]any`. Go structs have no implicit map conversion; the
fix requires reflection.

### Why "just use a map" is not the answer

Go application code typically models domain objects as structs with exported fields
and json tags. Requiring authors to convert every struct to `map[string]any` before
passing it to a component:
- duplicates type information already present in the struct definition
- defeats the IDE autocompletion that struct types provide
- conflicts with the pattern established by the existing `accessStructField` path,
  which already reads struct fields with reflection (used when an author writes
  `{{ user.Name }}` with a struct prop)

---

## 2. Goals

1. **`v-bind="structValue"` spreads exported struct fields as component props**,
   using the same field-name resolution order as `accessStructField`: json tag name
   first, then Go field name.
2. **`v-bind="mapValue"` continues to work unchanged** — no regression for the
   existing map spread path.
3. **`v-bind="nil"` remains a no-op** — existing nil-spread behaviour is preserved.
4. **Pointer-to-struct is supported**: when the value passed to `v-bind` is a
   pointer (or chain of pointers) to a struct, the engine dereferences it
   transparently. The template author uses the same syntax as for a plain struct
   value.
5. **Error message is actionable** when the value is an unsupported type (not a
   map, struct, or pointer-to-struct).
6. **No changes to the prop-discovery (`Props()`) path** — `Props()` scans template
   expressions for identifiers; field names accessed inside a child template are
   already discovered correctly.
7. **Anonymous (embedded) struct fields are promoted** into the top-level prop map,
   consistent with Go's own field-promotion rules and `encoding/json`.
8. **`accessStructField` handles embedded field promotion** so that `{{ user.Street }}`
   works consistently with `v-bind="user"` when `User` embeds a struct with a
   `Street` field.
9. **Types may implement a `Spreader` interface to supply props directly, bypassing
   reflection.**

---

## 3. Non-Goals

1. **Typed prop declarations (Vue `defineProps`)**: authors cannot declare that a
   prop must be of a specific Go type. This RFC covers only the spread-at-call-site
   problem.
2. **Deep flattening of named (non-embedded) nested struct fields**: `v-bind="user"`
   spreads the top-level fields of `User`. If a field value is itself a struct (and
   not anonymous), it is stored as-is under its key — the child component receives a
   typed prop and can access sub-fields via dot notation (see §4.1 and §6).
3. **Unexported fields**: unexported fields are never spread, matching the existing
   `accessStructField` behaviour.
4. **Interface types as props**: this RFC does not introduce any mechanism for
   passing an `interface{}` value and having `htmlc` introspect its dynamic type
   beyond what reflection already provides.
5. **v-bind on HTML elements** (not components): this RFC only affects `v-bind`
   spread on component invocations as a primary goal. However, because
   `applyAttrSpread` also calls `toStringMap`, HTML element struct spread gains
   support as a free side-effect.

---

## 4. Proposed Design

### 4.1 `toStringMap` — struct reflection

#### Current state

`toStringMap` (`renderer.go`) performs a single type assertion:

```go
// current
func toStringMap(val any) (map[string]any, bool) {
    if val == nil {
        return nil, true
    }
    m, ok := val.(map[string]any)
    return m, ok
}
```

#### Proposed extension

**`Spreader` interface** — any value that can expose itself as a flat string-keyed
collection of props implements this interface, allowing it to bypass reflection
entirely:

```go
// Spreader is implemented by any value that can expose itself as a
// flat string-keyed collection of props. Types that implement this
// interface are used by v-bind spread without reflection.
type Spreader interface {
    // SpreadProps returns the key/value pairs to be spread as component
    // props. Implementations must not mutate the returned map after
    // returning it. Returning nil is equivalent to an empty spread.
    SpreadProps() map[string]any
}
```

`toStringMap` checks for `Spreader` **before** the map fast-path and before
reflection. The full priority order is:

```text
priority order in toStringMap:
  1. nil → (nil, true)
  2. Spreader → call SpreadProps(), return result
  3. map[string]any → direct use (no copy)
  4. reflect.Struct / reflect.Ptr-to-struct → structToMap (reflection)
  5. anything else → (nil, false) — caller emits error
```

This aligns with the `expvar` pattern: the interface is the primary contract;
reflection is a fallback for types the engine does not control.

Replace the function with a reflection-aware version that also handles structs and
pointers-to-structs:

```go
// pseudo-code — not implementation
func toStringMap(val any) (map[string]any, bool) {
    if val == nil {
        return nil, true  // nil spread is a no-op
    }
    if s, ok := val.(Spreader); ok {
        return s.SpreadProps(), true  // interface fast-path — no reflection
    }
    if m, ok := val.(map[string]any); ok {
        return m, true    // fast path — no reflection needed
    }
    rv := reflect.ValueOf(val)
    for rv.Kind() == reflect.Ptr {
        if rv.IsNil() {
            return nil, true  // nil pointer is a no-op
        }
        rv = rv.Elem()
    }
    if rv.Kind() != reflect.Struct {
        return nil, false  // unsupported type — caller emits error
    }
    return structToMap(rv), true
}

// pseudo-code — not implementation
func structToMap(rv reflect.Value) map[string]any {
    out := make(map[string]any)
    collectStructFields(rv, out, false)
    return out
}

// pseudo-code — not implementation
// collectStructFields populates out with exported fields of rv.
// fromEmbedded: when true, do not overwrite keys already present (outer wins).
func collectStructFields(rv reflect.Value, out map[string]any, fromEmbedded bool) {
    rt := rv.Type()
    // First pass: collect direct (non-anonymous) fields — highest priority.
    for i := 0; i < rt.NumField(); i++ {
        f := rt.Field(i)
        if !f.IsExported() || f.Anonymous { continue }
        key := fieldKey(f)  // json tag name or Go field name; "" means skip
        if key == "" { continue }
        if fromEmbedded {
            if _, exists := out[key]; exists { continue }  // outer already set this key
        }
        // Store nil pointer fields as untyped nil so that v-if guards work
        // correctly (a typed nil pointer in an interface is non-nil, which
        // would mislead IsTruthy and cause v-if="Field" to evaluate as true).
        fval := rv.Field(i)
        if fval.Kind() == reflect.Ptr && fval.IsNil() {
            out[key] = nil
        } else {
            out[key] = fval.Interface()
        }
    }
    // Second pass: recurse into anonymous (embedded) struct fields.
    // Like encoding/json, recurse even when the anonymous field type is
    // unexported, so that its exported sub-fields are still promoted.
    for i := 0; i < rt.NumField(); i++ {
        f := rt.Field(i)
        if !f.Anonymous { continue }
        tag := f.Tag.Get("json")
        if tag != "" {
            parts := strings.SplitN(tag, ",", 2)
            if parts[0] == "-" { continue }  // json:"-" — exclude
            if parts[0] != "" {
                // Explicit json name on embedded field: not promoted.
                // Treat like a named field — but only if exported (unexported
                // field values cannot be retrieved via reflection).
                if f.IsExported() {
                    if _, exists := out[parts[0]]; !exists {
                        out[parts[0]] = rv.Field(i).Interface()
                    }
                }
                continue
            }
        }
        // Dereference pointer-to-struct embedded fields.
        fv := rv.Field(i)
        if fv.Kind() == reflect.Ptr {
            if fv.IsNil() { continue }
            fv = fv.Elem()
        }
        if fv.Kind() != reflect.Struct { continue }
        collectStructFields(fv, out, true)
    }
}
```

**Embedded struct flattening decision**: anonymous (embedded) struct fields are
flattened into the top-level prop map, consistent with how Go's `encoding/json`
handles embedded structs and with how Go itself promotes fields at the language
level. This resolves §10.1.

**Conflict resolution**: when an outer struct field has the same key as a promoted
field from an embedded struct, the outer field wins. This matches `encoding/json`
and Go's own field-promotion rules.

**json tag on embedded struct field**: if the embedded field itself has an explicit
json tag name (e.g., `Address \`json:"addr"\``), it is not promoted — it is stored
as a single prop under the tag name (here, `"addr"`). A tag of `json:"-"` causes
the embedded field to be skipped entirely.

**Named (non-embedded) struct fields**: when `structToMap` encounters a field whose
value is itself a struct but the field is not anonymous (e.g., `Address Address`),
it stores the struct value as-is under its key (e.g., `"Address": Address{...}`).
The child component receives this as a typed prop. The child template can access
sub-fields via standard dot notation (e.g., `{{ Address.City }}`), which resolves
via the existing `accessStructField` path. No changes to the expression evaluator
are required for this case — the struct value is passed through transparently.

**Chained v-bind**: `toStringMap` is applied at each component boundary. A parent
can spread a large struct containing a nested struct field, and a child that
receives the nested struct as a prop can in turn spread it to a grandchild with
`v-bind="Address"`. Because `toStringMap` handles any struct, this chain works
transitively without any additional changes.

**Nil pointer struct fields**: when a struct field is a pointer to a struct and its
value is `nil` (e.g., `Address *Address = nil`), `structToMap` includes
`"Address": nil` in the map. The child component receives a nil prop. If the child
template accesses `{{ Address.City }}` without a guard, the expression evaluator
returns an error on the nil dereference. Template authors must guard such access
with `v-if="Address"` (see §6 Example 7 and §8).

**Verdict**: the two-pass approach (direct fields first, then embedded recursion)
cleanly implements outer-wins semantics without a separate conflict-detection step.

#### Performance consideration

`structToMap` allocates a new `map[string]any` and copies all field values on each
call via reflection. This cost is O(fields) per call, paid even if the child
component reads only one or two props. For pages that render many component instances
in a loop, this means O(fields × instances) allocations and copies per request.

The preferred implementation direction for the struct path is a **lazy wrapper**
that avoids the upfront allocation. Rather than materialising a `map[string]any`,
an internal type holds the `reflect.Value` of the struct and resolves individual
field lookups on demand:

```go
// pseudo-code — not implementation
// reflectScope wraps a struct reflect.Value and resolves prop lookups
// lazily. It implements the same internal interface used by map scopes,
// so the rest of the engine is unaware of the difference.
type reflectScope struct{ rv reflect.Value }

func (s reflectScope) Lookup(key string) (any, bool) {
    // resolve via accessStructField logic — no full map materialisation
}
```

The engine's scope resolution already calls a `Lookup`-style function at each
`{{ expr }}` evaluation. Threading `reflectScope` through that path avoids the
upfront allocation entirely. `structToMap` is retained as a convenience helper
for the `Spreader` default implementation and for tests, but is not the hot path.

### 4.2 `accessStructField` — embedded field promotion

#### Current state

`accessStructField` in `expr/eval.go` iterates only direct fields of the struct
via `rt.NumField()`. Promoted fields from anonymous embedded structs are not
reachable via dot notation (e.g., `{{ user.Street }}` returns `Undefined` when
`User` embeds `Address`).

#### Proposed extension

Restructure `accessStructField` to use the same two-pass strategy as
`collectStructFields`:

1. **First pass** — check direct (non-anonymous) fields. Return immediately on a
   match. These have priority over promoted fields.
2. **Second pass** — recurse into anonymous embedded struct fields (dereferencing
   pointer-to-struct embedded fields as needed). An embedded field with an explicit
   json name is not promoted: only its own key is checked. An embedded field with
   `json:"-"` is skipped.

This makes `{{ user.Street }}` consistent with `v-bind="user"` spread when `User`
embeds a struct with a `Street` field.

```go
// pseudo-code — not implementation
func accessStructField(rv reflect.Value, name string) (any, error) {
    rt := rv.Type()
    // First pass: direct fields have priority.
    for i := 0; i < rt.NumField(); i++ {
        f := rt.Field(i)
        if !f.IsExported() || f.Anonymous { continue }
        if f.Name == name { return rv.Field(i).Interface(), nil }
        tag := f.Tag.Get("json")
        if tag != "" {
            tagName := strings.Split(tag, ",")[0]
            if tagName != "-" && tagName == name { return rv.Field(i).Interface(), nil }
        }
    }
    // Second pass: recurse into embedded fields.
    for i := 0; i < rt.NumField(); i++ {
        f := rt.Field(i)
        if !f.IsExported() || !f.Anonymous { continue }
        tag := f.Tag.Get("json")
        if tag != "" {
            parts := strings.Split(tag, ",")
            if parts[0] == "-" { continue }
            if parts[0] != "" {
                if parts[0] == name { return rv.Field(i).Interface(), nil }
                continue
            }
        }
        fv := rv.Field(i)
        if fv.Kind() == reflect.Ptr {
            if fv.IsNil() { continue }
            fv = fv.Elem()
        }
        if fv.Kind() != reflect.Struct { continue }
        val, err := accessStructField(fv, name)
        if err != nil { return nil, err }
        if _, ok := val.(UndefinedValue); !ok { return val, nil }
    }
    return Undefined, nil
}
```

**Verdict**: fixing `accessStructField` (option a) is the most coherent choice.
It aligns dot-access with v-bind spread and with how Go itself promotes fields.
The change is isolated to `expr/eval.go`.

### 4.3 Error message improvement

#### Current state

When `toStringMap` returns `false`, the caller emits:

```text
v-bind on component "user-card": expected map, got main.User
```

#### Proposed extension

After this RFC, any non-map, non-struct, non-nil value produces:

```text
v-bind on component "user-card": expected map or struct, got int
```

This change is a one-line update to the format string in `renderComponentElement`
and `applyAttrSpread`.

### 4.4 Static analysis limitation (`Props()`)

`Props()` on a child component scans the child's template expressions for
identifier names. It cannot know at static-analysis time that those identifiers
will be satisfied by a `v-bind` spread from a struct — there is no type information
available at parse time.

**Concrete consequence**: a tool that calls `Props()` on `UserCard` and verifies
that the parent passes each required prop individually will falsely report that
`Name` and `Email` are missing when the parent uses `v-bind="user"`. Such tools
must special-case `v-bind` spread:

> `Props()` returns the set of identifier names a template reads. When a parent
> uses `v-bind="structValue"`, the engine satisfies those props at runtime via
> `toStringMap`. Static tools that compare `Props()` output against explicit
> `:prop` bindings must also account for `v-bind` spreads; they cannot statically
> verify that a given struct type has the required fields.

No code changes are required for this; it is a documentation-only note.

### 4.5 `applyAttrSpread` — HTML element spread

`applyAttrSpread` (`renderer.go`) is used for `v-bind` on regular HTML elements.
It also calls `toStringMap`. Because `toStringMap` is being extended, HTML element
spread will also gain struct support at no extra cost. This is desirable and not a
regression risk since the function previously always errored on structs.

---

## 5. Syntax Summary

| Syntax                        | Meaning                                                              |
|-------------------------------|----------------------------------------------------------------------|
| `<Comp v-bind="mapValue" />`  | Spread all entries of `mapValue` (map[string]any) as props. Unchanged. |
| `<Comp v-bind="structValue"/>` | Spread all exported fields of `structValue` as props. **New.**      |
| `<Comp v-bind="nil" />`       | No-op spread. Unchanged.                                             |
| `<el v-bind="structValue" />` | Spread struct fields as HTML attributes on an element. **New.**     |

---

## 6. Examples

### Example 1 — Spread a plain struct

```
components/
  UserCard.vue
pages/
  Profile.vue
```

`UserCard.vue`:

```html
<template>
  <div class="card">
    <h2>{{ Name }}</h2>
    <p>{{ Email }}</p>
  </div>
</template>
```

`Profile.vue`:

```html
<template>
  <UserCard v-bind="user" />
</template>
```

Go handler:

```go
type User struct {
    Name  string
    Email string
}

eng.RenderPage(w, "Profile", map[string]any{
    "user": User{Name: "Alice", Email: "alice@example.com"},
})
```

**Rendered output**:

```html
<div class="card">
  <h2>Alice</h2>
  <p>alice@example.com</p>
</div>
```

### Example 2 — Struct with json tags

```go
type Product struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
    Price float64 `json:"price"`
}
```

```html
<template>
  <ProductCard v-bind="product" />
</template>
```

`ProductCard.vue`:

```html
<template>
  <div>{{ title }} — ${{ price }}</div>
</template>
```

The json tag `"title"` is used as the prop name, so `{{ title }}` in the child
resolves correctly.

### Example 3 — Pointer to struct

```go
eng.RenderPage(w, "UserPage", map[string]any{
    "user": &User{Name: "Bob", Email: "bob@example.com"},
})
```

```html
<template>
  <UserCard v-bind="user" />
</template>
```

The Go handler passes a pointer to a `User` struct. The engine dereferences any
number of pointer indirections transparently inside `toStringMap` — the template
author writes `v-bind="user"` exactly as they would for a plain struct value, with
no special operator required. Behaviour is identical to Example 1.

### Example 4 — Mixed: spread plus override

```html
<template>
  <!-- Spread all fields, then override a specific one -->
  <UserCard v-bind="user" :Name="'Admin: ' + user.Name" />
</template>
```

`v-bind` spread is processed in Phase 1 (lower priority). Individual `:prop`
bindings in Phase 2 override spread values for the same key. This matches the
existing priority rules and is unchanged by this RFC.

### Example 5 — Backward compatibility: plain map still works

```html
<template>
  <Card v-bind="attrs" />
</template>
```

```go
eng.RenderPage(w, "Page", map[string]any{
    "attrs": map[string]any{"title": "Hello", "color": "blue"},
})
```

Behaviour is identical to the current implementation. No change.

### Example 6 — Nested struct field access via dot notation

When `structToMap` encounters a named (non-anonymous) struct field, it stores the
struct value as-is under its key. The child template accesses sub-fields via
standard dot notation, which resolves via `accessStructField`.

```go
type Address struct { Street, City string }
type User struct {
    Name    string
    Address Address   // named field, NOT embedded
}
```

`Profile.vue`:

```html
<template>
  <UserCard v-bind="user" />
</template>
```

`UserCard.vue`:

```html
<template>
  <p>{{ Name }}</p>
  <p>{{ Address.City }}</p>
</template>
```

`structToMap` produces `{"Name": "Alice", "Address": Address{...}}`.
`Props()` discovers `Name` and `Address` as the prop identifiers — this is
correct; `Address` is the prop, not `Address.City`. The expression evaluator
calls `accessStructField` for `.City` on the `Address` value, which works
without any code change.

### Example 7 — Nil pointer nested struct field

When a struct field is a pointer to a struct and its value is `nil`, the child
receives a nil prop. Template authors must guard access with `v-if`:

```go
type User struct {
    Name    string
    Address *Address  // may be nil
}
```

`UserCard.vue`:

```html
<template>
  <p>{{ Name }}</p>
  <!-- Guard nil pointer before accessing sub-fields -->
  <p v-if="Address">{{ Address.City }}</p>
  <p v-else>No address</p>
</template>
```

`structToMap` produces `{"Name": "Alice", "Address": nil}` when `Address` is
nil. Without the `v-if` guard, `{{ Address.City }}` would error on nil
dereference.

### Example 8 — Embedded struct flattening

Anonymous embedded struct fields are promoted into the top-level prop map:

```go
type Address struct {
    Street string
    City   string
}

type User struct {
    Name    string
    Address          // anonymous (embedded)
}
```

`v-bind="user"` produces:

```go
map[string]any{
    "Name":   "Alice",
    "Street": "123 Main",
    "City":   "NYC",
}
```

Both `{{ Street }}` and `{{ Name }}` are available as top-level props in the
child. The same flattening applies to `{{ user.Street }}` via dot notation (see
§4.2).

When an outer field shadows a promoted field, the outer field wins:

```go
type Outer struct {
    City string        // outer field — wins
    Address            // embedded — City promoted but shadowed
}
```

`v-bind` spread produces `{"City": "OUTER", "Street": "..."}`.

### Example 9 — Chained v-bind with nested struct prop

```go
type Theme struct { Color, Font string }
type Page  struct { Title string; Theme Theme }
```

`Layout.vue`:

```html
<template>
  <Navbar v-bind="Theme" />
</template>
```

`Root.vue`:

```html
<template>
  <Layout v-bind="page" />
</template>
```

`Root` spreads `page`, producing `{"Title": "...", "Theme": Theme{...}}`.
`Layout` receives `Theme` as a typed prop and spreads it to `Navbar` with
`v-bind="Theme"`. Because `toStringMap` is applied at each component boundary,
this chain works transitively: `Navbar` receives `{"Color": "...", "Font": "..."}`.

---

## 7. Implementation Sketch

All changes are in `renderer.go` and `expr/eval.go` unless stated otherwise.

1. **`Spreader` interface** (`spreader.go`) — defined in a new small file to keep
   the interface close to its consumers. Placing it in its own file avoids
   cluttering `renderer.go` and makes the interface easy to document and discover.

2. **`toStringMap`** (`renderer.go`) — extend to check `Spreader` first, then
   `map[string]any` fast path, then `reflect.Struct` and `reflect.Ptr`-to-struct.
   ~20 lines added.

3. **`structToMap`** (`renderer.go`) — new private helper; delegates to
   `collectStructFields`. Retained as a convenience function for the `Spreader`
   default implementation and for tests; not the hot path. ~5 lines.

4. **`reflectScope`** (`renderer.go` or `expr/eval.go`) — new internal type
   wrapping a `reflect.Value` that resolves prop lookups lazily on demand (see
   §4.1 Performance consideration). Threading this through the scope-resolution
   path eliminates the upfront `map[string]any` allocation for the struct path.

5. **Benchmark** (`renderer_test.go` or `renderer_bench_test.go`) — a benchmark
   comparing the map-copy path against the lazy-lookup (`reflectScope`) path is
   **required before the RFC is accepted**, so the performance claim can be
   validated with data.

6. **`collectStructFields`** (`renderer.go`) — new private helper; two-pass
   algorithm (direct fields first, embedded recursion second). Handles:
   - `f.Anonymous` recursion (embedded struct flattening)
   - json tag name on embedded field prevents promotion
   - `json:"-"` on embedded field skips it
   - nil pointer-to-struct embedded field is skipped
   - outer-wins: `fromEmbedded=true` skips keys already present in `out`
   ~40 lines.

7. **`structFieldKey`** (`renderer.go`) — new private helper; returns json tag
   name or Go field name; `""` means skip. ~10 lines.

8. **Error message** (`renderer.go`) — update format strings in
   `renderComponentElement` and `applyAttrSpread` from `"expected map"` to
   `"expected map or struct"`. Two one-line changes.

9. **`accessStructField`** (`expr/eval.go`) — restructure to use two-pass
   algorithm matching `collectStructFields`, enabling embedded field promotion
   in dot-access expressions. ~35 lines (replacing ~20 lines).

10. **Tests** (`renderer_test.go`) — new table-driven and named tests covering:
   - plain struct spread on HTML element
   - plain struct spread on component
   - struct with json tags
   - pointer-to-struct spread
   - nil pointer spread (no-op)
   - embedded struct flattening (Gap 1)
   - embedded struct with explicit json name (not promoted)
   - outer field shadows embedded field
   - nil `*NestedStruct` field (Gap 6) — with `v-if` guard
   - chained v-bind spread (Gap 3)
   - updated error message check ("expected map or struct")

No changes required to `component.go` or the public `Engine` API.

---

## 8. Backward Compatibility

### `Engine` public API

No changes. `New`, `RenderPage`, `RenderFragment`, and all other exported methods
have identical signatures.

### `toStringMap` (private)

`toStringMap` is a package-private function. Its signature is unchanged; only its
behaviour for non-map, non-nil values changes from `(nil, false)` to
`(structMap, true)`. Callers that previously received an error for struct values
will now receive the spread map — this is a deliberate fix, not a breaking change.

### Template authors

Templates that previously relied on the error being surfaced (e.g., to detect
misconfiguration) will no longer see the error when passing a struct. This is
acceptable: the previous behaviour was a bug, not a feature.

### Existing `map[string]any` spread

Unchanged. The fast path in `toStringMap` exits before any reflection.

### `accessStructField` behaviour change

The two-pass restructure adds embedded-field promotion. This is a strictly
additive change: expressions that previously returned `Undefined` for promoted
fields (e.g., `{{ user.Street }}` when `User` embeds `Address`) will now return
the correct value. No previously-working expressions change behaviour.

### Static analysis tools using `Props()`

`Props()` returns template identifier names unchanged. Tools that verify prop
completeness by comparing `Props()` output against explicit `:prop` bindings in
the parent must also account for `v-bind` spreads. When a parent uses
`v-bind="structValue"`, the required props are satisfied at runtime by
`toStringMap`; there is no static way to verify that the struct type has all the
required fields. This is a known limitation — see §4.4.

### Nil pointer nested struct fields

When a struct field is a `*NestedStruct` with a nil value, `structToMap` includes
`"FieldName": nil` in the map. If a child template accesses sub-fields of a nil
prop without a `v-if` guard, the expression evaluator will return an error. This
is consistent with how nil values behave throughout the engine (e.g., a nil map
value accessed with dot notation already errors). Template authors should guard
nil nested struct props with `v-if` (see §6 Example 7).

### Pointer-to-struct — no new template syntax

Pointer-to-struct support is a **purely engine-internal change**. No new template
syntax is introduced: the engine dereferences any number of pointer indirections
inside `toStringMap` transparently, so template authors use `v-bind="expr"` for
both plain struct values and pointer-to-struct values. The template surface is
unchanged.

---

## 9. Alternatives Considered

### A. Require authors to implement a `Props() map[string]any` interface

Authors would add a method to each domain type. The engine would check for this
interface before falling back to reflection.

**Rejected**: adds boilerplate to every domain type; inconsistent with the
reflection-first approach already used in `accessStructField`.

### B. Accept only `map[string]any` and document the limitation

Document that callers must convert structs to maps before passing to `v-bind`.

**Rejected**: the conversion is mechanical and error-prone. The engine already uses
reflection for field access; not using it for spread is an inconsistency.

### C. Generate a `ToMap()` method via `go generate`

**Rejected**: adds a build step; changes the zero-configuration authoring model;
requires tooling not currently part of the project.

### D. Do not flatten embedded structs — treat them as a single named prop

Embedded struct `Address` in `User` would produce `{"Name": "Alice", "Address": Address{...}}`,
matching the behaviour of named fields.

**Rejected**: Go promotes embedded fields at the language level (both in the
type system and in `encoding/json`). Authors who write `v-bind="user"` with an
embedded struct will expect `{{ Street }}` to work in the child, not `{{ Address.Street }}`.
Inconsistency with `encoding/json` would be surprising. The blocking open question
in the original draft is resolved in favour of flattening.

### E. Leave `accessStructField` unchanged and document the inconsistency

`v-bind="user"` would flatten `Street` as a top-level prop, but `{{ user.Street }}`
would return `Undefined`.

**Rejected**: the inconsistency is confusing and error-prone. The fix is isolated
and low-risk (see §4.2). Option (a) — fixing `accessStructField` — is implemented.

### F. Require all types to implement an interface (pure-interface approach)

All values passed to `v-bind` would be required to implement a `Spreader`-like
interface; reflection would be eliminated entirely. This is distinct from the
`Spreader` interface introduced in §4.1, which is an **opt-in performance escape
hatch** — not a requirement.

**Rejected**: requiring every domain type to implement an interface adds boilerplate
and conflicts with the zero-configuration authoring model. Reflection is the right
default for types the engine does not control; `Spreader` is available for authors
who need deterministic, zero-allocation spread on a hot path. The two mechanisms
are complementary: reflection handles the common case; `Spreader` is the
performance valve.

---

## 10. Open Questions

1. **Embedded structs**: ~~should `v-bind="user"` with an embedded struct flatten
   the embedded fields?~~ **Resolved**: flatten promoted fields (consistent with
   `encoding/json` and Go's field promotion). Anonymous embedded fields are
   flattened; a json tag on the embedded field with a non-empty, non-`"-"` name
   prevents promotion and uses that name as the key. Outer fields shadow promoted
   fields. See §4.1 and §4.2.

2. **Unexported struct fields with struct tags**: some codebases tag unexported
   fields with `json:"-"` for documentation. The current `accessStructField`
   skips all unexported fields. Should `structToMap` do the same?
   *Recommendation*: yes — skip unexported fields. Consistent with `accessStructField`
   and with `encoding/json`. Non-blocking.

3. **`omitempty` json tag suffix**: should `structToMap` respect `omitempty` and
   skip zero-value fields?
   *Tentative recommendation*: no — spread all exported fields unconditionally.
   Template authors who want conditional props should use ternary expressions or
   explicit `:prop` bindings. Non-blocking.

4. **Nil pointer nested struct fields**: ~~open~~ **Resolved** (Option A):
   include `nil` in the map unconditionally. Template authors must guard sub-field
   access with `v-if="FieldName"`. This is consistent with how nil values behave
   throughout the engine. See §6 Example 7 and §8.

5. **Eager vs. lazy struct materialisation** (**blocking**): should the struct
   path materialise a `map[string]any` eagerly via `structToMap`, or use a lazy
   `reflectScope` that resolves field lookups on demand? A benchmark comparing the
   two approaches **must inform this decision before the RFC is accepted**. See
   §4.1 Performance consideration and §7.
