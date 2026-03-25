package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// ceScript is a minimal custom element JS body used across CE build tests.
const ceScript = `customElements.define('ui-date-picker', class extends HTMLElement {
  connectedCallback() { this.textContent = 'date-picker'; }
});`

// writeCEComponent writes a .vue file with <script customelement> to dir/relPath.
func writeCEComponent(t *testing.T, dir, relPath, tag, script string) {
	t.Helper()
	full := filepath.Join(dir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	content := "<template><div></div></template>\n<script customelement>" + script + "</script>\n"
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile %s: %v", relPath, err)
	}
}

// TestBuildCEComponent_ScriptsDirCreated verifies that building a project with
// a CE component creates out/scripts/ and writes the correct .js file.
func TestBuildCEComponent_ScriptsDirCreated(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// CE component in components dir
	writeCEComponent(t, componentsDir, "ui/DatePicker.vue", "ui-date-picker", ceScript)

	// Page that uses the CE component
	pageContent := `<template><html><head></head><body><DatePicker /></body></html></template>`
	if err := os.WriteFile(filepath.Join(pagesDir, "index.vue"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("WriteFile index.vue: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	// scripts/ directory must exist
	scriptsDir := filepath.Join(outDir, "scripts")
	if _, err := os.Stat(scriptsDir); err != nil {
		t.Fatalf("expected scripts/ directory to exist: %v", err)
	}

	// At least one .js file must be present
	entries, err := os.ReadDir(scriptsDir)
	if err != nil {
		t.Fatalf("ReadDir scripts/: %v", err)
	}
	var jsFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".js") {
			jsFiles = append(jsFiles, e.Name())
		}
	}
	if len(jsFiles) == 0 {
		t.Fatalf("expected at least one .js file in scripts/, got none")
	}
}

// TestBuildNoCEComponent_NoscriptsDir verifies that building a project with no
// CE components does NOT create out/scripts/.
func TestBuildNoCEComponent_NoScriptsDir(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// Plain component — no <script customelement>
	if err := os.WriteFile(filepath.Join(componentsDir, "AppCard.vue"),
		[]byte(`<template><div class="card">Content</div></template>`), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Page that uses the plain component
	pageContent := `<template><html><head></head><body><AppCard /></body></html></template>`
	if err := os.WriteFile(filepath.Join(pagesDir, "index.vue"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("WriteFile index.vue: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	// scripts/ must NOT be created
	scriptsDir := filepath.Join(outDir, "scripts")
	if _, err := os.Stat(scriptsDir); err == nil {
		t.Errorf("scripts/ directory should not exist when no CE components are used")
	}
}

// TestBuildCEDeduplication verifies that two pages using the same CE component
// produce exactly one .js file in scripts/.
func TestBuildCEDeduplication(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	// Single CE component
	writeCEComponent(t, componentsDir, "ui/DatePicker.vue", "ui-date-picker", ceScript)

	// Two pages, both using the same CE component
	pageTemplate := `<template><html><head></head><body><DatePicker /></body></html></template>`
	if err := os.WriteFile(filepath.Join(pagesDir, "index.vue"), []byte(pageTemplate), 0644); err != nil {
		t.Fatalf("WriteFile index.vue: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pagesDir, "about.vue"), []byte(pageTemplate), 0644); err != nil {
		t.Fatalf("WriteFile about.vue: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	entries, err := os.ReadDir(filepath.Join(outDir, "scripts"))
	if err != nil {
		t.Fatalf("ReadDir scripts/: %v", err)
	}
	var jsFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".js") {
			jsFiles = append(jsFiles, e.Name())
		}
	}
	if len(jsFiles) != 1 {
		t.Errorf("expected exactly 1 .js file after deduplication, got %d: %v", len(jsFiles), jsFiles)
	}
}

// TestBuildCEScriptContent verifies that the written .js file content matches
// the <script customelement> source from the component.
func TestBuildCEScriptContent(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	writeCEComponent(t, componentsDir, "ui/DatePicker.vue", "ui-date-picker", ceScript)

	pageContent := `<template><html><head></head><body><DatePicker /></body></html></template>`
	if err := os.WriteFile(filepath.Join(pagesDir, "index.vue"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("WriteFile index.vue: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	entries, err := os.ReadDir(filepath.Join(outDir, "scripts"))
	if err != nil {
		t.Fatalf("ReadDir scripts/: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one file in scripts/")
	}

	jsPath := filepath.Join(outDir, "scripts", entries[0].Name())
	content, err := os.ReadFile(jsPath)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", jsPath, err)
	}
	if string(content) != ceScript {
		t.Errorf("script content mismatch\ngot:  %q\nwant: %q", string(content), ceScript)
	}
}

// TestBuildCE_MultipleDistinctScripts verifies that two CE components with
// different scripts each produce their own .js file.
func TestBuildCE_MultipleDistinctScripts(t *testing.T) {
	componentsDir := t.TempDir()
	pagesDir := t.TempDir()
	outDir := t.TempDir()

	script1 := `customElements.define('ui-date-picker', class extends HTMLElement {});`
	script2 := `customElements.define('widgets-shape-canvas', class extends HTMLElement {});`

	writeCEComponent(t, componentsDir, "ui/DatePicker.vue", "ui-date-picker", script1)
	writeCEComponent(t, componentsDir, "widgets/ShapeCanvas.vue", "widgets-shape-canvas", script2)

	pageContent := `<template><html><head></head><body><DatePicker /><ShapeCanvas /></body></html></template>`
	if err := os.WriteFile(filepath.Join(pagesDir, "index.vue"), []byte(pageContent), 0644); err != nil {
		t.Fatalf("WriteFile index.vue: %v", err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"build", "-dir", componentsDir, "-pages", pagesDir, "-out", outDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}

	entries, err := os.ReadDir(filepath.Join(outDir, "scripts"))
	if err != nil {
		t.Fatalf("ReadDir scripts/: %v", err)
	}
	var jsFiles []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".js") {
			jsFiles = append(jsFiles, e.Name())
		}
	}
	if len(jsFiles) != 2 {
		t.Errorf("expected 2 .js files for 2 distinct CE components, got %d: %v", len(jsFiles), jsFiles)
	}
}

// TestBuildCE_DirHashFS is a simple sanity check that dirHashFS works with an
// in-memory FS (already covered by existing tests, but kept here for
// completeness alongside the CE test file).
func TestBuildCE_DirHashFS_Sanity(t *testing.T) {
	fsys := fstest.MapFS{
		"components/DatePicker.vue": &fstest.MapFile{Data: []byte(`<template><div/></template>`)},
	}
	h, err := dirHashFS(fsys, "components")
	if err != nil {
		t.Fatalf("dirHashFS: %v", err)
	}
	if h == "" {
		t.Error("expected non-empty hash")
	}
}
