package htmlc

import (
	stdhtml "html"
	htmltemplate "html/template"
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template/parse"

	xhtml "golang.org/x/net/html"
)

// simpleExprRe matches a simple identifier or dotted-path expression, e.g.
// "foo", "foo.bar", "foo.bar.baz". These are the only htmlc expressions that
// can be translated directly to html/template field-access syntax.
var simpleExprRe = regexp.MustCompile(`^[a-zA-Z_$][a-zA-Z0-9_$]*(\.[a-zA-Z_$][a-zA-Z0-9_$]*)*$`)

// translateExpr translates an htmlc expression string to an html/template
// field reference. Simple identifiers and dotted paths are prefixed with ".";
// all other expressions are unsupported and return an error.
//
//	"foo"       → ".foo"
//	"foo.bar"   → ".foo.bar"
//	"foo + bar" → error
func translateExpr(e string) (string, error) {
	e = strings.TrimSpace(e)
	if simpleExprRe.MatchString(e) {
		return "." + e, nil
	}
	return "", fmt.Errorf(
		"cannot translate expression %q to html/template: "+
			"only simple identifiers and dot-path expressions are supported", e)
}

// tmplInterpolationRe matches {{ ... }} mustache expressions inside text nodes.
var tmplInterpolationRe = regexp.MustCompile(`\{\{(.*?)\}\}`)

// translateTextContent rewrites {{ expr }} mustache interpolations in a text
// node to html/template {{ .expr }} form. Returns an error for any expression
// that cannot be translated.
func translateTextContent(text string) (string, error) {
	var sb strings.Builder
	lastEnd := 0
	for _, m := range tmplInterpolationRe.FindAllStringSubmatchIndex(text, -1) {
		sb.WriteString(text[lastEnd:m[0]])
		exprStr := strings.TrimSpace(text[m[2]:m[3]])
		translated, err := translateExpr(exprStr)
		if err != nil {
			return "", fmt.Errorf("text interpolation %q: %w", exprStr, err)
		}
		sb.WriteString("{{")
		sb.WriteString(translated)
		sb.WriteString("}}")
		lastEnd = m[1]
	}
	sb.WriteString(text[lastEnd:])
	return sb.String(), nil
}

// compileContext carries shared mutable state through a template source
// compilation pass.
type compileContext struct {
	// entries is a snapshot of the engine registry used to resolve sub-components.
	entries map[string]*engineEntry
	// visited tracks component names whose {{ define }} blocks have already been
	// emitted, preventing infinite recursion in circular component graphs.
	visited map[string]bool
	// defines accumulates {{ define "Name" }}...{{ end }} blocks for reachable
	// sub-components. The caller appends this to the root component's source.
	defines strings.Builder
}

// ExportTemplateSource compiles the named component (and all sub-components
// statically reachable from it) to a self-contained html/template source
// string. The output contains one {{ define "Name" }}...{{ end }} block per
// component, suitable for use with [html/template.New.Parse] or
// [html/template.ParseFiles].
//
// Only htmlc constructs with unambiguous html/template equivalents are
// translated. The supported set is:
//   - Text interpolation: {{ identifier }} and {{ identifier.path }}
//   - v-bind:attr / :attr with simple-identifier or dot-path expressions
//   - v-if / v-else-if / v-else with simple-identifier conditions
//   - v-for="item in collection" (simple collection expression only)
//   - <slot> / <slot name="…"> → {{ block "name" . }}…{{ end }}
//   - <ComponentName> → {{ template "ComponentName" . }}
//   - Static HTML attributes and regular HTML elements
//
// Unsupported features (v-show, v-html, v-text, v-bind spread, v-switch,
// complex expressions, custom directives) return an error so that callers
// are alerted rather than silently receiving incomplete output.
//
// ExportTemplateSource returns an error when the named component is not
// registered or is a synthetic component (imported via ImportTemplate).
func (e *Engine) ExportTemplateSource(name string) (string, error) {
	e.mu.RLock()
	entry, ok := e.entries[name]
	entries := make(map[string]*engineEntry, len(e.entries))
	for k, v := range e.entries {
		entries[k] = v
	}
	e.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("engine: unknown component %q: %w", name, ErrComponentNotFound)
	}
	if entry.renderFn != nil {
		return "", fmt.Errorf("engine: component %q is a synthetic (imported) component and cannot be exported to template source", name)
	}

	ctx := &compileContext{
		entries: entries,
		visited: make(map[string]bool),
	}
	rootBlock, err := compileEntryToDefineBlock(name, entry, ctx)
	if err != nil {
		return "", err
	}
	return rootBlock + ctx.defines.String(), nil
}

