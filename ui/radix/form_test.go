package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// Like progress_test.go/label_test.go, this module has no dependency on the
// root htmlc package, so a full render-based test (mounting Form.vue into a
// real htmlc.Engine and checking rendered HTML for both the with-error and
// without-error states) is out of scope here. These are content-sanity
// checks: they confirm the component's source file contains the markers
// this commit's design depends on, including the documented
// consumer-side contract this component cannot enforce itself.
func TestForm_ContainsBaselineMarkers(t *testing.T) {
	src := readForm(t)

	for _, marker := range []string{
		"<template>",
		"<style scoped>",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("Form.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestForm_HasNoCustomElementScript confirms this component ships no
// <script customelement> block: the error message is pure server-computed
// conditional markup (an `error` string and a `v-if` check, both fully
// known at render time), matching Progress.vue's/Label.vue's own "nothing
// left for a script to enhance" conclusion — see this file's header
// comment for the fuller reasoning, including why a live-clearing
// enhancement was considered and rejected.
func TestForm_HasNoCustomElementScript(t *testing.T) {
	src := readForm(t)

	if strings.Contains(src, "<script customelement>") {
		t.Error("Form.vue should have no <script customelement> block; it needs no JS enhancement")
	}
}

// TestForm_LabelForIdAssociation confirms the wrapper <label> is bound to
// the "id" prop via a native `for` attribute, matching Checkbox.vue's/
// Label.vue's own `for`/`id` association convention, and that it is marked
// v-native so it does not resolve back to Label.vue's own component
// (matching Checkbox.vue's identical cross-component collision fix).
func TestForm_LabelForIdAssociation(t *testing.T) {
	src := readForm(t)
	label := labelElementForm(t, src)

	if !strings.Contains(label, `:for="id"`) {
		t.Errorf(`Form.vue <label> missing expected marker :for="id"; label was:\n%s`, label)
	}
	if !strings.Contains(label, "v-native") {
		t.Errorf("Form.vue <label> missing expected v-native marker (needed to avoid resolving to Label.vue's own component); label was:\n%s", label)
	}
	if !strings.Contains(label, "{{ label }}") {
		t.Errorf("Form.vue <label> missing expected {{ label }} text interpolation; label was:\n%s", label)
	}
}

// TestForm_DefaultSlotForControl confirms the template renders a default
// <slot> between the label and the message region, for the consumer's own
// control (<input>, <select>, etc.) to be inserted into.
func TestForm_DefaultSlotForControl(t *testing.T) {
	src := readForm(t)
	template := templateBlockForm(t, src)

	if !strings.Contains(template, "<slot></slot>") {
		t.Errorf("Form.vue <template> missing expected default <slot></slot> for the consumer's control; template was:\n%s", template)
	}
}

// TestForm_MessageConditionallyRendered confirms the error message <p> is
// gated by a real v-if on the "error" prop (not a CSS-hide trick — an empty
// error genuinely should not exist in the DOM), and that its id follows
// the documented `id + '-message'` convention this file's own header
// comment promises consumers can rely on.
func TestForm_MessageConditionallyRendered(t *testing.T) {
	src := readForm(t)
	template := templateBlockForm(t, src)

	for _, marker := range []string{
		`v-if="error"`,
		`:id="id + '-message'"`,
		`role="alert"`,
		"{{ error }}",
	} {
		if !strings.Contains(template, marker) {
			t.Errorf("Form.vue <template> missing expected marker %q; template was:\n%s", marker, template)
		}
	}
}

// TestForm_MessageIsAP confirms the message element is a <p>, matching this
// commit's design brief exactly (`<p v-if="error" ...>`), not some other
// element that happens to carry the same v-if/id/role markers.
func TestForm_MessageIsAP(t *testing.T) {
	src := readForm(t)
	template := templateBlockForm(t, src)

	if !strings.Contains(template, `<p v-if="error"`) {
		t.Errorf(`Form.vue <template> missing expected <p v-if="error" ...> message element; template was:\n%s`, template)
	}
}

// TestForm_DocumentsConsumerSideAriaContract confirms the header comment
// explicitly documents that the consumer's own slotted control is
// responsible for setting aria-describedby/aria-invalid on itself
// (a real, honest limitation this component cannot work around, since it
// does not render the control it does not own) — and that the documented
// contract references this file's own `id + '-message'` convention, not
// some other id scheme a consumer would have to guess.
func TestForm_DocumentsConsumerSideAriaContract(t *testing.T) {
	src := readForm(t)
	header := headerCommentForm(t, src)

	for _, marker := range []string{
		"aria-describedby",
		"aria-invalid",
		"id + '-message'",
	} {
		if !strings.Contains(header, marker) {
			t.Errorf("Form.vue header comment missing expected marker %q documenting the consumer-side ARIA contract", marker)
		}
	}
}

// TestForm_UsageExampleShowsConsumerSideControl confirms the header
// comment's usage example actually shows a slotted <input> wired with both
// aria-describedby and aria-invalid, matching the process brief's
// requirement for "a slotted <input> example showing the full intended
// consumer-side usage" — not just prose describing the contract in the
// abstract.
func TestForm_UsageExampleShowsConsumerSideControl(t *testing.T) {
	src := readForm(t)
	header := headerCommentForm(t, src)

	for _, marker := range []string{
		"<input",
		"aria-describedby",
		"aria-invalid",
		"email-message",
	} {
		if !strings.Contains(header, marker) {
			t.Errorf("Form.vue header comment's usage example missing expected marker %q", marker)
		}
	}
}

// TestForm_DocumentsConsumerFormNeedsVNative confirms the header comment
// loudly documents the critical, empirically-confirmed finding: any
// literal <form> tag anywhere in an engine that mounts this package
// (including on pages that never use <Form> at all) resolves to this
// component's own lowercase "form" alias unless marked v-native, because
// this component follows the same entries[lower] = entry auto-alias
// mechanism as every other named component in this package. Also confirms
// the header comment's own usage examples show the fix in context
// (`<form v-native`), not just describe it in prose.
func TestForm_DocumentsConsumerFormNeedsVNative(t *testing.T) {
	src := readForm(t)
	header := headerCommentForm(t, src)

	for _, marker := range []string{
		"v-native",
		"NativeElements",
		"<form v-native",
	} {
		if !strings.Contains(header, marker) {
			t.Errorf("Form.vue header comment missing expected marker %q documenting the consumer <form> v-native requirement", marker)
		}
	}
}

// TestForm_DocumentsNestedFormAliasRequirement confirms the header comment
// documents the second, distinct trap this file's own name creates: even
// with the consumer's wrapping <form> correctly marked v-native (per
// "Critical" above), a literal <Form> reference nested inside it still
// lowercases to the same tag name ("form") as that wrapper during HTML
// tokenization, and golang.org/x/net/html's standard parser silently drops
// a <form> start tag encountered while already inside another <form> (the
// same nested-form suppression real browsers apply) — before this engine's
// own component resolution ever sees the node. Confirms both the
// explanation and the fix (the auto-registered radix-form kebab-case
// alias, which does not collide with the literal tag name) are documented,
// and that the header's own usage examples actually use <radix-form>, not
// just describe the fix in prose.
func TestForm_DocumentsNestedFormAliasRequirement(t *testing.T) {
	src := readForm(t)
	header := headerCommentForm(t, src)

	for _, marker := range []string{
		"radix-form",
		"nested",
		"golang.org/x/net/html",
	} {
		if !strings.Contains(header, marker) {
			t.Errorf("Form.vue header comment missing expected marker %q documenting the nested <Form>-reference trap", marker)
		}
	}
	if !strings.Contains(header, "<radix-form") {
		t.Errorf("Form.vue header comment's usage examples must use <radix-form>, not <Form>, for field references nested inside a literal <form>")
	}
	if strings.Contains(header, "<Form id=") {
		t.Errorf("Form.vue header comment still shows a literal <Form id=...> usage example, which silently fails to render when nested inside a wrapping <form> — should be <radix-form id=...>")
	}
}

// TestForm_DocumentsDialogFix confirms the header comment documents that
// Dialog.vue's/AlertDialog.vue's own internal <form method="dialog">
// elements needed the same v-native fix once Form.vue was added to the
// same package — a real regression this commit both caused and fixed, not
// swept under the rug.
func TestForm_DocumentsDialogFix(t *testing.T) {
	src := readForm(t)
	header := headerCommentForm(t, src)

	for _, marker := range []string{
		"Dialog.vue",
		"AlertDialog.vue",
	} {
		if !strings.Contains(header, marker) {
			t.Errorf("Form.vue header comment missing expected marker %q documenting the Dialog/AlertDialog v-native fix", marker)
		}
	}
}

// TestDialog_CloseFormIsVNative confirms Dialog.vue's own zero-JS close
// <form method="dialog"> is marked v-native, so it keeps rendering as a
// real native <form> (required for the native close-the-nearest-ancestor-
// dialog behavior to work at all) now that Form.vue's own name registers a
// competing lowercase "form" alias in the same package's component
// registry. Regression test for the collision this commit's own process
// caught by actually rendering Dialog.vue through a full htmlc.Engine with
// the complete ui/radix package mounted.
func TestDialog_CloseFormIsVNative(t *testing.T) {
	data, err := fs.ReadFile(FS(), "components/Dialog.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Dialog.vue) failed: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, `<form v-native method="dialog"`) {
		t.Errorf(`Dialog.vue missing expected marker <form v-native method="dialog">; source was:\n%s`, src)
	}
}

// TestAlertDialog_CancelFormIsVNative is TestDialog_CloseFormIsVNative's
// counterpart for AlertDialog.vue's own <form method="dialog"> Cancel
// mechanism.
func TestAlertDialog_CancelFormIsVNative(t *testing.T) {
	data, err := fs.ReadFile(FS(), "components/AlertDialog.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/AlertDialog.vue) failed: %v", err)
	}
	src := string(data)
	if !strings.Contains(src, `<form v-native method="dialog"`) {
		t.Errorf(`AlertDialog.vue missing expected marker <form v-native method="dialog">; source was:\n%s`, src)
	}
}

// TestForm_RequiredPropsDocumented confirms the header comment documents
// id/label/error as required props, including the "error=” means no
// error" convention this package's lack of optional-prop defaults forces
// (matching Toast.vue's own title/description precedent).
func TestForm_RequiredPropsDocumented(t *testing.T) {
	src := readForm(t)
	header := headerCommentForm(t, src)

	for _, marker := range []string{
		"id:",
		"label:",
		"error:",
		`error=""`,
	} {
		if !strings.Contains(header, marker) {
			t.Errorf("Form.vue header comment missing expected required-prop documentation marker %q", marker)
		}
	}
}

// TestForm_DataInvalidAttribute confirms the wrapper div exposes a
// data-invalid styling hook driven by the same "error" truthiness the
// message <p> itself is gated by, mirroring real Radix's own
// FormField/FormLabel data-invalid convention (verified by reading
// form.tsx's getInvalidAttribute) as a small, additive, non-required
// styling affordance.
func TestForm_DataInvalidAttribute(t *testing.T) {
	src := readForm(t)
	template := templateBlockForm(t, src)

	if !strings.Contains(template, `:data-invalid="error ? '' : undefined"`) {
		t.Errorf(`Form.vue wrapper missing expected marker :data-invalid="error ? '' : undefined"; template was:\n%s`, template)
	}
}

// labelElementForm extracts the single <label ...>...</label> element's
// source text so assertions can be scoped to it specifically, instead of
// matching markers anywhere in the whole file (including the header
// comment, which discusses the same attributes in prose).
func labelElementForm(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<label")
	if start == -1 {
		t.Fatalf("Form.vue missing expected <label> element; source was:\n%s", src)
	}
	end := strings.Index(src[start:], "</label>")
	if end == -1 {
		t.Fatalf("Form.vue could not find closing </label>; source was:\n%s", src)
	}
	return src[start : start+end+len("</label>")]
}

// templateBlockForm extracts the <template>...</template> block's source
// text so assertions can be scoped to actual markup, excluding the header
// comment that precedes it, matching Progress.vue's own templateBlock
// helper convention.
func templateBlockForm(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<template>")
	end := strings.Index(src, "</template>")
	if start == -1 || end == -1 {
		t.Fatalf("Form.vue missing <template>...</template> block; source was:\n%s", src)
	}
	return src[start : end+len("</template>")]
}

// headerCommentForm extracts the leading <!-- ... --> header comment block's
// source text, so assertions about documented conventions can be scoped to
// it specifically.
func headerCommentForm(t *testing.T, src string) string {
	t.Helper()
	start := strings.Index(src, "<!--")
	end := strings.Index(src, "-->")
	if start == -1 || end == -1 {
		t.Fatalf("Form.vue missing header comment block; source was:\n%s", src)
	}
	return src[start : end+len("-->")]
}

func readForm(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Form.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Form.vue) failed: %v", err)
	}
	return string(data)
}
