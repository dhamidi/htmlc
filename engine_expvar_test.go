package htmlc_test

import (
	"encoding/json"
	"expvar"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dhamidi/htmlc"
	"github.com/dhamidi/htmlc/htmlctest"
	"github.com/dhamidi/htmlc/internal/testhelpers"
)

// parseExpvarMap unmarshals the String() output of an expvar.Var into a map.
func parseExpvarMap(t *testing.T, v expvar.Var) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(v.String()), &m); err != nil {
		t.Fatalf("unmarshal expvar JSON: %v\ngot: %s", err, v.String())
	}
	return m
}

// TestPublishExpvars_registers verifies that PublishExpvars adds a map entry
// to the global expvar registry and that its String() output is valid JSON.
func TestPublishExpvars_registers(t *testing.T) {
	e := htmlctest.NewHarness(t, map[string]string{
		"Hello.vue": `<template><p>hello</p></template>`,
	}).Engine()
	e.PublishExpvars("htmlc_test_a1")

	v := expvar.Get("htmlc_test_a1")
	if v == nil {
		t.Fatal("expvar.Get(\"htmlc_test_a1\") returned nil; PublishExpvars did not register")
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(v.String()), &m); err != nil {
		t.Fatalf("String() is not valid JSON: %v\ngot: %s", err, v.String())
	}
}

// TestPublishExpvars_counters verifies that the renders and renderErrors
// counters reflect the actual number of render calls.
func TestPublishExpvars_counters(t *testing.T) {
	e := htmlctest.NewHarness(t, map[string]string{
		"Counter.vue": `<template><span>{{ n }}</span></template>`,
	}).Engine()
	e.PublishExpvars("htmlc_test_a2")

	const N = 10
	for i := 0; i < N; i++ {
		if _, err := e.RenderFragmentString("Counter", map[string]any{"n": i}); err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
	}

	v := expvar.Get("htmlc_test_a2")
	m := parseExpvarMap(t, v)

	if renders, ok := m["renders"].(float64); !ok || renders != N {
		t.Errorf("renders = %v, want %d", m["renders"], N)
	}
	if errs, ok := m["renderErrors"].(float64); !ok || errs != 0 {
		t.Errorf("renderErrors = %v, want 0", m["renderErrors"])
	}
}

// TestSetDebug_toggles verifies that SetDebug correctly enables and disables
// attribute-based debug annotations and updates the expvar counter.
func TestSetDebug_toggles(t *testing.T) {
	e := htmlctest.NewHarness(t, map[string]string{
		"DbgChild.vue":  `<template><span>child</span></template>`,
		"DbgParent.vue": `<template><div><DbgChild /></div></template>`,
	}, htmlc.Options{Debug: false}).Engine()
	e.PublishExpvars("htmlc_test_a3")

	// Render with debug off — no data-htmlc-* attributes expected.
	out, err := e.RenderFragmentString("DbgParent", nil)
	if err != nil {
		t.Fatalf("render (debug off): %v", err)
	}
	if strings.Contains(out, "data-htmlc-") {
		t.Errorf("debug off: unexpected data-htmlc-* attribute in output: %s", out)
	}

	// Enable debug — data-htmlc-* attributes should appear on component root elements.
	e.SetDebug(true)
	out, err = e.RenderFragmentString("DbgParent", nil)
	if err != nil {
		t.Fatalf("render (debug on): %v", err)
	}
	if !strings.Contains(out, `data-htmlc-component="DbgChild"`) {
		t.Errorf("debug on: expected data-htmlc-component attribute in output: %s", out)
	}

	// Verify expvar reflects debug=1.
	v := expvar.Get("htmlc_test_a3")
	m := parseExpvarMap(t, v)
	if debug, ok := m["debug"].(float64); !ok || debug != 1 {
		t.Errorf("debug expvar = %v, want 1", m["debug"])
	}

	// Disable debug again — no data-htmlc-* attributes expected.
	e.SetDebug(false)
	out, err = e.RenderFragmentString("DbgParent", nil)
	if err != nil {
		t.Fatalf("render (debug off again): %v", err)
	}
	if strings.Contains(out, "data-htmlc-") {
		t.Errorf("debug off again: unexpected data-htmlc-* attribute in output: %s", out)
	}

	// Verify expvar reflects debug=0.
	m = parseExpvarMap(t, v)
	if debug, ok := m["debug"].(float64); !ok || debug != 0 {
		t.Errorf("debug expvar after SetDebug(false) = %v, want 0", m["debug"])
	}
}

