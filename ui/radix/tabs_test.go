package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// This module has no dependency on the root htmlc package, so a full
// render-based test (mounting Tabs.vue into a real htmlc.Engine and
// checking the rendered HTML) is out of scope here — that proof is
// deliberately deferred to the examples/radix-demo commit, which does
// depend on root htmlc. These are content-sanity checks: they confirm the
// component's source file contains the markers this commit's design
// depends on.
func TestTabs_ContainsZeroJSBaseline(t *testing.T) {
	src := readTabs(t)

	for _, marker := range []string{
		"<template>",
		`type="radio"`,
		`name="radix-tabs"`,
		"<label",
		"v-for=\"(item, index) in items\"",
		":checked=\"index === 0\"",
		"radix-tabs-panel",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Tabs.vue missing expected baseline marker %q", marker)
		}
	}
}

func TestTabs_ContainsCustomElementEnhancement(t *testing.T) {
	src := readTabs(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-tabs'",
		`role="tablist"`,
		`'role', 'tab'`,
		`'role', 'tabpanel'`,
		"aria-selected",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Tabs.vue missing expected custom-element marker %q", marker)
		}
	}
}

func TestTabs_ContainsScopedStyle(t *testing.T) {
	src := readTabs(t)

	if !strings.Contains(src, "<style>") {
		t.Error("Tabs.vue missing expected <style scoped> block")
	}
}

// TestTabs_LabelTagIsMarkedNative guards against a real regression: once
// ui/radix also ships a Label.vue component (it does), that component's
// auto-registered lowercase "label" alias would otherwise shadow every
// literal <label> tag in the package when mounted as a whole — including
// this file's own tab-button <label>, silently dropping its class/id/
// aria-controls attributes and rendering Label.vue's markup instead. The
// literal <label> here must carry v-native so it always resolves as a
// plain native element regardless of what else is mounted alongside it.
func TestTabs_LabelTagIsMarkedNative(t *testing.T) {
	src := readTabs(t)

	if !strings.Contains(src, "<label\n        v-native") && !strings.Contains(src, "<label v-native") {
		t.Error("Tabs.vue's <label> tag must carry v-native to avoid being shadowed by Label.vue's auto-registered lowercase alias")
	}
}

func readTabs(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Tabs.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Tabs.vue) failed: %v", err)
	}
	return string(data)
}
