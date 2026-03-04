package htmlc

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Options holds configuration for creating a new Engine.
type Options struct {
	// ComponentDir is the directory to scan recursively for *.vue files.
	// Components are discovered by walking the tree in lexical order; each file
	// is registered by its base name without extension (e.g. "Button.vue"
	// becomes "Button"). When two files share the same base name the last one
	// encountered in lexical-order traversal wins.
	ComponentDir string
	// Reload enables hot-reload for development use. When true, the engine
	// checks the modification time of every registered component file before
	// each render and automatically re-parses any file that has changed since
	// it was last loaded. This lets you edit .vue files and see the results
	// without restarting the server. For production, leave Reload false and
	// create the Engine once at startup.
	Reload bool
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
type Engine struct {
	opts               Options
	entries            map[string]*engineEntry
	missingPropHandler MissingPropFunc
}

// WithMissingPropHandler sets the function called when any component rendered
// by this engine has a missing prop. If not set, missing props cause render errors.
func (e *Engine) WithMissingPropHandler(fn MissingPropFunc) *Engine {
	e.missingPropHandler = fn
	return e
}

// New creates an Engine configured by opts. If opts.ComponentDir is set the
// directory is walked recursively and all *.vue files are registered.
func New(opts Options) (*Engine, error) {
	e := &Engine{
		opts:    opts,
		entries: make(map[string]*engineEntry),
	}
	if opts.ComponentDir != "" {
		if err := e.discover(opts.ComponentDir); err != nil {
			return nil, err
		}
	}
	return e, nil
}

// discover walks dir in lexical order and registers every *.vue file found.
func (e *Engine) discover(dir string) error {
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
		return e.registerPath(name, path)
	})
}

// registerPath reads and parses the .vue file at path, then stores it under name.
func (e *Engine) registerPath(name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("engine: read %s: %w", path, err)
	}
	comp, err := ParseFile(path, string(data))
	if err != nil {
		return err
	}
	var modTime time.Time
	if info, statErr := os.Stat(path); statErr == nil {
		modTime = info.ModTime()
	}
	entry := &engineEntry{path: path, comp: comp, modTime: modTime}
	e.entries[name] = entry
	if lower := strings.ToLower(name); lower != name {
		e.entries[lower] = entry
	}
	return nil
}

// Register manually adds a component from path to the engine's registry under
// name, without requiring a directory scan. This is useful when components are
// generated programmatically or loaded from locations outside ComponentDir.
func (e *Engine) Register(name, path string) error {
	return e.registerPath(name, path)
}

// maybeReload re-parses any entry whose file mtime has advanced, when Reload
// is enabled.
func (e *Engine) maybeReload() error {
	if !e.opts.Reload {
		return nil
	}
	// Snapshot names to avoid modifying the map while iterating it.
	names := make([]string, 0, len(e.entries))
	for name := range e.entries {
		names = append(names, name)
	}
	for _, name := range names {
		entry := e.entries[name]
		info, err := os.Stat(entry.path)
		if err != nil {
			continue
		}
		if info.ModTime().After(entry.modTime) {
			if rerr := e.registerPath(name, entry.path); rerr != nil {
				return rerr
			}
		}
	}
	return nil
}

// buildRegistry returns a Registry snapshot of all current entries.
func (e *Engine) buildRegistry() Registry {
	reg := make(Registry, len(e.entries))
	for name, entry := range e.entries {
		reg[name] = entry.comp
	}
	return reg
}

// renderComponent renders the named component with the given data scope,
// returning the HTML output and collected styles.
func (e *Engine) renderComponent(name string, data map[string]any) (string, *StyleCollector, error) {
	if err := e.maybeReload(); err != nil {
		return "", nil, err
	}
	entry, ok := e.entries[name]
	if !ok {
		return "", nil, fmt.Errorf("engine: unknown component %q", name)
	}
	sc := &StyleCollector{}
	renderer := NewRenderer(entry.comp).
		WithStyles(sc).
		WithComponents(e.buildRegistry())
	if e.missingPropHandler != nil {
		renderer = renderer.WithMissingPropHandler(e.missingPropHandler)
	}
	out, err := renderer.Render(data)
	if err != nil {
		return "", nil, err
	}
	return out, sc, nil
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
		sb.WriteString(item.CSS)
	}
	sb.WriteString("</style>")
	return sb.String()
}

// RenderPage renders name as a full HTML page. It collects all scoped styles
// from the component tree and inserts them as a <style> block immediately
// before the first </head> tag, keeping styles in the document head where
// browsers expect them. If the output contains no </head> the style block is
// prepended to the output instead.
//
// Use RenderPage when rendering a complete HTML document (e.g. a page
// component that includes <!DOCTYPE html>, <html>, <head>, and <body>).
// For partial HTML — such as HTMX responses or turbo-frame updates — use
// RenderFragment instead, which prepends styles without searching for </head>.
func (e *Engine) RenderPage(name string, data map[string]any) (string, error) {
	out, sc, err := e.renderComponent(name, data)
	if err != nil {
		return "", err
	}
	style := styleBlock(sc)
	if style == "" {
		return out, nil
	}
	if idx := strings.Index(out, "</head>"); idx >= 0 {
		return out[:idx] + style + out[idx:], nil
	}
	return style + out, nil
}

// RenderFragment renders name as an HTML fragment and prepends the collected
// <style> block to the output. Unlike RenderPage, it does not search for a
// </head> tag — it simply places the styles before the HTML, making it
// suitable for partial page updates (e.g. HTMX responses, turbo frames, or
// any context where a complete HTML document structure is not present).
//
// For full HTML documents that include a <head> section, use RenderPage
// instead so that styles are injected in the document head.
func (e *Engine) RenderFragment(name string, data map[string]any) (string, error) {
	out, sc, err := e.renderComponent(name, data)
	if err != nil {
		return "", err
	}
	style := styleBlock(sc)
	if style == "" {
		return out, nil
	}
	return style + out, nil
}

// ServeComponent returns an http.HandlerFunc that renders name as a fragment
// and writes it with content-type "text/html; charset=utf-8". The data
// function is called on every request to obtain the data map passed to the
// template; it may be nil (in which case no data is provided).
func (e *Engine) ServeComponent(name string, data func(*http.Request) map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var scope map[string]any
		if data != nil {
			scope = data(r)
		}
		out, err := e.RenderFragment(name, scope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, out)
	}
}