// TestSetReload_reflected verifies that SetReload is mirrored in the expvar map.
func TestSetReload_reflected(t *testing.T) {
	e := htmlctest.NewHarness(t, map[string]string{
		"ReloadComp.vue": `<template><p>reload</p></template>`,
	}, htmlc.Options{Reload: false}).Engine()
	e.PublishExpvars("htmlc_test_a4")

	v := expvar.Get("htmlc_test_a4")
	m := parseExpvarMap(t, v)
	if reload, ok := m["reload"].(float64); !ok || reload != 0 {
		t.Errorf("initial reload = %v, want 0", m["reload"])
	}

	e.SetReload(true)
	m = parseExpvarMap(t, v)
	if reload, ok := m["reload"].(float64); !ok || reload != 1 {
		t.Errorf("after SetReload(true) reload = %v, want 1", m["reload"])
	}

	e.SetReload(false)
	m = parseExpvarMap(t, v)
	if reload, ok := m["reload"].(float64); !ok || reload != 0 {
		t.Errorf("after SetReload(false) reload = %v, want 0", m["reload"])
	}
}

// TestSetComponentDir_swaps verifies that SetComponentDir replaces the
// component registry and updates the componentDir expvar.
func TestSetComponentDir_swaps(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	testhelpers.WriteVue(t, dir1, "Alpha.vue", `<template><p>alpha</p></template>`)
	testhelpers.WriteVue(t, dir2, "Beta.vue", `<template><p>beta</p></template>`)

	e, err := htmlc.New(htmlc.Options{ComponentDir: dir1})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.PublishExpvars("htmlc_test_a5")

	if !e.Has("Alpha") {
		t.Error("Alpha should be registered from dir1")
	}
	if e.Has("Beta") {
		t.Error("Beta should not be registered before swap")
	}

	if err := e.SetComponentDir(dir2); err != nil {
		t.Fatalf("SetComponentDir: %v", err)
	}

	if e.Has("Alpha") {
		t.Error("Alpha should not be registered after swap to dir2")
	}
	if !e.Has("Beta") {
		t.Error("Beta should be registered after swap to dir2")
	}

	// Verify expvar reflects the new directory.
	v := expvar.Get("htmlc_test_a5")
	m := parseExpvarMap(t, v)
	if cd, ok := m["componentDir"].(string); !ok || cd != dir2 {
		t.Errorf("componentDir = %v, want %q", m["componentDir"], dir2)
	}
}

// TestSetFS_updatesVar verifies that SetFS is reflected in the "fs" expvar.
func TestSetFS_updatesVar(t *testing.T) {
	// Start with an OS-filesystem engine. ComponentDir "." is used so it is
	// compatible with both the OS walk (initial state) and the MapFS walk
	// (after SetFS).
	e, err := htmlc.New(htmlc.Options{ComponentDir: ".", FS: nil})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e.PublishExpvars("htmlc_test_a6")

	v := expvar.Get("htmlc_test_a6")
	m := parseExpvarMap(t, v)
	if fsVal, ok := m["fs"].(string); !ok || fsVal != "<nil>" {
		t.Errorf("initial fs = %v, want \"<nil>\"", m["fs"])
	}

	// Switch to an in-memory FS.
	mapFS := fstest.MapFS{
		"FsComp.vue": &fstest.MapFile{
			Data: []byte(`<template><p>mapfs</p></template>`),
		},
	}
	if err := e.SetFS(mapFS); err != nil {
		t.Fatalf("SetFS: %v", err)
	}

	m = parseExpvarMap(t, v)
	if fsVal, ok := m["fs"].(string); !ok || !strings.Contains(fsVal, "MapFS") {
		t.Errorf("after SetFS fs = %v, want string containing \"MapFS\"", m["fs"])
	}
}

