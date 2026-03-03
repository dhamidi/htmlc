// Package htmlc provides the Renderer that evaluates components and produces HTML output.
package htmlc

import (
	stdhtml "html"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/dhamidi/htmlc/expr"
	"golang.org/x/net/html"
)

// Registry maps component names to their parsed components.
// Keys may be PascalCase (e.g., "Card") or kebab-case (e.g., "my-card").
type Registry map[string]*Component

// Renderer walks a component's parsed template and produces HTML output.
type Renderer struct {
	component      *Component
	styleCollector *StyleCollector
	registry       Registry
}

// NewRenderer creates a Renderer for the given component.
func NewRenderer(c *Component) *Renderer {
	return &Renderer{component: c}
}

// WithStyles sets sc as the StyleCollector that will receive this component's
// style contribution when Render is called. It returns the Renderer for
// chaining.
func (r *Renderer) WithStyles(sc *StyleCollector) *Renderer {
	r.styleCollector = sc
	return r
}

// WithComponents attaches a component registry to this renderer, enabling
// component composition. Returns the Renderer for chaining.
func (r *Renderer) WithComponents(reg Registry) *Renderer {
	r.registry = reg
	return r
}

// Render evaluates the component's template against the given data scope and
// returns the rendered HTML string.
func (r *Renderer) Render(scope map[string]any) (string, error) {
	// Collect this component's styles before rendering.
	if r.styleCollector != nil && r.component.Style != "" {
		sid := ScopeID(r.component.Path)
		css := r.component.Style
		if r.component.Scoped {
			css = ScopeCSS(css, "["+sid+"]")
		} else {
			sid = ""
		}
		r.styleCollector.Add(StyleContribution{ScopeID: sid, CSS: css})
	}

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

// outAttr holds a resolved attribute ready for output.
type outAttr struct {
	key      string
	val      string
	boolOnly bool // boolean attr with no value (e.g. disabled)
}

// renderRaw serializes n and its descendants verbatim, without any directive
// processing or interpolation. The v-pre attribute itself is stripped from the
// root element's output. Used by v-pre.
func (r *Renderer) renderRaw(sb *strings.Builder, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		sb.WriteString(stdhtml.EscapeString(n.Data))

	case html.ElementNode:
		sb.WriteByte('<')
		sb.WriteString(n.Data)
		for _, attr := range n.Attr {
			if attr.Key == "v-pre" {
				continue // strip v-pre directive from output
			}
			sb.WriteByte(' ')
			sb.WriteString(attr.Key)
			if attr.Val != "" {
				sb.WriteString(`="`)
				sb.WriteString(stdhtml.EscapeString(attr.Val))
				sb.WriteByte('"')
			}
		}
		if isVoidElement(n.Data) {
			sb.WriteByte('>')
			return
		}
		sb.WriteByte('>')
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			r.renderRaw(sb, child)
		}
		sb.WriteString("</")
		sb.WriteString(n.Data)
		sb.WriteByte('>')

	case html.CommentNode:
		sb.WriteString("<!--")
		sb.WriteString(n.Data)
		sb.WriteString("-->")
	}
}

