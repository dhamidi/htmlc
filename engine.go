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
}

// engineEntry holds a parsed component together with its source path and the
// mtime at which it was last parsed.
type engineEntry struct {
	path    string
	comp    *Component
	modTime time.Time
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
	mu                 sync.RWMutex // guards entries and nsEntries
	entries            map[string]*engineEntry
	nsEntries          map[string]map[string]*engineEntry // relDir → localName → entry
	missingPropHandler MissingPropFunc
	directives         DirectiveRegistry
	funcs              map[string]any // per-engine functions, injected into every render scope
	dataMiddleware     []func(*http.Request, map[string]any) map[string]any

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
	e := &Engine{
		opts:       opts,
		entries:    make(map[string]*engineEntry),
		nsEntries:  make(map[string]map[string]*engineEntry),
		directives: opts.Directives,
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

	if opts.ComponentDir != "" {
		if err := e.discover(opts.ComponentDir); err != nil {
			return nil, err
		}
	}
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

	// Full re-walk: clear both registries and rebuild from ComponentDir.
	e.counterReloads.Add(1)
	e.entries = make(map[string]*engineEntry)
	e.nsEntries = make(map[string]map[string]*engineEntry)
	if e.opts.ComponentDir != "" {
		return e.discover(e.opts.ComponentDir)
	}
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

// relDirForPath returns the forward-slash directory of the given path relative
// to opts.ComponentDir, or "" for root-level components. Used when populating
// nsEntries in registerPathLocked.
func (e *Engine) relDirForPath(path string) string {
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
	if len(e.funcs) == 0 {
		return data
	}
	merged := make(map[string]any, len(data)+len(e.funcs))
	for k, v := range e.funcs {
		merged[k] = v
	}
	// data overrides engine funcs
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
// writing HTML to w and returning the collected styles.
func (e *Engine) renderComponent(ctx context.Context, w io.Writer, name string, data map[string]any) (*StyleCollector, error) {
	sc, _, err := e.renderComponentWithCollector(ctx, w, name, data, nil)
	return sc, err
}

// renderComponentWithCollector renders the named component with the given data
// scope, writing HTML to w. It returns the collected styles and passes
// collector (may be nil) into every renderer so custom element scripts are
// accumulated.
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

	sc := &StyleCollector{}
	renderer := NewRenderer(entry.comp).
		WithStyles(sc).
		WithCollector(collector).
		WithComponents(reg).
		WithDirectives(e.directives).
		WithFuncs(e.funcs).
		WithContext(ctx)
	if nsReg != nil {
		renderer = renderer.WithNSComponents(nsReg, e.opts.ComponentDir)
	}

	scope := e.applyEngineScope(data)
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

// RenderWithCollector renders the named component and records any custom
// element scripts in collector (may be nil for no collection).
// It returns the rendered HTML as a string.
func (e *Engine) RenderWithCollector(ctx context.Context, name string, props map[string]any, collector *CustomElementCollector) (string, error) {
	var buf strings.Builder
	_, _, err := e.renderComponentWithCollector(ctx, &buf, name, props, collector)
	if err != nil {
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
func (e *Engine) RenderPage(w io.Writer, name string, data map[string]any) error {
	return e.RenderPageContext(context.Background(), w, name, data)
}

// RenderPageContext is like RenderPage but accepts a context.Context. The
// render is aborted and ctx.Err() is returned if the context is cancelled or
// its deadline is exceeded during rendering.
func (e *Engine) RenderPageContext(ctx context.Context, w io.Writer, name string, data map[string]any) error {
	var buf strings.Builder
	sc, err := e.renderComponent(ctx, &buf, name, data)
	if err != nil {
		return err
	}
	out := buf.String()

	// Inject scoped styles before </head>.
	style := styleBlock(sc)
	if style != "" {
		if idx := strings.Index(out, "</head>"); idx >= 0 {
			out = out[:idx] + style + out[idx:]
		} else {
			out = style + out
		}
	}

	// Inject inspector script before </body> when debug mode is active.
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

// RenderPageString renders name as a full HTML page and returns the result as
// a string. It is a convenience wrapper around RenderPage for callers that
// need a string rather than writing to an io.Writer.
func (e *Engine) RenderPageString(name string, data map[string]any) (string, error) {
	var buf strings.Builder
	if err := e.RenderPage(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderFragmentString renders name as an HTML fragment and returns the result
// as a string. It is a convenience wrapper around RenderFragment for callers
// that need a string rather than writing to an io.Writer.
func (e *Engine) RenderFragmentString(name string, data map[string]any) (string, error) {
	var buf strings.Builder
	if err := e.RenderFragment(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
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
		if err := e.RenderPageContext(r.Context(), &buf, name, scope); err != nil {
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
func (e *Engine) SetComponentDir(dir string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	entries := make(map[string]*engineEntry)
	nsEntries := make(map[string]map[string]*engineEntry)
	if err := e.discoverInto(dir, entries, nsEntries); err != nil {
		return err
	}
	e.entries = entries
	e.nsEntries = nsEntries
	e.opts.ComponentDir = dir
	e.varComponentDir.Set(dir)
	return nil
}

// SetFS changes the fs.FS used for component discovery and file reads at
// runtime, re-running discovery atomically under the engine's write lock.
// If discovery fails, the engine's state is unchanged and the error is
// returned.
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
