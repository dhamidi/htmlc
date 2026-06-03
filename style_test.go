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
	// Non-group @-rules (@font-face here) must be passed through unmodified.
	css := "@font-face { font-family: Foo; src: url(foo.woff2); }"
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

// --- CSS verbatim extraction tests ---

// TestExtractSections_StyleFontFaceQuotesPreserved verifies that extractSections
// returns the @font-face CSS body with double-quoted string values intact.
// Previously tok.Data (HTML-decoded) was used for TextTokens, which could mangle
// quoted values inside <style> blocks.
func TestExtractSections_StyleFontFaceQuotesPreserved(t *testing.T) {
	src := `<template><p>x</p></template>
<style>
@font-face {
  font-family: "My Font";
  src: url("font.woff2") format("woff2");
}
</style>`
	sections, err := extractSections(src)
	if err != nil {
		t.Fatalf("extractSections: %v", err)
	}
	css, ok := sections["style"]
	if !ok {
		t.Fatal("style section not found")
	}
	for _, want := range []string{`"My Font"`, `"font.woff2"`, `format("woff2")`} {
		if !strings.Contains(css, want) {
			t.Errorf("style section %q: want %q preserved verbatim", css, want)
		}
	}
}

// TestExtractSections_StyleSingleQuotesPreserved checks that single-quoted
// CSS string values are also kept verbatim.
func TestExtractSections_StyleSingleQuotesPreserved(t *testing.T) {
	src := `<template><p>x</p></template>
<style>
@font-face {
  font-family: 'My Font';
  src: url('font.woff2') format('woff2');
}
</style>`
	sections, err := extractSections(src)
	if err != nil {
		t.Fatalf("extractSections: %v", err)
	}
	css := sections["style"]
	for _, want := range []string{`'My Font'`, `'font.woff2'`, `format('woff2')`} {
		if !strings.Contains(css, want) {
			t.Errorf("style section %q: want %q preserved verbatim", css, want)
		}
	}
}

// TestExtractSections_StyleSpecialCharsPreserved checks that &, <, > in CSS
// content property values are not HTML-decoded or corrupted.
func TestExtractSections_StyleSpecialCharsPreserved(t *testing.T) {
	src := "<template><p>x</p></template>\n<style>\n.arrow::before { content: \"a > b & c < d\"; }\n</style>"
	sections, err := extractSections(src)
	if err != nil {
		t.Fatalf("extractSections: %v", err)
	}
	css := sections["style"]
	want := `"a > b & c < d"`
	if !strings.Contains(css, want) {
		t.Errorf("style section %q: want %q preserved verbatim", css, want)
	}
}

// TestRenderer_FontFaceStyleVerbatim verifies the end-to-end path: a component
// with a global <style> block containing an @font-face rule with quoted values
// must emit those values byte-for-byte in the rendered output.
func TestRenderer_FontFaceStyleVerbatim(t *testing.T) {
	src := `<template><p>x</p></template>
<style>
@font-face {
  font-family: "My Font";
  src: url("font.woff2") format("woff2");
}
</style>`
	c, err := ParseFile("Font.vue", src)
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
	for _, want := range []string{`"My Font"`, `"font.woff2"`, `format("woff2")`} {
		if !strings.Contains(contribs[0].CSS, want) {
			t.Errorf("CSS contribution %q: want %q preserved verbatim", contribs[0].CSS, want)
		}
	}
}

// TestRenderer_ScopedFontFaceStyleVerbatim verifies that a scoped <style> block
// with an @font-face rule emits quoted values verbatim (not rewritten, since
// @-rules are passed through by ScopeCSS).
func TestRenderer_ScopedFontFaceStyleVerbatim(t *testing.T) {
	src := `<template><p>x</p></template>
<style scoped>
@font-face {
  font-family: "My Font";
  src: url("font.woff2") format("woff2");
}
.text { font-family: "My Font"; }
</style>`
	c, err := ParseFile("ScopedFont.vue", src)
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
	// @font-face must be passed through verbatim by ScopeCSS.
	for _, want := range []string{`"My Font"`, `"font.woff2"`, `format("woff2")`} {
		if !strings.Contains(contribs[0].CSS, want) {
			t.Errorf("scoped CSS contribution %q: want %q preserved verbatim", contribs[0].CSS, want)
		}
	}
}

