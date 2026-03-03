# Set up Go module and package skeleton

## Description

Initialize the Go module and create the empty package files that establish the directory structure defined in the spec: `go.mod`, `engine.go`, `component.go`, `renderer.go`, `style.go`, and the `expr/` subpackage with `ast.go`, `lexer.go`, `parser.go`, `eval.go`. Each file should contain only its package declaration and a brief doc comment. The `go.mod` should declare module `github.com/dhamidi/htmlc` with Go 1.22+.

## Acceptance Criteria

- `go.mod` exists and `go build ./...` succeeds with zero errors
- Every file listed in the spec's package layout exists with the correct package name
- `go vet ./...` passes

---

# Implement expression AST node types

## Description

Define the AST node types in `expr/ast.go` that the parser will produce and the evaluator will consume. This includes nodes for: literals (number, string, bool, null, undefined), unary expressions, binary expressions, ternary/conditional expressions, member access (dot and bracket), function calls, array literals, object literals, and identifiers. Every node type must satisfy a common `Node` interface. Also define the `Expr` interface (`Eval(scope map[string]any) (any, error)`) specified in the public API.

## Acceptance Criteria

- All node types from the spec's expression language (§1) are represented
- The `Expr` interface is defined with `Eval(scope map[string]any) (any, error)`
- `go build ./expr` succeeds

---

# Implement expression lexer

## Description

Build the tokeniser in `expr/lexer.go`. It must produce tokens for: integers, floats, single- and double-quoted strings, identifiers and keywords (`true`, `false`, `null`, `undefined`, `typeof`, `void`, `in`, `instanceof`), all operator symbols listed in the spec (arithmetic, comparison, logical, bitwise, nullish coalescing, exponentiation), parentheses, brackets, braces, dot, comma, colon, `?`, and EOF. The lexer should report the position of each token for error messages.

## Acceptance Criteria

- A test tokenises `user.name === 'admin' ? items[0] : null` into the correct token sequence
- All operator tokens from §1.2 and §1.3 are recognised
- String literals with single and double quotes are handled
- Integer and float literals including leading-dot floats (`.5`) are handled
- `typeof` and `void` are tokenised as keyword tokens, not identifiers

---

# Implement expression parser

## Description

Write a recursive-descent parser in `expr/parser.go` that consumes the token stream from the lexer and produces the AST defined in `ast.go`. The parser must enforce the operator precedence table from §1.3 (exponentiation is right-associative, all others left-associative). It must handle: grouping with parentheses, unary prefix operators, all binary operators, ternary conditional, member access (dot and bracket), function calls, array literals, object literals, and the `Compile(src string) (Expr, error)` public function.

## Acceptance Criteria

- `Compile("a + b * c")` produces an AST where `*` is nested deeper than `+`
- `Compile("a ** b ** c")` produces a right-associative tree
- `Compile("x ? y : z")` produces a conditional node
- `Compile("obj.foo[0](a, b)")` produces chained member-access and call nodes
- `Compile("[1, 2, 3]")` and `Compile("{ a: 1, 'b': 2 }")` produce array/object literal nodes
- Malformed expressions return a descriptive error

---

# Implement expression evaluator

## Description

Implement `Eval` in `expr/eval.go` that walks the AST and evaluates it against a `map[string]any` scope. This must handle: all literal types, identifier lookup from scope, member access on Go maps/structs/slices (including struct `json` tag support), function calls on `func(...any) (any, error)` values, all unary operators (!, -, +, ~, typeof, void), all binary operators with correct JS-style semantics (loose equality coercion for `==`/`!=`, short-circuit for `&&`/`||`/`??`, `in` for map key membership), and the ternary operator. Also implement the convenience `Eval(src string, scope map[string]any) (any, error)` wrapper.

## Acceptance Criteria

- `Eval("1 + 2", nil)` returns `3`
- `Eval("user.name", scope)` returns the correct value from a nested map
- `Eval("items[0]", scope)` returns the first element of a slice
- `Eval("0 == false", nil)` returns `true` (loose equality)
- `Eval("0 === false", nil)` returns `false` (strict equality)
- `Eval("null == undefined", nil)` returns `true`
- `Eval("a && b", scope)` short-circuits and returns the correct operand (not a bool)
- `Eval("a ?? 'default'", scope)` returns `"default"` when `a` is `nil`
- `Eval("typeof x", scope)` returns the correct type string
- Struct fields are accessible by exported name and by `json` tag
- `Eval("fn(1, 2)", scope)` calls a Go function from scope
- Out-of-bounds index or nil map access returns an error

---

# Implement SFC parser

## Description

