# RFC 013: Improved Expr Bindings for Go Methods

- **Status**: Draft
- **Date**: 2026-04-15
- **Author**: TBD

---

## 1. Motivation

`htmlc` template expressions can access exported struct fields and map keys via
dot notation (`post.Title`, `user["name"]`), but they cannot call exported Go
methods on those values. If an application type exposes behaviour through methods
rather than plain fields — a common Go pattern — the template author must either
duplicate the logic in a `RegisterBuiltin` wrapper or restructure their types,
both of which are unnecessary friction.

### The failure mode in practice

Consider a router that generates URLs:

```go
type Router struct { /* ... */ }

func (r *Router) LinkFor(route string) string { /* ... */ }
```

If an application puts the router in scope:

```go
scope["router"] = myRouter
```

the template expression `router.LinkFor("admin.dashboard")` fails today with:

```
value of type *Router is not callable
```

because `evalCall` in `expr/eval.go` only accepts values of type
`func(...any) (any, error)`. A Go method bound to a concrete receiver is a
`reflect.Method`; it does not satisfy that type assertion, so the expression
evaluator returns an error.

The same gap applies to zero-argument accessor methods:

```go
func (p Post) Summary() string { return p.Body[:100] }
```

`post.Summary` in a template resolves `accessMember` → struct case →
`accessStructField` → field lookup. `Summary` is not a field; the function
returns `UndefinedValue`, silently — there is no error telling the template
author that a callable method was found but ignored.

### Why the silent fallback is dangerous

`accessStructField` returns `UndefinedValue` (not an error) when the name is not
found. If the caller then interpolates the result, they see the string
`"undefined"` rendered in the page rather than an error they can act on. The
template appears to work but shows wrong data.

### The obvious alternative: `RegisterBuiltin`

`RegisterBuiltin` lets callers expose any `func(...any)(any, error)` globally. It
solves the case where a helper function is independent of scope, but it does not
solve the case where behaviour is naturally attached to a scope value (a router,
a model, a formatter). Requiring a builtin wrapper for every method on every type
passed into scope does not scale: callers must know at startup which methods
templates will need, and each adapter is boilerplate that the type author wrote
once but the template author must rewrite.

---

## 2. Goals

1. **Go methods callable from templates**: `post.Summary()` and
   `router.LinkFor("admin.dashboard")` evaluate correctly when `post` and
   `router` are Go values in scope.
2. **Parentheses optional for zero-parameter methods**: `post.Summary` (no
   parentheses) implicitly invokes `func (p Post) Summary() string` when the
   name resolves to a method rather than a field.
3. **Argument evaluation from expression scope**: method arguments are
   expression-language values (string literals, number literals, scope
   identifiers) evaluated by the normal `evalNode` path and coerced to Go
   parameter types via `reflect`.
4. **Lowercase-initial method aliases**: `post.summary` resolves
   `func (p Post) Summary() string`; `router.linkFor(r)` resolves
   `func (r *Router) LinkFor(route string) string` — matching the existing rule
   for struct fields.
5. **Field-first resolution priority**: the new method-resolution logic is
   attempted only after all existing field-resolution steps have failed, so
   existing templates are unaffected.
6. **Single-return-value and `(value, error)` methods supported**: both
   `func (p Post) Title() string` and `func (p Post) Validate() (bool, error)`
   are callable; results are normalised to `(any, error)`.

---

## 3. Non-Goals

1. **Pointer-receiver promotion on value types.** If a value `v` of type `T` is
   stored in scope and a method is defined on `*T` only, that method is not
   accessible unless the value is already addressable. This RFC accepts the Go
   reflect rule without special-casing: callers can store `&v` if pointer-receiver
   methods are required.
2. **Calling methods on maps.** Maps do not have methods in Go. Map keys continue
   to use the existing `reflect.Map` branch.
3. **Multiple return values beyond `(value, error)`.** Methods returning two
   non-error values or three or more values are not supported. Templates are
   display-oriented; functions returning multiple data values should be
   restructured to return a struct.
4. **Exposing `reflect.Method` or `reflect.Value` in the public API.** The
   `expr` package public surface does not change type signatures.

---

## 4. Proposed Design

### 4.1 Resolution priority in `accessMember`

#### Current state

`accessMember` (`expr/eval.go:327–391`) dispatches on `reflect.Kind`:

- `reflect.Map` → map key lookup
- `reflect.Struct` → `accessStructField(rv, name)`
- `reflect.Slice` / `reflect.Array` → `"length"` or numeric index
- `default` → error

`accessStructField` (`expr/eval.go:393–466`) does three passes for structs:

1. Direct field exact match (exported, non-anonymous).
2. `json` struct tag match.
3. First-rune-lowercased field name match (only when no `json` tag).
4. Recurse into anonymous (embedded) fields.