// TestScopeCSS_PseudoSelectors verifies that pseudo-class and pseudo-element
// selectors are preserved and the scope attribute is appended after them.
// This is a useful edge case because pseudo-selectors include special characters
// like ':', '::', and '(' that must not confuse the selector rewriter.
func TestScopeCSS_PseudoSelectors(t *testing.T) {
	cases := []struct {
		selector string
		want     string
	}{
		// Pseudo-elements use :: and must retain both colons.
		{"p::before", "p::before[data-v-abc]"},
		{"p::after", "p::after[data-v-abc]"},
		// Pseudo-classes use a single colon.
		{"a:hover", "a:hover[data-v-abc]"},
		{"input:focus", "input:focus[data-v-abc]"},
		// Functional pseudo-class with parenthesised argument.
		{"li:nth-child(2n+1)", "li:nth-child(2n+1)[data-v-abc]"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.selector, func(t *testing.T) {
			css := tc.selector + " { color: red; }"
			got := ScopeCSS(css, "[data-v-abc]")
			want := tc.want + " { color: red; }"
			if got != want {
				t.Errorf("ScopeCSS(%q):\ngot  %q\nwant %q", css, got, want)
			}
		})
	}
}

// TestScopeCSS_CombinatorSelectors verifies that combinator selectors (>, space,
// +, ~) have the scope attribute appended to the last simple selector only.
// This is a useful edge case because the rewriter must not split on spaces or
// combinator characters.
func TestScopeCSS_CombinatorSelectors(t *testing.T) {
	cases := []struct {
		name     string
		selector string
		want     string
	}{
		// Child combinator: scope goes on <span>, not <div>.
		{"child >", "div > span", "div > span[data-v-abc]"},
		// Descendant combinator (space): scope goes on <li>, not <ul>.
		{"descendant space", "ul li", "ul li[data-v-abc]"},
		// Adjacent sibling combinator.
		{"adjacent sibling +", "a + b", "a + b[data-v-abc]"},
		// General sibling combinator.
		{"general sibling ~", "a ~ b", "a ~ b[data-v-abc]"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			css := tc.selector + " { color: red; }"
			got := ScopeCSS(css, "[data-v-abc]")
			want := tc.want + " { color: red; }"
			if got != want {
				t.Errorf("ScopeCSS(%q):\ngot  %q\nwant %q", css, got, want)
			}
		})
	}
}

// TestScopeCSS_TenRules verifies that a stylesheet with many rules has all of
// them rewritten and none are lost.  This exercises the loop in ScopeCSS for
// more than one or two rules.
func TestScopeCSS_TenRules(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&b, ".rule%d { color: red; } ", i)
	}
	css := b.String()
	got := ScopeCSS(css, "[data-v-abc]")
	for i := 0; i < 10; i++ {
		want := fmt.Sprintf(".rule%d[data-v-abc]", i)
		if !strings.Contains(got, want) {
			t.Errorf("ScopeCSS 10 rules: output %q missing %q", got, want)
		}
	}
}

// TestScopeCSS_EmptyBody verifies that a rule with an empty declaration block
// is preserved intact.  An empty body must not be dropped or mangled.
func TestScopeCSS_EmptyBody(t *testing.T) {
	css := "p {}"
	got := ScopeCSS(css, "[data-v-abc]")
	want := "p[data-v-abc] {}"
	if got != want {
		t.Errorf("ScopeCSS empty body:\ngot  %q\nwant %q", got, want)
	}
}

// TestScopeCSS_AtRulePreserved verifies that @keyframes rules are passed
// through verbatim and the scope attribute is NOT injected into keyframe steps.
// @keyframes contain pseudo-rule names (from/to/percentages) that are not
// element selectors and must not be rewritten.
func TestScopeCSS_AtRulePreserved(t *testing.T) {
	css := "@keyframes slide { from { left: 0; } to { left: 100%; } }"
	got := ScopeCSS(css, "[data-v-abc]")
	if got != css {
		t.Errorf("ScopeCSS @keyframes:\ngot  %q\nwant verbatim %q", got, css)
	}
	if strings.Contains(got, "[data-v-abc]") {
		t.Errorf("ScopeCSS @keyframes: scope attr must not appear inside @keyframes")
	}
}

