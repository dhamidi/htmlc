package htmlc

import (
	stdhtml "html"
	htmltemplate "html/template"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"math"
	pathpkg "path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

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
	// Component is the component that authored this slot content.
	// It is used to stamp the correct scope attribute on slot elements.
	// May be nil when the parent has no component context (rare).
	Component *Component
	// SlotDefs holds the slot definitions that were active in the authoring
	// component at the time this slot content was captured.  When the slot
	// content itself contains <slot /> elements they must be resolved against
	// this map, not against the consuming component's slot definitions.
	SlotDefs map[string]*SlotDefinition
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

// isClientSideDirective reports whether an attribute key is a client-side
// directive that should be stripped from server-side rendered output.
func isClientSideDirective(key string) bool {
	return key == "v-model" ||
		key == "v-on" ||
		strings.HasPrefix(key, "v-on:") ||
		strings.HasPrefix(key, "@")
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

// ErrorOnMissingProp is a MissingPropFunc that aborts rendering with an error
// whenever a prop is missing. Use it to restore strict validation:
//
//	renderer.WithMissingPropHandler(htmlc.ErrorOnMissingProp)
func ErrorOnMissingProp(name string) (any, error) {
	return nil, fmt.Errorf("missing prop %q", name)
}

// Renderer walks a component's parsed template and produces HTML output.
// It is the low-level rendering primitive — most callers should use Engine
// (via RenderPage or RenderFragment) rather than constructing a Renderer
// directly. Use NewRenderer when you need fine-grained control over style
// collection or registry attachment.
//
// Engine-level functions (from Engine.RegisterFunc) are available in child
// components only when the renderer is created by Engine — Engine calls
// WithFuncs automatically. Callers using the low-level NewRenderer API who
// want engine functions to propagate into child components must call WithFuncs
// explicitly.
type Renderer struct {
	component          *Component
	styleCollector     *StyleCollector
	registry           Registry
	nsRegistry         map[string]map[string]*Component // nil = NS resolution disabled
	componentDir       string                           // ComponentDir for NS relative-path computation
	missingPropHandler MissingPropFunc
	slotDefs           map[string]*SlotDefinition
	directives         DirectiveRegistry
	ctx                context.Context // optional; nil means no cancellation
	debug              bool
	debugW             *debugWriter
	funcs              map[string]any // engine-registered functions, propagated to child renderers
	logger             *slog.Logger  // nil = no slog output
	cw                 countingWriter // Reset()ed at each child dispatch
}

// NewRenderer creates a Renderer for c. Call WithStyles and WithComponents
// before Render to enable style collection and component composition. Call
// WithFuncs to make engine-registered functions available in this component and
// all child components rendered from it.
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

// WithNSComponents attaches a namespaced component registry and the engine's
// ComponentDir to this renderer, enabling proximity-based component resolution.
// ns maps forward-slash relative directory paths to local component names to
// parsed components; componentDir is the value of Options.ComponentDir used
// when the engine was created.
//
// When set, resolveComponent walks up the directory tree from the caller
// component's location before falling back to the flat registry.
// Returns the Renderer for chaining.
func (r *Renderer) WithNSComponents(ns map[string]map[string]*Component, componentDir string) *Renderer {
	r.nsRegistry = ns
	r.componentDir = componentDir
	return r
}

// WithMissingPropHandler sets a handler that is called when a prop referenced
// in the template is not present in the render scope. Returns the Renderer for
// chaining.
func (r *Renderer) WithMissingPropHandler(fn MissingPropFunc) *Renderer {
	r.missingPropHandler = fn
	return r
}

// WithDirectives attaches a custom directive registry. Directives registered
// here are invoked when the renderer encounters v-<name> attributes that are
// not built-in directives. Returns the Renderer for chaining.
func (r *Renderer) WithDirectives(dr DirectiveRegistry) *Renderer {
	r.directives = dr
	return r
}

// WithContext attaches a context.Context to the renderer. The render is
// aborted and ctx.Err() is returned if the context is cancelled or its
// deadline is exceeded. Returns the Renderer for chaining.
func (r *Renderer) WithContext(ctx context.Context) *Renderer {
	r.ctx = ctx
	return r
}

// withDebug enables debug render mode on this renderer. dw must wrap the same
// io.Writer that will be passed to Render so that debug comments are
// interleaved with HTML output in the correct order.
func (r *Renderer) withDebug(dw *debugWriter) *Renderer {
	r.debug = true
	r.debugW = dw
	return r
}

// WithFuncs attaches engine-registered functions to this renderer so they are
// available in template expressions and propagated to all child renderers.
// Returns the Renderer for chaining.
func (r *Renderer) WithFuncs(funcs map[string]any) *Renderer {
	r.funcs = funcs
	return r
}

// WithLogger attaches a *slog.Logger to this renderer. When non-nil, one
// structured log record is emitted per child component dispatch. Returns
// the Renderer for chaining.
func (r *Renderer) WithLogger(l *slog.Logger) *Renderer {
	r.logger = l
	return r
}

// locateExpr searches the component's source for the first occurrence of expr
// and returns a SourceLocation pointing to it. Returns nil when the source is
// unavailable or the expression cannot be found.
//
// This is Option A (lightweight, approximate) from the design notes. Option B
// (precise per-node positions built during parseTemplateHTML using the
// tokenizer's byte offsets) is left as a future improvement.
func (r *Renderer) locateExpr(exprStr string) *SourceLocation {
	if r.component == nil || r.component.Source == "" {
		return nil
	}
	idx := strings.Index(r.component.Source, exprStr)
	if idx < 0 {
		return nil
	}
	ln, col := lineCol(r.component.Source, idx)
	return &SourceLocation{
		File:    r.component.Path,
		Line:    ln,
		Column:  col,
		Snippet: snippet(r.component.Source, ln),
	}
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
			// Default: render a visible placeholder so the author can see
			// which prop is missing without having to configure a handler.
			val := fmt.Sprintf("[missing: %s]", p.Name)
			if augmented == nil {
				augmented = make(map[string]any, len(scope)+1)
				for k, v := range scope {
					augmented[k] = v
				}
			}
			augmented[p.Name] = val
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
		// <style> and <script> are raw text elements — browsers never
		// HTML-decode their content, so we must not HTML-escape it.
		if n.Parent != nil && isRawTextElement(n.Parent.Data) {
			io.WriteString(w, n.Data)
			return nil
		}
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
// isRawTextElement reports whether tag is an HTML raw text element whose
// content browsers never HTML-decode (<style>, <script>, <noscript>).
func isRawTextElement(tag string) bool {
	return tag == "style" || tag == "script" || tag == "noscript"
}

// Literal segments are HTML-escaped; {{ expr }} segments are evaluated and escaped.
// If a value is of type html/template.HTML it is emitted verbatim (already safe),
// matching the behaviour of v-html for pre-escaped content.
func (r *Renderer) interpolate(w io.Writer, text string, scope map[string]any) error {
	lastEnd := 0

	for _, loc := range mustacheRe.FindAllStringSubmatchIndex(text, -1) {
		// Write literal text before this match, HTML-escaped.
		io.WriteString(w, stdhtml.EscapeString(text[lastEnd:loc[0]]))

		// Evaluate the captured expression (loc[2]:loc[3]).
		exprSrc := strings.TrimSpace(text[loc[2]:loc[3]])
		val, err := expr.Eval(exprSrc, scope)
		if err != nil {
			return &RenderError{
				Component: r.component.Path,
				Expr:      exprSrc,
				Wrapped:   err,
				Location:  r.locateExpr(exprSrc),
			}
		}
		// html/template.HTML values are already safe — emit verbatim.
		if r.debug {
			r.debugW.exprValue(exprSrc, val)
		}
		if safe, ok := val.(htmltemplate.HTML); ok {
			io.WriteString(w, string(safe))
		} else {
			io.WriteString(w, stdhtml.EscapeString(valueToString(val)))
		}

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

// checkContext returns ctx.Err() if the context has been cancelled, nil
// otherwise. It is a no-op when no context is set.
func (r *Renderer) checkContext() error {
	if r.ctx == nil {
		return nil
	}
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	default:
		return nil
	}
}

// renderChildren iterates the children of parent and renders them, handling
// v-if/v-else-if/v-else chains and v-for directives.
func (r *Renderer) renderChildren(w io.Writer, parent *html.Node, scope map[string]any) error {
	if err := r.checkContext(); err != nil {
		return err
	}
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
			// Intercept <template v-switch> at the child level.
			if child.Data == "template" {
				if switchExpr, ok := attrValue(child, "v-switch"); ok {
					if err := r.renderSwitchBlock(w, child, switchExpr, scope); err != nil {
						return err
					}
					child = child.NextSibling
					continue
				}
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
			if r.debug && !b.isElse {
				r.debugW.vifSkipped(b.expr, false)
			}
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
		if n.Parent != nil && isRawTextElement(n.Parent.Data) {
			io.WriteString(w, n.Data)
		} else {
			io.WriteString(w, stdhtml.EscapeString(n.Data))
		}

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

// shallowCloneNode returns a new *html.Node with the same Type, Data, and a
// copy of Attr. It does NOT copy parent/child/sibling pointers; the clone is
// used as a mutable scratch buffer by directive Created hooks and is never
// inserted into the document tree.
func shallowCloneNode(n *html.Node) *html.Node {
	clone := &html.Node{
		Type:      n.Type,
		DataAtom:  n.DataAtom,
		Data:      n.Data,
		Namespace: n.Namespace,
		Attr:      make([]html.Attribute, len(n.Attr)),
	}
	copy(clone.Attr, n.Attr)
	return clone
}

// removeAttr returns a shallow clone of n with the named attribute removed.
// The clone's FirstChild is linked to n.FirstChild so children are accessible.
func removeAttr(n *html.Node, key string) *html.Node {
	clone := shallowCloneNode(n)
	clone.FirstChild = n.FirstChild
	filtered := clone.Attr[:0]
	for _, a := range clone.Attr {
		if a.Key != key {
			filtered = append(filtered, a)
		}
	}
	clone.Attr = filtered
	return clone
}

// renderSwitchBlock evaluates a v-switch expression on switchNode and renders
// the first matching v-case child, or the first v-default child if none match.
func (r *Renderer) renderSwitchBlock(w io.Writer, switchNode *html.Node, switchExpr string, scope map[string]any) error {
	switchVal, err := expr.Eval(strings.TrimSpace(switchExpr), scope)
	if err != nil {
		return fmt.Errorf("v-switch %q: %w", switchExpr, err)
	}

	matched := false
	for child := switchNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		if caseExpr, ok := attrValue(child, "v-case"); ok {
			if matched {
				continue
			}
			caseVal, err := expr.Eval(strings.TrimSpace(caseExpr), scope)
			if err != nil {
				return fmt.Errorf("v-case %q: %w", caseExpr, err)
			}
			if switchVal == caseVal {
				matched = true
				if err := r.renderNode(w, removeAttr(child, "v-case"), scope); err != nil {
					return err
				}
			}
		} else if _, ok := attrValue(child, "v-default"); ok {
			if !matched {
				matched = true
				if err := r.renderNode(w, removeAttr(child, "v-default"), scope); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// invokedDirective records a directive that was applied during Created hooks
// so its Mounted hook can be called after the element's closing tag.
type invokedDirective struct {
	dir     Directive
	binding DirectiveBinding
	name    string
}

// applyCreatedHooks scans node.Attr for custom directive keys, evaluates their
// expressions, calls Created on each matching Directive, and removes the
// directive attribute from node.Attr afterwards. Returns the list of invoked
// directives so their Mounted hooks can be called later.
func (r *Renderer) applyCreatedHooks(node *html.Node, scope map[string]any, ctx DirectiveContext) ([]invokedDirective, error) {
	builtinNames := map[string]bool{
		"text": true, "html": true, "show": true, "once": true, "model": true,
		"if": true, "else-if": true, "else": true, "for": true, "pre": true,
		"slot": true, "bind": true, "on": true,
		"switch": true, "case": true, "default": true,
	}

	// pendingDir holds a matched custom directive waiting to be Called.
	type pendingDir struct {
		dir     Directive
		binding DirectiveBinding
		name    string
	}

	// First pass: separate kept (non-custom-directive) attrs from pending hooks.
	var kept []html.Attribute
	var pending []pendingDir
	for _, attr := range node.Attr {
		name, arg, mods := parseDirectiveKey(attr.Key)
		if name == "" || builtinNames[name] {
			kept = append(kept, attr)
			continue
		}
		dir, ok := r.directives[name]
		if !ok {
			// Unknown custom directive — pass through as a regular attribute.
			kept = append(kept, attr)
			continue
		}
		val, _ := expr.Eval(strings.TrimSpace(attr.Val), scope) // best-effort; nil on error
		pending = append(pending, pendingDir{
			dir: dir,
			binding: DirectiveBinding{
				Value:     val,
				RawExpr:   attr.Val,
				Arg:       arg,
				Modifiers: mods,
			},
			name: name,
		})
	}

	// Set node.Attr to kept attrs before calling hooks so that hook mutations
	// (appending to node.Attr) are not overwritten afterwards.
	node.Attr = kept

	// Second pass: call Created hooks.
	var invoked []invokedDirective
	for _, pd := range pending {
		if err := pd.dir.Created(node, pd.binding, ctx); err != nil {
			return nil, fmt.Errorf("directive v-%s Created: %w", pd.name, err)
		}
		invoked = append(invoked, invokedDirective{dir: pd.dir, binding: pd.binding, name: pd.name})
	}
	return invoked, nil
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
	// so this branch only fires for plain <template> elements (or v-switch).
	if n.Data == "template" {
		if switchExpr, ok := attrValue(n, "v-switch"); ok {
			return r.renderSwitchBlock(w, n, switchExpr, scope)
		}
		return r.renderChildren(w, n, scope)
	}

	// v-switch is only valid on <template> elements.
	if _, ok := attrValue(n, "v-switch"); ok {
		return fmt.Errorf("v-switch is only valid on <template> elements, got <%s>", n.Data)
	}

	// <slot>: emit slot content from the caller's SlotDefinition, or fallback children.
	if n.Data == "slot" {
		slotName := "default"
		if nameAttr, ok := attrValue(n, "name"); ok {
			slotName = nameAttr
		}
		if def, ok := r.slotDefs[slotName]; ok {
			if r.debug {
				r.debugW.slotStart(slotName, len(def.Nodes))
			}
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

			// slotRenderer uses the authoring component for correct scope attribute stamping.
			// Use the authoring component's slot definitions so that any <slot /> elements
			// inside the slot content are resolved against the authoring context, not the
			// consuming component's context (which would cause infinite recursion).
			slotRenderer := r.rendererWithComponent(def.Component)
			slotRenderer.slotDefs = def.SlotDefs

			nodes := def.Nodes
			for i := 0; i < len(nodes); {
				node := nodes[i]
				if node.Type == html.ElementNode {
					if vforExpr, ok := attrValue(node, "v-for"); ok {
						if err := slotRenderer.renderVFor(w, node, vforExpr, renderScope); err != nil {
							return err
						}
						i++
						continue
					}
					switch conditionalDirective(node) {
					case "v-if":
						lastInChain, err := slotRenderer.renderConditionalChain(w, node, renderScope)
						if err != nil {
							return err
						}
						// Advance past all nodes consumed by the chain.
						for i < len(nodes) && nodes[i] != lastInChain {
							i++
						}
						i++
						continue
					case "v-else-if":
						return fmt.Errorf("v-else-if without preceding v-if or v-else-if")
					case "v-else":
						return fmt.Errorf("v-else without preceding v-if or v-else-if")
					}
				}
				if err := slotRenderer.renderNode(w, node, renderScope); err != nil {
					return err
				}
				i++
			}
			if r.debug {
				r.debugW.slotEnd(slotName)
			}
			return nil
		}
		// No slot definition: render fallback children (if any).
		return r.renderChildren(w, n, scope)
	}

	// --- custom directive Created hooks ---
	// Clone the node so hooks can mutate tag and attrs without touching the
	// shared parsed template AST. Link the original children so that
	// renderComponentElement can traverse them for slot definitions.
	working := shallowCloneNode(n)
	working.FirstChild = n.FirstChild

	// Pre-render children into a buffer so directives can inspect the fully
	// evaluated inner HTML (template expressions resolved, components expanded).
	// Skip pre-rendering for:
	//   - void elements (no children)
	//   - elements using v-text/v-html (content comes from expressions)
	//   - component elements and dynamic <component :is="..."> (children are
	//     slot content; their scope is set up by the component, not here)
	// preRendered tracks whether childBuf was populated; if false the default
	// content branch falls back to calling renderChildren directly.
	var childBuf bytes.Buffer
	var preRendered bool
	_, hasVText := attrValue(n, "v-text")
	_, hasVHTML := attrValue(n, "v-html")
	isComponentElement := n.Data == "component" || r.resolveComponent(n.Data) != nil
	if !isVoidElement(n.Data) && !hasVText && !hasVHTML && !isComponentElement {
		if err := r.renderChildren(&childBuf, n, scope); err != nil {
			return err
		}
		preRendered = true
	}
	renderedChildHTML := childBuf.String()

	ctx := DirectiveContext{
		Registry:          r.registry,
		RenderedChildHTML: renderedChildHTML,
	}

	var invoked []invokedDirective
	if len(r.directives) > 0 {
		var err error
		invoked, err = r.applyCreatedHooks(working, scope, ctx)
		if err != nil {
			return err
		}
	}

	// --- <component is="..."> or <component :is="..."> ---
	if working.Data == "component" {
		isVal, isFound, isDynamic := "", false, false
		var keptAttrs []html.Attribute
		for _, attr := range working.Attr {
			switch attr.Key {
			case ":is", "v-bind:is":
				isVal = attr.Val
				isFound = true
				isDynamic = true
			case "is":
				isVal = attr.Val
				isFound = true
				isDynamic = false
			default:
				keptAttrs = append(keptAttrs, attr)
			}
		}
		if !isFound {
			return fmt.Errorf("<component>: :is or is attribute is required")
		}
		var compName string
		if isDynamic {
			val, err := expr.Eval(strings.TrimSpace(isVal), scope)
			if err != nil {
				return fmt.Errorf("<component> :is %q: %w", isVal, err)
			}
			var ok bool
			compName, ok = val.(string)
			if !ok || compName == "" {
				return fmt.Errorf("<component> :is: expected non-empty string, got %T", val)
			}
		} else {
			compName = isVal
		}
		working.Attr = keptAttrs

		// Path-based reference: if compName contains "/" treat it as a
		// directory-qualified path for direct NS registry lookup.
		if strings.Contains(compName, "/") {
			// Strip a leading "/" for root-relative addressing.
			stripped := strings.TrimPrefix(compName, "/")
			parts := strings.Split(stripped, "/")
			localName := parts[len(parts)-1]
			dirPart := pathpkg.Join(parts[:len(parts)-1]...)

			if r.nsRegistry == nil {
				return fmt.Errorf("<component is=%q>: path-based references require a namespaced registry", compName)
			}
			dirMap, ok := r.nsRegistry[dirPart]
			if !ok {
				return fmt.Errorf("<component is=%q>: no components registered in directory %q", compName, dirPart)
			}
			comp, ok := dirMap[localName]
			if !ok {
				return fmt.Errorf("<component is=%q>: component %q not found in directory %q", compName, localName, dirPart)
			}
			return r.renderComponentElement(w, working, scope, comp)
		}

		working.Data = compName
		// Fall through to resolveComponent / native element logic below.
	}

	// Component: resolve the tag name against the registry.
	if comp := r.resolveComponent(working.Data); comp != nil {
		return r.renderComponentElement(w, working, scope, comp)
	}
	// Unknown component-like tag (kebab-case with hyphen, not in registry).
	if r.registry != nil && isComponentLike(working.Data) {
		return fmt.Errorf("unknown component: %q", working.Data)
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

	for _, attr := range working.Attr {
		switch attr.Key {
		case "v-text":
			vTextExpr = attr.Val
		case "v-html":
			vHTMLExpr = attr.Val
		case "v-show":
			vShowExpr = attr.Val
		case "v-once":
			// server-side: render normally; consume directive, don't emit
		case "v-model":
			// strip: no meaning in server-side rendering
		case "v-bind":
			// Argument-less v-bind: spread a map of attributes onto the element.
			val, err := expr.Eval(strings.TrimSpace(attr.Val), scope)
			if err != nil {
				return fmt.Errorf("v-bind %q: %w", attr.Val, err)
			}
			if err := applyAttrSpread(val, &dynClassParts, &dynStyleParts, &dynAttrs); err != nil {
				return err
			}
		case "v-if", "v-else-if", "v-else", "v-for":
			// consumed by directives; not emitted as attributes
		case "class":
			staticClass = attr.Val
		case "style":
			staticStyle = attr.Val
		default:
			if isClientSideDirective(attr.Key) {
				continue
			}
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
	io.WriteString(w, working.Data)

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

	if isVoidElement(working.Data) {
		w.Write([]byte{'>'})
		// Mounted hooks (void element has no children, but Mounted still fires).
		for _, inv := range invoked {
			if err := inv.dir.Mounted(w, working, inv.binding, ctx); err != nil {
				return fmt.Errorf("directive v-%s Mounted: %w", inv.name, err)
			}
		}
		return nil
	}
	w.Write([]byte{'>'})

	// Check if any invoked directive wants to replace the element's inner HTML.
	var replacementHTML string
	for _, inv := range invoked {
		if dc, ok := inv.dir.(DirectiveWithContent); ok {
			if h, hasContent := dc.InnerHTML(); hasContent {
				replacementHTML = h
				break
			}
		}
	}

	// Content: directive replacement, v-text, v-html, or child nodes.
	switch {
	case replacementHTML != "":
		io.WriteString(w, replacementHTML)

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
		if preRendered {
			// Children were already rendered into renderedChildHTML above.
			io.WriteString(w, renderedChildHTML)
		} else {
			if err := r.renderChildren(w, n, scope); err != nil {
				return err
			}
		}
	}

	// Close tag.
	io.WriteString(w, "</")
	io.WriteString(w, working.Data)
	w.Write([]byte{'>'})

	// Mounted hooks: called after the closing tag has been written.
	for _, inv := range invoked {
		if err := inv.dir.Mounted(w, working, inv.binding, ctx); err != nil {
			return fmt.Errorf("directive v-%s Mounted: %w", inv.name, err)
		}
	}
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

// resolveComponent looks up tagName using proximity-based resolution (when the
// NS registry is available) followed by flat-registry fallback.
//
// Proximity walk strategy (NS registry):
//  1. Start at the caller component's directory (callerDir).
//  2. For each directory level, try name variants: exact, capitalised,
//     kebab-to-Pascal, case-insensitive.
//  3. Walk toward the root until a match is found or the root is exhausted.
//
// Flat registry fallback (same four strategies, no directory walk).
func (r *Renderer) resolveComponent(tagName string) *Component {
	// Proximity walk using the NS registry.
	if r.nsRegistry != nil {
		callerDir := r.callerDir()
		if c := resolveInNSRegistry(r.nsRegistry, callerDir, tagName); c != nil {
			return c
		}
	}

	// Flat registry fallback.
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

// callerDir returns the forward-slash relative directory of the current
// component with respect to componentDir. Returns "" when the component is at
// the root level or has no path.
func (r *Renderer) callerDir() string {
	if r.component == nil || r.component.Path == "" {
		return ""
	}
	return nsRelDir(r.component.Path, r.componentDir)
}

// resolveInNSRegistry performs the proximity walk over ns starting at
// callerDir, trying each name variant at every directory level.
// Returns nil when no match is found.
func resolveInNSRegistry(ns map[string]map[string]*Component, callerDir, tagName string) *Component {
	d := callerDir
	for {
		if c := lookupNSDir(ns, d, tagName); c != nil {
			return c
		}
		if d == "" {
			break
		}
		// Move one level toward the root using the path package (forward slashes).
		parent := pathpkg.Dir(d)
		if parent == "." || parent == d {
			d = ""
		} else {
			d = parent
		}
	}
	return nil
}

// lookupNSDir looks up tagName in the namespace directory dir using the four
// standard name-folding strategies: exact, capitalised, kebab-to-Pascal, and
// case-insensitive scan.
func lookupNSDir(ns map[string]map[string]*Component, dir, tagName string) *Component {
	dirMap, ok := ns[dir]
	if !ok {
		return nil
	}
	// 1. Exact match.
	if c, ok := dirMap[tagName]; ok {
		return c
	}
	// 2. First letter capitalised.
	if len(tagName) > 0 {
		cap := strings.ToUpper(tagName[:1]) + tagName[1:]
		if cap != tagName {
			if c, ok := dirMap[cap]; ok {
				return c
			}
		}
	}
	// 3. kebab-case to PascalCase.
	if strings.Contains(tagName, "-") {
		pascal := kebabToPascal(tagName)
		if c, ok := dirMap[pascal]; ok {
			return c
		}
	}
	// 4. Case-insensitive scan.
	lower := strings.ToLower(tagName)
	for k, c := range dirMap {
		if strings.ToLower(k) == lower {
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
// parentComp is the component that authored the slot content and is stored in
// each SlotDefinition for correct scope attribute stamping.
func collectSlotDefs(n *html.Node, parentScope map[string]any, parentComp *Component, parentSlotDefs map[string]*SlotDefinition) map[string]*SlotDefinition {
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
					Component:   parentComp,
					SlotDefs:    parentSlotDefs,
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
			Component:   parentComp,
			SlotDefs:    parentSlotDefs,
		}
	}

	return defs
}

// renderComponentElement renders n as a component invocation: props are built
// from the element's attributes, slot definitions are collected from the
// children, and then the child component's template is rendered with those props.
func (r *Renderer) renderComponentElement(w io.Writer, n *html.Node, scope map[string]any, comp *Component) error {
	if r.debug {
		r.debugW.componentStart(n.Data, comp.Path)
		defer r.debugW.componentEnd(n.Data)
	}
	childScope := make(map[string]any)

	// Phase 1: Apply v-bind spread maps (lower priority).
	for _, attr := range n.Attr {
		if attr.Key != "v-bind" {
			continue
		}
		val, err := expr.Eval(strings.TrimSpace(attr.Val), scope)
		if err != nil {
			return fmt.Errorf("v-bind %q: %w", attr.Val, err)
		}
		if m, ok := toStringMap(val); ok {
			for k, v := range m {
				childScope[k] = v
			}
		} else if val != nil {
			return fmt.Errorf("v-bind on component %q: expected map, got %T", n.Data, val)
		}
	}

	// Look for a v-slot / # directive on the component tag itself.
	var componentSlotName string
	var componentSlotAttrVal string
	var hasComponentSlot bool

	for _, attr := range n.Attr {
		// Directives that have already been consumed or don't apply to components.
		switch attr.Key {
		case "v-if", "v-else-if", "v-else", "v-for",
			"v-pre", "v-once", "v-show", "v-text", "v-html",
			"v-model", "v-bind", "v-switch", "v-case", "v-default":
			continue
		}
		if isClientSideDirective(attr.Key) {
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
				Component:   r.component,
				SlotDefs:    r.slotDefs,
			},
		}
	} else {
		// Collect slot definitions from children.
		slotDefs = collectSlotDefs(n, scope, r.component, r.slotDefs)
	}

	// Apply engine funcs as lower-priority values (explicit props win over funcs).
	if len(r.funcs) > 0 {
		merged := make(map[string]any, len(r.funcs)+len(childScope))
		for k, v := range r.funcs {
			merged[k] = v
		}
		for k, v := range childScope {
			merged[k] = v // explicit props override engine funcs
		}
		childScope = merged
	}

	// Build a child renderer that shares the registry and style collector.
	childRenderer := &Renderer{
		component:          comp,
		styleCollector:     r.styleCollector,
		registry:           r.registry,
		nsRegistry:         r.nsRegistry,        // propagate NS registry to child renderers
		componentDir:       r.componentDir,       // propagate componentDir to child renderers
		missingPropHandler: r.missingPropHandler,
		slotDefs:           slotDefs,
		directives:         r.directives,
		ctx:                r.ctx,
		debug:              r.debug,
		debugW:             r.debugW,
		funcs:              r.funcs,              // propagate engine functions to child renderers
		logger:             r.logger,
	}

	if r.logger == nil {
		if err := childRenderer.Render(w, childScope); err != nil {
			return fmt.Errorf("component %q: %w", n.Data, err)
		}
		return nil
	}

	compName := strings.TrimSuffix(pathpkg.Base(comp.Path), pathpkg.Ext(comp.Path))
	childRenderer.cw.Reset(w)
	start := time.Now()
	renderErr := childRenderer.Render(&childRenderer.cw, childScope)
	elapsed := time.Since(start)
	if renderErr != nil {
		r.logger.ErrorContext(r.ctx, MsgComponentFailed,
			slog.String("component", compName),
			slog.Duration("duration", elapsed),
			slog.Int64("bytes", childRenderer.cw.n),
			slog.Any("error", renderErr),
		)
		return fmt.Errorf("component %q: %w", n.Data, renderErr)
	}
	r.logger.DebugContext(r.ctx, MsgComponentRendered,
		slog.String("component", compName),
		slog.Duration("duration", elapsed),
		slog.Int64("bytes", childRenderer.cw.n),
	)
	return nil
}

// rendererWithComponent returns a shallow copy of r with component replaced.
// All other fields (registry, styleCollector, directives, …) are shared.
// Passing nil leaves the component unchanged (no-op).
func (r *Renderer) rendererWithComponent(comp *Component) *Renderer {
	copy := *r
	if comp != nil {
		copy.component = comp
	}
	return &copy
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
// applyAttrSpread applies the entries of a map value (from v-bind="obj")
// to the accumulator slices used when building element attributes.
// It applies the same per-key logic as individual :attr bindings.
func applyAttrSpread(
	val any,
	dynClassParts *[]string,
	dynStyleParts *[]string,
	dynAttrs *[]outAttr,
) error {
	m, ok := toStringMap(val)
	if !ok {
		return fmt.Errorf("v-bind: expected map, got %T", val)
	}
	// Sort keys for deterministic output.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := m[k]
		switch k {
		case "class":
			s, err := resolveClass(v)
			if err != nil {
				return fmt.Errorf("v-bind class: %w", err)
			}
			*dynClassParts = append(*dynClassParts, s)
		case "style":
			s, err := resolveStyle(v)
			if err != nil {
				return fmt.Errorf("v-bind style: %w", err)
			}
			*dynStyleParts = append(*dynStyleParts, s)
		default:
			if isBooleanAttr(k) {
				if expr.IsTruthy(v) {
					*dynAttrs = append(*dynAttrs, outAttr{key: k, boolOnly: true})
				}
			} else {
				*dynAttrs = append(*dynAttrs, outAttr{key: k, val: valueToString(v)})
			}
		}
	}
	return nil
}

// toStringMap converts a value to map[string]any if possible.
// A nil value is treated as an empty map (no-op spread).
func toStringMap(val any) (map[string]any, bool) {
	if val == nil {
		return nil, true // nil spread is a no-op; treat as ok
	}
	m, ok := val.(map[string]any)
	return m, ok
}

// Values that implement fmt.Stringer are converted via their String() method.
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
		// Honor fmt.Stringer: call String() for types that implement it.
		if s, ok := v.(fmt.Stringer); ok {
			return s.String()
		}
		return fmt.Sprintf("%v", v)
	}
}
