package htmlc_test

import (
	htmltemplate "html/template"
	"strings"
	"testing"

	htmlc "github.com/dhamidi/htmlc"
)

// ---- helpers ----------------------------------------------------------------

func mustParseVue(t *testing.T, src string) *htmlc.Component {
	t.Helper()
	comp, err := htmlc.ParseFile("test.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return comp
}

func mustVueToTemplate(t *testing.T, src, name string) *htmlc.VueToTemplateResult {
	t.Helper()
	comp := mustParseVue(t, src)
	result, err := htmlc.VueToTemplate(comp.Template, name)
	if err != nil {
		t.Fatalf("VueToTemplate: %v", err)
	}
	return result
}

func assertContains(t *testing.T, haystack, needle, label string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: expected %q to contain %q\ngot: %s", label, haystack, needle, haystack)
	}
}

func assertError(t *testing.T, err error, label string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: expected an error but got nil", label)
	}
}

func assertNoError(t *testing.T, err error, label string) {
	t.Helper()
	if err != nil {
		t.Errorf("%s: unexpected error: %v", label, err)
	}
}

// ---- ClassifyExpr -----------------------------------------------------------

func TestClassifyExpr(t *testing.T) {
	cases := []struct {
		expr string
		want htmlc.ExprKind
	}{
		{".", htmlc.ExprSimpleIdent},
		{"name", htmlc.ExprSimpleIdent},
		{"_name", htmlc.ExprSimpleIdent},
		{"Name123", htmlc.ExprSimpleIdent},
		{"a.b", htmlc.ExprDotPath},
		{"a.b.c", htmlc.ExprDotPath},
		{"post.title", htmlc.ExprDotPath},
		{"items[0]", htmlc.ExprComplex},
		{"a + b", htmlc.ExprComplex},
		{"foo()", htmlc.ExprComplex},
		{"a?.b", htmlc.ExprComplex},
	}
	for _, tc := range cases {
		got := htmlc.ClassifyExpr(tc.expr)
		if got != tc.want {
			t.Errorf("ClassifyExpr(%q) = %v, want %v", tc.expr, got, tc.want)
		}
	}
}

// ---- DotPrefix --------------------------------------------------------------

func TestDotPrefix(t *testing.T) {
	cases := []struct {
		expr    string
		want    string
		wantErr bool
	}{
		{"name", ".name", false},
		{"a.b.c", ".a.b.c", false},
		{".", ".", false},
		{"foo()", "", true},
		{"a + b", "", true},
	}
	for _, tc := range cases {
		got, err := htmlc.DotPrefix(tc.expr)
		if tc.wantErr {
			if err == nil {
				t.Errorf("DotPrefix(%q): expected error, got %q", tc.expr, got)
			}
		} else {
			if err != nil {
				t.Errorf("DotPrefix(%q): unexpected error: %v", tc.expr, err)
			}
			if got != tc.want {
				t.Errorf("DotPrefix(%q) = %q, want %q", tc.expr, got, tc.want)
			}
		}
	}
}

// ---- VueToTemplate: text interpolation --------------------------------------

func TestVueToTemplate_SimpleIdent(t *testing.T) {
	result := mustVueToTemplate(t, `<template><p>{{ title }}</p></template>`, "Test")
	assertContains(t, result.Text, "{{.title}}", "simple ident")
}

func TestVueToTemplate_DotPath(t *testing.T) {
	result := mustVueToTemplate(t, `<template><p>{{ post.title }}</p></template>`, "Test")
	assertContains(t, result.Text, "{{.post.title}}", "dot-path")
}

func TestVueToTemplate_ComplexExprError(t *testing.T) {
	comp := mustParseVue(t, `<template><p>{{ items[0] }}</p></template>`)
	_, err := htmlc.VueToTemplate(comp.Template, "Test")
	assertError(t, err, "complex expression")
	var ce *htmlc.ConversionError
	if ok := isConversionError(err, &ce); !ok {
		t.Errorf("expected *ConversionError, got %T: %v", err, err)
	}
}

// ---- VueToTemplate: v-if / v-else-if / v-else --------------------------------

