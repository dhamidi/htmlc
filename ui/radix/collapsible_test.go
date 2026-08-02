package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like accordion_test.go/switch_test.go, this module has no dependency on
// the root htmlc package, so a full render-based test (mounting
// Collapsible.vue into a real htmlc.Engine and checking rendered HTML) is
// out of scope here — that proof is deliberately deferred to the
// examples/radix-demo commit, which does depend on root htmlc. These are
// content-sanity checks: they confirm the component's source file
// contains the markers this commit's design depends on.
func TestCollapsible_ContainsZeroJSBaseline(t *testing.T) {
	src := readCollapsible(t)

	for _, marker := range []string{
		"<template>",
		"<details",
		`:open="open"`,
		"<summary",
		`<slot name="trigger">`,
		"<slot></slot>",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Collapsible.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestCollapsible_SummaryPrecedesContent confirms the <summary> (trigger
// slot) appears before the panel content div in document order, matching
// the native <details> requirement that <summary> be its first child.
func TestCollapsible_SummaryPrecedesContent(t *testing.T) {
	src := readCollapsible(t)
	tpl := collapsibleTemplateBlock(t, src)

	summaryIdx := strings.Index(tpl, "<summary")
	contentIdx := strings.Index(tpl, `class="radix-collapsible-content"`)
	if summaryIdx == -1 || contentIdx == -1 {
		t.Fatalf("Collapsible.vue <template> missing <summary> and/or content div; template was:\n%s", tpl)
	}
	if summaryIdx >= contentIdx {
		t.Fatalf("Collapsible.vue <template> must place <summary> before the content div; template was:\n%s", tpl)
	}
}

func TestCollapsible_ContainsScopedStyle(t *testing.T) {
	src := readCollapsible(t)

	if !strings.Contains(src, "<style>") {
		t.Error("Collapsible.vue missing expected <style scoped> block")
	}
}

// TestCollapsible_NoCustomElementScript confirms this component ships
// with no <script customelement> block, and documents why in the
// assertion itself: native <details>/<summary> already provides complete
// expand/collapse toggle behavior (click and native keyboard activation)
// and a real "toggle" DOM event a caller can listen to directly — with
// zero JavaScript. Unlike Accordion.vue (which adds a script solely for
// roving arrow-key navigation *between* multiple headers), a standalone
// Collapsible has exactly one focusable trigger, so there is nothing left
// for a script to add. See the file's header comment for the full
// reasoning, including why this does not just copy Switch.vue's
// conclusion without re-verifying it against MDN's <details>/<summary>
// accessibility documentation.
func TestCollapsible_NoCustomElementScript(t *testing.T) {
	src := readCollapsible(t)

	if strings.Contains(src, "<script customelement>") {
		t.Error("Collapsible.vue should not contain a <script customelement> block: native <details>/<summary> already provides full toggle behavior and a real native `toggle` event with zero JS — see the file's header comment")
	}
}

// collapsibleTemplateBlock extracts the <template>...</template> block's
// source text, excluding the header comment — same scoping technique
// switch_test.go's switchTemplateBlock uses.
func collapsibleTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<template>")
	if start == -1 {
		t.Fatalf("Collapsible.vue missing <template> opening tag; source was:\n%s", src)
	}
	relEnd := strings.Index(src[start:], "</template>")
	if relEnd == -1 {
		t.Fatalf("Collapsible.vue missing </template> closing tag after its opening tag; source was:\n%s", src)
	}
	end := start + relEnd
	return src[start : end+len("</template>")]
}

func readCollapsible(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Collapsible.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Collapsible.vue) failed: %v", err)
	}
	return string(data)
}