If none of these passes match, it returns `UndefinedValue` with no error.

#### Proposed extension

After `accessStructField` returns `UndefinedValue`, `accessMember` falls through
to a new helper `accessMethod`, which attempts method resolution on the original
`reflect.Value` (before pointer dereference for the struct case, so that
pointer-receiver methods are reachable):

```go
// pseudo-code — not implementation
func accessMember(obj, key any) (any, error) {
    // ... nil / undefined guards (unchanged) ...

    rv := reflect.ValueOf(obj)
    rvForMethod := rv                 // preserve pointer for method lookup
    for rv.Kind() == reflect.Ptr {
        if rv.IsNil() { return nil, fmt.Errorf(...) }
        rv = rv.Elem()
    }

    switch rv.Kind() {
    case reflect.Map:
        // ... unchanged ...
    case reflect.Struct:
        keyStr, ok := key.(string)
        if !ok { return nil, fmt.Errorf(...) }
        val, err := accessStructField(rv, keyStr)
        if err != nil { return nil, err }
        if _, isUndef := val.(UndefinedValue); !isUndef {
            return val, nil      // field found — return immediately
        }
        // Field not found; try methods on the original (possibly pointer) value.
        return accessMethod(rvForMethod, keyStr)
    case reflect.Slice, reflect.Array:
        // ... unchanged ...
    default:
        // Non-struct, non-map, non-collection: still try methods.
        if keyStr, ok := key.(string); ok {
            return accessMethod(rvForMethod, keyStr)
        }
        return nil, fmt.Errorf("cannot access member of %T", obj)
    }
}
```

The new `accessMethod` helper performs two passes:

```go
// pseudo-code — not implementation
func accessMethod(rv reflect.Value, name string) (any, error) {
    // Pass 1: exact method name.
    if m := rv.MethodByName(name); m.IsValid() {
        return wrapMethod(m), nil
    }
    // Pass 2: first-rune-uppercased alias (lowercase-initial in template → exported Go method).
    if len(name) > 0 {
        alias := string(unicode.ToUpper(rune(name[0]))) + name[1:]
        if m := rv.MethodByName(alias); m.IsValid() {
            return wrapMethod(m), nil
        }
    }
    return UndefinedValue{}, nil
}
```

`wrapMethod` converts a `reflect.Value` of kind `Func` into a Go value that
`evalCall` and the implicit-zero-arg path can later inspect:

```go
// pseudo-code — not implementation
type boundMethod struct {
    fn       reflect.Value  // reflect.Value with Kind == Func, already bound to receiver
    numIn    int            // number of fixed parameters (excluding receiver; already bound)
    numOut   int            // 1 or 2
    variadic bool           // true if the method has a variadic final parameter
}
```

`accessMethod` returns the `boundMethod` as an `any`. The calling paths in
`evalCall` and `evalMember` detect this type and handle it specially (see §4.3
and §4.4).

#### Resolution priority table

| Step | What is checked                                    | Function                 |
|------|----------------------------------------------------|--------------------------|
| 1    | Exported non-anonymous field, exact name           | `accessStructField`      |
| 2    | Field json tag match                               | `accessStructField`      |
| 3    | Field first-rune-lowercased match (no json tag)    | `accessStructField`      |
| 4    | Embedded (anonymous) field recursion               | `accessStructField`      |
| 5    | Method exact name (`MethodByName(name)`)           | `accessMethod` (new)     |
| 6    | Method first-rune-uppercased alias                 | `accessMethod` (new)     |

Steps 1–4 are unchanged. Steps 5–6 are new and are only reached when steps 1–4
all return `UndefinedValue`.

**Verdict**: field-first priority means zero risk of breaking existing templates
that use struct-field access today. A field named `Title` always wins over a
method named `Title`. This matches Go's own resolution rule for embedded types.

#### Option analysis

- ✅ Field-first: backward-compatible; consistent with Go's own shadowing rules.
- ✅ Two-pass method lookup (exact then alias): matches the existing
  `accessStructField` precedence pattern; predictable.
- ⚠️ A field and a method with the same capitalisation are disambiguated by
  the field winning. If a type genuinely has both `Title string` and
  `func (p Post) Title() string` (which Go would reject at compile time), this
  never occurs. Safe by the language rules.
- ❌ Method-first: would allow a method to shadow a field of the same name.
  This cannot occur in valid Go, but reversing the order would confuse template
  authors who read the struct definition to reason about which names are
  available.

---

### 4.2 Lowercase-initial method alias

#### Current state

`accessStructField` already implements the lowercase alias rule at step 3 above:

```go
// expr/eval.go:415–419
alias := string(unicode.ToLower(rune(f.Name[0]))) + f.Name[1:]
if alias == name {
    return rv.Field(i).Interface(), nil
}
```

This lets template authors write `post.title` to access `Post.Title` without
knowing the capitalisation.

