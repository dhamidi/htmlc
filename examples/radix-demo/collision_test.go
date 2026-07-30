package main

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dhamidi/htmlc"
	radixui "github.com/dhamidi/htmlc/ui/radix"
)

// TestCollision_Example3_LocalShadowsButValidateAllReportsAmbiguity
// reproduces RFC 014 §6 Example 3 end-to-end, in complete isolation from the
// live demo app (its own htmlc.New call, its own fstest.MapFS-backed
// Options.FS/ComponentDir) — proving the collision scenario without
// requiring examples/radix-demo's real components/ directory to actually
// contain a colliding file.
//
// Scenario: a project defines its own local Accordion.vue (unrelated to
// ui/radix's) alongside a Mounts entry for ui/radix, which also defines
// Accordion. An unqualified <Accordion/> reference inside the project's own
// tree resolves to the *local* component via RFC 001 proximity — nothing is
// broken for that caller. But the flat-registry name "Accordion" is still
// ambiguous (it was contributed by two distinct mount identities: "" and
// "radix"), so ValidateAll() must report it as a hard error regardless.
func TestCollision_Example3_LocalShadowsButValidateAllReportsAmbiguity(t *testing.T) {
	primary := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="local-accordion">local, unrelated accordion</div></template>`)},
		"HomePage.vue":  &fstest.MapFile{Data: []byte(`<template><Accordion /></template>`)},
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

	// The unqualified <Accordion/> reference inside HomePage.vue resolves to
	// the local component via proximity — unaffected by the collision.
	out, err := engine.RenderFragmentString(context.Background(), "HomePage", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString(HomePage): %v", err)
	}
	if !strings.Contains(out, "local, unrelated accordion") {
		t.Errorf("got %q, want the local Accordion to render via proximity, unaffected by the mount collision", out)
	}
	if strings.Contains(out, "radix-accordion") {
		t.Errorf("got %q, did not expect the mounted ui/radix Accordion to render here — proximity should have picked the local one", out)
	}

	// Despite that, ValidateAll() must still report the flat-registry name
	// "Accordion" as ambiguous across the primary source ("") and the
	// "radix" mount — a blanket check on the flat registry itself, not a
	// per-caller shadowing analysis (RFC 014 §4.2).
	errs := engine.ValidateAll()
	var collision *htmlc.ValidationError
	for i := range errs {
		if errs[i].Component == "Accordion" && strings.Contains(errs[i].Message, "ambiguous") {
			collision = &errs[i]
			break
		}
	}
	if collision == nil {
		t.Fatalf("ValidateAll() = %v, want a collision error for %q even though it currently renders fine via proximity", errs, "Accordion")
	}
	if !strings.Contains(collision.Message, "radix") {
		t.Errorf("collision message %q should mention the contributing mount %q", collision.Message, "radix")
	}
}

// TestCollision_Example3_ResolvedViaAlias confirms the fix RFC 014 §6
// Example 3 prescribes: once the collision above is reported, every
// genuinely ambiguous call site can be disambiguated immediately using the
// mounted component's auto-registered alias (or the explicit
// <component is="radix/Name"> form) — no further design work required,
// since the aliases already exist by construction (§4.1).
func TestCollision_Example3_ResolvedViaAlias(t *testing.T) {
	tests := []struct {
		name string
		tag  string
	}{
		{"PascalCaseAlias", `<RadixAccordion :items="items" />`},
		{"KebabCaseAlias", `<radix-accordion :items="items"></radix-accordion>`},
		{"ExplicitIs", `<component is="radix/Accordion" :items="items"></component>`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			primary := fstest.MapFS{
				"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="local-accordion">local, unrelated accordion</div></template>`)},
				"HomePage.vue":  &fstest.MapFile{Data: []byte(`<template>` + tc.tag + `</template>`)},
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

			// The collision still exists in the flat registry — ValidateAll()
			// still reports it (the ambiguity is about the bare name
			// "Accordion", not about whether this particular template used
			// it), but that does not prevent this specific, disambiguated
			// caller from resolving and rendering correctly.
			foundCollision := false
			for _, ve := range engine.ValidateAll() {
				if ve.Component == "Accordion" && strings.Contains(ve.Message, "ambiguous") {
					foundCollision = true
				}
			}
			if !foundCollision {
				t.Fatalf("expected ValidateAll() to still report the Accordion collision even when this caller uses a disambiguated form")
			}

			items := []any{
				map[string]any{"id": "x", "title": "T", "content": "<p>C</p>"},
			}
			out, err := engine.RenderFragmentString(context.Background(), "HomePage", map[string]any{"items": items})
			if err != nil {
				t.Fatalf("RenderFragmentString(HomePage) with %s: %v", tc.name, err)
			}
			if !strings.Contains(out, "radix-accordion") {
				t.Errorf("%s: got %q, want the mounted ui/radix Accordion (wrapped in <radix-accordion>) to render, resolved unambiguously via the disambiguation form", tc.name, out)
			}
			if strings.Contains(out, "local-accordion") {
				t.Errorf("%s: got %q, did not expect the local Accordion to render — the disambiguation form must reach the mounted one", tc.name, out)
			}
		})
	}
}