// ExportTemplate compiles the named component tree to an
// [*html/template.Template] by calling ExportTemplateSource and parsing the
// result. The returned template can be executed directly with stdlib
// [html/template.Template.Execute].
func (e *Engine) ExportTemplate(name string) (*htmltemplate.Template, error) {
	src, err := e.ExportTemplateSource(name)
	if err != nil {
		return nil, err
	}
	return htmltemplate.New(name).Parse(src)
}

// ImportTemplate wraps t as a virtual htmlc component registered under
// t.Name(). Named templates defined within t (via {{ define "N" }}) are
// registered as separate virtual components under "N", making them available
// as <N> tags within any .vue component tree.
//
// ImportTemplate does not write any file to disk.
//
// It returns an error if any template name in t.Templates() is already
// registered in the engine. Use ForceImportTemplate to overwrite existing
// registrations.
func (e *Engine) ImportTemplate(t *htmltemplate.Template) error {
	return e.importTemplate(t, false)
}

// ForceImportTemplate is like ImportTemplate but silently overwrites any
// existing component registrations that conflict with the template's names.
func (e *Engine) ForceImportTemplate(t *htmltemplate.Template) error {
	return e.importTemplate(t, true)
}

func (e *Engine) importTemplate(t *htmltemplate.Template, force bool) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, tmpl := range t.Templates() {
		if tmpl.Name() == "" {
			continue
		}
		if !force {
			if _, exists := e.entries[tmpl.Name()]; exists {
				return fmt.Errorf(
					"engine: component %q is already registered; use ForceImportTemplate to overwrite",
					tmpl.Name())
			}
		}
		captured := tmpl // capture for closure
		props := propsFromTemplate(captured)
		e.entries[tmpl.Name()] = &engineEntry{
			path:     "",
			comp:     nil,
			renderFn: func(w io.Writer, data map[string]any) error {
				return captured.Execute(w, data)
			},
			props: props,
		}
	}
	return nil
}

// propsFromTemplate walks the parse tree of t and returns a PropInfo slice
// for each distinct top-level field reference (.field or .field.subfield)
// found across all named templates in t.
//
// This uses the exported parse tree from [text/template/parse], which the
// standard library makes available via [html/template.Template.Tree].
func propsFromTemplate(t *htmltemplate.Template) []PropInfo {
	props := map[string]*PropInfo{}
	for _, tmpl := range t.Templates() {
		if tmpl.Tree != nil {
			collectParseTreeProps(tmpl.Tree.Root, props)
		}
	}
	result := make([]PropInfo, 0, len(props))
	for _, p := range props {
		result = append(result, *p)
	}
	return result
}

// collectParseTreeProps recursively walks a text/template/parse AST node and
// records all top-level field accesses (.identifier) in props. Only the first
// identifier in a chain is recorded as the prop name (e.g. ".user.name"
// contributes "user", not "name").
func collectParseTreeProps(node parse.Node, props map[string]*PropInfo) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *parse.FieldNode:
		if len(n.Ident) > 0 {
			name := n.Ident[0]
			if _, ok := props[name]; !ok {
				props[name] = &PropInfo{Name: name}
			}
		}
	case *parse.ListNode:
		for _, child := range n.Nodes {
			collectParseTreeProps(child, props)
		}
	case *parse.ActionNode:
		collectParseTreeProps(n.Pipe, props)
	case *parse.PipeNode:
		for _, cmd := range n.Cmds {
			collectParseTreeProps(cmd, props)
		}
		for _, decl := range n.Decl {
			collectParseTreeProps(decl, props)
		}
	case *parse.CommandNode:
		for _, arg := range n.Args {
			collectParseTreeProps(arg, props)
		}
	case *parse.IfNode:
		collectParseTreeProps(n.Pipe, props)
		collectParseTreeProps(n.List, props)
		collectParseTreeProps(n.ElseList, props)
	case *parse.RangeNode:
		collectParseTreeProps(n.Pipe, props)
		collectParseTreeProps(n.List, props)
		collectParseTreeProps(n.ElseList, props)
	case *parse.WithNode:
		collectParseTreeProps(n.Pipe, props)
		collectParseTreeProps(n.List, props)
		collectParseTreeProps(n.ElseList, props)
	case *parse.TemplateNode:
		collectParseTreeProps(n.Pipe, props)
	}
}

// compileEntryToDefineBlock compiles a single component to a
// {{ define "Name" }}...{{ end }} source block. Sub-component definitions are
// queued in ctx.defines.
func compileEntryToDefineBlock(name string, entry *engineEntry, ctx *compileContext) (string, error) {
	if ctx.visited[name] {
		return "", nil
	}
	ctx.visited[name] = true

	var body strings.Builder
	if err := compileChildren(&body, entry.comp.Template, ctx); err != nil {
		return "", fmt.Errorf("component %q: %w", name, err)
	}
	return fmt.Sprintf("{{ define %q }}\n%s\n{{ end }}\n", name, body.String()), nil
}

