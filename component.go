package htmlc

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/dhamidi/htmlc/expr"
)

// Component holds the parsed representation of a .vue Single File Component.
type Component struct {
	// Template is the root of the parsed HTML node tree for the <template> section.
	Template *html.Node
	// Script is the raw text content of the <script> section (empty if absent).
	Script string
	// Style is the raw text content of the <style> section (empty if absent).
	Style string
	// Scoped reports whether the <style> tag carried the scoped attribute.
	Scoped bool
	// Path is the source file path passed to ParseFile.
	Path string
	// Source is the raw source text of the file, stored for location-aware error reporting.
	Source string
	// Warnings holds non-fatal issues found during parsing, such as self-closing
	// custom component tags that were automatically rewritten.
	Warnings []string
}

// lineCol returns the 1-based line and column for byteOffset within src.
func lineCol(src string, byteOffset int) (line, col int) {
	line = 1
	col = 1
	for i, ch := range src {
		if i >= byteOffset {
			break
		}
		if ch == '\n' {
			line++
			col = 1
		} else {
			col++
		}
	}
	return
}

// snippet returns a 3-line context string centred on the given 1-based line.
func snippet(src string, line int) string {
	lines := strings.Split(src, "\n")
	start := line - 2
	if start < 0 {
		start = 0
	}
	end := line + 1
	if end > len(lines) {
		end = len(lines)
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		marker := "  "
		if i+1 == line {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%3d | %s\n", marker, i+1, lines[i])
	}
	return b.String()
}

// ParseFile parses a .vue Single File Component from src and returns a Component.
// Only the top-level <template>, <script>, and <style> sections are extracted.
// <script> and <style> are optional; <template> is required.
// The template HTML is parsed into a node tree accessible via Component.Template.
func ParseFile(path, src string) (*Component, error) {
	sections, err := extractSections(src)
	if err != nil {
		// Attempt to locate the error in the source for a better message.
		msg := err.Error()
		return nil, &ParseError{Path: path, Msg: msg}
	}

	if sections["script"] != "" {
		return nil, &ParseError{
			Path: path,
			Msg: `<script> blocks are not supported by htmlc; ` +
				`components are rendered on the server`,
		}
	}
	if sections["script:setup"] != "" {
		return nil, &ParseError{
			Path: path,
			Msg: `<script setup> blocks are not supported by htmlc; ` +
				`components are rendered on the server`,
		}
	}

	tmplContent, ok := sections["template"]
	if !ok {
		return nil, &ParseError{Path: path, Msg: "missing <template> section"}
	}

	normalized, count := normalizeSelfClosingComponents(tmplContent)

	templateRoot, err := parseTemplateHTML(normalized)
	if err != nil {
		// Find the template section start in src to compute a line number.
		loc := locateInSource(path, src, "template", err.Error())
		return nil, &ParseError{Path: path, Msg: "parsing template: " + err.Error(), Location: loc}
	}

	c := &Component{
		Template: templateRoot,
		Script:   sections["script"],
		Style:    sections["style"],
		Scoped:   sections["style:scoped"] == "true",
		Path:     path,
		Source:   src,
	}
	if count > 0 {
		c.Warnings = append(c.Warnings, fmt.Sprintf(
			"%s: %d self-closing custom component tag(s) were auto-corrected; "+
				"prefer explicit open/close tags", path, count))
	}
	return c, nil
}

// locateInSource tries to find the <template> section start offset in src and
// returns a SourceLocation pointing into it. errMsg is ignored for now; the
// location points to the opening <template> tag line as the best approximation
// when the HTML parser does not expose byte offsets.
func locateInSource(path, src, _ /*section*/, _ /*errMsg*/ string) *SourceLocation {
	idx := strings.Index(src, "<template")
	if idx < 0 {
		return nil
	}
	// Advance past "<template" and any attributes to the first ">".
	closeIdx := strings.Index(src[idx:], ">")
	if closeIdx < 0 {
		return nil
	}
	// Point to the line just after the opening tag.
	contentStart := idx + closeIdx + 1
	ln, col := lineCol(src, contentStart)
	return &SourceLocation{
		File:    path,
		Line:    ln,
		Column:  col,
		Snippet: snippet(src, ln),
	}
}

// sectionInfo tracks state while collecting a top-level section.
type sectionInfo struct {
	tag    string
	key    string // result map key; may differ from tag (e.g. "script:setup")
	depth  int
	scoped bool
	buf    strings.Builder
}

// extractSections tokenises src at the outer level and returns a map of
// tag-name → inner text for each recognised top-level section.
// It also returns "style:scoped" = "true" when the style tag is scoped.
func extractSections(src string) (map[string]string, error) {
	result := map[string]string{}

	z := html.NewTokenizer(strings.NewReader(src))
	var current *sectionInfo

	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			if current != nil {
				return nil, fmt.Errorf("unclosed <%s> section", current.tag)
			}
			break
		}

		// Save raw bytes before calling z.Token(), which may modify the
		// tokenizer's internal buffer and change what z.Raw() returns.
		// Using the raw bytes for inner section content preserves the
		// original casing of tag names (the HTML tokenizer lowercases Data).
		raw := string(z.Raw())
		tok := z.Token()

		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			tagName := tok.Data

			if current == nil {
				// Top-level start tag: begin a new section if it's one we care about.
				switch tagName {
				case "template", "script", "style":
					sectionKey := tagName
					scoped := false
					for _, attr := range tok.Attr {
						if attr.Key == "scoped" {
							scoped = true
						}
						if tagName == "script" && attr.Key == "setup" {
							sectionKey = "script:setup"
						}
					}
					if _, dup := result[sectionKey]; dup {
						return nil, fmt.Errorf("duplicate <%s> section", tagName)
					}
					if tt == html.SelfClosingTagToken {
						// Self-closing: treat as empty section.
						result[sectionKey] = ""
						if tagName == "style" && scoped {
							result["style:scoped"] = "true"
						}
					} else {
						current = &sectionInfo{tag: tagName, key: sectionKey, depth: 1, scoped: scoped}
					}
				}
			} else {
				// Inside a section: track nesting depth for the same tag.
				if tagName == current.tag {
					current.depth++
				}
				// Append the raw token text to the section buffer, preserving
				// the original casing of tag names and attribute names.
				current.buf.WriteString(raw)
			}

		case html.EndTagToken:
			tagName := tok.Data

			if current != nil && tagName == current.tag {
				current.depth--
				if current.depth == 0 {
					// Finished collecting this section.
					result[current.key] = current.buf.String()
					if current.tag == "style" && current.scoped {
						result["style:scoped"] = "true"
					}
					current = nil
					continue
				}
			}
			if current != nil {
				current.buf.WriteString(raw)
			}

		case html.TextToken:
			if current != nil {
				// Use the raw bytes rather than tok.Data: tok.Data for TextTokens
				// contains the HTML-decoded string, which corrupts CSS content
				// (e.g. quoted strings in @font-face, & in content properties).
				// raw holds the verbatim source bytes, which is what <style> and
				// <script> sections require.
				current.buf.WriteString(raw)
			}
		case html.CommentToken, html.DoctypeToken:
			if current != nil {
				current.buf.WriteString(raw)
			}

		}
	}

	return result, nil
}

