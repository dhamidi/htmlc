# RFC 003: expvar-Driven Engine Options and Performance Counters

- **Status**: Draft
- **Date**: 2026-03-15
- **Author**: TBD

---

## 1. Motivation

Once an `htmlc` `Engine` is constructed there is no standard way to inspect
its active configuration, modify tunable options without restarting the process,
or observe runtime performance data through a uniform interface.

### The runtime visibility gap

Today an operator who wants to know whether hot-reload is active on a running
server must read the source code or add custom logging.  There is no HTTP
endpoint or structured output that answers questions such as:

- Is this engine running in debug mode?
- How many components are registered?
- How many renders have completed and how long did they take?
- Has the hot-reload threshold been triggered recently?

Go's standard library ships `net/http/pprof` (CPU and memory profiling) and
`expvar` (exported variables) as complementary introspection facilities.
`expvar` is designed exactly for this use case: it provides a global registry
of named variables that are automatically serialised to JSON and exposed at
`/debug/vars`.  `htmlc` currently makes no use of it.

### The restart tax for option changes

Certain engine options — particularly `Reload` and `Debug` — are boolean
toggles that a developer routinely switches between environments.  Today the
only mechanism to change them is to reconstruct the `Engine`.  In a
long-running server this means a full process restart.  If `Reload` or `Debug`
were backed by a live `expvar.Int`, a developer could toggle them in place
(e.g. via a `curl` against `/debug/vars` paired with a small write endpoint,
or via a purpose-built admin handler) without interrupting in-flight requests
or discarding cached components.

### Why `opts` alone is not enough

`opts Options` is captured by value at construction time (`engine.go:151`).
After `New` returns it is effectively frozen — there is no way for an external
observer to read or modify it.  Making a copy of the struct available through a
getter would solve read-only inspection but would not enable live mutation.
`expvar` solves both problems in one package with well-understood semantics and
a standard HTTP integration point.

---

## 2. Goals

1. **Option inspection**: every field of `Options` (`ComponentDir`, `Reload`,
   `Debug`, `FS`, `Directives`) is exposed as a readable `expvar.Var` on the
   `Engine`.
2. **Runtime settability for scalar options**: `Reload` and `Debug` can be
   toggled at runtime by writing to their `expvar.Int` variables; the engine
   picks up the new value on the next render or reload check without restart.
3. **All options writable at runtime**: `ComponentDir` and `FS` are settable
   via `SetComponentDir` and `SetFS` methods that re-run discovery atomically
   under the engine's write-lock.
4. **Non-colliding multi-engine registration**: a process running multiple
   `Engine` instances can publish each under a caller-supplied name prefix; two
   engines with different prefixes never share or collide in the global registry.
5. **Performance counters**: the engine publishes atomic counters for total
   renders, render errors, hot-reload re-scans, cumulative render latency, and
   registered component count.
6. **Expvar as primary source of truth**: `varReload` and `varDebug` are the
   live state; `Options` seeds them at construction time but does not remain
   authoritative.
7. **Strictly opt-in publication**: callers that never call `PublishExpvars`
   are unaffected in behaviour and incur no global registry side effects.

---

## 3. Non-Goals

1. **Settable `Directives`**: directive registration already has a thread-safe
   post-construction API (`RegisterDirective`); bridging it to expvar is out of
   scope here.
2. **Custom HTTP admin endpoints**: this RFC adds standard `expvar` publication
   only.  Building a read/write admin UI or structured mutation API on top of
   expvar is a separate concern.
3. **Per-request metrics**: counters proposed here are process-lifetime
   aggregates.  Request-level histograms, percentile tracking, or distributed
   tracing integration are out of scope.
4. **Prometheus or OpenTelemetry bridges**: expvar is the sole target.  Bridging
   to other telemetry systems can be layered on top of expvar after this RFC.
5. **Engine pooling or hot-swapping**: swapping an `Engine` out of an HTTP
   handler while requests are in flight is outside this proposal's scope.

---

## 4. Proposed Design

### 4.1 Engine expvar map and registration API

#### Current state

`Engine` (`engine.go:80–89`) stores its configuration exclusively in
`opts Options`, which is an unexported value field set once during `New` and
never updated afterwards.  The struct has no expvar-related fields.

```go
// engine.go — current Engine struct (abridged)
type Engine struct {
    opts               Options
    mu                 sync.RWMutex
    entries            map[string]*engineEntry
    nsEntries          map[string]map[string]*engineEntry
    missingPropHandler MissingPropFunc
    directives         DirectiveRegistry
    funcs              map[string]any
    dataMiddleware     []func(*http.Request, map[string]any) map[string]any
}
```

#### Proposed extension — new Engine fields

```go
// pseudo-code — not implementation

type Engine struct {
    // ... existing fields unchanged ...

    // expvar-backed option vars — seeded from Options during New,
    // then authoritative for runtime reads.
    varReload       *expvar.Int    // 0 = false, 1 = true
    varDebug        *expvar.Int    // 0 = false, 1 = true

    // settable option vars — updated by SetComponentDir / SetFS
    varComponentDir *expvar.String
    varFS           *expvar.String // reflect.TypeOf(opts.FS).String() or "<nil>"

    // read-only info var (published under "info" sub-map)
    varDirectives   *expvar.Func   // returns []string of registered directive names

    // performance counters
    counterRenders      *expvar.Int  // total calls to renderComponent
    counterRenderErrors *expvar.Int  // renderComponent calls that returned non-nil error
    counterReloads      *expvar.Int  // maybeReload full re-walk triggers
    counterRenderNanos  *expvar.Int  // cumulative wall-clock render time in nanoseconds
    counterComponents   *expvar.Func // len(e.entries) de-duplicated, computed live

    // global registry integration
    expvarMap    *expvar.Map // the engine's own Map; nil until PublishExpvars is called
    expvarPrefix string      // e.g. "htmlc" or "htmlc.api"; set by PublishExpvars
}
```