func TestVueToTemplate_VIf(t *testing.T) {
	result := mustVueToTemplate(t, `<template><div v-if="show">yes</div></template>`, "Test")
	assertContains(t, result.Text, "{{if .show}}", "v-if open")
	assertContains(t, result.Text, "{{end}}", "v-if end")
}

func TestVueToTemplate_VIfElse(t *testing.T) {
	src := `<template><div v-if="show">yes</div><div v-else>no</div></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, "{{if .show}}", "v-if")
	assertContains(t, result.Text, "{{else}}", "v-else")
	assertContains(t, result.Text, "{{end}}", "end")
}

func TestVueToTemplate_VIfElseIf(t *testing.T) {
	src := `<template>
		<div v-if="a">A</div>
		<div v-else-if="b">B</div>
		<div v-else>C</div>
	</template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, "{{if .a}}", "v-if")
	assertContains(t, result.Text, "{{else if .b}}", "v-else-if")
	assertContains(t, result.Text, "{{else}}", "v-else")
	assertContains(t, result.Text, "{{end}}", "end")
}

// ---- VueToTemplate: v-for ---------------------------------------------------

func TestVueToTemplate_VFor(t *testing.T) {
	src := `<template><ul><li v-for="item in items">{{ item }}</li></ul></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, "{{range .items}}", "range")
	assertContains(t, result.Text, "{{end}}", "end")
	// Loop variable resolves to "."
	assertContains(t, result.Text, "{{.}}", "loop var as dot")
}

func TestVueToTemplate_VForOuterScopeRef(t *testing.T) {
	// "title" is an outer-scope variable, not the loop variable "item".
	src := `<template><ul><li v-for="item in items">{{ title }}</li></ul></template>`
	comp := mustParseVue(t, src)
	_, err := htmlc.VueToTemplate(comp.Template, "Test")
	assertError(t, err, "outer-scope reference inside v-for")
}

// ---- VueToTemplate: v-show --------------------------------------------------

func TestVueToTemplate_VShowNoStyle(t *testing.T) {
	src := `<template><div v-show="visible">content</div></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, "display:none", "display:none injected")
	assertContains(t, result.Text, ".visible", "condition")
}

func TestVueToTemplate_VShowWithStyle(t *testing.T) {
	src := `<template><div v-show="visible" style="color:red">content</div></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, "display:none", "display:none merged")
	assertContains(t, result.Text, "color:red", "existing style preserved")
	if len(result.Warnings) == 0 {
		t.Error("expected a warning for v-show + existing style, got none")
	}
}

// ---- VueToTemplate: v-html --------------------------------------------------

func TestVueToTemplate_VHtml(t *testing.T) {
	src := `<template><p v-html="content"></p></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, "{{.content}}", "v-html output")
	if len(result.Warnings) == 0 {
		t.Error("expected a warning for v-html, got none")
	}
}

// ---- VueToTemplate: v-bind spread -------------------------------------------

func TestVueToTemplate_VBindSpread(t *testing.T) {
	src := `<template><div v-bind="attrs">content</div></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, "{{.attrs}}", "spread output")
	if len(result.Warnings) == 0 {
		t.Error("expected a warning for v-bind spread, got none")
	}
}

// ---- VueToTemplate: v-text --------------------------------------------------

func TestVueToTemplate_VText(t *testing.T) {
	src := `<template><p v-text="label">ignored child</p></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, "{{.label}}", "v-text output")
	if strings.Contains(result.Text, "ignored child") {
		t.Error("v-text: children should be discarded")
	}
}

// ---- VueToTemplate: v-switch ------------------------------------------------

func TestVueToTemplate_VSwitch(t *testing.T) {
	src := `<template>
		<template v-switch="status">
			<span v-case="active">Active</span>
			<span v-case="inactive">Inactive</span>
			<span v-default>Unknown</span>
		</template>
	</template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, ".status", "switch expr")
	assertContains(t, result.Text, `"active"`, "first case")
	assertContains(t, result.Text, `"inactive"`, "second case")
	assertContains(t, result.Text, "{{else}}", "v-default as else")
}

// ---- VueToTemplate: <slot> --------------------------------------------------

func TestVueToTemplate_SlotDefault(t *testing.T) {
	src := `<template><slot>fallback</slot></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, `{{block "default" .}}`, "default slot block")
	assertContains(t, result.Text, "{{end}}", "end")
}

