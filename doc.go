// Package htmlc is a server-side Vue-style component engine for Go. It parses
// .vue Single File Components — each containing a <template>, an optional
// <script>, and an optional <style> section — and renders them to HTML strings
// ready to serve via net/http or any http.Handler-based framework.
//
// [htmlc logo]
//
// [htmlc logo]: ./logo.svg
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
// # Template directives
//
// Every directive below is processed server-side. Client-only directives
// (@click, v-model) are stripped from the output because there is no
// JavaScript runtime.
//
//	{{ expr }}
//	    Mustache text interpolation; the expression result is HTML-escaped.
//	    Example: <p>{{ user.name }}</p>
//
//	v-text="expr"
//	    Sets element text content; HTML-escaped; replaces any child nodes.
//	    Example: <p v-text="msg"></p>
//
//	v-html="expr"
//	    Sets element inner HTML; the value is NOT HTML-escaped.
//	    Example: <div v-html="rawHTML"></div>
//
//	v-if / v-else-if / v-else
//	    Conditional rendering; only the first truthy branch is emitted.
//	    Example: <span v-if="score >= 90">A</span>
//	             <span v-else-if="score >= 70">B</span>
//	             <span v-else>C</span>
//
//	v-switch="expr" (on <template>)
//	    Switch-statement conditional rendering. Evaluates the expression once;
//	    the first child with a matching v-case is rendered; v-default renders
//	    when no case matched. Not part of stable Vue.js; implements RFC #482.
//	    Example: <template v-switch="user.role">
//	               <Admin v-case="'admin'" />
//	               <User  v-default />
//	             </template>
//
//	v-case="expr"
//	    Child of <template v-switch>. Rendered when its expression equals the
//	    parent switch value (Go == comparison).
//
//	v-default
//	    Child of <template v-switch>. Rendered when no preceding v-case matched.
//
//	v-for="item in items"
//	    Iterate over a slice or array. Use (item, i) in items for zero-based
//	    index access.
//	    Example: <li v-for="(item, i) in items">{{ i }}: {{ item }}</li>
//
//	v-for="n in N"
//	    Integer range: n iterates 1 … N (inclusive).
//	    Example: <span v-for="n in 3">{{ n }}</span>
//
//	v-for="(val, key) in obj"
//	    Iterate map entries; val is the value, key is the string key.
//	    Example: <dt v-for="(val, key) in obj">{{ key }}: {{ val }}</dt>
//
//	:attr="expr"
//	    Dynamic attribute binding. Boolean attributes are omitted when the
//	    expression is falsy, present without a value when truthy.
//	    Example: <a :href="url">link</a>
//
//	:class="{ key: bool }"
//	    Object-syntax class binding: keys whose values are truthy are
//	    included; merged with any static class attribute.
//	    Example: <div class="base" :class="{ active: isActive }">…</div>
//
//	:class="[...]"
//	    Array-syntax class binding: non-empty string elements are included.
//	    Example: <div :class="['btn', flag ? 'primary' : '']">…</div>
//
//	:style="{ camelKey: val }"
//	    Inline style binding; camelCase keys are converted to kebab-case.
//	    Example: <p :style="{ fontSize: '14px', color: 'red' }">…</p>
//
//	v-bind="obj"
//	    Attribute spreading: each key-value pair in the map becomes an HTML
//	    attribute on the element. Keys "class" and "style" are merged with
//	    any static and dynamic class/style attributes. Boolean attribute
//	    semantics apply per key. On child components, the map is spread into
//	    the component's prop scope (explicit :prop bindings take precedence
//	    over spread values).
//	    Example: <div v-bind="attrs"></div>
//	             <Button v-bind="buttonProps" :type="'submit'" />
//
//	v-show="expr"
//	    Adds style="display:none" when the expression is falsy; the element
//	    is always present in the output.
//	    Example: <p v-show="visible">content</p>
//
//	v-pre
//	    Skips all interpolation and directive processing for the subtree;
//	    mustache syntax is emitted literally.
//	    Example: <code v-pre>{{ raw }}</code>
//
//	@click, v-model
//	    Client-side event and model directives; stripped on server render.
//	    Example: <button @click="handler">click</button>
//
// # Component composition
//
// Components can include other components in their templates. A child
// component name must start with an uppercase letter to distinguish it from
// HTML elements.
//
// ## Registering components
//
// There are two ways to make components available for composition:
//
// 1. Automatic discovery via ComponentDir: every .vue file in the directory
// tree is registered under its basename (without the .vue extension).
//
//	engine, err := htmlc.New(htmlc.Options{ComponentDir: "templates/"})
//
// 2. Manual registration via Engine.Register or by constructing a Registry
// directly and passing it to NewRenderer.WithComponents.
//
//	engine.Register("Card", cardComponent)
//
//	// Low-level API:
//	htmlc.NewRenderer(page).WithComponents(htmlc.Registry{"Card": card})
//
// ## Default slot
//
// A child component declares <slot /> as a placeholder. The parent places
// inner HTML inside the component tag and it is injected at the slot site.
//
// Card.vue:
//
//	<template><div class="card"><slot /></div></template>
//
// Page.vue:
//
//	<template><Card><p>inner</p></Card></template>
//
// Renders to: <div class="card"><p>inner</p></div>
//
// ## Named slots
//
// A child can declare multiple slots by name using <slot name="…">. The
// parent fills each slot with a <template #name> element. Unmatched content
// goes to the default slot.
//
// Layout.vue:
//
//	<template>
//	  <div class="layout">
//	    <slot name="header"></slot>
//	    <main><slot></slot></main>
//	    <slot name="footer"></slot>
//	  </div>
//	</template>
//
// Page.vue:
//
//	<template>
//	  <Layout>
//	    <template #header><h1>Title</h1></template>
//	    <p>Content</p>
//	    <template #footer><em>Footer</em></template>
//	  </Layout>
//	</template>
//
// ## Scoped slots
//
// The child passes data up to the parent via slot props. The parent
// destructures the props with v-slot="{ item, index }" or binds the whole
// map with v-slot="props".
//
// List.vue:
//
//	<template>
//	  <ul>
//	    <li v-for="(item, i) in items">
//	      <slot :item="item" :index="i"></slot>
//	    </li>
//	  </ul>
//	</template>
//
// Page.vue (destructuring):
//
//	<template>
//	  <List :items="items" v-slot="{ item, index }">
//	    <span>{{ index }}: {{ item }}</span>
//	  </List>
//	</template>
//
// Page.vue (whole map bound to a variable):
//
//	<template>
//	  <Child v-slot="props"><p>{{ props.user.name }}</p></Child>
//	</template>
//
// ## Slot fallback content
//
// Children placed inside a <slot> element in the child component are rendered
// when the parent provides no content for that slot.
//
// Card.vue:
//
//	<template>
//	  <div class="card"><slot><p>No content provided</p></slot></div>
//	</template>
//
// Page.vue (no slot content supplied):
//
//	<template><Card></Card></template>
//
// Renders to: <div class="card"><p>No content provided</p></div>
//
// ## Component resolution
//
// When the renderer encounters a component tag it resolves the name using
// proximity-based resolution, searching for the nearest matching .vue file
// relative to the calling component's directory.
//
// Algorithm (applied at each directory level, starting at the caller's dir):
//
//  1. Exact match            — "my-card"  matches "my-card.vue"
//  2. Capitalise first letter — "card"    matches "Card.vue"
//  3. Kebab to PascalCase    — "my-card"  matches "MyCard.vue"
//  4. Case-insensitive scan  — "CARD"     matches "card.vue"
//
// If none of the four strategies finds a match in the current directory, the
// engine walks one level toward the ComponentDir root and repeats.  After
// exhausting all directories it falls back to the flat registry (required
// for manually registered components and backward compatibility with
// single-directory projects).
//
// Example directory tree:
//
//	templates/
//	  Card.vue          <- generic card used by root templates
//	  blog/
//	    Card.vue        <- blog-specific card
//	    PostPage.vue    <- <Card> resolves to blog/Card.vue
//	  admin/
//	    Card.vue        <- admin-specific card
//	    Dashboard.vue   <- <Card> resolves to admin/Card.vue
//	  shop/
//	    ProductPage.vue <- no Card.vue here; walk-up finds Card.vue at root
//
// PostPage.vue and Dashboard.vue both use an unqualified <Card> tag.
// Because each has a same-named sibling, they resolve independently without
// any explicit import or path qualifier.
//
// ## Explicit cross-directory references
//
// To target a component in a specific directory regardless of the caller's
// location, use a path-qualified is attribute on <component>:
//
//	<!-- always resolves to blog/Card.vue -->
//	<component is="blog/Card" />
//
//	<!-- root-relative: always resolves to Card.vue at ComponentDir root -->
//	<component is="/Card" />
//
//	<!-- dynamic version -->
//	<component :is="'admin/Card'" />
//
// Path-based references do not apply name-folding and return a render error
// if the named component is not found.
//
// Proximity resolution is enabled automatically when ComponentDir is set.
// Manually registered components (via Engine.Register) are available through
// the flat-registry fallback regardless of directory.
//
// # Scope propagation
//
// Every component renders in an isolated scope that contains only the props
// explicitly passed to it as attributes. Parent scope variables are not
// inherited; this makes data flow visible and prevents accidental coupling.
//
// Engine-level functions (registered with Engine.RegisterFunc) are the one
// exception: they are injected into every component's scope at every depth.
// Use RegisterFunc for helper functions — URL builders, route helpers,
// formatters — that need to be callable from any component without explicit
// prop threading.
//
// WithDataMiddleware values are injected into the top-level render scope
// only. If a child component needs a middleware-supplied value, pass it
// down as an explicit prop.
//
//	// Good: helper functions via RegisterFunc — available everywhere
//	engine.RegisterFunc("url", buildURL)
//	engine.RegisterFunc("routeActive", checkActive)
//
//	// Good: per-request data via explicit props
//	// In Page.vue: <Shell :currentUser="currentUser" />
//	// In Shell.vue: {{ currentUser.Name }}
//
//	// Avoid: relying on middleware values inside child components
//	// — they are not automatically propagated.
//
// # Scoped styles
//
// Adding <style scoped> to a .vue file generates a unique data-v-XXXXXXXX
// attribute (derived from the file path) that is stamped on every element
// rendered by that component. The CSS is rewritten by ScopeCSS so every
// selector targets only elements bearing that attribute.
//
// When using Engine, styles are collected automatically:
//   - RenderPage injects a <style> block immediately before </head>.
//   - RenderFragment prepends the <style> block to the output.
//
// When using the low-level API, manage styles manually:
//
//	sc := &htmlc.StyleCollector{}
//	out, err := htmlc.NewRenderer(comp).WithStyles(sc).RenderString(nil)
//	items := sc.All() // []*htmlc.StyleItem; each has a CSS field
//
// # Missing prop handling
//
// By default, a prop name that appears in the template but is absent from the
// scope map causes a render error. Supply a handler to override this behaviour:
//
//	// Engine-wide:
//	engine.WithMissingPropHandler(htmlc.SubstituteMissingProp)
//
//	// Per-render with the low-level API:
//	out, err := htmlc.NewRenderer(comp).
//	    WithMissingPropHandler(htmlc.SubstituteMissingProp).
//	    RenderString(nil)
//
// SubstituteMissingProp is the built-in handler; it emits
// "MISSING PROP: <name>" in place of the missing value. It is useful during
// development to surface missing data without aborting the render.
//
// # Embedded filesystems
//
// Components can be loaded from an embedded filesystem using go:embed:
//
//	//go:embed templates
//	var templateFS embed.FS
//
//	engine, err := htmlc.New(htmlc.Options{
//	    FS:           templateFS,
//	    ComponentDir: "templates",
//	})
//
// Note: Engine.Reload only works when the fs.FS also implements fs.StatFS.
// The standard os.DirFS satisfies this; embed.FS does not, so Reload is a
// no-op for embedded filesystems.
//
// # Low-level API
//
// Use ParseFile, NewRenderer, WithComponents, WithStyles, and WithDirectives
// directly when you need request-scoped control over component registration,
// style collection, or custom directives — for example, in tests or one-off
// renders outside of a long-lived Engine.
//
//	// Parse two components from strings.
//	card, _ := htmlc.ParseFile("Card.vue", `<template><div class="card"><slot /></div></template>`)
//	page, _ := htmlc.ParseFile("Page.vue", `<template><Card><p>inner</p></Card></template>`)
//
//	// Collect scoped styles while rendering.
//	sc := &htmlc.StyleCollector{}
//	out, err := htmlc.NewRenderer(page).
//	    WithComponents(htmlc.Registry{"Card": card}).
//	    WithStyles(sc).
//	    RenderString(nil)
//	if err != nil { /* handle */ }
//
//	// Retrieve scoped CSS generated during the render.
//	for _, item := range sc.All() {
//	    fmt.Println(item.CSS)
//	}
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
// Register custom directives via Options.Directives or Engine.RegisterDirective:
//
//	engine, err := htmlc.New(htmlc.Options{
//	    ComponentDir: "templates/",
//	    Directives: htmlc.DirectiveRegistry{
//	        "my-dir": &MyDirective{},
//	    },
//	})
//
// The built-in VHighlight directive is the canonical example: it sets the
// background colour of the host element to the CSS colour string supplied by
// the directive's expression — mirroring the v-highlight example from the
// Vue.js custom directives guide. VHighlight is not pre-registered; to use it,
// add it via Options.Directives:
//
//	engine, err := htmlc.New(htmlc.Options{
//	    ComponentDir: "templates/",
//	    Directives: htmlc.DirectiveRegistry{
//	        "highlight": &htmlc.VHighlight{},
//	    },
//	})
//
// # DirectiveWithContent
//
// A directive that wants to replace the element's children with custom HTML
// may implement the optional DirectiveWithContent interface in addition to
// Directive.  After Created is called the renderer checks whether the
// directive implements DirectiveWithContent and, if InnerHTML() returns a
// non-empty string, writes it verbatim between the opening and closing tags
// instead of rendering the template children.
//
// # External Directives
//
// htmlc build discovers external directives automatically from the component
// tree.  Any executable file whose base name (without extension) matches
// v-<name> (lower-kebab-case) is registered as an external directive under
// that name.  The executable communicates with htmlc over newline-delimited
// JSON on stdin/stdout, receiving a request envelope for each hook invocation
// and responding with a result envelope.  See the README for the full
// protocol description.
//
// # Tutorial
//
// The fastest path to a working server is Engine + RenderPage:
//
//	engine, err := htmlc.New(htmlc.Options{ComponentDir: "templates/"})
//	if err != nil { /* handle */ }
//
//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//	    w.Header().Set("Content-Type", "text/html; charset=utf-8")
//	    if err := engine.RenderPage(w, "Page", map[string]any{"title": "Home"}); err != nil {
//	        http.Error(w, err.Error(), http.StatusInternalServerError)
//	    }
//	})
//
// Use RenderPage when the component template is a full HTML document
// (html/head/body); it injects collected <style> blocks before </head>.
// Use RenderFragment (or RenderFragmentString) for partial HTML snippets —
// for example, components rendered inside an existing layout or delivered
// over HTMX.
//
// For development, enable hot-reload so changes to .vue files are picked up
// without restarting the server:
//
//	engine, err := htmlc.New(htmlc.Options{
//	    ComponentDir: "templates/",
//	    Reload:       true,
//	})
//
// # Runtime introspection with expvar
//
// An Engine can publish its configuration and performance counters to the
// global expvar registry, making them accessible at /debug/vars (served
// automatically when net/http/pprof or expvar is imported).  Call
// PublishExpvars with a unique prefix after constructing the engine:
//
//	engine, err := htmlc.New(htmlc.Options{ComponentDir: "templates/"})
//	if err != nil { /* handle */ }
//	engine.PublishExpvars("htmlc")
//
//	// Visiting http://localhost:8080/debug/vars now shows, under "htmlc":
//	//   "reload": 0, "debug": 0, "renders": 42, "renderNanos": 1234567, …
//
// Published variables:
//
//	reload        – 1 if hot-reload is enabled, 0 otherwise
//	debug         – 1 if debug mode is enabled, 0 otherwise
//	componentDir  – the active component directory
//	fs            – the type name of the active fs.FS, or "<nil>"
//	renders       – total renderComponent calls (includes errors)
//	renderErrors  – total failed renders
//	reloads       – total hot-reload re-scans performed
//	renderNanos   – cumulative render time in nanoseconds
//	components    – number of unique registered components
//	info.directives – sorted list of registered custom directive names
//
// Two engines in the same process must use different prefixes:
//
//	adminEngine.PublishExpvars("htmlc/admin")
//	publicEngine.PublishExpvars("htmlc/public")
//
// # Runtime option mutation
//
// Reload and Debug can be toggled at runtime without restarting the server:
//
//	engine.SetReload(true)   // enable hot-reload
//	engine.SetDebug(false)   // disable debug mode
//
// Debug mode is currently a no-op. The HTML-comment annotation mechanism
// is being replaced; see docs/proposals/011-debugging.md.
//
// The component directory and filesystem can be changed atomically; discovery
// is re-run under the engine's write lock and the engine's state is only
// updated on success:
//
//	if err := engine.SetComponentDir("templates/v2"); err != nil {
//	    log.Printf("component dir change failed: %v", err)
//	}
//
//	if err := engine.SetFS(newFS); err != nil {
//	    log.Printf("fs change failed: %v", err)
//	}
package htmlc
