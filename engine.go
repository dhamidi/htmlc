package htmlc

import (
	"context"
	"expvar"
	"fmt"
	htmltmpl "html/template"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// Options holds configuration for creating a new Engine.
type Options struct {
	// ComponentDir is the directory to scan recursively for *.vue files.
	// Components are discovered by walking the tree in lexical order; each file
	// is registered by its base name without extension (e.g. "Button.vue"
	// becomes "Button").
	//
	// When ComponentDir is set the engine uses proximity-based resolution:
	// a tag reference in a template is first looked up in the same directory
	// as the calling component, then walks toward the root one level at a time.
	// This allows same-named components in different subdirectories to coexist
	// without conflict. See the README for details and examples.
	//
	// When two files share the same base name and directory the last one
	// encountered in lexical-order traversal wins in the flat registry
	// (backward-compatibility fallback path).
	ComponentDir string
	// Reload enables hot-reload for development use. When true, the engine
	// checks the modification time of every registered component file before
	// each render and automatically re-parses any file that has changed since
	// it was last loaded. This lets you edit .vue files and see the results
	// without restarting the server. For production, leave Reload false and
	// create the Engine once at startup.
	//
	// When FS is also set, reload only works if the FS implements fs.StatFS.
	// If it does not, reload is silently skipped for all entries.
	Reload bool
	// FS, when set, is used instead of the OS filesystem for all file reads
	// and directory walks. ComponentDir is then interpreted as a path within
	// this FS. This allows callers to use embedded filesystems (//go:embed),
	// in-memory virtual filesystems, or any other fs.FS implementation.
	//
	// When FS is nil, the OS filesystem is used (default behaviour).
	FS fs.FS
	// Directives registers custom directives available to all components rendered
	// by this engine. Keys are directive names without the "v-" prefix
	// (e.g. "switch" handles v-switch). Built-in directives (v-if, v-for, etc.)
	// cannot be overridden.
	Directives DirectiveRegistry
	// Debug enables debug render mode. When true, the root element of each
	// rendered component carries three data-htmlc-* attributes:
	// data-htmlc-component (component name), data-htmlc-file (source path),
	// and data-htmlc-props (JSON-encoded props). If props cannot be
	// serialised, data-htmlc-props-error is emitted instead. Debug mode
	// has no effect on components whose template has no single root element
	// (fragment templates). Debug mode should not be used in production as
	// it adds extra attributes and increases output size.
	Debug bool
	// Logger, if non-nil, receives one structured log record per component
	// rendered. Records are emitted at slog.LevelDebug for successful renders
	// and slog.LevelError for failed renders. Each record includes the
	// component name, render duration (subtree), bytes written (subtree), and
	// any error. The nil value (default) disables all slog output.
	Logger *slog.Logger
	// ComponentErrorHandler, if non-nil, is called in place of aborting the
	// render when a child component fails. The handler writes an HTML
	// placeholder to w and returns nil to continue rendering, or returns a
	// non-nil error to abort. When the handler returns nil for all failures,
	// the partial page (with placeholders) is written to the io.Writer passed
	// to RenderPage. The nil value (default) preserves the existing behaviour:
	// the first component error aborts the render and w receives nothing.
	ComponentErrorHandler ComponentErrorHandler
	// Mounts registers additional, independently-sourced component trees
	// alongside the primary FS/ComponentDir pair. Each Mount's components are
	// addressed under Mount.Prefix (e.g. Prefix: "radix" makes its
	// Accordion.vue reachable as <component is="radix/Accordion">) and also
	// participate in the ordinary flat-registry fallback, so an unqualified
	// <Accordion> resolves through it whenever the bare name is unambiguous.
	//
	// Every component discovered under a Mount also gets two automatic
	// flat-registry aliases derived from Mount.Prefix plus every
	// intermediate directory segment plus the base name: a PascalCase form
	// (radix/dialog/Trigger.vue -> "RadixDialogTrigger") and a kebab-case
	// form using the same conversion as custom-element tag derivation
	// (-> "radix-dialog-trigger"), so a disambiguated reference can use
	// ordinary tag syntax instead of always requiring the explicit
	// "prefix/Name" form.
	//
	// Mounts is purely additive: setting it does not require setting
	// FS/ComponentDir, and vice versa. Adding Mounts to an existing
	// FS/ComponentDir configuration never changes how the primary source's
	// own files are discovered, tagged, or resolved — only Mount.Prefix
	// itself must not collide with a top-level directory name already
	// present under ComponentDir (the engine does not currently detect this
	// case; it is a caller responsibility).
	//
	// When a mount-defined name (bare or alias) collides with another
	// mount's or with a locally-defined component, the local component (or
	// the earliest-processed Mount, in Mounts order) wins the flat-registry
	// slot; the loser is still reachable via its explicit "prefix/Name"
	// path. This is a coarse, temporary priority rule — real collision
	// detection (erroring instead of silently picking a winner) is not
	// implemented yet.
	Mounts []Mount
	// NativeElements lists hyphenated tag names that must never be resolved
	// against the component registry, even though they contain a hyphen.
	// Their attributes and children render exactly as a normal HTML element's
	// would (expressions evaluated, v-if/v-for honored, no component lookup).
	// Unlisted hyphenated tags retain today's behavior: resolve against the
	// registry, or fail with "unknown component" if not found.
	//
	// Matching is case-insensitive; the HTML parser already lowercases tag
	// names, but entries here are lowercased defensively too.
	//
	// For a one-off foreign tag that doesn't warrant a permanent, project-wide
	// entry here, use the per-element v-native attribute instead (see the
	// README/RFC for details); the two mechanisms are complementary.
	NativeElements []string
}

// Mount registers an additional, independently-sourced component tree
// alongside the primary Options.FS/ComponentDir pair.
type Mount struct {
	// Prefix is the namespace prefix this mount's components are addressed
	// under (e.g. "radix" makes its Accordion.vue reachable as
	// <component is="radix/Accordion">). Prefix must be non-empty, must not
	// contain "/", and must be unique across all of Options.Mounts — New
	// returns an error immediately if any of these is violated.
	Prefix string
	// FS is the component source for this mount.
	FS fs.FS
	// Dir is the path within FS to scan, mirroring ComponentDir. Empty
	// means the root of FS (equivalent to ".").
	Dir string
}

// engineEntry holds a parsed component together with its source path and the
// mtime at which it was last parsed.
type engineEntry struct {
	path    string
	comp    *Component
	modTime time.Time
	// mountID identifies which source registered this entry: "" for the
	// primary Options.FS/ComponentDir source, or the owning Mount.Prefix for
	// a component discovered inside a Mount. Not yet consulted by anything
	// (collision detection across mounts is a later commit) — it exists so
	// that later logic can distinguish "which source is this entry from"
	// without re-deriving it from the entry's path.
	mountID string
}

// Engine is the entry point for rendering .vue components. Create one with
// New; call RenderPage or RenderFragment to produce HTML. ServeComponent wraps
// a component as a net/http handler so it can be mounted directly in an
// http.Handler-based server.
//
// Engine is safe for concurrent use. All render methods may be called from
// multiple goroutines simultaneously.
type Engine struct {
	opts               Options
	mu                 sync.RWMutex // guards entries, nsEntries, entryMountIDs, and collector
	entries            map[string]*engineEntry
	nsEntries          map[string]map[string]*engineEntry // relDir → localName → entry
	collector          *CustomElementCollector
	missingPropHandler MissingPropFunc
	directives         DirectiveRegistry
	funcs              map[string]any // per-engine functions, injected into every render scope
	dataMiddleware     []func(*http.Request, map[string]any) map[string]any

	// mountPrefixes holds every configured Options.Mounts[i].Prefix, set once
	// in New and never mutated afterward (Mounts cannot currently be changed
	// at runtime), so it is safe to read without holding mu. It is consulted
	// by relDirForPath and threaded into the Renderer (via
	// WithMountPrefixes) so a mount-sourced component's proximity directory
	// is computed relative to its own mount instead of collapsing to "".
	mountPrefixes []string

	// entryMountIDs records, for every flat registry name ever attempted
	// during discovery, every distinct mountID ("" for the primary source, or
	// a Mount.Prefix) that attempted to register that name — in registration
	// order, deduplicated per (name, mountID) pair. Unlike e.entries (which
	// only holds the single entry that actually won the flat-registry slot,
	// per the insert-if-absent priority rule), entryMountIDs also remembers
	// attempts that lost — a mount whose registration was shadowed by a local
	// or earlier-mount entry of the same name is still recorded here.
	//
	// Nothing reads this field yet: it exists so a later commit implementing
	// cross-mount collision detection (RFC 014 §4.2's checkMountCollisions)
	// does not need to re-plumb discovery to recover this information.
	entryMountIDs map[string][]string

	// expvar-backed option vars
	varReload       *expvar.Int
	varDebug        *expvar.Int
	varComponentDir *expvar.String
	varFS           *expvar.String
	varDirectives   *expvar.Func

	// performance counters
	counterRenders      *expvar.Int
	counterRenderErrors *expvar.Int
	counterReloads      *expvar.Int
	counterRenderNanos  *expvar.Int
	counterComponents   *expvar.Func

	// global registry integration
	expvarMap    *expvar.Map
	expvarPrefix string

	cw countingWriter // used by loggedRender for root-level instrumentation
}

// WithMissingPropHandler sets the function called when any component rendered
// by this engine has a missing prop. The default behaviour (when no handler is
// set) is to render a visible "[missing: <name>]" placeholder in place of the
// prop value. Use ErrorOnMissingProp to restore strict error behaviour, or
// SubstituteMissingProp to use the legacy "MISSING PROP: <name>" format.
func (e *Engine) WithMissingPropHandler(fn MissingPropFunc) *Engine {
	e.missingPropHandler = fn
	return e
}

// RegisterDirective adds a custom directive to the engine under the given name
// (without the "v-" prefix). It replaces any previously registered directive
// with the same name. Panics if dir is nil.
func (e *Engine) RegisterDirective(name string, dir Directive) {
	if dir == nil {
		panic("htmlc: RegisterDirective: dir must not be nil")
	}
	if e.directives == nil {
		e.directives = make(DirectiveRegistry)
	}
	e.directives[name] = dir
}

// RegisterFunc adds a per-engine function available in all template expressions
// rendered by this engine. The function can be called from templates as name().
// Engine-level functions act as lower-priority builtins: the render data scope
// takes precedence over them, which in turn takes precedence over the global
// expr.RegisterBuiltin table.
//
// Functions registered here are propagated automatically into every child
// component's scope, so they are available at any nesting depth without
// being threaded through as explicit props.
//
// RegisterFunc returns the Engine so calls can be chained.
func (e *Engine) RegisterFunc(name string, fn func(...any) (any, error)) *Engine {
	if e.funcs == nil {
		e.funcs = make(map[string]any)
	}
	e.funcs[name] = fn
	return e
}

// WithDataMiddleware adds a function that is called on every HTTP-triggered
// render to augment the data map. Middleware functions are called in
// registration order; later middleware can overwrite keys set by earlier ones.
//
// Data middleware applies only to the HTTP-triggered top-level render and is
// not automatically propagated into child component scopes. If child components
// need access to values injected by middleware, those values must be passed as
// explicit props or the same values should be registered via RegisterFunc.
//
// WithDataMiddleware returns the Engine so calls can be chained.
func (e *Engine) WithDataMiddleware(fn func(*http.Request, map[string]any) map[string]any) *Engine {
	e.dataMiddleware = append(e.dataMiddleware, fn)
	return e
}

// New creates an Engine configured by opts. If opts.ComponentDir is set the
// directory is walked recursively and all *.vue files are registered.
func New(opts Options) (*Engine, error) {
	if err := validateMountPrefixes(opts.Mounts); err != nil {
		return nil, err
	}

	// mountPrefixes is captured before opts.FS is possibly rewritten to the
	// union below, since it only ever needs the Prefix strings, not the
	// mounts' filesystems.
	var mountPrefixes []string
	for _, m := range opts.Mounts {
		mountPrefixes = append(mountPrefixes, m.Prefix)
	}

	if len(opts.Mounts) > 0 {
		primaryFS := opts.FS
		if primaryFS == nil {
			// Mounts are combined into a union fs.FS (see unionfs.go), which
			// requires a real fs.FS for the primary source too — fall back
			// to the OS filesystem rooted at the process's working
			// directory. This only supports a *relative* ComponentDir
			// (matching every example in RFC 014, §4.1) because fs.FS's own
			// path rules (fs.ValidPath) reject absolute paths outright.
			// Detect that unsupported combination here and fail clearly,
			// rather than letting it surface later as a confusing
			// "open <path>: invalid argument" error during discovery.
			if filepath.IsAbs(opts.ComponentDir) {
				return nil, fmt.Errorf("engine: Options.Mounts requires a relative ComponentDir when Options.FS is nil (got absolute path %q)", opts.ComponentDir)
			}
			primaryFS = os.DirFS(".")
		}

		mounts := make([]fsMount, 0, len(opts.Mounts)+1)
		mounts = append(mounts, fsMount{prefix: "", fs: primaryFS})
		for i, m := range opts.Mounts {
			dir := m.Dir
			if dir == "" {
				dir = "."
			}
			sub, err := fs.Sub(m.FS, dir)
			if err != nil {
				return nil, fmt.Errorf("engine: Options.Mounts[%d] (prefix %q): fs.Sub(%q): %w", i, m.Prefix, dir, err)
			}
			mounts = append(mounts, fsMount{prefix: m.Prefix, fs: sub})
		}

		// e.opts.FS becomes the union of the primary source (mounted at "")
		// and every Mount (mounted at its own Prefix). Deliberately NOT
		// touched: opts.ComponentDir. Resetting it to "" (as RFC 014 §4.1's
		// simplified pseudocode suggests) would make discoverInto's
		// ComponentDir-relative custom-element tag re-derivation operate
		// against the wrong base for every existing primary-source
		// component the moment any Mount is added — see the Options.Mounts
		// doc comment and this package's design notes. Leaving
		// ComponentDir untouched means the discovery call below behaves
		// byte-for-byte as it did before Mounts existed; only the mount
		// walk further down is new.
		opts.FS = newUnionFS(mounts)
	}

	e := &Engine{
		opts:          opts,
		entries:       make(map[string]*engineEntry),
		nsEntries:     make(map[string]map[string]*engineEntry),
		directives:    opts.Directives,
		mountPrefixes: mountPrefixes,
		entryMountIDs: make(map[string][]string),
	}

	// Initialise expvar-backed option vars (not registered globally).
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
	if opts.FS == nil {
		e.varFS.Set("<nil>")
	} else {
		e.varFS.Set(reflect.TypeOf(opts.FS).String())
	}
	varDirectives := expvar.Func(func() any {
		e.mu.RLock()
		defer e.mu.RUnlock()
		names := make([]string, 0, len(e.directives))
		for k := range e.directives {
			names = append(names, k)
		}
		sort.Strings(names)
		return names
	})
	e.varDirectives = &varDirectives

	// Performance counters.
	e.counterRenders = new(expvar.Int)
	e.counterRenderErrors = new(expvar.Int)
	e.counterReloads = new(expvar.Int)
	e.counterRenderNanos = new(expvar.Int)
	counterComponents := expvar.Func(func() any {
		return int64(e.componentCountDedup())
	})
	e.counterComponents = &counterComponents

	e.collector = NewCustomElementCollector()
	if opts.ComponentDir != "" {
		if err := e.discover(opts.ComponentDir); err != nil {
			return nil, err
		}
	}
	// Record the primary source's own attempts ("" mountID) before touching
	// any mount, so entryMountIDs preserves registration order across
	// sources (primary always precedes every mount).
	recordPrimaryMountIDs(e.entries, e.entryMountIDs)
	for _, m := range opts.Mounts {
		if err := e.discoverMountInto(m, e.entries, e.nsEntries, e.entryMountIDs); err != nil {
			return nil, err
		}
	}
	e.rebuildCollectorLocked()
	return e, nil
}

// componentCountDedup returns the number of unique components registered in
// the engine (excluding automatic lowercase aliases).
func (e *Engine) componentCountDedup() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	seen := make(map[*engineEntry]bool, len(e.entries))
	for _, entry := range e.entries {
		seen[entry] = true
	}
	return len(seen)
}

