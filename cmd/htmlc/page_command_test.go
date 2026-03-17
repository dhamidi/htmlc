package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPageDebugFlag(t *testing.T) {
	// TODO(RFC-011): re-enable debug comment assertion when attribute-based debug is implemented.
	// Debug mode is currently a no-op; verify the flag is accepted without error.
	dir := t.TempDir()
	path := filepath.Join(dir, "MyPage.vue")
	os.WriteFile(path, []byte(`<template><div>{{ title }}</div></template>`), 0644)

	var stdout bytes.Buffer
	code := run([]string{"page", "-debug", "-dir", dir, "-props", `{"title":"Hello"}`, "MyPage"}, &stdout, io.Discard)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	out := stdout.String()
	if strings.Contains(out, "[htmlc:debug]") {
		t.Errorf("debug is a no-op: unexpected debug comment in output:\n%s", out)
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

func TestRun_StrictFlag_MissingProp_Page(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "MyPage.vue"), []byte(`<template><html><body>{{ title }}</body></html></template>`), 0644)

	var stdout, stderr bytes.Buffer
	code := run([]string{"page", "-strict", "-dir", dir, "-props", "{}", "MyPage"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("expected exit 1, got %d; stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "missing prop") {
		t.Errorf("expected 'missing prop' in stderr, got: %s", stderr.String())
	}
}
