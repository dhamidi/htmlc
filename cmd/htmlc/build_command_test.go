package main

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func writeJSON(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", path, err)
	}
}

func makeEntry(t *testing.T, pagesRoot, relVue string) pageEntry {
	t.Helper()
	absPath := filepath.Join(pagesRoot, relVue)
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(absPath, []byte(`<template><div></div></template>`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	dataPath := strings.TrimSuffix(absPath, ".vue") + ".json"
	if _, err := os.Stat(dataPath); err != nil {
		dataPath = ""
	}
	return pageEntry{
		relPath:  relVue,
		absPath:  absPath,
		dataPath: dataPath,
		outPath:  strings.TrimSuffix(relVue, ".vue") + ".html",
	}
}

func TestBuildHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-h"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "SYNOPSIS") {
		t.Errorf("stdout missing 'SYNOPSIS', got: %q", out)
	}
}

func TestBuildMissingPagesDir(t *testing.T) {
	dir := t.TempDir()
	missingPages := filepath.Join(dir, "nonexistent-pages")
	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-pages", missingPages}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "cannot find pages directory") {
		t.Errorf("stderr missing expected error, got: %q", errOut)
	}
}

func TestLoadPageData_NoDataFiles(t *testing.T) {
	pagesRoot := t.TempDir()
	entry := makeEntry(t, pagesRoot, "index.vue")

	got, err := loadPageData(entry, pagesRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestLoadPageData_PageLevelOnly(t *testing.T) {
	pagesRoot := t.TempDir()
	writeJSON(t, filepath.Join(pagesRoot, "index.json"), `{"title":"Hello","count":3}`)
	entry := makeEntry(t, pagesRoot, "index.vue")
	// Manually set dataPath since makeEntry uses stat after vue creation
	entry.dataPath = filepath.Join(pagesRoot, "index.json")

	got, err := loadPageData(entry, pagesRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["title"] != "Hello" {
		t.Errorf("expected title=Hello, got %v", got["title"])
	}
	if got["count"] != float64(3) {
		t.Errorf("expected count=3, got %v", got["count"])
	}
}

func TestLoadPageData_RootDataPlusPageLevel(t *testing.T) {
	pagesRoot := t.TempDir()
	writeJSON(t, filepath.Join(pagesRoot, "_data.json"), `{"site":"MySite","author":"root"}`)
	writeJSON(t, filepath.Join(pagesRoot, "index.json"), `{"author":"override","title":"Home"}`)
	entry := makeEntry(t, pagesRoot, "index.vue")
	entry.dataPath = filepath.Join(pagesRoot, "index.json")

	got, err := loadPageData(entry, pagesRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["site"] != "MySite" {
		t.Errorf("expected site=MySite, got %v", got["site"])
	}
	// page-level "author" overrides root default
	if got["author"] != "override" {
		t.Errorf("expected author=override, got %v", got["author"])
	}
	if got["title"] != "Home" {
		t.Errorf("expected title=Home, got %v", got["title"])
	}
}

func TestLoadPageData_TwoLevelChain(t *testing.T) {
	pagesRoot := t.TempDir()
	writeJSON(t, filepath.Join(pagesRoot, "_data.json"), `{"site":"Root","section":"root","page":"root"}`)
	writeJSON(t, filepath.Join(pagesRoot, "posts", "_data.json"), `{"section":"posts","page":"posts"}`)
	writeJSON(t, filepath.Join(pagesRoot, "posts", "hello.json"), `{"page":"hello"}`)
	entry := makeEntry(t, pagesRoot, filepath.Join("posts", "hello.vue"))
	entry.dataPath = filepath.Join(pagesRoot, "posts", "hello.json")

	got, err := loadPageData(entry, pagesRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["site"] != "Root" {
		t.Errorf("expected site=Root, got %v", got["site"])
	}
	if got["section"] != "posts" {
		t.Errorf("expected section=posts (subdir wins over root), got %v", got["section"])
	}
	if got["page"] != "hello" {
		t.Errorf("expected page=hello (page-level wins over all), got %v", got["page"])
	}
}

func TestLoadPageData_InvalidRootDataJSON(t *testing.T) {
	pagesRoot := t.TempDir()
	writeJSON(t, filepath.Join(pagesRoot, "_data.json"), `not-json`)
	entry := makeEntry(t, pagesRoot, "index.vue")

	_, err := loadPageData(entry, pagesRoot)
	if err == nil {
		t.Fatal("expected error for invalid JSON in _data.json, got nil")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in error, got: %v", err)
	}
}

func TestLoadPageData_InvalidPageJSON(t *testing.T) {
	pagesRoot := t.TempDir()
	writeJSON(t, filepath.Join(pagesRoot, "index.json"), `{bad json}`)
	entry := makeEntry(t, pagesRoot, "index.vue")
	entry.dataPath = filepath.Join(pagesRoot, "index.json")

	_, err := loadPageData(entry, pagesRoot)
	if err == nil {
		t.Fatal("expected error for invalid JSON in page file, got nil")
	}
	if !strings.Contains(err.Error(), "invalid JSON") {
		t.Errorf("expected 'invalid JSON' in error, got: %v", err)
	}
}

func TestBuildRendersPages(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// index page
	os.WriteFile(filepath.Join(pagesDir, "index.vue"),
		[]byte(`<template><html><body><h1>{{ title }}</h1></body></html></template>`), 0644)
	os.WriteFile(filepath.Join(pagesDir, "index.json"),
		[]byte(`{"title":"Home"}`), 0644)

	// posts/hello page
	if err := os.MkdirAll(filepath.Join(pagesDir, "posts"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(pagesDir, "posts", "hello.vue"),
		[]byte(`<template><html><body><p>{{ body }}</p></body></html></template>`), 0644)
	os.WriteFile(filepath.Join(pagesDir, "posts", "hello.json"),
		[]byte(`{"body":"Hello World"}`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	// Check index.html
	indexHTML := filepath.Join(outDir, "index.html")
	if _, err := os.Stat(indexHTML); err != nil {
		t.Errorf("expected %s to exist: %v", indexHTML, err)
	} else {
		content, _ := os.ReadFile(indexHTML)
		if !strings.Contains(string(content), "Home") {
			t.Errorf("index.html should contain 'Home', got: %s", content)
		}
	}

	// Check posts/hello.html
	helloHTML := filepath.Join(outDir, "posts", "hello.html")
	if _, err := os.Stat(helloHTML); err != nil {
		t.Errorf("expected %s to exist: %v", helloHTML, err)
	} else {
		content, _ := os.ReadFile(helloHTML)
		if !strings.Contains(string(content), "Hello World") {
			t.Errorf("posts/hello.html should contain 'Hello World', got: %s", content)
		}
	}

	// Summary line
	outStr := stdout.String()
	if !strings.Contains(outStr, "2 pages") {
		t.Errorf("summary should say '2 pages', got: %s", outStr)
	}
	if !strings.Contains(outStr, "0 errors") {
		t.Errorf("summary should say '0 errors', got: %s", outStr)
	}
}

func TestBuildCreatesOutDir(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := filepath.Join(t.TempDir(), "new-output-dir")

	os.WriteFile(filepath.Join(pagesDir, "index.vue"),
		[]byte(`<template><html><body>Hello</body></html></template>`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	if _, err := os.Stat(outDir); err != nil {
		t.Errorf("output directory should have been created: %v", err)
	}
	if _, err := os.Stat(filepath.Join(outDir, "index.html")); err != nil {
		t.Errorf("index.html should exist in created output dir: %v", err)
	}
}

func TestBuildReportsErrors(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// valid page
	os.WriteFile(filepath.Join(pagesDir, "good.vue"),
		[]byte(`<template><html><body>Good</body></html></template>`), 0644)

	// broken page: missing <template> section causes a parse error
	os.WriteFile(filepath.Join(pagesDir, "broken.vue"),
		[]byte(`<script>// no template section</script>`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}

	// good.html should still be rendered
	goodHTML := filepath.Join(outDir, "good.html")
	if _, err := os.Stat(goodHTML); err != nil {
		t.Errorf("good.html should exist despite other page failing: %v", err)
	}

	// Summary should reflect errors
	outStr := stdout.String()
	if !strings.Contains(outStr, "1 errors") {
		t.Errorf("summary should say '1 errors', got: %s", outStr)
	}
}

func TestBuildEmpty(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// No .vue files in pages dir

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, "0 pages") {
		t.Errorf("summary should say '0 pages', got: %s", outStr)
	}
	if !strings.Contains(outStr, "Build complete") {
		t.Errorf("summary should say 'Build complete', got: %s", outStr)
	}
}

func TestBuildWithLayout(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// Layout component with v-html="content"
	os.WriteFile(filepath.Join(componentsDir, "AppLayout.vue"),
		[]byte(`<template><html><body><div class="layout" v-html="content"></div></body></html></template>`), 0644)

	// Two page components
	os.WriteFile(filepath.Join(pagesDir, "index.vue"),
		[]byte(`<template><h1>{{ title }}</h1></template>`), 0644)
	os.WriteFile(filepath.Join(pagesDir, "index.json"),
		[]byte(`{"title":"Home"}`), 0644)
	os.WriteFile(filepath.Join(pagesDir, "about.vue"),
		[]byte(`<template><p>About</p></template>`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir, "-layout", "AppLayout"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	// Both output files should contain the layout wrapper
	for _, name := range []string{"index.html", "about.html"} {
		content, err := os.ReadFile(filepath.Join(outDir, name))
		if err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
			continue
		}
		if !strings.Contains(string(content), `class="layout"`) {
			t.Errorf("%s should contain layout wrapper, got: %s", name, content)
		}
	}

	// index.html should contain the page content injected via layout
	indexContent, _ := os.ReadFile(filepath.Join(outDir, "index.html"))
	if !strings.Contains(string(indexContent), "Home") {
		t.Errorf("index.html should contain page content 'Home', got: %s", indexContent)
	}
}

func TestBuildLayoutNotFound(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// A valid page so we can verify it is NOT rendered
	os.WriteFile(filepath.Join(pagesDir, "index.vue"),
		[]byte(`<template><html><body>Hello</body></html></template>`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir, "-layout", "Missing"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}

	// Error should mention the missing layout
	errOut := stderr.String()
	if !strings.Contains(errOut, "Missing") {
		t.Errorf("stderr should mention missing layout name, got: %s", errOut)
	}

	// No pages should have been rendered
	if _, err := os.Stat(filepath.Join(outDir, "index.html")); err == nil {
		t.Error("index.html should not exist when layout is missing")
	}
}

func TestRun_StrictFlag_ValidateAll_Build(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// Component that references a non-existent child component
	os.WriteFile(filepath.Join(componentsDir, "Wrapper.vue"),
		[]byte(`<template><div><NonExistentChild /></div></template>`), 0644)

	// A valid page
	os.WriteFile(filepath.Join(pagesDir, "index.vue"),
		[]byte(`<template><html><body>Hello</body></html></template>`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-strict", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d; stderr: %s", code, stderr.String())
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "validation error") {
		t.Errorf("expected 'validation error' in stderr, got: %s", errOut)
	}
}

func TestDirHash_StableWhenUnchanged(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.vue"), []byte(`<template><div></div></template>`), 0644)

	h1, err := dirHash(dir)
	if err != nil {
		t.Fatalf("dirHash error: %v", err)
	}
	h2, err := dirHash(dir)
	if err != nil {
		t.Fatalf("dirHash error: %v", err)
	}
	if h1 != h2 {
		t.Errorf("expected same hash on unchanged dir, got %q vs %q", h1, h2)
	}
}

func TestDirHash_ChangesOnModification(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.vue")
	os.WriteFile(path, []byte(`<template><div></div></template>`), 0644)

	h1, err := dirHash(dir)
	if err != nil {
		t.Fatalf("dirHash error: %v", err)
	}

	// Bump mtime by 1 second.
	future := time.Now().Add(time.Second)
	if err := os.Chtimes(path, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	h2, err := dirHash(dir)
	if err != nil {
		t.Fatalf("dirHash error: %v", err)
	}
	if h1 == h2 {
		t.Errorf("expected different hash after mtime change, but got same: %q", h1)
	}
}

func TestRunBuild_Dev_RebuildsOnChange(t *testing.T) {
	t.Parallel()
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// Write initial page.
	pagePath := filepath.Join(pagesDir, "index.vue")
	os.WriteFile(pagePath, []byte(`<template><html><body>v1</body></html></template>`), 0644)

	// Initial build.
	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("initial build failed: %s", stderr.String())
	}

	content, _ := os.ReadFile(filepath.Join(outDir, "index.html"))
	if !strings.Contains(string(content), "v1") {
		t.Fatalf("expected v1 in initial output, got: %s", content)
	}

	// Simulate a source change: update the page and bump mtime.
	os.WriteFile(pagePath, []byte(`<template><html><body>v2</body></html></template>`), 0644)
	future := time.Now().Add(time.Second)
	os.Chtimes(pagePath, future, future)

	// Compute initial hash so the rebuild function detects the change.
	initialHash, _ := dirHash(componentsDir, pagesDir)
	lastHash := initialHash

	// Change the file so hash differs.
	future2 := future.Add(time.Second)
	os.Chtimes(pagePath, future2, future2)

	// Build a handler that rebuilds, then check output.
	rebuildCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h, err := dirHash(componentsDir, pagesDir)
		if err == nil && h != lastHash {
			lastHash = h
			rebuildCalled = true
			run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, io.Discard, io.Discard)
		}
		http.FileServer(http.Dir(outDir)).ServeHTTP(w, r)
	})

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/index.html", nil)
	handler.ServeHTTP(rec, req)

	if !rebuildCalled {
		t.Error("expected rebuild to be triggered by file change")
	}

	content2, _ := os.ReadFile(filepath.Join(outDir, "index.html"))
	if !strings.Contains(string(content2), "v2") {
		t.Errorf("expected v2 in rebuilt output, got: %s", content2)
	}
}

func TestBuildDiscoversPages(t *testing.T) {
	pagesDir := t.TempDir()

	// Create a small page tree
	files := []string{
		"index.vue",
		"about.vue",
		"index.json",
		filepath.Join("posts", "hello.vue"),
		filepath.Join("posts", "hello.json"),
		filepath.Join("posts", "world.vue"),
		// Underscore-prefixed should be skipped
		"_partial.vue",
		filepath.Join("posts", "_shared.vue"),
		// Non-.vue files should be skipped
		"style.css",
	}
	for _, f := range files {
		full := filepath.Join(pagesDir, f)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(full, []byte(""), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", f, err)
		}
	}

	entries, err := discoverPages(pagesDir)
	if err != nil {
		t.Fatalf("discoverPages: %v", err)
	}

	// Expected pages (sorted by relPath)
	type expected struct {
		relPath string
		outPath string
		hasData bool
	}
	want := []expected{
		{relPath: "about.vue", outPath: "about.html", hasData: false},
		{relPath: "index.vue", outPath: "index.html", hasData: true},
		{relPath: filepath.Join("posts", "hello.vue"), outPath: filepath.Join("posts", "hello.html"), hasData: true},
		{relPath: filepath.Join("posts", "world.vue"), outPath: filepath.Join("posts", "world.html"), hasData: false},
	}

	if len(entries) != len(want) {
		relPaths := make([]string, len(entries))
		for i, e := range entries {
			relPaths[i] = e.relPath
		}
		t.Fatalf("got %d entries %v, want %d", len(entries), relPaths, len(want))
	}

	for i, w := range want {
		e := entries[i]
		if e.relPath != w.relPath {
			t.Errorf("entry[%d].relPath = %q, want %q", i, e.relPath, w.relPath)
		}
		if e.outPath != w.outPath {
			t.Errorf("entry[%d].outPath = %q, want %q", i, e.outPath, w.outPath)
		}
		if (e.dataPath != "") != w.hasData {
			t.Errorf("entry[%d].dataPath hasData = %v, want %v (dataPath=%q)", i, e.dataPath != "", w.hasData, e.dataPath)
		}
		if !slices.Contains([]string{e.absPath}, filepath.Join(pagesDir, w.relPath)) {
			t.Errorf("entry[%d].absPath = %q, want %q", i, e.absPath, filepath.Join(pagesDir, w.relPath))
		}
	}
}
