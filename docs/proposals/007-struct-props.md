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
9. **First-rune lowercase aliasing**: `StructProps.Get` and `accessStructField`
   match a key against the first-rune-lowercased form of a Go field name as a
   fallback, so that Vue/JS authors can write `user.address` to access a Go field
   named `Address` when no json tag is present. Only the first rune is affected.

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
   `applyAttrSpread` also calls `ToProps`, HTML element struct spread gains
   support as a free side-effect.

---

## 4. Proposed Design

### 4.1 `Props` interface — unified spread abstraction

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

#### Proposed design

Rather than extending `toStringMap` with more type-dispatch cases, the redesign
asks the more fundamental question: *what does `htmlc` actually need from a props
value?* The engine needs exactly two operations:

1. **Key enumeration** — "what prop names does this object expose?"
2. **Value lookup by key** — "what is the value of prop `X`?"

These two operations define the **`Props`** interface — the single contract the
engine demands of any value used as component props:

```go
// pseudo-code — not implementation
// Props is the interface htmlc demands of any value used as component props.
// It models two operations: key enumeration and value lookup.
type Props interface {
    // Keys returns the set of prop names this object exposes.
    // The returned slice must not be mutated by the caller.
    Keys() []string

    // Get returns the value associated with key and whether it was found.
    Get(key string) (any, bool)
}
```

The engine dispatches on `Props` at every boundary where it previously called
`toStringMap` or consulted a scope map.

**`MapProps`** — wraps `map[string]any`, replacing the map fast-path:

```go
// pseudo-code — not implementation
type MapProps struct{ m map[string]any }

func NewMapProps(m map[string]any) MapProps { return MapProps{m} }

func (p MapProps) Keys() []string {
    keys := make([]string, 0, len(p.m))
    for k := range p.m { keys = append(keys, k) }
    return keys
}

func (p MapProps) Get(key string) (any, bool) {
    v, ok := p.m[key]
    return v, ok
}
```

**`StructProps`** — wraps a `reflect.Value` lazily, replacing both the old eager
`structToMap` allocation and any lazy-wrapper variants:

```go
// pseudo-code — not implementation
type StructProps struct{ rv reflect.Value }

func NewStructProps(v any) (StructProps, bool) {
    rv := reflect.ValueOf(v)
    for rv.Kind() == reflect.Ptr {
        if rv.IsNil() { return StructProps{}, false }
        rv = rv.Elem()
    }
    if rv.Kind() != reflect.Struct { return StructProps{}, false }
    return StructProps{rv}, true
}

func (p StructProps) Keys() []string {
    // enumerate exported fields using structFieldKey() resolution order:
    // json tag name first, then Go field name; recurse into anonymous embeds
}

func (p StructProps) Get(key string) (any, bool) {
    // Resolution order:
    //   1. exact json tag match
    //   2. exact Go field name match
    //   3. first-rune-lowercased Go field name match (Vue/JS alias)
    // Only the first rune is lowercased; rest of the name is unchanged.
    // resolve via accessStructField logic — no full map materialisation
}
```

`StructProps.Get` resolves field lookups **lazily**: only the specific field being
accessed is read via reflection. There is no upfront materialisation of a
`map[string]any`, so the O(fields × instances) allocation cost of an eager approach
is avoided entirely.

**`ToProps`** — the single dispatch entry point, replacing `toStringMap`:

```go
// pseudo-code — not implementation
// ToProps converts val into a Props implementation, or returns an error
// for unsupported types. This is the sole dispatch point for all spread
// operations in the engine.
func ToProps(val any) (Props, error) {
    if val == nil {
        return nil, nil         // nil spread is a no-op
    }
    if p, ok := val.(Props); ok {
        return p, nil           // identity — already implements Props
    }
    if m, ok := val.(map[string]any); ok {
        return NewMapProps(m), nil  // map fast-path
    }
    if sp, ok := NewStructProps(val); ok {
        return sp, nil          // struct or pointer-to-struct
    }
    return nil, fmt.Errorf("expected map or struct, got %T", val)
}
```

The priority order is:

