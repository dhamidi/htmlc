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
v-bind on component "user-card": expected map, got main.User
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

  <!-- Fails today with "expected map, got main.User" -->
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
4. **Pointer-to-struct is supported**: `v-bind="&user"` dereferences and spreads.
5. **Error message is actionable** when the value is an unsupported type (not a
   map, struct, or pointer-to-struct).
6. **No changes to the prop-discovery (`Props()`) path** — `Props()` scans template
   expressions for identifiers; field names accessed inside a child template are
   already discovered correctly.

---

## 3. Non-Goals

1. **Typed prop declarations (Vue `defineProps`)**: authors cannot declare that a
   prop must be of a specific Go type. This RFC covers only the spread-at-call-site
   problem.
2. **Nested struct spread**: `v-bind="user"` spreads the top-level fields of
   `User`, not the fields of nested struct values. Deep flattening is out of scope.
3. **Unexported fields**: unexported fields are never spread, matching the existing
   `accessStructField` behaviour.
4. **Interface types as props**: this RFC does not introduce any mechanism for
   passing an `interface{}` value and having `htmlc` introspect its dynamic type
   beyond what reflection already provides.
5. **v-bind on HTML elements** (not components): this RFC only affects `v-bind`
   spread on component invocations. Attribute spread on regular HTML elements is a
   separate concern.

---

## 4. Proposed Design

### 4.1 `toStringMap` — struct reflection

#### Current state

`toStringMap` (`renderer.go:1938`) performs a single type assertion:

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

Replace the function with a reflection-aware version that also handles structs and
pointers-to-structs:

```go
// pseudo-code — not implementation
func toStringMap(val any) (map[string]any, bool) {
    if val == nil {
        return nil, true  // nil spread is a no-op
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
    rt := rv.Type()
    out := make(map[string]any, rt.NumField())
    for i := 0; i < rt.NumField(); i++ {
        f := rt.Field(i)
        if !f.IsExported() {
            continue
        }
        key := f.Name
        if tag := f.Tag.Get("json"); tag != "" {
            if parts := strings.SplitN(tag, ",", 2); parts[0] != "-" && parts[0] != "" {
                key = parts[0]
            }
        }
        out[key] = rv.Field(i).Interface()
    }
    return out
}
```

**Verdict**: reflection is the only viable approach given the Go type system. The
fast path (`map[string]any`) preserves performance for the common case.

#### Option analysis

- ✅ Reflection on struct fields: handles any struct type without code generation;
  consistent with the existing `accessStructField` implementation in `expr/eval.go`.
- ⚠️ Requires `reflect` import in `renderer.go` (already imported).
- ❌ Code generation (`go generate`): would require a build step, breaks zero-
  dependency authoring flow, out of scope for this project.
- ❌ Requiring a `ToMap() map[string]any` method: forces application authors to
  implement boilerplate on every domain type; inconsistent with how the engine
  already reads struct fields.

### 4.2 Error message improvement

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

### 4.3 `applyAttrSpread` — HTML element spread

`applyAttrSpread` (`renderer.go:1891`) is used for `v-bind` on regular HTML
elements. It also calls `toStringMap`. Because `toStringMap` is being extended, HTML
element spread will also gain struct support at no extra cost. This is desirable
and not a regression risk since the function previously always errored on structs.

---

## 5. Syntax Summary

| Syntax                        | Meaning                                                              |
|-------------------------------|----------------------------------------------------------------------|
| `<Comp v-bind="mapValue" />`  | Spread all entries of `mapValue` (map[string]any) as props. Unchanged. |
| `<Comp v-bind="structValue"/>` | Spread all exported fields of `structValue` as props. **New.**      |
| `<Comp v-bind="&structPtr" />` | Dereference pointer, then spread. **New.**                          |
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

The pointer is dereferenced before spread. Behaviour is identical to Example 1.

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

---

## 7. Implementation Sketch

All changes are in `renderer.go` unless stated otherwise.

1. **`toStringMap`** — extend to handle `reflect.Struct` and `reflect.Ptr`-to-struct
   using a new private `structToMap(rv reflect.Value) map[string]any` helper.
   One new function (~20 lines); `toStringMap` gains ~10 lines.

2. **`structToMap`** — iterates exported fields of a `reflect.Value` of kind
   `Struct`. For each field, resolves the prop key using the json tag (first
   segment, skip `-` and empty). Stores `rv.Field(i).Interface()` as the value.
   Mirrors the existing `accessStructField` logic in `expr/eval.go`.

3. **Error message** — update the format string in `renderComponentElement` (line
   ~1568) and `applyAttrSpread` (line ~1899) from `"expected map"` to
   `"expected map or struct"`.

4. **Tests** — add table-driven tests in `renderer_test.go` (or a new
   `vbind_struct_test.go`) covering:
   - plain struct spread
   - pointer-to-struct spread
   - struct with json tags
   - nil pointer spread (no-op)
   - unsupported type error (e.g., `int`)
   - spread-then-override priority

No changes required to `expr/eval.go`, `component.go`, or the public `Engine` API.

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

---

## 10. Open Questions

1. **Embedded structs**: should `v-bind="user"` with an embedded struct flatten the
   embedded fields into the prop map, or treat the embedded field as a single prop
   named by the embedded type's name?
   *Tentative recommendation*: flatten promoted fields (consistent with how Go's
   `encoding/json` handles embedded structs). Blocking — must be resolved before
   implementation to avoid a subsequent breaking change.

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