// renderElement writes the HTML element n into sb, processing directives and
// dynamic attribute bindings (:attr / v-bind:attr).
func (r *Renderer) renderElement(sb *strings.Builder, n *html.Node, scope map[string]any) error {
	// v-pre: emit the element and all descendants verbatim, no processing.
	if _, hasPre := attrValue(n, "v-pre"); hasPre {
		r.renderRaw(sb, n)
		return nil
	}

	// <slot>: emit the default slot content injected by the parent component.
	if n.Data == "slot" {
		if slotContent, ok := scope["$slot"]; ok {
			if s, ok2 := slotContent.(string); ok2 {
				sb.WriteString(s)
				return nil
			}
		}
		// No slot content: render fallback children (if any).
		return r.renderChildren(sb, n, scope)
	}

	// Component: resolve the tag name against the registry.
	if comp := r.resolveComponent(n.Data); comp != nil {
		return r.renderComponentElement(sb, n, scope, comp)
	}
	// Unknown component-like tag (kebab-case with hyphen, not in registry).
	if r.registry != nil && isComponentLike(n.Data) {
		return fmt.Errorf("unknown component: %q", n.Data)
	}

	var vTextExpr, vHTMLExpr, vShowExpr string

	// Static class/style values.
	var staticClass, staticStyle string
	// Other static (non-class/style) attributes.
	var staticAttrs []html.Attribute
	// Resolved dynamic attributes.
	var dynAttrs []outAttr
	// Merged class/style parts from dynamic bindings.
	var dynClassParts []string
	var dynStyleParts []string

	for _, attr := range n.Attr {
		switch attr.Key {
		case "v-text":
			vTextExpr = attr.Val
		case "v-html":
			vHTMLExpr = attr.Val
		case "v-show":
			vShowExpr = attr.Val
		case "v-once":
			// server-side: render normally; consume directive, don't emit
		case "v-if", "v-else-if", "v-else", "v-for":
			// consumed by directives; not emitted as attributes
		case "class":
			staticClass = attr.Val
		case "style":
			staticStyle = attr.Val
		default:
			if strings.HasPrefix(attr.Key, ":") {
				dynKey := attr.Key[1:]
				val, err := expr.Eval(strings.TrimSpace(attr.Val), scope)
				if err != nil {
					return fmt.Errorf("%s %q: %w", attr.Key, attr.Val, err)
				}
				switch dynKey {
				case "key":
					dynAttrs = append(dynAttrs, outAttr{key: "data-key", val: valueToString(val)})
				case "class":
					s, err := resolveClass(val)
					if err != nil {
						return fmt.Errorf(":class: %w", err)
					}
					dynClassParts = append(dynClassParts, s)
				case "style":
					s, err := resolveStyle(val)
					if err != nil {
						return fmt.Errorf(":style: %w", err)
					}
					dynStyleParts = append(dynStyleParts, s)
				default:
					if isBooleanAttr(dynKey) {
						if expr.IsTruthy(val) {
							dynAttrs = append(dynAttrs, outAttr{key: dynKey, boolOnly: true})
						}
						// falsy → omit entirely
					} else {
						dynAttrs = append(dynAttrs, outAttr{key: dynKey, val: valueToString(val)})
					}
				}
			} else {
				staticAttrs = append(staticAttrs, attr)
			}
		}
	}

	// Merge class.
	var classParts []string
	for _, c := range strings.Fields(staticClass) {
		classParts = append(classParts, c)
	}
	for _, p := range dynClassParts {
		for _, c := range strings.Fields(p) {
			classParts = append(classParts, c)
		}
	}

	// Merge style.
	var styleParts []string
	if staticStyle != "" {
		styleParts = append(styleParts, staticStyle)
	}
	for _, p := range dynStyleParts {
		if p != "" {
			styleParts = append(styleParts, p)
		}
	}

	// v-show: inject display:none when falsy.
	if vShowExpr != "" {
		val, err := expr.Eval(strings.TrimSpace(vShowExpr), scope)
		if err != nil {
			return fmt.Errorf("v-show %q: %w", vShowExpr, err)
		}
		if !expr.IsTruthy(val) {
			styleParts = append([]string{"display:none"}, styleParts...)
		}
	}

	// Open tag.
	sb.WriteByte('<')
	sb.WriteString(n.Data)

	// Static non-class/style attrs.
	for _, attr := range staticAttrs {
		sb.WriteByte(' ')
		sb.WriteString(attr.Key)
		sb.WriteString(`="`)
		sb.WriteString(stdhtml.EscapeString(attr.Val))
		sb.WriteByte('"')
	}

	// Merged class.
	if len(classParts) > 0 {
		sb.WriteString(` class="`)
		sb.WriteString(stdhtml.EscapeString(strings.Join(classParts, " ")))
		sb.WriteByte('"')
	}

	// Merged style.
	if len(styleParts) > 0 {
		sb.WriteString(` style="`)
		sb.WriteString(stdhtml.EscapeString(strings.Join(styleParts, ";")))
		sb.WriteByte('"')
	}

	// Dynamic attrs (data-key, boolean, regular).
	for _, a := range dynAttrs {
		sb.WriteByte(' ')
		sb.WriteString(a.key)
		if !a.boolOnly {
			sb.WriteString(`="`)
			sb.WriteString(stdhtml.EscapeString(a.val))
			sb.WriteByte('"')
		}
	}

	// Add scope attribute for scoped components.
	if r.component.Scoped {
		sb.WriteByte(' ')
		sb.WriteString(ScopeID(r.component.Path))
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

// kebabToPascal converts a kebab-case name to PascalCase (e.g. "my-card" → "MyCard").
func kebabToPascal(s string) string {
	parts := strings.Split(s, "-")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

// isComponentLike reports whether a (lowercased) tag name looks like a
// component reference rather than a plain HTML element.  A kebab-case name
// containing a hyphen is the unambiguous indicator.
func isComponentLike(tagName string) bool {
	return strings.Contains(tagName, "-")
}

// resolveComponent looks up tagName in the registry using several strategies:
//  1. Exact match (e.g. "my-card")
//  2. First-letter capitalised (e.g. "card" → "Card", handles lowercased PascalCase)
//  3. kebab-case to PascalCase (e.g. "my-card" → "MyCard")
//  4. Case-insensitive scan (handles multi-capital PascalCase like "postcard" → "PostCard")
func (r *Renderer) resolveComponent(tagName string) *Component {
	if r.registry == nil {
		return nil
	}
	if c, ok := r.registry[tagName]; ok {
		return c
	}
	if len(tagName) > 0 {
		capitalized := strings.ToUpper(tagName[:1]) + tagName[1:]
		if capitalized != tagName {
			if c, ok := r.registry[capitalized]; ok {
				return c
			}
		}
	}
	if strings.Contains(tagName, "-") {
		if c, ok := r.registry[kebabToPascal(tagName)]; ok {
			return c
		}
	}
	lower := strings.ToLower(tagName)
	for key, c := range r.registry {
		if strings.ToLower(key) == lower {
			return c
		}
	}
	return nil
}

// renderComponentElement renders n as a component invocation: props are built
// from the element's attributes, inner content is pre-rendered as the default
// slot, and then the child component's template is rendered with those props.
func (r *Renderer) renderComponentElement(sb *strings.Builder, n *html.Node, scope map[string]any, comp *Component) error {
	childScope := make(map[string]any)

	for _, attr := range n.Attr {
		// Directives that have already been consumed or don't apply to components.
		switch attr.Key {
		case "v-if", "v-else-if", "v-else", "v-for",
			"v-pre", "v-once", "v-show", "v-text", "v-html":
			continue
		}

		if strings.HasPrefix(attr.Key, ":") {
			propName := attr.Key[1:]
			val, err := expr.Eval(strings.TrimSpace(attr.Val), scope)
			if err != nil {
				return fmt.Errorf("%s %q: %w", attr.Key, attr.Val, err)
			}
			childScope[propName] = val
		} else if strings.HasPrefix(attr.Key, "v-bind:") {
			propName := attr.Key[7:]
			val, err := expr.Eval(strings.TrimSpace(attr.Val), scope)
			if err != nil {
				return fmt.Errorf("%s %q: %w", attr.Key, attr.Val, err)
			}
			childScope[propName] = val
		} else {
			// Static attribute → string prop.
			childScope[attr.Key] = attr.Val
		}
	}

	// Pre-render inner content as the default slot (evaluated in the caller's scope).
	var slotSB strings.Builder
	if err := r.renderChildren(&slotSB, n, scope); err != nil {
		return err
	}
	childScope["$slot"] = slotSB.String()

	// Build a child renderer that shares the registry and style collector.
	childRenderer := &Renderer{
		component:      comp,
		styleCollector: r.styleCollector,
		registry:       r.registry,
	}

	result, err := childRenderer.Render(childScope)
	if err != nil {
		return fmt.Errorf("component %q: %w", n.Data, err)
	}
	sb.WriteString(result)
	return nil
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

// booleanAttrs is the set of HTML boolean attributes.
var booleanAttrs = map[string]bool{
	"disabled": true, "checked": true, "selected": true, "readonly": true,
	"required": true, "multiple": true, "autofocus": true, "open": true,
}

func isBooleanAttr(key string) bool { return booleanAttrs[key] }

// resolveClass converts a :class binding value to a space-separated class string.
// Supports object syntax (map[string]any), array syntax ([]any), and string.
func resolveClass(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			if expr.IsTruthy(v[k]) {
				parts = append(parts, k)
			}
		}
		return strings.Join(parts, " "), nil
	case []any:
		var parts []string
		for _, elem := range v {
			s := valueToString(elem)
			if s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, " "), nil
	case nil:
		return "", nil
	default:
		if _, ok := val.(expr.UndefinedValue); ok {
			return "", nil
		}
		return valueToString(val), nil
	}
}

// resolveStyle converts a :style binding value to an inline style string.
// Supports object syntax (map[string]any, keys in camelCase) and string.
func resolveStyle(val any) (string, error) {
	switch v := val.(type) {
	case string:
		return v, nil
	case map[string]any:
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var parts []string
		for _, k := range keys {
			parts = append(parts, camelToKebab(k)+":"+valueToString(v[k]))
		}
		return strings.Join(parts, ";"), nil
	case nil:
		return "", nil
	default:
		if _, ok := val.(expr.UndefinedValue); ok {
			return "", nil
		}
		return valueToString(val), nil
	}
}

// camelToKebab converts a camelCase string to kebab-case (e.g. "fontSize" → "font-size").
func camelToKebab(s string) string {
	var sb strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				sb.WriteByte('-')
			}
			sb.WriteRune(r + 32)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

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
