package htmlc

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

// TestImportMapFunc_WithCollector verifies that the importMap template function
// returns valid import-map JSON when a populated collector is attached.
func TestImportMapFunc_WithCollector(t *testing.T) {
	fsys := fstest.MapFS{
		"Page.vue": &fstest.MapFile{
			Data: []byte(`<template><script type="importmap">{{ importMap("/scripts/") }}</script></template>`),
		},
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

	collector, err := e.CollectCustomElements()
	if err != nil {
		t.Fatalf("CollectCustomElements: %v", err)
	}

	out, err := e.RenderWithCollector(context.Background(), "Page", nil, collector)
	if err != nil {
		t.Fatalf("RenderWithCollector: %v", err)
	}

	// The importMap call should emit JSON inside the script tag.
	if !strings.Contains(out, `"imports"`) {
		t.Errorf("expected output to contain importmap JSON with 'imports' key; got: %q", out)
	}

	// Extract the JSON from inside the script tag.
	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start == -1 || end == -1 || end < start {
		t.Fatalf("could not locate JSON object in output: %q", out)
	}
	raw := out[start : end+1]

	var result struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("importMap output is not valid JSON: %v\ngot: %s", err, raw)
	}
	if _, ok := result.Imports["ui-date-picker"]; !ok {
		t.Errorf("imports missing 'ui-date-picker'; got %v", result.Imports)
	}
	if url := result.Imports["ui-date-picker"]; !strings.HasPrefix(url, "/scripts/") {
		t.Errorf("URL %q does not start with /scripts/", url)
	}
}

// TestImportMapFunc_NilCollector verifies that importMap returns "" (not an error)
// when no collector is attached to the render.
func TestImportMapFunc_NilCollector(t *testing.T) {
	fsys := fstest.MapFS{
		"Page.vue": &fstest.MapFile{
			Data: []byte(`<template><div>{{ importMap("/scripts/") }}</div></template>`),
		},
	}
	e, err := New(Options{ComponentDir: ".", FS: fsys})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Pass nil collector — importMap should silently return "".
	out, err := e.RenderWithCollector(context.Background(), "Page", nil, nil)
	if err != nil {
		t.Fatalf("RenderWithCollector: %v", err)
	}

	// The interpolated value should be empty — no JSON, no error marker.
	if strings.Contains(out, "imports") {
		t.Errorf("expected no imports JSON with nil collector; got: %q", out)
	}
}

// TestImportMapFunc_EmptyCollector verifies that importMap returns valid (but
// empty) import-map JSON when the collector has no entries.
func TestImportMapFunc_EmptyCollector(t *testing.T) {
	fsys := fstest.MapFS{
		"Page.vue": &fstest.MapFile{
			Data: []byte(`<template><script type="importmap">{{ importMap("/scripts/") }}</script></template>`),
		},
	}
	e, err := New(Options{ComponentDir: ".", FS: fsys})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// An empty collector — no custom elements added.
	collector := &CustomElementCollector{}

	out, err := e.RenderWithCollector(context.Background(), "Page", nil, collector)
	if err != nil {
		t.Fatalf("RenderWithCollector: %v", err)
	}

	// Extract the JSON from inside the script tag.
	start := strings.Index(out, "{")
	end := strings.LastIndex(out, "}")
	if start == -1 || end == -1 || end < start {
		t.Fatalf("could not locate JSON object in output: %q", out)
	}
	raw := out[start : end+1]

	// The output should be a valid JSON object (possibly {"imports":{}}).
	var result map[string]any
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("importMap with empty collector is not valid JSON: %v\ngot: %q", err, raw)
	}
}

// TestImportMapFunc_PropagatedToChildren verifies that importMap is also
// available inside child components rendered from a page.
func TestImportMapFunc_PropagatedToChildren(t *testing.T) {
	fsys := fstest.MapFS{
		"Page.vue": &fstest.MapFile{
			Data: []byte(`<template><Header /></template>`),
		},
		"Header.vue": &fstest.MapFile{
			Data: []byte(`<template><script type="importmap">{{ importMap("/s/") }}</script></template>`),
		},
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

	collector, err := e.CollectCustomElements()
	if err != nil {
		t.Fatalf("CollectCustomElements: %v", err)
	}

	out, err := e.RenderWithCollector(context.Background(), "Page", nil, collector)
	if err != nil {
		t.Fatalf("RenderWithCollector: %v", err)
	}

	if !strings.Contains(out, `"imports"`) {
		t.Errorf("expected importmap JSON in child component output; got: %q", out)
	}
	if !strings.Contains(out, "ui-date-picker") {
		t.Errorf("expected 'ui-date-picker' in child importmap output; got: %q", out)
	}
}