#### Proposed extension

`accessMethod` mirrors this rule in the reverse direction: the template name
(`name`) has its first rune **uppercased** to produce the Go exported method
name. A template expression `post.summary` → `accessMethod` → tries
`MethodByName("Summary")`.

```go
// pseudo-code — not implementation
alias := string(unicode.ToUpper(rune(name[0]))) + name[1:]
if m := rv.MethodByName(alias); m.IsValid() {
    return wrapMethod(m), nil
}
```

The alias pass is attempted only when the exact `MethodByName(name)` check fails.
This means a method literally named with a lowercase first rune (which Go would
make unexported and therefore inaccessible via `reflect.Value.MethodByName`) does
not cause incorrect alias resolution.

**Verdict**: mirrors `accessStructField` symmetrically. No new concepts for
template authors; the same mental model ("lowercase-initial in template maps to
capitalised Go name") covers both fields and methods.

---

### 4.3 Zero-parameter method implicit call

#### Current state

`evalMember` (`expr/eval.go:287–302`) calls `accessMember` and returns the
result directly. If `accessMember` returns a `func(...any)(any, error)` (a
builtin-style function), the result is a callable that requires an explicit
`callee()` syntax.

Currently, returning a method from `accessMember` is impossible (methods are
not recognised), so no implicit-call problem arises.

#### Proposed extension

After `accessMember` returns, `evalMember` checks whether the result is a
`boundMethod` with `numIn == 0`. If so, it immediately invokes the method and
returns the result:

```go
// pseudo-code — not implementation
func evalMember(n *MemberExpr, scope map[string]any) (any, error) {
    obj, err := evalNode(n.Object, scope)
    if err != nil { return nil, err }
    var key any
    if n.Computed {
        key, err = evalNode(n.Property, scope)
        if err != nil { return nil, err }
    } else {
        key = n.Property.(*Identifier).Name
    }
    val, err := accessMember(obj, key)
    if err != nil { return nil, err }
    if bm, ok := val.(boundMethod); ok && bm.numIn == 0 {
        return invokeMethod(bm, nil) // implicit zero-arg call
    }
    return val, nil
}
```

`evalOptionalMember` receives the identical change so that `post?.summary` also
invokes zero-arg methods implicitly.

Methods with one or more parameters are returned as a `boundMethod` value and
are callable only via an explicit `CallExpr` (i.e., `post.summary()` or
`router.linkFor("admin.dashboard")`). Accessing a non-zero-arg method without
parentheses returns the `boundMethod` value itself; this is equivalent to
referencing a function object in JavaScript and may be tested with `typeof`.

#### Option analysis

- ✅ Implicit zero-arg call: `post.Summary` in a template "just works" without
  forcing the author to know whether `Summary` is a field or a method.
- ✅ Parentheses also accepted: `post.Summary()` routes through `evalCall` →
  `boundMethod` branch with zero args — still valid.
- ⚠️ An author who assigns a zero-arg method result to a variable in scope and
  then later passes it to another expression would receive the return value, not
  the method. This is the expected Go mental model: calling `post.Summary` returns
  a string.
- ❌ Always require parentheses: forces a different syntax for Go method calls vs.
  field access. Template authors would need to know whether a name is a field or
  a method. This inconsistency is worse than the implicit-call rule.

**Verdict**: implicit zero-arg call is the correct default. Explicit `()` is
additionally allowed and produces the same result.

---

### 4.4 Method invocation in `evalCall`

#### Current state

`evalCall` (`expr/eval.go:470–487`) evaluates the callee node, asserts it to
`func(...any)(any, error)`, and calls it directly. Any other type causes:

```
value of type X is not callable
```

#### Proposed extension

`evalCall` gains a second branch after the existing type assertion:

```go
// pseudo-code — not implementation
func evalCall(n *CallExpr, scope map[string]any) (any, error) {
    callee, err := evalNode(n.Callee, scope)
    if err != nil { return nil, err }

    // Evaluate arguments eagerly (same as before).
    args := make([]any, len(n.Args))
    for i, arg := range n.Args {
        args[i], err = evalNode(arg, scope)
        if err != nil { return nil, err }
    }

    switch fn := callee.(type) {
    case func(...any) (any, error):
        return fn(args...)                 // existing path (builtins, RegisterBuiltin)

    case boundMethod:
        return invokeMethod(fn, args)      // new path (Go methods)

    default:
        return nil, fmt.Errorf("value of type %T is not callable", callee)
    }
}
```

`invokeMethod` handles type coercion and invocation, including the variadic case:

```go
// pseudo-code — not implementation
func invokeMethod(bm boundMethod, args []any) (any, error) {
    mt := bm.fn.Type()
    if bm.variadic {
        fixedCount := bm.numIn - 1 // last param is the variadic slice
        if len(args) < fixedCount {
            return nil, fmt.Errorf("method expects at least %d argument(s), got %d",
                fixedCount, len(args))
        }
        in := make([]reflect.Value, bm.numIn)
        // Coerce fixed arguments normally.
        for i := 0; i < fixedCount; i++ {
            paramType := mt.In(i)
            coerced, err := coerceToType(args[i], paramType)
            if err != nil {
                return nil, fmt.Errorf("argument %d: %w", i+1, err)
            }
            in[i] = coerced
        }
        // Gather remaining arguments into a slice of the variadic element type.
        varType := mt.In(fixedCount).Elem()
        varSlice := reflect.MakeSlice(reflect.SliceOf(varType), len(args)-fixedCount, len(args)-fixedCount)
        for j, arg := range args[fixedCount:] {
            coerced, err := coerceToType(arg, varType)
            if err != nil {
                return nil, fmt.Errorf("argument %d: %w", fixedCount+j+1, err)
            }
            varSlice.Index(j).Set(coerced)
        }
        in[fixedCount] = varSlice
        out := bm.fn.Call(in)
        return normaliseMethodResult(out, bm.numOut)
    }

    // Non-variadic path.
    if len(args) != bm.numIn {
        return nil, fmt.Errorf("method expects %d argument(s), got %d",
            bm.numIn, len(args))
    }
    in := make([]reflect.Value, bm.numIn)
    for i, arg := range args {
        paramType := mt.In(i)
        coerced, err := coerceToType(arg, paramType)
        if err != nil {
            return nil, fmt.Errorf("argument %d: %w", i+1, err)
        }
        in[i] = coerced
    }
    out := bm.fn.Call(in)
    return normaliseMethodResult(out, bm.numOut)
}
```

`coerceToType` converts expression-language scalar values to the required Go
parameter type:

```go
// pseudo-code — not implementation
func coerceToType(v any, t reflect.Type) (reflect.Value, error) {
    // If v is already assignable to t, use it directly.
    rv := reflect.ValueOf(v)
    if rv.IsValid() && rv.Type().AssignableTo(t) {
        return rv, nil
    }
    // Expression-language numeric coercion: float64 → int*, uint*, float*.
    if f, ok := v.(float64); ok {
        if t.Kind() >= reflect.Int && t.Kind() <= reflect.Int64 {
            return reflect.ValueOf(f).Convert(t), nil
        }
        if t.Kind() >= reflect.Uint && t.Kind() <= reflect.Uint64 {
            return reflect.ValueOf(f).Convert(t), nil
        }
        if t.Kind() == reflect.Float32 || t.Kind() == reflect.Float64 {
            return reflect.ValueOf(f).Convert(t), nil
        }
    }
    // string → string (already handled by assignability above; listed for clarity)
    // bool → bool
    // any/interface{}: wrap as-is
    if t.Kind() == reflect.Interface {
        if rv.IsValid() {
            return rv, nil
        }
        return reflect.Zero(t), nil
    }
    return reflect.Value{}, fmt.Errorf("cannot coerce %T to %s", v, t)
}
```

`normaliseMethodResult` converts the `[]reflect.Value` returned by
`reflect.Value.Call` to `(any, error)`:

```go
// pseudo-code — not implementation
func normaliseMethodResult(out []reflect.Value, numOut int) (any, error) {
    switch numOut {
    case 1:
        return out[0].Interface(), nil
    case 2:
        // Second return value must implement error.
        if errVal := out[1].Interface(); errVal != nil {
            return nil, errVal.(error)
        }
        return out[0].Interface(), nil
    default:
        return nil, fmt.Errorf("unsupported method return arity %d", numOut)
    }
}
```

#### Validation of `numOut` at `wrapMethod` time

`wrapMethod` (called in `accessMethod`) inspects the method signature and rejects
unsupported arities immediately:

```go
// pseudo-code — not implementation
func wrapMethod(m reflect.Value) (boundMethod, error) {
    mt := m.Type()
    numOut := mt.NumOut()
    // Zero-return-value methods are a hard error at both evaluation time (here)
    // and compile time (see §7). A method that returns nothing cannot be used
    // in an expression context.
    if numOut == 0 {
        return boundMethod{}, fmt.Errorf(
            "method returns no values; use a func(...any)(any,error) wrapper instead")
    }
    if numOut > 2 {
        return boundMethod{}, fmt.Errorf(
            "method returns %d values; only 1 or (value, error) is supported", numOut)
    }
    if numOut == 2 {
        errType := reflect.TypeOf((*error)(nil)).Elem()
        if !mt.Out(1).Implements(errType) {
            return boundMethod{}, fmt.Errorf(
                "method second return value must implement error, got %s", mt.Out(1))
        }
    }
    return boundMethod{
        fn:       m,
        numIn:    mt.NumIn(),
        numOut:   numOut,
        variadic: mt.IsVariadic(),
    }, nil
}
```

Errors from `wrapMethod` are returned as the error from `accessMember`; they
surface as expression evaluation errors that the template renderer reports.

#### Panic recovery scope

Panic recovery must **not** be placed inside `invokeMethod`. Doing so would
add overhead to every method invocation and would be too granular to catch
panics that originate elsewhere in expression evaluation.

Instead, a single `recover` is added at the top of the `Eval` function (and
the compiled-eval entry point, if separate). Any unexpected panic anywhere in
expression evaluation — including inside `invokeMethod`, `coerceToType`, or the
reflect call — is caught there and converted to an error, so it never reaches
the caller. This is consistent with the principle: "evaluating expr expressions
(whether compiled or not) should never panic in production."

```go
// pseudo-code — not implementation
func Eval(expr string, scope map[string]any) (_ any, retErr error) {
    defer func() {
        if r := recover(); r != nil {
            retErr = fmt.Errorf("expr: internal panic: %v", r)
        }
    }()
    // ... normal evaluation ...
}
```

The same deferred recover is added to the compiled-eval entry point.

**Verdict**: a `boundMethod` sentinel type keeps the change contained within
`evalCall` and `evalMember`; it does not alter the public API. Rejecting
unsupported arities at access time (not call time) means errors are raised as
soon as the name is resolved, not only when the expression is fully evaluated.

---

### 4.5 Interaction with `evalOptionalMember`

`evalOptionalMember` (`expr/eval.go:304–325`) is structurally identical to
`evalMember` except for the nullish short-circuit. It calls `accessMember`
directly. The same zero-arg implicit call logic must be added here:

```go
// pseudo-code — not implementation
val, err := accessMember(obj, key)
if err != nil { return nil, err }
if bm, ok := val.(boundMethod); ok && bm.numIn == 0 {
    return invokeMethod(bm, nil)
}
return val, nil
```

This ensures `post?.Summary` and `post?.Summary()` are both valid.

---

## 5. Syntax Summary

| Syntax                              | Meaning                                                                               |
|-------------------------------------|---------------------------------------------------------------------------------------|
| `obj.Method()`                      | Explicit call: invoke exported Go method `Method` on `obj` with no arguments.        |
| `obj.Method(arg1, arg2)`            | Explicit call: invoke exported Go method `Method` on `obj` with two arguments.       |
| `obj.method()`                      | Lowercase-initial alias: resolves to exported `Method` (first rune uppercased).      |
| `obj.method(arg1)`                  | Lowercase-initial alias with arguments.                                               |
| `obj.ZeroArgMethod`                 | Implicit call: zero-parameter method invoked automatically (parentheses optional).    |
| `obj.zeroArgMethod`                 | Implicit call via lowercase alias.                                                    |
| `obj?.Method(arg)`                  | Optional chaining: if `obj` is null/undefined, evaluates to `undefined`.             |
| `obj?.ZeroArgMethod`                | Optional chaining with implicit zero-arg call.                                        |
| `typeof obj.ZeroArgMethod`          | Returns the type of the method's **return value** (e.g. `"string"`). Zero-parameter methods behave like computed properties: `evalMember` invokes them immediately, so `typeof` sees the result, not the method. |
| `typeof obj.NonZeroArgMethod`       | Returns `"function"`. Methods with one or more parameters are returned as a `boundMethod` value without being called; `typeofValue` returns `"function"` for any `boundMethod`. |

**No new operators or parser changes are required.** The existing `MemberExpr`,
`OptionalMemberExpr`, and `CallExpr` AST nodes are sufficient. All syntax in the
table was already parseable; the change is in the evaluator.

---

## 6. Examples

### Example 1 — Zero-arg method, implicit call (no parentheses)

Go type in caller scope:

```go
type Post struct {
    Body string
}

func (p Post) Summary() string {
    if len(p.Body) > 100 { return p.Body[:100] + "…" }
    return p.Body
}
```

Scope:

```go
scope["post"] = Post{Body: "A long article body..."}
```

Template expression:

```html
<p>{{ post.Summary }}</p>
```

Resolution:
1. `accessStructField(rv, "Summary")` — no field named `Summary` → returns `UndefinedValue`.
2. `accessMethod(rv, "Summary")` → `MethodByName("Summary")` found → `boundMethod{numIn:0, numOut:1}`.
3. `evalMember` detects `numIn == 0` → implicit call → returns `"A long article body…"`.

Equivalent explicit form `post.Summary()` produces the same result via `evalCall`.

---

### Example 2 — Zero-arg method via lowercase alias

Same `Post` type. Template expression:

```html
<p>{{ post.summary }}</p>
```

Resolution:
1. `accessStructField(rv, "summary")` — no field or json tag `summary` → `UndefinedValue`.
2. `accessMethod(rv, "summary")`:
   - `MethodByName("summary")` — not found (unexported).
   - Alias: `string(unicode.ToUpper('s')) + "ummary"` = `"Summary"`.
   - `MethodByName("Summary")` found → `boundMethod{numIn:0, numOut:1}`.
3. `evalMember` detects `numIn == 0` → implicit call → returns the summary string.

---

### Example 3 — Method with an argument

Go type:

```go
type Router struct{}

func (r *Router) LinkFor(route string) string {
    return "/routes/" + route
}
```

Scope:

```go
scope["router"] = &Router{}
```

Template expression:

```html
<a :href="router.LinkFor('admin.dashboard')">Dashboard</a>
```

Resolution:
1. `evalMember` resolves `router` → `*Router`.
2. The outer `CallExpr` has callee `router.LinkFor` and args `["admin.dashboard"]`.
3. `evalNode(callee)` → `evalMember` → `accessMember(*Router, "LinkFor")`:
   - Not a struct (it's a pointer, dereferenced to struct, but methods checked on pointer).
   - `accessStructField(rv, "LinkFor")` — no field → `UndefinedValue`.
   - `accessMethod(rvForMethod, "LinkFor")` where `rvForMethod` is the original `*Router`.
   - `MethodByName("LinkFor")` on `*Router` → found → `boundMethod{numIn:1, numOut:1}`.
4. `evalCall`: callee is `boundMethod`, args = `["admin.dashboard"]` (string).
5. `invokeMethod`: `coerceToType("admin.dashboard", reflect.TypeOf(""))` → `reflect.Value` string.
6. `LinkFor("admin.dashboard")` called → returns `"/routes/admin.dashboard"`.

The `:href` attribute is set to `"/routes/admin.dashboard"`.

---

### Example 4 — Method with (value, error) return

Go type:

```go
type Formatter struct{}

func (f Formatter) FormatCurrency(amount float64) (string, error) {
    if amount < 0 { return "", fmt.Errorf("negative amount") }
    return fmt.Sprintf("$%.2f", amount), nil
}
```

Template expression:

```html
<span>{{ formatter.FormatCurrency(price) }}</span>
```

When `price` is `29.99` (float64 from scope), `invokeMethod` coerces it to
`float64` (already the right type), calls `FormatCurrency(29.99)`, and returns
`("$29.99", nil)`. `normaliseMethodResult` extracts `"$29.99"`.

If `price` is `-5`, the method returns `("", fmt.Errorf("negative amount"))`.
`normaliseMethodResult` propagates the error, which surfaces as a template
render error.

---

### Example 5 — Optional chaining with method call

```html
<a :href="page.router?.LinkFor('home')">Home</a>
```

If `page.router` is `nil` (null), `evalOptionalMember` short-circuits before
`accessMember` is called and returns `Undefined`. The `:href` attribute receives
`"undefined"`. If `page.router` is a valid `*Router`, resolution proceeds as in
Example 3.

This combination (`?.` followed by a named call) is already parseable. No AST
changes are required; `evalOptionalMember` is modified to handle `boundMethod`
in the same way as `evalMember`.

---

## 7. Implementation Sketch

### `expr/eval.go`

1. **New type `boundMethod`** (4 fields: `fn reflect.Value`, `numIn int`,
   `numOut int`, `variadic bool`). This is a private type. Approximately 8 lines.

2. **New function `wrapMethod(m reflect.Value) (boundMethod, error)`**: validates
   return arity (zero-return is a hard error), checks that the second return (if
   present) implements `error`, sets `variadic: mt.IsVariadic()`. Variadic methods
   are accepted. Approximately 20 lines.

3. **New function `accessMethod(rv reflect.Value, name string) (any, error)`**:
   two-pass lookup (exact then uppercase-alias). Calls `wrapMethod`. Returns
   `UndefinedValue` when no method is found. Approximately 20 lines.

4. **Modified `accessMember`**: preserve `rvForMethod` before pointer
   dereference; add method fallback after `accessStructField` returns
   `UndefinedValue`; add method-only branch in `default` case. Net diff:
   approximately +15 lines.

5. **Modified `evalMember`**: after `accessMember`, check for `boundMethod`
   with `numIn == 0` and invoke. Approximately +5 lines.

6. **Modified `evalOptionalMember`**: identical addition to `evalMember`.
   Approximately +5 lines.

7. **New function `coerceToType(v any, t reflect.Type) (reflect.Value, error)`**:
   handles `float64` → integer/float conversions, `string` → `string`, `bool` →
   `bool`, `any`-typed parameters, direct assignability. Approximately 30 lines.

8. **New function `normaliseMethodResult(out []reflect.Value, numOut int) (any, error)`**:
   handles 1-return and 2-return cases. Approximately 15 lines.

9. **New function `invokeMethod(bm boundMethod, args []any) (any, error)`**:
   arity check (at least `numIn-1` args for variadic, exactly `numIn` for
   non-variadic), `coerceToType` for fixed arguments, variadic-tail gathering
   into a `reflect.SliceOf` the element type, `bm.fn.Call(in)`,
   `normaliseMethodResult`. Approximately 40 lines.

10. **Modified `evalCall`**: add `case boundMethod:` branch before `default`.
    Approximately +3 lines.

11. **Modified `typeofValue`** (or equivalent): add `case boundMethod:` returning
    `"function"`. Approximately +2 lines.

12. **Modified `Eval` (and compiled-eval entry point)**: add a top-level
    `defer`/`recover` that converts any unexpected panic in expression evaluation
    to an error. This is the sole panic-recovery point; `invokeMethod` does not
    have its own recovery. Approximately +5 lines per entry point.

13. **`expr/compile.go` (compile-time check)**: the compiled-expression path must
    detect methods with zero return values at compile time and return a compile
    error rather than generating code that would panic or silently produce no
    value. Add a check analogous to the `wrapMethod` runtime check in the
    method-resolution step of the compiler. Approximately +10 lines.

Total estimated net additions: approximately 155 lines.

The `reflect` package is already imported. No new imports are required.

### `expr/ast.go`

No changes required. `MemberExpr`, `OptionalMemberExpr`, and `CallExpr` already
represent the necessary structure. The change is entirely in the evaluator.

### `expr/eval_test.go`

Add a new test function `TestMethodBindings` with sub-tests covering:

| Expression                          | Scope                                    | Expected result          |
|-------------------------------------|------------------------------------------|--------------------------|
| `post.Summary`                      | `post` with zero-arg `Summary()` method  | method return value      |
| `post.Summary()`                    | same                                     | same (explicit call ok)  |
| `post.summary`                      | same (lowercase alias)                   | same                     |
| `router.LinkFor("admin.dashboard")` | `router` with one-arg `LinkFor` method   | `"/routes/admin.dashboard"` |
| `router.linkFor("home")`            | same (lowercase alias)                   | `"/routes/home"`         |
| `formatter.FormatCurrency(9.99)`    | `formatter` with `(val, error)` method   | `"$9.99"`                |
| `formatter.FormatCurrency(-1)`      | same, error return                       | error propagated         |
| `obj?.Summary`                      | `obj` is nil                             | `UndefinedValue`         |
| `obj?.Summary`                      | `obj` is valid                           | method return value      |
| `post.Title`                        | `post` has both `Title` field and method | field value (field wins) |

Also extend `TestMemberAccess` with a backward-compatibility case confirming
that struct field access (including json-tag and lowercase alias) is unaffected.

---

## 8. Backward Compatibility

### Struct field access — no change

`accessStructField` is called before `accessMethod`. Templates that access
struct fields today continue to do so via exactly the same code path. The new
code is never reached for names that resolve to a field.

### `expr.Eval` and `expr.Compile` — no signature change

Both functions remain unchanged. Existing callers that only use field access or
`func(...any)(any, error)` builtins see identical behaviour.

### `expr.IsTruthy` — no change

### `expr.RegisterBuiltin` — no change

Functions registered via `RegisterBuiltin` continue to work. If a scope value
named `X` is a `func(...any)(any, error)`, it is still called via the existing
`evalCall` branch. The `boundMethod` branch is only reached for Go types that
are not already that function type.

### Template authors — additive change only

Templates that previously evaluated to `UndefinedValue` (because a method name
was used where no field existed) now evaluate to the method's return value. This
is a **semantic change for previously-broken templates**. Templates that were
working correctly (using field names that exist) are unaffected.

The only risk is a template that intentionally relied on `UndefinedValue` for a
name that happens to be a method on the type — for example, using `post.String`
to get `UndefinedValue` where `Post` has a `func (p Post) String() string`
method. Such a template would now call `String()` instead of returning
`UndefinedValue`. This edge case is unlikely in practice (authors do not write
templates that depend on a field being absent), but should be noted in release
documentation.

---

## 9. Alternatives Considered

### A. Always require explicit parentheses for method calls

Template authors would write `post.Summary()` instead of `post.Summary`.

**Rejected**: this imposes a burden on template authors to know whether a name is
a field or a method. The point of the expression language is to abstract over Go
types; forcing authors to know the difference defeats that abstraction. The
implicit zero-arg call rule is consistent with how Vue.js computed properties and
many template languages work.

### B. Register method wrappers in scope via a helper

Callers would call a helper such as `expr.Methods(obj)` to produce a
`map[string]func(...any)(any, error)` that could be merged into scope, without
any evaluator changes.

**Rejected**: this puts the burden on every Go caller to wrap every value they
put in scope. It is verbose, error-prone (methods added to a type after the
`expr.Methods` call are not automatically available), and defeats the goal of
letting Go types work naturally in templates. It is strictly worse than the
proposed approach for callers who use rich types.

### C. Use a `func(...any)(any, error)` adapter generated via reflection

`accessMethod` could return a `func(...any)(any, error)` closure that wraps the
reflect call, fitting into the existing `evalCall` path without a new type.

**Rejected**: this conflates the "method lookup" result with the "builtin function"
result, making it impossible to distinguish them for purposes of the
implicit-zero-arg-call rule. A zero-parameter method stored as
`func(...any)(any, error)` cannot be called implicitly from `evalMember` without
also implicitly calling every builtin function that happens to be accessed via
member syntax. Introducing `boundMethod` as a distinct sentinel type is the
minimal change that makes the two cases distinguishable.

### D. Extend the parser to support a `@method` prefix notation

Template authors would write `post.@Summary()` to explicitly select a method.

**Rejected**: introduces non-JavaScript syntax, breaking the principle that the
expression language is a subset of JavaScript expression syntax. The field-first
resolution order already makes the choice deterministic without new syntax.

---

## 10. Open Questions

1. **Variadic method support.** `func (f Formatter) Join(sep string, parts
   ...string) string` cannot be called with the current `coerceToType` approach
   because variadic parameter mapping requires knowing the boundary between the
   fixed and variadic arguments at call time. Should variadic support be added in
   a follow-up RFC, or deferred indefinitely?

   **Resolved.** Variadic methods are supported in this RFC. `wrapMethod` accepts
   variadic methods and sets `variadic: true` on the `boundMethod`. `invokeMethod`
   coerces the fixed arguments normally and gathers any remaining arguments into a
   slice of the variadic element type via `reflect.SliceOf`. See the updated §4.4
   pseudocode and §7 item 9 for details. Variadic methods from §3 Non-Goals have
   been removed.

2. **Pointer-receiver methods on non-addressable values.** `reflect.Value.MethodByName`
   on a non-addressable value of type `T` does not find methods defined on `*T`.
   The proposal stores `rvForMethod` before pointer dereference, which handles
   the case where the scope value is already a pointer. Should we attempt
   `reflect.New` + copy to make non-pointer values addressable for method lookup?
   *Tentative recommendation*: no. Automatically taking the address of a
   scope value would be surprising and could have unexpected side effects. Document
   that callers should store `&value` when pointer-receiver methods are required.
   Non-blocking.

   **Status**: still open / non-blocking. The tentative recommendation stands:
   do not auto-address scope values.

3. **Error messages for unsupported method signatures.** Currently `wrapMethod`
   returns an error for zero-return-value methods and methods with more than two
   return values. Should these be surfaced as `UndefinedValue` (silently skipped)
   or as hard errors?

   **Resolved.** Hard errors at both evaluation time and compile time.
   - At evaluation time: `wrapMethod` returns an error for zero-return-value
     methods; this error propagates from `accessMember` and surfaces as a template
     evaluation error. This behaviour is kept and must not be silenced.
   - At compile time: the compiled-expression path (`expr/compile.go`) must also
     detect a method with zero return values and return a compile error rather than
     generating code that would panic or produce no value silently. See §7 item 13.

4. **`boundMethod` nil-safety in `invokeMethod`.** If the receiver stored in
   `boundMethod.fn` becomes invalid between `accessMember` and `evalCall` (e.g.,
   the scope was mutated concurrently), `reflect.Value.Call` panics. Should
   `invokeMethod` recover from panics and return an error?

   **Resolved.** Panic recovery must not be placed inside `invokeMethod`. Adding a
   deferred `recover` to every method invocation is too granular and too costly.
   Instead, a single `recover` is added at the top of the `Eval` function (and
   the compiled-eval entry point), so any unexpected panic anywhere in expression
   evaluation — including inside `invokeMethod` — is converted to an error and
   never reaches the caller. This is consistent with the principle: "evaluating
   expr expressions (whether compiled or not) should never panic in production."
   See §4.4 "Panic recovery scope" and §7 item 12.

5. **`typeof` for `boundMethod` values.** Currently `typeofValue` returns
   `"object"` for any unknown type. Should `boundMethod` return `"function"`
   so that `typeof post.Summary === "function"` is `true` before the implicit
   call in `evalMember`?

   **Resolved.** `typeof` reflects the evaluation result:
   - `typeof post.Summary` — `Summary` takes no arguments, so `evalMember`
     invokes it immediately (implicit zero-arg call). `typeof` sees the *return
     value* (e.g. a `string`), and returns `"string"`. Zero-parameter methods
     behave like computed properties: `typeof` returns the type of their result.
   - `typeof router.LinkFor` — `LinkFor` takes an argument, so `evalMember`
     returns the `boundMethod` value without calling it. `typeofValue` returns
     `"function"` for any `boundMethod`. See §5 Syntax Summary and §7 item 11.
