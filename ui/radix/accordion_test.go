package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// This module has no dependency on the root htmlc package, so a full
// render-based test (mounting Accordion.vue into a real htmlc.Engine and
// checking the rendered HTML) is out of scope here — that proof is
// deliberately deferred to the examples/radix-demo commit, which does
// depend on root htmlc. These are content-sanity checks: they confirm the
// component's source file contains the markers this commit's design
// depends on.
func TestAccordion_ContainsZeroJSBaseline(t *testing.T) {
	src := readAccordion(t)

	for _, marker := range []string{
		"<template>",
		"<details",
		"<summary",
		`name="radix-accordion"`,
		"v-for=\"item in items\"",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Accordion.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestAccordion_ContainsCustomElementEnhancement(t *testing.T) {
	src := readAccordion(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-accordion'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Accordion.vue missing expected custom-element marker %q", marker)
		}
	}
}

func TestAccordion_ContainsScopedStyle(t *testing.T) {
	src := readAccordion(t)

	if !strings.Contains(src, "<style>") {
		t.Error("Accordion.vue missing expected <style scoped> block")
	}
}

func readAccordion(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Accordion.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Accordion.vue) failed: %v", err)
	}
	return string(data)
}
