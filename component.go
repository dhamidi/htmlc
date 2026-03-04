package htmlc

import (
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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
}

// ParseFile parses a .vue Single File Component from src and returns a Component.
// Only the top-level <template>, <script>, and <style> sections are extracted.
// <script> and <style> are optional; <template> is required.
// The template HTML is parsed into a node tree accessible via Component.Template.
func ParseFile(path, src string) (*Component, error) {
	sections, err := extractSections(src)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	tmplContent, ok := sections["template"]
	if !ok {
		return nil, fmt.Errorf("%s: missing <template> section", path)
	}

	templateRoot, err := parseTemplateHTML(tmplContent)
	if err != nil {
		return nil, fmt.Errorf("%s: parsing template: %w", path, err)
	}

	c := &Component{
		Template: templateRoot,
		Script:   sections["script"],
		Style:    sections["style"],
		Scoped:   sections["style:scoped"] == "true",
		Path:     path,
	}
	return c, nil
}

// sectionInfo tracks state while collecting a top-level section.
type sectionInfo struct {
	tag    string
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

		tok := z.Token()

		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			tagName := tok.Data

			if current == nil {
				// Top-level start tag: begin a new section if it's one we care about.
				switch tagName {
				case "template", "script", "style":
					if _, dup := result[tagName]; dup {
						return nil, fmt.Errorf("duplicate <%s> section", tagName)
					}
					scoped := false
					for _, attr := range tok.Attr {
						if attr.Key == "scoped" {
							scoped = true
							break
						}
					}
					if tt == html.SelfClosingTagToken {
						// Self-closing: treat as empty section.
						result[tagName] = ""
						if tagName == "style" && scoped {
							result["style:scoped"] = "true"
						}
					} else {
						current = &sectionInfo{tag: tagName, depth: 1, scoped: scoped}
					}
				}
			} else {
				// Inside a section: track nesting depth for the same tag.
				if tagName == current.tag {
					current.depth++
				}
				// Append the raw token text to the section buffer.
				current.buf.WriteString(tok.String())
			}

		case html.EndTagToken:
			tagName := tok.Data

			if current != nil && tagName == current.tag {
				current.depth--
				if current.depth == 0 {
					// Finished collecting this section.
					result[current.tag] = current.buf.String()
					if current.tag == "style" && current.scoped {
						result["style:scoped"] = "true"
					}
					current = nil
					continue
				}
			}
			if current != nil {
				current.buf.WriteString(tok.String())
			}

		case html.TextToken, html.CommentToken, html.DoctypeToken:
			if current != nil {
				current.buf.WriteString(tok.String())
			}

		}
	}

	return result, nil
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