// TestScopeCSS_NestedAtRule verifies that a style selector nested inside a
// conditional group @-rule (@media/@supports/@container) IS scoped, so styles
// declared inside a media query still match the scoped element.
func TestScopeCSS_NestedAtRule(t *testing.T) {
	css := "@media (max-width: 600px) { .card { color: red } }"
	got := ScopeCSS(css, "[data-v-abc]")
	want := "@media (max-width: 600px) { .card[data-v-abc] { color: red } }"
	if got != want {
		t.Errorf("ScopeCSS nested @-rule:\ngot  %q\nwant %q", got, want)
	}
}

// TestScopeCSS_GroupRules verifies that @media, @supports and @container bodies
// are scoped, while @keyframes and @font-face are left verbatim. Nested group
// rules are scoped recursively.
func TestScopeCSS_GroupRules(t *testing.T) {
	cases := []struct {
		name string
		css  string
		want string
	}{
		{
			"media",
			"@media (min-width: 700px) { .a { x: 1 } .b { y: 2 } }",
			"@media (min-width: 700px) { .a[data-v-abc] { x: 1 } .b[data-v-abc] { y: 2 } }",
		},
		{
			"supports",
			"@supports (display: grid) { .a { x: 1 } }",
			"@supports (display: grid) { .a[data-v-abc] { x: 1 } }",
		},
		{
			"container",
			"@container (min-width: 200px) { .a { x: 1 } }",
			"@container (min-width: 200px) { .a[data-v-abc] { x: 1 } }",
		},
		{
			"nested media in supports",
			"@supports (display: grid) { @media (min-width: 700px) { .a { x: 1 } } }",
			"@supports (display: grid) { @media (min-width: 700px) { .a[data-v-abc] { x: 1 } } }",
		},
		{
			"keyframes verbatim",
			"@keyframes slide { from { left: 0 } to { left: 100% } }",
			"@keyframes slide { from { left: 0 } to { left: 100% } }",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScopeCSS(tc.css, "[data-v-abc]")
			if got != tc.want {
				t.Errorf("ScopeCSS(%q):\ngot  %q\nwant %q", tc.css, got, tc.want)
			}
		})
	}
}

