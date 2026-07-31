package radix

import (
	"io/fs"
	"strings"
	"testing"
)

// This module has no dependency on the root htmlc package, so a full
// render-based test (mounting AlertDialog.vue into a real htmlc.Engine and
// checking the rendered HTML) is out of scope here — see Dialog.vue's own
// test file for the same deferral rationale. These are content-sanity
// checks: they confirm the component's source file contains the markers
// this commit's design depends on.
func TestAlertDialog_ContainsZeroJSBaseline(t *testing.T) {
	src := readAlertDialog(t)

	for _, marker := range []string{
		"<template>",
		"<dialog",
		":open=\"open\"",
		"<slot",
		`method="dialog"`,
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("AlertDialog.vue missing expected baseline marker %q", marker)
		}
	}
}

// TestAlertDialog_HasExplicitAlertdialogRole confirms role="alertdialog" is
// set explicitly on the <dialog> element. Unlike role="dialog" (which a
// modally-shown <dialog> already implies, and which Dialog.vue therefore
// omits), there is no implicit "alertdialog" role for <dialog> — it must be
// authored explicitly or assistive tech announces this as a plain dialog.
func TestAlertDialog_HasExplicitAlertdialogRole(t *testing.T) {
	src := readAlertDialog(t)

	if !strings.Contains(src, `role="alertdialog"`) {
		t.Errorf(`AlertDialog.vue must set role="alertdialog" explicitly on its <dialog> element; got source:\n%s`, src)
	}
}

// TestAlertDialog_DialogTagDeclaredNative confirms the component's own
// <dialog> tag carries v-native, matching every other component in this
// package that has to guard against the same auto-registered-lowercase-
// alias self-reference cycle (see Dialog.vue's own test for the full
// explanation).
func TestAlertDialog_DialogTagDeclaredNative(t *testing.T) {
	src := readAlertDialog(t)

	if !strings.Contains(src, "<dialog v-native") {
		t.Errorf("AlertDialog.vue's own <dialog> tag must carry v-native to avoid a self-reference cycle against its own auto-registered lowercase alias; got source:\n%s", src)
	}
}

// TestAlertDialog_ContainsCustomElementEnhancement confirms the
// progressive-enhancement script carries both Dialog.vue's reused
// modal-promotion sequencing and this component's one genuinely new
// behavior: intercepting the native "cancel" event (fired on Escape) with
// preventDefault() so the alert dialog cannot be dismissed without an
// explicit choice.
func TestAlertDialog_ContainsCustomElementEnhancement(t *testing.T) {
	src := readAlertDialog(t)

	for _, marker := range []string{
		"<script customelement>",
		"customElements.define('radix-alert-dialog'",
		"showModal()",
		"removeAttribute('open')",
		"addEventListener('cancel'",
		"event.preventDefault()",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("AlertDialog.vue missing expected custom-element marker %q", marker)
		}
	}
}

// TestAlertDialog_CancelUsesFormMethodDialog confirms the built-in Cancel
// button closes via the same zero-JS <form method="dialog"> mechanism
// Dialog.vue's Close button uses, rather than a hand-rolled click handler.
func TestAlertDialog_CancelUsesFormMethodDialog(t *testing.T) {
	src := readAlertDialog(t)

	for _, marker := range []string{
		`<form method="dialog"`,
		"Cancel",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("AlertDialog.vue missing expected Cancel-button marker %q", marker)
		}
	}
}

// TestAlertDialog_HasActionsSlot confirms a dedicated "actions" slot exists
// for the caller-provided confirm/destructive action button, alongside the
// default slot used for title/description content.
func TestAlertDialog_HasActionsSlot(t *testing.T) {
	src := readAlertDialog(t)

	if !strings.Contains(src, `<slot name="actions">`) {
		t.Errorf(`AlertDialog.vue missing expected <slot name="actions"> marker; got source:\n%s`, src)
	}
}

func TestAlertDialog_ContainsScopedStyle(t *testing.T) {
	src := readAlertDialog(t)

	for _, marker := range []string{
		"<style scoped>",
		"::backdrop",
	} {
		if !strings.Contains(src, marker) {
			t.Errorf("AlertDialog.vue missing expected style marker %q", marker)
		}
	}
}

func readAlertDialog(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/AlertDialog.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/AlertDialog.vue) failed: %v", err)
	}
	return string(data)
}
