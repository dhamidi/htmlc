package htmlc

import (
	htmltemplate "html/template"
	"strings"
	"testing"
)

// --- translateExpr ---

func TestTranslateExpr_simple(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"foo", ".foo"},
		{"foo.bar", ".foo.bar"},
		{"foo.bar.baz", ".foo.bar.baz"},
		{"  foo  ", ".foo"},
	}
	for _, tc := range tests {
		got, err := translateExpr(tc.input)
		if err != nil {
			t.Errorf("translateExpr(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("translateExpr(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestTranslateExpr_unsupported(t *testing.T) {
	unsupported := []string{
		"foo + bar",
		"items[0]",
		"fn()",
		"a ? b : c",
		"a || b",
	}
	for _, e := range unsupported {
		_, err := translateExpr(e)
		if err == nil {
			t.Errorf("translateExpr(%q): expected error, got nil", e)
		}
	}
}

// --- translateTextContent ---

func TestTranslateTextContent(t *testing.T) {
	got, err := translateTextContent("Hello {{ name }}, count: {{ user.count }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "Hello {{.name}}, count: {{.user.count}}"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTranslateTextContent_noInterpolation(t *testing.T) {
	got, err := translateTextContent("plain text")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "plain text" {
		t.Errorf("got %q, want %q", got, "plain text")
	}
}

func TestTranslateTextContent_complexExprError(t *testing.T) {
	_, err := translateTextContent("Hello {{ a + b }}")
	if err == nil {
		t.Error("expected error for complex expression, got nil")
	}
}

// --- ExportTemplateSource ---

func TestExportTemplateSource_simple(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	card, err := ParseFile("Card.vue", `<template><div class="card"><h2>{{ title }}</h2><p>{{ body }}</p></div></template>`)
	if err != nil {
		t.Fatal(err)
	}
	if err := e.Register("Card", "Card.vue"); err != nil {
		// Register requires a file; use direct entry insertion instead.
		_ = err
	}
	// Insert directly for testing.
	e.entries["Card"] = &engineEntry{path: "Card.vue", comp: card}

	src, err := e.ExportTemplateSource("Card")
	if err != nil {
		t.Fatalf("ExportTemplateSource: %v", err)
	}
	if !strings.Contains(src, `{{ define "Card" }}`) {
		t.Errorf("missing define block; got:\n%s", src)
	}
	if !strings.Contains(src, `{{.title}}`) {
		t.Errorf("expected translated interpolation {{.title}}; got:\n%s", src)
	}
	if !strings.Contains(src, `{{.body}}`) {
		t.Errorf("expected translated interpolation {{.body}}; got:\n%s", src)
	}
}

func TestExportTemplateSource_vBindAttr(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp, err := ParseFile("Button.vue", `<template><button :class="variant" v-bind:disabled="isDisabled">{{ label }}</button></template>`)
	if err != nil {
		t.Fatal(err)
	}
	e.entries["Button"] = &engineEntry{path: "Button.vue", comp: comp}

	src, err := e.ExportTemplateSource("Button")
	if err != nil {
		t.Fatalf("ExportTemplateSource: %v", err)
	}
	if !strings.Contains(src, `class="{{.variant}}"`) {
		t.Errorf("expected class={{.variant}}; got:\n%s", src)
	}
	if !strings.Contains(src, `disabled="{{.isDisabled}}"`) {
		t.Errorf("expected disabled={{.isDisabled}}; got:\n%s", src)
	}
}

func TestExportTemplateSource_vIf(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp, err := ParseFile("C.vue", `<template><span v-if="enabled">yes</span><span v-else>no</span></template>`)
	if err != nil {
		t.Fatal(err)
	}
	e.entries["C"] = &engineEntry{path: "C.vue", comp: comp}

	src, err := e.ExportTemplateSource("C")
	if err != nil {
		t.Fatalf("ExportTemplateSource: %v", err)
	}
	if !strings.Contains(src, "{{ if .enabled }}") {
		t.Errorf("expected {{if .enabled}}; got:\n%s", src)
	}
	if !strings.Contains(src, "{{ else }}") {
		t.Errorf("expected {{else}}; got:\n%s", src)
	}
	if !strings.Contains(src, "{{ end }}") {
		t.Errorf("expected {{end}}; got:\n%s", src)
	}
}

func TestExportTemplateSource_slot(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp, err := ParseFile("Card.vue", `<template><div><slot></slot></div></template>`)
	if err != nil {
		t.Fatal(err)
	}
	e.entries["Card"] = &engineEntry{path: "Card.vue", comp: comp}

	src, err := e.ExportTemplateSource("Card")
	if err != nil {
		t.Fatalf("ExportTemplateSource: %v", err)
	}
	if !strings.Contains(src, `{{ block "default" . }}`) {
		t.Errorf("expected block default; got:\n%s", src)
	}
}

func TestExportTemplateSource_unsupportedVShow(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp, err := ParseFile("C.vue", `<template><div v-show="visible">x</div></template>`)
	if err != nil {
		t.Fatal(err)
	}
	e.entries["C"] = &engineEntry{path: "C.vue", comp: comp}

	_, err = e.ExportTemplateSource("C")
	if err == nil {
		t.Error("expected error for v-show, got nil")
	}
}

func TestExportTemplateSource_complexExprError(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp, err := ParseFile("C.vue", `<template><div :class="a + b">x</div></template>`)
	if err != nil {
		t.Fatal(err)
	}
	e.entries["C"] = &engineEntry{path: "C.vue", comp: comp}

	_, err = e.ExportTemplateSource("C")
	if err == nil {
		t.Error("expected error for complex expression, got nil")
	}
}

func TestExportTemplate_parseable(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp, err := ParseFile("Card.vue", `<template><div class="card"><h2>{{ title }}</h2></div></template>`)
	if err != nil {
		t.Fatal(err)
	}
	e.entries["Card"] = &engineEntry{path: "Card.vue", comp: comp}

	tmpl, err := e.ExportTemplate("Card")
	if err != nil {
		t.Fatalf("ExportTemplate: %v", err)
	}
	if tmpl == nil {
		t.Fatal("ExportTemplate returned nil template")
	}

	var sb strings.Builder
	if execErr := tmpl.ExecuteTemplate(&sb, "Card", map[string]any{"title": "Hello"}); execErr != nil {
		t.Fatalf("Execute: %v", execErr)
	}
	got := sb.String()
	if !strings.Contains(got, "Hello") {
		t.Errorf("expected 'Hello' in output; got: %s", got)
	}
}

// --- ImportTemplate / propsFromTemplate ---

func TestImportTemplate_basic(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}

	t2 := htmltemplate.Must(htmltemplate.New("Nav").Parse(`<nav>{{ .user }}</nav>`))
	if err := e.ImportTemplate(t2); err != nil {
		t.Fatalf("ImportTemplate: %v", err)
	}

	if !e.Has("Nav") {
		t.Error("expected 'Nav' to be registered after ImportTemplate")
	}
}

func TestImportTemplate_collision(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp, _ := ParseFile("Nav.vue", `<template><nav></nav></template>`)
	e.entries["Nav"] = &engineEntry{path: "Nav.vue", comp: comp}

	t2 := htmltemplate.Must(htmltemplate.New("Nav").Parse(`<nav>x</nav>`))
	if err := e.ImportTemplate(t2); err == nil {
		t.Error("expected collision error, got nil")
	}
}

func TestForceImportTemplate_overwrite(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	comp, _ := ParseFile("Nav.vue", `<template><nav></nav></template>`)
	e.entries["Nav"] = &engineEntry{path: "Nav.vue", comp: comp}

	t2 := htmltemplate.Must(htmltemplate.New("Nav").Parse(`<nav>x</nav>`))
	if err := e.ForceImportTemplate(t2); err != nil {
		t.Fatalf("ForceImportTemplate: %v", err)
	}
}

func TestPropsFromTemplate(t *testing.T) {
	t2 := htmltemplate.Must(htmltemplate.New("T").Parse(`<p>{{ .title }}</p><span>{{ .user.name }}</span>`))
	props := propsFromTemplate(t2)

	found := map[string]bool{}
	for _, p := range props {
		found[p.Name] = true
	}
	if !found["title"] {
		t.Error("expected 'title' prop")
	}
	if !found["user"] {
		t.Error("expected 'user' prop")
	}
}

func TestImportTemplate_renderSynthetic(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}

	t2 := htmltemplate.Must(htmltemplate.New("Nav").Parse(`<nav>{{ .user }}</nav>`))
	if err := e.ImportTemplate(t2); err != nil {
		t.Fatal(err)
	}

	got, err := e.RenderFragmentString("Nav", map[string]any{"user": "Alice"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(got, "Alice") {
		t.Errorf("expected 'Alice' in output; got: %s", got)
	}
}

func TestImportTemplate_asChildComponent(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}

	// Register synthetic Nav component.
	navTmpl := htmltemplate.Must(htmltemplate.New("Nav").Parse(`<nav>{{ .user }}</nav>`))
	if err := e.ImportTemplate(navTmpl); err != nil {
		t.Fatal(err)
	}

	// Register a .vue Page component that uses <Nav>.
	page, err := ParseFile("Page.vue", `<template><body><Nav :user="currentUser" /></body></template>`)
	if err != nil {
		t.Fatal(err)
	}
	e.entries["Page"] = &engineEntry{path: "Page.vue", comp: page}

	got, err := e.RenderFragmentString("Page", map[string]any{"currentUser": "Bob"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(got, "Bob") {
		t.Errorf("expected 'Bob' in output; got: %s", got)
	}
}

func TestExportTemplateSource_notFound(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = e.ExportTemplateSource("Missing")
	if err == nil {
		t.Error("expected error for missing component")
	}
}

func TestExportTemplateSource_syntheticError(t *testing.T) {
	e, err := New(Options{})
	if err != nil {
		t.Fatal(err)
	}
	navTmpl := htmltemplate.Must(htmltemplate.New("Nav").Parse(`<nav>{{ .user }}</nav>`))
	_ = e.ImportTemplate(navTmpl)

	_, err = e.ExportTemplateSource("Nav")
	if err == nil {
		t.Error("expected error for synthetic component export")
	}
}