```text
priority order in ToProps:
  1. nil → (nil, nil) — nil spread is a no-op
  2. Props → return as-is (identity)
  3. map[string]any → NewMapProps
  4. struct / ptr-to-struct → NewStructProps
  5. anything else → error
```

Types that need to supply props without reflection can implement `Props` directly.
The `Keys()` and `Get()` methods give implementors full control over prop exposure
without forcing materialisation of a `map[string]any`.

**Embedded struct flattening decision**: anonymous (embedded) struct fields are
flattened into the top-level prop set returned by `StructProps.Keys()`, consistent
with how Go's `encoding/json` handles embedded structs and with how Go itself
promotes fields at the language level. This resolves §10.1.

**Conflict resolution**: when an outer struct field has the same key as a promoted
field from an embedded struct, the outer field wins. This matches `encoding/json`
and Go's own field-promotion rules.

**json tag on embedded struct field**: if the embedded field itself has an explicit
json tag name (e.g., `Address \`json:"addr"\``), it is not promoted — it is stored
as a single prop under the tag name (here, `"addr"`). A tag of `json:"-"` causes
the embedded field to be skipped entirely.

**Named (non-embedded) struct fields**: when `StructProps` encounters a field whose
value is itself a struct but the field is not anonymous (e.g., `Address Address`),
it exposes the struct value as-is under its key (e.g., `"Address": Address{...}`).
The child component receives this as a typed prop. The child template can access
sub-fields via standard dot notation (e.g., `{{ Address.City }}`), which resolves
via the existing `accessStructField` path. No changes to the expression evaluator
are required for this case — the struct value is passed through transparently.

**Chained v-bind**: `ToProps` is applied at each component boundary. A parent can
spread a large struct containing a nested struct field, and a child that receives
the nested struct as a prop can in turn spread it to a grandchild with
`v-bind="Address"`. Because `ToProps` handles any struct, this chain works
transitively without any additional changes.

**Nil pointer struct fields**: when a struct field is a pointer to a struct and its
value is `nil` (e.g., `Address *Address = nil`), `StructProps.Get("Address")`
returns `nil, true`. The child component receives a nil prop. If the child template
accesses `{{ Address.City }}` without a guard, the expression evaluator returns an
error on the nil dereference. Template authors must guard such access with
`v-if="Address"` (see §6 Example 7 and §8).

**Verdict**: the two-pass approach (direct fields first, then embedded recursion)
cleanly implements outer-wins semantics without a separate conflict-detection step.
`StructProps` subsumes both the eager `structToMap` and any lazy-wrapper variants
into one coherent type.

### 4.2 `accessStructField` — embedded field promotion

#### Current state

`accessStructField` in `expr/eval.go` iterates only direct fields of the struct
via `rt.NumField()`. Promoted fields from anonymous embedded structs are not
reachable via dot notation (e.g., `{{ user.Street }}` returns `Undefined` when
`User` embeds `Address`).

#### Proposed extension

Restructure `accessStructField` to use the same two-pass strategy as
`collectStructFields`, and apply the same three-step field-name resolution order
used by `StructProps.Get`:

1. **First pass** — check direct (non-anonymous) fields using three-step resolution
   per field:
   1. Exact json tag match (case-sensitive).
   2. Exact Go field name match (case-sensitive).
   3. First-rune-lowercased Go field name match — so that `{{ user.address }}`
      and `{{ user.Address }}` both resolve to the `Address` field when no json
      tag is present. Only the first rune is lowercased; the rest of the name is
      unchanged.
   Return immediately on a match. Direct fields have priority over promoted fields.
2. **Second pass** — recurse into anonymous embedded struct fields (dereferencing
   pointer-to-struct embedded fields as needed). An embedded field with an explicit
   json name is not promoted: only its own key is checked. An embedded field with
   `json:"-"` is skipped. The same three-step resolution is applied at each level
   of recursion.

This makes `{{ user.Street }}` consistent with `v-bind="user"` spread when `User`
embeds a struct with a `Street` field, and makes `{{ user.address }}` equivalent
to `{{ user.Address }}` when no json tag overrides the field name.