Build the `.vue` Single File Component parser in `component.go`. It must extract the `<template>`, `<script>`, and `<style>` top-level sections from a `.vue` file. The `<style>` tag's `scoped` attribute must be detected. The template content should be parsed into an HTML node tree (using `golang.org/x/net/html` or a similar approach). Define a `Component` struct holding the parsed template, raw script content, raw style content, the scoped flag, and the source file path.

## Acceptance Criteria

- Parsing a `.vue` file with all three sections extracts each section's content correctly
- Parsing a file with only `<template>` succeeds (script and style are optional)
- The `scoped` attribute on `<style scoped>` is detected and stored
- The template HTML is parsed into a tree of nodes that can be walked
- Malformed input (e.g., unclosed `<template>`) returns a descriptive error

---

# Implement template renderer — interpolation and v-text/v-html

## Description

Create the template renderer in `renderer.go` that walks the parsed HTML node tree and produces output HTML. Start with mustache interpolation (`{{ expr }}`), `v-text`, and `v-html`. Interpolated expressions and `v-text` must be HTML-escaped. `v-html` inserts raw HTML. The renderer takes a component and a `map[string]any` data scope.

## Acceptance Criteria

- `{{ user.name }}` renders as the HTML-escaped value of `user.name` from scope
- `{{ price * qty }}` evaluates the expression and renders the result
- `<p v-text="msg"></p>` renders as `<p>escaped content</p>`
- `<div v-html="raw"></div>` renders as `<div><b>bold</b></div>` without escaping the inner HTML
- Whitespace inside `{{ }}` is trimmed

---

# Implement v-if / v-else-if / v-else directives

## Description

Add conditional rendering to the template renderer. `v-if` evaluates its expression; if truthy the element is rendered, otherwise it is omitted. `v-else-if` and `v-else` must follow a preceding `v-if` or `v-else-if` sibling. Only the first truthy branch is rendered. The directives must also work on `<template>` wrapper elements (rendering children only, not the `<template>` tag itself).

## Acceptance Criteria

- `<div v-if="true">yes</div>` renders `<div>yes</div>`
- `<div v-if="false">yes</div>` produces no output
- A `v-if`/`v-else-if`/`v-else` chain renders only the first truthy branch
- `v-else` renders when all preceding conditions are falsy
- `<template v-if="show"><p>a</p><p>b</p></template>` renders only the `<p>` tags, not a `<template>` element
- `v-else` not preceded by `v-if` returns a parse/render error

---

# Implement v-for directive

## Description

Add list rendering to the template renderer. `v-for` supports the forms: `item in items`, `(item, index) in items`, `(value, key) in obj`, and `n in 5`. For arrays/slices, provide element and optional index. For maps/objects, provide value and optional key. For integers, iterate 1..n. The `:key` attribute should be evaluated and rendered as `data-key`. The `v-for` directive must work on `<template>` wrapper elements.

## Acceptance Criteria

- `v-for="item in items"` renders one copy per array element with `item` in scope
- `v-for="(item, index) in items"` provides both item and index
- `v-for="(value, key) in obj"` iterates map entries
- `v-for="n in 5"` renders 5 copies with n = 1..5
- `:key="item.id"` renders as `data-key="<value>"`
- `<template v-for="...">` renders only children per iteration
- Empty array produces no output

---

# Implement attribute binding (v-bind / :attr)

## Description

Add dynamic attribute binding to the renderer. `:attr="expr"` evaluates the expression and sets the attribute. Special handling is needed for: `:class` with object syntax (`{ active: bool }`) and array syntax (`['a', cond ? 'b' : '']`); `:style` with object syntax (`{ color: 'red' }`); boolean attributes (`disabled`, `checked`, `selected`, `readonly`, `required`, `multiple`, `autofocus`, `open`) which are rendered without a value when truthy and omitted when falsy. Static attributes and dynamic attributes on the same element must be merged.

## Acceptance Criteria

- `:href="url"` renders as `href="<value>"`
- `:class="{ active: true, hidden: false }"` renders as `class="active"`
- `:class="['a', condition ? 'b' : '']"` renders as `class="a b"` or `class="a"` based on condition
- `:style="{ color: 'red', fontSize: '14px' }"` renders as `style="color:red;font-size:14px"`
- `:disabled="false"` omits the attribute; `:disabled="true"` renders `disabled`
- Static `class="foo"` and `:class="{ bar: true }"` merge to `class="foo bar"`

---

# Implement v-show, v-pre, v-once, and client-side directive pass-through

## Description