// rebuildCollectorLocked rebuilds the internal CustomElementCollector from
// e.entries. It must be called while holding e.mu for writing.
func (e *Engine) rebuildCollectorLocked() {
	c := NewCustomElementCollector()
	seen := make(map[*engineEntry]bool, len(e.entries))
	for _, entry := range e.entries {
		if seen[entry] {
			continue
		}
		seen[entry] = true
		if entry.comp != nil && entry.comp.CustomElementScript != "" {
			c.Add(entry.comp.CustomElementTag, entry.comp.CustomElementScript)
		}
	}
	e.collector = c
}

// discover walks dir in lexical order and registers every *.vue file found.
func (e *Engine) discover(dir string) error {
	return e.discoverInto(dir, e.entries, e.nsEntries)
}

// discoverInto walks dir in lexical order and registers every *.vue file into
// the provided entries and nsEntries maps. It does not modify e.entries or
// e.nsEntries directly, allowing callers to swap maps atomically.
func (e *Engine) discoverInto(dir string, entries map[string]*engineEntry, nsEntries map[string]map[string]*engineEntry) error {
	registerInto := func(name, path string) error {
		var (
			data []byte
			err  error
		)
		if e.opts.FS != nil {
			data, err = fs.ReadFile(e.opts.FS, path)
		} else {
			data, err = os.ReadFile(path)
		}
		if err != nil {
			return fmt.Errorf("engine: read %s: %w", path, err)
		}
		comp, err := ParseFile(path, string(data))
		if err != nil {
			return err
		}
		// Re-derive the custom element tag using the path relative to
		// ComponentDir so the directory name is not included in the tag.
		if e.opts.ComponentDir != "" && comp.CustomElementTag != "" {
			compSlash := filepath.ToSlash(filepath.Clean(path))
			dirSlash := filepath.ToSlash(filepath.Clean(e.opts.ComponentDir))
			relPath := compSlash
			if dirSlash != "" && dirSlash != "." {
				prefix := dirSlash + "/"
				if strings.HasPrefix(compSlash, prefix) {
					relPath = compSlash[len(prefix):]
				}
			}
			reviseCustomElementTagWarning(comp, path, deriveCustomElementTag(relPath))
		}
		var modTime time.Time
		if e.opts.FS != nil {
			if statFS, ok := e.opts.FS.(fs.StatFS); ok {
				if info, statErr := statFS.Stat(path); statErr == nil {
					modTime = info.ModTime()
				}
			}
		} else {
			if info, statErr := os.Stat(path); statErr == nil {
				modTime = info.ModTime()
			}
		}
		entry := &engineEntry{path: path, comp: comp, modTime: modTime}
		entries[name] = entry
		if lower := strings.ToLower(name); lower != name {
			entries[lower] = entry
		}
		// Populate the namespaced registry when ComponentDir is set.
		if e.opts.ComponentDir != "" {
			relDir := nsRelDir(path, e.opts.ComponentDir)
			if nsEntries[relDir] == nil {
				nsEntries[relDir] = make(map[string]*engineEntry)
			}
			nsEntries[relDir][name] = entry
		}
		return nil
	}

	if e.opts.FS != nil {
		return fs.WalkDir(e.opts.FS, dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				// When Options.Mounts is set, e.opts.FS is the union of the
				// primary source and every mount (see New/unionfs.go). If
				// dir is (or contains) the union root, an ordinary recursive
				// walk would descend straight into a mount's own
				// synthesized subdirectory and register its files as if
				// they were primary-source entries — bypassing
				// discoverMountInto's insert-if-absent priority rule
				// entirely and re-deriving their tags/relDir against the
				// wrong base. Prune any directory that is exactly a
				// registered mount's own root so the primary walk only ever
				// sees the primary source's own files, regardless of
				// ComponentDir.
				if isMountRootDir(e.mountPrefixes, dir, path) {
					return fs.SkipDir
				}
				return nil
			}
			base := filepath.Base(path)
			ext := filepath.Ext(base)
			if !strings.EqualFold(ext, ".vue") {
				return nil
			}
			name := strings.TrimSuffix(base, ext)
			return registerInto(name, path)
		})
	}
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		ext := filepath.Ext(base)
		if !strings.EqualFold(ext, ".vue") {
			return nil
		}
		name := strings.TrimSuffix(base, ext)
		return registerInto(name, path)
	})
}

