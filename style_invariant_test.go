package htmlc

import (
	"strings"
	"testing"
)

// scopeSentinel is a recognizable scope-attribute value used by the invariant
// verifiers. Its exact text does not matter to ScopeCSS (the attr is opaque),
// but a fixed, unusual value lets the verifiers locate every injected attr.
const scopeSentinel = "[data-v-scope]"

// stripSentinel removes every injected scope-attribute sentinel from s.
func stripSentinel(s string) string {
	return strings.ReplaceAll(s, scopeSentinel, "")
}

// sentinelPlacements scans css — the OUTPUT of ScopeCSS — and counts, for each
// occurrence of the sentinel, whether it sits in a position that is VALID for a
// scope attribute. A scope attribute may only appear in selector text: never
// inside a comment, a string, a (...) or [...] group, or an at-rule prelude
// (the text from an at-rule's @ up to its '{' or ';').
//
// This scanner is intentionally a separate, simpler implementation than the one
// under test in style.go, so it can serve as an independent cross-check. It is
// comment-, string-, and bracket-aware; it does NOT attempt to fully parse CSS.
func sentinelPlacements(css string) (valid, invalid int) {
	var (
		inComment   bool
		stringQuote byte   // the open quote when inside a string, else 0
		groupDepth  int    // nesting depth of () and []
		inPrelude   bool   // inside an at-rule prelude
		atBoundary  = true // at a construct boundary (start, or after { } ;)
	)
	i, n := 0, len(css)
	for i < n {
		// Detect an injected sentinel here, regardless of state, and skip it as
		// a unit so its own brackets don't perturb the scanner.
		if strings.HasPrefix(css[i:], scopeSentinel) {
			if inComment || stringQuote != 0 || groupDepth > 0 || inPrelude {
				invalid++
			} else {
				valid++
			}
			atBoundary = false
			i += len(scopeSentinel)
			continue
		}
		c := css[i]
		switch {
		case inComment:
			if c == '*' && i+1 < n && css[i+1] == '/' {
				inComment = false
				i += 2
				continue
			}
		case stringQuote != 0:
			if c == '\\' && i+1 < n {
				i += 2 // skip the escaped character
				continue
			}
			if c == stringQuote {
				stringQuote = 0
			}
		default:
			switch {
			case c == '/' && i+1 < n && css[i+1] == '*':
				inComment = true
				i += 2
				continue
			case c == '"' || c == '\'':
				stringQuote = c
				atBoundary = false
			case c == '(' || c == '[':
				groupDepth++
				atBoundary = false
			case c == ')' || c == ']':
				if groupDepth > 0 {
					groupDepth--
				}
				atBoundary = false
			case c == '@':
				// An at-rule prelude begins only with an '@' at a construct
				// boundary; an '@' elsewhere is just (malformed) selector text.
				if groupDepth == 0 && atBoundary {
					inPrelude = true
				}
				atBoundary = false
			case c == '{' || c == ';' || c == '}':
				// A brace/semicolon ends the current selector, declaration, or
				// statement. In valid CSS, () and [] are always balanced within
				// those, so reset group depth here — a no-op for valid input and
				// sane recovery from malformed input (an unbalanced bracket in
				// one block must not leak into a sibling).
				groupDepth = 0
				inPrelude = false
				atBoundary = true
			case isCSSSpace(c):
				// whitespace is transparent to the boundary
			default:
				atBoundary = false
			}
		}
		i++
	}
	return valid, invalid
}