All `*expvar.Int` and `*expvar.String` types from the standard library are
**safe for concurrent use** (they use `sync/atomic` internally).  No additional
locking is required when reading or writing through these vars.

#### New field initialisation in `New`

```go
// pseudo-code — not implementation

func New(opts Options) (*Engine, error) {
    e := &Engine{
        opts:      opts,
        entries:   make(map[string]*engineEntry),
        nsEntries: make(map[string]map[string]*engineEntry),
        directives: opts.Directives,
    }

    // Seed the live expvar vars from Options.
    e.varReload = new(expvar.Int)
    if opts.Reload {
        e.varReload.Set(1)
    }
    e.varDebug = new(expvar.Int)
    if opts.Debug {
        e.varDebug.Set(1)
    }

    e.varComponentDir = new(expvar.String)
    e.varComponentDir.Set(opts.ComponentDir)

    e.varFS = new(expvar.String)
    if opts.FS != nil {
        e.varFS.Set(reflect.TypeOf(opts.FS).String())
    } else {
        e.varFS.Set("<nil>")
    }

    e.varDirectives = &expvar.Func{F: func() any {
        e.mu.RLock()
        defer e.mu.RUnlock()
        names := make([]string, 0, len(e.directives))
        for name := range e.directives {
            names = append(names, name)
        }
        sort.Strings(names)
        return names
    }}

    // Performance counters — all zero at start.
    e.counterRenders      = new(expvar.Int)
    e.counterRenderErrors = new(expvar.Int)
    e.counterReloads      = new(expvar.Int)
    e.counterRenderNanos  = new(expvar.Int)
    e.counterComponents   = &expvar.Func{F: func() any {
        return int64(e.componentCountDedup())
    }}

    if opts.ComponentDir != "" {
        if err := e.discover(opts.ComponentDir); err != nil {
            return nil, err
        }
    }
    return e, nil
}
```

The vars are created as **unregistered** `expvar` values (allocated with `new`
or a struct literal, never through `expvar.NewInt` or `expvar.NewMap`).  They
hold live state and can be inspected or mutated directly on the `Engine`
without any global side effect until `PublishExpvars` is called.

#### `PublishExpvars` — the registration builder method

```go
// pseudo-code — not implementation

// PublishExpvars registers this engine's configuration and performance
// counters in the global expvar registry under prefix. Callers may pass
// any non-empty string as the prefix; "htmlc" is the conventional default.
//
// If an entry named prefix already exists in the global registry this method
// panics (consistent with expvar.Publish, which panics on duplicate names).
//
// PublishExpvars returns the Engine so calls can be chained.
func (e *Engine) PublishExpvars(prefix string) *Engine {
    m := expvar.NewMap(prefix)   // panics if prefix already registered

    // writable options
    m.Set("reload",       e.varReload)
    m.Set("debug",        e.varDebug)
    m.Set("componentDir", e.varComponentDir)
    m.Set("fs",           e.varFS)

    // performance counters / gauges
    m.Set("renders",      e.counterRenders)
    m.Set("renderErrors", e.counterRenderErrors)
    m.Set("reloads",      e.counterReloads)
    m.Set("renderNanos",  e.counterRenderNanos)
    m.Set("components",   e.counterComponents)

    // read-only info sub-map
    info := new(expvar.Map)
    info.Set("directives", e.varDirectives)
    m.Set("info", info)

    e.expvarMap    = m
    e.expvarPrefix = prefix
    return e
}
```

`expvar.NewMap(name)` creates a new `expvar.Map`, registers it globally under
`name`, and returns a pointer to it.  The map's `Set(key, Var)` method stores
a sub-var under `key` with no additional global registration.  This results in
a single global entry (`prefix`) whose JSON serialisation is a nested object:

```json
{
  "htmlc": {
    "reload": 0,
    "debug": 0,
    "componentDir": "templates/",
    "fs": "<nil>",
    "renders": 1042,
    "renderErrors": 3,
    "reloads": 0,
    "renderNanos": 48392011,
    "components": 27,
    "info": {
      "directives": ["highlight", "switch"]
    }
  }
}
```

#### Design evaluation — `PublishExpvars` vs `Options.ExpvarPrefix`

Two approaches to triggering registration were considered:

**Option A — `Options.ExpvarPrefix string`**

Set `ExpvarPrefix: "htmlc"` at construction time; `New` calls `expvar.NewMap`
internally.

- ✅ No separate call required; everything configured in one place.
- ⚠️ Forces global registration at construction time; testing code that
  constructs engines in parallel would need unique prefix per test.
- ❌ Makes expvar side effects unavoidable for callers who set `ExpvarPrefix`
  in a shared `Options` struct.
- ❌ Cannot defer registration to after construction (e.g. for engines that are
  conditionally published based on runtime flags).

**Option B — `e.PublishExpvars(prefix string) *Engine` (builder method)** ✅ **Recommended**

- ✅ Strictly opt-in; callers that never call `PublishExpvars` have zero global
  side effects.
- ✅ Chainable with existing builder methods (`WithMissingPropHandler`,
  `RegisterFunc`, etc.).
- ✅ Can be called conditionally (e.g. only in non-test binaries, or only when
  a `--metrics` flag is set).
- ✅ Consistent with the existing post-construction builder pattern.
- ⚠️ One extra method call required; easy to document and discover.

**Verdict**: Option B.  `PublishExpvars(prefix string) *Engine` is the sole
registration mechanism.  `Options` gains no new fields.

---

### 4.2 Option variables (readable and settable)

#### `Reload` and `Debug` — settable `*expvar.Int`

