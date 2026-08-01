package main

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dhamidi/htmlc"
	radixui "github.com/dhamidi/htmlc/ui/radix"
)

// TestNewEngine_ValidateAll_NoErrors confirms the live demo app itself —
// the real "components" directory mounted alongside ui/radix under the
// "radix" prefix — is fully valid and collision-free. This is the demo's
// own proof that Options.Mounts, mount aliases, and the components/
// ComponentDir combine without any registration ambiguity or dangling
// reference in a real, non-synthetic project.
func TestNewEngine_ValidateAll_NoErrors(t *testing.T) {
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("newEngine: %v", err)
	}
	errs := engine.ValidateAll()
	if len(errs) != 0 {
		t.Fatalf("ValidateAll() = %v, want zero errors for the live demo app", errs)
	}
}

// TestHomePage_Render_AllFourReferenceForms renders the real HomePage and
// asserts the output actually contains markers proving each of the four
// RFC 014 §5 reference forms rendered the correct, real ui/radix component —
// not merely that the render call did not error.
func TestHomePage_Render_AllFourReferenceForms(t *testing.T) {
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("newEngine: %v", err)
	}
	out, err := engine.RenderPageString("HomePage", pageData())
	if err != nil {
		t.Fatalf("RenderPageString(HomePage): %v", err)
	}

	// Sanity: a real HTML page shell.
	if !strings.Contains(out, "<!DOCTYPE html>") {
		t.Errorf("expected output to start with a doctype")
	}

	// Form 1 — plain unqualified <Accordion>: resolves to the mounted
	// component, which wraps its output in the mount-prefixed custom-element
	// tag <radix-accordion> (RFC 014's tag-derivation fix, commit 8a087ed)
	// and renders the faqItems content passed to it.
	if !strings.Contains(out, "<radix-accordion>") {
		t.Errorf("expected <radix-accordion> wrapper tag in output (form 1: unqualified <Accordion>)")
	}
	if !strings.Contains(out, "How fast is shipping?") {
		t.Errorf("expected faqItems content from form 1's Accordion instance")
	}

	// Form 2 — PascalCase alias <RadixTabs>: resolves to the mounted Tabs
	// component, wrapped in <radix-tabs>.
	if !strings.Contains(out, "<radix-tabs>") {
		t.Errorf("expected <radix-tabs> wrapper tag in output (form 2: PascalCase alias RadixTabs)")
	}
	if !strings.Contains(out, "Radix-inspired, headless, zero-JS-baseline components for htmlc.") {
		t.Errorf("expected tabItems content from form 2's Tabs instance")
	}

	// Form 3 — kebab-case alias <radix-dialog>, used as a literal tag:
	// resolves to the mounted Dialog component, which itself wraps a
	// v-native <dialog> element. Confirms v-native's own <dialog> renders as
	// plain native HTML (no self-reference cycle) even in this real,
	// full-engine Options.Mounts + ComponentDir context.
	if !strings.Contains(out, "<radix-dialog>") {
		t.Errorf("expected <radix-dialog> wrapper tag in output (form 3: kebab-case alias radix-dialog)")
	}
	if !strings.Contains(out, `<dialog class="radix-dialog"`) {
		t.Errorf("expected the v-native <dialog> element inside the radix-dialog wrapper")
	}
	if !strings.Contains(out, "This dialog is referenced via its auto-registered kebab-case") {
		t.Errorf("expected the dialog's default-slot content to render")
	}

	// Form 4 — explicit <component is="radix/Accordion">: resolves to the
	// exact same mounted Accordion component as form 1, so it must also
	// render as <radix-accordion> — this time with moreFaqItems' content,
	// proving it is a genuinely independent render, not a cached copy of
	// form 1's output.
	if strings.Count(out, "<radix-accordion>") < 2 {
		t.Errorf("expected two separate <radix-accordion> instances (forms 1 and 4), got %d", strings.Count(out, "<radix-accordion>"))
	}
	if !strings.Contains(out, "Is there a warranty?") {
		t.Errorf("expected moreFaqItems content from form 4's explicit <component is=\"radix/Accordion\"> instance")
	}

	// The paragraph immediately after form 4's <component> tag must have
	// rendered — confirming the explicit-form section was not truncated.
	if !strings.Contains(out, `class="after-component"`) {
		t.Errorf("expected the paragraph following the explicit <component> reference to render")
	}
}

