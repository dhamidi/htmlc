package datastar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	htmlc "github.com/dhamidi/htmlc"
	dsdatastar "github.com/starfederation/datastar-go/datastar"
)

// newTestEngine builds a minimal fstest.MapFS-backed *htmlc.Engine with two
// simple scoped-style components, mirroring the shape used by
// engine_render_session_test.go (TestRenderFragmentSession_StyleDedup_ThreeTicks)
// in the root module.
func newTestEngine(t *testing.T) *htmlc.Engine {
	t.Helper()
	memFS := fstest.MapFS{
		"Counter.vue": &fstest.MapFile{Data: []byte(`<template><div class="counter">{{ count }}</div></template>
<style scoped>.counter { color: red; }</style>`)},
		"Plain.vue": &fstest.MapFile{Data: []byte(`<template><p>plain</p></template>`)},
	}
	e, err := htmlc.New(htmlc.Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("htmlc.New: %v", err)
	}
	return e
}

// newSSE upgrades a fresh httptest.NewRecorder() to a
// *dsdatastar.ServerSentEventGenerator, exactly as a real Datastar handler
// would via dsdatastar.NewSSE(w, r).
func newSSE(t *testing.T) (*httptest.ResponseRecorder, *dsdatastar.ServerSentEventGenerator) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/sse", nil)
	rec := httptest.NewRecorder()
	return rec, dsdatastar.NewSSE(rec, req)
}

