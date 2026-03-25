package htmlc

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func TestIntegration_CustomElementCollection(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement>
export default class DatePicker extends HTMLElement {}
</script>
`),
		},
		"widgets/ShapeCanvas.vue": &fstest.MapFile{
			Data: []byte(`<template><canvas>canvas</canvas></template>
<script customelement shadowdom>
export default class ShapeCanvas extends HTMLElement {}
</script>
`),
		},
		"Normal.vue": &fstest.MapFile{
			Data: []byte(`<template><p>plain</p></template>`),
		},
		// Page includes DatePicker twice and ShapeCanvas once and Normal once.
		"Page.vue": &fstest.MapFile{
			Data: []byte(`<template>
<div>
  <DatePicker />
  <ShapeCanvas />
  <Normal />
  <DatePicker />
</div>
</template>`),
		},
	}

	e, err := New(Options{ComponentDir: ".", FS: fsys})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	collector := NewCustomElementCollector()
	out, err := e.RenderWithCollector(context.Background(), "Page", nil, collector)
	if err != nil {
		t.Fatalf("RenderWithCollector: %v", err)
	}

	// Verify rendered output contains both CE wrappers.
	if !strings.Contains(out, "<ui-date-picker>") {
		t.Errorf("expected <ui-date-picker> in output, got: %s", out)
	}
	if !strings.Contains(out, "<widgets-shape-canvas>") {
		t.Errorf("expected <widgets-shape-canvas> in output, got: %s", out)
	}

	// 2. Collector should have exactly two unique scripts (DatePicker and ShapeCanvas).
	if collector.Len() != 2 {
		t.Errorf("collector.Len() = %d, want 2 (DatePicker and ShapeCanvas, deduplicated)", collector.Len())
	}

	// 3. ScriptsFS should contain exactly two .js files.
	sfs := collector.ScriptsFS()
	entries, err := fs.ReadDir(sfs, ".")
	if err != nil {
		t.Fatalf("ReadDir ScriptsFS: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("ScriptsFS has %d files, want 2", len(entries))
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".js") {
			t.Errorf("ScriptsFS file %q does not end in .js", e.Name())
		}
	}

	// 4. ImportMapJSON should reference both files.
	raw := collector.ImportMapJSON("/scripts/")
	var importMap struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(raw), &importMap); err != nil {
		t.Fatalf("ImportMapJSON not valid JSON: %v\ngot: %s", err, raw)
	}
	if len(importMap.Imports) != 2 {
		t.Errorf("import map has %d entries, want 2; got: %s", len(importMap.Imports), raw)
	}
	for tag, url := range importMap.Imports {
		if !strings.HasPrefix(url, "/scripts/") {
			t.Errorf("import map entry %q URL %q does not start with /scripts/", tag, url)
		}
		if !strings.HasSuffix(url, ".js") {
			t.Errorf("import map entry %q URL %q does not end with .js", tag, url)
		}
		// Verify the referenced file exists in ScriptsFS.
		hash := strings.TrimPrefix(url, "/scripts/")
		if _, err := fs.Stat(sfs, hash); err != nil {
			t.Errorf("import map references %q but file not in ScriptsFS: %v", hash, err)
		}
	}
}
