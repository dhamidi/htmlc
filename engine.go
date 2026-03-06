package htmlc

import (
	"fmt"
	"io"
	"io/fs"
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
			return e.registerPath(name, path)
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
		return e.registerPath(name, path)
	})
}

// registerPath reads and parses the .vue file at path, then stores it under name.
func (e *Engine) registerPath(name, path string) error {
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
		if e.opts.FS != nil {
			statFS, ok := e.opts.FS.(fs.StatFS)
			if !ok {
				// FS does not support Stat; skip reload for this entry.
				continue
			}
			info, err := statFS.Stat(entry.path)
			if err != nil {
				continue
			}
			if info.ModTime().After(entry.modTime) {
				if rerr := e.registerPath(name, entry.path); rerr != nil {
					return rerr
				}
			}
		} else {
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
// writing HTML to w and returning the collected styles.
func (e *Engine) renderComponent(w io.Writer, name string, data map[string]any) (*StyleCollector, error) {
	if err := e.maybeReload(); err != nil {
		return nil, err
	}
	entry, ok := e.entries[name]
	if !ok {
		return nil, fmt.Errorf("engine: unknown component %q", name)
	}
	sc := &StyleCollector{}
	renderer := NewRenderer(entry.comp).
		WithStyles(sc).
		WithComponents(e.buildRegistry())
	if e.missingPropHandler != nil {
		renderer = renderer.WithMissingPropHandler(e.missingPropHandler)
	}
	if err := renderer.Render(w, data); err != nil {
		return nil, err
	}
	return sc, nil
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
	var buf strings.Builder
	sc, err := e.renderComponent(&buf, name, data)
	if err != nil {
		return err
	}
	out := buf.String()
	style := styleBlock(sc)
	if style == "" {
		_, err = io.WriteString(w, out)
		return err
	}
	if idx := strings.Index(out, "</head>"); idx >= 0 {
		if _, err = io.WriteString(w, out[:idx]); err != nil {
			return err
		}
		if _, err = io.WriteString(w, style); err != nil {
			return err
		}
		_, err = io.WriteString(w, out[idx:])
		return err
	}
	if _, err = io.WriteString(w, style); err != nil {
		return err
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
	var buf strings.Builder
	sc, err := e.renderComponent(&buf, name, data)
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
func (e *Engine) ServeComponent(name string, data func(*http.Request) map[string]any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var scope map[string]any
		if data != nil {
			scope = data(r)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := e.RenderFragment(w, name, scope); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}
