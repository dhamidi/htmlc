// Package htmlctest provides helpers for testing htmlc components.
//
// It is intended for use in *_test.go files. The helpers reduce the boilerplate
// of creating an in-memory filesystem, wiring up an Engine, and asserting on
// rendered HTML output.
//
// Example:
//
//	func TestGreeting(t *testing.T) {
//	    e := htmlctest.NewEngine(t, map[string]string{
//	        "Greeting.vue": `<template><p>Hello {{ name }}!</p></template>`,
//	    })
//	    htmlctest.AssertFragment(t, e, "Greeting",
//	        map[string]any{"name": "World"},
//	        "<p>Hello World!</p>",
//	    )
//	}
package htmlctest

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dhamidi/htmlc"
)

// NewEngine creates a test Engine backed by an in-memory filesystem built from
// files. The keys of files are file paths (e.g. "Button.vue") and the values
// are their contents.
//
// If any option is provided via opts it replaces the default Options; the FS
// and ComponentDir fields are always overridden by the in-memory filesystem.
//
// t.Cleanup is registered to satisfy the testing.TB contract; no goroutines
// are started so there is nothing to clean up at this time — the hook is
// reserved for future use.
func NewEngine(t testing.TB, files map[string]string, opts ...htmlc.Options) *htmlc.Engine {
	t.Helper()

	mapFS := make(fstest.MapFS, len(files))
	for name, content := range files {
		mapFS[name] = &fstest.MapFile{Data: []byte(content)}
	}

	var o htmlc.Options
	if len(opts) > 0 {
		o = opts[0]
	}
	o.FS = mapFS
	o.ComponentDir = "."

	e, err := htmlc.New(o)
	if err != nil {
		t.Fatalf("htmlctest.NewEngine: %v", err)
	}
	t.Cleanup(func() {}) // reserved for future cleanup
	return e
}

// normaliseWS collapses runs of whitespace (including newlines) into single
// spaces and trims the result. This makes HTML comparisons robust to
// formatting differences.
func normaliseWS(s string) string {
	var sb strings.Builder
	inSpace := false
	for _, ch := range s {
		switch ch {
		case ' ', '\t', '\r', '\n':
			if !inSpace {
				sb.WriteByte(' ')
				inSpace = true
			}
		default:
			inSpace = false
			sb.WriteRune(ch)
		}
	}
	return strings.TrimSpace(sb.String())
}

// AssertRendersHTML asserts that rendering name as a full HTML page with data
// produces HTML equal to want (after normalising whitespace).
func AssertRendersHTML(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string) {
	t.Helper()
	got, err := e.RenderPageString(name, data)
	if err != nil {
		t.Fatalf("AssertRendersHTML: RenderPage(%q): %v", name, err)
	}
	if normaliseWS(got) != normaliseWS(want) {
		t.Errorf("AssertRendersHTML(%q):\ngot:  %s\nwant: %s", name, got, want)
	}
}

// AssertFragment asserts that rendering name as an HTML fragment with data
// produces HTML equal to want (after normalising whitespace).
func AssertFragment(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string) {
	t.Helper()
	got, err := e.RenderFragmentString(name, data)
	if err != nil {
		t.Fatalf("AssertFragment: RenderFragment(%q): %v", name, err)
	}
	if normaliseWS(got) != normaliseWS(want) {
		t.Errorf("AssertFragment(%q):\ngot:  %s\nwant: %s", name, got, want)
	}
}