// validateMountPrefixes checks that every Mount.Prefix in mounts is
// non-empty, contains no "/", and is unique across the slice, and that every
// Mount.FS is non-nil. It is called eagerly from New (not deferred to
// ValidateAll) per RFC 014 §4.2: two mounts sharing a prefix is a narrow,
// authoring-time mistake, not something that needs a full-registry-scan
// validation pass to catch. The nil-FS check exists because fs.Sub(nil, dir)
// does not itself return an error — it defers the failure to a later Open/
// Stat/ReadDir call, which panics on a nil fs.FS rather than returning a
// clean error; catching it here turns that into an ordinary New() error.
func validateMountPrefixes(mounts []Mount) error {
	seen := make(map[string]bool, len(mounts))
	for i, m := range mounts {
		if m.Prefix == "" {
			return fmt.Errorf("engine: Options.Mounts[%d]: Prefix must not be empty", i)
		}
		if strings.Contains(m.Prefix, "/") {
			return fmt.Errorf("engine: Options.Mounts[%d]: Prefix %q must not contain %q", i, m.Prefix, "/")
		}
		if seen[m.Prefix] {
			return fmt.Errorf("engine: Options.Mounts: duplicate Prefix %q", m.Prefix)
		}
		if m.FS == nil {
			return fmt.Errorf("engine: Options.Mounts[%d] (prefix %q): FS must not be nil", i, m.Prefix)
		}
		seen[m.Prefix] = true
	}
	return nil
}

// appendMountIDAttempt records that name was registered under mountID,
// appending (name, mountID) to ids[name] in call order and skipping the
// append if that exact pair was already recorded (so re-running discovery,
// or a name/lowercase-alias pair sharing the same underlying entry, does not
// produce duplicate bookkeeping).
func appendMountIDAttempt(ids map[string][]string, name, mountID string) {
	for _, existing := range ids[name] {
		if existing == mountID {
			return
		}
	}
	ids[name] = append(ids[name], mountID)
}

// recordPrimaryMountIDs records every name currently in entries as belonging
// to the primary source (mountID ""). It is called once, immediately after
// the primary discovery walk and before any mount is processed, so
// entryMountIDs preserves "primary before every mount" registration order
// regardless of Go's randomized map iteration order (names are sorted here
// purely for determinism between runs, not because order matters among
// primary entries themselves — they all share the same mountID).
func recordPrimaryMountIDs(entries map[string]*engineEntry, mountIDs map[string][]string) {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		appendMountIDAttempt(mountIDs, name, "")
	}
}

// isMountRootDir reports whether path is exactly the WalkDir path a
// registered mount's own top-level directory would have when the primary
// discovery walk is rooted at dir (i.e. path == pathpkg.Join(dir, prefix)
// for some prefix in prefixes). This is only ever true when dir is "."  (or
// otherwise resolves to the union's own root) and a same-named mount exists
// — for the common case of ComponentDir set to a real subdirectory (e.g.
// "templates"), no registered mount prefix can ever collide with a path the
// walk would report, so this is a no-op guard in that case.
func isMountRootDir(prefixes []string, dir, path string) bool {
	for _, p := range prefixes {
		if pathpkg.Join(dir, p) == path {
			return true
		}
	}
	return false
}

// mountRelDirFor returns the forward-slash proximity directory of path when
// path is owned by one of prefixes (path itself equals a prefix, or path has
// a prefix followed by "/"), or ("", false) if no prefix owns it.
//
// The returned directory is deliberately *not* stripped of the owning
// mount's own prefix — it is exactly pathpkg.Dir(path), the same "full,
// union-relative directory" shape that <component is="radix/dialog/Trigger">
// already splits into (dirPart="radix/dialog", localName="Trigger") via its
// own, unrelated string-splitting logic (renderer.go's "path-based
// reference" branch). Keeping both computations in agreement is what makes
// that explicit syntax find the same nsRegistry entry proximity resolution
// does; stripping the mount's own prefix here (i.e. computing "dialog"
// instead of "radix/dialog") would desynchronize the two and break explicit
// mount-qualified addressing. It also means a mount's own entries never
// share nsEntries[""] with the primary source's root-level entries, since
// every mount-owned path's directory always starts with that mount's own
// (non-empty) prefix.
// mountIDForPath returns the Mount.Prefix that owns path, or "" if path does
// not belong to any registered mount (the primary source owns it instead).
// Used by registerPathLocked (Register) to populate engineEntry.mountID and
// to pick the correct custom-element-tag stripping base for a path that
// arrives outside the normal discovery walk.
func mountIDForPath(prefixes []string, path string) string {
	for _, p := range prefixes {
		if p == "" {
			continue
		}
		if path == p || strings.HasPrefix(path, p+"/") {
			return p
		}
	}
	return ""
}

func mountRelDirFor(prefixes []string, path string) (string, bool) {
	if mountIDForPath(prefixes, path) == "" {
		return "", false
	}
	return nsRelDir(path, ""), true
}

