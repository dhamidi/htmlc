package htmlc

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// ScopeID computes the scope attribute name for a component at the given file
// path. The result is "data-v-" followed by the 8 lower-case hex digits of the
// FNV-1a 32-bit hash of path.
func ScopeID(path string) string {
	h := fnv.New32a()
	h.Write([]byte(path))
	return fmt.Sprintf("data-v-%08x", h.Sum32())
}

// ScopeCSS rewrites the CSS text so that every selector in a qualified style
// rule has scopeAttr inserted into its last compound selector. scopeAttr should
// be a full attribute selector string, e.g. "[data-v-a1b2c3d4]".
//
// The scanner is token-aware: CSS comments, strings, and bracket nesting are
// respected, so commas inside :is()/:not()/:where()/:has(), braces inside
// strings or comments, and leading comments before an at-rule never confuse it.
//
// At-rules are classified:
//   - Conditional group rules (@media, @supports, @container, and the block form
//     of @layer) wrap ordinary style rules, so ScopeCSS recurses into their
//     bodies and scopes the nested selectors.
//   - Statement at-rules (@import, @charset, @namespace, @layer a, b;) and other
//     block at-rules (@keyframes, @font-face, @page, …) are emitted verbatim,
//     because their contents are not element selectors.
//
// The transform only ever inserts scopeAttr; all other bytes are preserved
// exactly. An empty scopeAttr is a no-op.
func ScopeCSS(css, scopeAttr string) string {
	if scopeAttr == "" {
		return css
	}
	var out strings.Builder
	scopeRuleList(&out, css, scopeAttr)
	return out.String()
}

// scopeRuleList walks a list of CSS rules at one nesting level, writing the
// scoped result to out. It is called recursively for the bodies of conditional
// group at-rules.
func scopeRuleList(out *strings.Builder, css, scopeAttr string) {
	i, n := 0, len(css)
	for i < n {
		start := i
		term, termPos := scanPrelude(css, i)
		switch term {
		case 0:
			// No terminator: trailing content (whitespace, comments).
			out.WriteString(css[start:])
			return
		case ';':
			// Statement at-rule or stray content terminated by ';': verbatim.
			out.WriteString(css[start : termPos+1])
			i = termPos + 1
		case '{':
			prelude := css[start:termPos]
			bodyStart := termPos + 1
			bodyEnd := scanBlockEnd(css, bodyStart) // index of matching '}', or n
			if isAtRulePrelude(prelude) {
				if isScopableGroupRule(prelude) && bodyEnd < n {
					out.WriteString(prelude)
					out.WriteByte('{')
					scopeRuleList(out, css[bodyStart:bodyEnd], scopeAttr)
					out.WriteByte('}')
					i = bodyEnd + 1
				} else {
					// Verbatim block (non-group at-rule, or unterminated).
					end := bodyEnd
					if end < n {
						end++ // include the closing '}'
					}
					out.WriteString(css[start:end])
					i = end
				}
			} else {
				// Qualified rule: rewrite the selector list, body verbatim.
				out.WriteString(rewriteSelectorList(prelude, scopeAttr))
				out.WriteByte('{')
				if bodyEnd < n {
					out.WriteString(css[bodyStart:bodyEnd])
					out.WriteByte('}')
					i = bodyEnd + 1
				} else {
					out.WriteString(css[bodyStart:]) // unterminated
					i = n
				}
			}
		}
	}
}

// skipCSSAtom returns the index just past a CSS comment or string starting at i.
// If no comment or string starts at i, it returns i unchanged. Unterminated
// comments/strings consume to end of input.
func skipCSSAtom(s string, i int) int {
	if i+1 < len(s) && s[i] == '/' && s[i+1] == '*' {
		j := i + 2
		for j+1 < len(s) && !(s[j] == '*' && s[j+1] == '/') {
			j++
		}
		if j+1 < len(s) {
			return j + 2
		}
		return len(s)
	}
	if s[i] == '"' || s[i] == '\'' {
		q := s[i]
		j := i + 1
		for j < len(s) {
			if s[j] == '\\' && j+1 < len(s) {
				j += 2
				continue
			}
			if s[j] == q {
				return j + 1
			}
			j++
		}
		return len(s)
	}
	return i
}

