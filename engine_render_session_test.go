package htmlc

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
)

// TestRenderFragmentSession_StyleDedup_ThreeTicks is the core motivating
// scenario from RFC 014 §1.2/§4.5/§6 Example 6: a component with
// <style scoped> rendered three times via RenderFragmentSession on the same
// *RenderSession (mirroring three Datastar SSE ticks) must emit its <style>
// block on the first call only; the second and third calls must render the
// component's HTML without repeating the block.
func TestRenderFragmentSession_StyleDedup_ThreeTicks(t *testing.T) {
	memFS := fstest.MapFS{
		"Counter.vue": &fstest.MapFile{Data: []byte(`<template><div class="counter">{{ count }}</div></template>
<style scoped>.counter { color: red; }</style>`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sess := e.NewRenderSession()

	var outs [3]string
	for i := 0; i < 3; i++ {
		var buf strings.Builder
		if err := e.RenderFragmentSession(context.Background(), &buf, "Counter", map[string]any{"count": i}, sess); err != nil {
			t.Fatalf("RenderFragmentSession call %d: %v", i, err)
		}
		outs[i] = buf.String()
	}

	if !strings.Contains(outs[0], "<style>") {
		t.Errorf("call 0: expected <style> block, got %q", outs[0])
	}
	if !strings.Contains(outs[0], "color: red") {
		t.Errorf("call 0: expected CSS body, got %q", outs[0])
	}
	for i := 1; i < 3; i++ {
		if strings.Contains(outs[i], "<style>") {
			t.Errorf("call %d: expected no <style> block (already emitted), got %q", i, outs[i])
		}
	}
	for i, out := range outs {
		if !strings.Contains(out, `class="counter"`) {
			t.Errorf("call %d: expected component HTML present, got %q", i, out)
		}
	}
}

// TestRenderFragmentSession_DifferentComponent_EmitsOwnStyle verifies that a
// second, different component rendered on the same session after the first
// still gets its own style block — only truly-already-seen contributions are
// suppressed, not everything after the first call.
func TestRenderFragmentSession_DifferentComponent_EmitsOwnStyle(t *testing.T) {
	memFS := fstest.MapFS{
		"Alpha.vue": &fstest.MapFile{Data: []byte(`<template><p class="alpha">alpha</p></template>
<style scoped>.alpha { color: blue; }</style>`)},
		"Beta.vue": &fstest.MapFile{Data: []byte(`<template><p class="beta">beta</p></template>
<style scoped>.beta { color: green; }</style>`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sess := e.NewRenderSession()

	var buf1 strings.Builder
	if err := e.RenderFragmentSession(context.Background(), &buf1, "Alpha", nil, sess); err != nil {
		t.Fatalf("render Alpha: %v", err)
	}
	if !strings.Contains(buf1.String(), "color: blue") {
		t.Errorf("Alpha call: expected alpha style, got %q", buf1.String())
	}

	var buf2 strings.Builder
	if err := e.RenderFragmentSession(context.Background(), &buf2, "Beta", nil, sess); err != nil {
		t.Fatalf("render Beta: %v", err)
	}
	if !strings.Contains(buf2.String(), "color: green") {
		t.Errorf("Beta call: expected beta style, got %q", buf2.String())
	}
	if strings.Contains(buf2.String(), ".alpha") {
		t.Errorf("Beta call: unexpectedly repeated alpha style, got %q", buf2.String())
	}

	// Re-render Alpha: its style must not repeat a third time.
	var buf3 strings.Builder
	if err := e.RenderFragmentSession(context.Background(), &buf3, "Alpha", nil, sess); err != nil {
		t.Fatalf("re-render Alpha: %v", err)
	}
	if strings.Contains(buf3.String(), "<style>") {
		t.Errorf("re-render Alpha: expected no style block, got %q", buf3.String())
	}
	if !strings.Contains(buf3.String(), `class="alpha"`) {
		t.Errorf("re-render Alpha: expected HTML present, got %q", buf3.String())
	}
}

// TestRenderFragmentSession_IndependentSessions verifies that two
// independent *RenderSessions don't leak state into each other: styles
// emitted on one session do not suppress the same component's styles on the
// other.
func TestRenderFragmentSession_IndependentSessions(t *testing.T) {
	memFS := fstest.MapFS{
		"Widget.vue": &fstest.MapFile{Data: []byte(`<template><span class="widget">w</span></template>
<style scoped>.widget { color: purple; }</style>`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sessA := e.NewRenderSession()
	sessB := e.NewRenderSession()

	var bufA strings.Builder
	if err := e.RenderFragmentSession(context.Background(), &bufA, "Widget", nil, sessA); err != nil {
		t.Fatalf("render on sessA: %v", err)
	}
	if !strings.Contains(bufA.String(), "<style>") {
		t.Errorf("sessA first call: expected style block, got %q", bufA.String())
	}

	var bufB strings.Builder
	if err := e.RenderFragmentSession(context.Background(), &bufB, "Widget", nil, sessB); err != nil {
		t.Fatalf("render on sessB: %v", err)
	}
	if !strings.Contains(bufB.String(), "<style>") {
		t.Errorf("sessB first call: expected its own style block (independent session), got %q", bufB.String())
	}
}

// TestRenderFragmentSession_NoStyle verifies a component with no
// <style scoped> at all renders correctly via RenderFragmentSession with no
// style block and no crash/empty-string failure.
func TestRenderFragmentSession_NoStyle(t *testing.T) {
	memFS := fstest.MapFS{
		"Plain.vue": &fstest.MapFile{Data: []byte(`<template><p>plain</p></template>`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sess := e.NewRenderSession()

	var buf strings.Builder
	if err := e.RenderFragmentSession(context.Background(), &buf, "Plain", nil, sess); err != nil {
		t.Fatalf("RenderFragmentSession: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "<style>") {
		t.Errorf("expected no style block, got %q", out)
	}
	if !strings.Contains(out, "<p>plain</p>") {
		t.Errorf("expected component HTML, got %q", out)
	}
}

// TestRenderFragmentSession_ScriptPassThrough proves the session's
// script-collector field is the same *CustomElementCollector plumbing
// RenderWithCollector already exposes: rendering the same custom-element
// component twice on one session reaches CustomElementCollector's existing
// content-hash dedup, so the script appears exactly once in
// sess.scripts.IndexJS()/ScriptsFS(). This commit does not add any new script
// dedup logic — it only threads the existing collector through the session.
func TestRenderFragmentSession_ScriptPassThrough(t *testing.T) {
	memFS := fstest.MapFS{
		"Widget.vue": &fstest.MapFile{Data: []byte(`<template><div>w</div></template>
<script customelement>
export default class Widget extends HTMLElement {}
</script>
`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sess := e.NewRenderSession()

	for i := 0; i < 2; i++ {
		var buf strings.Builder
		if err := e.RenderFragmentSession(context.Background(), &buf, "Widget", nil, sess); err != nil {
			t.Fatalf("RenderFragmentSession call %d: %v", i, err)
		}
		// No inline <script> markup is ever written to w for custom-element
		// scripts: script delivery stays out-of-band via the collector.
		if strings.Contains(buf.String(), "<script") {
			t.Errorf("call %d: unexpected inline <script> markup written to w: %q", i, buf.String())
		}
	}

	if sess.scripts.Len() != 1 {
		t.Errorf("sess.scripts.Len() = %d, want 1 (deduplicated across both calls)", sess.scripts.Len())
	}
	indexJS := sess.scripts.IndexJS()
	if !strings.Contains(indexJS, "widget") {
		t.Errorf("IndexJS() = %q, want it to reference the widget script", indexJS)
	}
	if strings.Count(indexJS, "import ") != 1 {
		t.Errorf("IndexJS() = %q, want exactly one import statement (script recorded once)", indexJS)
	}
}

// TestRenderFragmentSession_ErrorDoesNotAdvanceStyleEmitted verifies that a
// mid-render error on a RenderFragmentSession call (e.g. a reference to an
// unknown nested component) does not advance sess.styleEmitted, so a
// subsequent successful retry on the same session still emits the style that
// was never actually written to any real writer.
func TestRenderFragmentSession_ErrorDoesNotAdvanceStyleEmitted(t *testing.T) {
	memFS := fstest.MapFS{
		"Broken.vue": &fstest.MapFile{Data: []byte(`<template><div><missing-widget></missing-widget></div></template>
<style scoped>.broken { color: orange; }</style>`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	sess := e.NewRenderSession()

	var buf strings.Builder
	err = e.RenderFragmentSession(context.Background(), &buf, "Broken", nil, sess)
	if err == nil {
		t.Fatalf("expected error for unknown nested component, got nil (output: %q)", buf.String())
	}
	if buf.Len() != 0 {
		t.Errorf("expected nothing written to w on error, got %q", buf.String())
	}
	if sess.styleEmitted != 0 {
		t.Errorf("sess.styleEmitted = %d after failed render, want 0 (nothing was actually written)", sess.styleEmitted)
	}

	// Fix the component (register the missing tag as a native element) and
	// retry on the same session: the style must still be emitted, because it
	// was never actually written to a writer by the failed call.
	e2, err := New(Options{FS: memFS, ComponentDir: ".", NativeElements: []string{"missing-widget"}})
	if err != nil {
		t.Fatalf("New (with NativeElements): %v", err)
	}
	var buf2 strings.Builder
	if err := e2.RenderFragmentSession(context.Background(), &buf2, "Broken", nil, sess); err != nil {
		t.Fatalf("retry RenderFragmentSession: %v", err)
	}
	if !strings.Contains(buf2.String(), "color: orange") {
		t.Errorf("retry: expected style to be emitted (never actually written before), got %q", buf2.String())
	}
}
