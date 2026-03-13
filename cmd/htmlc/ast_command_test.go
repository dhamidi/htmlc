package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