`Reload bool` and `Debug bool` are represented as `*expvar.Int` with the
convention that `0` means `false` and any non-zero value means `true`.  Using
`expvar.Int` (rather than a hypothetical `expvar.Bool`) avoids introducing a
custom type and keeps the JSON representation as a JSON number, which is
directly machine-parseable.

The live value is read from `varReload` and `varDebug`, not from `opts.Reload`
and `opts.Debug`.  `opts` is the **seed** only; after `New` returns, the
engine no longer consults `opts.Reload` or `opts.Debug`.

An external caller can toggle debug mode without restarting the process:

```go
// pseudo-code — not implementation

// Setter method for programmatic use (recommended over direct field access).
func (e *Engine) SetReload(enabled bool) {
    if enabled {
        e.varReload.Set(1)
    } else {
        e.varReload.Set(0)
    }
}

func (e *Engine) SetDebug(enabled bool) {
    if enabled {
        e.varDebug.Set(1)
    } else {
        e.varDebug.Set(0)
    }
}
```

These setter methods are thin convenience wrappers.  They are not strictly
required — a caller with access to the `expvar.Map` can also call
`m.Get("reload").(*expvar.Int).Set(1)` — but the methods provide a
typed, discoverable API that does not require a type assertion.

#### Consistency safety under concurrent renders

`expvar.Int` uses `sync/atomic` internally.  A write to `varReload` or
`varDebug` is immediately visible to any goroutine that subsequently reads from
it.  Because `maybeReload` (described in §4.5) reads `varReload` atomically and
only changes engine state while holding `e.mu` for writing, there is no window
in which a partially updated state is observable.

The edge case of toggling `Reload` from `0` to `1` while a render is already
in flight through `renderComponent` is safe: the in-flight render has already
passed the `maybeReload` gate and holds `e.mu` for reading.  The toggle will
take effect on the next render invocation.

The `Debug` flag is read at the start of each `renderComponent` call.  If a
caller toggles `Debug` while a render is in flight, the in-flight render either
fully uses or fully omits debug wrapping — it never switches mid-render — because
the value is read once at the top of `renderComponent` and the `debugWriter`
is either created or not.

**Verdict**: `sync/atomic` semantics through `expvar.Int` are sufficient for
both `varReload` and `varDebug` without additional locking.

---

### 4.3 Settable option variables — `ComponentDir` and `FS`

`ComponentDir` and `FS` are exposed as writable `*expvar.String` vars and are
updated by `SetComponentDir` and `SetFS` respectively.  Unlike `varReload` and
`varDebug`, changing these options requires re-running the discovery walk, so
the setter methods (described in §4.6) perform additional work under
`e.mu` write-lock.

#### `ComponentDir` — `*expvar.String`

`varComponentDir` is seeded during `New` from `opts.ComponentDir`.  Calling
`SetComponentDir(dir)` updates both `e.opts.ComponentDir` and `varComponentDir`
atomically after a successful re-discover.

#### `FS` — `*expvar.String` (type name only)

`fs.FS` is an interface.  Exposing the concrete type via
`reflect.TypeOf(opts.FS).String()` (or `"<nil>"` when `opts.FS` is `nil`)
gives operators enough information to understand which filesystem backend is in
use (`"embed.FS"`, `"os.dirFS"`, `"afero.MemMapFs"`, etc.) without exposing
internal state of the FS implementation.

`varFS` is seeded during `New` and updated by `SetFS` after a successful
re-discover.

#### Read-only info sub-map — `varDirectives`

`DirectiveRegistry` is `map[string]Directive`.  Since `Directive` is an
interface with no meaningful string representation, only the **names** of
registered directives are exposed.  An `expvar.Func` wrapping a closure over
`e.directives` computes the sorted list of directive names on each JSON
serialisation call.  This is read-only: the expvar interface provides no
mechanism to write a slice back into the map.

`varDirectives` is published under the nested `"info"` sub-map (see §4.1
`PublishExpvars`), not at the top level:

```go
// pseudo-code — not implementation

e.varDirectives = &expvar.Func{F: func() any {
    e.mu.RLock()
    defer e.mu.RUnlock()
    names := make([]string, 0, len(e.directives))
    for name := range e.directives {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}}
```

The closure holds a reference to `e` so it always reflects post-construction
`RegisterDirective` calls, not just the initial `opts.Directives` map.

---

### 4.4 Performance counter variables

The following counters are proposed.  All are `*expvar.Int` (or `*expvar.Func`
for computed values) and safe for concurrent access.

| Field | Type | Unit | Incremented by |
|-------|------|------|----------------|
| `counterRenders` | `*expvar.Int` | count | `renderComponent` entry |
| `counterRenderErrors` | `*expvar.Int` | count | `renderComponent` non-nil error return |
| `counterReloads` | `*expvar.Int` | count | `maybeReload` full re-walk trigger |
| `counterRenderNanos` | `*expvar.Int` | nanoseconds | `renderComponent` completion |
| `counterComponents` | `*expvar.Func` | count | live: `len(dedup(e.entries))` |

#### Render counter and error counter

```go
// pseudo-code — not implementation

func (e *Engine) renderComponent(ctx context.Context, w io.Writer, name string, data map[string]any) (*StyleCollector, error) {
    e.counterRenders.Add(1)

    if err := e.maybeReload(); err != nil {
        e.counterRenderErrors.Add(1)
        return nil, err
    }
    // ... existing resolution and render logic ...
    if err := renderer.Render(w, scope); err != nil {
        e.counterRenderErrors.Add(1)
        return nil, err
    }
    return sc, nil
}
```

#### Render latency counter

