package htmlc

import (
	stdhtml "html"
	"fmt"
	"io"
	"math"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/dhamidi/htmlc/expr"
	"golang.org/x/net/html"
)

// SlotDefinition captures everything needed to defer slot rendering: the AST
// nodes from the caller's template, the caller's scope at invocation time, and
// the parsed binding information from the v-slot / # directive.
type SlotDefinition struct {
	Nodes       []*html.Node
	ParentScope map[string]any
	BindingVar  string
	Bindings    []string
}

// identRe matches a valid JS/Vue identifier: starts with letter, _ or $,
// followed by letters, digits, _ or $.
var identRe = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*$`)

// parseBindingPattern parses the value of a v-slot / # directive into binding
// information. Returns (bindingVar, bindings, err).
//   - Empty string → no bindings
//   - Single identifier → bindingVar set, bindings nil
//   - Destructured "{ a, b }" → bindings set, bindingVar empty
//   - Anything else → error
func parseBindingPattern(s string) (bindingVar string, bindings []string, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil, nil
	}
	if strings.HasPrefix(s, "{") {
		if !strings.HasSuffix(s, "}") {
			return "", nil, fmt.Errorf("parseBindingPattern: invalid pattern %q", s)
		}
		inner := strings.TrimSpace(s[1 : len(s)-1])
		if inner == "" {
			return "", nil, fmt.Errorf("parseBindingPattern: empty destructure in %q", s)
		}
		parts := strings.Split(inner, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				return "", nil, fmt.Errorf("parseBindingPattern: trailing comma in %q", s)
			}
			if !identRe.MatchString(p) {
				return "", nil, fmt.Errorf("parseBindingPattern: invalid identifier %q in %q", p, s)
			}
			result = append(result, p)
		}
		return "", result, nil
	}
	if identRe.MatchString(s) {
		return s, nil, nil
	}
	return "", nil, fmt.Errorf("parseBindingPattern: invalid pattern %q", s)
}

// parseSlotDirective parses an attribute name and returns the slot name and
// whether it is a slot directive.
//   - "v-slot"        → ("default", true)
//   - "v-slot:header" → ("header", true)
//   - "#header"       → ("header", true)
//   - "#default"      → ("default", true)
//   - anything else   → ("", false)
func parseSlotDirective(attrName string) (slotName string, isSlotDirective bool) {
	if attrName == "v-slot" {
		return "default", true
	}
	if strings.HasPrefix(attrName, "v-slot:") {
		return attrName[7:], true
	}
	if strings.HasPrefix(attrName, "#") {
		return attrName[1:], true
	}
	return "", false
}

// Registry maps component names to their parsed components.
// Keys may be PascalCase (e.g., "Card") or kebab-case (e.g., "my-card").
// Registry is part of the low-level API; most callers should use Engine, which
// builds and maintains a Registry automatically from a component directory.
type Registry map[string]*Component

// MissingPropFunc is called when a prop expected by the component's template
// is not present in the render scope. It receives the prop name and returns
// a substitute value, or an error to abort rendering.
type MissingPropFunc func(name string) (any, error)

// SubstituteMissingProp returns a placeholder string "MISSING PROP: <name>"
// for any missing prop.
func SubstituteMissingProp(name string) (any, error) {
	return fmt.Sprintf("MISSING PROP: %s", name), nil
}

// Renderer walks a component's parsed template and produces HTML output.
// It is the low-level rendering primitive — most callers should use Engine
// (via RenderPage or RenderFragment) rather than constructing a Renderer
// directly. Use NewRenderer when you need fine-grained control over style
// collection or registry attachment.
type Renderer struct {
	component          *Component
	styleCollector     *StyleCollector
	registry           Registry
	missingPropHandler MissingPropFunc
	slotDefs           map[string]*SlotDefinition
}

// NewRenderer creates a Renderer for c. Call WithStyles and WithComponents
// before Render to enable style collection and component composition.
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

// WithMissingPropHandler sets a handler that is called when a prop referenced
// in the template is not present in the render scope. Returns the Renderer for
// chaining.
func (r *Renderer) WithMissingPropHandler(fn MissingPropFunc) *Renderer {
	r.missingPropHandler = fn
	return r
}

// validateProps checks scope against the component's expected props. If a prop
// is missing and a handler is set, the handler's returned value is injected
// into a copy of the scope. If no handler is set, an error is returned.
//
// HTML parsers lowercase all attribute names, so a caller that writes
// :myProp="x" will produce a scope key "myprop" instead of "myProp". When an
// exact match fails, validateProps does a case-insensitive fallback: if a
// matching key exists under a different case, the correct-case entry is added
// to the augmented scope so template expressions like {{ myProp }} resolve.
func (r *Renderer) validateProps(scope map[string]any) (map[string]any, error) {
	props := r.component.Props()
	var augmented map[string]any
	for _, p := range props {
		if _, ok := scope[p.Name]; ok {
			continue
		}
		// Fallback: HTML parsers lowercase attribute names. Look for a
		// case-insensitive match and re-inject under the expected casing.
		lowerName := strings.ToLower(p.Name)
		var caseVal any
		caseFound := false
		for k, v := range scope {
			if strings.ToLower(k) == lowerName {
				caseVal = v
				caseFound = true
				break
			}
		}
		if caseFound {
			if augmented == nil {
				augmented = make(map[string]any, len(scope))
				for k, v := range scope {
					augmented[k] = v
				}
			}
			augmented[p.Name] = caseVal
			continue
		}
		if r.missingPropHandler != nil {
			val, err := r.missingPropHandler(p.Name)
			if err != nil {
				return nil, err
			}
			if augmented == nil {
				augmented = make(map[string]any, len(scope)+1)
				for k, v := range scope {
					augmented[k] = v
				}
			}
			augmented[p.Name] = val
		} else {
			return nil, fmt.Errorf("missing prop %q (used in: %s)", p.Name, strings.Join(p.Expressions, ", "))
		}
	}
	if augmented != nil {
		return augmented, nil
	}
	return scope, nil
}

// Render evaluates the component's template against the given data scope and
// writes the rendered HTML directly to w.
func (r *Renderer) Render(w io.Writer, scope map[string]any) error {
	var err error
	scope, err = r.validateProps(scope)
	if err != nil {
		return err
	}

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

	return r.renderNode(w, r.component.Template, scope)
}

// RenderString evaluates the component's template against the given data scope
// and returns the rendered HTML as a string. It is a convenience wrapper
// around Render.
func (r *Renderer) RenderString(scope map[string]any) (string, error) {
	var sb strings.Builder
	if err := r.Render(&sb, scope); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// RenderString is a convenience wrapper that creates a temporary Renderer for c
// and renders it against scope, returning the result as a string. It does not
// collect styles or support component composition; use NewRenderer with
// WithStyles and WithComponents for those features.
func RenderString(c *Component, scope map[string]any) (string, error) {
	return NewRenderer(c).RenderString(scope)
}

// Render is a convenience wrapper that creates a temporary Renderer for c and
// writes the rendered HTML directly to w. It does not collect styles or support
// component composition; use NewRenderer with WithStyles and WithComponents for
// those features.
func Render(w io.Writer, c *Component, scope map[string]any) error {
	return NewRenderer(c).Render(w, scope)
}

// mustacheRe matches {{ expression }} patterns inside text nodes.
var mustacheRe = regexp.MustCompile(`\{\{(.*?)\}\}`)

// renderNode recursively writes n into w.
func (r *Renderer) renderNode(w io.Writer, n *html.Node, scope map[string]any) error {
	switch n.Type {
	case html.DocumentNode:
		if err := r.renderChildren(w, n, scope); err != nil {
			return err
		}

	case html.TextNode:
		if err := r.interpolate(w, n.Data, scope); err != nil {
			return err
		}

	case html.ElementNode:
		if err := r.renderElement(w, n, scope); err != nil {
			return err
		}

	case html.CommentNode:
		io.WriteString(w, "<!--")
		io.WriteString(w, n.Data)
		io.WriteString(w, "-->")

	case html.DoctypeNode:
		io.WriteString(w, "<!DOCTYPE ")
		io.WriteString(w, n.Data)
		w.Write([]byte{'>'})
	}
	return nil
}

// interpolate processes mustache expressions within text and writes the result to w.
// Literal segments are HTML-escaped; {{ expr }} segments are evaluated and escaped.
func (r *Renderer) interpolate(w io.Writer, text string, scope map[string]any) error {
	lastEnd := 0

	for _, loc := range mustacheRe.FindAllStringSubmatchIndex(text, -1) {
		// Write literal text before this match, HTML-escaped.
		io.WriteString(w, stdhtml.EscapeString(text[lastEnd:loc[0]]))

		// Evaluate the captured expression (loc[2]:loc[3]).
		exprSrc := strings.TrimSpace(text[loc[2]:loc[3]])
		val, err := expr.Eval(exprSrc, scope)
		if err != nil {
			return fmt.Errorf("interpolation %q: %w", exprSrc, err)
		}
		io.WriteString(w, stdhtml.EscapeString(valueToString(val)))

		lastEnd = loc[1]
	}
	// Write remaining literal text.
	io.WriteString(w, stdhtml.EscapeString(text[lastEnd:]))
	return nil
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
func (r *Renderer) renderChildren(w io.Writer, parent *html.Node, scope map[string]any) error {
	child := parent.FirstChild
	for child != nil {
		if child.Type == html.ElementNode {
			// v-for takes precedence; render the loop and move on.
			if vforExpr, ok := attrValue(child, "v-for"); ok {
				if err := r.renderVFor(w, child, vforExpr, scope); err != nil {
					return err
				}
				child = child.NextSibling
				continue
			}
			switch conditionalDirective(child) {
			case "v-if":
				lastInChain, err := r.renderConditionalChain(w, child, scope)
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
		if err := r.renderNode(w, child, scope); err != nil {
			return err
		}
		child = child.NextSibling
	}
	return nil
}

// renderConditionalChain collects and evaluates a v-if/v-else-if/v-else chain
// starting at vIfNode. It renders the first truthy branch and returns the last
// node consumed in the chain so the caller can advance past it.
func (r *Renderer) renderConditionalChain(w io.Writer, vIfNode *html.Node, scope map[string]any) (*html.Node, error) {
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
			if err := r.renderChildren(w, b.node, scope); err != nil {
				return nil, err
			}
		} else {
			if err := r.renderElement(w, b.node, scope); err != nil {
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
func (r *Renderer) renderRaw(w io.Writer, n *html.Node) {
	switch n.Type {
	case html.TextNode:
		io.WriteString(w, stdhtml.EscapeString(n.Data))

	case html.ElementNode:
		w.Write([]byte{'<'})
		io.WriteString(w, n.Data)
		for _, attr := range n.Attr {
			if attr.Key == "v-pre" {
				continue // strip v-pre directive from output
			}
			w.Write([]byte{' '})
			io.WriteString(w, attr.Key)
			if attr.Val != "" {
				io.WriteString(w, `="`)
				io.WriteString(w, stdhtml.EscapeString(attr.Val))
				w.Write([]byte{'"'})
			}
		}
		if isVoidElement(n.Data) {
			w.Write([]byte{'>'})
			return
		}
		w.Write([]byte{'>'})
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			r.renderRaw(w, child)
		}
		io.WriteString(w, "</")
		io.WriteString(w, n.Data)
		w.Write([]byte{'>'})

	case html.CommentNode:
		io.WriteString(w, "<!--")
		io.WriteString(w, n.Data)
		io.WriteString(w, "-->")
	}
}