// splitEvents splits a recorded SSE body into its individual
// "datastar-patch-elements" events. Confirmed against datastar-go@v1.2.2's
// own sse.go (ServerSentEventGenerator.Send): every event is terminated by
// one trailing newline from its last "data: " line followed by the
// DoubleNewLine ("\n\n") blank-line event separator, i.e. three consecutive
// newlines ("\n\n\n") between the end of one event's data and the start of
// the next "event: " line (and after the last event). Splitting on "\n\n\n"
// and dropping the trailing empty tail therefore recovers exactly the
// sequence of raw events written to the wire.
func splitEvents(body string) []string {
	parts := strings.Split(body, "\n\n\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

// TestPatchElementsFragment_StyleDedup_ThreeCalls is the core motivating
// scenario this module exists for (RFC 014 §4.6/§6 Example 6): three
// PatchElementsFragment calls on the same *htmlc.RenderSession, mirroring
// three Datastar SSE ticks, must emit the component's <style scoped> block
// in the first event's "elements" payload only — the second and third
// events must carry bare HTML, proving RenderSession's dedup reaches through
// this wrapper.
func TestPatchElementsFragment_StyleDedup_ThreeCalls(t *testing.T) {
	e := newTestEngine(t)
	sess := e.NewRenderSession()
	rec, sse := newSSE(t)

	for i := 1; i <= 3; i++ {
		patch := Patch{Name: "Counter", Data: map[string]any{"count": i}, Selector: "#ds-counter"}
		if err := PatchElementsFragment(context.Background(), sse, e, sess, patch); err != nil {
			t.Fatalf("PatchElementsFragment call %d: %v", i, err)
		}
	}

	events := splitEvents(rec.Body.String())
	if len(events) != 3 {
		t.Fatalf("got %d SSE events, want 3; body=%q", len(events), rec.Body.String())
	}

	for i, ev := range events {
		if !strings.HasPrefix(ev, "event: datastar-patch-elements\n") {
			t.Errorf("event %d: unexpected framing, got %q", i, ev)
		}
		if !strings.Contains(ev, "data: selector #ds-counter\n") {
			t.Errorf("event %d: expected selector data line, got %q", i, ev)
		}
		if !strings.Contains(ev, `class="counter"`) {
			t.Errorf("event %d: expected component HTML present, got %q", i, ev)
		}
	}

	if !strings.Contains(events[0], "color: red") {
		t.Errorf("event 0: expected <style scoped> CSS body in first event, got %q", events[0])
	}
	for i := 1; i < 3; i++ {
		if strings.Contains(events[i], "color: red") {
			t.Errorf("event %d: expected no repeated style block, got %q", i, events[i])
		}
		if strings.Contains(events[i], "<style") {
			t.Errorf("event %d: expected no <style> tag at all, got %q", i, events[i])
		}
	}
}

// TestPatchElementsFragment_EmptySelector_OmitsSelectorOption verifies that
// an empty selector does not produce a "selector" data line at all — falling
// back to Datastar's own default id-based targeting — rather than sending an
// empty/meaningless selector option. Confirmed against datastar-go@v1.2.2's
// elements.go: PatchElements only appends a "selector " data row when its
// internal Selector option is non-empty, so this also documents that
// WithSelector("") would itself be a harmless no-op — PatchElementsFragment
// still avoids calling it at all when selector == "", so the "no selector
// means default targeting" intent is visible at this call site.
func TestPatchElementsFragment_EmptySelector_OmitsSelectorOption(t *testing.T) {
	e := newTestEngine(t)
	sess := e.NewRenderSession()
	rec, sse := newSSE(t)

	if err := PatchElementsFragment(context.Background(), sse, e, sess, Patch{Name: "Plain"}); err != nil {
		t.Fatalf("PatchElementsFragment: %v", err)
	}

	events := splitEvents(rec.Body.String())
	if len(events) != 1 {
		t.Fatalf("got %d SSE events, want 1; body=%q", len(events), rec.Body.String())
	}
	if strings.Contains(events[0], "data: selector") {
		t.Errorf("expected no selector data line for empty selector, got %q", events[0])
	}
	if !strings.Contains(events[0], "data: elements <p>plain</p>") {
		t.Errorf("expected component HTML in elements data line, got %q", events[0])
	}
}

// TestPatchElementsFragment_NonEmptySelector_ProducesSelectorLine verifies
// the positive case in isolation: a non-empty selector produces exactly one
// "data: selector <value>" line, using the real wire format confirmed above.
func TestPatchElementsFragment_NonEmptySelector_ProducesSelectorLine(t *testing.T) {
	e := newTestEngine(t)
	sess := e.NewRenderSession()
	rec, sse := newSSE(t)

	if err := PatchElementsFragment(context.Background(), sse, e, sess, Patch{Name: "Plain", Selector: "#target"}); err != nil {
		t.Fatalf("PatchElementsFragment: %v", err)
	}

	events := splitEvents(rec.Body.String())
	if len(events) != 1 {
		t.Fatalf("got %d SSE events, want 1; body=%q", len(events), rec.Body.String())
	}
	if !strings.Contains(events[0], "data: selector #target\n") {
		t.Errorf("expected selector data line, got %q", events[0])
	}
}

// TestPatchElementsFragment_RenderError_NoEventSent verifies that a render
// error (name referencing an unknown component) is returned to the caller
// and that PatchElementsFragment never calls sse.PatchElements in that case:
// no "datastar-patch-elements" event is written to the wire at all for the
// failed call. This is not merely documented but verified against the real
// behavior of Engine.RenderFragmentSession, which renders into an internal
// buffer first and only writes to its destination writer (here,
// PatchElementsFragment's own internal buf) once rendering succeeds in full —
// so a failed render can never leave a partial/corrupted "elements" payload
// behind for PatchElements to send.
func TestPatchElementsFragment_RenderError_NoEventSent(t *testing.T) {
	e := newTestEngine(t)
	sess := e.NewRenderSession()
	rec, sse := newSSE(t)

	err := PatchElementsFragment(context.Background(), sse, e, sess, Patch{Name: "DoesNotExist"})
	if err == nil {
		t.Fatal("expected an error for an unknown component, got nil")
	}

	if body := rec.Body.String(); body != "" {
		t.Errorf("expected no SSE event written for a failed render, got body=%q", body)
	}

	// A subsequent successful call on the same session must still work
	// normally — the failed call must not have left sess or sse in a bad
	// state.
	if err := PatchElementsFragment(context.Background(), sse, e, sess, Patch{Name: "Plain"}); err != nil {
		t.Fatalf("PatchElementsFragment after prior error: %v", err)
	}
	events := splitEvents(rec.Body.String())
	if len(events) != 1 {
		t.Fatalf("got %d SSE events after recovery, want 1; body=%q", len(events), rec.Body.String())
	}
}