// scanPrelude scans from i to the end of a rule prelude, returning the
// terminating byte ('{' for a block, ';' for a statement, or 0 at end of input)
// and its index. Comments, strings, and (...)/[...] nesting are skipped so that
// braces and semicolons inside them are not mistaken for terminators.
func scanPrelude(s string, i int) (term byte, pos int) {
	depth := 0
	for i < len(s) {
		if j := skipCSSAtom(s, i); j != i {
			i = j
			continue
		}
		switch s[i] {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case '{':
			if depth == 0 {
				return '{', i
			}
		case ';':
			if depth == 0 {
				return ';', i
			}
		}
		i++
	}
	return 0, len(s)
}

// scanBlockEnd returns the index of the '}' that matches the block whose opening
// '{' was already consumed (so brace depth starts at 1). Comments and strings
// are skipped. Returns len(s) if the block is unterminated.
func scanBlockEnd(s string, i int) int {
	depth := 1
	for i < len(s) {
		if j := skipCSSAtom(s, i); j != i {
			i = j
			continue
		}
		switch s[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
		i++
	}
	return len(s)
}

// trimLeadingWSComments returns s with leading whitespace and CSS comments
// removed. Used to classify a prelude regardless of decorative comments.
func trimLeadingWSComments(s string) string {
	i := 0
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f' {
			i++
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			i = skipCSSAtom(s, i)
			continue
		}
		break
	}
	return s[i:]
}

// isAtRulePrelude reports whether prelude (ignoring leading whitespace and
// comments) begins an at-rule.
func isAtRulePrelude(prelude string) bool {
	s := trimLeadingWSComments(prelude)
	return len(s) > 0 && s[0] == '@'
}

// isScopableGroupRule reports whether prelude introduces a conditional group
// at-rule whose body contains ordinary style rules that should be scoped:
// @media, @supports, @container, or the block form of @layer. Other at-rules
// (@keyframes, @font-face, @page, @import, …) hold non-selector content.
func isScopableGroupRule(prelude string) bool {
	s := trimLeadingWSComments(prelude)
	if len(s) == 0 || s[0] != '@' {
		return false
	}
	k := 1
	for k < len(s) && isIdentByte(s[k]) {
		k++
	}
	switch strings.ToLower(s[1:k]) {
	case "media", "supports", "container", "layer":
		return true
	}
	return false
}

// rewriteSelectorList inserts scopeAttr into each comma-separated selector in
// sel, splitting only on top-level commas (commas inside (...) / [...] / strings
// are left untouched).
func rewriteSelectorList(sel, scopeAttr string) string {
	var b strings.Builder
	depth := 0
	segStart := 0
	i := 0
	for i < len(sel) {
		if j := skipCSSAtom(sel, i); j != i {
			i = j
			continue
		}
		switch sel[i] {
		case '(', '[':
			depth++
		case ')', ']':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				b.WriteString(scopeOneSelector(sel[segStart:i], scopeAttr))
				b.WriteByte(',')
				segStart = i + 1
			}
		}
		i++
	}
	b.WriteString(scopeOneSelector(sel[segStart:], scopeAttr))
	return b.String()
}

// scopeOneSelector inserts scopeAttr into a single selector, after its last
// compound selector's simple selectors but before any trailing pseudo-element.
// Leading and trailing whitespace (and comments) are preserved.
func scopeOneSelector(part, scopeAttr string) string {
	end := len(part)
	for end > 0 && isCSSSpace(part[end-1]) {
		end--
	}
	if end == 0 {
		return part // empty or whitespace-only segment
	}
	work := part[:end]
	cs := lastCompoundStart(work)
	insertAt := cs + pseudoElementOffset(work[cs:])
	return part[:insertAt] + scopeAttr + part[insertAt:]
}

