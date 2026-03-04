package htmlc

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
	out, err := e.RenderFragment("Card", map[string]any{"title": "Hello"})
	if err != nil {
		t.Fatalf("RenderFragment Card: %v", err)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("Card: got %q, want 'Hello'", out)
	}

	// Alert (from ui/ subdir) should register as Alert.
	out, err = e.RenderFragment("Alert", map[string]any{"msg": "Warning"})
	if err != nil {
		t.Fatalf("RenderFragment Alert: %v", err)
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

	out, err := e.RenderFragment("Card", nil)
	if err != nil {
		t.Fatalf("RenderFragment: %v", err)
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

	out, err := e.RenderFragment("Alias", map[string]any{"val": "manual"})
	if err != nil {
		t.Fatalf("RenderFragment: %v", err)
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
	_, err = e.RenderFragment("Missing", nil)
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

	out, err := e.RenderPage("Page", map[string]any{
		"content": "<head><title>T</title></head><body>hello</body>",
	})
	if err != nil {
		t.Fatalf("RenderPage: %v", err)
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

	out, err := e.RenderPage("Frag", nil)
	if err != nil {
		t.Fatalf("RenderPage: %v", err)
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

	out, err := e.RenderFragment("Badge", nil)
	if err != nil {
		t.Fatalf("RenderFragment: %v", err)
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

	out, err := e.RenderPage("Layout", nil)
	if err != nil {
		t.Fatalf("RenderPage: %v", err)
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

	_, err = e.RenderFragment("Greeter", nil)
	if err == nil {
		t.Error("expected error for missing prop, got nil")
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

	out, err := e.RenderFragment("Greeter", nil)
	if err != nil {
		t.Fatalf("RenderFragment: %v", err)
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

	out, err := e.RenderFragment("Parent", nil)
	if err != nil {
		t.Fatalf("RenderFragment: %v", err)
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

	out, err := e.RenderFragment("Nameplate", map[string]any{"text": "hello"})
	if err != nil {
		t.Fatalf("RenderFragment: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected 'hello' in output, got: %q", out)
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

	out, err := e.RenderFragment("Live", nil)
	if err != nil {
		t.Fatalf("RenderFragment (before): %v", err)
	}
	if !strings.Contains(out, "original") {
		t.Errorf("before reload: got %q, want 'original'", out)
	}

	// Overwrite the file and bump the mtime.
	time.Sleep(10 * time.Millisecond)
	writeVue(t, p, `<template><p>updated</p></template>`)

	out, err = e.RenderFragment("Live", nil)
	if err != nil {
		t.Fatalf("RenderFragment (after): %v", err)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("after reload: got %q, want 'updated'", out)
	}
}