// TestScopeCSS_GroupRulesEdgeCases exhaustively exercises the conditional
// group at-rule scoping path so the @media fix stays rock solid across the
// shapes real stylesheets take.
func TestScopeCSS_GroupRulesEdgeCases(t *testing.T) {
	cases := []struct {
		name string
		css  string
		want string
	}{
		{
			// Comma-separated selectors inside a media query: each branch scoped.
			"comma selectors inside media",
			"@media screen { .a, .b { x: 1 } }",
			"@media screen { .a[data-v-abc], .b[data-v-abc] { x: 1 } }",
		},
		{
			// Combinator selectors: scope appends to the last compound selector.
			"combinator inside media",
			"@media screen { .a > .b span { x: 1 } }",
			"@media screen { .a > .b span[data-v-abc] { x: 1 } }",
		},
		{
			// Pseudo-classes inside a media query.
			"pseudo inside media",
			"@media screen { a:hover { x: 1 } }",
			"@media screen { a:hover[data-v-abc] { x: 1 } }",
		},
		{
			// At-rule name matching is case-insensitive (CSS keywords are ASCII
			// case-insensitive).
			"uppercase at-rule name",
			"@MEDIA screen { .a { x: 1 } }",
			"@MEDIA screen { .a[data-v-abc] { x: 1 } }",
		},
		{
			"mixed-case at-rule name",
			"@Media screen { .a { x: 1 } }",
			"@Media screen { .a[data-v-abc] { x: 1 } }",
		},
		{
			// No space between the keyword and the condition's opening paren.
			"no space before paren",
			"@media(max-width: 600px) { .a { x: 1 } }",
			"@media(max-width: 600px) { .a[data-v-abc] { x: 1 } }",
		},
		{
			// Named container query.
			"named container",
			"@container sidebar (min-width: 200px) { .a { x: 1 } }",
			"@container sidebar (min-width: 200px) { .a[data-v-abc] { x: 1 } }",
		},
		{
			// Empty group-rule body: nothing to scope, structure preserved.
			"empty media body",
			"@media screen { }",
			"@media screen { }",
		},
		{
			// A bare style rule preceding a media query: both are handled.
			"rule then media",
			".a { x: 1 } @media screen { .b { y: 2 } }",
			".a[data-v-abc] { x: 1 } @media screen { .b[data-v-abc] { y: 2 } }",
		},
		{
			// A media query preceding a bare style rule.
			"media then rule",
			"@media screen { .a { x: 1 } } .b { y: 2 }",
			"@media screen { .a[data-v-abc] { x: 1 } } .b[data-v-abc] { y: 2 }",
		},
		{
			// Two independent media blocks.
			"two media blocks",
			"@media a { .x { p: 1 } } @media b { .y { q: 2 } }",
			"@media a { .x[data-v-abc] { p: 1 } } @media b { .y[data-v-abc] { q: 2 } }",
		},
		{
			// Triple nesting: @container > @supports > @media > rule.
			"triple nested group rules",
			"@container (min-width: 1px) { @supports (x: y) { @media screen { .a { z: 1 } } } }",
			"@container (min-width: 1px) { @supports (x: y) { @media screen { .a[data-v-abc] { z: 1 } } } }",
		},
		{
			// A non-group at-rule nested inside a group rule stays verbatim while
			// its sibling style rule is scoped. (@font-face inside @media is valid
			// CSS.)
			"font-face sibling inside media",
			"@media screen { @font-face { font-family: F } .a { x: 1 } }",
			"@media screen { @font-face { font-family: F } .a[data-v-abc] { x: 1 } }",
		},
		{
			// @keyframes nested inside @media: keyframe steps must NOT be scoped.
			"keyframes inside media",
			"@media screen { @keyframes k { from { o: 0 } to { o: 1 } } .a { x: 1 } }",
			"@media screen { @keyframes k { from { o: 0 } to { o: 1 } } .a[data-v-abc] { x: 1 } }",
		},
		{
			// Multiline / arbitrary whitespace inside the body is preserved.
			"multiline body whitespace preserved",
			"@media screen {\n  .a {\n    color: red;\n  }\n}",
			"@media screen {\n  .a[data-v-abc] {\n    color: red;\n  }\n}",
		},
		{
			// @page is not a scopable group rule; passed through verbatim.
			"page verbatim",
			"@page { margin: 1cm }",
			"@page { margin: 1cm }",
		},
		{
			// A property named like a keyword (e.g. a declaration mentioning the
			// word media) inside a normal rule must not trip the at-rule branch.
			"normal rule unaffected",
			".media-box { color: red }",
			".media-box[data-v-abc] { color: red }",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScopeCSS(tc.css, "[data-v-abc]")
			if got != tc.want {
				t.Errorf("ScopeCSS(%q):\ngot  %q\nwant %q", tc.css, got, tc.want)
			}
		})
	}
}

// TestScopeCSS_GroupRuleEmptyScopeAttr verifies that an empty scopeAttr is a
// no-op even inside group rules: recursion must not mangle the body.
func TestScopeCSS_GroupRuleEmptyScopeAttr(t *testing.T) {
	css := "@media screen { .a, .b { x: 1 } }"
	got := ScopeCSS(css, "")
	if got != css {
		t.Errorf("ScopeCSS empty attr inside @media:\ngot  %q\nwant %q", got, css)
	}
}

// TestScopeCSS_KeyframesNeverScoped guards the most dangerous regression: a
// scope attribute appearing inside @keyframes (where there are no element
// selectors) would corrupt the animation.
func TestScopeCSS_KeyframesNeverScoped(t *testing.T) {
	for _, css := range []string{
		"@keyframes slide { from { left: 0 } to { left: 100% } }",
		"@keyframes pulse { 0% { o: 0 } 50% { o: .5 } 100% { o: 1 } }",
		"@media screen { @keyframes k { from { o: 0 } to { o: 1 } } }",
	} {
		got := ScopeCSS(css, "[data-v-abc]")
		if strings.Contains(got, "data-v-abc] {") && strings.Contains(got, "@keyframes") {
			// The scope attr is only legal on the .a-style rules, never on a
			// keyframe step. Detect a step being scoped.
			for _, step := range []string{"from", "to", "0%", "50%", "100%"} {
				if strings.Contains(got, step+"[data-v-abc]") {
					t.Errorf("ScopeCSS(%q): keyframe step %q must not be scoped:\n%q", css, step, got)
				}
			}
		}
	}
}

