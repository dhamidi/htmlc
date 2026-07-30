package htmlc

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

// TestMounts_Baseline_NoRegression confirms Options{ComponentDir, FS} with no
// Mounts set behaves exactly as before Mounts existed — the existing engine
// tests already cover this extensively; this is a smoke test specific to the
// mount_test.go file so a reviewer scanning this file alone can see the
// baseline is exercised here too.
func TestMounts_Baseline_NoRegression(t *testing.T) {
	memFS := fstest.MapFS{
		"Card.vue": &fstest.MapFile{Data: []byte(`<template><div>{{ title }}</div></template>`)},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if len(e.opts.Mounts) != 0 {
		t.Fatalf("opts.Mounts = %v, want empty", e.opts.Mounts)
	}
	out, err := e.RenderFragmentString(context.Background(), "Card", map[string]any{"title": "hi"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "hi") {
		t.Errorf("got %q, want it to contain %q", out, "hi")
	}
}

// TestMounts_UnqualifiedReference_ResolvesMountedComponent is the core
// multi-source scenario: a primary template references a component that
// exists only in a mount, using ordinary unqualified tag syntax, and it must
// render via a full RenderFragment round-trip (not just registry
// inspection).
func TestMounts_UnqualifiedReference_ResolvesMountedComponent(t *testing.T) {
	primary := fstest.MapFS{
		"Page.vue": &fstest.MapFile{Data: []byte(`<template><div><Accordion :label="'from primary page'" /></div></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><section class="accordion">{{ label }}</section></template>`)},
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
	if !strings.Contains(out, "accordion") || !strings.Contains(out, "from primary page") {
		t.Errorf("got %q, want it to contain the mounted Accordion's rendering", out)
	}
}

// TestMounts_ExplicitPathIs confirms <component is="radix/Accordion">
// resolves the mounted component even when it would otherwise be ambiguous.
func TestMounts_ExplicitPathIs(t *testing.T) {
	primary := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="local-accordion">local</div></template>`)},
		"Page.vue":      &fstest.MapFile{Data: []byte(`<template><component is="radix/Accordion"></component></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="mounted-accordion">mounted</div></template>`)},
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
	if !strings.Contains(out, "mounted-accordion") {
		t.Errorf("got %q, want the mounted Accordion via explicit is=\"radix/Accordion\"", out)
	}
	if strings.Contains(out, "local-accordion") {
		t.Errorf("got %q, should not contain the local Accordion", out)
	}
}

// TestMounts_RegisterManual_MountAddressedPath confirms Register(name, path)
// works for a path living in a mount, reading through the union e.opts.FS —
// requiring zero code changes to Register itself.
func TestMounts_RegisterManual_MountAddressedPath(t *testing.T) {
	primary := fstest.MapFS{
		"Page.vue": &fstest.MapFile{Data: []byte(`<template><div>page</div></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="manual">manual registration</div></template>`)},
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

	if err := e.Register("ManualAccordion", "radix/Accordion.vue"); err != nil {
		t.Fatalf("Register: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "ManualAccordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "manual registration") {
		t.Errorf("got %q, want manually-registered mount component", out)
	}
}

// TestMounts_RegisterManual_TagDerivation_MatchesAutoDiscovery confirms a
// manually Register()'d mount path derives its custom-element tag the same
// way discoverMountInto's automatic walk would (including Mount.Prefix,
// e.g. "radix-accordion" not the hyphen-less, spec-invalid "accordion") —
// self-review finding: without mountID-aware handling in registerPathLocked,
// a manually registered mount path would derive its tag relative to
// ComponentDir (which never strips anything from a mount path), producing an
// inconsistent tag compared to the same file discovered automatically.
func TestMounts_RegisterManual_TagDerivation_MatchesAutoDiscovery(t *testing.T) {
	primary := fstest.MapFS{
		"templates/Page.vue": &fstest.MapFile{Data: []byte(`<template><div>page</div></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div></div></template>
<script customelement>
export default {}
</script>
`)},
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

	// Auto-discovered mount entry.
	e.mu.RLock()
	autoTag := e.entries["Accordion"].comp.CustomElementTag
	e.mu.RUnlock()
	if autoTag != "radix-accordion" {
		t.Fatalf("auto-discovered tag = %q, want %q (sanity check)", autoTag, "radix-accordion")
	}

	// Manually registered under a different flat name, same mount path.
	if err := e.Register("ManualAccordion", "radix/Accordion.vue"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	e.mu.RLock()
	manualTag := e.entries["ManualAccordion"].comp.CustomElementTag
	manualMountID := e.entries["ManualAccordion"].mountID
	e.mu.RUnlock()

	if manualTag != autoTag {
		t.Errorf("manually-registered tag = %q, want it to match auto-discovered tag %q", manualTag, autoTag)
	}
	if manualMountID != "radix" {
		t.Errorf("manually-registered entry mountID = %q, want %q", manualMountID, "radix")
	}
}

// TestMounts_MultipleMounts_EarliestWinsFlatSlot confirms that when two
// distinct mounts both define the same flat name, the earliest-processed
// mount (in Options.Mounts order) wins the flat-registry slot, and the
// losing mount's attempt is still recorded in entryMountIDs.
func TestMounts_MultipleMounts_EarliestWinsFlatSlot(t *testing.T) {
	first := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="first">first mount</div></template>`)},
	}
	second := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="second">second mount</div></template>`)},
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

	out, err := e.RenderFragmentString(context.Background(), "Accordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "first mount") {
		t.Errorf("got %q, want the earliest-processed mount (\"first\") to win the flat slot", out)
	}

	e.mu.RLock()
	ids := append([]string(nil), e.entryMountIDs["Accordion"]...)
	e.mu.RUnlock()
	if len(ids) != 2 || ids[0] != "first" || ids[1] != "second" {
		t.Errorf("entryMountIDs[Accordion] = %v, want [\"first\", \"second\"]", ids)
	}
}

// TestMounts_DuplicatePrefix_ReturnsImmediateError confirms New fails fast,
// synchronously, without deferring to ValidateAll.
func TestMounts_DuplicatePrefix_ReturnsImmediateError(t *testing.T) {
	a := fstest.MapFS{"A.vue": &fstest.MapFile{Data: []byte(`<template><div/></template>`)}}
	b := fstest.MapFS{"B.vue": &fstest.MapFile{Data: []byte(`<template><div/></template>`)}}

	_, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: a},
			{Prefix: "radix", FS: b},
		},
	})
	if err == nil {
		t.Fatal("New: got nil error, want a duplicate-prefix error")
	}
	if !strings.Contains(err.Error(), "radix") {
		t.Errorf("New err = %v, want it to mention the duplicate prefix %q", err, "radix")
	}
}

// TestMounts_InvalidPrefix_EmptyOrSlash confirms New rejects an empty prefix
// and a prefix containing "/".
func TestMounts_InvalidPrefix_EmptyOrSlash(t *testing.T) {
	fsys := fstest.MapFS{"A.vue": &fstest.MapFile{Data: []byte(`<template><div/></template>`)}}

	if _, err := New(Options{Mounts: []Mount{{Prefix: "", FS: fsys}}}); err == nil {
		t.Error("New with empty Prefix: got nil error, want an error")
	}
	if _, err := New(Options{Mounts: []Mount{{Prefix: "a/b", FS: fsys}}}); err == nil {
		t.Error("New with slash-containing Prefix: got nil error, want an error")
	}
}

// TestMounts_NilFS_ReturnsImmediateError confirms New rejects a Mount with a
// nil FS with a clean error instead of a later panic (self-review finding:
// fs.Sub(nil, dir) does not error eagerly; without this check the panic
// would only surface on first use, inside discoverMountInto's WalkDir).
func TestMounts_NilFS_ReturnsImmediateError(t *testing.T) {
	_, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: nil},
		},
	})
	if err == nil {
		t.Fatal("New: got nil error, want an error for a nil Mount.FS")
	}
}

// TestMounts_NestedSubdirectory_ProximityWithinMount confirms that a
// component nested inside a mount's own subdirectory resolves a sibling
// reference via the mount's own nsEntries namespace, without needing full
// qualification.
func TestMounts_NestedSubdirectory_ProximityWithinMount(t *testing.T) {
	primary := fstest.MapFS{
		"Page.vue": &fstest.MapFile{Data: []byte(`<template><component is="radix/dialog/Content"></component></template>`)},
	}
	radix := fstest.MapFS{
		"dialog/Content.vue": &fstest.MapFile{Data: []byte(`<template><div><Widget /></div></template>`)},
		"dialog/Widget.vue":  &fstest.MapFile{Data: []byte(`<template><span>nested-widget</span></template>`)},
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
	if !strings.Contains(out, "nested-widget") {
		t.Errorf("got %q, want the nested sibling Widget resolved without full qualification", out)
	}
}

// TestMounts_NestedSubdirectory_GenuineProximityDisambiguation is a stronger
// version of the sibling-resolution test above: it defines Widget.vue at
// *two* different depths within the same mount (mount root and a nested
// subdirectory) so that resolving <Widget> only succeeds correctly if
// genuine directory-based proximity is consulted — the flat-registry
// fallback alone cannot disambiguate between two same-named entries, so this
// test would fail (or resolve to the wrong one) if the mount's own
// nsEntries/callerDir wiring were not actually working.
func TestMounts_NestedSubdirectory_GenuineProximityDisambiguation(t *testing.T) {
	primary := fstest.MapFS{
		"Page.vue": &fstest.MapFile{Data: []byte(`<template><component is="radix/dialog/Content"></component></template>`)},
	}
	radix := fstest.MapFS{
		"Widget.vue":         &fstest.MapFile{Data: []byte(`<template><span>root-widget</span></template>`)},
		"dialog/Content.vue": &fstest.MapFile{Data: []byte(`<template><div><Widget /></div></template>`)},
		"dialog/Widget.vue":  &fstest.MapFile{Data: []byte(`<template><span>nested-widget</span></template>`)},
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
	if !strings.Contains(out, "nested-widget") {
		t.Errorf("got %q, want dialog/Content's <Widget> to resolve to dialog/Widget.vue (closest), not the mount root one", out)
	}
	if strings.Contains(out, "root-widget") {
		t.Errorf("got %q, want dialog/Content's <Widget> to prefer its own directory's Widget over the mount root one", out)
	}
}

// TestMounts_TagDerivation_UnaffectedByMounts is the regression proof for
// this commit's core design decision: adding Mounts to an existing working
// ComponentDir+FS configuration must not change the primary source's own
// derived custom-element tag. Resetting ComponentDir to "" for the whole
// union (as RFC 014 §4.1's simplified pseudocode suggests) would break this.
func TestMounts_TagDerivation_UnaffectedByMounts(t *testing.T) {
	const src = `<template><canvas></canvas></template>
<script customelement>
export default {}
</script>
`
	primaryWithoutMounts := fstest.MapFS{
		"templates/RaindropCanvas.vue": &fstest.MapFile{Data: []byte(src)},
	}
	withoutMounts, err := New(Options{FS: primaryWithoutMounts, ComponentDir: "templates"})
	if err != nil {
		t.Fatalf("New (no Mounts): %v", err)
	}
	withoutMounts.mu.RLock()
	tagWithout := withoutMounts.entries["RaindropCanvas"].comp.CustomElementTag
	withoutMounts.mu.RUnlock()

	primaryWithMounts := fstest.MapFS{
		"templates/RaindropCanvas.vue": &fstest.MapFile{Data: []byte(src)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div/></template>`)},
	}
	withMounts, err := New(Options{
		FS:           primaryWithMounts,
		ComponentDir: "templates",
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New (with Mounts): %v", err)
	}
	withMounts.mu.RLock()
	tagWith := withMounts.entries["RaindropCanvas"].comp.CustomElementTag
	withMounts.mu.RUnlock()

	if tagWithout != "raindrop-canvas" {
		t.Fatalf("tagWithout = %q, want %q (sanity check)", tagWithout, "raindrop-canvas")
	}
	if tagWith != tagWithout {
		t.Errorf("CustomElementTag with Mounts = %q, want unchanged from the no-Mounts case %q", tagWith, tagWithout)
	}
}

