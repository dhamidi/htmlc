package htmlc

import (
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"
)

func TestCollectCustomElements_Basic(t *testing.T) {
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement>
export default class DatePicker extends HTMLElement {}
</script>
`),
		},
		"Normal.vue": &fstest.MapFile{
			Data: []byte(`<template><div>hello</div></template>`),
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
	if collector.Len() != 1 {
		t.Errorf("collector.Len() = %d, want 1", collector.Len())
	}

	raw := collector.ImportMapJSON("/s/")
	var result struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("ImportMapJSON not valid JSON: %v\ngot: %s", err, raw)
	}
	if _, ok := result.Imports["ui-date-picker"]; !ok {
		t.Errorf("imports missing 'ui-date-picker'; got %v", result.Imports)
	}
	if url := result.Imports["ui-date-picker"]; !strings.HasPrefix(url, "/s/") {
		t.Errorf("URL %q does not start with /s/", url)
	}
}

func TestCollectCustomElements_MultipleAndNonCE(t *testing.T) {
	script1 := "export default class DatePicker extends HTMLElement {}"
	script2 := "export default class ShapeCanvas extends HTMLElement {}"
	fsys := fstest.MapFS{
		"ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte("<template><div>picker</div></template>\n<script customelement>\n" + script1 + "\n</script>\n"),
		},
		"widgets/ShapeCanvas.vue": &fstest.MapFile{
			Data: []byte("<template><canvas></canvas></template>\n<script customelement>\n" + script2 + "\n</script>\n"),
		},
		"Plain.vue": &fstest.MapFile{
			Data: []byte(`<template><p>plain</p></template>`),
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
	// Only the two custom-element components should be collected.
	if collector.Len() != 2 {
		t.Errorf("collector.Len() = %d, want 2", collector.Len())
	}

	raw := collector.ImportMapJSON("/s/")
	var result struct {
		Imports map[string]string `json:"imports"`
	}
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		t.Fatalf("ImportMapJSON not valid JSON: %v\ngot: %s", err, raw)
	}
	if len(result.Imports) != 2 {
		t.Errorf("imports has %d entries, want 2; got %v", len(result.Imports), result.Imports)
	}
	for _, tag := range []string{"ui-date-picker", "widgets-shape-canvas"} {
		if _, ok := result.Imports[tag]; !ok {
			t.Errorf("imports missing %q; got %v", tag, result.Imports)
		}
	}
	// Plain component must not appear.
	if _, ok := result.Imports["plain"]; ok {
		t.Errorf("imports unexpectedly contains 'plain'")
	}
}
