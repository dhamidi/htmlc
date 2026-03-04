package htmlc

import (
	"fmt"
	"hash/fnv"
	"strings"
	"testing"
)

// --- ScopeID tests ---

func TestScopeID_Deterministic(t *testing.T) {
	id1 := ScopeID("components/Card.vue")
	id2 := ScopeID("components/Card.vue")
	if id1 != id2 {
		t.Errorf("ScopeID not deterministic: %q != %q", id1, id2)
	}
}

func TestScopeID_Format(t *testing.T) {
	id := ScopeID("components/Card.vue")
	if !strings.HasPrefix(id, "data-v-") {
		t.Errorf("ScopeID %q should start with 'data-v-'", id)
	}
	hex := id[len("data-v-"):]
	if len(hex) != 8 {
		t.Errorf("ScopeID %q should have exactly 8 hex chars, got %d", id, len(hex))
	}
	for _, c := range hex {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("ScopeID %q contains non-hex char %q", id, c)
		}
	}
}

func TestScopeID_FNV1aOffsetBasis(t *testing.T) {
	// FNV-1a 32-bit hash of the empty string is the offset basis 0x811c9dc5.
	got := ScopeID("")
	want := "data-v-811c9dc5"
	if got != want {
		t.Errorf("ScopeID(\"\") = %q, want %q (FNV-1a 32-bit offset basis)", got, want)
	}
}

func TestScopeID_MatchesFNVPackage(t *testing.T) {
	// ScopeID must produce the same result as hash/fnv.New32a.
	path := "src/components/Button.vue"
	h := fnv.New32a()
	h.Write([]byte(path))
	want := fmt.Sprintf("data-v-%08x", h.Sum32())
	if got := ScopeID(path); got != want {
		t.Errorf("ScopeID(%q) = %q, want %q", path, got, want)
	}
}

func TestScopeID_DifferentPaths(t *testing.T) {
	a := ScopeID("components/A.vue")
	b := ScopeID("components/B.vue")
	if a == b {
		t.Errorf("ScopeID should differ for different paths, both got %q", a)
	}
}

// --- ScopeCSS tests ---

func TestScopeCSS_SimpleSelector(t *testing.T) {
	// ".card { ... }" becomes ".card[data-v-a1b2c3d4] { ... }"
	css := ".card { color: red; }"
	got := ScopeCSS(css, "[data-v-a1b2c3d4]")
	want := ".card[data-v-a1b2c3d4] { color: red; }"
	if got != want {
		t.Errorf("ScopeCSS simple selector:\ngot  %q\nwant %q", got, want)
	}
}

func TestScopeCSS_DescendantSelector(t *testing.T) {
	// "h2 .title { ... }" becomes "h2 .title[data-v-a1b2c3d4] { ... }"
	// The scope attr is appended to the last simple selector only.
	css := "h2 .title { font-size: 2em; }"
	got := ScopeCSS(css, "[data-v-a1b2c3d4]")
	want := "h2 .title[data-v-a1b2c3d4] { font-size: 2em; }"
	if got != want {
		t.Errorf("ScopeCSS descendant selector:\ngot  %q\nwant %q", got, want)
	}
}

func TestScopeCSS_MultipleRules(t *testing.T) {
	css := ".a { color: red; } .b { color: blue; }"
	got := ScopeCSS(css, "[data-v-a1b2c3d4]")
	if !strings.Contains(got, ".a[data-v-a1b2c3d4]") {
		t.Errorf("ScopeCSS: got %q, want .a[data-v-a1b2c3d4]", got)
	}
	if !strings.Contains(got, ".b[data-v-a1b2c3d4]") {
		t.Errorf("ScopeCSS: got %q, want .b[data-v-a1b2c3d4]", got)
	}
}

func TestScopeCSS_CommaSelectors(t *testing.T) {
	css := ".a, .b { margin: 0; }"
	got := ScopeCSS(css, "[data-v-a1b2c3d4]")
	if !strings.Contains(got, ".a[data-v-a1b2c3d4]") {
		t.Errorf("ScopeCSS comma selectors: got %q, want .a[data-v-a1b2c3d4]", got)
	}
	if !strings.Contains(got, ".b[data-v-a1b2c3d4]") {
		t.Errorf("ScopeCSS comma selectors: got %q, want .b[data-v-a1b2c3d4]", got)
	}
}

