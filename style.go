// Package htmlc provides style scoping utilities for htmlc components.
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

// ScopeCSS rewrites the CSS text so that every selector in every non-@-rule has
// scopeAttr appended to its last compound selector. scopeAttr should be a full
// attribute selector string, e.g. "[data-v-a1b2c3d4]".
//
// @-rules (such as @media or @keyframes) are passed through verbatim, including
// their nested blocks.
func ScopeCSS(css, scopeAttr string) string {
	var out strings.Builder
	i, n := 0, len(css)

	for i < n {
		// Collect everything up to the next '{', which starts a rule body.
		ruleStart := i
		for i < n && css[i] != '{' {
			i++
		}

		if i >= n {
			// Trailing content after the last rule (whitespace, etc.).
			out.WriteString(css[ruleStart:])
			break
		}

		selectorText := css[ruleStart:i]
		i++ // consume '{'

		if strings.HasPrefix(strings.TrimSpace(selectorText), "@") {
			// @-rule: emit verbatim, tracking nested braces to find the end.
			out.WriteString(selectorText)
			out.WriteByte('{')
			depth := 1
			bodyStart := i
			for i < n && depth > 0 {
				switch css[i] {
				case '{':
					depth++
				case '}':
					depth--
				}
				i++
			}
			// css[bodyStart:i] includes all nested content and the final '}'.
			out.WriteString(css[bodyStart:i])
		} else {
			// Regular rule: find closing '}' and rewrite the selector.
			bodyStart := i
			for i < n && css[i] != '}' {
				i++
			}
			body := css[bodyStart:i]
			if i < n {
				i++ // consume '}'
			}
			out.WriteString(rewriteSelectors(selectorText, scopeAttr))
			out.WriteByte('{')
			out.WriteString(body)
			out.WriteByte('}')
		}
	}

	return out.String()
}

// rewriteSelectors appends scopeAttr to the last compound selector of each
// comma-separated selector in selectorText, preserving surrounding whitespace.
func rewriteSelectors(selectorText, scopeAttr string) string {
	parts := strings.Split(selectorText, ",")
	for i, part := range parts {
		// Preserve trailing whitespace.
		rtrimmed := strings.TrimRight(part, " \t\n\r")
		trailing := part[len(rtrimmed):]
		// Preserve leading whitespace.
		ltrimmed := strings.TrimLeft(rtrimmed, " \t\n\r")
		leading := rtrimmed[:len(rtrimmed)-len(ltrimmed)]
		// Append scopeAttr right after the actual selector text.
		parts[i] = leading + ltrimmed + scopeAttr + trailing
	}
	return strings.Join(parts, ",")
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
// renders into a single ordered list.
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

// All returns the contributions in the order they were added.
func (sc *StyleCollector) All() []StyleContribution {
	return sc.items
}
