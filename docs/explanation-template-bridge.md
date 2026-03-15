# Explanation: Why bidirectional — the design of the template bridge

This document explains the design trade-offs behind the htmlc ↔ `html/template` bridge. It is for developers who want to understand the reasoning, not just use the feature.

---

## The interoperability problem

Go web applications accumulate `html/template` partials over time. Navigation bars, form elements, error pages, email layouts — each tied to framework middleware that injects CSRF tokens, session data, and request context. The middleware is written around `html/template`'s `*Template` type; changing that type means rewriting the middleware.

At the same time, teams want to adopt htmlc's component model for new UI work: reusable `.vue` files, scoped styles, and a structured prop interface. The two goals are in tension. There is no safe, incremental path from one to the other if the two systems cannot interoperate.

The bridge exists to make the transition incremental. A team can adopt htmlc for new components without touching existing templates. Existing templates can appear as leaf nodes inside htmlc trees. New `.vue` components can be exported as `*html/template.Template` for libraries that require the stdlib type. Neither system needs to own everything.

---

## Why conservative mapping wins

The first design question was how much of each template language to translate. The answer — translate only constructs with unambiguous equivalents, and error on everything else — was chosen for two reasons.

**Silent incorrectness is worse than a clear error.** A converter that tries to translate complex expressions produces output that may compile and render, but produces wrong HTML for edge-case inputs. A bug of this kind often survives testing and surfaces in production. An error at conversion time, by contrast, forces the developer to confront the incompatibility immediately and fix it deliberately.

**Symmetry between the two directions.** The `vue→tmpl` converter and the `tmpl→vue` converter apply the same policy: error on the first unsupported construct, produce no partial output. This means the error contract is predictable — if a construct is listed in the mapping table (§3.1 of the reference), it works in both directions. If it is not listed, it fails in both directions, loudly.

This is the same trade-off that Go's `encoding/json` package makes when it encounters an unrecognised type: return an error rather than guess. The alternative — silently drop or partially translate the unsupported construct — would be a source of subtle bugs that are difficult to attribute to the conversion layer.

---

## The scope-loss problem in `v-for`

In htmlc's expression evaluator, `v-for="item in list"` binds `item` to the current element and leaves all outer-scope variables accessible:

```html
<ul>
  <li v-for="post in posts">{{ post.title }} — by {{ site.name }}</li>
</ul>
```

`site.name` is accessible because htmlc evaluates expressions against a flat scope dictionary. There is no scoping boundary at the loop.

In Go's `html/template`, `{{ range .posts }}` rebinds `.` to the current element. The outer data map is no longer `.`; it is gone. Accessing `site.name` inside the range block requires either `$.site.name` (the Go template `$` root shorthand) or restructuring the data.

The bridge cannot use `$` automatically because htmlc's expression language does not have a `$` root accessor. Introducing one would change the expression language, which is explicitly out of scope for this RFC. The two systems have genuinely different scoping semantics, not just syntax differences.

The consequence is that a `v-for` body that references outer-scope variables cannot be automatically converted. The converter errors rather than producing output that silently drops `site.name` or replaces it with an empty string. The fix — embedding outer-scope data into each item struct — is mechanical and easy to apply once the error points to the problem.

This is not a limitation of the bridge that will be removed later. It is a reflection of a semantic difference between the two languages. Restructuring the data is the correct fix, not a workaround to be replaced by a smarter converter.

---

## Truthiness and what it means for `v-if`

htmlc inherits JavaScript's truthiness rules: `false`, `0`, `""`, `null`, and `undefined` are falsy; everything else, including empty arrays and empty objects, is truthy.

Go's `html/template` uses Go's truthiness rules: `false`, `0`, `""`, `nil`, empty slices, and empty maps are all falsy.

The divergence is narrow — it only affects empty slices and empty maps — but it is silent. A `v-if="items"` condition that checks whether a list is non-empty behaves correctly in htmlc (truthy when the list exists but is empty) and incorrectly in the compiled `html/template` (falsy when the list is empty). No error is raised; the wrong branch is taken.

The bridge emits a warning rather than an error for `v-if` conditions because the condition may be safe even when its static type cannot be verified. A `bool` field is always safe. A `string` field is safe if the semantics are "is this non-empty?". The warning prompts the developer to verify that the data they pass to the compiled template matches Go's truthiness rules for their specific use case.

When in doubt: pass a `bool`. Compute `len(items) > 0` in the handler and pass the result. This eliminates the truthiness question entirely.

---

## The props barrier

htmlc allows sub-component calls with static prop values:

```html
<Card title="Welcome" variant="primary" />
```

These static values are substituted at render time by the expression evaluator. In a compiled `html/template`, there is no equivalent mechanism: `html/template` does not accept literal values in `{{ template }}` calls. It passes a single data value (`.`) to the sub-template.

The bridge could try to embed static props into the `{{ define }}` block by inlining them as `{{ $title := "Welcome" }}` and similar assignments — but this requires variable assignments, which `tmpl→vue` explicitly does not support (and for good reason: variable assignments in Go templates complicate scope reasoning). More importantly, it would produce a template that cannot be called with different prop values from different call sites, which defeats the purpose of a sub-component.

The correct model is the data-at-root pattern: instead of passing props as attributes on the component call, pass a data map to the root template that contains all the values the component tree needs. The sub-component reads from `.` directly. This is what `html/template`'s `{{ template "Name" . }}` means: pass the full current scope to the named template.

The constraint that sub-component calls must have zero static props is therefore not a gap to be filled; it is a reflection of what `html/template` can express. The error at conversion time prevents a developer from unknowingly writing a component that will silently ignore its props when compiled.

---

## Incremental adoption as a design principle

The bridge is not designed to replace full migration. Its purpose is to make full migration unnecessary as a prerequisite for starting.

`RegisterTemplate` lets a team use any existing `*html/template.Template` as a leaf node in an htmlc component tree. The team writes new features as `.vue` components. Legacy partials remain in their original form. The migration can happen component by component, driven by natural product changes, not by a big-bang rewrite project.

`CompileToTemplate` lets a team export any htmlc component to a library that requires `*html/template.Template`. The team writes new components in `.vue` and calls into those libraries without duplication.

The two directions are not symmetric in production use. `CompileToTemplate` is the primary direction for new development — compile a `.vue` component for use in stdlib code. `RegisterTemplate` is the primary direction for migration — bring legacy templates into the htmlc world. The CLI tools (`vue-to-tmpl`, `tmpl-to-vue`) serve one-time conversion tasks: generating a starting `.vue` file from a legacy template, or generating a standalone `.html` file from a `.vue` component for use with `go:embed` or a CDN.

This design deliberately avoids the "rewrite everything" approach. Migrations succeed when they reduce risk at each step. The bridge reduces risk by keeping both systems running simultaneously and letting the team move at whatever pace the product allows.