```go
// pseudo-code — not implementation

func (e *Engine) renderComponent(ctx context.Context, w io.Writer, name string, data map[string]any) (*StyleCollector, error) {
    e.counterRenders.Add(1)
    start := time.Now()
    defer func() {
        e.counterRenderNanos.Add(time.Since(start).Nanoseconds())
    }()
    // ... rest of renderComponent ...
}
```

Cumulative nanoseconds as an `expvar.Int` gives operators two useful derived
metrics: average render latency (`counterRenderNanos / counterRenders`) and
render throughput.  Storing raw nanoseconds keeps the counter dimensionless and
avoids floating-point representation in expvar's JSON output.

#### Reload counter

```go
// pseudo-code — not implementation

func (e *Engine) maybeReload() error {
    if e.varReload.Value() == 0 {   // reads from varReload, not opts.Reload
        return nil
    }
    // ... stat checks ...
    if !anyChanged {
        return nil
    }
    e.counterReloads.Add(1)  // only triggered on actual re-walk
    e.entries  = make(map[string]*engineEntry)
    e.nsEntries = make(map[string]map[string]*engineEntry)
    if e.opts.ComponentDir != "" {
        return e.discover(e.opts.ComponentDir)
    }
    return nil
}
```

`counterReloads` is incremented only when a full re-walk is triggered (i.e.
`anyChanged == true`), not on every `maybeReload` call.  This gives a precise
count of how often the filesystem changed and caused a re-parse cycle.

#### Component count — `*expvar.Func`

The component count is computed live rather than tracked incrementally to avoid
the risk of under- or over-counting during concurrent `Register` calls and
hot-reload cycles.  The closure de-duplicates entries the same way
`Engine.Components()` does:

```go
// pseudo-code — not implementation

e.counterComponents = &expvar.Func{F: func() any {
    return int64(e.componentCountDedup())
}}

// componentCountDedup counts unique engineEntry pointers in e.entries.
// It must hold e.mu.RLock for safety.
func (e *Engine) componentCountDedup() int {
    e.mu.RLock()
    defer e.mu.RUnlock()
    seen := make(map[*engineEntry]bool, len(e.entries))
    for _, entry := range e.entries {
        seen[entry] = true
    }
    return len(seen)
}
```

Because `expvar.Func` calls `F()` on every JSON serialisation of `/debug/vars`,
the implementation keeps the lock scope minimal.  A brief read-lock for a map
iteration is acceptable at operator-query frequency.

---

### 4.5 Integration with `maybeReload` and `renderComponent`

This section specifies exactly how the engine reads from its expvar vars instead
of from `opts`.

#### `maybeReload` — reading `varReload`

Current code (engine.go:271–272):

```go
func (e *Engine) maybeReload() error {
    if !e.opts.Reload {
        return nil
    }
```

Proposed replacement:

```go
// pseudo-code — not implementation

func (e *Engine) maybeReload() error {
    if e.varReload.Value() == 0 {
        return nil
    }
    // ... remainder unchanged except counterReloads increment ...
}
```

`e.varReload.Value()` is an atomic read (`sync/atomic.LoadInt64`) that is safe
without `e.mu`.  The guard check returns `nil` immediately when reload is
disabled, so the lock is never acquired in the common production path (where
`Reload` is false).

#### `renderComponent` — reading `varDebug`

Current code (engine.go:600–604):

```go
    if e.opts.Debug {
        dw := newDebugWriter(w)
        renderer = renderer.withDebug(dw)
        w = dw
    }
```

Proposed replacement:

```go
// pseudo-code — not implementation

    if e.varDebug.Value() != 0 {
        dw := newDebugWriter(w)
        renderer = renderer.withDebug(dw)
        w = dw
    }
```

The rest of `renderComponent` is unchanged.

#### Summary of `opts` fields and runtime settability

| `Options` field | Engine reads from | Settable at runtime |
|----------------|------------------|---------------------|
| `ComponentDir` | `varComponentDir` / `opts.ComponentDir` | Yes — via `SetComponentDir` |
| `Reload` | `varReload.Value()` | Yes — via `SetReload` |
| `Debug` | `varDebug.Value()` | Yes — via `SetDebug` |
| `FS` | `varFS` / `opts.FS` | Yes — via `SetFS` |
| `Directives` | `e.directives` (managed by `RegisterDirective`) | Via `RegisterDirective` only |

`SetComponentDir` and `SetFS` update both the `opts.*` field and the
corresponding `var*` string under the write-lock.  `maybeReload`'s filesystem
stat calls continue to use `e.opts.ComponentDir` and `e.opts.FS` directly, but
because the setters update `e.opts.*` under the same write-lock they use for
entry replacement, there is no race.

---

### 4.6 Runtime FS and ComponentDir replacement

This section documents the two new setter methods that allow the engine's
component source to be replaced at runtime without restarting the process.

#### `SetComponentDir(dir string) error`

Replaces the component directory with `dir` and re-runs discovery.

```go
// pseudo-code — not implementation

// SetComponentDir replaces the engine's component directory and re-runs
// discovery under the write-lock. Returns an error if discovery fails;
// on error the engine state is left unchanged.
func (e *Engine) SetComponentDir(dir string) error {
    e.mu.Lock()
    defer e.mu.Unlock()
    entries := make(map[string]*engineEntry)
    nsEntries := make(map[string]map[string]*engineEntry)
    // run discover into scratch maps, then swap atomically
    if err := e.discoverInto(dir, entries, nsEntries); err != nil {
        return err
    }
    e.entries = entries
    e.nsEntries = nsEntries
    e.opts.ComponentDir = dir
    e.varComponentDir.Set(dir)
    return nil
}
```

#### `SetFS(fsys fs.FS) error`

Replaces the engine's `fs.FS` and re-runs discovery.

