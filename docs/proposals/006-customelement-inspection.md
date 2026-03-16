# RFC 006 Inspection Report: Custom Element Compilation

**Date**: 2026-03-16
**RFC**: [006-customelement.md](006-customelement.md)
**Reviewer**: Inspect Stage

---

## What the Feature Enables

RFC 006 closes the gap between `htmlc`'s server-side rendering pipeline and client-side interactivity. Today, authors who need even the simplest interactive island — a tab switcher, a live counter, a dismissible alert — must leave the `.vue` file ecosystem entirely and write raw Web Components boilerplate or wire in a JS framework. This RFC gives them a single, standards-based path that lives inside the component file:

- **Progressive enhancement**: `<template>` provides a fully rendered, SEO-accessible, no-JS-compatible HTML structure; `<script customelement>` adds behaviour after the browser has parsed the DOM. Authors never ship a broken page to users with JS disabled.
- **Zero runtime overhead**: the emitted script is a plain `class extends HTMLElement` with a `customElements.define` call. No framework runtime, no VDOM, no reactive system is loaded. The feature is as thin as the browser's own Web Components API.
- **Component co-location**: interactivity lives in the same `.vue` file as the template. Renaming, moving, or deleting the component file keeps the behaviour and the markup in sync automatically.
- **Automatic deduplication**: the same component rendered 50 times in a list page emits its `<script>` block exactly once. Authors do not need to manage this manually.
- **Deterministic naming**: `DatePicker.vue` always becomes `htmlc-date-picker` / `HtmlcDatePicker`. Tag names are predictable and collision-resistant without a registry.

---

## What Gaps Remain

### Critical gaps (should resolve before implementation)

1. **Per-page opt-in is impossible**: `<script customelement>` is component-scoped, not usage-scoped. An author who wants analytics tracking on `Button.vue` for one page only must create a wrapper component. The RFC acknowledges this but does not provide a worked example or a roadmap for a finer-grained opt-in mechanism. Before implementation, the expected pattern for wrapper composition should be documented in §6.

2. **Template root element is not validated**: the RFC does not require or verify that the `<template>` block's root element is the custom element tag name (`<htmlc-counter>`). Authors may write `<div>` as the root and be confused when the element is never upgraded. A lint-time check or documentation rule is needed.

3. **`FlushCustomElements` concurrency model is unspecified**: §7 defers to `Engine` exposing `FlushCustomElements()`, but in a concurrent Go HTTP server, multiple render passes run simultaneously. The collector must be request-scoped (e.g., carried in `context.Context`). The current sketch implies a method on `Engine` with no concurrency mechanism — this is a design hole that must be resolved before implementation.

4. **Duplicate tag name across namespaced components**: `blog/Counter.vue` and `admin/Counter.vue` both produce `htmlc-counter`. Open Question 5 proposes a startup-time error. This is the correct resolution, but it is labelled "blocking" and must be designed and implemented before the feature ships — otherwise two teams in the same codebase will silently override each other's custom element definitions in the browser.

### Moderate gaps (can be addressed in follow-up)

5. **`RenderFragment` flush contract is underdefined**: the RFC states callers must flush manually but shows no example. A `RenderFragmentWithElements() (html, scripts template.HTML, err error)` API (or returning the collector alongside the fragment) would make this ergonomic. Without it, fragment-based page assembly is error-prone.

6. **CDN / URL-based delivery is deferred but has no implementation plan**: Option B (§4.5) is the right long-term answer for production deployments. The "when to write asset files" question (startup vs. build step vs. first render) is non-trivial and should be explored in a follow-up RFC rather than left entirely open.

7. **`observedAttributes` auto-wiring is deferred**: for authors used to Vue's reactivity, the expectation that component props automatically reflect as `observedAttributes` is strong. The manual pattern should be shown in §6; the auto-wiring RFC should be explicitly numbered as a planned successor.

8. **CSP nonce propagation path is incomplete**: Open Question 4 raises the need for a nonce parameter on `FlushCustomElements`. The recommendation is sound, but the mechanism for threading a per-request nonce through the render pipeline (presumably via `context.Context`) is not described.

### Minor gaps (documentation / polish)

9. **Acronym handling in tag-name derivation**: `XMLParser.vue` → `htmlc-x-m-l-parser` is technically correct but ugly. The `tag="..."` override resolves it, but the RFC's derivation algorithm section does not mention this limitation or recommend the override.

10. **Idempotency of `FlushCustomElements` is not specified**: the test stage finding notes this; it should be resolved in the spec (non-destructive recommended) before implementation.

---

## Prioritised List of Open Questions to Resolve Before Implementation

| Priority | Question | Blocking? |
|----------|----------|-----------|
| 1 | How does `CustomElementCollector` integrate with concurrent request handling? (request-scoped via `context.Context`?) | **Blocking** |
| 2 | What is the startup-time behavior for duplicate tag names across namespaced components? | **Blocking** |
| 3 | Is `FlushCustomElements` non-destructive (idempotent reads) or destructive (drains the collector)? | **Blocking** |
| 4 | Should `RenderPage` auto-flush before `</body>` by default, and is this opt-out? | **Blocking** |
| 5 | Is the `<template>` root element required to match the custom element tag name? | **Blocking** |
| 6 | How is a per-request CSP nonce threaded into `FlushCustomElements`? | Non-blocking (v1 can omit nonce support) |
| 7 | Should `FlushCustomElements` be exposed as a Go template function or only as a method? | Non-blocking |
| 8 | What is the `RenderFragment` ergonomic API for callers who need the flushed scripts? | Non-blocking |
| 9 | Should single-character or purely numeric component file names be a load-time error? | Non-blocking |
| 10 | What is the timeline for Option B (URL-based delivery)? Should a follow-up RFC number be assigned? | Non-blocking |

---

## Summary Verdict

RFC 006 is well-structured and the core design (verbatim body insertion, `CustomElementCollector` mirroring `StyleCollector`, depth-first emission order) is sound. The feature will meaningfully improve the `htmlc` authoring experience for interactive pages while staying true to the engine's "no runtime, server-first" philosophy.

The most important work before implementation begins is resolving the concurrency model for the collector, specifying the deduplication collision policy, and defining the `FlushCustomElements` idempotency contract. The other gaps are addressable in follow-up RFCs or documentation PRs.
