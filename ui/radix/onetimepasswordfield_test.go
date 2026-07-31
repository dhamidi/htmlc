package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like Slider.vue's/PasswordToggleField.vue's own tests, this module has no
// dependency on the root htmlc package, so a full render-based test
// (mounting OneTimePasswordField.vue into a real htmlc.Engine and checking
// rendered HTML) is out of scope here — that proof is deliberately
// deferred to the examples/radix-demo commit, which does depend on root
// htmlc. These are content-sanity checks: they confirm the component's
// source file contains the markers this commit's design depends on.
func TestOneTimePasswordField_ContainsZeroJSBaseline(t *testing.T) {
	src := readOneTimePasswordField(t)

	for _, marker := range []string{
		"<template>",
		"<input",
		`type="text"`,
		`inputmode="numeric"`,
		`pattern="[0-9]*"`,
		`autocomplete="one-time-code"`,
		`class="radix-otp-field-input"`,
		`:maxlength="length.length"`,
		`:name="name"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("OneTimePasswordField.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestOneTimePasswordField_LoopsOverLengthArray confirms one visual box is
// rendered per entry in the `length` prop, via v-for — the load-bearing
// structural decision documented in this file's header comment ("The
// `length`-as-array decision"): this template language has no
// range-generation construct (see expr/doc.go's own "Unsupported
// Constructs" list), so `length` is an array prop (e.g. Array(6).fill(null))
// rather than a bare number, the same workaround Slider.vue's own
// `values: number[]` prop already established for an analogous need.
func TestOneTimePasswordField_LoopsOverLengthArray(t *testing.T) {
	src := readOneTimePasswordField(t)
	tpl := otpTemplateBlock(t, src)

	if !strings.Contains(tpl, `v-for="(digit, index) in length"`) {
		t.Errorf(`OneTimePasswordField.vue <template> missing expected "v-for=\"(digit, index) in length\"" loop over the length prop; template was:\n%s`, tpl)
	}

	for _, marker := range []string{
		`class="radix-otp-field-box"`,
		`role="textbox"`,
		`:aria-label="'Character ' + (index + 1) + ' of ' + length.length"`,
		`tabindex="-1"`,
	} {
		if !strings.Contains(tpl, marker) {
			t.Errorf("OneTimePasswordField.vue <template> missing expected per-box marker %q; template was:\n%s", marker, tpl)
		}
	}
}

// TestOneTimePasswordField_VisibilitySwapTechnique confirms the
// data-state-driven visibility swap documented in this file's header
// comment: the box row starts hidden (data-state="hidden", so a no-JS user
// only ever sees/operates the real input) and the enhancement script flips
// both the boxes' and the input's own data-state once it runs — a
// data-state-attribute-driven CSS toggle, reusing Avatar.vue's own
// established data-state="hidden"/data-state="visible" pattern, rather than
// display:none/hidden set directly via inline style or the DOM `.hidden`
// property.
func TestOneTimePasswordField_VisibilitySwapTechnique(t *testing.T) {
	src := readOneTimePasswordField(t)

	for _, marker := range []string{
		`class="radix-otp-field-boxes" data-state="hidden"`,
		`.radix-otp-field-boxes[data-state='hidden']`,
		`.radix-otp-field-input[data-state='hidden']`,
		"display: none;",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("OneTimePasswordField.vue missing expected visibility-swap marker %q", marker)
		}
	}

	script := otpScriptBlock(t, src)
	for _, marker := range []string{
		`setAttribute('data-state', 'visible')`,
		`setAttribute('data-state', 'hidden')`,
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("OneTimePasswordField.vue <script customelement> missing expected data-state toggle %q; script was:\n%s", marker, script)
		}
	}
}

func TestOneTimePasswordField_ContainsCustomElementEnhancement(t *testing.T) {
	src := readOneTimePasswordField(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-one-time-password-field'",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("OneTimePasswordField.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestOneTimePasswordField_SyncsBoxesFromRealInput confirms the boxes'
// displayed characters are derived from the real input's own .value — the
// single source of truth — and never the reverse.
func TestOneTimePasswordField_SyncsBoxesFromRealInput(t *testing.T) {
	src := readOneTimePasswordField(t)
	script := otpScriptBlock(t, src)

	for _, marker := range []string{
		"#syncFromInput",
		"this.#input.value",
		"box.textContent = value[index]",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("OneTimePasswordField.vue <script customelement> missing expected sync marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestOneTimePasswordField_TypingAutoAdvancesAndDispatchesEvents confirms
// typing a digit updates the real input's value at the focused box's own
// position, dispatches real input/change events (so external listeners —
// form validation, a controlled-value caller — still fire correctly,
// matching Select.vue's own precedent), and auto-advances focus to the next
// box, blurring instead once the last box has just been filled.
func TestOneTimePasswordField_TypingAutoAdvancesAndDispatchesEvents(t *testing.T) {
	src := readOneTimePasswordField(t)
	script := otpScriptBlock(t, src)

	for _, marker := range []string{
		"/^[0-9]$/.test(event.key)",
		"new Event('input', { bubbles: true })",
		"new Event('change', { bubbles: true })",
		"this.#focusBox(index + 1)",
		"event.target.blur()",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("OneTimePasswordField.vue <script customelement> missing expected type/auto-advance marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestOneTimePasswordField_BackspaceMovesToPreviousWhenEmpty confirms the
// two-case Backspace behavior documented in this file's header comment
// (point 3): clearing the focused box's own character when it holds one,
// and — the case this commit's brief calls out by name — moving focus to
// the previous box and clearing *its* character when the focused box is
// already empty.
func TestOneTimePasswordField_BackspaceMovesToPreviousWhenEmpty(t *testing.T) {
	src := readOneTimePasswordField(t)
	script := otpScriptBlock(t, src)

	for _, marker := range []string{
		"event.key === 'Backspace'",
		"this.#focusBox(index - 1)",
		"value.slice(0, index - 1) + value.slice(index)",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("OneTimePasswordField.vue <script customelement> missing expected backspace-to-previous marker %q; script was:\n%s", marker, script)
		}
	}
}

// TestOneTimePasswordField_PasteStripsNonDigitsAndFocusesCorrectBox
// confirms the paste handler strips non-digit characters (a real, common
// OTP-paste robustness concern — a code copied with spaces or dashes),
// caps the result at the box count, writes it into the real input, and
// focuses the last filled box (full-length paste) or the first still-empty
// one (short paste) — this file's header comment, point 4.
func TestOneTimePasswordField_PasteStripsNonDigitsAndFocusesCorrectBox(t *testing.T) {
	src := readOneTimePasswordField(t)
	script := otpScriptBlock(t, src)

	for _, marker := range []string{
		"addEventListener('paste'",
		"event.preventDefault()",
		"replace(/[^0-9]/g, '')",
		"slice(0, this.#boxes.length)",
		"this.#setValue(digits)",
		"digits.length >= this.#boxes.length",
	} {
		if !strings.Contains(script, marker) {
			t.Errorf("OneTimePasswordField.vue <script customelement> missing expected paste-splitting marker %q; script was:\n%s", marker, script)
		}
	}
}

func TestOneTimePasswordField_ContainsScopedStyle(t *testing.T) {
	src := readOneTimePasswordField(t)

	if !strings.Contains(src, "<style scoped>") {
		t.Error("OneTimePasswordField.vue missing expected <style scoped> block")
	}
}

// otpTemplateBlock extracts the <template>...</template> block's source
// text, searching only after the header comment closes (the last "-->"
// before end of file) so it can't be fooled by this file's own header
// comment prose quoting template syntax as documentation.
func otpTemplateBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("OneTimePasswordField.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<template>")
	end := strings.Index(body, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("OneTimePasswordField.vue missing <template>...</template> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</template>")]
}

// otpScriptBlock extracts the <script customelement>...</script> block's
// source text, with the same after-header-comment search as
// otpTemplateBlock above.
func otpScriptBlock(t *testing.T, src string) string {
	t.Helper()
	afterComment := strings.LastIndex(src, "-->")
	if afterComment == -1 {
		t.Fatalf("OneTimePasswordField.vue missing header comment closer \"-->\"; source was:\n%s", src)
	}
	body := src[afterComment:]

	start := strings.Index(body, "<script customelement>")
	end := strings.Index(body, "</script>")
	if start == -1 || end == -1 {
		t.Fatalf("OneTimePasswordField.vue missing <script customelement>...</script> block after header comment; source was:\n%s", body)
	}
	return body[start : end+len("</script>")]
}

func readOneTimePasswordField(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/OneTimePasswordField.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/OneTimePasswordField.vue) failed: %v", err)
	}
	return string(data)
}
