package htmlc

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// TestMountAlias_NoSubdirectory confirms that a mount with no internal
// subdirectories gets both a PascalCase alias (RadixAccordion) and a
// kebab-case alias (radix-accordion), each resolving to the same mounted
// component via a real RenderFragment round-trip.
func TestMountAlias_NoSubdirectory(t *testing.T) {
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="accordion">mounted accordion</div></template>`)},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, tag := range []string{"RadixAccordion", "radix-accordion"} {
		t.Run(tag, func(t *testing.T) {
			out, err := e.RenderFragmentString(context.Background(), tag, nil)
			if err != nil {
				t.Fatalf("RenderFragmentString(%q): %v", tag, err)
			}
			if !strings.Contains(out, "mounted accordion") {
				t.Errorf("got %q, want it to contain the mounted Accordion's rendering", out)
			}
		})
	}

	// Both aliases must point to the exact same *engineEntry as the mount's
	// own bare-name registration, not independent copies.
	e.mu.RLock()
	bare := e.entries["Accordion"]
	pascal := e.entries["RadixAccordion"]
	kebab := e.entries["radix-accordion"]
	e.mu.RUnlock()
	if bare == nil || pascal == nil || kebab == nil {
		t.Fatalf("expected all three entries to be registered: bare=%v pascal=%v kebab=%v", bare, pascal, kebab)
	}
	if pascal != bare || kebab != bare {
		t.Errorf("alias entries must be the same *engineEntry as the bare mount registration: bare=%p pascal=%p kebab=%p", bare, pascal, kebab)
	}
}

// TestMountAlias_NestedSubdirectory confirms that a component nested inside
// a mount's own subdirectory aliases to the full-path form
// (RadixDialogTrigger / radix-dialog-trigger), not the shorter,
// collision-prone Prefix+baseName form (RadixTrigger / radix-trigger) — RFC
// 014 §10 Open Question 10.
func TestMountAlias_NestedSubdirectory(t *testing.T) {
	radix := fstest.MapFS{
		"dialog/Trigger.vue": &fstest.MapFile{Data: []byte(`<template><button>open</button></template>`)},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, tag := range []string{"RadixDialogTrigger", "radix-dialog-trigger"} {
		t.Run(tag, func(t *testing.T) {
			out, err := e.RenderFragmentString(context.Background(), tag, nil)
			if err != nil {
				t.Fatalf("RenderFragmentString(%q): %v", tag, err)
			}
			if !strings.Contains(out, "open") {
				t.Errorf("got %q, want it to contain the mounted Trigger's rendering", out)
			}
		})
	}

	// The naive, wrong short-form aliases must NOT be registered.
	e.mu.RLock()
	_, gotShortPascal := e.entries["RadixTrigger"]
	_, gotShortKebab := e.entries["radix-trigger"]
	e.mu.RUnlock()
	if gotShortPascal {
		t.Errorf("entries[\"RadixTrigger\"] should not exist; the alias must use the full path (RadixDialogTrigger)")
	}
	if gotShortKebab {
		t.Errorf("entries[\"radix-trigger\"] should not exist; the alias must use the full path (radix-dialog-trigger)")
	}
}

// TestMountAlias_NeverShadowsLocalRegistration confirms an exact local
// registration of the same name as a mount's alias always wins — aliases are
// insert-if-absent, just like the bare mount name.
func TestMountAlias_NeverShadowsLocalRegistration(t *testing.T) {
	primary := fstest.MapFS{
		"RadixAccordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="local">local wins</div></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="mounted">mounted loses</div></template>`)},
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

	out, err := e.RenderFragmentString(context.Background(), "RadixAccordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "local wins") {
		t.Errorf("got %q, want the local RadixAccordion to win over the mount's alias", out)
	}

	// The losing alias attempt must still be recorded for the upcoming
	// collision-detection commit.
	e.mu.RLock()
	ids := append([]string(nil), e.entryMountIDs["RadixAccordion"]...)
	e.mu.RUnlock()
	found := false
	for _, id := range ids {
		if id == "radix" {
			found = true
		}
	}
	if !found {
		t.Errorf("entryMountIDs[\"RadixAccordion\"] = %v, want it to include \"radix\" even though the alias lost the flat slot", ids)
	}
}