// discoverMountInto walks mount's own subtree (reachable through the union
// e.opts.FS at mount.Prefix) and registers every *.vue file it contains into
// entries, nsEntries, and mountIDs — the bare component name, its automatic
// PascalCase/kebab-case mount aliases (RFC 014 §4.1 "Automatic mount
// aliases"), and their lowercase forms.
//
// Unlike discoverInto's registerInto (used only for the primary source),
// registration into the flat entries map here is insert-if-absent: an
// existing entry at the same flat name — whether a local/primary entry, an
// earlier-processed mount's bare name, or an earlier-processed alias — is
// never overwritten. This is a coarse, temporary form of "local wins"
// priority; real cross-mount collision detection is a later commit. Every
// attempted (name, mount.Prefix) pairing — for the bare name and both
// aliases alike — is nonetheless recorded into mountIDs, win or lose, since
// that is the data the collision-detection commit needs.
//
// Aliases are flat-registry-only (never written to nsEntries): nsEntries
// implements RFC 001 proximity resolution, which is meaningless for an
// alias — "RadixDialogTrigger" is not a real path segment anywhere in the
// union tree for proximity to walk up from, only a project-wide flat name,
// exactly like the explicit-path form's target `radix/dialog/Trigger` is
// keyed in nsEntries by its own directory rather than by the alias.
//
// Custom-element tag derivation uses mount.Prefix as the effective
// "ComponentDir", mirroring registerInto's own ComponentDir-relative
// derivation exactly — so a mounted "radix/dialog/Trigger.vue" derives the
// same tag a local "dialog/Trigger.vue" would relative to its own
// ComponentDir (i.e. mount.Prefix itself is stripped and never appears in
// the tag).
//
// nsEntries keying is intentionally *not* stripped of mount.Prefix — see
// mountRelDirFor's doc comment for why the union-relative directory (e.g.
// "radix/dialog") is the key that must be used, not the mount-relative one
// (e.g. "dialog"). nsEntries writes are insert-if-absent, protecting an
// existing primary (or earlier-mount) entry from ever being silently
// overwritten if Mount.Prefix happens to collide with a real top-level
// directory name already present under ComponentDir.
func (e *Engine) discoverMountInto(mount Mount, entries map[string]*engineEntry, nsEntries map[string]map[string]*engineEntry, mountIDs map[string][]string) error {
	prefix := mount.Prefix + "/"
	return fs.WalkDir(e.opts.FS, mount.Prefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := pathpkg.Base(path)
		ext := pathpkg.Ext(base)
		if !strings.EqualFold(ext, ".vue") {
			return nil
		}
		name := strings.TrimSuffix(base, ext)

		data, err := fs.ReadFile(e.opts.FS, path)
		if err != nil {
			return fmt.Errorf("engine: read %s: %w", path, err)
		}
		comp, err := ParseFile(path, string(data))
		if err != nil {
			return err
		}
		if comp.CustomElementTag != "" {
			relPath := path
			if strings.HasPrefix(path, prefix) {
				relPath = path[len(prefix):]
			}
			reviseCustomElementTagWarning(comp, path, deriveCustomElementTag(relPath))
		}

		var modTime time.Time
		if statFS, ok := e.opts.FS.(fs.StatFS); ok {
			if info, statErr := statFS.Stat(path); statErr == nil {
				modTime = info.ModTime()
			}
		}
		entry := &engineEntry{path: path, comp: comp, modTime: modTime, mountID: mount.Prefix}

		insertIfAbsent := func(key string) {
			if _, exists := entries[key]; !exists {
				entries[key] = entry
			}
			appendMountIDAttempt(mountIDs, key, mount.Prefix)
		}
		insertIfAbsent(name)
		if lower := strings.ToLower(name); lower != name {
			insertIfAbsent(lower)
		}

		// Automatic mount aliases (RFC 014 §4.1 "Automatic mount aliases"):
		// two additional flat-registry names derived from the *full*
		// union-relative path (mount.Prefix plus every intermediate
		// directory segment plus the base name), not just mount.Prefix +
		// name — so a nested "radix/dialog/Trigger.vue" aliases to
		// "RadixDialogTrigger"/"radix-dialog-trigger", not the shorter,
		// collision-prone "RadixTrigger"/"radix-trigger" (§10 Open Question
		// 10).
		//
		// deriveCustomElementTag is reused as-is (not reimplemented) and,
		// unlike the CustomElementTag re-derivation above, is deliberately
		// called against the *unstripped* path (including mount.Prefix)
		// since the alias is meant to carry the prefix, whereas the custom
		// element tag deliberately omits it.
		kebabAlias := deriveCustomElementTag(path)
		insertIfAbsent(kebabAlias)

		// kebabToPascal is reused as-is against the kebab alias itself, not
		// recomputed independently from segments, so both aliases are
		// derived from the exact same underlying value.
		pascalAlias := kebabToPascal(kebabAlias)
		insertIfAbsent(pascalAlias)
		// Mirror the existing local-component convention (registerInto,
		// above) of also registering an automatic all-lowercase fallback
		// for every PascalCase name whenever it differs from the name
		// itself — e.g. a template author who forgets the exact casing and
		// writes "<radixdialogtrigger>" still resolves. This keeps the
		// PascalCase alias consistent with how every other PascalCase name
		// in this registry already behaves, rather than carving out a
		// special case for mount aliases.
		if lower := strings.ToLower(pascalAlias); lower != pascalAlias {
			insertIfAbsent(lower)
		}

		// Deliberately unstripped — see mountRelDirFor's doc comment.
		relDir := nsRelDir(path, "")
		if nsEntries[relDir] == nil {
			nsEntries[relDir] = make(map[string]*engineEntry)
		}
		if _, exists := nsEntries[relDir][name]; !exists {
			nsEntries[relDir][name] = entry
		}

		return nil
	})
}

// registerPathLocked reads and parses the .vue file at path, then stores it
// under name. The caller must hold e.mu for writing.
func (e *Engine) registerPathLocked(name, path string) error {
	var (
		data []byte
		err  error
	)
	if e.opts.FS != nil {
		data, err = fs.ReadFile(e.opts.FS, path)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return fmt.Errorf("engine: read %s: %w", path, err)
	}
	comp, err := ParseFile(path, string(data))
	if err != nil {
		return err
	}
	// mountID identifies path as belonging to a registered Mount (by Prefix)
	// or, if empty, the primary source. This matters for a path registered
	// manually via Register — e.g. Register("X", "radix/Accordion.vue") —
	// which reaches this same code path as any other registration.
	mountID := mountIDForPath(e.mountPrefixes, path)

	// Re-derive the custom-element tag using the path relative to its
	// effective source root, so the directory name is not included in the
	// tag (e.g. "templates/Button.vue" with ComponentDir="templates" yields
	// "button", not "templates-button"). For a mount-owned path, that root
	// is the owning Mount.Prefix (mirroring discoverMountInto's own
	// derivation, e.g. "radix/Accordion.vue" yields "accordion", not
	// "radix-accordion") rather than ComponentDir, which would never strip
	// anything from a mount path (they never share ComponentDir's prefix).
	if comp.CustomElementTag != "" {
		base := e.opts.ComponentDir
		if mountID != "" {
			base = mountID
		}
		if base != "" {
			compSlash := filepath.ToSlash(filepath.Clean(path))
			dirSlash := filepath.ToSlash(filepath.Clean(base))
			relPath := compSlash
			if dirSlash != "" && dirSlash != "." {
				prefix := dirSlash + "/"
				if strings.HasPrefix(compSlash, prefix) {
					relPath = compSlash[len(prefix):]
				}
			}
			reviseCustomElementTagWarning(comp, path, deriveCustomElementTag(relPath))
		}
	}
	var modTime time.Time
	if e.opts.FS != nil {
		if statFS, ok := e.opts.FS.(fs.StatFS); ok {
			if info, statErr := statFS.Stat(path); statErr == nil {
				modTime = info.ModTime()
			}
		}
	} else {
		if info, statErr := os.Stat(path); statErr == nil {
			modTime = info.ModTime()
		}
	}
	entry := &engineEntry{path: path, comp: comp, modTime: modTime, mountID: mountID}
	e.entries[name] = entry
	if lower := strings.ToLower(name); lower != name {
		e.entries[lower] = entry
	}

	// Populate the namespaced registry when ComponentDir is set.
	if e.opts.ComponentDir != "" {
		relDir := e.relDirForPath(path)
		if e.nsEntries[relDir] == nil {
			e.nsEntries[relDir] = make(map[string]*engineEntry)
		}
		e.nsEntries[relDir][name] = entry
	}

	e.rebuildCollectorLocked()
	return nil
}

// registerPath reads and parses the .vue file at path, stores it under name,
// and is safe for concurrent use.
func (e *Engine) registerPath(name, path string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.registerPathLocked(name, path)
}

// Register manually adds a component from path to the engine's registry under
// name, without requiring a directory scan. This is useful when components are
// generated programmatically or loaded from locations outside ComponentDir.
func (e *Engine) Register(name, path string) error {
	return e.registerPath(name, path)
}

