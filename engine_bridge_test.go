package htmlc

import (
	"errors"
	htmltmpl "html/template"
	"path/filepath"
	"strings"
	"testing"
)

// ---- CompileToTemplate tests ------------------------------------------------

func TestEngine_CompileToTemplate_SimpleInterpolation(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Greet.vue"), `<template><p>{{ message }}</p></template>`)
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tmpl, err := e.CompileToTemplate("Greet")
	if err != nil {
		t.Fatalf("CompileToTemplate: %v", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{"message": "hello world"}); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(buf.String(), "hello world") {
		t.Errorf("got %q, want to contain 'hello world'", buf.String())
	}
}

func TestEngine_CompileToTemplate_SubComponent(t *testing.T) {
	dir := t.TempDir()
	// Use kebab-case so the HTML parser emits a non-standard (DataAtom==0) element.
	writeVue(t, filepath.Join(dir, "foot-note.vue"), `<template><span>footer</span></template>`)
	writeVue(t, filepath.Join(dir, "Article.vue"), `<template><div><foot-note></foot-note></div></template>`)
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tmpl, err := e.CompileToTemplate("Article")
	if err != nil {
		t.Fatalf("CompileToTemplate: %v", err)
	}

	// "foot-note" should appear as a named template in the set.
	if tmpl.Lookup("foot-note") == nil {
		t.Error("expected 'foot-note' template in the compiled set")
	}

	// Executing the root template should render the sub-component inline.
	var buf strings.Builder
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(buf.String(), "footer") {
		t.Errorf("got %q, want to contain 'footer'", buf.String())
	}
}

func TestEngine_CompileToTemplate_VIf(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Cond.vue"), `<template><div v-if="show"><p>visible</p></div></template>`)
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tmpl, err := e.CompileToTemplate("Cond")
	if err != nil {
		t.Fatalf("CompileToTemplate: %v", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{"show": true}); err != nil {
		t.Fatalf("Execute (show=true): %v", err)
	}
	if !strings.Contains(buf.String(), "visible") {
		t.Errorf("show=true: got %q, want to contain 'visible'", buf.String())
	}

	buf.Reset()
	if err := tmpl.Execute(&buf, map[string]any{"show": false}); err != nil {
		t.Fatalf("Execute (show=false): %v", err)
	}
	if strings.Contains(buf.String(), "visible") {
		t.Errorf("show=false: got %q, expected no 'visible'", buf.String())
	}
}

func TestEngine_CompileToTemplate_ScopedStyleStripped(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Styled.vue"),
		`<template><p>content</p></template><style scoped>p { color: red; }</style>`)
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tmpl, err := e.CompileToTemplate("Styled")
	if err != nil {
		t.Fatalf("CompileToTemplate: %v", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(buf.String(), "color: red") || strings.Contains(buf.String(), "<style") {
		t.Errorf("style should be stripped, got %q", buf.String())
	}
}

func TestEngine_CompileToTemplate_UnsupportedConstruct(t *testing.T) {
	dir := t.TempDir()
	// Complex expression not supported by bridge.
	writeVue(t, filepath.Join(dir, "Bad.vue"), `<template><p>{{ items[0] }}</p></template>`)
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = e.CompileToTemplate("Bad")
	if err == nil {
		t.Fatal("expected error for unsupported construct")
	}
	var cerr *ConversionError
	if !errors.As(err, &cerr) {
		t.Errorf("expected *ConversionError, got %T: %v", err, err)
	}
	if !errors.Is(err, ErrConversion) {
		t.Errorf("expected error to wrap ErrConversion, got %v", err)
	}
}

func TestEngine_CompileToTemplate_NotFound(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = e.CompileToTemplate("NonExistent")
	if err == nil {
		t.Fatal("expected error for non-existent component")
	}
	if !errors.Is(err, ErrComponentNotFound) {
		t.Errorf("expected ErrComponentNotFound, got %v", err)
	}
}

func TestEngine_CompileToTemplate_NamesAreLowercased(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "MyCard.vue"), `<template><div>card</div></template>`)
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tmpl, err := e.CompileToTemplate("MyCard")
	if err != nil {
		t.Fatalf("CompileToTemplate: %v", err)
	}

	// The root template name must be the lowercased component name.
	if tmpl.Name() != "mycard" {
		t.Errorf("template name = %q, want 'mycard'", tmpl.Name())
	}
	if tmpl.Lookup("mycard") == nil {
		t.Error("expected 'mycard' in compiled template set")
	}
}

// ---- RegisterTemplate tests -------------------------------------------------

func TestEngine_RegisterTemplate_SimpleStdlib(t *testing.T) {
	dir := t.TempDir()
	// Parent Vue component uses <foot-note> (kebab-case → non-standard HTML element).
	writeVue(t, filepath.Join(dir, "Page.vue"), `<template><div><foot-note></foot-note></div></template>`)
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	subTmpl := htmltmpl.Must(htmltmpl.New("foot-note").Parse("<footer>static content</footer>"))
	if err := e.RegisterTemplate("foot-note", subTmpl); err != nil {
		t.Fatalf("RegisterTemplate: %v", err)
	}

	out, err := e.RenderFragmentString("Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "static content") {
		t.Errorf("got %q, want to contain 'static content'", out)
	}
}

func TestEngine_RegisterTemplate_WithUnsupported(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	withTmpl := htmltmpl.Must(htmltmpl.New("bad-tmpl").Parse(`{{with .x}}<div>{{.}}</div>{{end}}`))
	regErr := e.RegisterTemplate("bad-tmpl", withTmpl)
	if regErr == nil {
		t.Fatal("expected error for {{with}}")
	}
	var cerr *ConversionError
	if !errors.As(regErr, &cerr) {
		t.Errorf("expected *ConversionError, got %T: %v", regErr, regErr)
	}
	// Component must not be registered on error.
	if e.Has("bad-tmpl") {
		t.Error("component should not be registered when conversion fails")
	}
}

func TestEngine_RegisterTemplate_LastWriteWins(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t1 := htmltmpl.Must(htmltmpl.New("my-comp").Parse("<p>first</p>"))
	t2 := htmltmpl.Must(htmltmpl.New("my-comp").Parse("<p>second</p>"))

	if err := e.RegisterTemplate("my-comp", t1); err != nil {
		t.Fatalf("RegisterTemplate t1: %v", err)
	}
	if err := e.RegisterTemplate("my-comp", t2); err != nil {
		t.Fatalf("RegisterTemplate t2: %v", err)
	}

	out, err := e.RenderFragmentString("my-comp", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "second") {
		t.Errorf("got %q, expected second registration to win", out)
	}
}
