package htmlc

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
)

// This file regression-tests the fix for a bug where resolveComponent ran
// unconditionally *before* the Options.NativeElements/v-native check in the
// implicit tag-name resolution path (renderer.go, around the
// "// Component: resolve the tag name against the registry." comment). A
// component literally named Dialog auto-registers a lowercase "dialog"
// alias in the flat registry (the pre-existing entries[lower] = entry
// convention); a literal <dialog> tag inside Dialog's own template used to
// resolve right back to Dialog itself before the native check ever ran,
// producing an infinite self-reference ("cycle detected"), and no amount of
// v-native or Options.NativeElements could prevent it since that check was
// never reached. See docs/proposals/014-extension-mechanism.md §4.3.

// TestRender_VNative_PreventsSelfReferenceCycle_ComponentNamedAfterOwnTag is
// the exact bug scenario: a component named Dialog contains a literal
// <dialog v-native> in its own template. Before the fix this failed with
// "cycle detected"; after the fix it renders successfully because v-native
// causes resolveComponent to be skipped for that tag entirely.
func TestRender_VNative_PreventsSelfReferenceCycle_ComponentNamedAfterOwnTag(t *testing.T) {
	dialog := mustParseComponent(t, "Dialog.vue", `<dialog v-native class="d"><slot></slot></dialog>`)
	main := mustParseComponent(t, "main.vue", `<Dialog>hello</Dialog>`)

	out, err := NewRenderer(main).WithComponents(Registry{"Dialog": dialog}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<dialog") {
		t.Errorf("got %q, want a literal <dialog> element in the output", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("got %q, want slot content \"hello\"", out)
	}
	if strings.Contains(out, "v-native") {
		t.Errorf("got %q, want v-native stripped from output", out)
	}
}

// TestRender_NativeElementsOption_PreventsSelfReferenceCycle_ComponentNamedAfterOwnTag
// is the same scenario, but declares the tag native via the project-wide
// Options.NativeElements allowlist (WithNativeElements at the Renderer
// level) instead of the per-element v-native attribute, proving that path is
// equally fixed.
func TestRender_NativeElementsOption_PreventsSelfReferenceCycle_ComponentNamedAfterOwnTag(t *testing.T) {
	dialog := mustParseComponent(t, "Dialog.vue", `<dialog class="d"><slot></slot></dialog>`)
	main := mustParseComponent(t, "main.vue", `<Dialog>hello</Dialog>`)

	out, err := NewRenderer(main).WithComponents(Registry{"Dialog": dialog}).
		WithNativeElements([]string{"dialog"}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "<dialog") {
		t.Errorf("got %q, want a literal <dialog> element in the output", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("got %q, want slot content \"hello\"", out)
	}
}

// TestRender_WithoutNativeDeclaration_SelfReferencingTagStillCycles is the
// negative control: the same self-referencing Dialog/<dialog> shape, but
// with neither v-native nor Options.NativeElements declared, must still
// fail with a cycle error. This proves the fix only changes behavior for
// tags an author has *explicitly* declared native — it must not widen into
// treating every coincidentally-named tag as implicitly native.
func TestRender_WithoutNativeDeclaration_SelfReferencingTagStillCycles(t *testing.T) {
	dialog := mustParseComponent(t, "Dialog.vue", `<dialog class="d"><slot></slot></dialog>`)
	main := mustParseComponent(t, "main.vue", `<Dialog>hello</Dialog>`)

	_, err := NewRenderer(main).WithComponents(Registry{"Dialog": dialog}).RenderString(nil)
	if err == nil {
		t.Fatal("expected a cycle-detection error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle detected") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "cycle detected")
	}
}

// TestRender_ComponentResolution_StillWinsWhenNothingDeclaredNative confirms
// plain, undeclared component resolution is completely unaffected by the
// reordering: a tag with no native declaration anywhere still resolves
// against the registry exactly as before.
func TestRender_ComponentResolution_StillWinsWhenNothingDeclaredNative(t *testing.T) {
	widget := mustParseComponent(t, "Widget.vue", `<div class="widget">{{ label }}</div>`)
	main := mustParseComponent(t, "main.vue", `<Widget :label="'from registry'"></Widget>`)

	out, err := NewRenderer(main).WithComponents(Registry{"Widget": widget}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, `class="widget"`) || !strings.Contains(out, "from registry") {
		t.Errorf("got %q, want Widget's own rendering", out)
	}
}

// TestRender_ExplicitComponentIs_IgnoresNativeDeclaration_StillResolvesComponent
// verifies the explicit <component is="..."> / :is="..." path is genuinely
// untouched: even when the target tag name is declared native via
// Options.NativeElements, an explicit is="dialog" reference is a
// deliberate, unambiguous request for the registered component and must
// still resolve to it rather than falling through to a plain native
// element. This is the scenario called out in the RFC as one that should
// not be silently defeated by a native declaration.
func TestRender_ExplicitComponentIs_IgnoresNativeDeclaration_StillResolvesComponent(t *testing.T) {
	dialog := mustParseComponent(t, "Dialog.vue", `<div class="dialog-component-body">dialog component content</div>`)
	main := mustParseComponent(t, "main.vue", `<component is="dialog"></component>`)

	out, err := NewRenderer(main).WithComponents(Registry{"Dialog": dialog}).
		WithNativeElements([]string{"dialog"}).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(out, "dialog-component-body") {
		t.Errorf("got %q, want the resolved Dialog component's rendering, not a plain <dialog> element", out)
	}
	if strings.Contains(out, "<dialog") {
		t.Errorf("got %q, explicit is=\"dialog\" must resolve the component, not a native <dialog> element", out)
	}
}

// TestRender_UnknownComponent_StillErrors_NotWidenedByFix is a co-located
// sanity check (alongside TestRender_ComponentUnknown) that the reordering
// did not accidentally widen the native-declaration passthrough: a
// genuinely unregistered, non-native, hyphenated tag must still produce the
// "unknown component" error.
func TestRender_UnknownComponent_StillErrors_NotWidenedByFix(t *testing.T) {
	main := mustParseComponent(t, "main.vue", `<some-unregistered-widget></some-unregistered-widget>`)
	_, err := NewRenderer(main).WithComponents(Registry{}).RenderString(nil)
	if err == nil {
		t.Fatal("expected \"unknown component\" error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown component") {
		t.Errorf("error = %q, want it to contain %q", err.Error(), "unknown component")
	}
}

// TestEngine_VNative_PreventsSelfReferenceCycle_RealComponentLoadingShape
// reproduces the exact real-world shape of ui/radix/components/Dialog.vue:
// a component loaded from the filesystem via Options{FS, ComponentDir} (so
// it goes through the engine's real entries[lower] = entry auto-alias
// registration, not just a hand-built Registry{}), containing a literal
// self-referencing <dialog v-native> tag, composed by a caller template
// using ordinary <Dialog> syntax with slot content — a full RenderFragment
// round-trip, matching the mount_test.go-style pattern used elsewhere in
// this suite.
func TestEngine_VNative_PreventsSelfReferenceCycle_RealComponentLoadingShape(t *testing.T) {
	memFS := fstest.MapFS{
		"Dialog.vue": &fstest.MapFile{Data: []byte(
			`<template><dialog v-native class="radix-dialog" :open="open"><slot></slot></dialog></template>`,
		)},
		"Page.vue": &fstest.MapFile{Data: []byte(
			`<template><Dialog :open="true">dialog body</Dialog></template>`,
		)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Sanity: confirm the auto-lowercase alias this bug hinges on really is
	// registered by the engine's normal component-loading path.
	e.mu.RLock()
	_, hasLowerAlias := e.entries["dialog"]
	e.mu.RUnlock()
	if !hasLowerAlias {
		t.Fatal("engine did not auto-register the lowercase \"dialog\" alias for Dialog.vue; test no longer reproduces the bug's precondition")
	}

	out, err := e.RenderFragmentString(context.Background(), "Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "<dialog") {
		t.Errorf("got %q, want a literal <dialog> element in the output", out)
	}
	if !strings.Contains(out, "dialog body") {
		t.Errorf("got %q, want slot content \"dialog body\"", out)
	}
	if !strings.Contains(out, "open") {
		t.Errorf("got %q, want the open attribute rendered", out)
	}
}

// TestEngine_NativeElementsOption_PreventsSelfReferenceCycle_RealComponentLoadingShape
// is the Options.NativeElements variant of the above, at the full engine
// level: the component's own <dialog> tag carries no v-native, relying
// entirely on the project-wide allowlist instead.
func TestEngine_NativeElementsOption_PreventsSelfReferenceCycle_RealComponentLoadingShape(t *testing.T) {
	memFS := fstest.MapFS{
		"Dialog.vue": &fstest.MapFile{Data: []byte(
			`<template><dialog class="radix-dialog" :open="open"><slot></slot></dialog></template>`,
		)},
		"Page.vue": &fstest.MapFile{Data: []byte(
			`<template><Dialog :open="true">dialog body</Dialog></template>`,
		)},
	}

	e, err := New(Options{FS: memFS, ComponentDir: ".", NativeElements: []string{"dialog"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "<dialog") {
		t.Errorf("got %q, want a literal <dialog> element in the output", out)
	}
	if !strings.Contains(out, "dialog body") {
		t.Errorf("got %q, want slot content \"dialog body\"", out)
	}
}
