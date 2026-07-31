package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like the other components in this package, this module has no dependency
// on the root htmlc package, so a full render-based test (mounting
// Avatar.vue into a real htmlc.Engine and checking rendered HTML) is out of
// scope here — that proof is deliberately deferred to the examples/
// radix-demo commit, which does depend on root htmlc. These are content-
// sanity checks: they confirm the component's source file contains the
// markers this commit's design depends on.
func TestAvatar_ContainsBaselineMarkers(t *testing.T) {
	src := readAvatar(t)

	for _, marker := range []string{
		"<template>",
		"<style scoped>",
		"<img",
		`:src="src"`,
		`:alt="alt"`,
		`<slot name="fallback">`,
		`data-state="hidden"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Avatar.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestAvatar_FallbackHiddenByDefault confirms the zero-JS baseline hides the
// fallback via CSS (display: none keyed off data-state="hidden"), matching
// this component's documented "show the <img> by default" reasoning.
func TestAvatar_FallbackHiddenByDefault(t *testing.T) {
	src := readAvatar(t)

	if !strings.Contains(src, `[data-state='hidden']`) {
		t.Errorf("Avatar.vue missing expected CSS selector for the hidden fallback state; source was:\n%s", src)
	}
	if !strings.Contains(src, "display: none") {
		t.Errorf("Avatar.vue missing expected \"display: none\" rule hiding the fallback by default; source was:\n%s", src)
	}
}

// TestAvatar_NoVNativeNeeded confirms Avatar.vue does not need the
// v-native escape hatch that Label.vue/Dialog.vue require: "avatar" is not
// a native HTML element, so this component's own auto-registered lowercase
// alias has no literal <avatar> tag anywhere in its own template to collide
// with (unlike Label's <label> or Dialog's <dialog>).
func TestAvatar_NoVNativeNeeded(t *testing.T) {
	src := readAvatar(t)

	templateStart := strings.Index(src, "<template>")
	templateEnd := strings.LastIndex(src, "</template>")
	if templateStart == -1 || templateEnd == -1 || templateEnd < templateStart {
		t.Fatalf("Avatar.vue missing expected <template>...</template> block; source was:\n%s", src)
	}
	// Scoped to the <template> block itself, not this file's header
	// comment, which legitimately discusses a hypothetical literal
	// <avatar> tag in prose while explaining why none exists.
	template := src[templateStart : templateEnd+len("</template>")]

	if strings.Contains(template, "<avatar") {
		t.Errorf("Avatar.vue's <template> unexpectedly contains a literal <avatar tag, which would need v-native to avoid a self-reference cycle; template was:\n%s", template)
	}
	if strings.Contains(template, "v-native") {
		t.Errorf("Avatar.vue's <template> unexpectedly uses v-native; per this component's design there is no native <avatar> tag to collide with, so no v-native escape hatch should be needed; template was:\n%s", template)
	}
}

// TestAvatar_ContainsCustomElementEnhancement confirms the <script
// customelement> block listens for the <img>'s native `error` event and
// registers the expected custom element tag name.
func TestAvatar_ContainsCustomElementEnhancement(t *testing.T) {
	src := readAvatar(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-avatar'",
		"addEventListener('error'",
		"addEventListener('load'",
		"setAttribute('data-state', 'visible')",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Avatar.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestAvatar_ErrorAndFallbackVisibilityNeverOverlapAccidentally confirms
// the error handler hides the <img> (style.display = 'none') before ever
// revealing the fallback, so the two are never shown at once as a result of
// the error path.
func TestAvatar_ErrorAndFallbackVisibilityNeverOverlapAccidentally(t *testing.T) {
	src := readAvatar(t)

	onErrorStart := strings.Index(src, "#onError = () => {")
	if onErrorStart == -1 {
		t.Fatalf("Avatar.vue missing expected #onError handler; source was:\n%s", src)
	}
	onErrorEnd := strings.Index(src[onErrorStart:], "\n  }")
	if onErrorEnd == -1 {
		t.Fatalf("Avatar.vue could not find end of #onError handler; source was:\n%s", src)
	}
	onError := src[onErrorStart : onErrorStart+onErrorEnd]

	hideIdx := strings.Index(onError, "this.#image.style.display = 'none'")
	showIdx := strings.Index(onError, "setAttribute('data-state', 'visible')")
	if hideIdx == -1 || showIdx == -1 || hideIdx > showIdx {
		t.Errorf("Avatar.vue #onError must hide the <img> before any code path can reveal the fallback; handler was:\n%s", onError)
	}
}

func readAvatar(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Avatar.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Avatar.vue) failed: %v", err)
	}
	return string(data)
}
