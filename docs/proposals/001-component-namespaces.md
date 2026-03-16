# RFC 001: Component Namespaces

- **Status**: Accepted
- **Date**: 2026-03-14
- **Author**: TBD

---

## 1. Motivation

`htmlc` currently maintains a **flat global namespace** for all components discovered under `ComponentDir`. Every `.vue` file is registered by its base name without extension — `blog/Card.vue` and `admin/Card.vue` both register as `"Card"`, and whichever one the lexical-order walk processes last silently wins.

### The problem in practice

Consider a medium-sized project where two subsystems — a public blog and a staff admin panel — live side by side:

```
components/
  blog/
    Card.vue       ← renders a blog post preview card
  admin/
    Card.vue       ← renders a data row card for the admin dashboard
```

Under the current engine, both files attempt to register as `"Card"`. Lexical order (`admin/Card.vue` comes before `blog/Card.vue`) means `admin/Card.vue` wins. Every `<Card>` tag anywhere in the project — including `blog/PostPage.vue` — now renders the admin card. The only indication something is wrong is a **silent mismatch**: the wrong component renders, no error is raised, and no warning is emitted.

The failure mode is especially treacherous because it is **order-dependent**. Adding a new subdirectory or renaming a folder can silently change which component gets used without any compile-time or startup-time signal.

### Why explicit imports are not the answer

Explicitly importing components (`import Card from './Card.vue'`) is the standard Vue 3 fix for this class of problem. It is not available here: `htmlc` is a **no-import, auto-discovery** engine. There is no JavaScript module graph. Templates are plain HTML files. Adding an import mechanism would require new syntax that breaks the "100% HTML-compatible template" property and would fundamentally change the user experience of the engine.

The goal, therefore, is to give the engine **structural awareness** — the ability to infer from directory layout which component the author almost certainly intended — while keeping templates syntactically valid HTML.

---

## 2. Goals

1. **Proximity-based resolution**: when a component name is ambiguous, the instance closest in the directory tree to the calling template wins.
2. **Full addressability**: any component can be referenced explicitly regardless of caller location.
3. **100% HTML-compatible syntax**: no new tag syntax that would make a `.vue` template fail an HTML validator or break an IDE that treats templates as HTML.
4. **No explicit imports**: resolution remains automatic and convention-based.
5. **Backward compatibility**: existing flat-namespace projects continue to work without modification.

---

## 3. Non-Goals

- **Runtime JavaScript namespacing**: this system is entirely server-side and has no effect on client-side component registries.
- **Module-level scoping**: there is no JS module graph; this is purely a structural / filesystem convention.
- **Changing or deprecating kebab/PascalCase flexibility**: the existing four-strategy name-folding lookup (`exact → capitalise → kebab-to-Pascal → case-insensitive`) is preserved within each directory level of the proximity walk.
- **Access control or visibility restrictions**: any component remains reachable from any template via explicit addressing; this is a resolution preference, not a hard boundary.

---

## 4. Proposed Design

### 4.1 Directory-Scoped Registry

#### Current state

`engine.go` maintains:

```go
entries map[string]*engineEntry   // flat: "Card" → entry
```

`registerPathLocked` extracts the bare filename (`base := filepath.Base(path)`), strips the extension, and uses that as the sole registry key. The full relative path is stored in `engineEntry.path` but is **not indexed** — there is no reverse lookup from directory to components.

#### Proposed extension

Add a second index alongside the existing flat map:

```go
// Engine gains a second registry alongside the existing flat one.
// nsEntries is keyed by (relDir, localName) where:
//   relDir   = path of the component file's directory relative to ComponentDir
//              (empty string "" for the root)
//   localName = bare filename without extension, exactly as used today
nsEntries map[string]map[string]*engineEntry
```

This is a `map[dirPath]map[localName]*engineEntry`. Example population for a project rooted at `components/`:

| File | `relDir` | `localName` |
|------|----------|-------------|
| `components/Card.vue` | `""` | `"Card"` |
| `components/blog/Card.vue` | `"blog"` | `"Card"` |
| `components/admin/Card.vue` | `"admin"` | `"Card"` |
| `components/blog/PostPage.vue` | `"blog"` | `"PostPage"` |

The flat `entries` map is retained unchanged for backward compatibility and for the `Register(name, path)` manual API.

#### Population during discovery