// TestMountAlias_CrossMountCollision confirms that two mounts whose derived
// *aliases* collide with each other — with no local file involved, and
// without their bare mount names colliding either, so this specifically
// isolates the alias-vs-alias case rather than the already-covered
// bare-name-vs-bare-name case (TestMounts_MultipleMounts_EarliestWinsFlatSlot,
// mount_test.go) — does not panic or error at this stage. Collision
// *detection* is a later commit; this only confirms entryMountIDs records
// both mount identities under the colliding alias name, which is the data
// that later commit needs.
//
// The collision is constructed, not contrived: mount "radix" containing
// "dialog/TriggerButton.vue" and mount "radixDialogTrigger" containing
// "Button.vue" both alias-derive to the identical
// "RadixDialogTriggerButton"/"radix-dialog-trigger-button" (verified by
// tracing deriveCustomElementTag/kebabToPascal directly), while their bare
// names ("TriggerButton" vs "Button") do not collide at all.
func TestMountAlias_CrossMountCollision(t *testing.T) {
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

	// Bare names do not collide.
	e.mu.RLock()
	_, hasTriggerButton := e.entries["TriggerButton"]
	_, hasButton := e.entries["Button"]
	ids := append([]string(nil), e.entryMountIDs["RadixDialogTriggerButton"]...)
	kebabIDs := append([]string(nil), e.entryMountIDs["radix-dialog-trigger-button"]...)
	e.mu.RUnlock()
	if !hasTriggerButton || !hasButton {
		t.Fatalf("expected both bare names registered independently: TriggerButton=%v Button=%v", hasTriggerButton, hasButton)
	}

	// The alias collision must not have panicked or errored (New succeeded
	// above), and both mount identities must be recorded for both the
	// PascalCase and kebab-case alias names.
	if len(ids) != 2 {
		t.Fatalf("entryMountIDs[\"RadixDialogTriggerButton\"] = %v, want two recorded attempts", ids)
	}
	seen := map[string]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	if !seen["radix"] || !seen["radixDialogTrigger"] {
		t.Errorf("entryMountIDs[\"RadixDialogTriggerButton\"] = %v, want it to include both \"radix\" and \"radixDialogTrigger\"", ids)
	}
	if len(kebabIDs) != 2 || !(kebabIDs[0] == "radix" && kebabIDs[1] == "radixDialogTrigger") {
		t.Errorf("entryMountIDs[\"radix-dialog-trigger-button\"] = %v, want [\"radix\", \"radixDialogTrigger\"] (earliest-processed mount first)", kebabIDs)
	}

	// The earliest-processed mount ("radix") wins the flat-registry slot for
	// the colliding alias.
	out, err := e.RenderFragmentString(context.Background(), "RadixDialogTriggerButton", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString(RadixDialogTriggerButton): %v", err)
	}
	if !strings.Contains(out, "from radix/dialog") {
		t.Errorf("got %q, want the earliest-processed mount (\"radix\") to win the colliding alias slot", out)
	}
}