```go
// pseudo-code — not implementation

// SetFS replaces the engine's filesystem and re-runs discovery under the
// write-lock. Returns an error if discovery fails; on error the engine's
// FS is restored to its previous value.
//
// If fsys does not implement fs.StatFS, enabling Reload has no effect:
// maybeReload performs stat checks via fs.StatFS and returns immediately
// when the interface is absent.
func (e *Engine) SetFS(fsys fs.FS) error {
    e.mu.Lock()
    defer e.mu.Unlock()
    entries := make(map[string]*engineEntry)
    nsEntries := make(map[string]map[string]*engineEntry)
    savedFS := e.opts.FS
    e.opts.FS = fsys
    if err := e.discoverInto(e.opts.ComponentDir, entries, nsEntries); err != nil {
        e.opts.FS = savedFS
        return err
    }
    e.entries = entries
    e.nsEntries = nsEntries
    if fsys != nil {
        e.varFS.Set(reflect.TypeOf(fsys).String())
    } else {
        e.varFS.Set("<nil>")
    }
    return nil
}
```

#### Concurrency analysis

Both setters acquire `e.mu` for writing before touching any mutable engine
state.  Discovery runs against scratch maps; the swap into `e.entries` and
`e.nsEntries` is a single pointer assignment under the lock.  Concurrent
renders that hold `e.mu` for reading will complete before the write-lock is
granted.

#### Interaction with `counterReloads`

`SetFS` and `SetComponentDir` do **not** increment `counterReloads`.  These
are deliberate operator-driven replacements (e.g. switching from an embedded
FS to a live filesystem for a development mode toggle), not hot-reload events
triggered by filesystem changes.  `counterReloads` remains an accurate count
of automatic `maybeReload` re-walks only.

#### `SetFS` and `Reload` interaction

If `SetFS` is called with a non-nil `fsys` that does not implement `fs.StatFS`,
enabling `Reload` via `SetReload(true)` will have no observable effect:
`maybeReload` checks `fs.StatFS` availability before performing stat calls and
returns `nil` immediately when the interface is absent.  This is documented in
the godoc of `SetFS` (see pseudo-code above).

---

### 4.7 Multiple-engine registration and naming

#### The collision problem

`expvar.Publish` (and the `expvar.NewMap` wrapper used in §4.1) panics when a
name is registered more than once.  A process that creates two engines and
calls `PublishExpvars("htmlc")` on both will panic on the second call.

#### Recommended naming convention

The caller is responsible for supplying a unique prefix.  Two patterns cover
the common cases:

1. **Single engine**: `e.PublishExpvars("htmlc")`
2. **Multiple engines**: `api.PublishExpvars("htmlc.api")`, `admin.PublishExpvars("htmlc.admin")`

The dot (`.`) separator is conventional in `expvar` practice (e.g. `go.goroutines`,
`go.memstats`) but is not enforced by the package.  Any string that does not
collide with an existing global name is valid.

#### Design evaluation — caller-supplied vs auto-generated name

**Option A — Caller-supplied prefix (recommended)**

- ✅ Deterministic and human-readable in JSON output.
- ✅ Stable across restarts; operators can write scripts that expect consistent
  key names.
- ✅ Already the standard pattern: `expvar.Publish` requires a caller-supplied
  name.
- ⚠️ Requires the caller to choose a unique name; a shared helper that creates
  engines must coordinate naming.

**Option B — Auto-generated name (e.g. sequential integer)**

```go
// pseudo-code — not implementation
var engineCount int64
name := fmt.Sprintf("htmlc.%d", atomic.AddInt64(&engineCount, 1))
expvar.NewMap(name)
```

- ✅ Never collides without any caller effort.
- ❌ Non-deterministic across restarts (order of engine construction may vary).
- ❌ Keys in `/debug/vars` JSON are opaque integers; operators cannot tell which
  engine is which.
- ❌ Requires a process-global counter, introducing package-level state.

**Verdict**: Option A.  The caller must supply a unique prefix to `PublishExpvars`.
A call with a duplicate prefix panics immediately (consistent with `expvar.Publish`
semantics), making the error impossible to miss.

#### JSON output — two engines

```json
{
  "htmlc.api": {
    "reload": 0,
    "debug": 0,
    "componentDir": "templates/api/",
    "fs": "<nil>",
    "renders": 500,
    "renderErrors": 0,
    "renderNanos": 21000000,
    "components": 15,
    "info": {
      "directives": []
    }
  },
  "htmlc.admin": {
    "reload": 1,
    "debug": 0,
    "componentDir": "templates/admin/",
    "fs": "<nil>",
    "renders": 120,
    "renderErrors": 2,
    "renderNanos": 8100000,
    "components": 9,
    "info": {
      "directives": []
    }
  }
}
```

---

## 5. Syntax Summary

*Not applicable — this RFC introduces no new template syntax.*

---

## 6. Examples

### Example 1 — Single engine with default prefix

A production server with one engine and standard expvar publication:

```go
// main.go
package main

import (
    "net/http"
    _ "net/http/expvar" // registers /debug/vars handler
    "github.com/dhamidi/htmlc"
)

func main() {
    engine, err := htmlc.New(htmlc.Options{
        ComponentDir: "templates/",
        Reload:       false,
        Debug:        false,
    })
    if err != nil {
        panic(err)
    }

    // Publish engine state and counters under "htmlc".
    engine.PublishExpvars("htmlc")

    http.Handle("/", engine.ServePageComponent("Page", nil))
    http.ListenAndServe(":8080", nil)
}
```

After 1 000 requests, `curl localhost:8080/debug/vars` returns:

```json
{
  "htmlc": {
    "reload": 0,
    "debug": 0,
    "componentDir": "templates/",
    "fs": "<nil>",
    "renders": 1000,
    "renderErrors": 0,
    "reloads": 0,
    "renderNanos": 45200000,
    "components": 23,
    "info": {
      "directives": []
    }
  }
}
```