`discover` already walks the full path. The change is to compute `relDir`:

```go
// pseudo-code — not implementation
relPath, _ := filepath.Rel(opts.ComponentDir, path)
relDir     := filepath.Dir(relPath)
if relDir == "." {
    relDir = ""
}
localName := strings.TrimSuffix(filepath.Base(path), ext)
e.nsEntries[relDir][localName] = entry
```

No path information is stored in the `Component` struct itself; the registry handles all path bookkeeping.

### 4.2 Proximity-Based Resolution Algorithm

The renderer currently receives a `Registry` snapshot and a component to render. To support proximity resolution the renderer also needs to know **which component file is currently being rendered**, so it can compute the caller's directory.

#### What the renderer knows today

`Renderer.component` is a `*Component` and `Component.Path` holds the source file path. This path is already available at every level of the recursive render. The renderer therefore already has everything needed to determine the caller's directory — no new field is required.

#### Proposed algorithm

When `resolveComponent(tagName)` is called:

1. Compute `callerDir` from `r.component.Path` relative to `ComponentDir`.
   - If `r.component` is nil or has no path, treat `callerDir` as `""` (root).
2. Try the **proximity walk**: starting at `callerDir`, walk toward the root one directory segment at a time:
   a. For the current directory `d`, attempt all four name-folding strategies (exact, capitalise, kebab-to-Pascal, case-insensitive) against `nsEntries[d]`.
   b. If a match is found, return it immediately.
   c. Move to the parent: `d = filepath.Dir(d)`. If `d` becomes `"."` or `""`, stop.
3. If the proximity walk finds nothing, **fall back to the existing flat-registry scan** (`entries`) for backward compatibility.

This means:

- Proximity is tried **before** name-folding only in the sense that proximity determines *which directory's entries* are searched first. Name-folding happens within each directory level.
- Root-level components are naturally reached on the last iteration of the walk (when `d == ""`).
- The flat-registry fallback ensures that manually registered components (via `Engine.Register`) and any legacy project that relies on the flat namespace continue to work.

#### Tie-breaking within a directory

If both `blog/Card.vue` and `blog/card.vue` exist (mixed case), the name-folding strategies are applied in order and the first match wins:

1. Exact match (case-sensitive)
2. First-letter capitalised
3. Kebab-to-PascalCase
4. Case-insensitive full scan

This preserves the existing behaviour and is deterministic.

#### The `ComponentDir` context in the renderer

`ComponentDir` is currently an `Engine` option but is not passed to the `Renderer`. Two approaches are available:

- **Option A**: pass `ComponentDir` (or a stripped prefix) into `Registry` or into `Renderer` as a new field.
- **Option B**: store the relative path in `Component.Path` from the moment of registration, so that `Component.Path` is already relative to `ComponentDir`.

Option B is cleaner — it avoids threading a new string through the render call chain — and it is what the implementation sketch in §7 assumes.

### 4.3 Explicit Cross-Namespace Addressing

Three syntactic options were considered for explicitly addressing a component outside the caller's proximity path.

#### Option A — Path-encoded tag names using `--` as path separator

```html
<!-- Resolves to admin/dashboard/Chart.vue -->
<admin--dashboard--Chart></admin--dashboard--Chart>
```

Double-hyphen (`--`) is unambiguous as a path separator because single hyphens are already used as word separators in kebab-case names. The algorithm splits the tag name on `--`, treats all but the last segment as directory path components, and the last segment as the local component name.

**Evaluation**:
- ✅ Valid HTML custom-element tag name (contains hyphens, starts with letter)
- ✅ No new syntax; works with the existing HTML parser
- ⚠️ Visually awkward for deep paths (`admin--reports--monthly--Chart`)
- ⚠️ Double-hyphen is a CSS comment delimiter — while that only matters inside `<style>` blocks, it can cause confusion for developers
- ⚠️ Breaks down if a directory name itself contains a hyphen, making the splitting ambiguous (e.g. `my-blog--Card` could mean directory `my-blog`, component `Card` or directory `my`, component `blog--Card`)

**Verdict**: Valid but fragile for projects with hyphenated directory names. Described here as an alternative; not recommended as the primary mechanism.

#### Option B — Extend `<component is="...">` with path support ✅ **Recommended**

```html
<!-- Explicit relative path within ComponentDir -->
<component is="admin/Card"></component>

<!-- Dynamic explicit reference -->
<component :is="'admin/dashboard/Chart'"></component>
```

