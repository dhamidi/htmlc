package htmlc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dhamidi/htmlc"
)

// newTestLogger returns a *slog.Logger that writes JSON records to buf.
func newTestLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
}

// writeVueFile is a helper to write a .vue file in dir.
func writeVueFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", name, err)
	}
}

// parseRecords decodes JSON log lines from buf into a slice of maps.
func parseRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var records []map[string]any
	dec := json.NewDecoder(buf)
	for dec.More() {
		var m map[string]any
		if err := dec.Decode(&m); err != nil {
			t.Fatalf("json decode: %v", err)
		}
		records = append(records, m)
	}
	return records
}

// setupTwoLevel creates a directory with Page.vue that includes Child.vue.
// Returns the engine and the temp dir.
func setupTwoLevel(t *testing.T, buf *bytes.Buffer) *htmlc.Engine {
	t.Helper()
	dir := t.TempDir()
	writeVueFile(t, dir, "Child.vue", `<template><span>child</span></template>`)
	writeVueFile(t, dir, "Page.vue", `<template><div><Child /></div></template>`)

	e, err := htmlc.New(htmlc.Options{
		ComponentDir: dir,
		Logger:       newTestLogger(buf),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

// TestLoggerEmitsRecordPerComponent checks that one debug record per component
// is emitted for a two-level tree (Page includes Child).
func TestLoggerEmitsRecordPerComponent(t *testing.T) {
	var buf bytes.Buffer
	e := setupTwoLevel(t, &buf)

	if _, err := e.RenderFragmentString("Page", nil); err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	records := parseRecords(t, &buf)
	// Expect records for Child and Page (two components).
	if len(records) != 2 {
		t.Errorf("want 2 log records, got %d: %v", len(records), records)
	}
	for _, r := range records {
		if r["msg"] != htmlc.MsgComponentRendered {
			t.Errorf("want msg %q, got %q", htmlc.MsgComponentRendered, r["msg"])
		}
	}
}

// TestLoggerRecordsBytes checks that the bytes attribute is > 0 and that
// the parent's bytes >= child's bytes.
func TestLoggerRecordsBytes(t *testing.T) {
	var buf bytes.Buffer
	e := setupTwoLevel(t, &buf)

	if _, err := e.RenderFragmentString("Page", nil); err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	records := parseRecords(t, &buf)
	if len(records) != 2 {
		t.Fatalf("want 2 records, got %d", len(records))
	}

	// Find child and parent records.
	var childBytes, pageBytes float64
	for _, r := range records {
		b, ok := r["bytes"].(float64)
		if !ok {
			t.Fatalf("bytes field missing or wrong type in %v", r)
		}
		if b <= 0 {
			t.Errorf("bytes should be > 0, got %v in record %v", b, r)
		}
		comp, _ := r["component"].(string)
		if comp == "Child" {
			childBytes = b
		} else if comp == "Page" {
			pageBytes = b
		}
	}
	if pageBytes < childBytes {
		t.Errorf("page bytes (%v) should be >= child bytes (%v)", pageBytes, childBytes)
	}
}

// TestLoggerRecordsDuration checks that the duration attribute is > 0.
func TestLoggerRecordsDuration(t *testing.T) {
	var buf bytes.Buffer
	e := setupTwoLevel(t, &buf)

	if _, err := e.RenderFragmentString("Page", nil); err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	records := parseRecords(t, &buf)
	for _, r := range records {
		// duration is emitted as a float (nanoseconds) by slog.Duration
		d, ok := r["duration"]
		if !ok {
			t.Errorf("duration field missing in record %v", r)
			continue
		}
		// slog JSON handler emits duration as a string like "1.234µs" or as integer nanoseconds
		switch v := d.(type) {
		case float64:
			if v <= 0 {
				t.Errorf("duration should be > 0, got %v", v)
			}
		case string:
			if v == "0s" || v == "" {
				t.Errorf("duration should be non-zero, got %q", v)
			}
		default:
			t.Errorf("unexpected duration type %T: %v", d, d)
		}
	}
}

// TestLoggerErrorRecord checks that a failed render emits an ERROR-level record
// with the error attribute set.
func TestLoggerErrorRecord(t *testing.T) {
	dir := t.TempDir()
	// BadChild uses an invalid v-for expression to trigger a render error.
	writeVueFile(t, dir, "BadChild.vue", `<template><div v-for="x in "></div></template>`)
	// Parent includes BadChild so the child render failure propagates.
	writeVueFile(t, dir, "ParentWithBad.vue", `<template><div><BadChild /></div></template>`)

	var buf bytes.Buffer
	e, err := htmlc.New(htmlc.Options{
		ComponentDir: dir,
		Logger:       newTestLogger(&buf),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, renderErr := e.RenderFragmentString("ParentWithBad", nil)
	if renderErr == nil {
		t.Fatal("expected render error, got nil")
	}

	records := parseRecords(t, &buf)
	if len(records) == 0 {
		t.Fatal("expected at least one log record on error, got none")
	}
	// Find the error-level record.
	var found bool
	for _, r := range records {
		if r["level"] == "ERROR" {
			found = true
			if r["msg"] != htmlc.MsgComponentFailed {
				t.Errorf("want msg %q, got %q", htmlc.MsgComponentFailed, r["msg"])
			}
			if r["error"] == nil {
				t.Errorf("error field should be set in ERROR record: %v", r)
			}
			break
		}
	}
	if !found {
		t.Errorf("no ERROR-level record found in: %v", records)
	}
}

// TestLoggerNil checks that a nil Logger causes no panic and output matches
// non-logger output.
func TestLoggerNil(t *testing.T) {
	dir := t.TempDir()
	writeVueFile(t, dir, "Card.vue", `<template><p>hello</p></template>`)

	eWithoutLogger, err := htmlc.New(htmlc.Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New (no logger): %v", err)
	}
	eWithNilLogger, err := htmlc.New(htmlc.Options{ComponentDir: dir, Logger: nil})
	if err != nil {
		t.Fatalf("New (nil logger): %v", err)
	}

	out1, err := eWithoutLogger.RenderFragmentString("Card", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (no logger): %v", err)
	}
	out2, err := eWithNilLogger.RenderFragmentString("Card", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (nil logger): %v", err)
	}
	if out1 != out2 {
		t.Errorf("output mismatch: %q vs %q", out1, out2)
	}
}

// ctxKeyType is an unexported type for context keys to avoid collisions.
type ctxKeyType struct{}

// contextCapturingHandler is a slog.Handler that records the context passed to Handle.
type contextCapturingHandler struct {
	inner    slog.Handler
	captured []context.Context
}

func (h *contextCapturingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *contextCapturingHandler) Handle(ctx context.Context, r slog.Record) error {
	h.captured = append(h.captured, ctx)
	return h.inner.Handle(ctx, r)
}

func (h *contextCapturingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &contextCapturingHandler{inner: h.inner.WithAttrs(attrs), captured: h.captured}
}

func (h *contextCapturingHandler) WithGroup(name string) slog.Handler {
	return &contextCapturingHandler{inner: h.inner.WithGroup(name), captured: h.captured}
}

// TestLoggerContextPropagation checks that the context passed to RenderPageContext
// is the same context received by the slog handler.
func TestLoggerContextPropagation(t *testing.T) {
	dir := t.TempDir()
	writeVueFile(t, dir, "Simple.vue", `<template><p>ok</p></template>`)

	var buf bytes.Buffer
	inner := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	handler := &contextCapturingHandler{inner: inner}
	logger := slog.New(handler)

	e, err := htmlc.New(htmlc.Options{
		ComponentDir: dir,
		Logger:       logger,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	type ctxKey struct{}
	ctx := context.WithValue(context.Background(), ctxKey{}, "sentinel")

	var out strings.Builder
	if err := e.RenderPageContext(ctx, &out, "Simple", nil); err != nil {
		t.Fatalf("RenderPageContext: %v", err)
	}

	if len(handler.captured) == 0 {
		t.Fatal("no context captured by handler")
	}
	for i, c := range handler.captured {
		if c.Value(ctxKey{}) != "sentinel" {
			t.Errorf("captured[%d] context does not carry expected value", i)
		}
	}
}

// TestLoggerPostOrder checks that leaf component records appear before parent
// component records (post-order traversal).
func TestLoggerPostOrder(t *testing.T) {
	var buf bytes.Buffer
	e := setupTwoLevel(t, &buf)

	if _, err := e.RenderFragmentString("Page", nil); err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	records := parseRecords(t, &buf)
	if len(records) < 2 {
		t.Fatalf("want at least 2 records, got %d", len(records))
	}

	// Child should appear before Page.
	var childIdx, pageIdx int = -1, -1
	for i, r := range records {
		comp, _ := r["component"].(string)
		if comp == "Child" {
			childIdx = i
		} else if comp == "Page" {
			pageIdx = i
		}
	}
	if childIdx == -1 || pageIdx == -1 {
		t.Fatalf("did not find both Child and Page records in %v", records)
	}
	if childIdx > pageIdx {
		t.Errorf("expected Child record before Page record, got child=%d page=%d", childIdx, pageIdx)
	}
}

// TestLoggerMessageConstants checks that emitted records use the exported
// message constants.
func TestLoggerMessageConstants(t *testing.T) {
	dir := t.TempDir()
	writeVueFile(t, dir, "Leaf.vue", `<template><em>leaf</em></template>`)
	writeVueFile(t, dir, "Root.vue", `<template><div><Leaf /></div></template>`)
	// BadLeaf uses an invalid expression to trigger a render error.
	writeVueFile(t, dir, "BadLeaf.vue", `<template><div v-if=""></div></template>`)
	writeVueFile(t, dir, "Failing.vue", `<template><div><BadLeaf /></div></template>`)

	var buf bytes.Buffer
	logger := newTestLogger(&buf)

	eOk, err := htmlc.New(htmlc.Options{ComponentDir: dir, Logger: logger})
	if err != nil {
		t.Fatalf("New (ok): %v", err)
	}

	if _, err := eOk.RenderFragmentString("Root", nil); err != nil {
		t.Fatalf("RenderFragmentString Root: %v", err)
	}

	var bufFail bytes.Buffer
	loggerFail := newTestLogger(&bufFail)
	eFail, err := htmlc.New(htmlc.Options{ComponentDir: dir, Logger: loggerFail})
	if err != nil {
		t.Fatalf("New (fail): %v", err)
	}
	_, _ = eFail.RenderFragmentString("Failing", nil)

	// Successful records.
	for _, r := range parseRecords(t, &buf) {
		if r["msg"] != htmlc.MsgComponentRendered {
			t.Errorf("success record msg: want %q, got %q", htmlc.MsgComponentRendered, r["msg"])
		}
	}

	// Error records.
	failRecords := parseRecords(t, &bufFail)
	var foundError bool
	for _, r := range failRecords {
		if r["level"] == "ERROR" {
			foundError = true
			if r["msg"] != htmlc.MsgComponentFailed {
				t.Errorf("error record msg: want %q, got %q", htmlc.MsgComponentFailed, r["msg"])
			}
		}
	}
	if !foundError {
		t.Errorf("no ERROR record found; records: %v", failRecords)
	}
}
