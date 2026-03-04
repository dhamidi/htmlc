# Component Prop Discovery via Template AST Walking

## Summary

Add the ability to discover which props a component expects by walking its parsed template AST and collecting all variable references from expressions. This avoids parsing JavaScript entirely — the `<script>` section is not involved. A component can then be asked in Go code which props it uses, and the renderer validates that all expected props are provided at render time.

## Motivation

Currently props are fully implicit: any attribute passed to a component becomes a scope variable, and any variable referenced in a template that isn't in scope silently evaluates to `undefined`. There is no way to ask a component what data it expects, and missing props produce no errors or warnings.

## Design

### 1. Identifier Collection in the `expr` Package

Add a new exported function to the `expr` package:

```go
// CollectIdentifiers compiles src and returns the names of all Identifier
// nodes in the resulting AST, excluding property names in dot-notation
// member access (e.g. for "user.name" it returns ["user"], not ["user", "name"]).
// Bracket-notation property expressions are walked normally (e.g. for
// "items[idx]" it returns ["items", "idx"]).
// Duplicates are removed. Order is unspecified.
func CollectIdentifiers(src string) ([]string, error)
```

**AST walking rules:**
- `Identifier` → collect the name
- `MemberExpr` with `Computed=false` (dot notation) → walk `Object` only, skip `Property`
- `MemberExpr` with `Computed=true` (bracket notation) → walk both `Object` and `Property`
- `UnaryExpr` → walk `Operand`
- `BinaryExpr` → walk `Left` and `Right`
- `TernaryExpr` → walk `Condition`, `Consequent`, and `Alternate`
- `CallExpr` → walk `Callee` and all `Args`
- `ArrayLit` → walk all `Elements`
- `ObjectLit` → walk all property `Value` nodes (keys are string literals, not identifiers)
- Literal nodes (`NumberLit`, `StringLit`, `BoolLit`, `NullLit`, `UndefinedLit`) → skip

### 2. PropInfo Type

```go
// PropInfo describes a prop that a component's template references.
type PropInfo struct {
    // Name is the variable name as it appears in template expressions.
    Name string
    // Expressions lists the expression source strings where this prop is used.
    // For example, if a template has both {{ title }} and :class="title + ' bold'",
    // Expressions would contain ["title", "title + ' bold'"].
    Expressions []string
}
```

### 3. Component.Props() Method

```go
// Props walks the component's parsed template AST and returns the set of
// props (top-level variable references) that the template uses.
// Built-in identifiers (len) and renderer-injected variables ($slot) are
// excluded. Locally scoped variables from v-for are excluded.
func (c *Component) Props() []PropInfo
```

**Expression sources to scan** (these are all the places where `expr.Eval` is called during rendering):

| Source | How to extract the expression |
|---|---|
| `{{ expr }}` in text nodes | Match `\{\{(.*?)\}\}` regex on `TextNode.Data` |
| `:attr="expr"` | Attribute key starts with `:`, value is the expression |
| `v-bind:attr="expr"` | Attribute key starts with `v-bind:`, value is the expression |
| `v-if="expr"` | Attribute value |
| `v-else-if="expr"` | Attribute value |
| `v-show="expr"` | Attribute value |
| `v-text="expr"` | Attribute value |
| `v-html="expr"` | Attribute value |
| `v-for="vars in expr"` | Right side of ` in ` is the collection expression; left side declares local variables |

**Excluded identifiers:**
- `$slot` and any identifier starting with `$` (renderer-injected)
- `len` (built-in function from `expr.builtins`)

**Scope tracking for v-for:**
- `v-for="item in items"` → `item` is local to the v-for subtree
- `v-for="(item, index) in items"` → `item` and `index` are local
- `v-for="(value, key) in obj"` → `value` and `key` are local
- `v-for="(value, key, index) in obj"` → `value`, `key`, and `index` are local
- The collection expression (`items`, `obj`) is scanned for identifiers in the **parent** scope
- All child nodes of the v-for element are scanned with the v-for variables added to the set of locally scoped names

**Template walking algorithm:**

```
func collectProps(node, localVars) -> map[name][]expression:
    props = {}
    for each child of node:
        if child is TextNode:
            for each {{ expr }} match in child.Data:
                ids = expr.CollectIdentifiers(expr)
                for each id not in localVars and not excluded:
                    props[id].add(expr)
        if child is ElementNode:
            if child has v-for:
                vars, collExpr = parseVFor(v-for value)
                ids = expr.CollectIdentifiers(collExpr)
                for each id not in localVars and not excluded:
                    props[id].add(collExpr)
                newLocalVars = localVars ∪ vars
                scan attributes of child (except v-for) with localVars
                collectProps(child, newLocalVars)  // recurse with extended locals
            else:
                scan all directive/binding attributes for expressions
                collectProps(child, localVars)  // recurse with same locals
    return props
```

