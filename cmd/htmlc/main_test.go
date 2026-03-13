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

func TestRenderDebugFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "Card.vue")
	os.WriteFile(path, []byte(`<template><div>{{ title }}</div></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"render", "-debug", "-dir", dir, "-props", `{"title":"Hello"}`, "Card"}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "[htmlc:debug]") {
		t.Errorf("--debug flag: output should contain debug comments, got:\n%s", out)
	}
}

func TestPageDebugFlag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "MyPage.vue")
	os.WriteFile(path, []byte(`<template><div>{{ title }}</div></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"page", "-debug", "-dir", dir, "-props", `{"title":"Hello"}`, "MyPage"}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "[htmlc:debug]") {
		t.Errorf("--debug flag: output should contain debug comments, got:\n%s", out)
	}
}

func TestAstSubcommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "PostPage.vue")
	os.WriteFile(path, []byte(`<template><article><h1>{{ title }}</h1></article></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"ast", "-dir", dir, "PostPage"}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "Element[article]") {
		t.Errorf("ast output should contain Element[article], got:\n%s", out)
	}
	if !strings.Contains(out, "Element[h1]") {
		t.Errorf("ast output should contain Element[h1], got:\n%s", out)
	}
}

func TestAstSubcommand_MissingComponent(t *testing.T) {
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	code := run([]string{"ast", "-dir", dir, "NonExistent"}, &stdout, &stderr)
	if code == 0 {
		t.Error("expected non-zero exit code for missing component")
	}
	if !strings.Contains(stderr.String(), "NonExistent") {
		t.Errorf("stderr should mention the missing component, got:\n%s", stderr.String())
	}
}

func TestPageLayout_LayoutNotFound(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "PostPage.vue"), []byte(`<template><article>{{ body }}</article></template>`), 0644)

	var stderr bytes.Buffer
	code := run([]string{"page", "-dir", dir, "-layout", "AppLayout", "PostPage", "-props", `{"body":"Hello"}`}, io.Discard, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	errOut := stderr.String()
	if !strings.Contains(errOut, "AppLayout") {
		t.Errorf("expected layout name in stderr, got: %s", errOut)
	}
}

func TestPageLayout_ContentPassedToLayout(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "PostPage.vue"), []byte(`<template><article>{{ body }}</article></template>`), 0644)
	os.WriteFile(filepath.Join(dir, "AppLayout.vue"), []byte(`<template><html><body v-html="content"></body></html></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"page", "-dir", dir, "-layout", "AppLayout", "PostPage", "-props", `{"body":"Hello"}`}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "<article>") {
		t.Errorf("expected <article> in layout output, got: %s", out)
	}
	if !strings.Contains(out, "Hello") {
		t.Errorf("expected body content in layout output, got: %s", out)
	}
}

func TestPageLayout_PropsPassedToLayout(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "PostPage.vue"), []byte(`<template><article>{{ body }}</article></template>`), 0644)
	os.WriteFile(filepath.Join(dir, "AppLayout.vue"), []byte(`<template><html><head><title>{{ title }}</title></head><body v-html="content"></body></html></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"page", "-dir", dir, "-layout", "AppLayout", "PostPage", "-props", `{"title":"My Title","body":"Hello"}`}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "My Title") {
		t.Errorf("expected title prop in layout output, got: %s", out)
	}
}

func TestHelpAst(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"help", "ast"}, &stdout, &stderr)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	out := stdout.String()
	if !strings.Contains(out, "ast") {
		t.Errorf("stdout missing 'ast', got: %q", out)
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

// --- loadPageData tests ---

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

// --- Build integration tests ---

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
		relPath  string
		outPath  string
		hasData  bool
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
