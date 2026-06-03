package htmlc

import "testing"

// TestScopeCSS_Spec is the golden specification for scoped-CSS rewriting: each
// row pins the exact output for a given input. It encodes Vue-compatible
// behavior across every token interaction the scoper must handle. Where the
// historical naive implementation diverged, these rows define the correct
// result.
func TestScopeCSS_Spec(t *testing.T) {
	const s = "[data-v-scope]"
	cases := []struct {
		name string
		in   string
		want string
	}{
		// --- ordinary selectors ---
		{"simple", ".card { color: red; }", ".card" + s + " { color: red; }"},
		{"descendant", "h2 .title { x:1 }", "h2 .title" + s + " { x:1 }"},
		{"child combinator", "div > span { x:1 }", "div > span" + s + " { x:1 }"},
		{"adjacent sibling", "a + b { x:1 }", "a + b" + s + " { x:1 }"},
		{"general sibling", "a ~ b { x:1 }", "a ~ b" + s + " { x:1 }"},
		{"comma list", ".a, .b { x:1 }", ".a" + s + ", .b" + s + " { x:1 }"},
		{"two rules", ".a { x:1 } .b { y:2 }", ".a" + s + " { x:1 } .b" + s + " { y:2 }"},

		// --- functional pseudo-classes: inner commas must NOT be split ---
		{"is comma", ":is(.a, .b) { x:1 }", ":is(.a, .b)" + s + " { x:1 }"},
		{"not comma", ".a:not(.b, .c) { x:1 }", ".a:not(.b, .c)" + s + " { x:1 }"},
		{"where + descendant", ":where(.a, .b) span { x:1 }", ":where(.a, .b) span" + s + " { x:1 }"},
		{"has", ".a:has(> img, > svg) { x:1 }", ".a:has(> img, > svg)" + s + " { x:1 }"},
		{"nth-child", "li:nth-child(2n+1) { x:1 }", "li:nth-child(2n+1)" + s + " { x:1 }"},

		// --- attribute selectors with commas/braces/semicolons in the value ---
		{"attr comma string", `a[title="hello, world"] { x:1 }`, `a[title="hello, world"]` + s + ` { x:1 }`},
		{"attr brace string", `a[data-x='{ } ;'] { x:1 }`, `a[data-x='{ } ;']` + s + ` { x:1 }`},
		{"double attr", `input[type="text"][required] { x:1 }`, `input[type="text"][required]` + s + ` { x:1 }`},

		// --- pseudo-elements: scope goes BEFORE the pseudo-element ---
		{"pseudo-element ::before", "p::before { content: '' }", "p" + s + "::before { content: '' }"},
		{"pseudo-element ::after", "p::after { content: '' }", "p" + s + "::after { content: '' }"},
		{"legacy :before", "p:before { content: '' }", "p" + s + ":before { content: '' }"},
		{"legacy :first-line", "p:first-line { x:1 }", "p" + s + ":first-line { x:1 }"},
		// pseudo-CLASS stays at the very end
		{"pseudo-class hover", "a:hover { x:1 }", "a:hover" + s + " { x:1 }"},
		{"pseudo-class focus", "input:focus { x:1 }", "input:focus" + s + " { x:1 }"},
		// pseudo-class then pseudo-element: scope before the element, after the class
		{"hover then ::after", "a:hover::after { x:1 }", "a:hover" + s + "::after { x:1 }"},

		// --- strings & comments containing braces ---
		{"brace in value string", `.x { content: "}" }`, `.x` + s + ` { content: "}" }`},
		{"open brace in value", `.x { content: "{" } .y { color: red }`, `.x` + s + ` { content: "{" } .y` + s + ` { color: red }`},
		{"comment with brace in body", ".a { /* } */ color: red }", ".a" + s + " { /* } */ color: red }"},
		{"leading comment before rule", "/* a } comment */ .a { color: red }", "/* a } comment */ .a" + s + " { color: red }"},

		// --- at-rules: statement (verbatim), then a scoped rule ---
		{"import then rule", "@import 'x.css'; .a { x:1 }", "@import 'x.css'; .a" + s + " { x:1 }"},
		{"charset then rule", `@charset "utf-8"; .a { x:1 }`, `@charset "utf-8"; .a` + s + ` { x:1 }`},
		{"namespace then rule", "@namespace svg url(http://www.w3.org/2000/svg); .a { x:1 }", "@namespace svg url(http://www.w3.org/2000/svg); .a" + s + " { x:1 }"},

		// --- conditional group at-rules: recurse into the body ---
		{"media", "@media (max-width: 48rem) { .a { x:1 } }", "@media (max-width: 48rem) { .a" + s + " { x:1 } }"},
		{"comment before media (marginalia)", "/* layout */ @media (max-width: 48rem) { .two-pane { x:1 } }", "/* layout */ @media (max-width: 48rem) { .two-pane" + s + " { x:1 } }"},
		{"import before media", "@import 'x.css'; @media screen { .a { x:1 } }", "@import 'x.css'; @media screen { .a" + s + " { x:1 } }"},
		{"supports", "@supports (display:grid) { .a { x:1 } }", "@supports (display:grid) { .a" + s + " { x:1 } }"},
		{"container named", "@container sidebar (min-width: 200px) { .a { x:1 } }", "@container sidebar (min-width: 200px) { .a" + s + " { x:1 } }"},
		{"nested group rules", "@supports (display:grid) { @media screen { .a, .b { x:1 } } }", "@supports (display:grid) { @media screen { .a" + s + ", .b" + s + " { x:1 } } }"},
		{"layer block scoped", "@layer base { .a { x:1 } }", "@layer base { .a" + s + " { x:1 } }"},

		// --- at-rules that must be passed through verbatim ---
		{"layer statement", "@layer a, b;", "@layer a, b;"},
		{"keyframes", "@keyframes k { from { o:0 } to { o:1 } }", "@keyframes k { from { o:0 } to { o:1 } }"},
		{"font-face", "@font-face { font-family: F; src: url(f.woff2) }", "@font-face { font-family: F; src: url(f.woff2) }"},
		{"page", "@page { margin: 1cm }", "@page { margin: 1cm }"},

		// --- boundary cases ---
		{"empty", "", ""},
		{"empty body", ".empty { }", ".empty" + s + " { }"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ScopeCSS(tc.in, s)
			if got != tc.want {
				t.Errorf("ScopeCSS(%q):\n got  %q\n want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestScopeCSS_Spec_EmptyAttrIsNoop verifies that an empty scope attribute is a
// structural no-op across all corpus inputs, including the tricky ones.
func TestScopeCSS_Spec_EmptyAttrIsNoop(t *testing.T) {
	for _, css := range cssCorpus {
		if got := ScopeCSS(css, ""); got != css {
			t.Errorf("empty-attr ScopeCSS(%q) = %q, want unchanged", css, got)
		}
	}
}
