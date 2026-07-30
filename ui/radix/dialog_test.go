package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// This module has no dependency on the root htmlc package, so a full
// render-based test (mounting Dialog.vue into a real htmlc.Engine and
// checking the rendered HTML) is out of scope here — that proof is
// deliberately deferred to the examples/radix-demo commit, which does
// depend on root htmlc. These are content-sanity checks: they confirm the
// component's source file contains the markers this commit's design
// depends on.
func TestDialog_ContainsZeroJSBaseline(t *testing.T) {
	src := readDialog(t)

	for _, marker := range []string{
		"<template>",
		"<dialog",
		":open=\"open\"",
		"<slot",
		`method="dialog"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Dialog.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestDialog_ContainsCustomElementEnhancement(t *testing.T) {
	src := readDialog(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-dialog'",
		"showModal()",
		"removeAttribute('open')",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Dialog.vue missing expected custom-element marker %q", marker)
		}
	}
}

func TestDialog_ContainsScopedStyle(t *testing.T) {
	src := readDialog(t)

	for _, marker := range []string{
		"<style scoped>",
		"::backdrop",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Dialog.vue missing expected style marker %q", marker)
		}
	}
}

func readDialog(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Dialog.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Dialog.vue) failed: %v", err)
	}
	return string(data)
}