```go
// pseudo-code — not implementation
func accessStructField(rv reflect.Value, name string) (any, error) {
    rt := rv.Type()
    // First pass: direct fields have priority.
    for i := 0; i < rt.NumField(); i++ {
        f := rt.Field(i)
        if !f.IsExported() || f.Anonymous { continue }
        // step 1: exact json tag match
        tag := f.Tag.Get("json")
        if tag != "" {
            tagName := strings.Split(tag, ",")[0]
            if tagName != "-" && tagName == name { return rv.Field(i).Interface(), nil }
            if tagName != "" && tagName != "-" { continue } // tag present — skip steps 2 & 3
        }
        // step 2: exact Go field name match
        if f.Name == name { return rv.Field(i).Interface(), nil }
        // step 3: first-rune lowercase alias
        lcName := strings.ToLower(string([]rune(f.Name)[:1])) + string([]rune(f.Name)[1:])
        if lcName == name { return rv.Field(i).Interface(), nil }
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

When `ToProps` returns an error, the caller emits:

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
> `ToProps`. Static tools that compare `Props()` output against explicit
> `:prop` bindings must also account for `v-bind` spreads; they cannot statically
> verify that a given struct type has the required fields.

No code changes are required for this; it is a documentation-only note.

### 4.5 `applyAttrSpread` — HTML element spread

`applyAttrSpread` (`renderer.go`) is used for `v-bind` on regular HTML elements.
It also calls `ToProps`. Because `ToProps` now handles structs, HTML element
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
| `{{ user.address }}`          | Accesses Go field `Address` via first-rune-lowercase alias when no json tag is present. **New.** |

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
number of pointer indirections transparently inside `NewStructProps` — the template
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

When `StructProps` encounters a named (non-anonymous) struct field, it exposes the
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

`StructProps.Keys()` returns `["Name", "Address"]`.
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

`StructProps.Get("Address")` returns `nil, true` when `Address` is nil.
Without the `v-if` guard, `{{ Address.City }}` would error on nil dereference.

### Example 8 — Embedded struct flattening

Anonymous embedded struct fields are promoted into the top-level prop set:

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

`v-bind="user"` exposes the following props via `StructProps.Keys()`:

```go
[]string{"Name", "Street", "City"}
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

`Root` spreads `page`; `StructProps.Keys()` on `page` returns `["Title", "Theme"]`.
`Layout` receives `Theme` as a typed prop and spreads it to `Navbar` with
`v-bind="Theme"`. Because `ToProps` is applied at each component boundary,
this chain works transitively: `Navbar` receives `Color` and `Font` as props.

### Example 10 — First-rune lowercase alias

When no json tag is present, `StructProps.Get` and `accessStructField` match the
first-rune-lowercased form of a Go field name as a fallback. This lets Vue/JS
authors use conventional camelCase initial-lowercase identifiers without requiring
json tags on every field.

```go
type User struct {
    Name    string
    Address string
}
```

```html
<template>
  <!-- Both are equivalent when no json tag is present -->
  <p>{{ user.Name }}</p>
  <p>{{ user.name }}</p>   <!-- first-rune lowercase alias -->
  <p>{{ user.Address }}</p>
  <p>{{ user.address }}</p> <!-- first-rune lowercase alias -->
</template>
```

**json tags take precedence**: if the struct has `Address \`json:"addr"\``, then
the canonical key is `"addr"`. The key `"address"` does **not** match — the
first-rune alias is only consulted when there is no json tag. Only `user.addr`
resolves to the `Address` field in that case.

```go
type User struct {
    Name    string
    Address string `json:"addr"`
}
```

```html
<template>
  <p>{{ user.addr }}</p>    <!-- resolves: json tag match -->
  <p>{{ user.address }}</p> <!-- does NOT resolve: json tag "addr" is the canonical key -->
  <p>{{ user.Address }}</p> <!-- does NOT resolve: json tag takes precedence -->
</template>
```

`Keys()` returns the canonical names only — `["Name", "addr"]` in the tagged
example above — and is unaffected by the alias. The lowercase alias is a
lookup-only affordance for `Get` and `accessStructField`.