### 4. Missing Prop Handling at Render Time

#### 4.1. MissingPropFunc Type

```go
// MissingPropFunc is called when a prop expected by the component's template
// is not present in the render scope. It receives the prop name and returns
// a substitute value, or an error to abort rendering.
type MissingPropFunc func(name string) (any, error)
```

#### 4.2. Built-in MissingPropFunc Implementations

```go
// SubstituteMissingProp returns a placeholder string "MISSING PROP: <name>"
// for any missing prop. Use this during development to make missing data
// visible in the rendered output without aborting.
func SubstituteMissingProp(name string) (any, error) {
    return fmt.Sprintf("MISSING PROP: %s", name), nil
}
```

#### 4.3. Renderer Configuration

```go
// WithMissingPropHandler sets the function called when a prop expected by
// the component template is not found in the render scope.
// If not set, rendering returns an error for any missing prop.
func (r *Renderer) WithMissingPropHandler(fn MissingPropFunc) *Renderer
```

#### 4.4. Validation Logic

When `Renderer.Render(scope)` is called:

1. Call `r.component.Props()` to get the expected props.
2. For each `PropInfo`, check if `Name` exists as a key in `scope`.
3. If a prop is missing:
   - If `MissingPropHandler` is set: call it, inject the returned value into a copy of the scope.
   - If no handler is set: return `fmt.Errorf("missing prop %q (used in: %s)", name, expressions)`.
4. Proceed with rendering using the (possibly augmented) scope.

This validation applies at every component boundary. Child components get their own scope built from attributes in `renderComponentElement`, and that scope is validated against the child component's own `Props()`.

#### 4.5. Engine Integration

The `Engine` propagates the `MissingPropFunc` to every `Renderer` it creates internally. Add a configuration option:

```go
// WithMissingPropHandler sets the function called when any component rendered
// by this engine has a missing prop. If not set, missing props cause render errors.
func (e *Engine) WithMissingPropHandler(fn MissingPropFunc) *Engine
```

## Scope of Changes

### Files to modify:
- **`expr/identifiers.go`** (new): add `CollectIdentifiers` function
- **`component.go`**: add `PropInfo` type and `Component.Props()` method
- **`renderer.go`**: add `MissingPropFunc`, `SubstituteMissingProp`, `WithMissingPropHandler`, and the validation logic in `Render()` and `renderComponentElement()`
- **`engine.go`**: add `WithMissingPropHandler` option and propagation

### New test coverage:
- `expr/identifiers_test.go`: test identifier collection for various expression forms (simple, member access, nested, calls, v-for-like)
- `component_test.go`: test `Props()` for templates with interpolations, bindings, directives, v-for scoping, nested v-for
- `renderer_test.go`: test missing prop error, `SubstituteMissingProp` behavior, custom handler
- `engine_test.go`: test engine-level missing prop handler propagation

## Examples

### Querying Props

```go
comp, _ := htmlc.ParseFile("card.vue", `
<template>
  <div :class="cardClass">
    <h1>{{ title }}</h1>
    <ul>
      <li v-for="item in items">{{ item.name }}</li>
    </ul>
  </div>
</template>`)

props := comp.Props()
// props contains:
// - {Name: "cardClass", Expressions: ["cardClass"]}
// - {Name: "title",     Expressions: ["title"]}
// - {Name: "items",     Expressions: ["items"]}
// "item" is NOT included (it's a v-for local variable)
```

### Rendering with Missing Props

```go
// Default: error on missing prop
_, err := htmlc.NewRenderer(comp).Render(map[string]any{"title": "Hello"})
// err: missing prop "cardClass" (used in: cardClass)

// With placeholder substitution
out, _ := htmlc.NewRenderer(comp).
    WithMissingPropHandler(htmlc.SubstituteMissingProp).
    Render(map[string]any{"title": "Hello"})
// out contains "MISSING PROP: cardClass" and "MISSING PROP: items" in the output

// With custom handler
out, _ := htmlc.NewRenderer(comp).
    WithMissingPropHandler(func(name string) (any, error) {
        return nil, nil  // treat all missing props as nil
    }).
    Render(map[string]any{"title": "Hello"})
```
