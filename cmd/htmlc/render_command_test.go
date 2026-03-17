package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_RenderHelpFlag_Long(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"render", "--help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "render") {
		t.Errorf("stdout missing 'render', got: %q", out)
	}
}

func TestRun_RenderHelpFlag_Short(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"render", "-h"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "render") {
		t.Errorf("stdout missing 'render', got: %q", out)
	}
}

func TestMissingComponentArg(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"render"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "missing component name") {
		t.Errorf("expected hint in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "htmlc render") {
		t.Errorf("expected usage hint in stderr, got: %s", errOut)
	}
}

func TestMissingComponentArg_UsageHint(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"render"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "USAGE") {
		t.Errorf("expected USAGE section in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "EXAMPLE") {
		t.Errorf("expected EXAMPLE section in stderr, got: %s", errOut)
	}
}

func TestRenderDirNotFound(t *testing.T) {
	var stderr bytes.Buffer
	code := run([]string{"render", "-dir", "/nonexistent-htmlc-test-dir", "Foo"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "cannot load components from") {
		t.Errorf("expected dir-not-found hint in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "/nonexistent-htmlc-test-dir") {
		t.Errorf("expected dir path in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "No such directory") {
		t.Errorf("expected 'No such directory' hint in stderr, got: %s", errOut)
	}
}

func TestRenderInvalidPropsFlag(t *testing.T) {
	var stderr bytes.Buffer
	// props parsing happens before dir check, so default dir "." is fine
	code := run([]string{"render", "Foo", "-props", "notjson"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "invalid JSON in -props flag") {
		t.Errorf("expected JSON hint in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Parse error:") {
		t.Errorf("expected Parse error detail in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "JSON object") {
		t.Errorf("expected JSON example hint in stderr, got: %s", errOut)
	}
}

func TestRenderInvalidPropsStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.WriteString("notjson"); err != nil {
		t.Fatal(err)
	}
	w.Close()
	old := os.Stdin
	os.Stdin = r
	defer func() { os.Stdin = old }()

	var stderr bytes.Buffer
	code := run([]string{"render", "Foo", "-props", "-"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "invalid JSON read from stdin") {
		t.Errorf("expected stdin JSON hint in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Parse error:") {
		t.Errorf("expected Parse error detail in stderr, got: %s", errOut)
	}
}

func TestRenderMissingComponent(t *testing.T) {
	dir := t.TempDir()
	// Create some .vue files so the engine loads and lists them
	for _, name := range []string{"Bar.vue", "Baz.vue", "Layout.vue"} {
		content := "<template><div></div></template>"
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var stderr bytes.Buffer
	code := run([]string{"render", "-dir", dir, "MissingComp"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, `"MissingComp"`) {
		t.Errorf("expected component name in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Available components:") {
		t.Errorf("expected available components list in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Bar") {
		t.Errorf("expected 'Bar' in available components, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Baz") {
		t.Errorf("expected 'Baz' in available components, got: %s", errOut)
	}
}

func TestRenderMissingComponentEmptyDir(t *testing.T) {
	dir := t.TempDir()
	// No .vue files — engine loads fine but no components available

	var stderr bytes.Buffer
	code := run([]string{"render", "-dir", dir, "Missing"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, `"Missing"`) {
		t.Errorf("expected component name in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Components are loaded from:") {
		t.Errorf("expected components-loaded-from hint in stderr, got: %s", errOut)
	}
}

func TestRenderMissingComponentTruncatesList(t *testing.T) {
	dir := t.TempDir()
	// Create more than 10 .vue files
	for i := 0; i < 12; i++ {
		name := filepath.Join(dir, strings.Repeat("A", i+1)+".vue")
		content := "<template><div></div></template>"
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	var stderr bytes.Buffer
	code := run([]string{"render", "-dir", dir, "Missing"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "and 2 more") {
		t.Errorf("expected truncation hint in stderr, got: %s", errOut)
	}
}

func TestErrorMessagesNoInternalPaths(t *testing.T) {
	// Error messages should not contain Go package paths or struct names
	var stderr bytes.Buffer
	run([]string{"render"}, io.Discard, &stderr)
	errOut := stderr.String()
	if strings.Contains(errOut, "github.com/") {
		t.Errorf("stderr contains Go package path, got: %s", errOut)
	}
}

func TestRenderDebugFlag(t *testing.T) {
	// Verify that -debug is accepted without error. A top-level component with no
	// child components produces no data-htmlc-* attributes (nothing to annotate).
	dir := t.TempDir()
	path := filepath.Join(dir, "Card.vue")
	os.WriteFile(path, []byte(`<template><div>{{ title }}</div></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"render", "-debug", "-dir", dir, "-props", `{"title":"Hello"}`, "Card"}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected rendered output to contain 'Hello', got:\n%s", out)
	}
}

func TestRun_StrictFlag_MissingProp_Render(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Card.vue"), []byte(`<template><div>{{ title }}</div></template>`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"render", "-strict", "-dir", dir, "-props", "{}", "Card"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "missing prop") {
		t.Errorf("expected 'missing prop' in stderr, got: %s", stderr.String())
	}
}

func TestRun_StrictFlag_NoError_WhenPropsProvided(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Card.vue"), []byte(`<template><div>{{ title }}</div></template>`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"render", "-strict", "-dir", dir, "-props", `{"title":"hi"}`, "Card"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "hi") {
		t.Errorf("expected 'hi' in output, got: %s", stdout.String())
	}
}

func TestRun_StrictFlag_PositionIndependent(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Card.vue"), []byte(`<template><div>{{ title }}</div></template>`), 0644)

	// -strict before subcommand
	var stdout1, stderr1 bytes.Buffer
	code1 := run([]string{"-strict", "render", "-dir", dir, "-props", `{"title":"hello"}`, "Card"}, &stdout1, &stderr1)
	if code1 != 0 {
		t.Errorf("-strict before subcommand: expected exit 0, got %d; stderr: %s", code1, stderr1.String())
	}

	// -strict after subcommand
	var stdout2, stderr2 bytes.Buffer
	code2 := run([]string{"render", "-strict", "-dir", dir, "-props", `{"title":"hello"}`, "Card"}, &stdout2, &stderr2)
	if code2 != 0 {
		t.Errorf("-strict after subcommand: expected exit 0, got %d; stderr: %s", code2, stderr2.String())
	}
}