---

## 7. Implementation Sketch

Primary changes are in a new `props.go` file, with secondary changes in
`renderer.go` and `expr/eval.go`.

1. **`Props` interface, `MapProps`, `StructProps`, `ToProps`** (`props.go`) — new
   file containing the complete props abstraction. Defines the `Props` interface,
   both standard implementations, and the `ToProps` dispatch constructor. Placing
   these in a dedicated file makes the interface easy to document, discover, and
   test in isolation. ~100 lines.

2. **`renderer.go`** — replace all call sites of `toStringMap` with `ToProps`;
   propagate `Props` through scope resolution instead of `map[string]any`. The
   engine dispatches on `Props.Keys()` for `v-bind` spread enumeration and
   `Props.Get()` for individual lookup. ~20 lines changed.

3. **`expr/eval.go`** — `accessStructField` is now used by `StructProps.Get`
   as its field-resolution back-end; the two-pass embedded-field logic (§4.2) is
   shared between both. Each direct field is checked with a three-step resolution
   (exact json tag → exact Go field name → first-rune-lowercased Go field name).
   No new types in this file; only the two-pass restructure with the added step-3
   comparison. ~45 lines (replacing ~20 lines). ~10 additional lines for
   first-rune alias logic.

4. **Tests** (`props_test.go`) — `MapProps` and `StructProps` are independently
   unit-testable without a running engine. Add table-driven tests for each,
   covering: plain struct, json tags, pointer-to-struct, embedded field promotion,
   embedded field with explicit json name, outer field shadows promoted field,
   nil pointer field. ~60 lines.

5. **Benchmark** (`renderer_bench_test.go`) — a benchmark verifying that
   `StructProps.Get` avoids the upfront allocation of the old `structToMap` path
   is **required before the RFC is accepted**, so the lazy-lookup performance
   claim can be validated with data.

6. **`collectStructKeys`** (`props.go`) — private helper used by `StructProps.Keys`;
   two-pass algorithm (direct fields first, embedded recursion second). Handles:
   - `f.Anonymous` recursion (embedded struct flattening)
   - json tag name on embedded field prevents promotion
   - `json:"-"` on embedded field skips it
   - nil pointer-to-struct embedded field is skipped
   - outer-wins: `fromEmbedded=true` skips keys already present in `out`
   ~40 lines.
   **`StructProps.Get`** applies the same three-step resolution as
   `accessStructField` (exact json tag → exact Go field name → first-rune-lowercased
   Go field name). `Keys()` is **unchanged** — it returns only the canonical key
   (json tag or Go field name); the lowercase alias is a lookup-only affordance and
   is never added to the `Keys()` output. ~10 additional lines for step-3 alias.

7. **`structFieldKey`** (`props.go`) — private helper; returns json tag name or Go
   field name; `""` means skip. ~10 lines.

8. **Error message** (`renderer.go`) — update format strings in
   `renderComponentElement` and `applyAttrSpread` from `"expected map"` to
   `"expected map or struct"`. Two one-line changes.

9. **Integration tests** (`renderer_test.go`) — table-driven and named tests
   covering the full render path:
   - plain struct spread on HTML element
   - plain struct spread on component
   - struct with json tags
   - pointer-to-struct spread
   - nil pointer spread (no-op)
   - embedded struct flattening
   - embedded struct with explicit json name (not promoted)
   - outer field shadows embedded field
   - nil `*NestedStruct` field — with `v-if` guard
   - chained v-bind spread
   - updated error message check ("expected map or struct")

No changes required to `component.go` or the public `Engine` API.

---

## 8. Backward Compatibility

### `Engine` public API

No changes. `New`, `RenderPage`, `RenderFragment`, and all other exported methods
have identical signatures.

### `toStringMap` → `ToProps` (private)

`toStringMap` is replaced by `ToProps`. Both are package-private. `ToProps`
returns a `Props` value rather than a `map[string]any`, but all call sites are
within the package and are updated together. Callers that previously received an
error for struct values will now receive a `StructProps` — this is a deliberate
fix, not a breaking change.