func TestVueToTemplate_SlotNamed(t *testing.T) {
	src := `<template><slot name="header">Header</slot></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, `{{block "header" .}}`, "named slot block")
	assertContains(t, result.Text, "{{end}}", "end")
}

// ---- VueToTemplate: child components ----------------------------------------

func TestVueToTemplate_ComponentZeroProps(t *testing.T) {
	src := `<template><my-card></my-card></template>`
	result := mustVueToTemplate(t, src, "Test")
	assertContains(t, result.Text, `{{template "my-card" .}}`, "component template call")
}

func TestVueToTemplate_ComponentWithPropsError(t *testing.T) {
	src := `<template><my-card :title="name"></my-card></template>`
	comp := mustParseVue(t, src)
	_, err := htmlc.VueToTemplate(comp.Template, "Test")
	assertError(t, err, "component with bound props")
}

// ---- VueToTemplate: custom directive ----------------------------------------

func TestVueToTemplate_CustomDirectiveError(t *testing.T) {
	src := `<template><div v-highlight="color">text</div></template>`
	comp := mustParseVue(t, src)
	_, err := htmlc.VueToTemplate(comp.Template, "Test")
	assertError(t, err, "custom directive")
}

// ---- VueToTemplate: style stripping -----------------------------------------

func TestVueToTemplate_StyleNotInOutput(t *testing.T) {
	src := `<template><p>{{ title }}</p></template>
<style>.p { color: red }</style>`
	result := mustVueToTemplate(t, src, "Test")
	if strings.Contains(result.Text, ".p { color: red }") {
		t.Error("style block should not appear in VueToTemplate output")
	}
}

// ---- TemplateToVue ----------------------------------------------------------

func TestTemplateToVue_SimpleIdent(t *testing.T) {
	result, err := htmlc.TemplateToVue(`<p>{{.name}}</p>`, "Test")
	assertNoError(t, err, "simple ident")
	assertContains(t, result.Text, "{{ name }}", "mustache output")
}

func TestTemplateToVue_DotPath(t *testing.T) {
	result, err := htmlc.TemplateToVue(`<p>{{.a.b}}</p>`, "Test")
	assertNoError(t, err, "dot-path")
	assertContains(t, result.Text, "{{ a.b }}", "dot-path output")
}

func TestTemplateToVue_PipelineError(t *testing.T) {
	_, err := htmlc.TemplateToVue(`<p>{{.items | len}}</p>`, "Test")
	assertError(t, err, "pipeline")
}

func TestTemplateToVue_If(t *testing.T) {
	result, err := htmlc.TemplateToVue(`{{if .cond}}<p>yes</p>{{end}}`, "Test")
	assertNoError(t, err, "if")
	assertContains(t, result.Text, `v-if="cond"`, "v-if attr")
}

func TestTemplateToVue_IfElse(t *testing.T) {
	result, err := htmlc.TemplateToVue(`{{if .cond}}<p>yes</p>{{else}}<p>no</p>{{end}}`, "Test")
	assertNoError(t, err, "if-else")
	assertContains(t, result.Text, `v-if="cond"`, "v-if")
	assertContains(t, result.Text, `v-else`, "v-else")
}

func TestTemplateToVue_Range(t *testing.T) {
	result, err := htmlc.TemplateToVue(`{{range .items}}item{{end}}`, "Test")
	assertNoError(t, err, "range")
	assertContains(t, result.Text, `v-for="item in items"`, "v-for attr")
}

func TestTemplateToVue_Block(t *testing.T) {
	result, err := htmlc.TemplateToVue(`{{block "default" .}}fallback{{end}}`, "Test")
	assertNoError(t, err, "block default")
	assertContains(t, result.Text, "<slot>", "slot element")
}

func TestTemplateToVue_BlockNamed(t *testing.T) {
	result, err := htmlc.TemplateToVue(`{{block "header" .}}Header{{end}}`, "Test")
	assertNoError(t, err, "block named")
	assertContains(t, result.Text, `<slot name="header">`, "named slot")
}

func TestTemplateToVue_Template(t *testing.T) {
	result, err := htmlc.TemplateToVue(`{{template "Card" .}}`, "Test")
	assertNoError(t, err, "template")
	assertContains(t, result.Text, "<Card />", "component element")
}

func TestTemplateToVue_With(t *testing.T) {
	_, err := htmlc.TemplateToVue(`{{with .x}}body{{end}}`, "Test")
	assertError(t, err, "with")
}

// ---- Round-trip tests -------------------------------------------------------

// TestRoundTrip_SimpleInterpolation verifies that a .vue component with simple
// text interpolation produces identical HTML output when rendered via htmlc and
// when rendered via the converted html/template.
func TestRoundTrip_SimpleInterpolation(t *testing.T) {
	src := `<template><p>{{ title }}</p></template>`
	comp, err := htmlc.ParseFile("test.vue", src)
	if err != nil {
		t.Fatal(err)
	}
	scope := map[string]any{"title": "Hello World"}

	// Render via htmlc.
	vueOut, err := htmlc.RenderString(comp, scope)
	if err != nil {
		t.Fatalf("htmlc render: %v", err)
	}

	// Convert to Go template and render.
	result, err := htmlc.VueToTemplate(comp.Template, "RoundTrip")
	if err != nil {
		t.Fatalf("VueToTemplate: %v", err)
	}
	tmpl, err := htmltemplate.New("").Parse(result.Text)
	if err != nil {
		t.Fatalf("html/template parse: %v", err)
	}
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "RoundTrip", scope); err != nil {
		t.Fatalf("html/template execute: %v", err)
	}
	tmplOut := buf.String()

	if vueOut != tmplOut {
		t.Errorf("round-trip mismatch:\nvue:  %q\ntmpl: %q", vueOut, tmplOut)
	}
}

// TestRoundTrip_DotPath verifies round-trip equivalence for dot-path expressions.
func TestRoundTrip_DotPath(t *testing.T) {
	src := `<template><p>{{ post.title }}</p></template>`
	comp, err := htmlc.ParseFile("test.vue", src)
	if err != nil {
		t.Fatal(err)
	}
	scope := map[string]any{"post": map[string]any{"title": "My Post"}}

	vueOut, err := htmlc.RenderString(comp, scope)
	if err != nil {
		t.Fatalf("htmlc render: %v", err)
	}

	result, err := htmlc.VueToTemplate(comp.Template, "RoundTrip")
	if err != nil {
		t.Fatalf("VueToTemplate: %v", err)
	}
	tmpl, err := htmltemplate.New("").Parse(result.Text)
	if err != nil {
		t.Fatalf("html/template parse: %v", err)
	}
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "RoundTrip", scope); err != nil {
		t.Fatalf("html/template execute: %v", err)
	}
	tmplOut := buf.String()

	if vueOut != tmplOut {
		t.Errorf("round-trip mismatch:\nvue:  %q\ntmpl: %q", vueOut, tmplOut)
	}
}

// TestRoundTrip_VFor verifies round-trip equivalence for v-for loops.
func TestRoundTrip_VFor(t *testing.T) {
	src := `<template><ul><li v-for="item in items">{{ item }}</li></ul></template>`
	comp, err := htmlc.ParseFile("test.vue", src)
	if err != nil {
		t.Fatal(err)
	}
	scope := map[string]any{"items": []any{"alpha", "beta", "gamma"}}

	vueOut, err := htmlc.RenderString(comp, scope)
	if err != nil {
		t.Fatalf("htmlc render: %v", err)
	}

	result, err := htmlc.VueToTemplate(comp.Template, "RoundTrip")
	if err != nil {
		t.Fatalf("VueToTemplate: %v", err)
	}
	tmpl, err := htmltemplate.New("").Parse(result.Text)
	if err != nil {
		t.Fatalf("html/template parse: %v", err)
	}
	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "RoundTrip", scope); err != nil {
		t.Fatalf("html/template execute: %v", err)
	}
	tmplOut := buf.String()

	if vueOut != tmplOut {
		t.Errorf("round-trip mismatch:\nvue:  %q\ntmpl: %q", vueOut, tmplOut)
	}
}

// ---- helpers (package-private) ----------------------------------------------

func isConversionError(err error, target **htmlc.ConversionError) bool {
	if err == nil {
		return false
	}
	if ce, ok := err.(*htmlc.ConversionError); ok {
		if target != nil {
			*target = ce
		}
		return true
	}
	return false
}