The existing `<component :is="expr">` syntax already handles dynamic component selection. This option simply extends the resolution of the `is` value: if the resolved string contains a `/`, it is treated as a **path** relative to `ComponentDir` rather than a flat component name.

Resolution for a path-valued `is`:

1. Strip any leading `/` to obtain a canonical relative path.
2. Split on `/` to get `(dirSegments..., localName)`.
3. Look up `nsEntries[filepath.Join(dirSegments...)][localName]` directly (exact match only, no proximity walk, no name-folding).
4. If not found, return an error.

**Evaluation**:
- ✅ Reuses existing syntax; no new HTML constructs
- ✅ Slashes are not valid in HTML tag names, so the disambiguation between "name" and "path" is clean
- ✅ Works equally well for static (`is="admin/Card"`) and dynamic (`:is="expr"`) values
- ✅ The HTML validator already accepts `<component>` as a tag name
- ✅ Root-relative addressing via leading `/` is natural (`is="/Card"` means root `Card`)
- ✅ No ambiguity from directory names with hyphens

**Verdict**: Recommended primary mechanism.

#### Option C — `data-namespace` block attribute on `<template>`

```html
<template data-namespace="admin">
  <Card></Card>  <!-- resolves to admin/Card.vue -->
</template>
```

Declares a namespace context at the file level. All unqualified component names within the template are resolved as if the caller lives in the declared namespace.

**Evaluation**:
- ✅ HTML-compatible (`data-*` attributes are always valid)
- ✅ Simple to implement for the common case of a whole file belonging to one namespace
- ❌ File-level only — cannot mix namespaces within a single template
- ❌ Overrides the natural proximity of the file's actual location, which can be confusing
- ❌ Does not help when a single template legitimately needs components from two different namespaces

**Verdict**: Less flexible than Option B; not recommended as a primary mechanism. May be considered in a follow-up RFC as an ergonomic shortcut.

#### Chosen approach

**Primary**: Option B (path-valued `is` in `<component>`).
**Alternative**: Option A documented for awareness; implementations may choose to support it as an additional resolution strategy.
**Option C**: explicitly not included in this RFC; left as an open question (see §10).

### 4.4 Root-Level Addressing

To explicitly reference a root-level component from deep in the directory tree, use a leading `/` in the `is` value:

```html
<!-- Always resolves to components/Card.vue, regardless of caller location -->
<component is="/Card"></component>
```

The leading `/` strips the directory portion and performs a direct lookup in `nsEntries[""]` (the root namespace). Name-folding (capitalise, kebab-to-Pascal, case-insensitive) **is** applied for root-relative addresses, since the caller is expressing intent clearly and the convenience is worthwhile.

For the non-path `<component :is="expr">` case (no `/`), the resolved name follows the existing flat-registry logic, which already effectively reaches root-level components last in the proximity walk.

### 4.5 Registration Changes

#### `engine.go`

`registerPathLocked(name, path string)` is extended to also populate `nsEntries`:

```go
// (existing flat registration unchanged)
e.entries[name] = entry
if lower := strings.ToLower(name); lower != name {
    e.entries[lower] = entry
}

// New: namespaced registration
relPath, err := filepath.Rel(e.opts.ComponentDir, path)
if err == nil {
    relDir := filepath.ToSlash(filepath.Dir(relPath))
    if relDir == "." {
        relDir = ""
    }
    if e.nsEntries[relDir] == nil {
        e.nsEntries[relDir] = make(map[string]*engineEntry)
    }
    e.nsEntries[relDir][name] = entry
}
```

For `fs.FS`-based discovery, `path` is already relative to the FS root, so `filepath.Rel` against `ComponentDir` works the same way.

`Engine.Register(name, path)` continues to function exactly as today. Manually registered components populate only the flat `entries` map (they have no structural directory to place them in). They remain globally resolvable from anywhere via the flat-registry fallback.

#### `renderer.go`

`Renderer` gains a field to receive the namespaced registry:

```go
type Renderer struct {
    // ... existing fields ...
    nsRegistry  map[string]map[string]*Component  // new; may be nil
    componentDir string                            // new; root prefix for relative-path resolution
}
```

`buildRegistryLocked` in `engine.go` also builds the `nsRegistry` snapshot:

```go
func (e *Engine) buildNSRegistryLocked() map[string]map[string]*Component {
    ns := make(map[string]map[string]*Component, len(e.nsEntries))
    for dir, entries := range e.nsEntries {
        ns[dir] = make(map[string]*Component, len(entries))
        for name, entry := range entries {
            ns[dir][name] = entry.comp
        }
    }
    return ns
}
```

The `renderComponent` method in `engine.go` passes both the flat registry and the NS registry to `NewRenderer`, along with `ComponentDir`.

`Component.Path` stores the path as given to `ParseFile`. For auto-discovered components the caller should pass the **relative** path (relative to `ComponentDir`) so that proximity can be computed without needing to know `ComponentDir` at render time. This is a **one-line change** in `discover`: replace the absolute path with the relative path when calling `registerPathLocked`.

#### API compatibility

- `Engine.Register(name, path)` is unchanged in signature and semantics.
- `NewRenderer(c)` is unchanged.
- `WithComponents(reg Registry)` continues to work; it populates only the flat registry path in `Renderer`. Callers that bypass `Engine` entirely and use `NewRenderer` directly will not get proximity resolution unless they also call the new `WithNSComponents` method.
- The new `WithNSComponents(ns, componentDir)` method is additive.

### 4.6 Directory-Name as Namespace Prefix (Nuxt-style alternative)

Nuxt 3's auto-import system registers `components/blog/Card.vue` under both `Card` (local) and `BlogCard` (prefixed) in the global registry. This gives callers a stable global alias that does not depend on their location.

For `htmlc`, the equivalent would be: during `discover`, in addition to registering `blog/Card.vue` under `nsEntries["blog"]["Card"]`, also register it in the flat `entries` map under `BlogCard` (PascalCase directory name + component name).

**Feasibility**: straightforward to implement alongside the proximity change.

**Collision risk**: if a file named `BlogCard.vue` also exists at the root, it would collide with the auto-generated alias. The proposal recommends that auto-generated Nuxt-style aliases have **lower priority** than explicit file registrations: a directly-registered name always wins.

**Recommendation**: Implement as an opt-in `Engine` option (`PrefixedAliases bool` in `Options`), disabled by default, to preserve backward compatibility. This feature is sufficiently self-contained that it could be addressed in a separate RFC or as a follow-on option once the core proximity mechanism is stable.

---

## 5. Syntax Summary

| Syntax | Meaning |
|--------|---------|
| `<Card></Card>` | Proximity-based: finds the nearest `Card.vue` walking up from the current file's directory |
| `<component is="blog/Card"></component>` | Explicit relative path: exactly `blog/Card.vue` under `ComponentDir` |
| `<component :is="'blog/Card'"></component>` | Same as above, dynamically evaluated |
| `<component is="/Card"></component>` | Root-relative: `Card.vue` at the root of `ComponentDir` |
| `<BlogCard></BlogCard>` *(if `PrefixedAliases` enabled)* | Always resolves to `blog/Card.vue`, regardless of caller location |
| `<admin--Card></admin--Card>` *(Option A, if implemented)* | Alternative path-encoded tag; resolves to `admin/Card.vue` |

---

## 6. Examples

### Example 1 — Flat project (backward compatible, no change)

```
components/
  Button.vue
  Card.vue
  Layout.vue
```

`<Card></Card>` in any template resolves exactly as today: the proximity walk tries `nsEntries[""]`, finds `Card`, and returns it. The flat-registry fallback is never needed. All existing behaviour is preserved.

### Example 2 — Namespaced project with proximity preference

```
components/
  Card.vue                ← root Card (generic)
  blog/
    Card.vue              ← blog-specific card
    PostPage.vue
    AuthorPage.vue
  admin/
    Card.vue              ← admin data-row card
    Dashboard.vue
    reports/
      Chart.vue
```

Resolution table:

| Template | Tag | Resolves to |
|----------|-----|-------------|
| `blog/PostPage.vue` | `<Card>` | `blog/Card.vue` — found in `blog/` on first walk step |
| `admin/Dashboard.vue` | `<Card>` | `admin/Card.vue` — found in `admin/` on first walk step |
| root template | `<Card>` | `Card.vue` — found in `""` (root) |
| `admin/Dashboard.vue` | `<component is="blog/Card">` | `blog/Card.vue` — explicit path |
| `admin/reports/Chart.vue` | `<Card>` | `admin/Card.vue` — not in `admin/reports/`, walks up to `admin/`, found |
| `admin/Dashboard.vue` | `<component is="/Card">` | `Card.vue` — root-relative explicit |