// TestScopeCSS_isScopableGroupRule unit-tests the prelude classifier directly,
// including the prefix-confusion cases it must reject.
func TestScopeCSS_isScopableGroupRule(t *testing.T) {
	cases := []struct {
		prelude string
		want    bool
	}{
		{"@media screen", true},
		{"@media (max-width: 600px)", true},
		{"@media(max-width: 600px)", true},
		{"  @media screen  ", true},
		{"@MEDIA screen", true},
		{"@supports (x: y)", true},
		{"@container sidebar (min-width: 1px)", true},
		{"@keyframes slide", false},
		{"@font-face", false},
		{"@page", false},
		{"@import url(x)", false},
		// Prefix-confusion: longer names that merely start with a group keyword
		// must not be treated as group rules.
		{"@medianana foo", false},
		{"@mediaquery foo", false},
		{"@supportsx foo", false},
		{"@containerize foo", false},
		// Bare keyword with no condition (degenerate but must classify by name).
		{"@media", true},
		{"@supports", true},
		{"@container", true},
	}
	for _, tc := range cases {
		if got := isScopableGroupRule(tc.prelude); got != tc.want {
			t.Errorf("isScopableGroupRule(%q) = %v, want %v", tc.prelude, got, tc.want)
		}
	}
}

// TestScopeCSS_GroupRuleIdempotent verifies that scoping is not applied twice
// when ScopeCSS is run over already-scoped CSS containing a media query — i.e.
// the recursion does not double-append the attribute on the first pass, and a
// re-run is a structural no-op for the selector count.
func TestScopeCSS_GroupRuleIdempotentSinglePass(t *testing.T) {
	css := "@media screen { .a { x: 1 } }"
	got := ScopeCSS(css, "[data-v-abc]")
	if n := strings.Count(got, "[data-v-abc]"); n != 1 {
		t.Errorf("ScopeCSS(%q) = %q: want exactly 1 scope attr, got %d", css, got, n)
	}
}