// TestHomePage_Render_ComponentGallery confirms the "Component Gallery"
// section (everything below the four RFC 014 reference forms) actually
// renders all 28 remaining ui/radix components — not just that the overall
// render call succeeds. Each marker below is that component's own root
// element class name (read directly from its .vue source, not guessed —
// most follow the kebab-cased-component-name convention, but Form.vue
// ("radix-form-field"), OneTimePasswordField.vue ("radix-otp-field"), and
// RadioGroup.vue ("radix-radiogroup") each chose a different root class of
// their own, confirmed by reading each file rather than assuming the
// pattern held universally). Accordion/Tabs/Dialog are already covered by
// TestHomePage_Render_AllFourReferenceForms above, so are not repeated here.
func TestHomePage_Render_ComponentGallery(t *testing.T) {
	engine, err := newEngine()
	if err != nil {
		t.Fatalf("newEngine: %v", err)
	}
	out, err := engine.RenderPageString("HomePage", pageData())
	if err != nil {
		t.Fatalf("RenderPageString(HomePage): %v", err)
	}

	markers := map[string]string{
		"AlertDialog":          `class="radix-alert-dialog"`,
		"AspectRatio":          `class="radix-aspect-ratio"`,
		"Avatar":               `class="radix-avatar"`,
		"Checkbox":             `class="radix-checkbox"`,
		"Collapsible":          `class="radix-collapsible"`,
		"ContextMenu":          `class="radix-context-menu"`,
		"DropdownMenu":         `class="radix-dropdown-menu"`,
		"Form":                 `class="radix-form-field"`,
		"HoverCard":            `class="radix-hover-card"`,
		"Label":                `class="radix-label"`,
		"Menubar":              `class="radix-menubar"`,
		"NavigationMenu":       `class="radix-navigation-menu"`,
		"OneTimePasswordField": `class="radix-otp-field"`,
		"PasswordToggleField":  `class="radix-password-toggle-field"`,
		"Popover":              `class="radix-popover"`,
		"Progress":             `class="radix-progress"`,
		"RadioGroup":           `class="radix-radiogroup"`,
		"ScrollArea":           `class="radix-scroll-area"`,
		"Select":               `class="radix-select"`,
		"Separator":            `class="radix-separator"`,
		"Slider":               `class="radix-slider"`,
		"Switch":               `class="radix-switch"`,
		"Toast":                `class="radix-toast"`,
		"Toggle":               `class="radix-toggle"`,
		"ToggleGroup":          `class="radix-toggle-group"`,
		"Toolbar":              `class="radix-toolbar"`,
		"Tooltip":              `class="radix-tooltip"`,
		"VisuallyHidden":       `class="radix-visually-hidden"`,
	}
	if len(markers) != 28 {
		t.Fatalf("test bug: expected 28 markers (one per non-Accordion/Tabs/Dialog ui/radix component), got %d", len(markers))
	}
	for component, marker := range markers {
		if !strings.Contains(out, marker) {
			t.Errorf("%s: expected marker %s in rendered gallery output, not found", component, marker)
		}
	}

	// No component in the gallery may have a missing required prop — this
	// package has no notion of an optional prop with a default, so any
	// unpassed one renders a visible, truthy "[missing: <name>]" placeholder
	// (component.go's validateProps) rather than failing the render outright.
	if strings.Contains(out, "[missing:") {
		t.Errorf("expected no \"[missing: <prop>]\" placeholders anywhere in the rendered gallery output")
	}
}

// TestSelfClosingComponentIs_DoesNotSwallowSiblings confirms, in isolation,
// the commit b8de838 fix that makes a self-closed
// <component is="..." /> safe to use (RFC 014 §4.7 item 1) — the exact
// syntax the README documents for explicit cross-directory references. The
// live demo page (HomePage.vue) deliberately does NOT self-close its own
// <component is="radix/Accordion"> reference, because doing so would add a
// Component.Warnings entry ("self-closing component tag(s) were
// auto-corrected") that ValidateAll() surfaces as a ValidationError,
// breaking TestNewEngine_ValidateAll_NoErrors above. This test proves the
// underlying bugfix independently, using its own isolated
// fstest.MapFS-backed engine, without that constraint: before the fix, the
// self-closing form left the tag open, and the parser folded every
// following sibling into it as a child — since the resolved target here has
// no <slot>, that swallowed subtree would simply vanish from the output.
func TestSelfClosingComponentIs_DoesNotSwallowSiblings(t *testing.T) {
	primary := fstest.MapFS{
		"HomePage.vue": &fstest.MapFile{Data: []byte(`<template>
  <section>
    <component is="radix/Accordion" :items="items" />
    <p class="after-self-closed">sibling after a self-closed component tag</p>
  </section>
</template>`)},
	}

	engine, err := htmlc.New(htmlc.Options{
		FS:           primary,
		ComponentDir: ".",
		Mounts: []htmlc.Mount{
			{Prefix: "radix", FS: radixui.FS(), Dir: "components"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	items := []any{
		map[string]any{"id": "x", "title": "Self-closed?", "content": "<p>yes</p>"},
	}
	out, err := engine.RenderFragmentString(context.Background(), "HomePage", map[string]any{"items": items})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}

	if !strings.Contains(out, "<radix-accordion>") {
		t.Errorf("expected the self-closed <component is=\"radix/Accordion\"/> to still render the mounted Accordion, got: %s", out)
	}
	if !strings.Contains(out, "Self-closed?") {
		t.Errorf("expected the Accordion's item content to render, got: %s", out)
	}
	if !strings.Contains(out, `class="after-self-closed"`) {
		t.Errorf("expected the sibling <p> after the self-closed <component> tag to render (not be swallowed as a child) — this is the exact regression commit b8de838 fixed, got: %s", out)
	}
}
