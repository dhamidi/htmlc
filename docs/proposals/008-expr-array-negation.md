# RFC 008: Consistent Negation of Array and Slice Values

- **Status**: Accepted
- **Date**: 2026-03-16
- **Author**: TBD

---

## 1. Motivation

The `!` operator in `htmlc` template expressions should follow JavaScript
truthiness semantics: all arrays and slices are truthy regardless of length, so
`![]` is `false` and `!nonEmptyArray` is also `false`.

Today `isTruthy` in `expr/eval.go` implements this via a `default: return true`
fallback for all types not explicitly handled. This implicit rule has two problems:

1. **Typed Go slices from caller scope behave inconsistently with array literals.**
   An array literal `[]` produces `[]any{}` (non-nil `[]any`), which hits
   `default: return true` — correct. A typed Go nil slice (`var s []string`)
   stored in `map[string]any` produces a *non-nil interface with a nil value*,
   which *also* hits `default: return true` — again correct, but for the wrong
   reason. If any code path ever stores a `reflect.Value`-wrapped slice in scope,
   `isTruthy` would return `true` for the `reflect.Value` object itself, not for
   the slice it contains.

2. **No test coverage.** The `TestUnaryEval` test checks `!true`, `!false`, and
   `!0` but contains no case for `![]`, `!someSlice`, or `!typedGoSlice`. A
   future refactor of the `default` branch could silently break all array negation
   without a failing test.

### The failure mode in practice

A template author writes:

```html
<template>
  <ul v-if="items.length > 0">
    <li v-for="item in items">{{ item }}</li>
  </ul>
  <p v-if="!items.length">No items found.</p>
</template>
```

`!items.length` works today (`!0 = true`, `!3 = false`). But an author who instead
writes:

```html
<p v-if="!items">No items.</p>
```

gets a surprising result: `v-if="!items"` is `false` even when `items` is an
empty `[]string{}`, because Go slices are always truthy. This is *correct* per
JavaScript semantics but violates the author's intuition.

The fix has two parts:
1. Make `isTruthy` explicitly handle all slice/array kinds using reflection so the
   rule is expressed in code, not in a comment.
2. Add tests so the rule cannot be broken silently.

The semantic choice (JS truthiness: all arrays are truthy) is not changed by this
RFC — only the *explicitness* and *test coverage* of the rule.

### Why a silent `default` is dangerous

The `isTruthy` function is called from four places: `!` (unary), `&&`, `||`, and
`?:` (ternary). Any future type added to the expression system (e.g., a custom
`Date` type, a `Set` value, a `reflect.Value` wrapper) falls through to `default`
with no explicit intention. This RFC makes the intention explicit for the most
common Go collection types: slices and arrays.

---

## 2. Goals

1. **`isTruthy` explicitly handles `reflect.Slice` and `reflect.Array` kinds**
   and always returns `true` for them, matching JavaScript's "all arrays are
   truthy" rule.
2. **`![]` evaluates to `false`** in all contexts (array literal, typed Go slice,
   nil slice stored as interface, struct-field slice accessed via `evalMember`).
3. **Test coverage for array/slice negation** is added to `expr/eval_test.go`.
4. **`v-if="!items"` where `items` is a Go slice** is documented as having
   JS-style semantics (always falsy result for `!`, regardless of length).
5. **No change to the boolean result** for any currently-tested expression — this
   RFC fixes latent brittleness without changing observable behaviour for existing
   valid inputs.

---

## 3. Non-Goals

1. **Changing array truthiness semantics to Python-style** (empty arrays falsy).
   `htmlc` expressions follow JavaScript semantics throughout; deviating here
   would create an inconsistency.
2. **Making empty Go slices falsy.** This is explicitly *not* the goal; the fix
   makes the existing correct behaviour explicit.
3. **Handling `reflect.Value` objects stored directly in scope.** Application code
   should never put `reflect.Value` in a scope map; this is not a supported usage.
4. **Changing `&&`, `||`, or `?:` behaviour** — these already delegate to
   `isTruthy` and will benefit from the fix automatically.

---

## 4. Proposed Design

### 4.1 `isTruthy` — explicit slice/array handling

#### Current state

`isTruthy` (`expr/eval.go:466–483`):

```go
// current
func isTruthy(v any) bool {
    if v == nil {
        return false
    }
    if _, ok := v.(UndefinedValue); ok {
        return false
    }
    switch val := v.(type) {
    case bool:
        return val
    case float64:
        return val != 0 && !math.IsNaN(val)
    case string:
        return val != ""
    default:
        return true
    }
}
```

