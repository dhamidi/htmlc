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