// PropInfo describes a single top-level prop (variable reference) found in a
// component's template, together with the expression strings in which it appears.
type PropInfo struct {
	Name        string
	Expressions []string
}

var interpolationRe = regexp.MustCompile(`\{\{(.*?)\}\}`)

// Props walks the component's parsed template AST and returns all top-level
// variable references (props) that the template uses.
//
// Identifiers starting with '$' are excluded.
// v-for loop variables are excluded within their subtree.
func (c *Component) Props() []PropInfo {
	props := map[string]*PropInfo{}
	walkForProps(c.Template, map[string]bool{}, props)

	result := make([]PropInfo, 0, len(props))
	for _, p := range props {
		result = append(result, *p)
	}
	return result
}

func walkForProps(n *html.Node, locals map[string]bool, props map[string]*PropInfo) {
	switch n.Type {
	case html.TextNode:
		for _, m := range interpolationRe.FindAllStringSubmatch(n.Data, -1) {
			collectExprIdents(strings.TrimSpace(m[1]), locals, props)
		}
		return
	case html.ElementNode:
		// v-pre: skip all prop collection for this element and its descendants.
		for _, attr := range n.Attr {
			if attr.Key == "v-pre" {
				return
			}
		}
		childLocals := cloneLocals(locals)

		for _, attr := range n.Attr {
			if attr.Key == "v-for" {
				if idx := strings.Index(attr.Val, " in "); idx >= 0 {
					collExpr := strings.TrimSpace(attr.Val[idx+4:])
					varsStr := strings.TrimSpace(attr.Val[:idx])
					// Collection expression is scanned in the parent scope.
					collectExprIdents(collExpr, locals, props)
					for _, v := range parseVForVars(varsStr) {
						childLocals[v] = true
					}
				}
				continue
			}
			// v-slot / # directives: binding variables are locals within the template.
			if _, isSlot := parseSlotDirective(attr.Key); isSlot {
				bindingVar, bindings, _ := parseBindingPattern(attr.Val)
				if bindingVar != "" {
					childLocals[bindingVar] = true
				}
				for _, b := range bindings {
					childLocals[b] = true
				}
				continue
			}
			var exprVal string
			switch {
			case strings.HasPrefix(attr.Key, ":"), strings.HasPrefix(attr.Key, "v-bind:"):
				exprVal = attr.Val
			case attr.Key == "v-if", attr.Key == "v-else-if", attr.Key == "v-show",
				attr.Key == "v-text", attr.Key == "v-html":
				exprVal = attr.Val
			}
			if exprVal != "" {
				collectExprIdents(exprVal, childLocals, props)
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walkForProps(child, childLocals, props)
		}
		return
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		walkForProps(child, locals, props)
	}
}

func collectExprIdents(exprStr string, locals map[string]bool, props map[string]*PropInfo) {
	idents, err := expr.CollectIdentifiers(exprStr)
	if err != nil {
		return
	}
	for _, name := range idents {
		if strings.HasPrefix(name, "$") || locals[name] {
			continue
		}
		if _, ok := props[name]; !ok {
			props[name] = &PropInfo{Name: name}
		}
		props[name].Expressions = append(props[name].Expressions, exprStr)
	}
}

func cloneLocals(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func parseVForVars(s string) []string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		s = s[1 : len(s)-1]
		parts := strings.Split(s, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				result = append(result, p)
			}
		}
		return result
	}
	return []string{s}
}

