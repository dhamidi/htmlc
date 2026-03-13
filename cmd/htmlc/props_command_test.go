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
		Component string                            `json:"component"`
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
