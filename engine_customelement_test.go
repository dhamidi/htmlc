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

// TestValidateAll_RootLevelCustomElement_HyphenGap reproduces the RFC 014
// §1.2 gap: a root-level component's tag is derived twice — once inside
// ParseFile against the full, ComponentDir-prefixed path (which typically
// contains a hyphen from the directory separator), and again inside
// discoverInto against the path relative to ComponentDir (which, for a
// root-level file, has no directory segment left to contribute a hyphen).
// The ParseFile-time check passes on the first tag, but the final,
// effective tag is invalid; ValidateAll must catch it.
func TestValidateAll_RootLevelCustomElement_HyphenGap(t *testing.T) {
	fsys := fstest.MapFS{
		"components/Accordion.vue": &fstest.MapFile{
			Data: []byte(`<template><div>accordion</div></template>
<script customelement>
export default class Accordion extends HTMLElement {}
</script>
`),
		},
	}
	e, err := New(Options{ComponentDir: "components", FS: fsys})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := e.ValidateAll()
	var found bool
	for _, ve := range errs {
		if strings.Contains(ve.Message, "no hyphen") {
			found = true
			if !strings.Contains(ve.Message, `"accordion"`) {
				t.Errorf("warning does not mention final tag %q: %s", "accordion", ve.Message)
			}
		}
	}
	if !found {
		t.Errorf("expected ValidateAll to report a missing-hyphen warning for the "+
			"final tag %q, got errors: %v", "accordion", errs)
	}

	// The final, effective tag really is the invalid hyphen-less one.
	e.mu.RLock()
	entry := e.entries["Accordion"]
	e.mu.RUnlock()
	if entry == nil {
		t.Fatal("entry for Accordion not found")
	}
	if entry.comp.CustomElementTag != "accordion" {
		t.Errorf("CustomElementTag = %q, want %q", entry.comp.CustomElementTag, "accordion")
	}
}

// TestValidateAll_NestedCustomElement_NoSpuriousHyphenWarning covers a
// component nested in a subdirectory of ComponentDir: both the ParseFile-time
// tag (derived from the full path) and the final, ComponentDir-relative tag
// contain a hyphen, so ValidateAll must not report anything about it.
func TestValidateAll_NestedCustomElement_NoSpuriousHyphenWarning(t *testing.T) {
	fsys := fstest.MapFS{
		"components/ui/DatePicker.vue": &fstest.MapFile{
			Data: []byte(`<template><div>picker</div></template>
<script customelement>
export default class DatePicker extends HTMLElement {}
</script>
`),
		},
	}
	e, err := New(Options{ComponentDir: "components", FS: fsys})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := e.ValidateAll()
	for _, ve := range errs {
		if strings.Contains(ve.Message, "no hyphen") {
			t.Errorf("unexpected missing-hyphen warning: %s", ve.Message)
		}
	}

	e.mu.RLock()
	entry := e.entries["DatePicker"]
	e.mu.RUnlock()
	if entry == nil {
		t.Fatal("entry for DatePicker not found")
	}
	if entry.comp.CustomElementTag != "ui-date-picker" {
		t.Errorf("CustomElementTag = %q, want %q", entry.comp.CustomElementTag, "ui-date-picker")
	}
	if len(entry.comp.Warnings) != 0 {
		t.Errorf("expected no warnings, got: %v", entry.comp.Warnings)
	}
}

// TestValidateAll_ComponentDirDot_NoDoubleWarning covers the case where
// re-derivation in discoverInto is a no-op (ComponentDir == ".", so the
// relative path equals the path already used by ParseFile): a genuinely
// hyphen-less root-level tag must be reported exactly once, not twice.
func TestValidateAll_ComponentDirDot_NoDoubleWarning(t *testing.T) {
	fsys := fstest.MapFS{
		"Button.vue": &fstest.MapFile{
			Data: []byte(`<template><div>btn</div></template>
<script customelement>
export default class Button extends HTMLElement {}
</script>
`),
		},
	}
	e, err := New(Options{ComponentDir: ".", FS: fsys})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := e.ValidateAll()
	count := 0
	for _, ve := range errs {
		if strings.Contains(ve.Message, "no hyphen") {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 missing-hyphen warning, got %d: %v", count, errs)
	}
}
