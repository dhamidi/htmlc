package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Toggle.vue's/Checkbox.vue's own tests, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting PasswordToggleField.vue into a real htmlc.Engine and checking
// rendered HTML) is out of scope here — that proof is deliberately
// deferred to the examples/radix-demo commit, which does depend on root
// htmlc. These are content-sanity checks: they confirm the component's
// source file contains the markers this commit's design depends on.
func TestPasswordToggleField_ContainsZeroJSBaseline(t *testing.T) {
	src := readPasswordToggleField(t)

	for _, marker := range []string{
		"<template>",
		"<input",
		`type="password"`,
		`:id="id"`,
		`:name="name"`,
		`:required="required"`,
		`:disabled="disabled"`,
		"<button",
		`type="button"`,
		`:aria-controls="id"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("PasswordToggleField.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestPasswordToggleField_ToggleButtonUsesStaticAriaLiterals confirms the
// toggle button's aria-pressed/aria-label are rendered as static literal
// text, not as a `:`-bound dynamic boolean expression. That is this
// component's own falsy-attribute-omission fix (see Toggle.vue's test
// suite/header comment for the general pitfall): there is no `pressed`-
// style boolean prop feeding this button at all (the zero-JS baseline has
// exactly one possible state), so a static literal can never be silently
// dropped by the rule that omits an attribute whenever its *bound*
// expression evaluates to Go's bare `false`. This test guards against a
// regression to a bare `:aria-pressed="someBoolean"` binding, which would
// reintroduce exactly the omission risk Toggle.vue's own header comment
// documents.
func TestPasswordToggleField_ToggleButtonUsesStaticAriaLiterals(t *testing.T) {
	src := readPasswordToggleField(t)

	for _, marker := range []string{
		`aria-pressed="false"`,
		`aria-label="Show password"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("PasswordToggleField.vue missing expected static ARIA literal %q", marker)
		}
	}

	// Search only the <template>...</script> body, after the header
	// comment closes, so this can't be fooled by the header comment's own
	// prose quoting Toggle.vue's `:aria-pressed="pressed ? ..."` pattern
	// as a documented contrast (see passwordToggleFieldScriptBlock's
	// identical rationale for the same "-->"-anchored search).
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("PasswordToggleField.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	for _, forbidden := range []string{
		`:aria-pressed="`,
		`:aria-label="`,
	} {
		if strings.Contains(body, forbidden) {
			t.Errorf("PasswordToggleField.vue toggle button should not use a dynamic %q binding in the baseline template — see header comment on why the static literal is the correct, omission-proof choice", forbidden)
		}
	}
}

// TestPasswordToggleField_ContainsCustomElementEnhancement confirms the
// load-bearing <script customelement> block exists and registers under
// the tag name the engine's path-derived kebab-case algorithm produces
// for this file (PasswordToggleField.vue -> "password-toggle-field",
// prefixed "radix-" per RFC 014's Mount{Prefix: "radix"} convention that
// every other component in this package already assumes).
func TestPasswordToggleField_ContainsCustomElementEnhancement(t *testing.T) {
	src := readPasswordToggleField(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-password-toggle-field'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("PasswordToggleField.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestPasswordToggleField_ClickHandlerFlipsInputType confirms the click
// handler actually mutates the real <input>'s `type` attribute between
// "password" and "text" — the one genuinely load-bearing piece of
// behavior this component needs script for (see file's header comment).
// A component that merely toggled a class or data-attribute without ever
// touching the real `type` attribute would not actually reveal anything;
// this test guards against that regression by requiring the literal
// setAttribute('type', ...) call sites for both values.
func TestPasswordToggleField_ClickHandlerFlipsInputType(t *testing.T) {
	src := readPasswordToggleField(t)
	script := passwordToggleFieldScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('click'",
		"getAttribute('type')",
		"setAttribute('type'",
		`'text'`,
		`'password'`,
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("PasswordToggleField.vue <script customelement> missing expected type-flipping marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestPasswordToggleField_ClickHandlerSyncsAriaAndLabel confirms the
// click handler keeps aria-pressed/aria-label/icon text in sync with the
// input's real type at runtime, using setAttribute (an explicit string,
// so no falsy-omission risk applies to these script-side writes either).
func TestPasswordToggleField_ClickHandlerSyncsAriaAndLabel(t *testing.T) {
	src := readPasswordToggleField(t)
	script := passwordToggleFieldScriptBlock(t, src)

	for _, marker := range []string{
		"setAttribute('aria-pressed'",
		"setAttribute('aria-label'",
		"Hide password",
		"Show password",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("PasswordToggleField.vue <script customelement> missing expected ARIA-sync marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestPasswordToggleField_ReturnsFocusToInput confirms the deliberate
// focus-management choice documented in the file's header comment: after
// toggling, focus moves back to the password input.
func TestPasswordToggleField_ReturnsFocusToInput(t *testing.T) {
	src := readPasswordToggleField(t)
	script := passwordToggleFieldScriptBlock(t, src)

	if !strings.Contains(script, "#input.focus()") {
		t.Errorf("PasswordToggleField.vue <script customelement> missing expected input.focus() call; script was:\n%s", script)
	}
}

func TestPasswordToggleField_ContainsScopedStyle(t *testing.T) {
	src := readPasswordToggleField(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("PasswordToggleField.vue missing expected <style scoped> block")
	}
}

// passwordToggleFieldScriptBlock extracts the
// <script customelement>...</script> block's source text. It searches only
// after the header comment closes (the last "-->" before end of file) so
// it can't be fooled by this file's own header comment prose mentioning
// "<script customelement>" literally while documenting it.
func passwordToggleFieldScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("PasswordToggleField.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("PasswordToggleField.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

func readPasswordToggleField(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/PasswordToggleField.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/PasswordToggleField.vue) failed: %v", err)
	}
	return string(data)
}
