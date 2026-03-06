/*
# Explanation: Design and Rationale

This section answers _why_ the expr package is built the way it is. For syntax
tables and API details, see the reference in doc.go.

## Purpose and Scope

The expr package exists because htmlc templates need to evaluate Vue.js-style
attribute bindings (`:attr="expr"`, `{{ interpolation }}`, `v-if="expr"`) on
the server side, in Go. It intentionally implements only the subset of
JavaScript expression semantics required by those bindings — nothing more —
because a full JS engine would introduce enormous complexity and dependency
weight that would never be justified by the narrow use case. Constructs that
have no place in a declarative template binding (assignment, closures, `new`,
`class`) are excluded by design, not by accident. This scoping decision is what
keeps the evaluator small enough to audit and reason about.

## Pipeline Architecture

Expression evaluation is split into three sequential stages: the Lexer turns
source text into a flat token stream, the Parser turns that stream into an AST,
and Eval walks the AST against a scope map to produce a value. The stages are
separate because different callers need different amounts of the pipeline.
CollectIdentifiers, for example, only needs the Lexer and Parser — it walks the
AST to find referenced names without evaluating anything. If the pipeline were
fused into a single function, that reuse would be impossible. Separate stages
also make testing tractable: each stage can be exercised in isolation, so a
failing test immediately narrows the fault to one layer rather than requiring
the entire pipeline to be traced.

## JavaScript-like Semantics in Go

The evaluator deliberately mirrors JavaScript semantics rather than inventing
its own, because the expressions being evaluated are authored by developers who
expect JS behaviour. Surprising them with different rules would introduce
subtle, hard-to-debug discrepancies between client-side (Vue) and server-side
(htmlc) rendering. All numeric literals are stored as float64 for this reason:
JavaScript has only one numeric type, so matching that model means numeric
expressions produce identical results on both sides. Similarly, the truthy/falsy
rules (empty string and 0 are falsy; any non-zero number or non-empty string is
truthy) follow JavaScript exactly, because Go's own truthiness rules differ and
applying them would silently break `v-if` conditions that work correctly in the
browser.

## Scope as map[string]any

The evaluator receives the template's data as a plain `map[string]any` rather
than a typed struct or an interface, because the template engine constructs
scopes dynamically — component props, loop variables, and slot data are all
assembled at runtime from heterogeneous sources. A typed struct would require
code generation or reflection-heavy wrappers, adding complexity without
meaningful benefit inside a package whose only consumers are other parts of
htmlc. The trade-off is that compile-time checking of scope keys is impossible;
a misspelled variable name in a template silently evaluates to UndefinedValue.
That cost is accepted because it matches how Vue templates themselves behave,
and because the primary defence against misspellings is the template compiler's
static analysis (CollectIdentifiers), not the evaluator.

## UndefinedValue vs nil

Two absent-value sentinels exist because null and undefined have distinct
semantics in JavaScript, and collapsing them onto a single Go nil would lose
information that matters for correctness. `nil` in the evaluator represents an
intentionally-null value — the result of a `null` literal or a JSON null field —
whereas `UndefinedValue` represents a missing key: an identifier that was not
found in the scope map or the built-ins. The distinction is observable via `===`
and `typeof`, and it matters for truthiness: both are falsy, but `null == undefined`
is true under abstract equality while `null === undefined` is false, matching
the JavaScript specification. Collapsing them would make it impossible to
distinguish a scope variable explicitly set to null from a variable that was
never provided at all, which would break expressions like `val ?? "default"`.

## Why a Subset, Not Full JS

Restricting the evaluator to side-effect-free expressions — no assignment, no
increment, no closures, no `new` — is what makes server-side rendering safe and
predictable. Because the evaluator is stateless, the same expression evaluated
twice against the same scope always produces the same result. This property is
essential for server-side rendering, where a template may be evaluated
concurrently across many requests; if expressions could mutate scope or external
state, race conditions and non-deterministic output would follow. The constraint
is enforced at parse time: the parser rejects assignment operators and other
stateful constructs before evaluation begins, so there is no way to write a
template expression that silently modifies shared state.
*/
package expr