// TestMountAlias_ExplicitPathStillWorks confirms the pre-existing
// <component is="mount/Name"> explicit-path form still works unchanged
// alongside the new automatic aliases.
func TestMountAlias_ExplicitPathStillWorks(t *testing.T) {
	primary := fstest.MapFS{
		"Page.vue": &fstest.MapFile{Data: []byte(`<template><component is="radix/Accordion"></component></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="mounted">mounted accordion</div></template>`)},
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

	out, err := e.RenderFragmentString(context.Background(), "Page", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "mounted accordion") {
		t.Errorf("got %q, want the explicit-path form to still resolve the mounted Accordion", out)
	}
}

// TestMountAlias_NotInNamespaceRegistry confirms aliases are flat-registry
// only: they must not appear as keys usable for proximity/namespace
// resolution (nsEntries), since an alias is not a real directory segment
// anywhere in the union tree.
func TestMountAlias_NotInNamespaceRegistry(t *testing.T) {
	radix := fstest.MapFS{
		"dialog/Trigger.vue": &fstest.MapFile{Data: []byte(`<template><button>open</button></template>`)},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	e.mu.RLock()
	defer e.mu.RUnlock()
	for relDir, byName := range e.nsEntries {
		for name := range byName {
			if name == "RadixDialogTrigger" || name == "radix-dialog-trigger" {
				t.Errorf("nsEntries[%q][%q] should not exist; aliases are flat-registry-only", relDir, name)
			}
		}
	}
}

// TestMountAlias_LowercaseFallback confirms the PascalCase alias also gets
// an automatic all-lowercase fallback registered, mirroring the existing
// local-component convention (registerInto: entries[lower] = entry whenever
// lower != name) for consistency across every PascalCase name in the
// registry.
func TestMountAlias_LowercaseFallback(t *testing.T) {
	radix := fstest.MapFS{
		"dialog/Trigger.vue": &fstest.MapFile{Data: []byte(`<template><button>open</button></template>`)},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	e.mu.RLock()
	pascal := e.entries["RadixDialogTrigger"]
	lower := e.entries["radixdialogtrigger"]
	e.mu.RUnlock()
	if pascal == nil {
		t.Fatalf("entries[\"RadixDialogTrigger\"] missing")
	}
	if lower == nil {
		t.Fatalf("entries[\"radixdialogtrigger\"] missing; want an automatic all-lowercase fallback for the PascalCase alias")
	}
	if lower != pascal {
		t.Errorf("lowercase fallback must point to the same *engineEntry as the PascalCase alias")
	}
}

// TestMountAlias_ReloadReplaysAliases confirms alias registration is wired
// into the same replay path as bare mount-name registration: a
// primary-source file change that triggers maybeReload's full mount rewalk
// must not drop the mount's aliases from the registry. Mirrors
// TestMounts_Register_And_Reload_UnchangedForPrimary_AndWorkForMounts's
// reload-triggering pattern (mount_test.go), which exists precisely to
// exercise this replay path with fstest.MapFS instead of the real OS
// filesystem.
func TestMountAlias_ReloadReplaysAliases(t *testing.T) {
	primary := fstest.MapFS{
		"Live.vue": &fstest.MapFile{Data: []byte(`<template><p>original</p></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div>mounted</div></template>`)},
	}

	e, err := New(Options{
		ComponentDir: ".",
		FS:           primary,
		Reload:       true,
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	e.mu.RLock()
	_, okBefore := e.entries["RadixAccordion"]
	e.mu.RUnlock()
	if !okBefore {
		t.Fatalf("entries[\"RadixAccordion\"] missing right after New")
	}

	// Trigger maybeReload's full rewalk by modifying the *primary* file
	// only (mtime bump), then reading through the engine, exactly like
	// TestMounts_Register_And_Reload_UnchangedForPrimary_AndWorkForMounts.
	primary["Live.vue"] = &fstest.MapFile{
		Data:    []byte(`<template><p>updated</p></template>`),
		ModTime: time.Now().Add(time.Second),
	}
	if _, err := e.RenderFragmentString(context.Background(), "Live", nil); err != nil {
		t.Fatalf("RenderFragmentString (trigger reload): %v", err)
	}

	e.mu.RLock()
	_, okAfter := e.entries["RadixAccordion"]
	e.mu.RUnlock()
	if !okAfter {
		t.Errorf("entries[\"RadixAccordion\"] missing after reload-triggered rewalk; alias registration must be replayed on every mount rewalk, not just done once at New")
	}

	out, err := e.RenderFragmentString(context.Background(), "RadixAccordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString(RadixAccordion) after reload: %v", err)
	}
	if !strings.Contains(out, "mounted") {
		t.Errorf("got %q, want the mounted Accordion via its alias after reload", out)
	}
}