### Example 3 — Missing local component, walk-up fallback

```
components/
  Button.vue              ← only at root
  blog/
    PostPage.vue
    deep/
      ThreadPage.vue
```

Resolution:

| Template | Tag | Walk steps | Resolves to |
|----------|-----|-----------|-------------|
| `blog/PostPage.vue` | `<Button>` | `blog/` → miss; `""` → hit | `Button.vue` |
| `blog/deep/ThreadPage.vue` | `<Button>` | `blog/deep/` → miss; `blog/` → miss; `""` → hit | `Button.vue` |

### Example 4 — Dynamic cross-namespace reference

```
components/
  admin/
    Dashboard.vue
  widgets/
    Chart.vue
    Table.vue
```

Inside `admin/Dashboard.vue`:

```html
<template>
  <div class="dashboard">
    <!-- explicit cross-namespace reference -->
    <component is="widgets/Chart"></component>
    <component is="widgets/Table"></component>
  </div>
</template>
```

Both `widgets/Chart.vue` and `widgets/Table.vue` are resolved directly via path lookup, bypassing the proximity walk entirely.

---

## 7. Implementation Sketch

This section outlines Go-level changes needed. It is intentionally high-level — full implementation is out of scope for this RFC.

### `engine.go`

1. **`Options`**: no new fields required for the core feature. Add `PrefixedAliases bool` for the Nuxt-style opt-in.

2. **`Engine`**: add `nsEntries map[string]map[string]*engineEntry` alongside `entries`.

3. **`New`**: initialise `nsEntries` as an empty map.

4. **`registerPathLocked`**: after writing to `entries`, compute `relDir` from `path` relative to `opts.ComponentDir` and write to `nsEntries[relDir][name]`.
   - When `opts.FS` is set, paths are already FS-relative; use `path.Dir` (the `path` package, not `filepath`) since FS paths use forward slashes.
   - Store `path` as a FS-relative (or `ComponentDir`-relative) path in `engineEntry.path` for use by the hot-reload logic.

5. **`buildNSRegistryLocked`**: new method returning `map[string]map[string]*Component`.

6. **`renderComponent`**: pass the NS registry and `ComponentDir` (or empty string if stored as relative) to the renderer via new builder methods.

### `renderer.go`

1. **`Renderer`**: add `nsRegistry map[string]map[string]*Component` and `componentDir string`.

2. **`WithNSComponents(ns map[string]map[string]*Component, componentDir string) *Renderer`**: new builder method.

3. **`resolveComponent`**: updated algorithm:
   ```
   if nsRegistry is not nil:
       callerDir = dir(r.component.Path relative to componentDir)
       d = callerDir
       loop:
           for each name-variant of tagName:
               if nsRegistry[d][variant] exists: return it
           if d == "": break
           d = parent(d)
   // fall through to existing flat registry logic
   ```

4. **`renderElement`** (the section handling `<component is="...">`)**: after evaluating the `is` expression to a string, check if it contains `/`:
   - If yes: treat as a path; look up in `nsRegistry` directly (no proximity walk, no name-folding except for leading-`/` root-relative).
   - If no: proceed as today (flat registry lookup, then proximity if available).

5. **`renderComponentElement`**: no changes needed; the resolved `*Component` is passed in from `resolveComponent`.

### `component.go`

No changes needed. `Component.Path` already stores whatever path string is passed to `ParseFile`.

### Reload behaviour

`maybeReload` iterates `e.entries` by key. After reload, it should also update `nsEntries` at the same `(relDir, localName)` coordinate. The simplest approach: after calling `registerPathLocked` during reload, re-derive `relDir` and update `nsEntries` in place.

---

## 8. Backward Compatibility

### Flat-namespace projects

The proximity walk always ends by trying the root directory (`""`), then falls back to the flat `entries` map. A project with all components at the root of `ComponentDir` will match on the first (and only) step of the proximity walk. Behaviour is identical to today.

### `Engine.Register` manual API

Manually registered components continue to populate `entries` only. They are accessible everywhere via the flat-registry fallback at the end of `resolveComponent`. No scoping or proximity restriction is applied to manually registered components.