// renderElement writes the HTML element n into w, processing directives and
// dynamic attribute bindings (:attr / v-bind:attr).
func (r *Renderer) renderElement(w io.Writer, n *html.Node, scope map[string]any) error {
	// v-pre: emit the element and all descendants verbatim, no processing.
	if _, hasPre := attrValue(n, "v-pre"); hasPre {
		r.renderRaw(w, n)
		return nil
	}

	// <template> without a controlling directive: render children transparently,
	// without emitting a <template> wrapper element. Directives like v-if and
	// v-for are intercepted in renderChildren before renderElement is reached,
	// so this branch only fires for plain <template> elements.
	if n.Data == "template" {
		return r.renderChildren(w, n, scope)
	}

	// <slot>: emit slot content from the caller's SlotDefinition, or fallback children.
	if n.Data == "slot" {
		slotName := "default"
		if nameAttr, ok := attrValue(n, "name"); ok {
			slotName = nameAttr
		}
		if def, ok := r.slotDefs[slotName]; ok {
			// Collect slot props from the <slot> element's attributes (child's scope).
			slotProps := make(map[string]any)
			for _, attr := range n.Attr {
				if attr.Key == "name" {
					continue
				}
				if strings.HasPrefix(attr.Key, ":") {
					propName := attr.Key[1:]
					val, err := expr.Eval(strings.TrimSpace(attr.Val), scope)
					if err != nil {
						return fmt.Errorf("slot prop %s %q: %w", attr.Key, attr.Val, err)
					}
					slotProps[propName] = val
				} else if strings.HasPrefix(attr.Key, "v-bind:") {
					propName := attr.Key[7:]
					val, err := expr.Eval(strings.TrimSpace(attr.Val), scope)
					if err != nil {
						return fmt.Errorf("slot prop %s %q: %w", attr.Key, attr.Val, err)
					}
					slotProps[propName] = val
				} else {
					slotProps[attr.Key] = attr.Val
				}
			}

			// Build render scope: clone the parent scope.
			renderScope := make(map[string]any, len(def.ParentScope)+len(slotProps))
			for k, v := range def.ParentScope {
				renderScope[k] = v
			}

			// Apply binding pattern (slot props override parent scope).
			switch {
			case def.BindingVar != "":
				renderScope[def.BindingVar] = slotProps
			case len(def.Bindings) > 0:
				for _, name := range def.Bindings {
					if val, ok := slotProps[name]; ok {
						renderScope[name] = val
					} else {
						renderScope[name] = nil
					}
				}
			// No binding: render with parent scope only; slot props discarded.
			}

			for _, node := range def.Nodes {
				if err := r.renderNode(w, node, renderScope); err != nil {
					return err
				}
			}
			return nil
		}
		// No slot definition: render fallback children (if any).
		return r.renderChildren(w, n, scope)
	}

	// Component: resolve the tag name against the registry.
	if comp := r.resolveComponent(n.Data); comp != nil {
		return r.renderComponentElement(w, n, scope, comp)
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
	w.Write([]byte{'<'})
	io.WriteString(w, n.Data)

	// Static non-class/style attrs.
	for _, attr := range staticAttrs {
		w.Write([]byte{' '})
		io.WriteString(w, attr.Key)
		io.WriteString(w, `="`)
		io.WriteString(w, stdhtml.EscapeString(attr.Val))
		w.Write([]byte{'"'})
	}

	// Merged class.
	if len(classParts) > 0 {
		io.WriteString(w, ` class="`)
		io.WriteString(w, stdhtml.EscapeString(strings.Join(classParts, " ")))
		w.Write([]byte{'"'})
	}

	// Merged style.
	if len(styleParts) > 0 {
		io.WriteString(w, ` style="`)
		io.WriteString(w, stdhtml.EscapeString(strings.Join(styleParts, ";")))
		w.Write([]byte{'"'})
	}

	// Dynamic attrs (data-key, boolean, regular).
	for _, a := range dynAttrs {
		w.Write([]byte{' '})
		io.WriteString(w, a.key)
		if !a.boolOnly {
			io.WriteString(w, `="`)
			io.WriteString(w, stdhtml.EscapeString(a.val))
			w.Write([]byte{'"'})
		}
	}

	// Add scope attribute for scoped components.
	if r.component.Scoped {
		w.Write([]byte{' '})
		io.WriteString(w, ScopeID(r.component.Path))
	}

	if isVoidElement(n.Data) {
		w.Write([]byte{'>'})
		return nil
	}
	w.Write([]byte{'>'})

	// Content: v-text, v-html, or child nodes.
	switch {
	case vTextExpr != "":
		val, err := expr.Eval(strings.TrimSpace(vTextExpr), scope)
		if err != nil {
			return fmt.Errorf("v-text %q: %w", vTextExpr, err)
		}
		io.WriteString(w, stdhtml.EscapeString(valueToString(val)))

	case vHTMLExpr != "":
		val, err := expr.Eval(strings.TrimSpace(vHTMLExpr), scope)
		if err != nil {
			return fmt.Errorf("v-html %q: %w", vHTMLExpr, err)
		}
		io.WriteString(w, valueToString(val))

	default:
		if err := r.renderChildren(w, n, scope); err != nil {
			return err
		}
	}

	// Close tag.
	io.WriteString(w, "</")
	io.WriteString(w, n.Data)
	w.Write([]byte{'>'})
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

// collectSlotDefs scans the direct children of n and returns a map of slot
// definitions. Children that are <template v-slot:name> / <template #name>
// elements become named slot definitions; all other children form the
// "default" slot definition. parentScope is shallow-cloned into each
// SlotDefinition so that slot content is rendered with the caller's bindings.
func collectSlotDefs(n *html.Node, parentScope map[string]any) map[string]*SlotDefinition {
	defs := make(map[string]*SlotDefinition)
	var defaultNodes []*html.Node

	cloneScope := func() map[string]any {
		s := make(map[string]any, len(parentScope))
		for k, v := range parentScope {
			s[k] = v
		}
		return s
	}

	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "template" {
			slotName := ""
			attrVal := ""
			isSlot := false
			for _, attr := range child.Attr {
				if name, ok := parseSlotDirective(attr.Key); ok {
					slotName = name
					attrVal = attr.Val
					isSlot = true
					break
				}
			}
			if isSlot {
				var nodes []*html.Node
				for c := child.FirstChild; c != nil; c = c.NextSibling {
					nodes = append(nodes, c)
				}
				bindingVar, bindings, _ := parseBindingPattern(attrVal)
				defs[slotName] = &SlotDefinition{
					Nodes:       nodes,
					ParentScope: cloneScope(),
					BindingVar:  bindingVar,
					Bindings:    bindings,
				}
				continue
			}
		}
		defaultNodes = append(defaultNodes, child)
	}

	if len(defaultNodes) > 0 {
		defs["default"] = &SlotDefinition{
			Nodes:       defaultNodes,
			ParentScope: cloneScope(),
		}
	}

	return defs
}

// renderComponentElement renders n as a component invocation: props are built
// from the element's attributes, slot definitions are collected from the
// children, and then the child component's template is rendered with those props.
func (r *Renderer) renderComponentElement(w io.Writer, n *html.Node, scope map[string]any, comp *Component) error {
	childScope := make(map[string]any)

	// Look for a v-slot / # directive on the component tag itself.
	var componentSlotName string
	var componentSlotAttrVal string
	var hasComponentSlot bool

	for _, attr := range n.Attr {
		// Directives that have already been consumed or don't apply to components.
		switch attr.Key {
		case "v-if", "v-else-if", "v-else", "v-for",
			"v-pre", "v-once", "v-show", "v-text", "v-html":
			continue
		}

		// Slot directive on the component tag itself — captured, not passed as prop.
		if slotName, ok := parseSlotDirective(attr.Key); ok {
			componentSlotName = slotName
			componentSlotAttrVal = attr.Val
			hasComponentSlot = true
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

	var slotDefs map[string]*SlotDefinition

	if hasComponentSlot {
		// Mixing v-slot on the component tag with <template #name> children is invalid.
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.ElementNode && child.Data == "template" {
				for _, attr := range child.Attr {
					if _, ok := parseSlotDirective(attr.Key); ok {
						return fmt.Errorf("component %q: v-slot on component tag cannot be mixed with named slot templates", n.Data)
					}
				}
			}
		}

		// Collect all children as the named slot (typically "default").
		var nodes []*html.Node
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			nodes = append(nodes, child)
		}
		parentScope := make(map[string]any, len(scope))
		for k, v := range scope {
			parentScope[k] = v
		}
		bindingVar, bindings, _ := parseBindingPattern(componentSlotAttrVal)
		slotDefs = map[string]*SlotDefinition{
			componentSlotName: {
				Nodes:       nodes,
				ParentScope: parentScope,
				BindingVar:  bindingVar,
				Bindings:    bindings,
			},
		}
	} else {
		// Collect slot definitions from children.
		slotDefs = collectSlotDefs(n, scope)
	}

	// Build a child renderer that shares the registry and style collector.
	childRenderer := &Renderer{
		component:          comp,
		styleCollector:     r.styleCollector,
		registry:           r.registry,
		missingPropHandler: r.missingPropHandler,
		slotDefs:           slotDefs,
	}

	if err := childRenderer.Render(w, childScope); err != nil {
		return fmt.Errorf("component %q: %w", n.Data, err)
	}
	return nil
}

// renderVFor renders n repeatedly for each element in the v-for collection.
func (r *Renderer) renderVFor(w io.Writer, n *html.Node, vforExpr string, scope map[string]any) error {
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
			return r.renderChildren(w, n, iterScope)
		}
		return r.renderElement(w, n, iterScope)
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