// maybeReload checks whether any discovered component file has changed and, if
// so, performs a full re-walk of ComponentDir, rebuilding both entries and
// nsEntries from scratch. This ensures nsEntries stays consistent with entries
// after any modification.
//
// It is safe for concurrent use.
func (e *Engine) maybeReload() error {
	if e.varReload.Value() == 0 {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	// Scan entries to find whether any file has been modified.
	anyChanged := false
	for _, entry := range e.entries {
		if e.opts.FS != nil {
			statFS, ok := e.opts.FS.(fs.StatFS)
			if !ok {
				// FS does not support Stat; skip reload entirely.
				return nil
			}
			info, err := statFS.Stat(entry.path)
			if err != nil {
				continue
			}
			if info.ModTime().After(entry.modTime) {
				anyChanged = true
				break
			}
		} else {
			info, err := os.Stat(entry.path)
			if err != nil {
				continue
			}
			if info.ModTime().After(entry.modTime) {
				anyChanged = true
				break
			}
		}
	}

	if !anyChanged {
		return nil
	}

	// Full re-walk: clear both registries and rebuild from ComponentDir, then
	// redo every Mount's own walk too (against the still-intact union
	// e.opts.FS) — otherwise a reload triggered by a primary-source file
	// change would silently drop every mount-sourced entry from the
	// registry, since the primary discover call below only ever repopulates
	// entries/nsEntries from ComponentDir.
	e.counterReloads.Add(1)
	e.entries = make(map[string]*engineEntry)
	e.nsEntries = make(map[string]map[string]*engineEntry)
	e.entryMountIDs = make(map[string][]string)
	if e.opts.ComponentDir != "" {
		if err := e.discover(e.opts.ComponentDir); err != nil {
			return err
		}
	}
	recordPrimaryMountIDs(e.entries, e.entryMountIDs)
	for _, m := range e.opts.Mounts {
		if err := e.discoverMountInto(m, e.entries, e.nsEntries, e.entryMountIDs); err != nil {
			return err
		}
	}
	e.rebuildCollectorLocked()
	return nil
}

// buildRegistryLocked returns a Registry snapshot of all current entries.
// The caller must hold at least e.mu.RLock().
func (e *Engine) buildRegistryLocked() Registry {
	reg := make(Registry, len(e.entries))
	for name, entry := range e.entries {
		if entry.comp != nil {
			reg[name] = entry.comp
		}
	}
	return reg
}

// buildNSRegistryLocked returns a snapshot of the namespaced component index.
// Keys are forward-slash relative directory paths (empty string for root).
// The caller must hold at least e.mu.RLock().
func (e *Engine) buildNSRegistryLocked() map[string]map[string]*Component {
	ns := make(map[string]map[string]*Component, len(e.nsEntries))
	for dir, dirEntries := range e.nsEntries {
		dirMap := make(map[string]*Component, len(dirEntries))
		for name, entry := range dirEntries {
			dirMap[name] = entry.comp
		}
		ns[dir] = dirMap
	}
	return ns
}

// relDirForPath returns the forward-slash directory of path relative to its
// effective source root: the longest registered Mount.Prefix that owns path,
// or opts.ComponentDir (the primary source) otherwise, or "" for root-level
// components. Used when populating nsEntries in registerPathLocked (so a
// manually Register()'d mount-addressed path, e.g. "radix/Accordion.vue",
// gets the same proximity namespace a discovered mount entry would) and by
// ValidateAll (so a mount component's own references are checked against its
// own mount's proximity namespace instead of always collapsing to "").
func (e *Engine) relDirForPath(path string) string {
	if dir, ok := mountRelDirFor(e.mountPrefixes, path); ok {
		return dir
	}
	if e.opts.ComponentDir == "" {
		return ""
	}
	return nsRelDir(path, e.opts.ComponentDir)
}

// nsRelDir computes the forward-slash relative directory of compPath with
// respect to componentDir. Returns "" for components at the root level.
// Works for both OS paths (converts separators) and FS forward-slash paths.
func nsRelDir(compPath, componentDir string) string {
	// Normalise both to forward slashes.
	compSlash := filepath.ToSlash(compPath)
	dirSlash := filepath.ToSlash(componentDir)

	// Build the prefix that represents componentDir + separator.
	var rel string
	if dirSlash == "" || dirSlash == "." {
		// componentDir is the working directory; path is already relative.
		rel = compSlash
	} else {
		prefix := dirSlash + "/"
		if !strings.HasPrefix(compSlash, prefix) {
			return ""
		}
		rel = compSlash[len(prefix):]
	}

	// rel is e.g. "blog/Card.vue" or "Card.vue".
	// Extract the directory component using the path package (forward slashes).
	d := pathpkg.Dir(rel)
	if d == "." {
		return ""
	}
	return d
}

// Components returns the names of all registered components in sorted order.
// Lowercase aliases added automatically by the engine are excluded.
func (e *Engine) Components() []string {
	e.mu.RLock()
	defer e.mu.RUnlock()
	seen := make(map[*engineEntry]bool, len(e.entries))
	names := make([]string, 0, len(e.entries))
	for name, entry := range e.entries {
		if seen[entry] {
			continue
		}
		seen[entry] = true
		names = append(names, name)
	}
	return names
}

// CollectCustomElements returns a [CustomElementCollector] populated with all
// custom element scripts registered in the engine, regardless of which pages
// reference them.
//
// It populates the collector without rendering any HTML, which makes it useful
// for pre-warming an import map or script bundle at startup time without
// needing a "warm render", and for testing that the expected scripts are
// registered.
//
// CollectCustomElements returns an error if the engine has no entries (i.e. no
// components are registered). In normal use the engine is always initialised by
// [New], so this error path is a safety guard for future use.
func (e *Engine) CollectCustomElements() (*CustomElementCollector, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if len(e.entries) == 0 {
		return nil, fmt.Errorf("htmlc: engine has no registered components")
	}
	collector := NewCustomElementCollector()
	seen := make(map[*engineEntry]bool, len(e.entries))
	for _, entry := range e.entries {
		if seen[entry] {
			continue
		}
		seen[entry] = true
		if entry.comp.CustomElementScript != "" {
			collector.Add(entry.comp.CustomElementTag, entry.comp.CustomElementScript)
		}
	}
	return collector, nil
}

// Has reports whether name is a registered component.
func (e *Engine) Has(name string) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, ok := e.entries[name]
	return ok
}

// ValidateAll checks every registered component for unresolvable child
// component references and returns a slice of ValidationError (one per
// problem). An empty slice means all components are valid.
//
// ValidateAll uses the same proximity-based resolution as the renderer: a
// reference that can be resolved via the proximity walk or the flat registry
// is considered valid. Only references that cannot be found by either
// mechanism are reported as errors.
//
// ValidateAll is intended to be called once at application startup to surface
// missing-component problems early ("fail fast").
func (e *Engine) ValidateAll() []ValidationError {
	e.mu.RLock()
	reg := e.buildRegistryLocked()
	nsReg := e.buildNSRegistryLocked()
	// Build a deduplicated list of component names (skip auto-lowercase aliases).
	seen := make(map[*engineEntry]bool, len(e.entries))
	type namedEntry struct {
		name  string
		entry *engineEntry
	}
	var entries []namedEntry
	for name, entry := range e.entries {
		if seen[entry] {
			continue
		}
		seen[entry] = true
		entries = append(entries, namedEntry{name, entry})
	}
	// entryMountIDs is only ever replaced wholesale (never mutated in place)
	// under e.mu.Lock() by New/SetComponentDir/SetFS/etc., so capturing the
	// map reference here, inside the same RLock section used for e.entries
	// above, is enough to read it safely after RUnlock below.
	mountIDs := e.entryMountIDs
	hasMounts := len(e.opts.Mounts) > 0
	e.mu.RUnlock()

	var errs []ValidationError
	for _, ne := range entries {
		for _, w := range ne.entry.comp.Warnings {
			errs = append(errs, ValidationError{
				Component: ne.name,
				Message:   w,
			})
		}
		refs := collectComponentRefs(ne.entry.comp)
		for _, ref := range refs {
			// Try proximity walk first, then fall back to flat registry.
			callerDir := e.relDirForPath(ne.entry.path)
			if resolveInNSRegistry(nsReg, callerDir, ref) == nil && resolveInRegistry(reg, ref) == nil {
				errs = append(errs, ValidationError{
					Component: ne.name,
					Message:   fmt.Sprintf("%s: unknown component %q referenced", ne.entry.path, ref),
				})
			}
		}
	}

	// Cross-mount collision detection (RFC 014 §4.2). Gated entirely behind
	// Options.Mounts being non-empty so single-source projects (the
	// overwhelming majority) pay zero cost and see zero behavior change.
	//
	// Deliberately blanket, not shadowing-aware: any flat name recorded in
	// entryMountIDs with two or more distinct mount identities is reported,
	// even when a local/primary entry currently shadows the mount-provided
	// one via proximity for every existing caller. The flat-registry name
	// itself remains ambiguous — a future component added anywhere outside
	// the shadowing directory's proximity reach would silently fall through
	// to whichever source lexical order visited last, with no error. See
	// RFC 014 §4.2 and §6 Example 3 ("A genuine collision, caught at
	// validation time and resolved via an alias").
	if hasMounts {
		errs = append(errs, checkMountCollisions(mountIDs)...)
	}

	return errs
}