### Kebab-case and case-insensitive lookup

All four name-folding strategies are applied at each directory level of the proximity walk. The only change is that they are tried per-directory rather than against the entire global registry.

### `ValidateAll`

`ValidateAll` calls `collectComponentRefs` and `resolveInRegistry`. The proximity-aware validation would require knowing the source component's path; without it, validation uses the flat registry (existing behaviour). This means `ValidateAll` may report false positives (unknown component) for references that would resolve correctly at runtime via proximity. A follow-up change to `ValidateAll` should pass the caller's path to a new `resolveInNSRegistry` helper.

### Low-level `NewRenderer` / `Registry` API

Callers using `NewRenderer` directly (without `Engine`) will get the existing flat-registry behaviour. This is acceptable: the flat API remains fully functional. Callers that want proximity resolution can opt in by calling `WithNSComponents`.

---

## 9. Alternatives Considered

### Explicit imports

Rejected. Adding import syntax requires non-HTML constructs in `.vue` templates (either inside `<script>` — which `htmlc` does not execute — or as a new top-level section). This undermines the "100% HTML-compatible template" principle and significantly raises the authoring complexity for a server-side-only tool.

### Mandatory unique component names

Rejected. Requiring every component across the whole tree to have a globally unique name imposes an unreasonable naming burden on large projects and prevents natural, localised naming conventions (`Card`, `Button`, `Layout` are legitimately useful at multiple levels).

### Separate registry per subdirectory

A variant of the proposed design where each subdirectory gets a completely isolated registry and components do not automatically inherit from parent directories. This would require explicit opt-in to share components and is more restrictive than necessary. The proposed proximity walk already provides isolation while allowing natural fallthrough.

### Option A: `--` separator in tag names

Described in §4.3. Valid HTML, but ambiguous for hyphenated directory names and visually noisy. Retained as a documented alternative that implementations may support alongside Option B.

### Option C: `data-namespace` block

Described in §4.3. Elegant for homogeneous files, but file-level only and unable to handle mixed-namespace templates. Not included in this RFC's primary design.

### Last-write-wins flat namespace (status quo)

The current behaviour. Fragile, undocumented, and invisible to the template author. Eliminated by the proposed design for projects that have same-named components in different directories.

---

## 10. Open Questions

1. **Opt-in vs opt-out**: Should proximity resolution be opt-in (new `ProximityResolution bool` in `Options`, default `false`) or enabled automatically for all new projects (default `true`, with opt-out)? Opt-in is safer for existing deployments; opt-out provides a better default experience for new projects. Recommendation: default `true` since the proximity walk degrades gracefully to flat-registry behaviour.

2. **Hot-reload interaction**: When `Reload: true` and a file is modified, `maybeReload` re-registers it. If the file moves (rename), both the old and new entries can coexist until the next full reload. The RFC assumes that hot-reload triggers a full re-walk of `nsEntries` rather than an incremental update. This may be an acceptable trade-off for development use but should be confirmed.

3. **`data-namespace` as file-level override**: Should Option C be supported as a convenience even if Option B is the primary explicit-addressing mechanism? It could simplify templates that draw exclusively from one non-local namespace. This is deferred to a follow-up RFC.

4. **Slot scope interaction**: When a slot is defined in template A (caller) and rendered inside template B (callee), component references inside the slot content are resolved against template A's scope. Does this mean proximity should also use A's directory, not B's? The proposed design uses the authoring component's path, which is stored in `SlotDefinition.Component`. This should be verified during implementation.

5. **`ValidateAll` accuracy**: As noted in §8, `ValidateAll` will produce false positives for proximity-resolved references after this change if it is not updated to be proximity-aware. Should `ValidateAll` be updated in the same change set, or is a follow-up acceptable?

6. **FS-relative paths and `filepath` vs `path`**: When `opts.FS` is set, path separators are forward slashes regardless of OS. The implementation must use `path` (not `filepath`) for FS-based path operations, and `filepath` for OS-based ones. The RFC's implementation sketch uses `filepath.ToSlash` as a compatibility shim; a cleaner abstraction may be warranted.

7. **Performance of the proximity walk**: For deeply nested templates, the walk performs up to `depth+1` map lookups per component reference. Given that maps are O(1) and component trees rarely exceed 6–8 levels deep, this is not expected to be a bottleneck. Profiling should confirm before optimising.
