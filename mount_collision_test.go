package htmlc

import (
	"context"
	"sort"
	"strings"
	"testing"
	"testing/fstest"
)

// TestValidateAll_MountCollision_Example3 is the RFC 014 §6 Example 3
// scenario: a local component shadows a mount-provided component of the same
// bare name via proximity for every current caller, so nothing is broken
// today — yet ValidateAll() must still report the flat-registry name as
// ambiguous, since a future template added outside the shadowing directory's
// proximity reach would silently fall through to whichever source wins
// lexical order, with no error. This is the single most important test in
// this commit: it is the one place the RFC's design deliberately reports an
// error even though nothing is currently broken.
func TestValidateAll_MountCollision_Example3(t *testing.T) {
	primary := fstest.MapFS{
		"templates/Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="local">local accordion</div></template>`)},
		"templates/HomePage.vue":  &fstest.MapFile{Data: []byte(`<template><Accordion /></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="vendored">vendored accordion</div></template>`)},
	}

	e, err := New(Options{
		ComponentDir: "templates",
		FS:           primary,
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// The shadowed reference currently works fine via proximity for every
	// caller inside templates/: <Accordion/> in HomePage.vue resolves to the
	// local component, not the vendored one.
	out, err := e.RenderFragmentString(context.Background(), "HomePage", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString(HomePage): %v", err)
	}
	if !strings.Contains(out, "local accordion") {
		t.Errorf("got %q, want the local Accordion to render via proximity, unaffected by the mount", out)
	}

	// Despite that, ValidateAll() must still report the flat-registry name
	// "Accordion" as ambiguous across the primary source and the "radix"
	// mount — this is a blanket check on the flat registry itself, not a
	// per-caller shadowing analysis.
	errs := e.ValidateAll()
	found := false
	for _, ve := range errs {
		if ve.Component == "Accordion" && strings.Contains(ve.Message, "ambiguous") {
			found = true
			if !strings.Contains(ve.Message, `"Accordion"`) {
				t.Errorf("ValidateAll message %q: expected it to name the ambiguous component", ve.Message)
			}
			if !strings.Contains(ve.Message, "radix") {
				t.Errorf("ValidateAll message %q: expected it to mention the contributing mount %q", ve.Message, "radix")
			}
			if !strings.Contains(ve.Message, `<component is="prefix/Name">`) && !strings.Contains(ve.Message, "PrefixName") {
				t.Errorf("ValidateAll message %q: expected a concrete disambiguation hint (alias or <component is=...> syntax)", ve.Message)
			}
		}
	}
	if !found {
		t.Fatalf("ValidateAll() = %v, want a collision error for \"Accordion\" even though it currently renders fine via proximity", errs)
	}
}

// TestValidateAll_MountCollision_TwoMounts_NoLocal confirms a collision is
// reported when two mounts (no local file at all) both provide the same bare
// name.
func TestValidateAll_MountCollision_TwoMounts_NoLocal(t *testing.T) {
	first := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div>first</div></template>`)},
	}
	second := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div>second</div></template>`)},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "first", FS: first},
			{Prefix: "second", FS: second},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := e.ValidateAll()
	var msg string
	for _, ve := range errs {
		if ve.Component == "Accordion" {
			msg = ve.Message
		}
	}
	if msg == "" {
		t.Fatalf("ValidateAll() = %v, want a collision error for \"Accordion\" (two mounts, no local file)", errs)
	}
	if !strings.Contains(msg, "first") || !strings.Contains(msg, "second") {
		t.Errorf("ValidateAll message %q: expected it to mention both contributing mounts %q and %q", msg, "first", "second")
	}
}