// TestRenderer_MediaQueryScopedCSSMatchesElement is the end-to-end regression
// for the reported bug: a component whose <style scoped> declares rules inside
// an @media block must emit CSS whose nested selector carries the SAME scope
// attribute that is stamped on the rendered element — otherwise the media-query
// styles silently never match.
func TestRenderer_MediaQueryScopedCSSMatchesElement(t *testing.T) {
	src := `<template><div class="card"><p>hi</p></div></template>` +
		`<style scoped>` +
		`.card { color: black } ` +
		`@media (max-width: 600px) { .card { color: red } .card p { font-size: 12px } }` +
		`</style>`
	c, err := ParseFile("Card.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	sc := &StyleCollector{}
	out, err := NewRenderer(c).WithStyles(sc).RenderString(nil)
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	scopeAttr := "[" + ScopeID("Card.vue") + "]"

	contribs := sc.All()
	if len(contribs) != 1 {
		t.Fatalf("got %d style contributions, want 1", len(contribs))
	}
	cssOut := contribs[0].CSS

	// The element carries the scope attribute (without the brackets).
	if !strings.Contains(out, ScopeID("Card.vue")) {
		t.Fatalf("rendered HTML %q missing scope attr %q", out, ScopeID("Card.vue"))
	}
	// The selectors inside the media query must be scoped so they match.
	if !strings.Contains(cssOut, ".card"+scopeAttr) {
		t.Errorf("media-query selector .card not scoped in CSS:\n%s", cssOut)
	}
	if !strings.Contains(cssOut, "p"+scopeAttr) {
		t.Errorf("media-query descendant selector .card p not scoped in CSS:\n%s", cssOut)
	}
	// The @media wrapper itself must survive intact.
	if !strings.Contains(cssOut, "@media (max-width: 600px)") {
		t.Errorf("@media prelude lost in CSS:\n%s", cssOut)
	}
	// Sanity: every scoped occurrence corresponds to a real selector — the base
	// rule plus the two inside the media query.
	if n := strings.Count(cssOut, scopeAttr); n != 3 {
		t.Errorf("expected 3 scoped selectors (.card base + .card + .card p in @media), got %d:\n%s", n, cssOut)
	}
}

// TestScopeCSS_EmptyInput verifies that ScopeCSS returns "" for empty input.
// This is a useful boundary case: the loop must not produce spurious output.
func TestScopeCSS_EmptyInput(t *testing.T) {
	got := ScopeCSS("", "[data-v-abc]")
	if got != "" {
		t.Errorf("ScopeCSS(\"\", ...): got %q, want \"\"", got)
	}
}

// TestStyleCollector_Deduplication exercises the deduplication logic of
// StyleCollector directly (without going through the renderer) to pin the
// exact key used for deduplication (ScopeID + CSS).
func TestStyleCollector_Deduplication(t *testing.T) {
	// Adding the same contribution twice must produce exactly one entry.
	// This prevents double-emitting the same stylesheet when a component is
	// rendered more than once in the same request.
	t.Run("same contribution twice is deduplicated", func(t *testing.T) {
		sc := &StyleCollector{}
		c := StyleContribution{ScopeID: "data-v-aabbccdd", CSS: ".a { color: red; }"}
		sc.Add(c)
		sc.Add(c)
		if got := len(sc.All()); got != 1 {
			t.Errorf("got %d contributions, want 1", got)
		}
	})

	// Same ScopeID but different CSS — the CSS content differs, so both must
	// be kept.  This can happen when the same component path is used with
	// different style content (e.g. hot-reload scenarios).
	t.Run("same ScopeID different CSS are both kept", func(t *testing.T) {
		sc := &StyleCollector{}
		sc.Add(StyleContribution{ScopeID: "data-v-aaaaaaaa", CSS: ".a { color: red; }"})
		sc.Add(StyleContribution{ScopeID: "data-v-aaaaaaaa", CSS: ".b { color: blue; }"})
		if got := len(sc.All()); got != 2 {
			t.Errorf("got %d contributions, want 2", got)
		}
	})

	// Same CSS but different ScopeID — two distinct scoped components with
	// identical stylesheet text must both be kept.
	t.Run("same CSS different ScopeID are both kept", func(t *testing.T) {
		sc := &StyleCollector{}
		sc.Add(StyleContribution{ScopeID: "data-v-aaaaaaaa", CSS: ".a { color: red; }"})
		sc.Add(StyleContribution{ScopeID: "data-v-bbbbbbbb", CSS: ".a { color: red; }"})
		if got := len(sc.All()); got != 2 {
			t.Errorf("got %d contributions, want 2", got)
		}
	})

	// All() on a zero-value (never-written) collector must return nil without
	// panicking.  Callers must be able to range over the result safely.
	t.Run("All() on zero-value collector does not panic", func(t *testing.T) {
		var sc StyleCollector
		got := sc.All()
		if got != nil {
			t.Errorf("zero-value collector All(): got %v, want nil", got)
		}
	})

	// Contributions must be returned in insertion order (FIFO) so that the
	// rendered <style> tags appear in a deterministic order.
	t.Run("FIFO order is preserved", func(t *testing.T) {
		sc := &StyleCollector{}
		c1 := StyleContribution{ScopeID: "data-v-1", CSS: "first"}
		c2 := StyleContribution{ScopeID: "data-v-2", CSS: "second"}
		c3 := StyleContribution{ScopeID: "data-v-3", CSS: "third"}
		sc.Add(c1)
		sc.Add(c2)
		sc.Add(c3)
		all := sc.All()
		if len(all) != 3 {
			t.Fatalf("got %d contributions, want 3", len(all))
		}
		if all[0] != c1 || all[1] != c2 || all[2] != c3 {
			t.Errorf("FIFO order not preserved: got %v", all)
		}
	})
}
