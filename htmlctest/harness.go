package htmlctest

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dhamidi/htmlc"
)

// Harness holds a test Engine backed by an in-memory filesystem.
// Create one with [NewHarness] or [Build].
type Harness struct {
	t        testing.TB
	eng      *htmlc.Engine
	mapFS    fstest.MapFS
	baseOpts htmlc.Options
}

// NewHarness creates a Harness from a file map and optional Options.
// The keys of files are file paths (e.g. "Button.vue") and the values are
// their contents. If opts is provided its first element is used as the base
// Options; FS and ComponentDir are always overridden by the in-memory FS.
func NewHarness(t testing.TB, files map[string]string, opts ...htmlc.Options) *Harness {
	t.Helper()
	mapFS := make(fstest.MapFS, len(files))
	for name, content := range files {
		mapFS[name] = &fstest.MapFile{Data: []byte(content)}
	}
	var baseOpts htmlc.Options
	if len(opts) > 0 {
		baseOpts = opts[0]
	}
	h := &Harness{t: t, mapFS: mapFS, baseOpts: baseOpts}
	h.rebuildEngine()
	return h
}

// Build wraps template in <template>…</template> if it does not already begin
// with that tag, registers the result as "Root.vue", and returns a *Harness.
// The inferred component name is "Root".
func Build(t testing.TB, template string) *Harness {
	t.Helper()
	src := template
	if !strings.HasPrefix(strings.TrimSpace(src), "<template>") {
		src = "<template>" + src + "</template>"
	}
	return NewHarness(t, map[string]string{"Root.vue": src})
}

// rebuildEngine creates a fresh Engine from the current mapFS and baseOpts.
func (h *Harness) rebuildEngine() {
	h.t.Helper()
	o := h.baseOpts
	o.FS = h.mapFS
	o.ComponentDir = "."
	e, err := htmlc.New(o)
	if err != nil {
		h.t.Fatalf("htmlctest: failed to build engine: %v", err)
	}
	h.eng = e
}

// With adds or replaces a component file and rebuilds the engine.
// Returns *Harness for call chaining.
func (h *Harness) With(filename, src string) *Harness {
	h.t.Helper()
	h.mapFS[filename] = &fstest.MapFile{Data: []byte(src)}
	h.rebuildEngine()
	return h
}

// Engine returns the underlying *htmlc.Engine.
func (h *Harness) Engine() *htmlc.Engine {
	return h.eng
}

// Page renders name as a full HTML page using [htmlc.Engine.RenderPageString].
func (h *Harness) Page(name string, data map[string]any) *Result {
	h.t.Helper()
	out, err := h.eng.RenderPageString(name, data)
	if err != nil {
		h.t.Fatalf("htmlctest: Page(%q): %v", name, err)
	}
	return &Result{t: h.t, html: out}
}

// Fragment renders name as an HTML fragment using [htmlc.Engine.RenderFragmentString].
func (h *Harness) Fragment(name string, data map[string]any) *Result {
	h.t.Helper()
	out, err := h.eng.RenderFragmentString(name, data)
	if err != nil {
		h.t.Fatalf("htmlctest: Fragment(%q): %v", name, err)
	}
	return &Result{t: h.t, html: out}
}

// ByTag delegates to the package-level [ByTag] constructor.
func (h *Harness) ByTag(name string) Query { return ByTag(name) }

// ByClass delegates to the package-level [ByClass] constructor.
func (h *Harness) ByClass(class string) Query { return ByClass(class) }

// ByAttr delegates to the package-level [ByAttr] constructor.
func (h *Harness) ByAttr(attr, value string) Query { return ByAttr(attr, value) }
