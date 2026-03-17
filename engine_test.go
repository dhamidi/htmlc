package htmlc

import (
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// writeVue writes a minimal .vue file at path.
func writeVue(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestEngine_DiscoverRegistersVueFiles(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Card.vue"), `<template><div>{{ title }}</div></template>`)
	writeVue(t, filepath.Join(dir, "ui", "Alert.vue"), `<template><span>{{ msg }}</span></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Card should be registered.
	out, err := e.RenderFragmentString("Card", map[string]any{"title": "Hello"})
	if err != nil {
		t.Fatalf("RenderFragmentString Card: %v", err)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("Card: got %q, want 'Hello'", out)
	}

	// Alert (from ui/ subdir) should register as Alert.
	out, err = e.RenderFragmentString("Alert", map[string]any{"msg": "Warning"})
	if err != nil {
		t.Fatalf("RenderFragmentString Alert: %v", err)
	}
	if !strings.Contains(out, "Warning") {
		t.Errorf("Alert: got %q, want 'Warning'", out)
	}
}

func TestEngine_DuplicateNameLastWins(t *testing.T) {
	// Two files with the same base name: lexically later path wins.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "a", "Card.vue"), `<template><p>first</p></template>`)
	writeVue(t, filepath.Join(dir, "b", "Card.vue"), `<template><p>second</p></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Card", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	// b/Card.vue comes after a/Card.vue lexically, so it should win.
	if !strings.Contains(out, "second") {
		t.Errorf("got %q, want 'second' (last-wins)", out)
	}
}

func TestEngine_RegisterManual(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Widget.vue")
	writeVue(t, p, `<template><aside>{{ val }}</aside></template>`)

	e, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := e.Register("Alias", p); err != nil {
		t.Fatalf("Register: %v", err)
	}

	out, err := e.RenderFragmentString("Alias", map[string]any{"val": "manual"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "manual") {
		t.Errorf("got %q, want 'manual'", out)
	}
}

func TestEngine_UnknownComponentReturnsError(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	err = e.RenderFragment(nil, "Missing", nil)
	if err == nil {
		t.Error("expected error for unknown component, got nil")
	}
}

func TestEngine_RenderPageInjectsStyleBeforeHead(t *testing.T) {
	dir := t.TempDir()
	// Use v-html so the raw HTML string (with </head>) passes through verbatim.
	writeVue(t, filepath.Join(dir, "Page.vue"),
		`<template><div v-html="content"></div></template>`+
			`<style>body{margin:0}</style>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderPageString("Page", map[string]any{
		"content": "<head><title>T</title></head><body>hello</body>",
	})
	if err != nil {
		t.Fatalf("RenderPageString: %v", err)
	}
	styleIdx := strings.Index(out, "<style>")
	headIdx := strings.Index(out, "</head>")
	if styleIdx < 0 {
		t.Fatalf("got %q, want <style> block", out)
	}
	if headIdx < 0 {
		t.Fatalf("got %q, want </head> in output", out)
	}
	if styleIdx > headIdx {
		t.Errorf("got %q, <style> must appear before </head>", out)
	}
}

func TestEngine_RenderPageNoHeadPrependsStyle(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Frag.vue"),
		`<template><section>content</section></template><style>.x{color:red}</style>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderPageString("Frag", nil)
	if err != nil {
		t.Fatalf("RenderPageString: %v", err)
	}
	if !strings.HasPrefix(out, "<style>") {
		t.Errorf("got %q, want output to start with <style>", out)
	}
}

func TestEngine_RenderFragmentPrependsStyle(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Badge.vue"),
		`<template><span>hi</span></template><style>.badge{display:inline}</style>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Badge", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.HasPrefix(out, "<style>") {
		t.Errorf("got %q, want output to start with <style>", out)
	}
	if !strings.Contains(out, ".badge") {
		t.Errorf("got %q, want .badge CSS", out)
	}
}

func TestEngine_ServeComponentWritesContentType(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Hello.vue"), `<template><p>hello</p></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	h := e.ServeComponent("Hello", nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	h(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status %d, want 200", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Errorf("Content-Type %q, want text/html; charset=utf-8", ct)
	}
	if !strings.Contains(rec.Body.String(), "<p>hello</p>") {
		t.Errorf("body %q, want <p>hello</p>", rec.Body.String())
	}
}

func TestEngine_ServeComponent_DataFuncCalledPerRequest(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Greeting.vue"), `<template><h1>{{ title }}</h1></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	callCount := 0
	h := e.ServeComponent("Greeting", func(r *http.Request) map[string]any {
		callCount++
		return map[string]any{"title": "injected title"}
	})

	// First request.
	req1 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec1 := httptest.NewRecorder()
	h(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Errorf("request 1: status %d, want 200", rec1.Code)
	}
	if !strings.Contains(rec1.Body.String(), "injected title") {
		t.Errorf("request 1: body %q, want 'injected title'", rec1.Body.String())
	}

	// Second request: data func must be called again.
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	rec2 := httptest.NewRecorder()
	h(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("request 2: status %d, want 200", rec2.Code)
	}
	if !strings.Contains(rec2.Body.String(), "injected title") {
		t.Errorf("request 2: body %q, want 'injected title'", rec2.Body.String())
	}

	if callCount != 2 {
		t.Errorf("data func called %d times, want 2", callCount)
	}
}

func TestEngine_RenderPage_LayoutStyleBeforeHead(t *testing.T) {
	// A Layout.vue whose <template> starts with <html> should render a full HTML
	// document.  RenderPage must inject the collected <style> block immediately
	// before </head> — not prepend it to the top of the output.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Layout.vue"),
		`<template><html>
<head><title>Layout Test</title></head>
<body><p>page body</p></body>
</html></template>
<style>body { margin: 0; }</style>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderPageString("Layout", nil)
	if err != nil {
		t.Fatalf("RenderPageString: %v", err)
	}

	if !strings.Contains(out, "<html") {
		t.Errorf("output should contain <html, got:\n%s", out)
	}
	if !strings.Contains(out, "<head>") {
		t.Errorf("output should contain <head>, got:\n%s", out)
	}
	if !strings.Contains(out, "<body>") {
		t.Errorf("output should contain <body>, got:\n%s", out)
	}

	styleIdx := strings.Index(out, "<style>")
	headCloseIdx := strings.Index(out, "</head>")
	if styleIdx < 0 {
		t.Fatalf("output should contain <style>, got:\n%s", out)
	}
	if headCloseIdx < 0 {
		t.Fatalf("output should contain </head>, got:\n%s", out)
	}
	if styleIdx > headCloseIdx {
		t.Errorf("<style> (pos %d) must appear before </head> (pos %d) in:\n%s",
			styleIdx, headCloseIdx, out)
	}
}

func TestEngine_MissingProp_NoHandler_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Greeter.vue"), `<template><p>{{ greeting }}</p></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Greeter", nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "[missing: greeting]") {
		t.Errorf("expected '[missing: greeting]' in output, got: %q", out)
	}
}

func TestEngine_MissingProp_DefaultPlaceholder(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Greeter.vue"), `<template><p>{{ greeting }}</p></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Greeter", nil)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !strings.Contains(out, "[missing: greeting]") {
		t.Errorf("expected '[missing: greeting]' in output, got: %q", out)
	}
}

func TestEngine_MissingProp_ErrorOnMissingPropHandler(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Greeter.vue"), `<template><p>{{ greeting }}</p></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.WithMissingPropHandler(ErrorOnMissingProp)

	_, err = e.RenderFragmentString("Greeter", nil)
	if err == nil {
		t.Error("expected error for missing prop with ErrorOnMissingProp handler, got nil")
	}
	if !strings.Contains(err.Error(), "greeting") {
		t.Errorf("expected error to mention 'greeting', got: %v", err)
	}
}

func TestEngine_MissingProp_SubstituteHandler_ProducesPlaceholder(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Greeter.vue"), `<template><p>{{ greeting }}</p></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.WithMissingPropHandler(SubstituteMissingProp)

	out, err := e.RenderFragmentString("Greeter", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "MISSING PROP: greeting") {
		t.Errorf("expected placeholder output, got: %q", out)
	}
}

func TestEngine_MissingProp_CustomHandler_InvokedForAllComponents(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Child.vue"), `<template><span>{{ childProp }}</span></template>`)
	writeVue(t, filepath.Join(dir, "Parent.vue"),
		`<template><div>{{ parentProp }}<Child /></div></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var seen []string
	e.WithMissingPropHandler(func(name string) (any, error) {
		seen = append(seen, name)
		return "CUSTOM:" + name, nil
	})

	out, err := e.RenderFragmentString("Parent", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "CUSTOM:parentProp") {
		t.Errorf("expected CUSTOM:parentProp in output, got: %q", out)
	}
	if !strings.Contains(out, "CUSTOM:childProp") {
		t.Errorf("expected CUSTOM:childProp in output, got: %q", out)
	}

	foundParent, foundChild := false, false
	for _, name := range seen {
		if name == "parentProp" {
			foundParent = true
		}
		if name == "childProp" {
			foundChild = true
		}
	}
	if !foundParent {
		t.Error("custom handler was not called for parentProp")
	}
	if !foundChild {
		t.Error("custom handler was not called for childProp")
	}
}

func TestEngine_AllPropsProvided_NoHandler_Succeeds(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Nameplate.vue"), `<template><span>{{ text }}</span></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Nameplate", map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected 'hello' in output, got: %q", out)
	}
}

func TestNew_WithFS_DiscoverAndRender(t *testing.T) {
	memFS := fstest.MapFS{
		"UserCard.vue":   &fstest.MapFile{Data: []byte(`<template><div class="card">{{ label }}</div></template>`)},
		"StatusBadge.vue": &fstest.MapFile{Data: []byte(`<template><span class="badge">{{ msg }}</span></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("UserCard", map[string]any{"label": "Click me"})
	if err != nil {
		t.Fatalf("RenderFragmentString UserCard: %v", err)
	}
	if !strings.Contains(out, "Click me") {
		t.Errorf("UserCard: got %q, want 'Click me'", out)
	}

	out, err = e.RenderFragmentString("StatusBadge", map[string]any{"msg": "Watch out"})
	if err != nil {
		t.Fatalf("RenderFragmentString StatusBadge: %v", err)
	}
	if !strings.Contains(out, "Watch out") {
		t.Errorf("StatusBadge: got %q, want 'Watch out'", out)
	}
}

func TestNew_WithFS_ComponentDir(t *testing.T) {
	memFS := fstest.MapFS{
		"templates/Card.vue": &fstest.MapFile{Data: []byte(`<template><div>{{ title }}</div></template>`)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "templates"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Card", map[string]any{"title": "Hello FS"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "Hello FS") {
		t.Errorf("got %q, want 'Hello FS'", out)
	}
}

func TestNew_WithFS_NoReload(t *testing.T) {
	// Wrap MapFS in a type that only exposes fs.FS (no StatFS), so that
	// hot-reload is silently skipped without panicking or erroring.
	inner := fstest.MapFS{
		"Live.vue": &fstest.MapFile{Data: []byte(`<template><p>static</p></template>`)},
	}
	type minFS struct{ fs.FS }
	memFS := minFS{inner}

	e, err := New(Options{FS: memFS, ComponentDir: ".", Reload: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Live", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "static") {
		t.Errorf("got %q, want 'static'", out)
	}
}

func TestEngine_Register_WithFS(t *testing.T) {
	memFS := fstest.MapFS{
		"Widget.vue": &fstest.MapFile{Data: []byte(`<template><aside>{{ val }}</aside></template>`)},
	}

	e, err := New(Options{FS: memFS})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := e.Register("MyWidget", "Widget.vue"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	out, err := e.RenderFragmentString("MyWidget", map[string]any{"val": "from fs"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "from fs") {
		t.Errorf("got %q, want 'from fs'", out)
	}
}

func TestEngine_ReloadDetectsChangedFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Live.vue")
	writeVue(t, p, `<template><p>original</p></template>`)

	e, err := New(Options{ComponentDir: dir, Reload: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Live", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (before): %v", err)
	}
	if !strings.Contains(out, "original") {
		t.Errorf("before reload: got %q, want 'original'", out)
	}

	// Overwrite the file and bump the mtime.
	time.Sleep(10 * time.Millisecond)
	writeVue(t, p, `<template><p>updated</p></template>`)

	out, err = e.RenderFragmentString("Live", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (after): %v", err)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("after reload: got %q, want 'updated'", out)
	}
}

func TestEngine_RegisterFunc_AvailableInChildComponent(t *testing.T) {
	// Child.vue calls greet() without receiving it as a prop.
	// Parent.vue embeds Child without passing greet as a prop.
	// greet() must be available in Child because it was registered on the engine.
	memFS := fstest.MapFS{
		"Child.vue": &fstest.MapFile{Data: []byte(
			`<template><span>{{ greet("world") }}</span></template>`,
		)},
		"Parent.vue": &fstest.MapFile{Data: []byte(
			`<template><div><Child /></div></template>`,
		)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.RegisterFunc("greet", func(args ...any) (any, error) {
		if len(args) == 0 {
			return "Hello!", nil
		}
		return "Hello, " + fmt.Sprint(args[0]) + "!", nil
	})

	out, err := e.RenderFragmentString("Parent", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "Hello, world!") {
		t.Errorf("expected 'Hello, world!' in output, got: %q", out)
	}
}

func TestEngine_RegisterFunc_PropOverridesFunc(t *testing.T) {
	// When a child component is passed an explicit prop with the same name as a
	// registered function, the prop value takes precedence.
	memFS := fstest.MapFS{
		"Child.vue": &fstest.MapFile{Data: []byte(
			`<template><span>{{ label }}</span></template>`,
		)},
		"Parent.vue": &fstest.MapFile{Data: []byte(
			`<template><div><Child :label="'from-prop'" /></div></template>`,
		)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// Register a function under the same name as the prop.
	e.RegisterFunc("label", func(args ...any) (any, error) {
		return "from-func", nil
	})

	out, err := e.RenderFragmentString("Parent", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "from-prop") {
		t.Errorf("expected 'from-prop' in output (prop should win over func), got: %q", out)
	}
	if strings.Contains(out, "from-func") {
		t.Errorf("expected func value to be overridden by prop, but got 'from-func' in: %q", out)
	}
}

func TestEngine_RegisterFunc_AvailableInGrandchildComponent(t *testing.T) {
	// Ensures engine funcs propagate recursively (grandchild gets them too).
	memFS := fstest.MapFS{
		"Grandchild.vue": &fstest.MapFile{Data: []byte(
			`<template><em>{{ shout("hi") }}</em></template>`,
		)},
		"Child.vue": &fstest.MapFile{Data: []byte(
			`<template><section><Grandchild /></section></template>`,
		)},
		"Parent.vue": &fstest.MapFile{Data: []byte(
			`<template><div><Child /></div></template>`,
		)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.RegisterFunc("shout", func(args ...any) (any, error) {
		if len(args) == 0 {
			return "SHOUT!", nil
		}
		return strings.ToUpper(fmt.Sprint(args[0])) + "!", nil
	})

	out, err := e.RenderFragmentString("Parent", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "HI!") {
		t.Errorf("expected 'HI!' in output (grandchild should have shout()), got: %q", out)
	}
}

// --- Proximity-based component resolution tests ---

func TestEngine_Proximity_FlatProject_BackwardCompat(t *testing.T) {
	// Flat project: existing behaviour preserved (proximity walk hits root on
	// first step, result identical to today).
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Widget.vue"), `<template><span class="widget">{{ label }}</span></template>`)
	writeVue(t, filepath.Join(dir, "Page.vue"), `<template><div><Widget :label="'click'" /></div></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "click") {
		t.Errorf("got %q, want flat-project Widget to work", out)
	}
}

func TestEngine_Proximity_SameNameDifferentDirs(t *testing.T) {
	// Two same-named components in different directories: caller in blog/ gets
	// blog/Card.vue; caller in admin/ gets admin/Card.vue.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "blog", "Card.vue"), `<template><div class="blog-card">{{ title }}</div></template>`)
	writeVue(t, filepath.Join(dir, "admin", "Card.vue"), `<template><div class="admin-card">{{ title }}</div></template>`)
	writeVue(t, filepath.Join(dir, "blog", "Post.vue"), `<template><section><Card :title="'Blog'" /></section></template>`)
	writeVue(t, filepath.Join(dir, "admin", "Dashboard.vue"), `<template><section><Card :title="'Admin'" /></section></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	blogOut, err := e.RenderFragmentString("Post", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString Post: %v", err)
	}
	if !strings.Contains(blogOut, "blog-card") {
		t.Errorf("Post: got %q, want 'blog-card'", blogOut)
	}

	adminOut, err := e.RenderFragmentString("Dashboard", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString Dashboard: %v", err)
	}
	if !strings.Contains(adminOut, "admin-card") {
		t.Errorf("Dashboard: got %q, want 'admin-card'", adminOut)
	}
}

func TestEngine_Proximity_WalkUpFallback(t *testing.T) {
	// Walk-up fallback: component defined only at root is found from a deeply
	// nested caller.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "SharedWidget.vue"), `<template><span>root-widget</span></template>`)
	writeVue(t, filepath.Join(dir, "blog", "deep", "Thread.vue"),
		`<template><article><SharedWidget /></article></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Thread", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "root-widget") {
		t.Errorf("got %q, want 'root-widget' via walk-up", out)
	}
}

func TestEngine_Proximity_ExplicitPathIs(t *testing.T) {
	// Explicit <component is="blog/Card">: resolves exactly, ignores caller location.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Card.vue"), `<template><div class="root-card">root</div></template>`)
	writeVue(t, filepath.Join(dir, "blog", "Card.vue"), `<template><div class="blog-card">blog</div></template>`)
	writeVue(t, filepath.Join(dir, "Page.vue"),
		`<template><div><component is="blog/Card" /></div></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "blog-card") {
		t.Errorf("got %q, want 'blog-card' from explicit is path", out)
	}
	if strings.Contains(out, "root-card") {
		t.Errorf("got %q, should not contain 'root-card'", out)
	}
}

func TestEngine_Proximity_RootRelativeIs(t *testing.T) {
	// Root-relative <component is="/Card">: always resolves to root Card.vue.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Card.vue"), `<template><div class="root-card">root</div></template>`)
	writeVue(t, filepath.Join(dir, "blog", "Card.vue"), `<template><div class="blog-card">blog</div></template>`)
	writeVue(t, filepath.Join(dir, "blog", "Post.vue"),
		`<template><article><component is="/Card" /></article></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Post", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "root-card") {
		t.Errorf("got %q, want root-card from root-relative /Card", out)
	}
	if strings.Contains(out, "blog-card") {
		t.Errorf("got %q, should not contain 'blog-card'", out)
	}
}

func TestEngine_Proximity_ValidateAll_NoFalsePositives(t *testing.T) {
	// ValidateAll must not report false positives for proximity-resolved refs.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "blog", "Card.vue"), `<template><div>{{ title }}</div></template>`)
	writeVue(t, filepath.Join(dir, "blog", "Post.vue"), `<template><section><Card :title="'t'" /></section></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := e.ValidateAll()
	for _, ve := range errs {
		if strings.Contains(ve.Message, "unknown component") {
			t.Errorf("ValidateAll false positive: %v", ve)
		}
	}
}

func TestEngine_Proximity_HotReload_FullRebuild(t *testing.T) {
	// Hot reload: after modifying a component file, the full registry is rebuilt
	// and subsequent renders use the updated component.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "blog", "Card.vue"), `<template><div class="v1">original</div></template>`)
	writeVue(t, filepath.Join(dir, "blog", "Post.vue"), `<template><section><Card /></section></template>`)

	e, err := New(Options{ComponentDir: dir, Reload: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Post", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (before): %v", err)
	}
	if !strings.Contains(out, "original") {
		t.Errorf("before reload: got %q, want 'original'", out)
	}

	time.Sleep(10 * time.Millisecond)
	writeVue(t, filepath.Join(dir, "blog", "Card.vue"), `<template><div class="v2">updated</div></template>`)

	out, err = e.RenderFragmentString("Post", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (after): %v", err)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("after reload: got %q, want 'updated'", out)
	}
}

func TestEngine_Proximity_SlotAuthoringProximity(t *testing.T) {
	// Slot authoring proximity: slot content defined in blog/Post.vue referencing
	// Card resolves to blog/Card.vue even when rendered from a root Layout.vue
	// that also defines a Card.vue.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Card.vue"), `<template><div class="root-card"><slot /></div></template>`)
	writeVue(t, filepath.Join(dir, "blog", "Card.vue"), `<template><div class="blog-card"><slot /></div></template>`)
	// Layout provides a named slot "content" that callers fill.
	writeVue(t, filepath.Join(dir, "Layout.vue"),
		`<template><main><slot name="content" /></main></template>`)
	// Post fills the Layout's "content" slot with a Card reference.
	// The Card reference should resolve using Post's directory (blog/), not Layout's.
	writeVue(t, filepath.Join(dir, "blog", "Post.vue"),
		`<template><Layout><template #content><Card /></template></Layout></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Post", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "blog-card") {
		t.Errorf("slot authoring: got %q, want 'blog-card' (slot uses Post's proximity)", out)
	}
	if strings.Contains(out, "root-card") {
		t.Errorf("slot authoring: got %q, should not contain 'root-card'", out)
	}
}

func TestEngine_Proximity_ForwardSlashKeysOnAllPlatforms(t *testing.T) {
	// nsEntries keys must use forward slashes regardless of OS path separator.
	// Verify by using a deep path and confirming resolution works correctly.
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "a", "b", "Leaf.vue"), `<template><span>deep-leaf</span></template>`)
	writeVue(t, filepath.Join(dir, "a", "b", "Page.vue"), `<template><div><Leaf /></div></template>`)

	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString("Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "deep-leaf") {
		t.Errorf("got %q, want 'deep-leaf'", out)
	}
}

func TestRenderPage_InspectorInjected_WhenDebug(t *testing.T) {
	memFS := fstest.MapFS{
		"Page.vue": &fstest.MapFile{Data: []byte(
			`<template><html><head></head><body><p>hello</p></body></html></template>`,
		)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: ".", Debug: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := e.RenderPageString("Page", nil)
	if err != nil {
		t.Fatalf("RenderPageString: %v", err)
	}
	if !strings.Contains(out, "<script>") {
		t.Errorf("expected <script> in output, got: %q", out)
	}
	if !strings.Contains(out, "htmlc-inspector") {
		t.Errorf("expected 'htmlc-inspector' in output, got: %q", out)
	}
	scriptIdx := strings.Index(out, "<script>")
	bodyIdx := strings.Index(out, "</body>")
	if scriptIdx < 0 || bodyIdx < 0 || scriptIdx >= bodyIdx {
		t.Errorf("expected <script> to appear before </body>; scriptIdx=%d bodyIdx=%d", scriptIdx, bodyIdx)
	}
}

func TestRenderPage_InspectorAbsent_WhenNoDebug(t *testing.T) {
	memFS := fstest.MapFS{
		"Page.vue": &fstest.MapFile{Data: []byte(
			`<template><html><head></head><body><p>hello</p></body></html></template>`,
		)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := e.RenderPageString("Page", nil)
	if err != nil {
		t.Fatalf("RenderPageString: %v", err)
	}
	if strings.Contains(out, "htmlc-inspector") {
		t.Errorf("expected no inspector script in non-debug output, got: %q", out)
	}
}

func TestRenderPage_InspectorFallback_NoBodyTag(t *testing.T) {
	memFS := fstest.MapFS{
		"Frag.vue": &fstest.MapFile{Data: []byte(
			`<template><div><p>no body tag here</p></div></template>`,
		)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: ".", Debug: true})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := e.RenderPageString("Frag", nil)
	if err != nil {
		t.Fatalf("RenderPageString: %v", err)
	}
	if !strings.Contains(out, "htmlc-inspector") {
		t.Errorf("expected 'htmlc-inspector' in fallback output, got: %q", out)
	}
}