All maps, structs, slices, arrays, and any other Go value fall through to
`default: return true`. This is correct for non-nil values but the correctness
is implicit.

#### Proposed extension

Add an explicit `case []any` fast path for array literals and a `reflect`-based
branch in `default` for typed Go slices and arrays:

```go
// pseudo-code — not implementation
func isTruthy(v any) bool {
    if v == nil {
        return false
    }
    if _, ok := v.(UndefinedValue); ok {
        return false
    }
    switch val := v.(type) {
    case bool:
        return val
    case float64:
        return val != 0 && !math.IsNaN(val)
    case string:
        return val != ""
    case []any:
        return true // fast path: array literals are always truthy (JS semantics)
    default:
        // For typed Go slices and arrays (e.g. []string, [3]int) passed
        // from caller scope, use reflection to confirm the kind and return
        // true unconditionally — matching JavaScript "all arrays are truthy".
        rv := reflect.ValueOf(val)
        switch rv.Kind() {
        case reflect.Slice, reflect.Array:
            return true
        }
        return true // all other non-nil, non-undefined types are truthy
    }
}
```

**Verdict**: adding an explicit `case []any` fast path covers the array-literal
case (most common in templates) without reflection. The `reflect` fallback covers
typed Go slices from caller scope. The `default: return true` at the end remains
correct for maps, structs, and any other type.

#### Option analysis

- ✅ Explicit `case []any`: zero-cost for the common array-literal case; self-
  documenting.
- ✅ `reflect.Slice`/`reflect.Array` branch: covers all typed Go slice types
  passed from application code; consistent with how `accessMember` already uses
  reflection for slice indexing.
- ⚠️ Reflection has a small cost. Mitigated by the `[]any` fast path and by the
  fact that `isTruthy` is called only at expression evaluation time, not during
  parsing.
- ❌ Documenting only in a comment and leaving the `default` branch: provides no
  test coverage guarantee; the next maintainer has no signal that the rule is
  intentional.

### 4.2 Test additions

Add the following cases to `expr/eval_test.go`:

| Expression   | Scope                           | Expected |
|--------------|---------------------------------|----------|
| `![]`        | nil                             | `false`  |
| `![1, 2, 3]` | nil                             | `false`  |
| `!items`     | `items: []any{"a", "b"}`        | `false`  |
| `!items`     | `items: []any{}`                | `false`  |
| `!items`     | `items: []string{"a"}`          | `false`  |
| `!items`     | `items: []string{}`             | `false`  |
| `!items`     | `items: []string(nil)` as `any` | `false`  |

All seven cases should return `false` because arrays are truthy and `!truthy = false`.

### 4.3 `IsTruthy` public function

`IsTruthy` (`expr/eval.go:463`) is a thin wrapper around `isTruthy`. It is part
of the public API (`expr.IsTruthy`). No signature change is required. The fix
propagates automatically.

---

## 5. Syntax Summary

No new syntax is introduced. This RFC is a bug fix for the expression evaluator.

| Expression          | Value when `items` is a non-nil Go slice | Value when `items` is nil stored as `any` |
|---------------------|------------------------------------------|-------------------------------------------|
| `!items`            | `false`                                  | `false`                                   |
| `items \|\| "none"` | `items` (truthy)                         | `items` (truthy, non-nil interface)       |
| `![]`               | `false`                                  | —                                         |

*Note*: `nil` passed directly as a scope value (not a typed nil slice) is falsy:
`v-if="!nothing"` where `nothing` is `nil` evaluates to `true`. This is unchanged.

---

## 6. Examples

### Example 1 — Array literal negation

```html
<template>
  <!-- v-if condition: ![] = false → div is NOT rendered -->
  <div v-if="![]">This is never shown</div>
</template>
```

Rendered output: *(empty)*

### Example 2 — Go slice prop negation

```go
eng.RenderPage(w, "List", map[string]any{
    "items": []string{"apple", "banana"},
})
```

```html
<template>
  <ul v-if="items">
    <li v-for="item in items">{{ item }}</li>
  </ul>
  <!-- v-if="!items" is false for any non-nil slice, even empty -->
  <p v-if="!items">No items.</p>
</template>
```

To show a "no items" message, authors should test `items.length`:

```html
<p v-if="!items.length">No items.</p>
```

### Example 3 — Empty typed Go slice

```go
eng.RenderPage(w, "List", map[string]any{
    "items": []string{},
})
```