### Example 2 — Development server with runtime Reload toggle

A development server starts with `Reload: false` and exposes an admin endpoint
that enables hot-reload without restarting:

```go
// pseudo-code — not implementation

engine, _ := htmlc.New(htmlc.Options{ComponentDir: "templates/"})
engine.PublishExpvars("htmlc")

// Admin endpoint — toggles reload without restart.
http.HandleFunc("/admin/reload/enable", func(w http.ResponseWriter, r *http.Request) {
    engine.SetReload(true)
    fmt.Fprintln(w, "reload enabled")
})
http.HandleFunc("/admin/reload/disable", func(w http.ResponseWriter, r *http.Request) {
    engine.SetReload(false)
    fmt.Fprintln(w, "reload disabled")
})
```

`/debug/vars` reflects the change immediately after the toggle:

```json
{ "htmlc": { "reload": 1, "debug": 0, "componentDir": "templates/", "fs": "<nil>", ... } }
```

No restart is required; the next `renderComponent` call will invoke
`maybeReload`, which now reads `varReload.Value() == 1` and proceeds with the
stat check.

### Example 3 — Multiple engines with distinct prefixes

A service that renders public-facing and admin pages from separate component
directories:

```go
// pseudo-code — not implementation

public, _ := htmlc.New(htmlc.Options{ComponentDir: "templates/public/"})
public.PublishExpvars("htmlc.public")

admin, _ := htmlc.New(htmlc.Options{ComponentDir: "templates/admin/", Debug: true})
admin.PublishExpvars("htmlc.admin")
```

`/debug/vars` output:

```json
{
  "htmlc.public": {
    "reload": 0,
    "debug": 0,
    "componentDir": "templates/public/",
    "fs": "<nil>",
    "renders": 800,
    "renderErrors": 0,
    "renderNanos": 36000000,
    "components": 18,
    "info": {
      "directives": []
    }
  },
  "htmlc.admin": {
    "reload": 0,
    "debug": 1,
    "componentDir": "templates/admin/",
    "fs": "<nil>",
    "renders": 42,
    "renderErrors": 0,
    "renderNanos": 3800000,
    "components": 9,
    "info": {
      "directives": ["highlight"]
    }
  }
}
```

### Example 4 — Embedded filesystem with expvar inspection

An engine backed by an embedded `fs.FS`:

```go
// pseudo-code — not implementation

//go:embed templates
var templateFS embed.FS

engine, _ := htmlc.New(htmlc.Options{
    FS:           templateFS,
    ComponentDir: "templates",
})
engine.PublishExpvars("htmlc")
```

The `fs` key reveals the concrete FS type without exposing private
fields of the `embed.FS` value:

```json
{
  "htmlc": {
    "reload": 0,
    "debug": 0,
    "componentDir": "templates",
    "fs": "embed.FS",
    "renders": 0,
    "renderErrors": 0,
    "reloads": 0,
    "renderNanos": 0,
    "components": 12,
    "info": {
      "directives": []
    }
  }
}
```

Because `embed.FS` does not implement `fs.StatFS`, hot-reload is silently
skipped — the `fs` value gives the operator exactly enough
information to understand why `reload: 1` would have no effect.

### Example 5 — Runtime ComponentDir replacement

Switching from a compiled-in directory to a live directory during development:

```go
// pseudo-code — not implementation

engine, _ := htmlc.New(htmlc.Options{ComponentDir: "templates/dist/"})
engine.PublishExpvars("htmlc")

// Admin endpoint — swap to live source directory without restart.
http.HandleFunc("/admin/dev-mode", func(w http.ResponseWriter, r *http.Request) {
    if err := engine.SetComponentDir("templates/src/"); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    engine.SetReload(true)
    fmt.Fprintln(w, "dev mode enabled")
})
```

After hitting `/admin/dev-mode`, `/debug/vars` reflects both changes:

```json
{
  "htmlc": {
    "reload": 1,
    "componentDir": "templates/src/",
    ...
  }
}
```

### Example 6 — Engine without `PublishExpvars` (backward compatibility)

```go
// pseudo-code — not implementation

// Existing code — no change required.
engine, _ := htmlc.New(htmlc.Options{
    ComponentDir: "templates/",
    Reload:       true,
})
// PublishExpvars is never called.
// No global expvar side effects.  Behaviour identical to today.
http.HandleFunc("/", engine.ServePageComponent("Page", nil).ServeHTTP)
```

The engine's `varReload`, `varDebug`, and counter fields are allocated and
seeded during `New`, but because `PublishExpvars` was never called they are
private to the `Engine` struct and invisible to `/debug/vars`.  The only
observable change from the caller's perspective is that internal reads go
through `varReload.Value()` instead of `opts.Reload` — a transparent substitution.

---

## 7. Implementation Sketch

This section lists Go-level changes by file.  Full implementation is out of
scope for this RFC.

### `engine.go`

1. **`Engine` struct**: add ten new fields as described in §4.1 (`varReload`,
   `varDebug`, `varComponentDir`, `varFS`, `varDirectives`,
   `counterRenders`, `counterRenderErrors`, `counterReloads`,
   `counterRenderNanos`, `counterComponents`, `expvarMap`, `expvarPrefix`).

2. **`New`**: after constructing `e`, allocate and seed all new fields before
   calling `discover` (so that `counterComponents` reflects the discovered set).
   Add import: `"reflect"`, `"sort"`.

3. **`maybeReload`**: replace `if !e.opts.Reload` with
   `if e.varReload.Value() == 0`.  Add `e.counterReloads.Add(1)` immediately
   before the `e.entries = make(...)` line.

