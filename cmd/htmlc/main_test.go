package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestRun_NoArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "htmlc") {
		t.Errorf("stdout missing help content, got: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("unexpected stderr: %q", stderr.String())
	}
}

func TestRun_HelpFlag_Long(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "SUBCOMMANDS") {
		t.Errorf("stdout missing SUBCOMMANDS section, got: %q", stdout.String())
	}
}

func TestRun_HelpFlag_Short(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-h"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "SUBCOMMANDS") {
		t.Errorf("stdout missing SUBCOMMANDS section, got: %q", stdout.String())
	}
}

func TestRun_HelpSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout.String(), "SUBCOMMANDS") {
		t.Errorf("stdout missing SUBCOMMANDS section, got: %q", stdout.String())
	}
}

func TestRun_HelpRender(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "render"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "render") {
		t.Errorf("stdout missing 'render', got: %q", out)
	}
	if !strings.Contains(out, "FLAGS") {
		t.Errorf("stdout missing FLAGS section, got: %q", out)
	}
	if !strings.Contains(out, "EXAMPLES") {
		t.Errorf("stdout missing EXAMPLES section, got: %q", out)
	}
}

func TestRun_HelpPage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "page"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "page") {
		t.Errorf("stdout missing 'page', got: %q", out)
	}
	if !strings.Contains(out, "FLAGS") {
		t.Errorf("stdout missing FLAGS section, got: %q", out)
	}
	if !strings.Contains(out, "EXAMPLES") {
		t.Errorf("stdout missing EXAMPLES section, got: %q", out)
	}
}

func TestRun_HelpProps(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "props"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "props") {
		t.Errorf("stdout missing 'props', got: %q", out)
	}
	if !strings.Contains(out, "FLAGS") {
		t.Errorf("stdout missing FLAGS section, got: %q", out)
	}
	if !strings.Contains(out, "EXAMPLES") {
		t.Errorf("stdout missing EXAMPLES section, got: %q", out)
	}
}

func TestRun_HelpUnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "unknowncmd"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "unknowncmd") {
		t.Errorf("stderr missing subcommand name, got: %q", stderr.String())
	}
}

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

func TestRun_UnknownSubcommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"unknowncmd"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "unknowncmd") {
		t.Errorf("stderr missing subcommand name, got: %q", errOut)
	}
	if !strings.Contains(errOut, "htmlc help") {
		t.Errorf("stderr missing hint to run 'htmlc help', got: %q", errOut)
	}
}

func TestRun_UnknownSubcommand_ExitMessage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"foo"}, &stdout, &stderr)
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), `"foo"`) {
		t.Errorf("stderr should quote the unknown subcommand, got: %q", stderr.String())
	}
}

// --- Error message tests ---

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

func TestPropsComponentNotFound(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	code := run([]string{"props", "-dir", dir, "Nonexistent"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, `"Nonexistent"`) {
		t.Errorf("expected component name in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Expected file:") {
		t.Errorf("expected file path hint in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, "Nonexistent.vue") {
		t.Errorf("expected .vue filename in stderr, got: %s", errOut)
	}
}

func TestPropsPathVsNameHint(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	code := run([]string{"props", "-dir", dir, "templates/Foo.vue"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "Tip:") {
		t.Errorf("expected Tip hint in stderr, got: %s", errOut)
	}
	if !strings.Contains(errOut, ".vue extension") {
		t.Errorf("expected .vue extension hint in stderr, got: %s", errOut)
	}
}

func TestPropsPathVsNameHint_DotVueOnly(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	// Name ending in .vue but no slash — still triggers tip
	code := run([]string{"props", "-dir", dir, "Foo.vue"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "Tip:") {
		t.Errorf("expected Tip hint in stderr, got: %s", errOut)
	}
}

func TestPropsSimpleNameNoTip(t *testing.T) {
	dir := t.TempDir()
	var stderr bytes.Buffer
	code := run([]string{"props", "-dir", dir, "Foo"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	// Simple name should NOT show the tip
	if strings.Contains(errOut, "Tip:") {
		t.Errorf("unexpected Tip hint for simple name, got: %s", errOut)
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

func TestPropsFormatJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Card.vue"), []byte(`
<template><div>{{ title }} {{ count }}</div></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"props", "-dir", dir, "-format", "json", "Card"}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("unexpected exit code %d", code)
	}
	var out struct {
		Component string                   `json:"component"`
		Props     []struct{ Name string `json:"name"` } `json:"props"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if out.Component != "Card" {
		t.Errorf("expected component=Card, got %q", out.Component)
	}
	names := make([]string, len(out.Props))
	for i, p := range out.Props {
		names[i] = p.Name
	}
	if !slices.Equal(names, []string{"count", "title"}) {
		t.Errorf("unexpected props: %v", names)
	}
}

func TestPropsFormatEnv(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Card.vue"), []byte(`
<template><div>{{ postTitle }}</div></template>`), 0644)

	var stdout bytes.Buffer
	run([]string{"props", "-dir", dir, "-format", "env", "Card"}, &stdout, io.Discard)
	if !strings.Contains(stdout.String(), "POST_TITLE=") {
		t.Errorf("expected POST_TITLE= in env output, got: %s", stdout.String())
	}
}

func TestPropsFormatInvalid(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Card.vue"), []byte(`<template><div></div></template>`), 0644)

	var stderr bytes.Buffer
	code := run([]string{"props", "-dir", dir, "-format", "yaml", "Card"}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), `"yaml"`) {
		t.Errorf("expected unknown format name in stderr, got: %s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Supported formats") {
		t.Errorf("expected supported formats hint in stderr, got: %s", stderr.String())
	}
}

func TestPropsPathStyle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PostCard.vue")
	os.WriteFile(path, []byte(`<template><div>{{ title }}</div></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"props", path}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "title") {
		t.Errorf("expected 'title' in output, got: %s", stdout.String())
	}
}

func TestPropsPathStyleJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PostCard.vue")
	os.WriteFile(path, []byte(`<template><div>{{ title }}</div></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"props", "-format", "json", path}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	var out struct {
		Component string `json:"component"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout.String())
	}
	if out.Component != "PostCard" {
		t.Errorf("expected component=PostCard, got %q", out.Component)
	}
}

func TestCamelToScreamingSnake(t *testing.T) {
	cases := []struct{ in, want string }{
		{"showDate", "SHOW_DATE"},
		{"postTitle", "POST_TITLE"},
		{"isActive", "IS_ACTIVE"},
		{"url", "URL"},
		{"htmlContent", "HTML_CONTENT"},
		{"title", "TITLE"},
	}
	for _, c := range cases {
		got := camelToScreamingSnake(c.in)
		if got != c.want {
			t.Errorf("camelToScreamingSnake(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