`v-if="!items"` evaluates to `false` (the empty slice is truthy). The correct
empty-state check is `v-if="!items.length"` which evaluates to `true` when
`items.length == 0`.

### Example 4 — Nil slice stored as interface

```go
var tags []string // nil
eng.RenderPage(w, "Page", map[string]any{"tags": tags})
```

`!tags` is `false` because a typed nil slice stored in `map[string]any` is a
non-nil interface value (it carries the `[]string` type). Authors should use
`!tags.length` or `tags == null` (which returns `false` because the interface is
non-nil) to detect the nil case.

*Open question §10.1 addresses whether nil typed slices should be treated
specially.*

### Example 5 — Backward compatibility: existing expressions unchanged

```html
<div v-if="!flag">...</div>      <!-- bool: unchanged -->
<div v-if="!count">...</div>     <!-- number: unchanged -->
<div v-if="!name">...</div>      <!-- string: unchanged -->
<div v-if="!null">...</div>      <!-- null: unchanged -->
<div v-if="!undefined">...</div> <!-- undefined: unchanged -->
```

All of these produce the same result before and after the fix.

---

## 7. Implementation Sketch

All changes are in `expr/eval.go` and `expr/eval_test.go`.

1. **`isTruthy` in `expr/eval.go`**:
   - Add `case []any: return true` immediately after the `case string:` branch.
     This is a one-liner that covers the array-literal case without reflection.
   - In the `default:` branch, add a `reflect.ValueOf(val).Kind()` check for
     `reflect.Slice` and `reflect.Array` that returns `true`. This is ~5 lines.
   - The `import "reflect"` is already present in `eval.go`.

2. **`expr/eval_test.go`**:
   - Extend `TestUnaryEval` with the seven cases from §4.2.
   - Add a new `TestIsTruthy_SliceAndArray` function that tests `expr.IsTruthy`
     directly with various Go slice and array types to serve as regression
     protection.

No changes to `renderer.go`, `component.go`, `engine.go`, or the public `Engine`
API.

---

## 8. Backward Compatibility

### `expr.IsTruthy` (public)

No signature change. Return values for all currently-tested inputs are unchanged.
The fix only makes latent correct behaviour explicit.

### `expr.Eval` (public)

No change. Expressions that previously evaluated correctly continue to do so.

### Template authors

No observable change for templates that use `!boolExpr`, `!numberExpr`, or
`!stringExpr`. Templates using `!arrayExpr` continue to receive `false` as
before (the fix only adds test coverage and explicitness, not a semantic change
for non-nil arrays).

---

## 9. Alternatives Considered

### A. Treat empty arrays as falsy (Python semantics)

`isTruthy([]any{})` would return `false`, making `![]` = `true`.

**Rejected**: `htmlc` expressions are explicitly documented as following
JavaScript semantics. Deviating here would make `v-if="items"` fail to show a
list when `items` is non-empty but wrapped in an empty outer array, and would
break `v-for="item in items" v-if="item"` patterns where items could be arrays.
Consistency with JavaScript is more important than intuitive empty-collection
handling.

### B. Leave the `default` branch and add only tests

Add test cases without changing the implementation.

**Rejected**: tests prove the current accidental behaviour but do not express
*intent*. Future maintainers could reasonably remove the `default: return true`
branch while adding an explicit error for unknown types, breaking the feature.
The explicit `case []any` and reflect branches express the intent.

### C. Add a `Truthy` interface

Define `type Truthy interface { IsTruthy() bool }` and check for it in
`isTruthy`.

**Rejected**: requires application types to implement the interface; adds
complexity for a case that is fully handled by a two-line reflect check.

---

## 10. Open Questions

1. **Typed nil slices**: a `var s []string` stored as `any` in scope is a non-nil
   interface. `isTruthy` returns `true`. If authors want falsy behaviour for
   uninitialized slices, they must use `s == null` (which also returns `false`
   since the interface is non-nil). Should `isTruthy` use `reflect.Value.IsNil()`
   to make typed nil slices falsy?
   *Tentative recommendation*: no — this diverges from JavaScript where there is
   no concept of a nil array. Keep the JS-compatible rule. Document it.
   Non-blocking.

2. **`reflect.Map` kind**: should maps (e.g., `map[string]any{}`) also be
   explicitly listed as truthy? In JavaScript, all objects (including `{}`) are
   truthy.
   *Tentative recommendation*: yes, add `reflect.Map` to the same explicit branch
   for consistency. Non-blocking; can be done in the same PR.