4. **`renderComponent`**: four changes:
   - Add `e.counterRenders.Add(1)` at function entry.
   - Add `start := time.Now()` and a `defer` that calls
     `e.counterRenderNanos.Add(time.Since(start).Nanoseconds())`.
   - Add `e.counterRenderErrors.Add(1)` at each `return nil, err` site.
   - Replace `if e.opts.Debug {` with `if e.varDebug.Value() != 0 {`.

5. **`componentCountDedup`**: new unexported method (4–8 lines) used by
   `counterComponents`'s `expvar.Func`.

6. **`PublishExpvars`**: new exported method (~20 lines, includes building the
   `info` sub-map).

7. **`SetReload` / `SetDebug`**: two new exported methods (~4 lines each).

8. **`SetComponentDir`**: new exported method (~15 lines); acquires write-lock,
   discovers into scratch maps, swaps atomically, updates `e.opts.ComponentDir`
   and `e.varComponentDir`.

9. **`SetFS`**: new exported method (~20 lines); acquires write-lock, saves old
   FS for rollback, discovers into scratch maps, swaps atomically, updates
   `e.opts.FS` and `e.varFS`.

10. **`discoverInto`**: new unexported helper that accepts target `entries` and
    `nsEntries` maps as parameters; extracted from `discover` to support the
    scratch-map pattern used by both `SetComponentDir` and `SetFS`.

11. **Imports**: add `"expvar"`, `"reflect"`, `"sort"` to the import block.
    `"time"` is already imported.

### `renderer.go`

No changes required.  All instrumentation is at the `Engine` layer.

### `doc.go`

Update the package-level comment to document `PublishExpvars`, `SetReload`,
`SetDebug`, `SetComponentDir`, and `SetFS` under the existing "Tutorial"
section.

### Platform considerations

- All `expvar` types use `sync/atomic` internally and are OS-agnostic.
- `reflect.TypeOf(opts.FS).String()` works correctly for any `fs.FS`
  implementation, including pointer receivers and non-pointer receivers.  When
  `opts.FS` is `nil` the explicit `"<nil>"` string branch avoids a nil-pointer
  dereference in `reflect.TypeOf`.
- The `expvar` HTTP handler is registered by importing `net/http/expvar` (blank
  import).  The `htmlc` package does **not** import `net/http/expvar` itself;
  doing so would register the `/debug/vars` handler as a side effect for every
  `htmlc` user regardless of whether they want it.  Callers who want the HTTP
  handler must import `net/http/expvar` explicitly.

---

## 8. Backward Compatibility

### `Options` struct

No new fields.  Fully backward compatible.

### `Engine` struct

`Engine` is an opaque struct; callers cannot create it directly and cannot
access its fields.  Adding unexported fields is backward compatible.

### `New` function

Signature unchanged: `func New(opts Options) (*Engine, error)`.  The internal
allocation of expvar vars inside `New` is transparent.

### Public methods — no changes to existing signatures

`RenderPage`, `RenderPageContext`, `RenderFragment`, `RenderFragmentContext`,
`RenderPageString`, `RenderFragmentString`, `ServeComponent`,
`ServePageComponent`, `Mount`, `Register`, `Components`, `Has`, `ValidateAll`,
`WithMissingPropHandler`, `RegisterDirective`, `RegisterFunc`,
`WithDataMiddleware` — all signatures and behaviours are unchanged.

### Behaviour change — `maybeReload` and `renderComponent`

The reads of `opts.Reload` and `opts.Debug` are replaced by `varReload.Value()`
and `varDebug.Value()` respectively.  Because these vars are seeded from `opts`
during `New`, the behaviour for all existing callers is identical.  A caller who
never calls `SetReload` or `PublishExpvars` and never modifies `varReload`
externally will observe no behavioural difference.

### New exported symbols

Five new exported methods are added to `Engine`:

| Symbol | Signature | Notes |
|--------|-----------|-------|
| `PublishExpvars` | `func (e *Engine) PublishExpvars(prefix string) *Engine` | Opt-in; panics on duplicate prefix |
| `SetReload` | `func (e *Engine) SetReload(enabled bool)` | Convenience wrapper |
| `SetDebug` | `func (e *Engine) SetDebug(enabled bool)` | Convenience wrapper |
| `SetComponentDir` | `func (e *Engine) SetComponentDir(dir string) error` | Re-runs discovery under write-lock |
| `SetFS` | `func (e *Engine) SetFS(fsys fs.FS) error` | Re-runs discovery under write-lock |

These are purely additive.  No existing code needs to be updated.

### Performance impact

Each `renderComponent` call incurs two additional `sync/atomic` operations
(`counterRenders.Add(1)` and `counterRenderNanos.Add(delta)`) plus one atomic
read (`varDebug.Value()`).  `maybeReload` incurs one additional atomic read
(`varReload.Value()`).  Atomic operations on modern amd64 and arm64 hardware
complete in 1–3 nanoseconds.  For a render that takes 10–100 µs the overhead
is below 0.1 % and should not be perceptible.

---

## 9. Alternatives Considered

### Flat naming (`"htmlc.reload"`, `"htmlc.debug"`, etc.)

Register each variable as a separate top-level expvar entry with a prefix:
`expvar.Publish("htmlc.reload", varReload)`.

- ✅ Simpler; no nested `expvar.Map` required.
- ❌ Each engine would register `O(n)` names in the global registry; with two
  engines each registering 10 vars under different prefixes, an operator looking
  at `/debug/vars` JSON gets a flat list of 20+ keys.
- ❌ Prefix-collision detection requires checking each key individually rather
  than checking a single map name.
- ❌ The JSON output is less structured; aggregation tools must infer grouping
  from key-name prefixes.