// TestPublishExpvars_duplicatePanics verifies that calling PublishExpvars with
// the same prefix twice causes a panic.
func TestPublishExpvars_duplicatePanics(t *testing.T) {
	e := htmlctest.NewHarness(t, map[string]string{
		"Dup.vue": `<template><p>dup</p></template>`,
	}).Engine()
	e.PublishExpvars("htmlc_test_a7")

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate PublishExpvars call, got none")
		}
	}()
	e.PublishExpvars("htmlc_test_a7") // must panic
}

// TestNoPublishExpvars_noSideEffect verifies that an engine without
// PublishExpvars does not register anything in the global expvar registry.
func TestNoPublishExpvars_noSideEffect(t *testing.T) {
	e := htmlctest.NewHarness(t, map[string]string{
		"Nopub.vue": `<template><p>nopub</p></template>`,
	}).Engine()

	for i := 0; i < 5; i++ {
		if _, err := e.RenderFragmentString("Nopub", nil); err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
	}

	if v := expvar.Get("htmlc_test_a8"); v != nil {
		t.Errorf("expected no expvar registration, got: %v", v)
	}
}

// TestRenderErrors_counter verifies that rendering an unknown component
// increments the renderErrors counter.
func TestRenderErrors_counter(t *testing.T) {
	e := htmlctest.NewHarness(t, map[string]string{
		"Existing.vue": `<template><p>exists</p></template>`,
	}).Engine()
	e.PublishExpvars("htmlc_test_a9")

	_, renderErr := e.RenderPageString("NonExistentComponent", nil)
	if renderErr == nil {
		t.Fatal("expected error for unknown component, got nil")
	}

	v := expvar.Get("htmlc_test_a9")
	m := parseExpvarMap(t, v)
	if errs, ok := m["renderErrors"].(float64); !ok || errs != 1 {
		t.Errorf("renderErrors = %v, want 1", m["renderErrors"])
	}
}

// TestExpvarHTTPEndpoint starts a real in-process HTTP server via
// httptest.NewServer and verifies that /debug/vars exposes the engine metrics.
func TestExpvarHTTPEndpoint(t *testing.T) {
	e := htmlctest.NewHarness(t, map[string]string{
		"Page.vue": `<template><p>page</p></template>`,
	}).Engine()

	// The blank import of net/http/expvar registers /debug/vars on DefaultServeMux.
	srv := httptest.NewServer(http.DefaultServeMux)
	defer srv.Close()

	e.PublishExpvars("htmlc_test_a10")

	// Perform 5 renders.
	for i := 0; i < 5; i++ {
		if _, err := e.RenderFragmentString("Page", nil); err != nil {
			t.Fatalf("render %d: %v", i, err)
		}
	}

	// Fetch /debug/vars and locate our metrics.
	getVars := func() map[string]any {
		t.Helper()
		resp, err := http.Get(srv.URL + "/debug/vars")
		if err != nil {
			t.Fatalf("GET /debug/vars: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)

		var allVars map[string]json.RawMessage
		if err := json.Unmarshal(body, &allVars); err != nil {
			t.Fatalf("unmarshal /debug/vars: %v\nbody: %s", err, body)
		}
		raw, ok := allVars["htmlc_test_a10"]
		if !ok {
			t.Fatalf("key \"htmlc_test_a10\" not found in /debug/vars")
		}
		var m map[string]any
		if err := json.Unmarshal(raw, &m); err != nil {
			t.Fatalf("unmarshal htmlc_test_a10: %v", err)
		}
		return m
	}

	m := getVars()
	if renders, ok := m["renders"].(float64); !ok || renders != 5 {
		t.Errorf("renders = %v, want 5", m["renders"])
	}
	if errs, ok := m["renderErrors"].(float64); !ok || errs != 0 {
		t.Errorf("renderErrors = %v, want 0", m["renderErrors"])
	}
	if components, ok := m["components"].(float64); !ok || components < 1 {
		t.Errorf("components = %v, want >= 1", m["components"])
	}

	// Toggle reload and verify via a second GET.
	e.SetReload(true)
	m2 := getVars()
	if reload, ok := m2["reload"].(float64); !ok || reload != 1 {
		t.Errorf("reload after SetReload(true) = %v, want 1", m2["reload"])
	}
}
