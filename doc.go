// Package htmlc is a server-side Vue-style component engine for Go. It parses
// .vue Single File Components — each containing a <template>, an optional
// <script>, and an optional <style> section — and renders them to HTML strings
// ready to serve via net/http or any http.Handler-based framework.
//
// # Problem it solves
//
// Writing server-rendered HTML in Go typically means either concatenating
// strings, using html/template (which has no component model), or pulling in a
// full JavaScript runtime. htmlc gives you Vue's familiar component authoring
// format — scoped styles, template directives, and component composition —
// while keeping rendering purely in Go with no JavaScript dependency.
//
// # Mental model
//
// There are four main concepts:
//
//   - Engine          – the high-level entry point. It owns a Registry of
//                       parsed components discovered from a directory tree.
//                       Create one with New; call RenderPage or RenderFragment
//                       to produce HTML output. ServeComponent wraps a
//                       component as an http.Handler for use with net/http.
//
//   - Component       – the parsed representation of one .vue file, produced
//                       by ParseFile. Holds the template node tree, script
//                       text, style text, and scoped-style metadata.
//
//   - Renderer        – the low-level walker that evaluates a Component's
//                       template against a data scope and writes HTML. Most
//                       callers should use Engine instead.
//
//   - StyleCollector  – accumulates scoped-style contributions from all
//                       components rendered in one request so they can be
//                       emitted as a single <style> block at the end.
//
// # Custom directives
//
// htmlc supports a custom directive system inspired by Vue's custom directives
// (https://vuejs.org/guide/reusability/custom-directives). Types that implement
// the Directive interface can be registered under a v-name attribute and are
// invoked during server-side rendering.
//
// Only the Created and Mounted hooks are supported because htmlc renders
// server-side only — there are no DOM updates or browser events.
//
//   - Created  – called before the element is rendered; may mutate the
//                element's tag (node.Data) and attributes (node.Attr).
//   - Mounted  – called after the element's closing tag has been written;
//                may write additional HTML to the output writer.
//
// Register directives via Options.Directives or Engine.RegisterDirective:
//
//	engine, err := htmlc.New(htmlc.Options{
//	    ComponentDir: "templates/",
//	    Directives: htmlc.DirectiveRegistry{
//	        "switch": &htmlc.VSwitch{},
//	    },
//	})
//
// The built-in VSwitch directive is the canonical example: it replaces the
// host element's tag with a registered component name supplied by the
// directive's expression, enabling dynamic component dispatch.
//
// # Typical use
//
//	engine, err := htmlc.New(htmlc.Options{ComponentDir: "templates/"})
//	if err != nil { /* handle */ }
//
//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//	    html, err := engine.RenderPage("Page", map[string]any{"title": "Home"})
//	    if err != nil { /* handle */ }
//	    w.Header().Set("Content-Type", "text/html; charset=utf-8")
//	    fmt.Fprint(w, html)
//	})
package htmlc
