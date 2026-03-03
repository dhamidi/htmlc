// Package htmlc provides the Renderer that evaluates components and produces HTML output.
package htmlc

import (
	stdhtml "html"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/dhamidi/htmlc/expr"
	"golang.org/x/net/html"
)

// Renderer walks a component's parsed template and produces HTML output.
type Renderer struct {
	component *Component
}

// NewRenderer creates a Renderer for the given component.
func NewRenderer(c *Component) *Renderer {
	return &Renderer{component: c}
}

// Render evaluates the component's template against the given data scope and
// returns the rendered HTML string.
func (r *Renderer) Render(scope map[string]any) (string, error) {
	var sb strings.Builder
	if err := r.renderNode(&sb, r.component.Template, scope); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// Render is a convenience function that creates a Renderer for c and calls Render.
func Render(c *Component, scope map[string]any) (string, error) {
	return NewRenderer(c).Render(scope)
}

// mustacheRe matches {{ expression }} patterns inside text nodes.
var mustacheRe = regexp.MustCompile(`\{\{(.*?)\}\}`)

// renderNode recursively writes n into sb.
func (r *Renderer) renderNode(sb *strings.Builder, n *html.Node, scope map[string]any) error {
	switch n.Type {
	case html.DocumentNode:
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if err := r.renderNode(sb, child, scope); err != nil {
				return err
			}
		}

	case html.TextNode:
		result, err := r.interpolate(n.Data, scope)
		if err != nil {
			return err
		}
		sb.WriteString(result)

	case html.ElementNode:
		if err := r.renderElement(sb, n, scope); err != nil {
			return err
		}

	case html.CommentNode:
		sb.WriteString("<!--")
		sb.WriteString(n.Data)
		sb.WriteString("-->")

	case html.DoctypeNode:
		sb.WriteString("<!DOCTYPE ")
		sb.WriteString(n.Data)
		sb.WriteByte('>')
	}
	return nil
}

// interpolate processes mustache expressions within text and returns the result.
// Literal segments are HTML-escaped; {{ expr }} segments are evaluated and escaped.
func (r *Renderer) interpolate(text string, scope map[string]any) (string, error) {
	var sb strings.Builder
	lastEnd := 0

	for _, loc := range mustacheRe.FindAllStringSubmatchIndex(text, -1) {
		// Write literal text before this match, HTML-escaped.
		sb.WriteString(stdhtml.EscapeString(text[lastEnd:loc[0]]))

		// Evaluate the captured expression (loc[2]:loc[3]).
		exprSrc := strings.TrimSpace(text[loc[2]:loc[3]])
		val, err := expr.Eval(exprSrc, scope)
		if err != nil {
			return "", fmt.Errorf("interpolation %q: %w", exprSrc, err)
		}
		sb.WriteString(stdhtml.EscapeString(valueToString(val)))

		lastEnd = loc[1]
	}
	// Write remaining literal text.
	sb.WriteString(stdhtml.EscapeString(text[lastEnd:]))
	return sb.String(), nil
}

// renderElement writes the HTML element n into sb, processing v-text/v-html directives.
func (r *Renderer) renderElement(sb *strings.Builder, n *html.Node, scope map[string]any) error {
	var vTextExpr, vHTMLExpr string
	var attrs []html.Attribute

	for _, attr := range n.Attr {
		switch attr.Key {
		case "v-text":
			vTextExpr = attr.Val
		case "v-html":
			vHTMLExpr = attr.Val
		default:
			attrs = append(attrs, attr)
		}
	}

	// Open tag.
	sb.WriteByte('<')
	sb.WriteString(n.Data)
	for _, attr := range attrs {
		sb.WriteByte(' ')
		sb.WriteString(attr.Key)
		sb.WriteString(`="`)
		sb.WriteString(stdhtml.EscapeString(attr.Val))
		sb.WriteByte('"')
	}

	if isVoidElement(n.Data) {
		sb.WriteByte('>')
		return nil
	}
	sb.WriteByte('>')

	// Content: v-text, v-html, or child nodes.
	switch {
	case vTextExpr != "":
		val, err := expr.Eval(strings.TrimSpace(vTextExpr), scope)
		if err != nil {
			return fmt.Errorf("v-text %q: %w", vTextExpr, err)
		}
		sb.WriteString(stdhtml.EscapeString(valueToString(val)))

	case vHTMLExpr != "":
		val, err := expr.Eval(strings.TrimSpace(vHTMLExpr), scope)
		if err != nil {
			return fmt.Errorf("v-html %q: %w", vHTMLExpr, err)
		}
		sb.WriteString(valueToString(val))

	default:
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if err := r.renderNode(sb, child, scope); err != nil {
				return err
			}
		}
	}

	// Close tag.
	sb.WriteString("</")
	sb.WriteString(n.Data)
	sb.WriteByte('>')
	return nil
}

// voidElements is the set of HTML void elements that must not have a closing tag.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

func isVoidElement(tag string) bool { return voidElements[tag] }

// valueToString converts an evaluated expression value to its string representation.
func valueToString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if math.IsNaN(val) {
			return "NaN"
		}
		if math.IsInf(val, 1) {
			return "Infinity"
		}
		if math.IsInf(val, -1) {
			return "-Infinity"
		}
		if val == math.Trunc(val) && val >= -1e15 && val <= 1e15 {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		if _, ok := v.(expr.UndefinedValue); ok {
			return "undefined"
		}
		return fmt.Sprintf("%v", v)
	}
}
