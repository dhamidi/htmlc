package htmlc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"testing/fstest"
)

// newMapFSEngine builds an Engine from an in-memory MapFS fixture set.
// ComponentDir must match the root key prefix used in the MapFS.
func newMapFSEngine(t *testing.T, files fstest.MapFS, opts Options) *Engine {
	t.Helper()
	opts.FS = files
	if opts.ComponentDir == "" {
		opts.ComponentDir = "."
	}
	e, err := New(opts)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

// TestComponentPathInError verifies RenderError.ComponentPath is populated for
// two-level and three-level component trees.
func TestComponentPathInError(t *testing.T) {
	t.Run("two-level", func(t *testing.T) {
		fsys := fstest.MapFS{
			"Root.vue":  &fstest.MapFile{Data: []byte(`<template><div><Child /></div></template>`)},
			"Child.vue": &fstest.MapFile{Data: []byte(`<template><span>{{ missing.field }}</span></template>`)},
		}
		e := newMapFSEngine(t, fsys, Options{})
		err := e.RenderPage(context.Background(), io.Discard, "Root", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var re *RenderError
		if !errors.As(err, &re) {
			t.Fatalf("expected *RenderError, got %T: %v", err, err)
		}
		if len(re.ComponentPath) != 2 {
			t.Fatalf("ComponentPath length = %d, want 2; path = %v", len(re.ComponentPath), re.ComponentPath)
		}
		if re.ComponentPath[0] != "Root" {
			t.Errorf("ComponentPath[0] = %q, want %q", re.ComponentPath[0], "Root")
		}
		if re.ComponentPath[1] != "child" {
			t.Errorf("ComponentPath[1] = %q, want %q", re.ComponentPath[1], "child")
		}
	})

	t.Run("three-level", func(t *testing.T) {
		fsys := fstest.MapFS{
			"Root.vue":    &fstest.MapFile{Data: []byte(`<template><div><Mid /></div></template>`)},
			"Mid.vue":     &fstest.MapFile{Data: []byte(`<template><section><Leaf /></section></template>`)},
			"Leaf.vue":    &fstest.MapFile{Data: []byte(`<template><p>{{ missing.x }}</p></template>`)},
		}
		e := newMapFSEngine(t, fsys, Options{})
		err := e.RenderPage(context.Background(), io.Discard, "Root", nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var re *RenderError
		if !errors.As(err, &re) {
			t.Fatalf("expected *RenderError, got %T: %v", err, err)
		}
		if len(re.ComponentPath) != 3 {
			t.Fatalf("ComponentPath length = %d, want 3; path = %v", len(re.ComponentPath), re.ComponentPath)
		}
	})
}

// TestComponentPathErrorString verifies Error() output includes the " > "-joined path.
func TestComponentPathErrorString(t *testing.T) {
	fsys := fstest.MapFS{
		"Root.vue":  &fstest.MapFile{Data: []byte(`<template><div><Child /></div></template>`)},
		"Child.vue": &fstest.MapFile{Data: []byte(`<template><span>{{ missing.x }}</span></template>`)},
	}
	e := newMapFSEngine(t, fsys, Options{})
	err := e.RenderPage(context.Background(), io.Discard, "Root", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, " > ") {
		t.Errorf("error string %q should contain \" > \" path separator", msg)
	}
	if !strings.Contains(msg, "Root") {
		t.Errorf("error string %q should contain root component name", msg)
	}
}

// TestComponentErrorHandlerContinues verifies the page is written to w when
// the handler returns nil, and the placeholder appears in the output.
func TestComponentErrorHandlerContinues(t *testing.T) {
	fsys := fstest.MapFS{
		"Root.vue":  &fstest.MapFile{Data: []byte(`<template><div><Bad /><p>after</p></div></template>`)},
		"Bad.vue":   &fstest.MapFile{Data: []byte(`<template><span>{{ missing.x }}</span></template>`)},
	}

	var handlerCalled bool
	handler := func(w io.Writer, path []string, err error) error {
		handlerCalled = true
		fmt.Fprintf(w, `<div class="placeholder">error</div>`)
		return nil
	}

	e := newMapFSEngine(t, fsys, Options{ComponentErrorHandler: handler})
	var buf strings.Builder
	renderErr := e.RenderPage(context.Background(), &buf, "Root", nil)
	if renderErr != nil {
		t.Fatalf("RenderPage returned error: %v", renderErr)
	}
	if !handlerCalled {
		t.Error("handler was not called")
	}
	out := buf.String()
	if !strings.Contains(out, "placeholder") {
		t.Errorf("output %q: expected placeholder from handler", out)
	}
	if !strings.Contains(out, "after") {
		t.Errorf("output %q: expected sibling content after error placeholder", out)
	}
}

// TestComponentErrorHandlerAborts verifies the render aborts and w receives
// nothing when the handler returns a non-nil error.
func TestComponentErrorHandlerAborts(t *testing.T) {
	fsys := fstest.MapFS{
		"Root.vue":  &fstest.MapFile{Data: []byte(`<template><div><Bad /></div></template>`)},
		"Bad.vue":   &fstest.MapFile{Data: []byte(`<template><span>{{ missing.x }}</span></template>`)},
	}

	abortErr := errors.New("abort")
	handler := func(w io.Writer, path []string, err error) error {
		return abortErr
	}

	e := newMapFSEngine(t, fsys, Options{ComponentErrorHandler: handler})
	var buf strings.Builder
	renderErr := e.RenderPage(context.Background(), &buf, "Root", nil)
	if renderErr == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(renderErr, abortErr) {
		t.Errorf("expected abortErr in chain, got: %v", renderErr)
	}
	if buf.Len() != 0 {
		t.Errorf("w should receive nothing when handler aborts, got %q", buf.String())
	}
}

// TestComponentErrorHandlerNil verifies nil handler preserves the existing
// abort-on-error behaviour (no regression).
func TestComponentErrorHandlerNil(t *testing.T) {
	fsys := fstest.MapFS{
		"Root.vue":  &fstest.MapFile{Data: []byte(`<template><div><Bad /></div></template>`)},
		"Bad.vue":   &fstest.MapFile{Data: []byte(`<template><span>{{ missing.x }}</span></template>`)},
	}

	e := newMapFSEngine(t, fsys, Options{ComponentErrorHandler: nil})
	var buf strings.Builder
	err := e.RenderPage(context.Background(), &buf, "Root", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if buf.Len() != 0 {
		t.Errorf("w should receive nothing when no handler is set, got %q", buf.String())
	}
}

// TestHTMLErrorHandler verifies the built-in helper produces HTML-escaped
// output and continues rendering.
func TestHTMLErrorHandler(t *testing.T) {
	fsys := fstest.MapFS{
		"Root.vue":  &fstest.MapFile{Data: []byte(`<template><div><Bad /></div></template>`)},
		"Bad.vue":   &fstest.MapFile{Data: []byte(`<template><span>{{ missing.x }}</span></template>`)},
	}

	e := newMapFSEngine(t, fsys, Options{ComponentErrorHandler: HTMLErrorHandler()})
	var buf strings.Builder
	err := e.RenderPage(context.Background(), &buf, "Root", nil)
	if err != nil {
		t.Fatalf("RenderPage returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `class="htmlc-error"`) {
		t.Errorf("output %q: expected htmlc-error class", out)
	}
	if !strings.Contains(out, `data-path=`) {
		t.Errorf("output %q: expected data-path attribute", out)
	}
}

// TestComponentPathPropagationDeep verifies a three-level path ["A", "B", "C"]
// when C fails inside B inside A.
func TestComponentPathPropagationDeep(t *testing.T) {
	fsys := fstest.MapFS{
		"A.vue": &fstest.MapFile{Data: []byte(`<template><div><B /></div></template>`)},
		"B.vue": &fstest.MapFile{Data: []byte(`<template><section><C /></section></template>`)},
		"C.vue": &fstest.MapFile{Data: []byte(`<template><p>{{ missing.val }}</p></template>`)},
	}

	e := newMapFSEngine(t, fsys, Options{})
	err := e.RenderPage(context.Background(), io.Discard, "A", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var re *RenderError
	if !errors.As(err, &re) {
		t.Fatalf("expected *RenderError, got %T: %v", err, err)
	}
	if len(re.ComponentPath) != 3 {
		t.Fatalf("ComponentPath = %v, want 3 elements", re.ComponentPath)
	}
	if re.ComponentPath[0] != "A" {
		t.Errorf("ComponentPath[0] = %q, want %q", re.ComponentPath[0], "A")
	}
	if re.ComponentPath[1] != "b" {
		t.Errorf("ComponentPath[1] = %q, want %q", re.ComponentPath[1], "b")
	}
	if re.ComponentPath[2] != "c" {
		t.Errorf("ComponentPath[2] = %q, want %q", re.ComponentPath[2], "c")
	}
}