// compileChildren writes the html/template source for all direct children of
// parent into sb. It handles v-if/else-if/else chains, v-for loops, and
// regular element and text nodes.
func compileChildren(sb *strings.Builder, parent *xhtml.Node, ctx *compileContext) error {
	child := parent.FirstChild
	for child != nil {
		switch child.Type {
		case xhtml.DoctypeNode:
			sb.WriteString("<!DOCTYPE html>")
			child = child.NextSibling

		case xhtml.TextNode:
			translated, err := translateTextContent(child.Data)
			if err != nil {
				return err
			}
			sb.WriteString(translated)
			child = child.NextSibling

		case xhtml.CommentNode:
			fmt.Fprintf(sb, "<!--%s-->", child.Data)
			child = child.NextSibling

		case xhtml.ElementNode:
			// v-for takes precedence over conditionals (same order as renderer).
			if vforExpr, ok := attrValue(child, "v-for"); ok {
				if err := compileVFor(sb, child, vforExpr, ctx); err != nil {
					return err
				}
				child = child.NextSibling
				continue
			}

			// v-if starts a conditional chain; siblings with v-else-if / v-else
			// are consumed as part of the same chain.
			if _, ok := attrValue(child, "v-if"); ok {
				last, err := compileConditionalChain(sb, child, ctx)
				if err != nil {
					return err
				}
				child = last.NextSibling
				continue
			}

			// Orphaned v-else-if / v-else are errors.
			if _, ok := attrValue(child, "v-else-if"); ok {
				return fmt.Errorf("v-else-if without preceding v-if")
			}
			if _, ok := attrValue(child, "v-else"); ok {
				return fmt.Errorf("v-else without preceding v-if")
			}

			if err := compileElementNode(sb, child, ctx); err != nil {
				return err
			}
			child = child.NextSibling

		default:
			child = child.NextSibling
		}
	}
	return nil
}

// compileConditionalChain emits a {{ if }}...{{ else if }}...{{ else }}...{{ end }}
// block for the v-if element and any immediately following v-else-if / v-else
// siblings. Returns the last node consumed.
func compileConditionalChain(sb *strings.Builder, vIfNode *xhtml.Node, ctx *compileContext) (*xhtml.Node, error) {
	ifExpr, _ := attrValue(vIfNode, "v-if")
	translated, err := translateExpr(ifExpr)
	if err != nil {
		return nil, fmt.Errorf("v-if %q: %w", ifExpr, err)
	}
	fmt.Fprintf(sb, "{{ if %s }}", translated)
	if err := compileElementBody(sb, vIfNode, ctx); err != nil {
		return nil, err
	}

	lastNode := vIfNode
	for {
		next := nextSignificantSibling(lastNode)
		if next == nil {
			break
		}
		if elseIfExpr, ok := attrValue(next, "v-else-if"); ok {
			translated, err := translateExpr(elseIfExpr)
			if err != nil {
				return nil, fmt.Errorf("v-else-if %q: %w", elseIfExpr, err)
			}
			fmt.Fprintf(sb, "{{ else if %s }}", translated)
			if err := compileElementBody(sb, next, ctx); err != nil {
				return nil, err
			}
			lastNode = next
		} else if _, ok := attrValue(next, "v-else"); ok {
			sb.WriteString("{{ else }}")
			if err := compileElementBody(sb, next, ctx); err != nil {
				return nil, err
			}
			lastNode = next
			break
		} else {
			break
		}
	}
	sb.WriteString("{{ end }}")
	return lastNode, nil
}

// compileVFor emits a {{ range .collection }}...{{ end }} block.
// Only the simple "item in collection" form with a translatable collection
// expression is supported; all other forms return an error.
func compileVFor(sb *strings.Builder, n *xhtml.Node, vforExpr string, ctx *compileContext) error {
	idx := strings.Index(vforExpr, " in ")
	if idx < 0 {
		return fmt.Errorf("v-for: invalid expression %q, expected 'x in expr'", vforExpr)
	}
	collExpr := strings.TrimSpace(vforExpr[idx+4:])
	translated, err := translateExpr(collExpr)
	if err != nil {
		return fmt.Errorf("v-for %q: collection expression: %w", vforExpr, err)
	}
	fmt.Fprintf(sb, "{{ range %s }}", translated)
	if err := compileElementBody(sb, n, ctx); err != nil {
		return err
	}
	sb.WriteString("{{ end }}")
	return nil
}

