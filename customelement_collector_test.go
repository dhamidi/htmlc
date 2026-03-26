package htmlc

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCustomElementCollector_Deduplication(t *testing.T) {
	c := NewCustomElementCollector()
	script := "export default class extends HTMLElement {}"
	c.Add("ui-date-picker", script)
	c.Add("ui-date-picker", script) // same tag, same script

	if c.Len() != 1 {
		t.Errorf("Len() = %d, want 1", c.Len())
	}

	sfs := c.ScriptsFS()
	entries, err := fs.ReadDir(sfs, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("ScriptsFS has %d files, want 1", len(entries))
	}
}

func TestCustomElementCollector_MultipleScripts(t *testing.T) {
	c := NewCustomElementCollector()
	script1 := "export default class DatePicker extends HTMLElement {}"
	script2 := "export default class ShapeCanvas extends HTMLElement {}"
	c.Add("ui-date-picker", script1)
	c.Add("widgets-shape-canvas", script2)

	if c.Len() != 2 {
		t.Errorf("Len() = %d, want 2", c.Len())
	}

	sfs := c.ScriptsFS()
	entries, err := fs.ReadDir(sfs, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("ScriptsFS has %d files, want 2", len(entries))
	}

	// Verify each file is a .js file with the correct content.
	hash1 := contentHash(script1)
	hash2 := contentHash(script2)
	// Verify file names include both tag and hash, and content is correct.
	data1, err := fs.ReadFile(sfs, "ui-date-picker."+hash1+".js")
	if err != nil {
		t.Errorf("ReadFile ui-date-picker.%s.js: %v", hash1, err)
	}
	if string(data1) != script1 {
		t.Errorf("file content for ui-date-picker = %q, want %q", data1, script1)
	}
	data2, err := fs.ReadFile(sfs, "widgets-shape-canvas."+hash2+".js")
	if err != nil {
		t.Errorf("ReadFile widgets-shape-canvas.%s.js: %v", hash2, err)
	}
	if string(data2) != script2 {
		t.Errorf("file content for widgets-shape-canvas = %q, want %q", data2, script2)
	}
}

func TestCustomElementCollector_ImportMapJSON(t *testing.T) {
	c := NewCustomElementCollector()
	script1 := "export default class DatePicker extends HTMLElement {}"
	script2 := "export default class ShapeCanvas extends HTMLElement {}"
	c.Add("ui-date-picker", script1)
	c.Add("widgets-shape-canvas", script2)

	raw := c.ImportMapJSON("/scripts/")
	var result struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("ImportMapJSON not valid JSON: %v\ngot: %s", err, raw)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports has %d entries, want 2", len(result.Imports))
	}
	hash1 := contentHash(script1)
	hash2 := contentHash(script2)
	want1 := "/scripts/ui-date-picker." + hash1 + ".js"
	if result.Imports["ui-date-picker"] != want1 {
		t.Errorf("imports[ui-date-picker] = %q, want %q", result.Imports["ui-date-picker"], want1)
	}
	want2 := "/scripts/widgets-shape-canvas." + hash2 + ".js"
	if result.Imports["widgets-shape-canvas"] != want2 {
		t.Errorf("imports[widgets-shape-canvas] = %q, want %q", result.Imports["widgets-shape-canvas"], want2)
	}
	// Verify URL prefix is used.
	for _, url := range result.Imports {
		if !strings.HasPrefix(url, "/scripts/") {
			t.Errorf("URL %q does not start with /scripts/", url)
		}
	}
}

func TestCustomElementCollector_Empty(t *testing.T) {
	c := NewCustomElementCollector()

	sfs := c.ScriptsFS()
	entries, err := fs.ReadDir(sfs, ".")
	if err != nil {
		t.Fatalf("ReadDir on empty ScriptsFS: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("empty ScriptsFS has %d files, want 0", len(entries))
	}

	raw := c.ImportMapJSON("/scripts/")
	var result struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("ImportMapJSON not valid JSON: %v", err)
	}
	if len(result.Imports) != 0 {
		t.Errorf("empty collector imports has %d entries, want 0", len(result.Imports))
	}
}

func TestCustomElementCollector_IndexJS_Empty(t *testing.T) {
	c := NewCustomElementCollector()
	got := c.IndexJS()
	if got != "" {
		t.Errorf("IndexJS on empty collector = %q, want \"\"", got)
	}
}

func TestCustomElementCollector_IndexJS_Single(t *testing.T) {
	c := NewCustomElementCollector()
	script := "export default class DatePicker extends HTMLElement {}"
	c.Add("ui-date-picker", script)

	hash := contentHash(script)
	got := c.IndexJS()
	want := `import "./ui-date-picker.` + hash + `.js"` + "\n"
	if got != want {
		t.Errorf("IndexJS = %q, want %q", got, want)
	}
}

func TestCustomElementCollector_IndexJS_TwoScripts(t *testing.T) {
	c := NewCustomElementCollector()
	script1 := "export default class DatePicker extends HTMLElement {}"
	script2 := "export default class ShapeCanvas extends HTMLElement {}"
	c.Add("ui-date-picker", script1)
	c.Add("widgets-shape-canvas", script2)

	hash1 := contentHash(script1)
	hash2 := contentHash(script2)
	got := c.IndexJS()
	want := `import "./ui-date-picker.` + hash1 + `.js"` + "\n" +
		`import "./widgets-shape-canvas.` + hash2 + `.js"` + "\n"
	if got != want {
		t.Errorf("IndexJS = %q, want %q", got, want)
	}
}

func TestCustomElementCollector_IndexJS_DedupByHash(t *testing.T) {
	c := NewCustomElementCollector()
	script := "export default class DatePicker extends HTMLElement {}"
	c.Add("ui-date-picker", script)
	c.Add("alt-date-picker", script) // same content, different tag

	hash := contentHash(script)
	got := c.IndexJS()
	// The first tag ("ui-date-picker") is used in the filename.
	want := `import "./ui-date-picker.` + hash + `.js"` + "\n"
	if got != want {
		t.Errorf("IndexJS = %q, want %q (expected dedup)", got, want)
	}
}

func TestRenderWithCollector_CEComponent(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement>
export default class DatePicker extends HTMLElement {}
</script>
`),
		},
	}
	e, err := New(Options{ComponentDir: ".", FS: fsys})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	collector := NewCustomElementCollector()
	_, err = e.RenderWithCollector(context.Background(), "DatePicker", nil, collector)
	if err != nil {
		t.Fatalf("RenderWithCollector: %v", err)
	}
	if collector.Len() != 1 {
		t.Errorf("collector.Len() = %d, want 1", collector.Len())
	}
}

func TestRenderWithCollector_NonCEComponent(t *testing.T) {
	fsys := fstest.MapFS{
		"Normal.vue": &fstest.MapFile{
			Data: []byte(`<template><div>hello</div></template>`),
		},
	}
	e, err := New(Options{ComponentDir: ".", FS: fsys})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	collector := NewCustomElementCollector()
	_, err = e.RenderWithCollector(context.Background(), "Normal", nil, collector)
	if err != nil {
		t.Fatalf("RenderWithCollector: %v", err)
	}
	if collector.Len() != 0 {
		t.Errorf("collector.Len() = %d, want 0 for non-CE component", collector.Len())
	}
}