// TestValidateAll_MountCollision_AliasCollision extends the previous
// commit's TestMountAlias_CrossMountCollision (mount_alias_test.go) scenario
// — two mounts whose automatically-derived aliases collide with each other —
// with the ValidateAll() assertion that test only set up the data for but
// never itself checked (it only asserted entryMountIDs directly). This is
// the "known collision scenario" the commit-6 task calls out as needing to
// be tested against ValidateAll(), not just entryMountIDs.
func TestValidateAll_MountCollision_AliasCollision(t *testing.T) {
	radix := fstest.MapFS{
		"dialog/TriggerButton.vue": &fstest.MapFile{Data: []byte(`<template><div>from radix/dialog</div></template>`)},
	}
	radixDialogTrigger := fstest.MapFS{
		"Button.vue": &fstest.MapFile{Data: []byte(`<template><div>from radixDialogTrigger</div></template>`)},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
			{Prefix: "radixDialogTrigger", FS: radixDialogTrigger},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := e.ValidateAll()
	var msg string
	for _, ve := range errs {
		if ve.Component == "RadixDialogTriggerButton" {
			msg = ve.Message
		}
	}
	if msg == "" {
		t.Fatalf("ValidateAll() = %v, want a collision error for the colliding alias \"RadixDialogTriggerButton\"", errs)
	}
	if !strings.Contains(msg, "radix") || !strings.Contains(msg, "radixDialogTrigger") {
		t.Errorf("ValidateAll message %q: expected it to mention both contributing mounts %q and %q", msg, "radix", "radixDialogTrigger")
	}

	// The kebab-case alias also collides and must be reported independently.
	var kebabMsg string
	for _, ve := range errs {
		if ve.Component == "radix-dialog-trigger-button" {
			kebabMsg = ve.Message
		}
	}
	if kebabMsg == "" {
		t.Errorf("ValidateAll() = %v, want a collision error for the colliding kebab alias \"radix-dialog-trigger-button\" too", errs)
	}

	// Bare names ("TriggerButton", "Button") do not collide with each other
	// or with anything else, so they must not be reported.
	for _, ve := range errs {
		if ve.Component == "TriggerButton" || ve.Component == "Button" {
			t.Errorf("ValidateAll() unexpectedly reported a collision for non-colliding bare name %q: %v", ve.Component, ve)
		}
	}
}

// TestValidateAll_NoMounts_Unchanged is a regression test: a single-source
// project (no Mounts at all) must see the exact same ValidateAll() behavior
// and error count as before this commit — not just "no crash".
func TestValidateAll_NoMounts_Unchanged(t *testing.T) {
	primary := fstest.MapFS{
		"Good.vue": &fstest.MapFile{Data: []byte(`<template><div>fine</div></template>`)},
		"Bad.vue":  &fstest.MapFile{Data: []byte(`<template><missing-thing></missing-thing></template>`)},
	}

	e, err := New(Options{FS: primary, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(e.opts.Mounts) != 0 {
		t.Fatalf("opts.Mounts = %v, want empty for this test", e.opts.Mounts)
	}

	errs := e.ValidateAll()
	if len(errs) != 1 {
		t.Fatalf("ValidateAll() = %v (len %d), want exactly 1 pre-existing unknown-component error, unaffected by this commit", errs, len(errs))
	}
	// The engine registers both "Bad" and "bad" (lowercase alias) as the
	// same *engineEntry; either name may appear here depending on map
	// iteration order (see TestIntegration_SelfClosingComponentWarning).
	if !strings.EqualFold(errs[0].Component, "Bad") || !strings.Contains(errs[0].Message, "missing-thing") {
		t.Errorf("ValidateAll()[0] = %+v, want the pre-existing unknown-component error for Bad/missing-thing", errs[0])
	}
	for _, ve := range errs {
		if strings.Contains(ve.Message, "ambiguous") {
			t.Errorf("ValidateAll() reported an ambiguity error %+v in a project with no Mounts at all", ve)
		}
	}
}

// TestValidateAll_MountsSet_NoCollisions confirms that when Mounts is set
// but no name actually collides, ValidateAll() reports no new errors — only
// pre-existing checks (e.g. broken references) still apply, exactly as
// before this commit.
func TestValidateAll_MountsSet_NoCollisions(t *testing.T) {
	primary := fstest.MapFS{
		"Page.vue": &fstest.MapFile{Data: []byte(`<template><Card></Card><broken-thing></broken-thing></template>`)},
		"Card.vue": &fstest.MapFile{Data: []byte(`<template><div>card</div></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div>accordion</div></template>`)},
	}

	e, err := New(Options{
		ComponentDir: ".",
		FS:           primary,
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	errs := e.ValidateAll()
	for _, ve := range errs {
		if strings.Contains(ve.Message, "ambiguous") {
			t.Errorf("ValidateAll() reported an unexpected ambiguity error %+v in a project with no actual name collisions", ve)
		}
	}
	// The pre-existing broken-reference check must still fire unaffected.
	// The engine registers both "Page" and "page" (lowercase alias) as the
	// same *engineEntry; either name may appear here depending on map
	// iteration order (see TestIntegration_SelfClosingComponentWarning).
	foundBroken := false
	for _, ve := range errs {
		if strings.EqualFold(ve.Component, "Page") && strings.Contains(ve.Message, "broken-thing") {
			foundBroken = true
		}
	}
	if !foundBroken {
		t.Errorf("ValidateAll() = %v, want the pre-existing broken-reference check for <broken-thing/> to still fire", errs)
	}
}

// TestValidateAll_MountCollision_Deterministic confirms multiple colliding
// names are reported in a stable, sorted order across repeated calls, so
// tests relying on ValidateAll() output are not flaky.
func TestValidateAll_MountCollision_Deterministic(t *testing.T) {
	primary := fstest.MapFS{
		"Widget.vue":    &fstest.MapFile{Data: []byte(`<template><div>local widget</div></template>`)},
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div>local accordion</div></template>`)},
	}
	radix := fstest.MapFS{
		"Widget.vue":    &fstest.MapFile{Data: []byte(`<template><div>mounted widget</div></template>`)},
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div>mounted accordion</div></template>`)},
	}

	e, err := New(Options{
		ComponentDir: ".",
		FS:           primary,
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var names []string
	for i := 0; i < 5; i++ {
		errs := e.ValidateAll()
		var collisionNames []string
		for _, ve := range errs {
			if strings.Contains(ve.Message, "ambiguous") {
				collisionNames = append(collisionNames, ve.Component)
			}
		}
		if i == 0 {
			names = collisionNames
			// Both PascalCase names collide (their automatic all-lowercase
			// fallbacks collide too, since those are additional flat-registry
			// names attempted by both sources) — assert the PascalCase pair
			// is present rather than hardcoding the exact set.
			hasAccordion, hasWidget := false, false
			for _, n := range names {
				switch n {
				case "Accordion":
					hasAccordion = true
				case "Widget":
					hasWidget = true
				}
			}
			if !hasAccordion || !hasWidget {
				t.Fatalf("collisionNames = %v, want it to include both \"Accordion\" and \"Widget\"", names)
			}
			if !sort.StringsAreSorted(names) {
				t.Errorf("collision errors = %v, want sorted order for deterministic output", names)
			}
			continue
		}
		if strings.Join(collisionNames, ",") != strings.Join(names, ",") {
			t.Errorf("ValidateAll() collision ordering changed across calls: first=%v, this=%v", names, collisionNames)
		}
	}
}