// cssCorpus is a set of stylesheets exercising the token interactions that the
// scoper must handle. It covers every known bug family plus ordinary CSS.
var cssCorpus = []string{
	// Ordinary selectors.
	"p { color: red }",
	".card { color: red; }",
	".a, .b { margin: 0 }",
	"h2 .title { font-size: 2em }",
	".a > .b span { x:1 }",
	"a + b { x:1 }",
	"a ~ b { x:1 }",
	// Functional pseudo-classes containing commas (must not be split).
	":is(.a, .b) { x:1 }",
	".a:not(.b, .c) { x:1 }",
	":where(.a, .b) span { x:1 }",
	".a:has(> img, > svg) { x:1 }",
	"li:nth-child(2n+1) { x:1 }",
	// Attribute selectors with commas / braces / semicolons in the value.
	`a[title="hello, world"] { x:1 }`,
	`a[data-x='{ } ;'] { x:1 }`,
	`input[type="text"][required] { x:1 }`,
	// Pseudo-elements.
	"p::before { content: '' }",
	"p:before { content: '' }",
	"a:hover::after { content: '' }",
	// Strings and comments containing braces.
	`.x { content: "}" }`,
	`.x { content: "{" } .y { color: red }`,
	".a { /* } */ color: red }",
	"/* a } comment */ .a { color: red }",
	// At-rules: statement, group, and verbatim.
	"@import 'x.css'; .a { x:1 }",
	`@charset "utf-8"; .a { x:1 }`,
	"@namespace svg url(http://www.w3.org/2000/svg); .a { x:1 }",
	"@media (max-width: 48rem) { .two-pane { x:1 } }",
	"/* layout */ @media (max-width: 48rem) { .two-pane { x:1 } }",
	"@import 'x.css'; @media screen { .a { x:1 } }",
	"@supports (display:grid) { @media screen { .a, .b { x:1 } } }",
	"@container sidebar (min-width: 200px) { .a { x:1 } }",
	"@layer base { .a { x:1 } }",
	"@layer a, b;",
	"@keyframes k { from { o:0 } to { o:1 } }",
	"@font-face { font-family: F; src: url(f.woff2) }",
	"@page { margin: 1cm }",
	// Whitespace / formatting.
	"@media screen {\n  .a {\n    color: red;\n  }\n}",
	"",
	"   ",
	".empty { }",
}

// checkScopeInvariants asserts the two structural invariants that must hold for
// ANY input: (1) the transform only INSERTS scope attributes and changes
// nothing else, and (2) every injected attribute lands in a valid position.
func checkScopeInvariants(t *testing.T, input string) {
	t.Helper()
	if strings.Contains(input, scopeSentinel) {
		return // can't tell injected from pre-existing; skip
	}
	out := ScopeCSS(input, scopeSentinel)

	// Invariant 1 — insertion-only: removing the injected attrs must restore the
	// input byte-for-byte. This forbids any reordering, whitespace loss, or
	// mutation of the surrounding CSS.
	if restored := stripSentinel(out); restored != input {
		t.Errorf("insertion-only violated:\n input    = %q\n restored = %q\n output   = %q",
			input, restored, out)
	}

	// Invariant 2 — placement-valid: no scope attr inside a comment, string,
	// (...)/[...] group, or at-rule prelude. This is only well-defined for
	// well-formed CSS; on malformed input (unbalanced braces, stray closers,
	// unterminated strings) "correct" placement is undefined, and insertion-only
	// above already guarantees the surrounding bytes are never corrupted.
	if isBalancedCSS(input) {
		if _, invalid := sentinelPlacements(out); invalid > 0 {
			t.Errorf("placement violated: %d scope attr(s) in a forbidden position:\n output = %q",
				invalid, out)
		}
	}
}

