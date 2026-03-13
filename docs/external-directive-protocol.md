# External Directive Protocol

This document is the reference specification for the **htmlc external directive
protocol** — the interface between `htmlc build` and external directive
executables.

---

## Table of contents

1. [Overview](#overview)
2. [Discovery](#discovery)
3. [Lifecycle](#lifecycle)
4. [Transport](#transport)
5. [Hooks](#hooks)
   - [created](#created-hook)
   - [mounted](#mounted-hook)
6. [Request fields](#request-fields)
7. [Response fields](#response-fields)
8. [Error handling](#error-handling)
9. [Example session](#example-session)

---

## Overview

An external directive is an executable file placed in the `htmlc` component
directory (`-dir`) whose name matches the pattern `v-<name>`.  When `htmlc
build` discovers such a file it registers a directive named `v-<name>` that can
be used in templates exactly like a built-in directive.

External directives allow third-party code — in any language — to participate
in the rendering pipeline without modifying `htmlc` itself.

---

## Discovery

`htmlc build` scans the component directory (`-dir`) for executable files whose
names begin with `v-`.  Each such file is registered as an external directive
using the file's base name as the directive name.

Examples:

| File | Registered directive |
|------|---------------------|
| `templates/v-syntax-highlight` | `v-syntax-highlight` |
| `templates/v-truncate` | `v-truncate` |

Non-executable files and files whose names do not begin with `v-` are ignored
by the directive discovery mechanism.

---

## Lifecycle

1. **Spawn** — at the start of a build `htmlc` spawns one process per
   discovered directive.
2. **Communicate** — for each element that carries the directive, `htmlc` sends
   a request and reads a response over stdin/stdout.
3. **Terminate** — when the build finishes `htmlc` closes the directive's
   stdin.  The directive should drain its input and exit cleanly.  A non-zero
   exit code is treated as a warning, not a build failure.

A single directive process handles all invocations of its directive across all
pages in a build.  Requests are serialised: `htmlc` sends one request at a time
and waits for the response before sending the next.

---

## Transport

Communication uses **newline-delimited JSON** (NDJSON):

- Each request is one JSON object followed by a newline (`\n`).
- Each response is one JSON object followed by a newline (`\n`).
- Requests and responses are strictly alternating: one response per request.
- The directive's **stderr** is forwarded verbatim to `htmlc`'s stderr.
- The directive must not write anything to stdout that is not a valid JSON
  response — doing so will cause a parse error on the `htmlc` side.

---

## Hooks

### `created` hook

The `created` hook is called when the renderer encounters an element that
carries the directive, before the element's children are rendered.

The directive may:

- Replace the element's HTML tag (via `tag`).
- Replace or augment the element's attributes (via `attrs`).
- Replace the element's inner HTML (via `inner_html`).

If `inner_html` is returned the element's template children are discarded and
the supplied HTML is used verbatim.

### `mounted` hook

The `mounted` hook is called after the element and all its children have been
rendered and written to the output buffer.

The directive may:

- Inject arbitrary HTML immediately after the closing tag of the element (via
  `html`).

Return an empty `html` field (or omit it) to inject nothing.

---

## Request fields

All requests include the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `hook` | string | `"created"` or `"mounted"` |
| `id` | string | Unique request identifier; must be echoed in the response |
| `tag` | string | HTML tag name of the directive's host element (e.g. `"pre"`) |
| `attrs` | object | Current attributes of the element as `{"name": "value"}` |
| `text` | string | Concatenated text-node content of the element's children |
| `binding` | object | Directive binding (see below) |

**Binding sub-object:**

| Field | Type | Description |
|-------|------|-------------|
| `binding.value` | any | Evaluated value of the directive expression |
| `binding.raw_expr` | string | Raw source text of the directive expression |
| `binding.arg` | string | Directive argument (the part after `:`, e.g. `href` in `v-bind:href`) |
| `binding.modifiers` | object | Modifier flags as `{"modifier": true}` |

---

## Response fields

All responses must include:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | string | yes | Must match the request `id` |

### `created` hook response

| Field | Type | Description |
|-------|------|-------------|
| `tag` | string | New tag name for the element; omit to keep the current tag |
| `attrs` | object | Full replacement attribute map; omit to keep current attributes |
| `inner_html` | string | HTML to use as the element's inner content; omit to render children normally |
| `error` | string | If non-empty, the build records an error for this element |

### `mounted` hook response

| Field | Type | Description |
|-------|------|-------------|
| `html` | string | HTML to inject immediately after the element's closing tag |
| `error` | string | If non-empty, the build records an error for this element |

---

## Error handling

| Situation | Behaviour |
|-----------|-----------|
| Request JSON cannot be marshalled | Build logs a warning; directive invocation is skipped |
| Directive stdout closes unexpectedly | Build logs a warning; directive invocation returns a no-op |
| Response JSON is malformed | Build logs a warning; directive invocation returns a no-op |
| Response `id` does not match request `id` | Build logs an error; directive invocation returns a no-op |
| Response contains a non-empty `error` field | Build records the error for the affected element |
| Directive exits with non-zero status | Build logs a warning after the build completes |

In all warning cases the build continues.  Failed elements produce a warning or
error in the build summary but do not abort rendering of other elements or
pages.

---

## Example session

The following illustrates a two-invocation session for a `v-syntax-highlight`
directive applied to two `<pre>` elements.

**stdin (requests sent by htmlc):**

```
{"hook":"created","id":"1","tag":"pre","attrs":{},"text":"func main() {}","binding":{"value":"go","raw_expr":"'go'","arg":"","modifiers":{}}}
{"hook":"mounted","id":"2","tag":"pre","attrs":{"class":"language-go"},"text":"func main() {}","binding":{"value":"go","raw_expr":"'go'","arg":"","modifiers":{}}}
```

**stdout (responses written by the directive):**

```
{"id":"1","attrs":{"class":"language-go"},"inner_html":"<span class=\"kd\">func</span> <span class=\"nx\">main</span><span class=\"p\">()</span> <span class=\"p\">{}</span>"}
{"id":"2","html":""}
```

`htmlc` closes stdin after the build is complete; the directive should exit
when its stdin is exhausted.