// checkMountCollisions reports one ValidationError for every flat registry
// name in mountIDs that was attempted by two or more distinct mount
// identities (the empty string for the primary source, or a Mount.Prefix).
// This is always a hard, reported error — RFC 014 §4.2/§10 item 2 rules out
// a warning-only mode for cross-mount collisions.
//
// Names are processed in sorted order so results are deterministic
// regardless of map iteration order.
func checkMountCollisions(mountIDs map[string][]string) []ValidationError {
	names := make([]string, 0, len(mountIDs))
	for name, ids := range mountIDs {
		if len(ids) > 1 {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	errs := make([]ValidationError, 0, len(names))
	for _, name := range names {
		ids := mountIDs[name]
		labels := make([]string, len(ids))
		for i, id := range ids {
			if id == "" {
				labels[i] = "(primary)"
				continue
			}
			labels[i] = id
		}

		// The fix is described generically ("PrefixName"/"prefix-name",
		// "prefix/Name"), not reconstructed into a specific alias or path,
		// because entryMountIDs records only which mount identities
		// attempted a name, not the source path each one attempted it
		// from — and the colliding name itself may already *be* a derived
		// alias rather than a bare filename (e.g. two mounts whose
		// PascalCase aliases collide with each other, as in
		// TestMountAlias_CrossMountCollision), so there is no reliable way
		// to compute the one true qualified name from name+mountID alone.
		// The mount identities listed in the message are real and always
		// correct; the disambiguation syntax shown is illustrative.
		message := fmt.Sprintf(
			`component name %q is ambiguous across sources %v; use one of its qualified aliases (e.g. "PrefixName"/"prefix-name") or <component is="prefix/Name"> to disambiguate`,
			name, labels,
		)

		errs = append(errs, ValidationError{
			Component: name,
			Message:   message,
		})
	}
	return errs
}

// collectComponentRefs walks a component's template and returns all tag names
// that look like component references (PascalCase or kebab-case with hyphen).
func collectComponentRefs(comp *Component) []string {
	if comp == nil || comp.Template == nil {
		return nil
	}
	seen := make(map[string]bool)
	return walkTemplateForRefs(comp.Template, seen)
}

// walkTemplateForRefs recursively collects component-like tag names from the
// HTML node tree. A tag is considered a component reference if it is
// PascalCase (first letter uppercase) or kebab-case (contains a hyphen).
func walkTemplateForRefs(n *html.Node, seen map[string]bool) []string {
	var refs []string
	if n.Type == html.ElementNode {
		name := n.Data
		if isComponentLikeName(name) && !seen[name] {
			seen[name] = true
			refs = append(refs, name)
		}
	}
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		refs = append(refs, walkTemplateForRefs(child, seen)...)
	}
	return refs
}

// isComponentLikeName reports whether name looks like a component reference:
// either PascalCase (starts with uppercase) or kebab-case (contains a hyphen).
func isComponentLikeName(name string) bool {
	if len(name) == 0 {
		return false
	}
	if name[0] >= 'A' && name[0] <= 'Z' {
		return true
	}
	return strings.Contains(name, "-")
}

// resolveInRegistry tries to find a component by name using the same resolution
// logic as the renderer (exact, capitalized, kebab-to-pascal, case-insensitive).
func resolveInRegistry(reg Registry, name string) *Component {
	if c, ok := reg[name]; ok {
		return c
	}
	if len(name) > 0 {
		cap := strings.ToUpper(name[:1]) + name[1:]
		if cap != name {
			if c, ok := reg[cap]; ok {
				return c
			}
		}
	}
	if strings.Contains(name, "-") {
		pascal := kebabToPascal(name)
		if c, ok := reg[pascal]; ok {
			return c
		}
	}
	lower := strings.ToLower(name)
	for k, c := range reg {
		if strings.ToLower(k) == lower {
			return c
		}
	}
	return nil
}

// applyEngineScope merges the engine's registered functions into the render
// scope. Engine functions have lower priority than user-provided data: keys
// already present in scope are not overwritten.
func (e *Engine) applyEngineScope(data map[string]any) map[string]any {
	return e.applyScope(data, e.funcs)
}

// applyScope merges funcs into data, with data taking precedence over funcs.
// Returns data unchanged when funcs is empty.
func (e *Engine) applyScope(data map[string]any, funcs map[string]any) map[string]any {
	if len(funcs) == 0 {
		return data
	}
	merged := make(map[string]any, len(data)+len(funcs))
	for k, v := range funcs {
		merged[k] = v
	}
	// data overrides funcs
	for k, v := range data {
		merged[k] = v
	}
	return merged
}

// applyDataMiddleware applies registered middleware functions in order,
// augmenting the data map for an HTTP-triggered render.
func (e *Engine) applyDataMiddleware(r *http.Request, data map[string]any) map[string]any {
	for _, mw := range e.dataMiddleware {
		data = mw(r, data)
	}
	return data
}

// renderComponent renders the named component with the given data scope,
// writing HTML to w and returning the collected styles. It uses the engine's
// internal collector so importMap is available in all renders.
func (e *Engine) renderComponent(ctx context.Context, w io.Writer, name string, data map[string]any) (*StyleCollector, error) {
	sc, _, err := e.renderComponentWithCollector(ctx, w, name, data, e.collector)
	return sc, err
}

// renderComponentWithCollector renders the named component with the given data
// scope, writing HTML to w. It returns the collected styles and passes
// collector (may be nil) into every renderer so custom element scripts are
// accumulated. An importMap template function is injected into the render
// scope for every call; it closes over collector and returns the same value
// as collector.ImportMapJSON(urlPrefix). When collector is nil, importMap
// returns "". The function is propagated to all child components automatically.
func (e *Engine) renderComponentWithCollector(ctx context.Context, w io.Writer, name string, data map[string]any, collector *CustomElementCollector) (*StyleCollector, *CustomElementCollector, error) {
	e.counterRenders.Add(1)
	start := time.Now()
	defer func() { e.counterRenderNanos.Add(time.Since(start).Nanoseconds()) }()

	if err := e.maybeReload(); err != nil {
		e.counterRenderErrors.Add(1)
		return nil, nil, err
	}

	e.mu.RLock()
	entry, ok := e.entries[name]
	var reg Registry
	var nsReg map[string]map[string]*Component
	if ok {
		reg = e.buildRegistryLocked()
		if len(e.nsEntries) > 0 {
			nsReg = e.buildNSRegistryLocked()
		}
	}
	e.mu.RUnlock()

	if !ok {
		e.counterRenderErrors.Add(1)
		return nil, nil, fmt.Errorf("engine: unknown component %q: %w", name, ErrComponentNotFound)
	}

	// Build per-render funcs: engine-level funcs + importMap (depends on the per-call collector).
	renderFuncs := make(map[string]any, len(e.funcs)+1)
	for k, v := range e.funcs {
		renderFuncs[k] = v
	}
	renderFuncs["importMap"] = func(args ...any) (any, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("importMap: expected 1 argument, got %d", len(args))
		}
		urlPrefix, ok := args[0].(string)
		if !ok {
			return "", fmt.Errorf("importMap: argument must be a string, got %T", args[0])
		}
		if collector == nil {
			return "", nil
		}
		return collector.ImportMapJSON(urlPrefix), nil
	}

	sc := &StyleCollector{}
	renderer := NewRenderer(entry.comp).
		WithStyles(sc).
		WithCollector(collector).
		WithComponents(reg).
		WithDirectives(e.directives).
		WithFuncs(renderFuncs).
		WithContext(ctx)
	if nsReg != nil {
		renderer = renderer.WithNSComponents(nsReg, e.opts.ComponentDir)
	}
	if len(e.mountPrefixes) > 0 {
		renderer = renderer.WithMountPrefixes(e.mountPrefixes)
	}
	if len(e.opts.NativeElements) > 0 {
		renderer = renderer.WithNativeElements(e.opts.NativeElements)
	}

	scope := e.applyScope(data, renderFuncs)
	if e.missingPropHandler != nil {
		renderer = renderer.WithMissingPropHandler(e.missingPropHandler)
	}
	if e.varDebug.Value() != 0 {
		renderer.debug = true
	}
	if e.opts.Logger != nil {
		renderer = renderer.WithLogger(e.opts.Logger)
	}
	if e.opts.ComponentErrorHandler != nil {
		renderer = renderer.WithComponentErrorHandler(e.opts.ComponentErrorHandler)
	}
	renderer = renderer.WithComponentPath([]string{name})
	renderFn := func(out io.Writer) error {
		return renderer.Render(out, scope)
	}
	if err := e.loggedRender(ctx, name, w, renderFn); err != nil {
		e.counterRenderErrors.Add(1)
		return nil, nil, err
	}
	return sc, collector, nil
}

// RenderWithCollector renders the named component into w and records any custom
// element scripts in collector (may be nil for no collection).
//
// Most callers should use [Engine.RenderPage] or [Engine.RenderFragment]
// instead; those methods manage the collector lifecycle automatically.
// RenderWithCollector is intended for advanced use cases where the caller
// controls the collector lifecycle — for example, when rendering multiple
// fragments into a single response and accumulating scripts from all of them.
func (e *Engine) RenderWithCollector(ctx context.Context, w io.Writer, name string, props map[string]any, collector *CustomElementCollector) error {
	_, _, err := e.renderComponentWithCollector(ctx, w, name, props, collector)
	return err
}