// isBalancedCSS reports whether s has balanced, correctly nested () [] {} with
// terminated strings and comments, and no closer without a matching opener. It
// is the precondition under which scope-attribute placement is well-defined.
func isBalancedCSS(s string) bool {
	var stack []byte
	matches := func(open, close byte) bool {
		return (open == '{' && close == '}') ||
			(open == '(' && close == ')') ||
			(open == '[' && close == ']')
	}
	i, n := 0, len(s)
	for i < n {
		c := s[i]
		if c == '/' && i+1 < n && s[i+1] == '*' {
			i += 2
			closed := false
			for i+1 < n {
				if s[i] == '*' && s[i+1] == '/' {
					i += 2
					closed = true
					break
				}
				i++
			}
			if !closed {
				return false
			}
			continue
		}
		if c == '"' || c == '\'' {
			i++
			closed := false
			for i < n {
				if s[i] == '\\' && i+1 < n {
					i += 2
					continue
				}
				if s[i] == c {
					i++
					closed = true
					break
				}
				i++
			}
			if !closed {
				return false
			}
			continue
		}
		switch c {
		case '{', '(', '[':
			stack = append(stack, c)
		case '}', ')', ']':
			if len(stack) == 0 || !matches(stack[len(stack)-1], c) {
				return false
			}
			stack = stack[:len(stack)-1]
		}
		i++
	}
	return len(stack) == 0
}

func TestScopeCSS_Invariants_Corpus(t *testing.T) {
	for _, css := range cssCorpus {
		css := css
		t.Run(strings.TrimSpace(css), func(t *testing.T) {
			checkScopeInvariants(t, css)
		})
	}
}

// FuzzScopeCSS_Invariants verifies the structural invariants on arbitrary input,
// not merely that ScopeCSS does not panic.
func FuzzScopeCSS_Invariants(f *testing.F) {
	for _, css := range cssCorpus {
		f.Add(css)
	}
	f.Fuzz(func(t *testing.T, css string) {
		checkScopeInvariants(t, css)
	})
}

// TestIsBalancedCSS_SelfCheck guards the placement gate: it must accept
// well-formed CSS and reject the malformed shapes for which placement is
// undefined.
func TestIsBalancedCSS_SelfCheck(t *testing.T) {
	balanced := []string{
		"", "   ", ".a { x:1 }", ":is(.a, .b) { x:1 }",
		`a[title="hello, world"] { x:1 }`, "@media s { .a { x:1 } }",
		".x { content: \"}\" }", "/* } */ .a { x:1 }", `a[data-x='{ } ;'] {}`,
	}
	for _, s := range balanced {
		if !isBalancedCSS(s) {
			t.Errorf("isBalancedCSS(%q) = false, want true", s)
		}
	}
	malformed := []string{
		"{", "}", "0@{", ")@{", "{(}0{", "}@{", ".a (", "[", `"unterminated`,
		"/* unterminated", ".a { (} )", "([)]",
	}
	for _, s := range malformed {
		if isBalancedCSS(s) {
			t.Errorf("isBalancedCSS(%q) = true, want false", s)
		}
	}
}

// TestSentinelPlacements_SelfCheck guards the verifier itself: the independent
// placement scanner must classify hand-written good/bad placements correctly,
// so a bug in the scanner cannot mask a bug in ScopeCSS.
func TestSentinelPlacements_SelfCheck(t *testing.T) {
	cases := []struct {
		name           string
		css            string
		valid, invalid int
	}{
		{"valid in selector", ".a[data-v-scope] { x:1 }", 1, 0},
		{"valid in media body", "@media s { .a[data-v-scope] { x:1 } }", 1, 0},
		{"invalid in prelude", "@media s[data-v-scope] { .a { x:1 } }", 0, 1},
		{"invalid in paren", ":is(.a[data-v-scope]) { x:1 }", 0, 1},
		{"invalid in attr string", `a[title="x[data-v-scope]"] { y:1 }`, 0, 1},
		{"invalid in comment", "/* [data-v-scope] */ .a { x:1 }", 0, 1},
		{"none", ".a { x:1 }", 0, 0},
		{"mixed", ".a[data-v-scope] { x:1 } :is(.b[data-v-scope]) {}", 1, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, inv := sentinelPlacements(tc.css)
			if v != tc.valid || inv != tc.invalid {
				t.Errorf("sentinelPlacements(%q) = (valid=%d, invalid=%d), want (valid=%d, invalid=%d)",
					tc.css, v, inv, tc.valid, tc.invalid)
			}
		})
	}
}