// compileElementBody writes the element body (children or transparent template
// content) for n, skipping the element wrapper for <template> nodes.
func compileElementBody(sb *strings.Builder, n *xhtml.Node, ctx *compileContext) error {
	if n.Data == "template" {
		return compileChildren(sb, n, ctx)
	}
	return compileElementNode(sb, n, ctx)
}

// compileElementNode writes the opening tag, attributes, children, and closing
// tag for a single HTML element node. Returns an error for any htmlc directive
// that has no html/template equivalent.
func compileElementNode(sb *strings.Builder, n *xhtml.Node, ctx *compileContext) error {
	// <template> without a controlling directive: render children transparently.
	if n.Data == "template" {
		if _, ok := attrValue(n, "v-switch"); ok {
			return fmt.Errorf("v-switch has no html/template equivalent")
		}
		return compileChildren(sb, n, ctx)
	}

	// <slot> / <slot name="…"> → {{ block "name" . }}...{{ end }}
	if n.Data == "slot" {
		slotName := "default"
		if nameAttr, ok := attrValue(n, "name"); ok {
			slotName = nameAttr
		}
		fmt.Fprintf(sb, `{{ block %q . }}`, slotName)
		if err := compileChildren(sb, n, ctx); err != nil {
			return err
		}
		sb.WriteString("{{ end }}")
		return nil
	}

	// PascalCase component reference → {{ template "Name" . }}
	if isComponentLikeName(n.Data) {
		compName := n.Data
		fmt.Fprintf(sb, `{{ template %q . }}`, compName)
		// Queue sub-component define block if not yet emitted.
		if !ctx.visited[compName] {
			if subEntry, ok := ctx.entries[compName]; ok && subEntry.renderFn == nil {
				subSrc, err := compileEntryToDefineBlock(compName, subEntry, ctx)
				if err != nil {
					return fmt.Errorf("sub-component %q: %w", compName, err)
				}
				ctx.defines.WriteString(subSrc)
			}
		}
		return nil
	}

	// Regular HTML element: validate directives before emitting anything.
	for _, attr := range n.Attr {
		key := attr.Key
		// These are consumed by the parent compileChildren/compileConditionalChain/compileVFor.
		if key == "v-if" || key == "v-else-if" || key == "v-else" || key == "v-for" {
			continue
		}
		if isClientSideDirective(key) {
			continue
		}
		if strings.HasPrefix(key, ":") || strings.HasPrefix(key, "v-bind:") {
			continue // supported — attribute binding
		}
		switch key {
		case "v-show", "v-html", "v-text", "v-bind",
			"v-switch", "v-case", "v-default", "v-pre", "v-once":
			return fmt.Errorf("directive %q has no html/template equivalent", key)
		}
		if strings.HasPrefix(key, "v-") {
			return fmt.Errorf("directive %q has no html/template equivalent", key)
		}
	}

	// Open tag.
	fmt.Fprintf(sb, "<%s", n.Data)

	// Attributes.
	for _, attr := range n.Attr {
		key := attr.Key
		val := attr.Val

		// Skip directives already consumed or client-side.
		if key == "v-if" || key == "v-else-if" || key == "v-else" || key == "v-for" {
			continue
		}
		if isClientSideDirective(key) {
			continue
		}

		// :attr="expr" → attr="{{.expr}}"
		if strings.HasPrefix(key, ":") {
			attrName := key[1:]
			translated, err := translateExpr(val)
			if err != nil {
				return fmt.Errorf("%s %q: %w", key, val, err)
			}
			fmt.Fprintf(sb, " %s=\"{{%s}}\"", attrName, translated)
			continue
		}

		// v-bind:attr="expr" → attr="{{.expr}}"
		if strings.HasPrefix(key, "v-bind:") {
			attrName := key[7:]
			translated, err := translateExpr(val)
			if err != nil {
				return fmt.Errorf("%s %q: %w", key, val, err)
			}
			fmt.Fprintf(sb, " %s=\"{{%s}}\"", attrName, translated)
			continue
		}

		// Static attribute.
		if val == "" {
			// Boolean-style attribute: emit without value.
			fmt.Fprintf(sb, " %s", key)
		} else {
			fmt.Fprintf(sb, " %s=%q", key, stdhtml.EscapeString(val))
		}
	}

	if isVoidElement(n.Data) {
		sb.WriteByte('>')
		return nil
	}
	sb.WriteByte('>')

	// Children.
	if err := compileChildren(sb, n, ctx); err != nil {
		return err
	}

	// Close tag.
	fmt.Fprintf(sb, "</%s>", n.Data)
	return nil
}
