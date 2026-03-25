package htmlc

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestRenderCustomElement_LightDOM(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement>
export default {}
</script>
`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "ui/DatePicker.vue")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := RenderString(c, nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if !strings.HasPrefix(out, "<ui-date-picker>") {
		t.Errorf("expected output to start with <ui-date-picker>, got: %q", out)
	}
	if !strings.HasSuffix(out, "</ui-date-picker>") {
		t.Errorf("expected output to end with </ui-date-picker>, got: %q", out)
	}
	if strings.Contains(out, "shadowrootmode") {
		t.Errorf("light DOM should not contain shadowrootmode, got: %q", out)
	}
}

func TestRenderCustomElement_OpenShadowDOM(t *testing.T) {
	fsys := fstest.MapFS{
		"widgets/ShapeCanvas.vue": &fstest.MapFile{
			Data: []byte(`<template><canvas>canvas</canvas></template>
<script customelement shadowdom>
export default {}
</script>
`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "widgets/ShapeCanvas.vue")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := RenderString(c, nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if !strings.HasPrefix(out, "<widgets-shape-canvas>") {
		t.Errorf("expected output to start with <widgets-shape-canvas>, got: %q", out)
	}
	if !strings.HasSuffix(out, "</widgets-shape-canvas>") {
		t.Errorf("expected output to end with </widgets-shape-canvas>, got: %q", out)
	}
	if !strings.Contains(out, `<template shadowrootmode="open">`) {
		t.Errorf("expected shadowrootmode=open, got: %q", out)
	}
}

func TestRenderCustomElement_ClosedShadowDOM(t *testing.T) {
	fsys := fstest.MapFS{
		"widgets/ShapeCanvas.vue": &fstest.MapFile{
			Data: []byte(`<template><canvas>canvas</canvas></template>
<script customelement shadowdom="closed">
export default {}
</script>
`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "widgets/ShapeCanvas.vue")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := RenderString(c, nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if !strings.Contains(out, `shadowrootmode="closed"`) {
		t.Errorf("expected shadowrootmode=closed, got: %q", out)
	}
}

func TestRenderCustomElement_NormalComponent_Unaffected(t *testing.T) {
	fsys := fstest.MapFS{
		"Normal.vue": &fstest.MapFile{
			Data: []byte(`<template><div>hello</div></template>`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "Normal.vue")
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := RenderString(c, nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if strings.Contains(out, "<normal>") || strings.Contains(out, "</normal>") {
		t.Errorf("normal component should not be wrapped, got: %q", out)
	}
	if !strings.Contains(out, "<div>hello</div>") {
		t.Errorf("expected <div>hello</div> in output, got: %q", out)
	}
}

func TestRenderCustomElement_NestedInPage(t *testing.T) {
	files := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><span>date</span></template>
<script customelement>
export default {}
</script>
`),
		},
		"Page.vue": &fstest.MapFile{
			Data: []byte(`<template><div><DatePicker /></div></template>`),
		},
	}
	opts := Options{
		ComponentDir: ".",
		FS:           files,
	}
	e, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := e.RenderFragmentString("Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	// The custom element wrapper should appear exactly once
	count := strings.Count(out, "<ui-date-picker>")
	if count != 1 {
		t.Errorf("expected <ui-date-picker> wrapper exactly once, got %d times in: %q", count, out)
	}
	count = strings.Count(out, "</ui-date-picker>")
	if count != 1 {
		t.Errorf("expected </ui-date-picker> closing tag exactly once, got %d times in: %q", count, out)
	}
}
