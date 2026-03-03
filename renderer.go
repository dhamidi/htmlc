// Package htmlc provides the Renderer that evaluates components and produces HTML output.
package htmlc

import (
	stdhtml "html"
	"fmt"
	"math"
	"reflect"
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
		if err := r.renderChildren(sb, n, scope); err != nil {
			return err
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

// conditionalDirective returns the conditional directive key on an element node,
// or "" if none is present.
func conditionalDirective(n *html.Node) string {
	if n.Type != html.ElementNode {
		return ""
	}
	for _, attr := range n.Attr {
		switch attr.Key {
		case "v-if", "v-else-if", "v-else":
			return attr.Key
		}
	}
	return ""
}

// attrValue returns the value of the named attribute, and whether it was present.
func attrValue(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

// nextSignificantSibling returns the next sibling that is not a whitespace-only text node.
func nextSignificantSibling(n *html.Node) *html.Node {
	for sib := n.NextSibling; sib != nil; sib = sib.NextSibling {
		if sib.Type == html.TextNode && strings.TrimSpace(sib.Data) == "" {
			continue
		}
		return sib
	}
	return nil
}

// renderChildren iterates the children of parent and renders them, handling
// v-if/v-else-if/v-else chains and v-for directives.
func (r *Renderer) renderChildren(sb *strings.Builder, parent *html.Node, scope map[string]any) error {
	child := parent.FirstChild
	for child != nil {
		if child.Type == html.ElementNode {
			// v-for takes precedence; render the loop and move on.
			if vforExpr, ok := attrValue(child, "v-for"); ok {
				if err := r.renderVFor(sb, child, vforExpr, scope); err != nil {
					return err
				}
				child = child.NextSibling
				continue
			}
			switch conditionalDirective(child) {
			case "v-if":
				lastInChain, err := r.renderConditionalChain(sb, child, scope)
				if err != nil {
					return err
				}
				child = lastInChain.NextSibling
				continue
			case "v-else-if":
				return fmt.Errorf("v-else-if without preceding v-if or v-else-if")
			case "v-else":
				return fmt.Errorf("v-else without preceding v-if or v-else-if")
			}
		}
		if err := r.renderNode(sb, child, scope); err != nil {
			return err
		}
		child = child.NextSibling
	}
	return nil
}

// renderConditionalChain collects and evaluates a v-if/v-else-if/v-else chain
// starting at vIfNode. It renders the first truthy branch and returns the last
// node consumed in the chain so the caller can advance past it.
func (r *Renderer) renderConditionalChain(sb *strings.Builder, vIfNode *html.Node, scope map[string]any) (*html.Node, error) {
	type condBranch struct {
		node   *html.Node
		expr   string
		isElse bool
	}

	ifExpr, _ := attrValue(vIfNode, "v-if")
	branches := []condBranch{{node: vIfNode, expr: ifExpr}}
	lastNode := vIfNode

	for {
		next := nextSignificantSibling(lastNode)
		if next == nil {
			break
		}
		dir := conditionalDirective(next)
		if dir == "v-else-if" {
			elseIfExpr, _ := attrValue(next, "v-else-if")
			branches = append(branches, condBranch{node: next, expr: elseIfExpr})
			lastNode = next
		} else if dir == "v-else" {
			branches = append(branches, condBranch{node: next, isElse: true})
			lastNode = next
			break
		} else {
			break
		}
	}

	for _, b := range branches {
		var truthy bool
		if b.isElse {
			truthy = true
		} else {
			val, err := expr.Eval(strings.TrimSpace(b.expr), scope)
			if err != nil {
				return nil, fmt.Errorf("v-if %q: %w", b.expr, err)
			}
			truthy = expr.IsTruthy(val)
		}
		if !truthy {
			continue
		}
		// Render the branch. <template> renders only its children.
		if b.node.Data == "template" {
			if err := r.renderChildren(sb, b.node, scope); err != nil {
				return nil, err
			}
		} else {
			if err := r.renderElement(sb, b.node, scope); err != nil {
				return nil, err
			}
		}
		break
	}

	return lastNode, nil
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
		case "v-if", "v-else-if", "v-else", "v-for":
			// consumed by directives; not emitted as attributes
		case ":key":
			// Evaluate and emit as data-key attribute.
			val, err := expr.Eval(strings.TrimSpace(attr.Val), scope)
			if err != nil {
				return fmt.Errorf(":key %q: %w", attr.Val, err)
			}
			attrs = append(attrs, html.Attribute{Key: "data-key", Val: valueToString(val)})
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
		if err := r.renderChildren(sb, n, scope); err != nil {
			return err
		}
	}

	// Close tag.
	sb.WriteString("</")
	sb.WriteString(n.Data)
	sb.WriteByte('>')
	return nil
}

// parseVFor parses a v-for expression into variable names and collection expression.
// Handles: "item in items", "(item, index) in items", "(value, key) in obj", "n in 5".
func parseVFor(vforExpr string) (vars []string, collExpr string, err error) {
	idx := strings.Index(vforExpr, " in ")
	if idx < 0 {
		return nil, "", fmt.Errorf("v-for: invalid expression %q, expected 'x in expr'", vforExpr)
	}
	lhs := strings.TrimSpace(vforExpr[:idx])
	collExpr = strings.TrimSpace(vforExpr[idx+4:])
	if strings.HasPrefix(lhs, "(") && strings.HasSuffix(lhs, ")") {
		inner := lhs[1 : len(lhs)-1]
		for _, p := range strings.Split(inner, ",") {
			vars = append(vars, strings.TrimSpace(p))
		}
	} else {
		vars = []string{lhs}
	}
	return vars, collExpr, nil
}

// scopeWith returns a shallow copy of base with the added key→val binding.
func scopeWith(base map[string]any, key string, val any) map[string]any {
	next := make(map[string]any, len(base)+1)
	for k, v := range base {
		next[k] = v
	}
	next[key] = val
	return next
}

// renderVFor renders n repeatedly for each element in the v-for collection.
func (r *Renderer) renderVFor(sb *strings.Builder, n *html.Node, vforExpr string, scope map[string]any) error {
	vars, collExpr, err := parseVFor(vforExpr)
	if err != nil {
		return err
	}

	collection, err := expr.Eval(collExpr, scope)
	if err != nil {
		return fmt.Errorf("v-for %q: %w", collExpr, err)
	}

	renderOne := func(iterScope map[string]any) error {
		if n.Data == "template" {
			return r.renderChildren(sb, n, iterScope)
		}
		return r.renderElement(sb, n, iterScope)
	}

	if collection == nil {
		return nil
	}
	rv := reflect.ValueOf(collection)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Float64:
		count := int(rv.Float())
		for i := 1; i <= count; i++ {
			if err := renderOne(scopeWith(scope, vars[0], float64(i))); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < rv.Len(); i++ {
			iterScope := scopeWith(scope, vars[0], rv.Index(i).Interface())
			if len(vars) >= 2 {
				iterScope = scopeWith(iterScope, vars[1], float64(i))
			}
			if err := renderOne(iterScope); err != nil {
				return err
			}
		}
	case reflect.Map:
		for _, k := range rv.MapKeys() {
			iterScope := scopeWith(scope, vars[0], rv.MapIndex(k).Interface())
			if len(vars) >= 2 {
				iterScope = scopeWith(iterScope, vars[1], k.Interface())
			}
			if err := renderOne(iterScope); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("v-for: cannot iterate over %T", collection)
	}
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