### Template authors

Templates that previously relied on the error being surfaced (e.g., to detect
misconfiguration) will no longer see the error when passing a struct. This is
acceptable: the previous behaviour was a bug, not a feature.

### Existing `map[string]any` spread

Unchanged. `ToProps` returns a `MapProps` for `map[string]any` without any
reflection.

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
`ToProps`; there is no static way to verify that the struct type has all the
required fields. This is a known limitation — see §4.4.

### First-rune lowercase aliasing

Additive change. Templates that previously accessed `{{ user.Address }}` continue
to work identically — exact Go field name match (step 2) succeeds before the alias
is consulted. Templates that mistakenly wrote `{{ user.address }}` previously
returned `Undefined`; they now return the correct value. No existing working
template changes behaviour.

### Nil pointer nested struct fields

When a struct field is a `*NestedStruct` with a nil value, `StructProps.Get`
returns `nil, true` for that field. If a child template accesses sub-fields of a
nil prop without a `v-if` guard, the expression evaluator will return an error.
This is consistent with how nil values behave throughout the engine (e.g., a nil
map value accessed with dot notation already errors). Template authors should guard
nil nested struct props with `v-if` (see §6 Example 7).

### Pointer-to-struct — no new template syntax

Pointer-to-struct support is a **purely engine-internal change**. No new template
syntax is introduced: `NewStructProps` dereferences any number of pointer
indirections transparently, so template authors use `v-bind="expr"` for both plain
struct values and pointer-to-struct values. The template surface is unchanged.

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

All values passed to `v-bind` would be required to implement a `Props`-like
interface; reflection would be eliminated entirely.

**Rejected**: requiring every domain type to implement an interface adds boilerplate
and conflicts with the zero-configuration authoring model. Reflection is the right
default for types the engine does not control. Types that need custom prop exposure
can implement `Props` directly (an opt-in, not a requirement). The two mechanisms
are complementary: reflection handles the common case via `StructProps`; `Props`
is the escape hatch for authors who need deterministic, zero-allocation spread.

### G. Retain `Spreader` as-is

Keep the `Spreader` interface and `toStringMap` unchanged. `StructProps` /
`MapProps` are not introduced.

**Rejected**: `Spreader` only addresses the *escape hatch* case (custom
materialisation) and does not eliminate the dual-path problem (`toStringMap`
vs. the lazy-wrapper `reflectScope`). The `Props` interface models the problem
at the right level of abstraction — it encodes exactly what the engine needs
(key enumeration and value lookup) — and subsumes both mechanisms into one
coherent type hierarchy.

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
   skips all unexported fields. Should `StructProps` do the same?
   *Recommendation*: yes — skip unexported fields. Consistent with `accessStructField`
   and with `encoding/json`. Non-blocking.

3. **`omitempty` json tag suffix**: should `StructProps` respect `omitempty` and
   skip zero-value fields?
   *Tentative recommendation*: no — expose all exported fields unconditionally.
   Template authors who want conditional props should use ternary expressions or
   explicit `:prop` bindings. Non-blocking.

4. **Nil pointer nested struct fields**: ~~open~~ **Resolved** (Option A):
   `StructProps.Get` returns `nil, true` for nil pointer fields unconditionally.
   Template authors must guard sub-field access with `v-if="FieldName"`. This is
   consistent with how nil values behave throughout the engine. See §6 Example 7
   and §8.

5. **`StructProps` performance validation** (**blocking**): a benchmark
   demonstrating that `StructProps.Get` avoids the upfront allocation of the old
   eager materialisation approach **must be produced before the RFC is accepted**,
   so the lazy-lookup performance claim is validated with data. See §7.

6. **Full camelCase vs. first-rune-only lowercasing**: should `MyFieldName` also
   be accessible as `myFieldName` (first-rune only, proposed) or as a fully
   camel-cased variant? *Recommendation*: first-rune only. Full camelCase
   conversion is ambiguous (acronyms, multi-word abbreviations) and the Vue/JS
   convention for props received from a parent is to match the name as provided —
   only the initial capital is conventionally lowercased. Non-blocking.
