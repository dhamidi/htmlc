package htmlc_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"

	"github.com/dhamidi/htmlc"
)

// Example demonstrates end-to-end use of the htmlc engine: create an Engine
// from a directory of .vue files, then render a component as an HTML fragment.
func Example() {
	dir, err := os.MkdirTemp("", "htmlc-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Write a simple component with no scoped styles so the output is stable.
	vue := `<template><p>Hello, {{ name }}!</p></template>`
	if err := os.WriteFile(filepath.Join(dir, "Greeting.vue"), []byte(vue), 0644); err != nil {
		log.Fatal(err)
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: dir})
	if err != nil {
		log.Fatal(err)
	}

	out, err := engine.RenderFragmentString("Greeting", map[string]any{"name": "World"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <p>Hello, World!</p>
}

// ExampleParseFile parses an inline .vue source string and inspects the
// resulting Component's path.
func ExampleParseFile() {
	comp, err := htmlc.ParseFile("Greeting.vue", `<template><p>Hello!</p></template>`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(comp.Path)
	// Output:
	// Greeting.vue
}

// ExampleRender_interpolation shows {{ expr }} text interpolation: member
// access, arithmetic, and ternary expressions all evaluated at render time.
func ExampleRender_interpolation() {
	const src = `<template><p>{{ user.name }}, total {{ price * qty }}, active: {{ active ? "yes" : "no" }}</p></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.RenderString(comp, map[string]any{
		"user":   map[string]any{"name": "Alice"},
		"price":  float64(10),
		"qty":    float64(3),
		"active": true,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <p>Alice, total 30, active: yes</p>
}

// ExampleRender_vText shows v-text="expr" which sets element text content
// with HTML escaping, replacing any child nodes.
func ExampleRender_vText() {
	const src = `<template><p v-text="msg"></p></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.RenderString(comp, map[string]any{"msg": "Hello & World"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <p>Hello &amp; World</p>
}

// ExampleRender_vHtml shows v-html="expr" which renders a raw HTML string as
// element content without escaping angle brackets or other HTML characters.
func ExampleRender_vHtml() {
	const src = `<template><div v-html="raw"></div></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.RenderString(comp, map[string]any{"raw": "<b>bold</b>"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <div><b>bold</b></div>
}

// ExampleRender_vIf shows v-if/v-else-if/v-else conditional rendering: only
// the first truthy branch produces output; the rest are skipped entirely.
func ExampleRender_vIf() {
	const src = `<template><span v-if="score >= 90">A</span><span v-else-if="score >= 70">B</span><span v-else>C</span></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.RenderString(comp, map[string]any{"score": float64(75)})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <span>B</span>
}

// ExampleRender_vFor shows v-for iterating over an array, with (item, index)
// destructuring, and over an integer range (n in N produces 1..N).
func ExampleRender_vFor() {
	// Plain array.
	c1, _ := htmlc.ParseFile("t.vue", `<template><li v-for="item in items">{{ item }}</li></template>`)
	o1, _ := htmlc.RenderString(c1, map[string]any{"items": []any{"a", "b", "c"}})
	fmt.Println(o1)

	// Array with zero-based index.
	c2, _ := htmlc.ParseFile("t.vue", `<template><li v-for="(item, i) in items">{{ i }}:{{ item }}</li></template>`)
	o2, _ := htmlc.RenderString(c2, map[string]any{"items": []any{"x", "y"}})
	fmt.Println(o2)

	// Integer range: n iterates 1, 2, 3.
	c3, _ := htmlc.ParseFile("t.vue", `<template><span v-for="n in 3">{{ n }}</span></template>`)
	o3, _ := htmlc.RenderString(c3, nil)
	fmt.Println(o3)
	// Output:
	// <li>a</li><li>b</li><li>c</li>
	// <li>0:x</li><li>1:y</li>
	// <span>1</span><span>2</span><span>3</span>
}

// ExampleRender_vForObject shows v-for iterating map entries using the
// (value, key) destructuring form.
func ExampleRender_vForObject() {
	const src = `<template><dt v-for="(value, key) in obj">{{ key }}: {{ value }}</dt></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	// Use a single-entry map so iteration order is deterministic.
	out, err := htmlc.RenderString(comp, map[string]any{
		"obj": map[string]any{"lang": "Go"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <dt>lang: Go</dt>
}

// ExampleRender_vBind shows :attr dynamic attribute binding, and how boolean
// attributes are handled: :disabled="false" omits the attribute entirely while
// :disabled="true" emits it without a value.
func ExampleRender_vBind() {
	// Dynamic href binding.
	c1, _ := htmlc.ParseFile("t.vue", `<template><a :href="url">link</a></template>`)
	o1, _ := htmlc.RenderString(c1, map[string]any{"url": "https://example.com"})
	fmt.Println(o1)

	// :disabled="false" — attribute is omitted.
	c2, _ := htmlc.ParseFile("t.vue", `<template><button :disabled="false">enabled</button></template>`)
	o2, _ := htmlc.RenderString(c2, nil)
	fmt.Println(o2)

	// :disabled="true" — attribute is present without a value.
	c3, _ := htmlc.ParseFile("t.vue", `<template><button :disabled="true">disabled</button></template>`)
	o3, _ := htmlc.RenderString(c3, nil)
	fmt.Println(o3)
	// Output:
	// <a href="https://example.com">link</a>
	// <button>enabled</button>
	// <button disabled>disabled</button>
}

// ExampleRender_vBindClass shows :class with object syntax (keys whose values
// are truthy are included) and array syntax, both merged with a static class.
func ExampleRender_vBindClass() {
	// Object syntax merged with static class="base".
	c1, _ := htmlc.ParseFile("t.vue", `<template><div class="base" :class="{ active: isActive, hidden: isHidden }">x</div></template>`)
	o1, _ := htmlc.RenderString(c1, map[string]any{"isActive": true, "isHidden": false})
	fmt.Println(o1)

	// Array syntax: non-empty strings are included.
	c2, _ := htmlc.ParseFile("t.vue", `<template><div :class="['btn', flag ? 'primary' : '']">y</div></template>`)
	o2, _ := htmlc.RenderString(c2, map[string]any{"flag": true})
	fmt.Println(o2)
	// Output:
	// <div class="base active">x</div>
	// <div class="btn primary">y</div>
}

// ExampleRender_vBindStyle shows :style with an object whose camelCase keys
// are automatically converted to kebab-case CSS property names.
func ExampleRender_vBindStyle() {
	const src = `<template><p :style="{ color: 'red', fontSize: '14px' }">styled</p></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.RenderString(comp, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <p style="color:red;font-size:14px">styled</p>
}

// ExampleRender_vShow shows v-show: a falsy expression injects
// style="display:none" while the element is still present in the DOM;
// a truthy expression renders the element normally.
func ExampleRender_vShow() {
	const src = `<template><p v-show="false">hidden</p><p v-show="true">visible</p></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.RenderString(comp, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <p style="display:none">hidden</p><p>visible</p>
}

// ExampleRender_vPre shows v-pre: mustache syntax inside the element is
// emitted literally without any interpolation or directive processing.
func ExampleRender_vPre() {
	const src = `<template><code v-pre>{{ raw }}</code></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.RenderString(comp, map[string]any{"raw": "ignored"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <code>{{ raw }}</code>
}

// ExampleRender_componentSlot shows a parent component passing inner HTML
// content into a child component's <slot /> placeholder.
func ExampleRender_componentSlot() {
	card, _ := htmlc.ParseFile("Card.vue", `<template><div class="card"><slot /></div></template>`)
	page, _ := htmlc.ParseFile("Page.vue", `<template><Card><p>inner</p></Card></template>`)
	out, err := htmlc.NewRenderer(page).
		WithComponents(htmlc.Registry{"Card": card}).
		RenderString(nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <div class="card"><p>inner</p></div>
}

// ExampleRender_componentProps shows a parent passing a dynamic prop via
// :title="expr" and a static string prop via class="x" to a child component.
func ExampleRender_componentProps() {
	header, _ := htmlc.ParseFile("Header.vue", `<template><h1>{{ title }}</h1></template>`)
	page, _ := htmlc.ParseFile("Page.vue", `<template><Header :title="heading" class="main"></Header></template>`)
	out, err := htmlc.NewRenderer(page).
		WithComponents(htmlc.Registry{"Header": header}).
		RenderString(map[string]any{"heading": "My Site"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <h1>My Site</h1>
}

// ExampleComponent_Props shows how to call Props() to discover the prop names
// a component template uses and the expressions in which they appear.
func ExampleComponent_Props() {
	comp, err := htmlc.ParseFile("t.vue", `<template><p>{{ message }}</p></template>`)
	if err != nil {
		log.Fatal(err)
	}
	props := comp.Props()
	fmt.Println(len(props))
	fmt.Println(props[0].Name)
	fmt.Println(props[0].Expressions[0])
	// Output:
	// 1
	// message
	// message
}

// ExampleRenderer_WithMissingPropHandler shows SubstituteMissingProp
// substituting a placeholder string when a required template prop is absent.
func ExampleRenderer_WithMissingPropHandler() {
	comp, err := htmlc.ParseFile("t.vue", `<template><p>{{ name }}</p></template>`)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.NewRenderer(comp).
		WithMissingPropHandler(htmlc.SubstituteMissingProp).
		RenderString(nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <p>MISSING PROP: name</p>
}

// ExampleEngine_RenderPage demonstrates full-page rendering: collected
// <style> blocks are injected immediately before </head>.
func ExampleEngine_RenderPage() {
	dir, err := os.MkdirTemp("", "htmlc-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	const vue = `<template><html><head><title>Demo</title></head><body><p>Hello</p></body></html></template><style>body{margin:0}</style>`
	if err := os.WriteFile(filepath.Join(dir, "Page.vue"), []byte(vue), 0644); err != nil {
		log.Fatal(err)
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: dir})
	if err != nil {
		log.Fatal(err)
	}

	out, err := engine.RenderPageString("Page", nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <html><head><title>Demo</title><style>body{margin:0}</style></head><body><p>Hello</p></body></html>
}

// ExampleEngine_ServeComponent shows how ServeComponent wraps a component as
// an http.HandlerFunc, demonstrated with an httptest round-trip.
func ExampleEngine_ServeComponent() {
	dir, err := os.MkdirTemp("", "htmlc-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "Hello.vue"), []byte(`<template><p>hello</p></template>`), 0644); err != nil {
		log.Fatal(err)
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: dir})
	if err != nil {
		log.Fatal(err)
	}

	h := engine.ServeComponent("Hello", nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	fmt.Println(rec.Code)
	fmt.Println(rec.Header().Get("Content-Type"))
	fmt.Println(rec.Body.String())
	// Output:
	// 200
	// text/html; charset=utf-8
	// <p>hello</p>
}

// ExampleRender_namedSlots demonstrates named slots: a Layout component
// declares header and footer named slots plus a default slot for body content,
// and the caller fills each slot with a <template #name> element.
func ExampleRender_namedSlots() {
	layout, _ := htmlc.ParseFile("Layout.vue",
		`<template><div class="layout"><slot name="header"></slot><main><slot></slot></main><slot name="footer"></slot></div></template>`)
	page, _ := htmlc.ParseFile("Page.vue",
		`<template><Layout><template #header><h1>Title</h1></template><p>Content</p><template #footer><em>Footer</em></template></Layout></template>`)
	out, err := htmlc.NewRenderer(page).
		WithComponents(htmlc.Registry{"Layout": layout}).
		RenderString(nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <div class="layout"><h1>Title</h1><main><p>Content</p></main><em>Footer</em></div>
}

// ExampleRender_scopedSlots demonstrates scoped slots: a List component
// passes each item and its index to the caller via slot props, and the caller
// uses v-slot="{ item, index }" to render a custom item template.
func ExampleRender_scopedSlots() {
	list, _ := htmlc.ParseFile("List.vue",
		`<template><ul><li v-for="(item, i) in items"><slot :item="item" :index="i"></slot></li></ul></template>`)
	page, _ := htmlc.ParseFile("Page.vue",
		`<template><List :items="items" v-slot="{ item, index }"><span>{{ index }}: {{ item }}</span></List></template>`)
	out, err := htmlc.NewRenderer(page).
		WithComponents(htmlc.Registry{"List": list}).
		RenderString(map[string]any{"items": []any{"a", "b"}})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <ul><li><span>0: a</span></li><li><span>1: b</span></li></ul>
}

// ExampleRender_slotFallbackContent demonstrates slot fallback content:
// when the caller provides no content for a slot, the child component's
// fallback children inside <slot>…</slot> are rendered instead.
func ExampleRender_slotFallbackContent() {
	card, _ := htmlc.ParseFile("Card.vue",
		`<template><div class="card"><slot><p>No content provided</p></slot></div></template>`)
	page, _ := htmlc.ParseFile("Page.vue", `<template><Card></Card></template>`)
	out, err := htmlc.NewRenderer(page).
		WithComponents(htmlc.Registry{"Card": card}).
		RenderString(nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <div class="card"><p>No content provided</p></div>
}

// ExampleRender_singleVariableSlotBinding demonstrates v-slot="slotProps"
// binding: the entire slot props map is bound to a single variable, allowing
// the caller to access any prop via slotProps.key.
func ExampleRender_singleVariableSlotBinding() {
	child, _ := htmlc.ParseFile("Child.vue",
		`<template><div><slot :user="theuser" :count="total"></slot></div></template>`)
	page, _ := htmlc.ParseFile("Page.vue",
		`<template><Child :theuser="u" :total="n" v-slot="props"><p>{{ props.user.name }}: {{ props.count }}</p></Child></template>`)
	out, err := htmlc.NewRenderer(page).
		WithComponents(htmlc.Registry{"Child": child}).
		RenderString(map[string]any{
			"u": map[string]any{"name": "Alice"},
			"n": float64(3),
		})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <div><p>Alice: 3</p></div>
}

// ExampleRender_eventPassthrough shows that client-side directives such as
// @click and v-model are preserved in the server-rendered output unchanged,
// ready to be activated by the client-side framework.
func ExampleRender_eventPassthrough() {
	const src = `<template><button @click="handler">click</button><input v-model="name"></template>`
	comp, err := htmlc.ParseFile("t.vue", src)
	if err != nil {
		log.Fatal(err)
	}
	out, err := htmlc.RenderString(comp, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <button @click="handler">click</button><input v-model="name">
}

// ExampleEngine_scopedStyles shows that a component with <style scoped> adds
// a unique data attribute to each HTML element and scopes its CSS selectors
// to match only elements within that component.
func ExampleEngine_scopedStyles() {
	// ParseFile with a fixed path gives a deterministic scope ID.
	const path = "Button.vue"
	comp, err := htmlc.ParseFile(path, `<template><p>hello</p></template><style scoped>p{color:red}</style>`)
	if err != nil {
		log.Fatal(err)
	}
	sc := &htmlc.StyleCollector{}
	out, err := htmlc.NewRenderer(comp).WithStyles(sc).RenderString(nil)
	if err != nil {
		log.Fatal(err)
	}
	items := sc.All()
	fmt.Println(out)
	fmt.Println(items[0].CSS)
	// Output:
	// <p data-v-6fc690bb>hello</p>
	// p[data-v-6fc690bb]{color:red}
}