// TestMounts_MountOnly_NoPrimarySource confirms Options{FS: nil,
// ComponentDir: "", Mounts: [...]} works: Mounts function with zero primary
// configuration, and — because an empty ComponentDir skips the primary walk
// entirely — this does not require touching real OS files.
func TestMounts_MountOnly_NoPrimarySource(t *testing.T) {
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="mount-only">mount-only</div></template>`)},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Accordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "mount-only") {
		t.Errorf("got %q, want the mount-only Accordion", out)
	}
}

// TestMounts_Dir_ScansSubdirectoryOfMountFS confirms Mount.Dir selects a
// sub-path within Mount.FS to scan (mirroring ComponentDir), matching the
// documented Example 2 usage (Dir: "components").
func TestMounts_Dir_ScansSubdirectoryOfMountFS(t *testing.T) {
	radix := fstest.MapFS{
		"components/Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div>scoped</div></template>`)},
		"README.md":                &fstest.MapFile{Data: []byte("not a component")},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: radix, Dir: "components"},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	out, err := e.RenderFragmentString(context.Background(), "Accordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "scoped") {
		t.Errorf("got %q, want the Accordion from Mount.Dir=\"components\"", out)
	}
}

// TestMounts_LocalWinsOverMount confirms a local component of the same flat
// name always wins the flat-registry slot over a mount's own attempt — the
// insert-if-absent priority rule.
func TestMounts_LocalWinsOverMount(t *testing.T) {
	primary := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="local">local wins</div></template>`)},
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

	out, err := e.RenderFragmentString(context.Background(), "Accordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "local wins") {
		t.Errorf("got %q, want the local Accordion to win the flat-registry slot", out)
	}

	// The mounted component must still be reachable via its explicit path
	// despite losing the flat-registry slot.
	e.mu.RLock()
	_, mounted := e.nsEntries["radix"]["Accordion"]
	e.mu.RUnlock()
	if !mounted {
		t.Errorf("mounted Accordion should still be registered in nsEntries under \"radix\", just not the flat-registry winner")
	}
}

// TestMounts_EntryMountIDs_TracksBothAttempts confirms entryMountIDs records
// every distinct (name, mountID) pairing attempted during discovery,
// including the one that lost the flat-registry slot — this is the data a
// later commit's cross-mount collision detection will consume.
func TestMounts_EntryMountIDs_TracksBothAttempts(t *testing.T) {
	primary := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div/></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div/></template>`)},
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

	e.mu.RLock()
	ids := append([]string(nil), e.entryMountIDs["Accordion"]...)
	e.mu.RUnlock()

	if len(ids) != 2 || ids[0] != "" || ids[1] != "radix" {
		t.Errorf("entryMountIDs[Accordion] = %v, want [\"\", \"radix\"] (primary before mount, both attempts recorded)", ids)
	}
}