Add the remaining simple directives. `v-show`: when falsy, add `style="display:none"` (merge with existing style if present); when truthy, render normally. `v-pre`: emit the element and all descendants verbatim without any interpolation or directive processing. `v-once`: process normally (equivalent to standard rendering in server-side context). Client-side directives (`v-model`, `v-on`/`@`) must be preserved as-is in the output HTML, not stripped or evaluated.

## Acceptance Criteria

- `<div v-show="false">x</div>` renders `<div style="display:none">x</div>`
- `<div v-show="true">x</div>` renders `<div>x</div>`
- `<div v-pre>{{ raw }}</div>` renders literally as `<div>{{ raw }}</div>`
- `v-pre` skips processing for all descendant elements too
- `<input v-model="name">` is preserved as `<input v-model="name">` in output
- `<button @click="handler">` is preserved as-is in output
- `v-once` renders expressions normally

---

# Implement style scoping

## Description

Build the style processing in `style.go`. Each component gets a scope ID: `data-v-` + first 8 hex chars of FNV-1a hash of the component file path. For `<style scoped>`, every rendered HTML element of that component receives the scope attribute, and every CSS rule is rewritten by appending `[data-v-XXXXXXXX]` to the last simple selector. Global styles (`<style>` without `scoped`) are included unchanged. Provide a function to collect all style contributions during a render.

## Acceptance Criteria

- Scope ID for a given file path is deterministic and matches the FNV-1a spec
- `.card { ... }` in a scoped style becomes `.card[data-v-a1b2c3d4] { ... }`
- `h2 .title { ... }` becomes `h2 .title[data-v-a1b2c3d4] { ... }` (appended to last simple selector)
- Every element rendered by a scoped component has the `data-v-XXXXXXXX` attribute
- Global styles are passed through without modification
- Multiple components' styles are collected into a single list during render

---

# Implement component composition and slots

## Description

Add component rendering to the template walker. When a tag name matches a registered component (PascalCase or kebab-case), render it by: evaluating prop bindings and passing them as the child component's scope, passing static attributes as string props, capturing inner content as the default slot, and rendering the child component's template. `<slot />` in the child template emits the caller's inner content. The renderer must detect unknown component names and return an error.

## Acceptance Criteria

- `<Card :title="t">content</Card>` renders Card's template with `title` in scope and `content` as the default slot
- `<slot />` in a child component emits the caller's inner content
- Static attributes like `<Card class="x">` pass `class` as a prop string
- kebab-case `<my-card>` resolves to a component registered as `MyCard` or `my-card`
- Unknown component tag name returns an error
- Components can nest: a component's template may use other components

---

# Implement Engine with component registry, auto-discovery, and reload

## Description

Build the `Engine` in `engine.go`. It accepts `Options` with `ComponentDir` and `Reload`. On creation, it recursively scans `ComponentDir` for `*.vue` files and registers them by filename (without extension). `Register(name, path)` allows manual registration. With `Reload: true`, the engine checks mtime before each render and re-parses changed files. Implement `RenderPage` (injects collected `<style>` before `</head>` or prepends it), `RenderFragment` (prepends `<style>` block), and `ServeComponent` (returns an `http.HandlerFunc`).

## Acceptance Criteria

- `New(Options{ComponentDir: dir})` discovers and registers all `.vue` files recursively
- `Card.vue` registers as `Card`; `ui/Button.vue` registers as `Button`
- Duplicate names: last file in alphabetical traversal order wins
- `Register("Alias", path)` manually registers a component
- `RenderPage` returns HTML with a `<style>` block before `</head>`
- `RenderPage` on HTML without `<head>` prepends the style block
- `RenderFragment` prepends the collected `<style>` block to the output
- `ServeComponent` returns an `http.HandlerFunc` that writes `text/html; charset=utf-8`
- With `Reload: true`, modifying a `.vue` file causes re-parse on next render
- Rendering an unknown component name returns an error

---

# Add end-to-end integration tests

## Description

Create integration tests that exercise the full pipeline: write `.vue` files to a temp directory, create an engine, and render pages/fragments. Cover: a page with conditional rendering, list rendering, attribute binding, mustache interpolation, scoped styles, nested components with slots, and the HTTP handler. These tests validate that all subsystems (expression evaluator, SFC parser, renderer, style scoping, engine) work together correctly.

## Acceptance Criteria

- A test renders a component with `v-if`, `v-for`, `:class`, `{{ }}`, and scoped styles and asserts the full HTML output
- A test renders a parent component containing a child component with a slot and asserts the composed output
- A test uses `ServeComponent` via `httptest` and asserts status code, content-type, and body
- A test verifies that `Reload: true` picks up a modified component file
- All tests pass with `go test ./...`
