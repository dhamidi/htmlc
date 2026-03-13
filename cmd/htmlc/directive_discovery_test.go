package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverDirectives_Empty(t *testing.T) {
	dir := t.TempDir()
	got, err := discoverDirectives(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestDiscoverDirectives_FindsExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v-highlight")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := discoverDirectives(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["highlight"] != path {
		t.Errorf("expected highlight=%q, got %v", path, got)
	}
}

func TestDiscoverDirectives_IgnoresNonExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v-highlight")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := discoverDirectives(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["highlight"]; ok {
		t.Errorf("expected non-executable to be ignored, got %v", got)
	}
}

func TestDiscoverDirectives_StripsExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v-syntax-highlight.sh")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := discoverDirectives(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if abs, ok := got["syntax-highlight"]; !ok {
		t.Errorf("expected syntax-highlight in map, got %v", got)
	} else if abs != path {
		t.Errorf("expected path %q, got %q", path, abs)
	}
}

func TestDiscoverDirectives_SkipsHiddenDirs(t *testing.T) {
	dir := t.TempDir()
	hidden := filepath.Join(dir, ".git")
	if err := os.MkdirAll(hidden, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(hidden, "v-hidden")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := discoverDirectives(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := got["hidden"]; ok {
		t.Errorf("expected hidden dir to be skipped, got %v", got)
	}
}

func TestDiscoverDirectives_InvalidName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v-BadName")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := discoverDirectives(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected invalid name to be ignored, got %v", got)
	}
}

func TestDiscoverDirectives_SubdirDirectives(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "directives")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(subdir, "v-foo")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got, err := discoverDirectives(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["foo"] != path {
		t.Errorf("expected foo=%q in subdirectory, got %v", path, got)
	}
}