// TestMounts_Register_And_Reload_UnchangedForPrimary_AndWorkForMounts is the
// regression + new-capability test for §item 6: Register and Reload-
// triggered maybeReload continue to work unchanged for the primary source,
// and Register additionally works for a mount-addressed path through the
// union e.opts.FS, and a Reload-triggered rewalk (triggered by a *primary*
// file change) does not silently drop mount-sourced entries from the
// registry.
func TestMounts_Register_And_Reload_UnchangedForPrimary_AndWorkForMounts(t *testing.T) {
	primary := fstest.MapFS{
		"Live.vue": &fstest.MapFile{Data: []byte(`<template><p>original</p></template>`)},
	}
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(`<template><div class="mounted">mounted</div></template>`)},
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

	// Primary-source reload: unchanged behaviour.
	out, err := e.RenderFragmentString(context.Background(), "Live", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (before): %v", err)
	}
	if !strings.Contains(out, "original") {
		t.Errorf("before reload: got %q, want 'original'", out)
	}

	// Mount-addressed component resolves via the union before any reload.
	out, err = e.RenderFragmentString(context.Background(), "Accordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (mount, before reload): %v", err)
	}
	if !strings.Contains(out, "mounted") {
		t.Errorf("before reload: got %q, want mounted Accordion", out)
	}

	// Trigger a reload by modifying the *primary* file only.
	primary["Live.vue"] = &fstest.MapFile{
		Data:    []byte(`<template><p>updated</p></template>`),
		ModTime: time.Now().Add(time.Second),
	}

	out, err = e.RenderFragmentString(context.Background(), "Live", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (after reload): %v", err)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("after reload: got %q, want 'updated'", out)
	}

	// The mount-sourced component must still be resolvable after the
	// reload-triggered full rewalk — this is the regression this commit's
	// maybeReload fix guards against (an unfixed maybeReload would wipe
	// e.entries and only re-walk the primary ComponentDir, permanently
	// dropping every mount entry on the very next reload).
	out, err = e.RenderFragmentString(context.Background(), "Accordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (mount, after reload): %v", err)
	}
	if !strings.Contains(out, "mounted") {
		t.Errorf("after reload: got %q, want mounted Accordion still resolvable", out)
	}

	// Register(mount-addressed path) continues to work after reload too.
	if err := e.Register("ExplicitAccordion", "radix/Accordion.vue"); err != nil {
		t.Fatalf("Register: %v", err)
	}
	out, err = e.RenderFragmentString(context.Background(), "ExplicitAccordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString (Register mount path): %v", err)
	}
	if !strings.Contains(out, "mounted") {
		t.Errorf("got %q, want manually-registered mount component", out)
	}
}

