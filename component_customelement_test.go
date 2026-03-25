package htmlc

import (
	"strings"
	"testing"
	"testing/fstest"
)

func parseFileFromMapFS(fsys fstest.MapFS, path string) (*Component, error) {
	data, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer data.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, readErr := data.Read(buf)
		if n > 0 {
			sb.Write(buf[:n])
		}
		if readErr != nil {
			break
		}
	}
	return ParseFile(path, sb.String())
}

func TestCustomElement_LightDOM(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement>
export default { name: 'date-picker' }
</script>
`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "ui/DatePicker.vue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.CustomElementScript == "" {
		t.Error("CustomElementScript should be set")
	}
	if c.ShadowDOMMode != "" {
		t.Errorf("ShadowDOMMode = %q, want %q", c.ShadowDOMMode, "")
	}
	if c.CustomElementTag != "ui-date-picker" {
		t.Errorf("CustomElementTag = %q, want %q", c.CustomElementTag, "ui-date-picker")
	}
}

func TestCustomElement_OpenShadowDOM(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement shadowdom>
export default {}
</script>
`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "ui/DatePicker.vue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ShadowDOMMode != "open" {
		t.Errorf("ShadowDOMMode = %q, want %q", c.ShadowDOMMode, "open")
	}
}

func TestCustomElement_ExplicitOpenShadowDOM(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement shadowdom="open">
export default {}
</script>
`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "ui/DatePicker.vue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ShadowDOMMode != "open" {
		t.Errorf("ShadowDOMMode = %q, want %q", c.ShadowDOMMode, "open")
	}
}

func TestCustomElement_ClosedShadowDOM(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement shadowdom="closed">
export default {}
</script>
`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "ui/DatePicker.vue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.ShadowDOMMode != "closed" {
		t.Errorf("ShadowDOMMode = %q, want %q", c.ShadowDOMMode, "closed")
	}
}

func TestCustomElement_ConflictWithScript(t *testing.T) {
	src := `<template><div>x</div></template>
<script>
console.log('plain script')
</script>
<script customelement>
export default {}
</script>
`
	_, err := ParseFile("Conflict.vue", src)
	if err == nil {
		t.Fatal("expected error for combined <script> and <script customelement>, got nil")
	}
}

func TestCustomElement_ConflictWithScriptSetup(t *testing.T) {
	src := `<template><div>x</div></template>
<script setup>
const x = 1
</script>
<script customelement>
export default {}
</script>
`
	_, err := ParseFile("Conflict.vue", src)
	if err == nil {
		t.Fatal("expected error for combined <script setup> and <script customelement>, got nil")
	}
}

func TestCustomElement_TagDerivation(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"ui/DatePicker.vue", "ui-date-picker"},
		{"widgets/ShapeCanvas.vue", "widgets-shape-canvas"},
		{"XmlParser.vue", "xml-parser"},
	}

	scriptContent := `export default {}`
	templateContent := `<template><div>x</div></template>`

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			src := templateContent + "\n<script customelement>\n" + scriptContent + "\n</script>\n"
			c, err := ParseFile(tc.path, src)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.CustomElementTag != tc.want {
				t.Errorf("CustomElementTag = %q, want %q", c.CustomElementTag, tc.want)
			}
		})
	}
}

func TestCustomElement_SingleWordWarning(t *testing.T) {
	src := `<template><div>btn</div></template>
<script customelement>
export default {}
</script>
`
	c, err := ParseFile("Button.vue", src)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.CustomElementTag != "button" {
		t.Errorf("CustomElementTag = %q, want %q", c.CustomElementTag, "button")
	}
	found := false
	for _, w := range c.Warnings {
		if strings.Contains(w, "no hyphen") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected a warning about missing hyphen, got warnings: %v", c.Warnings)
	}
}

func TestCustomElement_NoOpForNormalComponents(t *testing.T) {
	fsys := fstest.MapFS{
		"Normal.vue": &fstest.MapFile{
			Data: []byte(`<template><div>hello</div></template>
<style scoped>
div { color: red; }
</style>
`),
		},
	}
	c, err := parseFileFromMapFS(fsys, "Normal.vue")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.CustomElementScript != "" {
		t.Errorf("CustomElementScript = %q, want empty", c.CustomElementScript)
	}
	if c.CustomElementTag != "" {
		t.Errorf("CustomElementTag = %q, want empty", c.CustomElementTag)
	}
	if c.ShadowDOMMode != "" {
		t.Errorf("ShadowDOMMode = %q, want empty", c.ShadowDOMMode)
	}
}

func TestDeriveCustomElementTag(t *testing.T) {
	cases := []struct {
		path string
		want string
	}{
		{"Button.vue", "button"},
		{"ui/DatePicker.vue", "ui-date-picker"},
		{"widgets/ShapeCanvas.vue", "widgets-shape-canvas"},
		{"XmlParser.vue", "xml-parser"},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			got := deriveCustomElementTag(tc.path)
			if got != tc.want {
				t.Errorf("deriveCustomElementTag(%q) = %q, want %q", tc.path, got, tc.want)
			}
		})
	}
}