**Verdict**: Rejected.  The nested `expvar.Map` approach (§4.1) is more
structured and scales cleanly to multiple engines.

### `expvar.Bool` custom type

Implement a thread-safe `Bool` wrapping `sync/atomic` and register `Reload` and
`Debug` as booleans in JSON.

- ✅ More semantically correct type.
- ❌ Requires a new unexported or exported type; adds package surface.
- ❌ `expvar` does not provide a built-in `Bool`; a custom type would not be
  automatically discovered by tools that scrape `/debug/vars` expecting only
  `Int`, `Float`, `String`, and `Map`.
- ❌ Consumers of the JSON cannot simply do `if v == 1` — they must handle a
  JSON boolean, which increases client-side complexity for no real gain.

**Verdict**: Rejected.  `expvar.Int` with `0`/`1` convention is sufficient and
standard.

### `Options.ExpvarPrefix` field

Include an `ExpvarPrefix string` field in `Options`; non-empty triggers
`expvar.NewMap` during `New`.

- ✅ One configuration point; consistent with other `Options` fields.
- ❌ Unavoidable side effects at construction time; makes test isolation harder.
- ❌ Cannot defer publication to after construction.
- ❌ Breaks the established opt-in pattern for instrumentation.

**Verdict**: Rejected in favour of `PublishExpvars` builder method.

### Tracking `counterComponents` incrementally

Increment `counterComponents` in `registerPathLocked` and decrement (or reset)
in `maybeReload` instead of computing it live.

- ✅ O(1) read at serialisation time.
- ❌ De-duplication logic (same `*engineEntry` pointer registered under both
  `"Button"` and `"button"`) is duplicated or must call a helper.
- ❌ A reset-and-recount during hot-reload could temporarily expose a zero count
  between clearing `entries` and completing `discover`.
- ⚠️ `expvar.Func` with a read-locked map scan is O(n) in component count but
  only evaluated at operator query frequency, not on every render.

**Verdict**: Rejected.  The `expvar.Func` live computation avoids a class of
consistency bugs for minimal cost.

### Per-component render counters

Track render counts per component name, not just globally.

- ✅ Enables identification of hot paths.
- ❌ Requires dynamic creation of `expvar.Map` sub-keys during the first render
  of each component; introduces synchronisation at the `expvar.Map` level.
- ❌ An engine with hundreds of components would generate hundreds of sub-keys;
  the JSON output becomes unwieldy.
- ❌ Out of scope for an initial implementation.

**Verdict**: Deferred to a follow-on RFC or an additive extension to §4.4.

### Embedding `expvar.Map` directly in `Engine`

Store the `expvar.Map` as a value field rather than a pointer, initialised
during `New`.

- ❌ `expvar.Map` must be registered globally via `expvar.NewMap` or
  `expvar.Publish` to appear in `/debug/vars`.  An un-registered in-line `Map`
  value is invisible to the HTTP handler.
- ❌ Embedding a zero `expvar.Map` value in every `Engine` (including those that
  never call `PublishExpvars`) wastes memory and requires its `Init` method to
  be called before use.

**Verdict**: Rejected.  `expvarMap *expvar.Map` (nil until `PublishExpvars` is
called) is the correct design.

---

## 10. Open Questions

1. **(blocking) Panic vs error on duplicate prefix**: `expvar.NewMap` panics on
   a duplicate name, consistent with `expvar.Publish`.  Should `PublishExpvars`
   follow this convention or return an error?  Returning an error would require
   the signature to become `PublishExpvars(prefix string) (*Engine, error)`,
   which breaks the builder chain.  An alternative is a separate
   `TryPublishExpvars(prefix string) error` alongside the panicking
   `PublishExpvars`.  Recommendation: follow `expvar.Publish` semantics and
   panic; document this explicitly in the method godoc.

2. **(blocking) Exported `SetReload` / `SetDebug` vs direct field access**:
   Should `varReload` and `varDebug` be exported so callers can write
   `e.VarReload.Set(1)` directly?  Recommendation: keep them unexported and
   provide `SetReload` / `SetDebug` for controlled mutation.  Exporting the
   concrete `*expvar.Int` types would allow callers to call `Add(-1)` or
   other arithmetic operations that make no semantic sense for a boolean toggle.

3. **(non-blocking) `SetReload(true)` with no `ComponentDir`**: if `Reload` is
   enabled at runtime but `ComponentDir` was empty at construction time, there
   are no entries to stat and `maybeReload` returns immediately.  This is
   already the correct behaviour (the `for _, entry := range e.entries` loop
   over an empty map exits immediately).  A diagnostic warning log might be
   useful here; deferred to implementation.

4. **(non-blocking) `counterRenderNanos` overflow**: `expvar.Int` is `int64`;
   the maximum value is approximately 9.2 × 10¹⁸ nanoseconds ≈ 292 years of
   cumulative render time.  Overflow in practice is impossible.  No action
   required.

5. **(non-blocking) Gauge vs counter semantics for `counterComponents`**: the
   component count behaves as a gauge (it can decrease after hot-reload removes
   a file), not a monotonic counter.  The field name `counterComponents` may be
   misleading.  Alternative names: `gaugeComponents`, `components`.  Naming
   decision can be made during implementation without blocking the design.

6. **(non-blocking) Per-component latency tracking**: identifying slow
   components requires per-component counters, which are explicitly deferred
   (see §9).  If a follow-on RFC adds per-component counters, the existing
   `counterRenders` and `counterRenderNanos` at the engine level remain useful
   as aggregate summaries.

7. **(non-blocking) Integration with `net/http/pprof`**: `expvar` and `pprof`
   both register under `/debug/` by default.  A caller who imports both
   automatically gets both endpoints.  No coordination between `htmlc` and
   `pprof` is required; this is purely an operator concern.