// RenderWithCollectorString renders the named component and records any custom
// element scripts in collector (may be nil for no collection). It returns the
// rendered HTML as a string.
//
// This is a convenience wrapper around [Engine.RenderWithCollector] for callers
// that need the output as a string rather than streaming it to an [io.Writer].
func (e *Engine) RenderWithCollectorString(ctx context.Context, name string, props map[string]any, collector *CustomElementCollector) (string, error) {
	var buf strings.Builder
	if err := e.RenderWithCollector(ctx, &buf, name, props, collector); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// loggedRender wraps a root render call with slog instrumentation when
// opts.Logger is non-nil. name is the root component name; render is a
// closure that writes to the supplied io.Writer.
func (e *Engine) loggedRender(
	ctx context.Context,
	name string,
	w io.Writer,
	render func(io.Writer) error,
) error {
	if e.opts.Logger == nil {
		return render(w)
	}
	e.cw.Reset(w)
	start := time.Now()
	renderErr := render(&e.cw)
	elapsed := time.Since(start)
	if renderErr != nil {
		e.opts.Logger.ErrorContext(ctx, MsgComponentFailed,
			slog.String("component", name),
			slog.Duration("duration", elapsed),
			slog.Int64("bytes", e.cw.n),
			slog.Any("error", renderErr),
		)
	} else {
		e.opts.Logger.DebugContext(ctx, MsgComponentRendered,
			slog.String("component", name),
			slog.Duration("duration", elapsed),
			slog.Int64("bytes", e.cw.n),
		)
	}
	return renderErr
}

// styleBlock builds a "<style>…</style>" string from sc's contributions.
// Returns an empty string when there are no contributions.
func styleBlock(sc *StyleCollector) string {
	items := sc.All()
	if len(items) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<style>")
	for _, item := range items {
		// CSS is written verbatim — intentionally not HTML-escaped.
		// html.EscapeString must never be applied here: it would corrupt
		// quoted string values, & characters, and other valid CSS syntax.
		sb.WriteString(item.CSS)
	}
	sb.WriteString("</style>")
	return sb.String()
}

// RenderPage renders name as a full HTML page, writing the result to w. It
// collects all scoped styles from the component tree and inserts them as a
// <style> block immediately before the first </head> tag, keeping styles in
// the document head where browsers expect them. If the output contains no
// </head> the style block is prepended to the output instead.
//
// Use RenderPage when rendering a complete HTML document (e.g. a page
// component that includes <!DOCTYPE html>, <html>, <head>, and <body>).
// For partial HTML — such as HTMX responses or turbo-frame updates — use
// RenderFragment instead, which prepends styles without searching for </head>.
func (e *Engine) RenderPage(ctx context.Context, w io.Writer, pageName string, data map[string]any) error {
	var buf strings.Builder
	sc, _, err := e.renderComponentWithCollector(ctx, &buf, pageName, data, e.collector)
	if err != nil {
		return err
	}
	out := buf.String()
	style := styleBlock(sc)
	if style != "" {
		if idx := strings.Index(out, "</head>"); idx >= 0 {
			out = out[:idx] + style + out[idx:]
		} else {
			out = style + out
		}
	}
	if e.varDebug.Value() != 0 {
		script := "\n<script>\n" + InspectorScript + "\n</script>\n"
		if idx := strings.Index(out, "</body>"); idx >= 0 {
			out = out[:idx] + script + out[idx:]
		} else {
			out = out + script
		}
	}
	_, err = io.WriteString(w, out)
	return err
}

// RenderPageContext is like RenderPage but accepts a context.Context. The
// render is aborted and ctx.Err() is returned if the context is cancelled or
// its deadline is exceeded during rendering.
//
// Deprecated: Use RenderPage, which now accepts a context.Context directly.
func (e *Engine) RenderPageContext(ctx context.Context, w io.Writer, name string, data map[string]any) error {
	return e.RenderPage(ctx, w, name, data)
}

// RenderFragment renders name as an HTML fragment, writing the result to w,
// and prepends the collected <style> block to the output. Unlike RenderPage,
// it does not search for a </head> tag — it simply places the styles before
// the HTML, making it suitable for partial page updates (e.g. HTMX responses,
// turbo frames, or any context where a complete HTML document structure is not
// present).
//
// For full HTML documents that include a <head> section, use RenderPage
// instead so that styles are injected in the document head.
func (e *Engine) RenderFragment(w io.Writer, name string, data map[string]any) error {
	return e.RenderFragmentContext(context.Background(), w, name, data)
}

// RenderFragmentContext is like RenderFragment but accepts a context.Context.
// The render is aborted and ctx.Err() is returned if the context is cancelled
// or its deadline is exceeded during rendering.
func (e *Engine) RenderFragmentContext(ctx context.Context, w io.Writer, name string, data map[string]any) error {
	var buf strings.Builder
	sc, err := e.renderComponent(ctx, &buf, name, data)
	if err != nil {
		return err
	}
	style := styleBlock(sc)
	if style != "" {
		if _, err = io.WriteString(w, style); err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, buf.String())
	return err
}

// RenderFragmentString renders name as an HTML fragment and returns the result
// as a string. Scoped styles are prepended before the HTML. It uses the
// engine's internal collector so importMap is available in all renders.
func (e *Engine) RenderFragmentString(ctx context.Context, name string, data map[string]any) (string, error) {
	var buf strings.Builder
	sc, _, err := e.renderComponentWithCollector(ctx, &buf, name, data, e.collector)
	if err != nil {
		return "", err
	}
	var result strings.Builder
	if style := styleBlock(sc); style != "" {
		result.WriteString(style)
	}
	result.WriteString(buf.String())
	return result.String(), nil
}

// RenderPageString renders name as a full HTML page and returns the result as
// a string. It is a convenience wrapper around RenderPage for callers that
// need a string rather than writing to an io.Writer.
func (e *Engine) RenderPageString(name string, data map[string]any) (string, error) {
	var buf strings.Builder
	if err := e.RenderPage(context.Background(), &buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ScriptHandler returns an http.Handler that serves the engine's collected
// custom element scripts.
//
// Hashed .js files are served with Cache-Control: immutable so browsers and
// CDNs can cache them indefinitely. index.js (generated by
// [CustomElementCollector.IndexJS]) is served without a long-lived cache
// header so it is always fresh.
//
// Typical mount:
//
//	http.Handle("/scripts/", http.StripPrefix("/scripts/", engine.ScriptHandler()))
func (e *Engine) ScriptHandler() http.Handler {
	return NewScriptFSServer(e.collector)
}

// WriteScripts writes all files that would be served by [Engine.ScriptHandler]
// to dir. It creates dir if it does not exist. For each collected script a
// file named "<hash>.js" is written, and an "index.js" file is written that
// imports all scripts using relative imports.
//
// WriteScripts is a no-op when no custom element scripts have been collected.
// It is the static-build equivalent of ScriptHandler: use ScriptHandler for
// live HTTP serving and WriteScripts when generating a static site.
func (e *Engine) WriteScripts(dir string) error {
	coll := e.collector
	if coll.Len() == 0 {
		return nil
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("WriteScripts: create dir: %w", err)
	}
	fsys := coll.ScriptsFS()
	if err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, readErr := fs.ReadFile(fsys, path)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(filepath.Join(dir, path), data, 0644)
	}); err != nil {
		return err
	}
	indexContent := coll.IndexJS()
	if indexContent == "" {
		return nil
	}
	return os.WriteFile(filepath.Join(dir, "index.js"), []byte(indexContent), 0644)
}

// Collector returns the engine's internal [CustomElementCollector], populated
// with all custom element scripts registered in the engine. The returned
// collector is already populated after the first [Engine.Register] or
// [New] call and is safe for read access after engine construction.
//
// Use Collector to call ImportMapJSON directly or to integrate with an
// existing asset pipeline.
func (e *Engine) Collector() *CustomElementCollector {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.collector
}

// ServeComponent returns an http.HandlerFunc that renders name as a fragment
// and writes it with content-type "text/html; charset=utf-8". The data
// function is called on every request to obtain the data map passed to the
// template; it may be nil (in which case no data is provided).
//
// Data middleware registered via WithDataMiddleware is applied after the data
// function, allowing common data (e.g. the current user or CSRF token) to be
// injected globally.
func (e *Engine) ServeComponent(name string, data func(*http.Request) map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var scope map[string]any
		if data != nil {
			scope = data(r)
		}
		if scope == nil {
			scope = make(map[string]any)
		}
		scope = e.applyDataMiddleware(r, scope)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := e.RenderFragmentContext(r.Context(), w, name, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// ServePageComponent returns an http.HandlerFunc that renders name as a full
// HTML page (using RenderPage, which injects styles into </head>) and writes
// it with content-type "text/html; charset=utf-8".
//
// The data function is called on every request to obtain the data map and the
// HTTP status code to send. If the data function is nil, a 200 OK response
// with no template data is used. A status code of 0 is treated as 200.
//
// Data middleware registered via WithDataMiddleware is applied after the data
// function.
func (e *Engine) ServePageComponent(name string, data func(*http.Request) (map[string]any, int)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var scope map[string]any
		statusCode := http.StatusOK
		if data != nil {
			scope, statusCode = data(r)
			if statusCode == 0 {
				statusCode = http.StatusOK
			}
		}
		if scope == nil {
			scope = make(map[string]any)
		}
		scope = e.applyDataMiddleware(r, scope)

		// Buffer output so we can set the status code before writing the body.
		var buf strings.Builder
		if err := e.RenderPage(r.Context(), &buf, name, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(statusCode)
		io.WriteString(w, buf.String())
	}
}

// Mount registers a set of component routes on mux. The routes map keys are
// patterns accepted by http.ServeMux (e.g. "GET /{$}", "GET /about"), and
// values are component names. Each component is served as a full HTML page via
// ServePageComponent with no data function (use WithDataMiddleware to inject
// common data, or register routes manually for per-route data).
func (e *Engine) Mount(mux *http.ServeMux, routes map[string]string) {
	for pattern, name := range routes {
		mux.HandleFunc(pattern, e.ServePageComponent(name, nil))
	}
}

// TemplateText returns the raw html/template-compatible text for componentName
// and all its statically-referenced sub-components.  The text consists of
// {{ define }} blocks suitable for html/template.New("").Parse(text).
//
// This is the text form of CompileToTemplate; see that method for full
// semantics.  warnings contains any non-fatal conversion warnings emitted by
// the html/template conversion (for example, data-contract notices for v-html and v-bind spread).
//
// Error types follow the same conventions as CompileToTemplate: ErrComponentNotFound
// when componentName is not registered, and *ConversionError (wrapped with
// ErrConversion) when a directive or expression cannot be converted.
func (e *Engine) TemplateText(componentName string) (text string, warnings []string, err error) {
	e.mu.RLock()
	reg := e.buildRegistryLocked()
	e.mu.RUnlock()

	root := resolveInRegistry(reg, componentName)
	if root == nil {
		return "", nil, fmt.Errorf("engine: unknown component %q: %w", componentName, ErrComponentNotFound)
	}

	// DFS to collect all transitively-referenced sub-components in dependency
	// order (leaves first, root last).  visited is keyed on the lowercase tag
	// name as it appears in the HTML tree (already lowercased by the parser).
	type compEntry struct {
		lowerName string
		comp      *Component
	}
	visited := make(map[string]bool)
	var order []compEntry

	var dfs func(lowerName string, comp *Component) error
	dfs = func(lowerName string, comp *Component) error {
		if visited[lowerName] {
			return nil
		}
		visited[lowerName] = true

		// Walk the HTML node tree and recurse into any element whose name
		// resolves to a registered component (same strategy as the renderer).
		if comp.Template != nil {
			refSeen := make(map[string]bool)
			var walkNode func(*html.Node) error
			walkNode = func(n *html.Node) error {
				if n.Type == html.ElementNode {
					if subComp := resolveInRegistry(reg, n.Data); subComp != nil {
						// n.Data is already lowercased by the HTML parser.
						refName := n.Data
						if !refSeen[refName] {
							refSeen[refName] = true
							if err := dfs(refName, subComp); err != nil {
								return err
							}
						}
					}
				}
				for child := n.FirstChild; child != nil; child = child.NextSibling {
					if err := walkNode(child); err != nil {
						return err
					}
				}
				return nil
			}
			if err := walkNode(comp.Template); err != nil {
				return err
			}
		}

		order = append(order, compEntry{lowerName, comp})
		return nil
	}

	rootLower := strings.ToLower(componentName)
	if err = dfs(rootLower, root); err != nil {
		return "", nil, err
	}

	// Convert each component to a {{define}} block and concatenate them.
	var combined strings.Builder
	for _, ce := range order {
		result, convErr := VueToTemplate(ce.comp.Template, ce.lowerName)
		if convErr != nil {
			return "", nil, fmt.Errorf("%w: %w", ErrConversion, convErr)
		}
		warnings = append(warnings, result.Warnings...)
		combined.WriteString(result.Text)
		combined.WriteString("\n")
	}

	return combined.String(), warnings, nil
}

// CompileToTemplate compiles the named component (and all components it
// statically references) into a single *html/template.Template.
//
// The root component becomes the primary named template; all sub-components are
// added as named {{ define }} blocks in the same template set.  Template names
// follow Go convention: the component name is lowercased (e.g. "Card" →
// "card").
//
// Scoped <style> blocks are stripped from the output.  Non-recoverable
// conversion errors (unsupported directives, complex expressions) are returned
// as *ConversionError with source location information, wrapped together
// with ErrConversion so callers can test with either errors.Is or errors.As.
//
// The returned *html/template.Template is safe to call with Execute or
// ExecuteTemplate for any data value compatible with the component's props.
func (e *Engine) CompileToTemplate(componentName string) (*htmltmpl.Template, error) {
	text, _, err := e.TemplateText(componentName)
	if err != nil {
		return nil, err
	}
	rootLower := strings.ToLower(componentName)
	tmpl, err := htmltmpl.New(rootLower).Parse(text)
	if err != nil {
		return nil, err
	}
	return tmpl, nil
}

// RegisterTemplate registers an existing *html/template.Template as a virtual
// htmlc component under name.  The template is converted to htmlc's internal
// representation using the tmpl→vue conversion.
//
// All named {{ define }} blocks within tmpl are also registered as components
// accessible by their block names.
//
// If conversion fails (unsupported template constructs), RegisterTemplate
// returns a *ConversionError wrapped with ErrConversion and does not
// register anything.
//
// RegisterTemplate validates the template at registration time (fail-fast
// behaviour); it does not defer validation to render time.
//
// When called with a name already in use, the new registration wins
// ("last write wins"), consistent with flat-registry behaviour.
func (e *Engine) RegisterTemplate(name string, tmpl *htmltmpl.Template) error {
	type parsed struct {
		name string
		comp *Component
	}

	// Convert all templates first; if any fails, return the error without
	// side effects (nothing is registered).
	var results []parsed
	for _, t := range tmpl.Templates() {
		if t.Tree == nil || t.Tree.Root == nil {
			continue
		}
		src := t.Tree.Root.String()
		tname := t.Name()
		if tname == tmpl.Name() {
			// Map the root template to the caller-provided name.
			tname = name
		}
		result, err := TemplateToVue(src, tname)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrConversion, err)
		}
		comp, err := ParseFile("", result.Text)
		if err != nil {
			return err
		}
		results = append(results, parsed{tname, comp})
	}

	// All conversions succeeded; register under the write lock.
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, r := range results {
		entry := &engineEntry{comp: r.comp}
		e.entries[r.name] = entry
		if lower := strings.ToLower(r.name); lower != r.name {
			e.entries[lower] = entry
		}
	}
	return nil
}

// PublishExpvars registers the engine's configuration and performance counters
// in the global expvar registry under the given prefix. The prefix must be
// unique across all engines in the process; calling PublishExpvars with a
// prefix that is already registered panics (same as expvar.NewMap).
//
// After calling PublishExpvars, the engine's vars are accessible at
// /debug/vars under the key prefix, and the following sub-keys are available:
//
//	reload        – 1 if hot-reload is enabled, 0 otherwise
//	debug         – 1 if debug mode is enabled, 0 otherwise
//	componentDir  – the current component directory
//	fs            – the type name of the current fs.FS, or "<nil>"
//	renders       – total number of renderComponent calls
//	renderErrors  – total number of failed renders
//	reloads       – total number of hot-reload re-scans performed
//	renderNanos   – cumulative render time in nanoseconds
//	components    – number of unique registered components
//	info.directives – sorted list of registered custom directive names
//
// PublishExpvars returns the Engine so calls can be chained.
func (e *Engine) PublishExpvars(prefix string) *Engine {
	m := expvar.NewMap(prefix)
	m.Set("reload", e.varReload)
	m.Set("debug", e.varDebug)
	m.Set("componentDir", e.varComponentDir)
	m.Set("fs", e.varFS)
	m.Set("renders", e.counterRenders)
	m.Set("renderErrors", e.counterRenderErrors)
	m.Set("reloads", e.counterReloads)
	m.Set("renderNanos", e.counterRenderNanos)
	m.Set("components", e.counterComponents)
	info := new(expvar.Map)
	info.Set("directives", e.varDirectives)
	m.Set("info", info)
	e.expvarMap = m
	e.expvarPrefix = prefix
	return e
}

// SetReload enables or disables hot-reload at runtime. When enabled, the
// engine checks component file modification times before each render and
// automatically re-parses changed files. The change takes effect on the next
// render call.
func (e *Engine) SetReload(enabled bool) {
	if enabled {
		e.varReload.Set(1)
	} else {
		e.varReload.Set(0)
	}
}

// SetDebug enables or disables debug render mode at runtime. When enabled,
// the root element of each rendered component carries data-htmlc-* attributes
// for component name, source file, and serialised props. See Options.Debug for
// the full description.
func (e *Engine) SetDebug(enabled bool) {
	if enabled {
		e.varDebug.Set(1)
	} else {
		e.varDebug.Set(0)
	}
}

// SetComponentDir changes the component directory at runtime, re-running
// discovery atomically under the engine's write lock. If discovery fails,
// the engine's state is unchanged and the error is returned.
//
// Any Options.Mounts configured at New time are preserved: every Mount is
// re-walked (against the unchanged union e.opts.FS) after the primary
// ComponentDir walk, so mount-sourced entries are not lost by this call.
func (e *Engine) SetComponentDir(dir string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entries := make(map[string]*engineEntry)
	nsEntries := make(map[string]map[string]*engineEntry)
	if err := e.discoverInto(dir, entries, nsEntries); err != nil {
		return err
	}
	mountIDs := make(map[string][]string)
	recordPrimaryMountIDs(entries, mountIDs)
	for _, m := range e.opts.Mounts {
		if err := e.discoverMountInto(m, entries, nsEntries, mountIDs); err != nil {
			return err
		}
	}
	e.entries = entries
	e.nsEntries = nsEntries
	e.entryMountIDs = mountIDs
	e.opts.ComponentDir = dir
	e.varComponentDir.Set(dir)
	return nil
}

// SetFS changes the fs.FS used for component discovery and file reads at
// runtime, re-running discovery atomically under the engine's write lock.
// If discovery fails, the engine's state is unchanged and the error is
// returned.
//
// SetFS is not Mounts-aware: when Options.Mounts was configured at New time,
// e.opts.FS is the internal union fs.FS wrapping the primary source and
// every mount (see unionfs.go); calling SetFS replaces that union outright
// with fsys, so every mount becomes unreachable until the engine is
// reconstructed with New. Rebuilding the union in place is not implemented
// in this release.
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