// selfClosingComponentRe matches self-closing tags whose name starts with an
// uppercase ASCII letter (PascalCase custom components). Attribute values
// containing quoted strings (including ones with /> inside) are handled by the
// alternation in the attribute capture group.
var selfClosingComponentRe = regexp.MustCompile(
	`<([A-Z][a-zA-Z0-9]*)((?:[^"'>]|"[^"]*"|'[^']*')*?)\s*/>`,
)

// normalizeSelfClosingComponents rewrites <Name ... /> as <Name ...></Name>
// for any tag whose name begins with an uppercase letter, so that the HTML5
// parser does not silently ignore the self-closing syntax. It returns the
// rewritten source and the number of replacements made.
func normalizeSelfClosingComponents(src string) (string, int) {
	matches := selfClosingComponentRe.FindAllString(src, -1)
	count := len(matches)
	result := selfClosingComponentRe.ReplaceAllString(src, "<$1$2></$1>")
	return result, count
}

// parseTemplateHTML parses the raw HTML string from a <template> section into
// a root node whose children are the actual template nodes.
//
// When the trimmed content begins with "<html" or "<!doctype" (case-insensitive)
// it is treated as a full HTML document and parsed with html.Parse, which
// correctly preserves the <html>, <head>, and <body> elements that
// html.ParseFragment (with a <div> context) would silently discard.
//
// For all other templates html.ParseFragment with a <div> context is used so
// that partial components (e.g. <article>, <li>) continue to work correctly.
// The fragment nodes are wrapped in a synthetic DocumentNode for uniform
// traversal by the renderer.
func parseTemplateHTML(content string) (*html.Node, error) {
	lower := strings.ToLower(strings.TrimSpace(content))
	if strings.HasPrefix(lower, "<html") || strings.HasPrefix(lower, "<!doctype") {
		doc, err := html.Parse(strings.NewReader(content))
		if err != nil {
			return nil, err
		}
		return doc, nil
	}

	context := &html.Node{
		Type:     html.ElementNode,
		DataAtom: atom.Div,
		Data:     "div",
	}
	nodes, err := html.ParseFragment(strings.NewReader(content), context)
	if err != nil {
		return nil, err
	}

	// Wrap the fragment nodes under a synthetic root so callers always have a
	// single entry point.
	root := &html.Node{Type: html.DocumentNode}
	for _, n := range nodes {
		root.AppendChild(n)
	}
	return root, nil
}