func TestScopeCSS_AtRulePassthrough(t *testing.T) {
	// @-rules must be passed through without modification.
	css := "@media (max-width: 600px) { .card { width: 100%; } }"
	got := ScopeCSS(css, "[data-v-a1b2c3d4]")
	if got != css {
		t.Errorf("ScopeCSS @-rule passthrough:\ngot  %q\nwant %q", got, css)
	}
}

func TestScopeCSS_EmptyScopeAttr(t *testing.T) {
	// An empty scopeAttr leaves the CSS structurally identical.
	css := ".global { color: blue; }"
	got := ScopeCSS(css, "")
	want := ".global { color: blue; }"
	if got != want {
		t.Errorf("ScopeCSS empty attr: got %q, want %q", got, want)
	}
}

// --- Scoped rendering: elements get the scope attribute ---

func TestRenderer_ScopedElementsHaveScopeAttr(t *testing.T) {
	src := `<template><div><p>hello</p></div></template><style scoped>.x{}</style>`
	c, err := ParseFile("Card.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := RenderString(c, nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	sid := ScopeID("Card.vue")
	// Both <div> and <p> should carry the scope attribute.
	if strings.Count(out, sid) < 2 {
		t.Errorf("rendered %q: want scope attr %q on every element (at least 2), got %d",
			out, sid, strings.Count(out, sid))
	}
}

func TestRenderer_ScopedVoidElementHasScopeAttr(t *testing.T) {
	src := `<template><img src="a.png"></template><style scoped>img{}</style>`
	c, err := ParseFile("Img.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := RenderString(c, nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	sid := ScopeID("Img.vue")
	if !strings.Contains(out, sid) {
		t.Errorf("rendered %q: void element should have scope attr %q", out, sid)
	}
}

func TestRenderer_GlobalComponentNoScopeAttr(t *testing.T) {
	src := `<template><div>hello</div></template><style>.g{}</style>`
	c, err := ParseFile("Global.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := RenderString(c, nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if strings.Contains(out, "data-v-") {
		t.Errorf("rendered %q: global component must not add scope attr", out)
	}
}

// --- StyleCollector tests ---

func TestStyleCollector_ScopedCSSRewritten(t *testing.T) {
	src := `<template><p>x</p></template><style scoped>.card { color: red; }</style>`
	c, err := ParseFile("Card.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	sc := &StyleCollector{}
	if _, err := NewRenderer(c).WithStyles(sc).RenderString(nil); err != nil {
		t.Fatalf("Render: %v", err)
	}
	contribs := sc.All()
	if len(contribs) != 1 {
		t.Fatalf("got %d contributions, want 1", len(contribs))
	}
	if contribs[0].ScopeID != ScopeID("Card.vue") {
		t.Errorf("ScopeID = %q, want %q", contribs[0].ScopeID, ScopeID("Card.vue"))
	}
	scopeAttr := "[" + ScopeID("Card.vue") + "]"
	if !strings.Contains(contribs[0].CSS, scopeAttr) {
		t.Errorf("scoped CSS %q should contain scope attr %q", contribs[0].CSS, scopeAttr)
	}
	if !strings.Contains(contribs[0].CSS, ".card") {
		t.Errorf("scoped CSS %q should still contain .card", contribs[0].CSS)
	}
}

func TestStyleCollector_GlobalCSSUnchanged(t *testing.T) {
	rawCSS := ".global { color: blue; }"
	src := "<template><p>x</p></template><style>" + rawCSS + "</style>"
	c, err := ParseFile("Global.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	sc := &StyleCollector{}
	if _, err := NewRenderer(c).WithStyles(sc).RenderString(nil); err != nil {
		t.Fatalf("Render: %v", err)
	}
	contribs := sc.All()
	if len(contribs) != 1 {
		t.Fatalf("got %d contributions, want 1", len(contribs))
	}
	if contribs[0].ScopeID != "" {
		t.Errorf("ScopeID = %q, want empty for global style", contribs[0].ScopeID)
	}
	if !strings.Contains(contribs[0].CSS, rawCSS) {
		t.Errorf("global CSS contribution %q should contain original CSS %q", contribs[0].CSS, rawCSS)
	}
}

func TestStyleCollector_MultipleComponents(t *testing.T) {
	src1 := `<template><p>a</p></template><style scoped>.a { color: red; }</style>`
	src2 := `<template><p>b</p></template><style>.b { color: blue; }</style>`

	c1, _ := ParseFile("A.vue", src1)
	c2, _ := ParseFile("B.vue", src2)

	sc := &StyleCollector{}
	NewRenderer(c1).WithStyles(sc).RenderString(nil) //nolint:errcheck
	NewRenderer(c2).WithStyles(sc).RenderString(nil) //nolint:errcheck

	got := sc.All()
	if len(got) != 2 {
		t.Fatalf("got %d contributions, want 2", len(got))
	}
	// First: scoped component A
	if got[0].ScopeID != ScopeID("A.vue") {
		t.Errorf("contribution[0].ScopeID = %q, want %q", got[0].ScopeID, ScopeID("A.vue"))
	}
	// Second: global component B
	if got[1].ScopeID != "" {
		t.Errorf("contribution[1].ScopeID = %q, want empty for global", got[1].ScopeID)
	}
}

func TestStyleCollector_NoStyleNoContribution(t *testing.T) {
	src := `<template><p>x</p></template>`
	c, err := ParseFile("NoStyle.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	sc := &StyleCollector{}
	if _, err := NewRenderer(c).WithStyles(sc).RenderString(nil); err != nil {
		t.Fatalf("Render: %v", err)
	}
	if len(sc.All()) != 0 {
		t.Errorf("got %d contributions, want 0 for component with no style", len(sc.All()))
	}
}

func TestStyleCollector_DeduplicatesSameComponent(t *testing.T) {
	// Rendering the same scoped component multiple times (e.g. via v-for) must
	// produce exactly one CSS contribution, not one per render.
	src := `<template><p>x</p></template><style scoped>.card { color: red; }</style>`
	c, err := ParseFile("Card.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	sc := &StyleCollector{}
	for range 3 {
		if _, err := NewRenderer(c).WithStyles(sc).RenderString(nil); err != nil {
			t.Fatalf("Render: %v", err)
		}
	}
	if got := len(sc.All()); got != 1 {
		t.Errorf("rendering same component 3 times: got %d contributions, want 1", got)
	}
}

func TestStyleCollector_DifferentComponentsBothKept(t *testing.T) {
	// Rendering two different components must produce two contributions.
	src1 := `<template><p>a</p></template><style scoped>.a { color: red; }</style>`
	src2 := `<template><p>b</p></template><style scoped>.b { color: blue; }</style>`
	c1, _ := ParseFile("A.vue", src1)
	c2, _ := ParseFile("B.vue", src2)

	sc := &StyleCollector{}
	NewRenderer(c1).WithStyles(sc).RenderString(nil) //nolint:errcheck
	NewRenderer(c2).WithStyles(sc).RenderString(nil) //nolint:errcheck
	// Render c1 again — should still be deduplicated.
	NewRenderer(c1).WithStyles(sc).RenderString(nil) //nolint:errcheck

	if got := len(sc.All()); got != 2 {
		t.Errorf("two different components: got %d contributions, want 2", got)
	}
}

func TestStyleCollector_DeduplicatesGlobalCSS(t *testing.T) {
	// The same global (unscoped) CSS block added twice must appear only once.
	rawCSS := ".global { color: blue; }"
	src := "<template><p>x</p></template><style>" + rawCSS + "</style>"
	c, err := ParseFile("Global.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	sc := &StyleCollector{}
	NewRenderer(c).WithStyles(sc).RenderString(nil) //nolint:errcheck
	NewRenderer(c).WithStyles(sc).RenderString(nil) //nolint:errcheck

	if got := len(sc.All()); got != 1 {
		t.Errorf("same global component rendered twice: got %d contributions, want 1", got)
	}
}

func TestStyleCollector_NilCollectorDoesNotPanic(t *testing.T) {
	src := `<template><p>x</p></template><style scoped>.x{}</style>`
	c, err := ParseFile("X.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	// No WithStyles call — styleCollector is nil.
	if _, err := NewRenderer(c).RenderString(nil); err != nil {
		t.Fatalf("Render with nil collector: %v", err)
	}
}