// TestMounts_CustomElementTag_MatchesAlias is the end-to-end regression test
// for the bug fixed by this commit: discoverMountInto and registerPathLocked
// used to derive a mounted component's CustomElementTag by stripping
// Mount.Prefix from its path, producing e.g. "accordion" for a
// no-subdirectory mount component — a hyphen-less tag that is invalid per
// the Custom Elements spec (customElements.define('accordion', ...) throws
// in a real browser) and, per RFC 014 §4.1, must instead be the exact same
// string as the component's own automatic kebab-case alias.
//
// This test covers both the no-subdirectory case (radix/Accordion.vue,
// exactly the scenario that previously produced the invalid hyphen-less
// "accordion" tag) and the nested-subdirectory case (radix/dialog/Trigger.vue),
// confirming via a real RenderFragment round-trip that the rendered wrapper
// tag is "prefix-name" (matching the alias), and that ValidateAll() reports
// no hyphen-validity warning for either.
func TestMounts_CustomElementTag_MatchesAlias(t *testing.T) {
	const ceScript = `<script customelement>
export default {}
</script>
`
	radix := fstest.MapFS{
		"Accordion.vue": &fstest.MapFile{Data: []byte(
			`<template><div>accordion body</div></template>` + "\n" + ceScript)},
		"dialog/Trigger.vue": &fstest.MapFile{Data: []byte(
			`<template><button>trigger body</button></template>` + "\n" + ceScript)},
	}

	e, err := New(Options{
		Mounts: []Mount{
			{Prefix: "radix", FS: radix},
		},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// No-subdirectory case: radix/Accordion.vue.
	e.mu.RLock()
	accordionEntry := e.entries["Accordion"]
	accordionAliasEntry := e.entries["radix-accordion"]
	e.mu.RUnlock()
	if accordionEntry == nil || accordionAliasEntry == nil {
		t.Fatalf("expected both bare and alias entries for Accordion to be registered")
	}
	if accordionEntry != accordionAliasEntry {
		t.Fatalf("bare and alias entries for Accordion must be the same *engineEntry")
	}
	if got, want := accordionEntry.comp.CustomElementTag, "radix-accordion"; got != want {
		t.Errorf("Accordion CustomElementTag = %q, want %q (matching its own alias)", got, want)
	}
	if accordionEntry.comp.CustomElementTag != "radix-accordion" {
		t.Errorf("CustomElementTag must equal the kebab alias exactly: tag=%q alias=%q",
			accordionEntry.comp.CustomElementTag, "radix-accordion")
	}

	out, err := e.RenderFragmentString(context.Background(), "Accordion", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString(Accordion): %v", err)
	}
	if !strings.Contains(out, "<radix-accordion>") || !strings.Contains(out, "</radix-accordion>") {
		t.Errorf("rendered output = %q, want it wrapped in <radix-accordion>...</radix-accordion>", out)
	}
	if strings.Contains(out, "<accordion>") {
		t.Errorf("rendered output = %q, must not contain the old, invalid hyphen-less <accordion> wrapper", out)
	}

	// Nested-subdirectory case: radix/dialog/Trigger.vue.
	e.mu.RLock()
	triggerEntry := e.entries["Trigger"]
	triggerAliasEntry := e.entries["radix-dialog-trigger"]
	e.mu.RUnlock()
	if triggerEntry == nil || triggerAliasEntry == nil {
		t.Fatalf("expected both bare and alias entries for Trigger to be registered")
	}
	if triggerEntry != triggerAliasEntry {
		t.Fatalf("bare and alias entries for Trigger must be the same *engineEntry")
	}
	// Assert CustomElementTag == kebabAlias directly (string-equal, not just
	// "also has a hyphen").
	kebabAlias := "radix-dialog-trigger"
	if triggerEntry.comp.CustomElementTag != kebabAlias {
		t.Errorf("Trigger CustomElementTag = %q, want it to equal its own alias %q exactly",
			triggerEntry.comp.CustomElementTag, kebabAlias)
	}

	out, err = e.RenderFragmentString(context.Background(), "Trigger", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString(Trigger): %v", err)
	}
	if !strings.Contains(out, "<radix-dialog-trigger>") || !strings.Contains(out, "</radix-dialog-trigger>") {
		t.Errorf("rendered output = %q, want it wrapped in <radix-dialog-trigger>...</radix-dialog-trigger>", out)
	}

	// ValidateAll() must report no hyphen-validity warning for either
	// component — the old bug's stripped "accordion" tag would have tripped
	// customElementHyphenWarning (added by an earlier commit in this series).
	for _, ve := range e.ValidateAll() {
		if strings.Contains(ve.Message, "no hyphen") {
			t.Errorf("ValidateAll() reported an unexpected hyphen-validity warning: %+v", ve)
		}
	}
}