// lastCompoundStart returns the index in s where the last compound selector
// begins — i.e. the position just after the last top-level combinator
// (whitespace, '>', '+', '~'). Combinator-like characters inside (...) / [...]
// (e.g. the '+' in :nth-child(2n+1)) are ignored.
func lastCompoundStart(s string) int {
	depth := 0
	compoundStart := 0
	sawSep := false
	i := 0
	for i < len(s) {
		if j := skipCSSAtom(s, i); j != i {
			i = j
			continue
		}
		c := s[i]
		if depth == 0 {
			if isCSSSpace(c) || c == '>' || c == '+' || c == '~' {
				sawSep = true
			} else {
				if sawSep {
					compoundStart = i
					sawSep = false
				}
				if c == '(' || c == '[' {
					depth++
				}
			}
		} else {
			if c == '(' || c == '[' {
				depth++
			} else if c == ')' || c == ']' {
				depth--
			}
		}
		i++
	}
	return compoundStart
}

// pseudoElementOffset returns the index in compound where a pseudo-element
// begins (so the scope attribute is inserted before it), or len(compound) if
// there is none. Pseudo-classes are skipped; the scope attribute is placed
// after them. Pseudo-elements are '::name' or the legacy single-colon forms
// :before, :after, :first-line, :first-letter.
func pseudoElementOffset(compound string) int {
	depth := 0
	i := 0
	for i < len(compound) {
		if j := skipCSSAtom(compound, i); j != i {
			i = j
			continue
		}
		c := compound[i]
		if depth > 0 {
			if c == '(' || c == '[' {
				depth++
			} else if c == ')' || c == ']' {
				depth--
			}
			i++
			continue
		}
		switch c {
		case '(', '[':
			depth++
			i++
		case ':':
			if i+1 < len(compound) && compound[i+1] == ':' {
				return i // ::pseudo-element
			}
			identStart := i + 1
			k := identStart
			for k < len(compound) && isIdentByte(compound[k]) {
				k++
			}
			hasArg := k < len(compound) && compound[k] == '('
			switch strings.ToLower(compound[identStart:k]) {
			case "before", "after", "first-line", "first-letter":
				if !hasArg {
					return i // legacy single-colon pseudo-element
				}
			}
			// Pseudo-class: skip the name and any (...) argument.
			i = k
			if hasArg {
				depth++
				i++
			}
		default:
			i++
		}
	}
	return len(compound)
}

// isCSSSpace reports whether b is CSS whitespace.
func isCSSSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r' || b == '\f'
}

// isIdentByte reports whether b can appear in a CSS identifier (ASCII subset).
func isIdentByte(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') ||
		(b >= '0' && b <= '9') || b == '-' || b == '_'
}

// StyleContribution holds a style block contributed by a component during render.
type StyleContribution struct {
	// ScopeID is the scope attribute name (e.g. "data-v-a1b2c3d4") for a
	// scoped component's styles, or empty for global (non-scoped) styles.
	ScopeID string
	// CSS is the stylesheet text. For scoped components it has already been
	// rewritten by ScopeCSS; for global styles it is passed through verbatim.
	CSS string
}

// StyleCollector accumulates StyleContributions from one or more component
// renders into a single ordered list, deduplicating repeated contributions
// from the same scoped component. It is part of the low-level API; Engine
// creates and manages a StyleCollector automatically on each render call.
type StyleCollector struct {
	items []StyleContribution
	seen  map[string]struct{}
}

// Add appends c to the collector, skipping duplicates. Two contributions are
// considered duplicates when they share the same composite key (ScopeID + CSS),
// so the same scoped component rendered N times contributes its CSS only once,
// while different components or differing global CSS blocks are each kept.
func (sc *StyleCollector) Add(c StyleContribution) {
	key := c.ScopeID + "\x00" + c.CSS
	if sc.seen == nil {
		sc.seen = make(map[string]struct{})
	}
	if _, ok := sc.seen[key]; ok {
		return
	}
	sc.seen[key] = struct{}{}
	sc.items = append(sc.items, c)
}

// All returns all StyleContributions in the order they were added.
// The slice is nil when no contributions have been collected.
func (sc *StyleCollector) All() []StyleContribution {
	return sc.items
}
