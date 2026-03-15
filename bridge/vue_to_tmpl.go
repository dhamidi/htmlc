// Package bridge implements bidirectional conversion between htmlc .vue
// components and Go's standard html/template format.
//
// # vue→tmpl direction
//
// VueToTemplate converts a parsed *htmlc.Component to a string containing
// one or more {{define "name"}}…{{end}} blocks, suitable for parsing with
// html/template.New("").Parse(result).
//
// Supported constructs:
//   - Text interpolation: {{ ident }} and {{ a.b.c }}
//   - Bound attributes: :attr="ident" and v-bind:attr="ident"
//   - v-if / v-else-if / v-else conditional chains
//   - v-for="item in list" loops (outer-scope refs produce ConversionError)
//   - v-show="ident" injects conditional style="display:none"
//   - v-html="ident" emits {{.ident}} with a warning (caller handles HTML safety)
//   - v-text="ident" emits {{.ident}}, discarding children
//   - v-bind="ident" spread emits {{ .ident }} with a warning
//   - <template v-switch="ident"> with v-case / v-default children
//   - <slot> and <slot name="N"> emit {{block}} blocks
//   - Zero-prop child components emit {{template "name" .}}
//
// Unsupported constructs (return ConversionError):
//   - Complex expressions (anything beyond simple identifiers and dot-paths)
//   - Bound props on child components
//   - Custom directives
//   - Outer-scope variable references inside v-for loops
//
// # tmpl→vue direction
//
// TemplateToVue converts html/template source text to .vue component source.
// This direction is explicitly best-effort; see TemplateToVue for details.
package bridge

import (
	stdhtml "html"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// VueToTemplateResult holds the converted template text and any non-fatal
// warnings generated during conversion.
type VueToTemplateResult struct {
	Text     string
	Warnings []string
}

// VueToTemplate converts a parsed .vue component template tree to Go
// html/template syntax.  tmpl is the root *html.Node from the component's
// Template field (i.e. the parsed <template> section).
//
// The result is a string containing one {{define "componentName"}}…{{end}}
// block, suitable for combining with other such blocks and parsing via
// html/template.New("").Parse(combined).
//
// componentName is the base name used for the {{define}} block.
//
// Returns *ConversionError on the first unsupported construct encountered.
func VueToTemplate(tmpl *html.Node, componentName string) (*VueToTemplateResult, error) {
	ctx := &vueConvCtx{
		sb:       new(strings.Builder),
		warnings: new([]string),
	}
	fmt.Fprintf(ctx.sb, `{{define "%s"}}`, componentName)
	if err := ctx.writeChildren(tmpl); err != nil {
		return nil, err
	}
	ctx.sb.WriteString(`{{end}}`)
	return &VueToTemplateResult{
		Text:     ctx.sb.String(),
		Warnings: *ctx.warnings,
	}, nil
}

// vueConvCtx carries mutable state through the recursive walk.
type vueConvCtx struct {
	sb       *strings.Builder
	warnings *[]string
	forVar   string // current v-for loop variable; empty when not inside a loop
}

func (ctx *vueConvCtx) addWarning(msg string) {
	*ctx.warnings = append(*ctx.warnings, msg)
}

// withForVar returns a shallow copy of ctx with forVar set to v.
func (ctx *vueConvCtx) withForVar(v string) *vueConvCtx {
	return &vueConvCtx{sb: ctx.sb, warnings: ctx.warnings, forVar: v}
}

// bridgeMustacheRe matches {{ expression }} patterns in text nodes.
var bridgeMustacheRe = regexp.MustCompile(`\{\{(.*?)\}\}`)

// convertExpr converts a Vue/htmlc expression to a Go template dot-accessor
// string (e.g. ".name", ".a.b.c", or ".").
//
// When inside a v-for (ctx.forVar != ""), only the loop variable and its
// sub-paths are allowed; any other identifier is an outer-scope reference and
// returns ConversionError.
func (ctx *vueConvCtx) convertExpr(exprStr string) (string, error) {
	exprStr = strings.TrimSpace(exprStr)
	kind := ClassifyExpr(exprStr)
	if kind == ExprComplex {
		return "", &ConversionError{
			Message: fmt.Sprintf("complex expression %q is not supported; only simple identifiers and dot-paths are convertible to Go templates", exprStr),
		}
	}

	if ctx.forVar != "" {
		switch kind {
		case ExprSimpleIdent:
			if exprStr == "." || exprStr == ctx.forVar {
				return ".", nil
			}
			return "", &ConversionError{
				Message: fmt.Sprintf("expression %q references outer-scope variable (loop variable is %q); outer-scope references are not supported inside v-for", exprStr, ctx.forVar),
			}
		case ExprDotPath:
			parts := strings.SplitN(exprStr, ".", 2)
			if parts[0] == ctx.forVar {
				return "." + parts[1], nil
			}
			return "", &ConversionError{
				Message: fmt.Sprintf("expression %q starts with outer-scope variable %q; outer-scope references are not supported inside v-for", exprStr, parts[0]),
			}
		}
	}

	return DotPrefix(exprStr)
}

// writeChildren renders all direct children of parent, handling v-if chains
// and v-for directives.
func (ctx *vueConvCtx) writeChildren(parent *html.Node) error {
	child := parent.FirstChild
	for child != nil {
		if child.Type == html.ElementNode {
			// v-for takes precedence over v-if.
			if vfor, ok := getAttr(child, "v-for"); ok {
				next, err := ctx.writeVFor(child, vfor)
				if err != nil {
					return err
				}
				child = next
				continue
			}
			// v-if starts a conditional chain.
			if _, ok := getAttr(child, "v-if"); ok {
				next, err := ctx.writeConditionalChain(child)
				if err != nil {
					return err
				}
				child = next
				continue
			}
			// v-else-if / v-else without a preceding v-if: skip (invalid Vue,
			// but we consume rather than error to be lenient).
			if _, ok := getAttr(child, "v-else-if"); ok {
				child = child.NextSibling
				continue
			}
			if _, ok := getAttr(child, "v-else"); ok {
				child = child.NextSibling
				continue
			}
		}
		if err := ctx.writeNode(child); err != nil {
			return err
		}
		child = child.NextSibling
	}
	return nil
}

// writeConditionalChain emits a v-if/v-else-if/v-else chain and returns the
// next sibling after the last element in the chain.
func (ctx *vueConvCtx) writeConditionalChain(vIfNode *html.Node) (*html.Node, error) {
	ifExpr, _ := getAttr(vIfNode, "v-if")
	dotExpr, err := ctx.convertExpr(ifExpr)
	if err != nil {
		return nil, &ConversionError{Directive: "v-if", Message: fmt.Sprintf("v-if %q: %s", ifExpr, err)}
	}
	fmt.Fprintf(ctx.sb, "{{if %s}}", dotExpr)
	if err := ctx.writeElement(vIfNode, map[string]bool{"v-if": true}); err != nil {
		return nil, err
	}

	// Scan following siblings for v-else-if / v-else, skipping whitespace.
	current := vIfNode.NextSibling
	for current != nil {
		if current.Type == html.TextNode && strings.TrimSpace(current.Data) == "" {
			current = current.NextSibling
			continue
		}
		if current.Type != html.ElementNode {
			break
		}
		if elseIfExpr, ok := getAttr(current, "v-else-if"); ok {
			dotElseIf, err := ctx.convertExpr(elseIfExpr)
			if err != nil {
				return nil, &ConversionError{Directive: "v-else-if", Message: fmt.Sprintf("v-else-if %q: %s", elseIfExpr, err)}
			}
			fmt.Fprintf(ctx.sb, "{{else if %s}}", dotElseIf)
			if err := ctx.writeElement(current, map[string]bool{"v-else-if": true}); err != nil {
				return nil, err
			}
			current = current.NextSibling
			continue
		}
		if _, ok := getAttr(current, "v-else"); ok {
			ctx.sb.WriteString("{{else}}")
			if err := ctx.writeElement(current, map[string]bool{"v-else": true}); err != nil {
				return nil, err
			}
			current = current.NextSibling
			break
		}
		break
	}
	ctx.sb.WriteString("{{end}}")
	return current, nil
}

// writeVFor emits a {{range .list}}…{{end}} block and returns the next sibling.
func (ctx *vueConvCtx) writeVFor(n *html.Node, vforExpr string) (*html.Node, error) {
	idx := strings.Index(vforExpr, " in ")
	if idx < 0 {
		return nil, &ConversionError{Directive: "v-for", Message: fmt.Sprintf("v-for: invalid expression %q, expected 'var in collection'", vforExpr)}
	}
	rawVars := strings.TrimSpace(vforExpr[:idx])
	// Strip parentheses from destructured patterns like "(item, index)".
	loopVar := strings.Trim(rawVars, "()")
	if commaIdx := strings.Index(loopVar, ","); commaIdx >= 0 {
		loopVar = strings.TrimSpace(loopVar[:commaIdx])
	}
	collExpr := strings.TrimSpace(vforExpr[idx+4:])

	dotColl, err := ctx.convertExpr(collExpr)
	if err != nil {
		return nil, &ConversionError{Directive: "v-for", Message: fmt.Sprintf("v-for collection %q: %s", collExpr, err)}
	}
	fmt.Fprintf(ctx.sb, "{{range %s}}", dotColl)

	innerCtx := ctx.withForVar(loopVar)
	if err := innerCtx.writeElement(n, map[string]bool{"v-for": true}); err != nil {
		return nil, err
	}
	ctx.sb.WriteString("{{end}}")
	return n.NextSibling, nil
}

// writeNode dispatches a single HTML node to the appropriate writer.
func (ctx *vueConvCtx) writeNode(n *html.Node) error {
	switch n.Type {
	case html.DocumentNode:
		return ctx.writeChildren(n)
	case html.TextNode:
		return ctx.writeTextNode(n.Data)
	case html.ElementNode:
		return ctx.writeElement(n, nil)
	case html.CommentNode:
		fmt.Fprintf(ctx.sb, "<!--%s-->", n.Data)
	}
	return nil
}

// writeTextNode processes a text node, converting {{expr}} mustache patterns
// to Go template actions and HTML-escaping the surrounding literal text.
func (ctx *vueConvCtx) writeTextNode(text string) error {
	lastEnd := 0
	for _, match := range bridgeMustacheRe.FindAllStringSubmatchIndex(text, -1) {
		if match[0] > lastEnd {
			ctx.sb.WriteString(stdhtml.EscapeString(text[lastEnd:match[0]]))
		}
		expr := strings.TrimSpace(text[match[2]:match[3]])
		converted, err := ctx.convertExpr(expr)
		if err != nil {
			return err
		}
		fmt.Fprintf(ctx.sb, "{{%s}}", converted)
		lastEnd = match[1]
	}
	if lastEnd < len(text) {
		ctx.sb.WriteString(stdhtml.EscapeString(text[lastEnd:]))
	}
	return nil
}

// writeElement emits an element and its content.  skipAttrs lists attribute
// keys that have already been consumed by the caller (e.g. "v-if") and must
// not be re-emitted.
func (ctx *vueConvCtx) writeElement(n *html.Node, skipAttrs map[string]bool) error {
	// <template v-switch="…"> → if/else chain.
	// <template> without v-switch → render children transparently.
	if n.Data == "template" {
		if switchExpr, ok := getAttr(n, "v-switch"); ok {
			return ctx.writeSwitchBlock(n, switchExpr)
		}
		return ctx.writeChildren(n)
	}

	// <slot> / <slot name="N"> → {{block "name" .}}…{{end}}.
	if n.Data == "slot" {
		slotName := "default"
		if name, ok := getAttr(n, "name"); ok && name != "" {
			slotName = name
		}
		fmt.Fprintf(ctx.sb, `{{block "%s" .}}`, slotName)
		if err := ctx.writeChildren(n); err != nil {
			return err
		}
		ctx.sb.WriteString("{{end}}")
		return nil
	}

	// Non-standard element → child component.
	if n.DataAtom == 0 {
		for _, attr := range n.Attr {
			if strings.HasPrefix(attr.Key, ":") || strings.HasPrefix(attr.Key, "v-bind:") {
				return &ConversionError{
					Directive: attr.Key,
					Message:   fmt.Sprintf("component <%s> has bound prop %q; only zero-prop components can be converted", n.Data, attr.Key),
				}
			}
		}
		fmt.Fprintf(ctx.sb, `{{template "%s" .}}`, n.Data)
		return nil
	}

	// ---- Regular HTML element ----

	// Phase 1: collect control directives and check for custom ones.
	var vShowExpr, vHTMLExpr, vTextExpr, vBindSpreadExpr string
	var staticStyleVal string
	var hasStaticStyle, hasBindStyle bool

	for _, attr := range n.Attr {
		if skipAttrs != nil && skipAttrs[attr.Key] {
			continue
		}
		switch attr.Key {
		case "v-show":
			vShowExpr = attr.Val
		case "v-html":
			vHTMLExpr = attr.Val
		case "v-text":
			vTextExpr = attr.Val
		case "v-bind":
			vBindSpreadExpr = attr.Val
		case "style":
			staticStyleVal = attr.Val
			hasStaticStyle = true
		case ":style", "v-bind:style":
			hasBindStyle = true
		}
		// Custom directive check: starts with "v-" but not a recognised built-in.
		if strings.HasPrefix(attr.Key, "v-") && !isKnownVueAttr(attr.Key) {
			return &ConversionError{
				Directive: attr.Key,
				Message:   fmt.Sprintf("custom directive %q is not supported in vue→tmpl conversion", attr.Key),
			}
		}
	}

	if vShowExpr != "" && hasBindStyle {
		ctx.addWarning(fmt.Sprintf("v-show on <%s> combined with :style; the converted template may not behave identically", n.Data))
	}

	// Phase 2: emit opening tag.
	fmt.Fprintf(ctx.sb, "<%s", n.Data)

	// v-bind spread: emit as {{.ident}} with a warning.
	if vBindSpreadExpr != "" {
		dotExpr, err := ctx.convertExpr(vBindSpreadExpr)
		if err != nil {
			return &ConversionError{Directive: "v-bind", Message: fmt.Sprintf("v-bind spread %q: %s", vBindSpreadExpr, err)}
		}
		ctx.addWarning(fmt.Sprintf("v-bind spread on <%s> converted to {{%s}}; caller must ensure the value satisfies html/template.HTMLAttr", n.Data, dotExpr))
		fmt.Fprintf(ctx.sb, " {{%s}}", dotExpr)
	}

	// Emit regular attributes.
	for _, attr := range n.Attr {
		if skipAttrs != nil && skipAttrs[attr.Key] {
			continue
		}
		// Skip all consumed directive attributes.
		if shouldSkipAttrEmit(attr.Key) {
			continue
		}
		// Skip static style when v-show is present (handled separately below).
		if attr.Key == "style" && vShowExpr != "" {
			continue
		}
		// Bound :style or v-bind:style.
		if attr.Key == ":style" || attr.Key == "v-bind:style" {
			dotExpr, err := ctx.convertExpr(attr.Val)
			if err != nil {
				return &ConversionError{Directive: attr.Key, Message: fmt.Sprintf("%s=%q: %s", attr.Key, attr.Val, err)}
			}
			fmt.Fprintf(ctx.sb, ` style="{{%s}}"`, dotExpr)
			continue
		}
		// Shorthand bound attribute :attr="expr".
		if strings.HasPrefix(attr.Key, ":") {
			attrName := attr.Key[1:]
			dotExpr, err := ctx.convertExpr(attr.Val)
			if err != nil {
				return &ConversionError{Directive: attr.Key, Message: fmt.Sprintf("%s=%q: %s", attr.Key, attr.Val, err)}
			}
			fmt.Fprintf(ctx.sb, ` %s="{{%s}}"`, attrName, dotExpr)
			continue
		}
		// Long-form v-bind:attr="expr".
		if strings.HasPrefix(attr.Key, "v-bind:") {
			attrName := attr.Key[7:]
			dotExpr, err := ctx.convertExpr(attr.Val)
			if err != nil {
				return &ConversionError{Directive: attr.Key, Message: fmt.Sprintf("%s=%q: %s", attr.Key, attr.Val, err)}
			}
			fmt.Fprintf(ctx.sb, ` %s="{{%s}}"`, attrName, dotExpr)
			continue
		}
		// Static attribute.
		if attr.Val == "" {
			fmt.Fprintf(ctx.sb, " %s", attr.Key)
		} else {
			fmt.Fprintf(ctx.sb, ` %s="%s"`, attr.Key, stdhtml.EscapeString(attr.Val))
		}
	}

	// v-show handling: inject or merge style.
	if vShowExpr != "" {
		dotShow, err := ctx.convertExpr(vShowExpr)
		if err != nil {
			return &ConversionError{Directive: "v-show", Message: fmt.Sprintf("v-show %q: %s", vShowExpr, err)}
		}
		if hasStaticStyle {
			ctx.addWarning(fmt.Sprintf("v-show on <%s> merged with existing style attribute", n.Data))
			fmt.Fprintf(ctx.sb, ` style="{{if not %s}}display:none; {{end}}%s"`, dotShow, stdhtml.EscapeString(staticStyleVal))
		} else if !hasBindStyle {
			fmt.Fprintf(ctx.sb, `{{if not %s}} style="display:none"{{end}}`, dotShow)
		}
	}

	// Close the opening tag.
	ctx.sb.WriteString(">")
	if isVoidElement(n.Data) {
		return nil
	}

	// Phase 3: emit content.
	switch {
	case vHTMLExpr != "":
		dotExpr, err := ctx.convertExpr(vHTMLExpr)
		if err != nil {
			return &ConversionError{Directive: "v-html", Message: fmt.Sprintf("v-html %q: %s", vHTMLExpr, err)}
		}
		ctx.addWarning(fmt.Sprintf("v-html on <%s> converted to {{%s}}; caller must ensure the value is of type html/template.HTML", n.Data, dotExpr))
		fmt.Fprintf(ctx.sb, "{{%s}}", dotExpr)
	case vTextExpr != "":
		dotExpr, err := ctx.convertExpr(vTextExpr)
		if err != nil {
			return &ConversionError{Directive: "v-text", Message: fmt.Sprintf("v-text %q: %s", vTextExpr, err)}
		}
		fmt.Fprintf(ctx.sb, "{{%s}}", dotExpr)
		// Children discarded (v-text replaces element content).
	default:
		if err := ctx.writeChildren(n); err != nil {
			return err
		}
	}

	// Phase 4: closing tag.
	fmt.Fprintf(ctx.sb, "</%s>", n.Data)
	return nil
}

// writeSwitchBlock converts a <template v-switch="expr"> to an
// {{if eq .expr "case1"}}…{{else if eq .expr "case2"}}…{{else}}…{{end}} chain.
func (ctx *vueConvCtx) writeSwitchBlock(n *html.Node, switchExpr string) error {
	dotExpr, err := ctx.convertExpr(switchExpr)
	if err != nil {
		return &ConversionError{Directive: "v-switch", Message: fmt.Sprintf("v-switch %q: %s", switchExpr, err)}
	}
	first := true
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type != html.ElementNode {
			continue
		}
		if caseVal, ok := getAttr(child, "v-case"); ok {
			if first {
				fmt.Fprintf(ctx.sb, `{{if eq %s %q}}`, dotExpr, caseVal)
				first = false
			} else {
				fmt.Fprintf(ctx.sb, `{{else if eq %s %q}}`, dotExpr, caseVal)
			}
			if err := ctx.writeElement(child, map[string]bool{"v-case": true}); err != nil {
				return err
			}
		} else if _, ok := getAttr(child, "v-default"); ok {
			if first {
				// No v-case elements, just a default.
				ctx.sb.WriteString("{{if true}}")
				first = false
			} else {
				ctx.sb.WriteString("{{else}}")
			}
			if err := ctx.writeElement(child, map[string]bool{"v-default": true}); err != nil {
				return err
			}
		}
	}
	if !first {
		ctx.sb.WriteString("{{end}}")
	}
	return nil
}

// getAttr returns the value of the named attribute and whether it was found.
func getAttr(n *html.Node, key string) (string, bool) {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val, true
		}
	}
	return "", false
}

// isKnownVueAttr reports whether the attribute key is a recognised Vue
// directive or shorthand that this converter handles.
func isKnownVueAttr(key string) bool {
	switch key {
	case "v-if", "v-else-if", "v-else", "v-for",
		"v-show", "v-html", "v-text",
		"v-bind", "v-on", "v-model",
		"v-pre", "v-once", "v-cloak",
		"v-switch", "v-case", "v-default",
		"v-slot":
		return true
	}
	switch {
	case strings.HasPrefix(key, "v-bind:"),
		strings.HasPrefix(key, "v-on:"),
		strings.HasPrefix(key, "v-slot:"),
		strings.HasPrefix(key, ":"),
		strings.HasPrefix(key, "@"),
		strings.HasPrefix(key, "#"):
		return true
	}
	return false
}

// shouldSkipAttrEmit reports whether an attribute should be omitted from the
// emitted template text (it was consumed by directive handling above).
func shouldSkipAttrEmit(key string) bool {
	switch key {
	case "v-show", "v-html", "v-text", "v-bind",
		"v-if", "v-else-if", "v-else", "v-for",
		"v-switch", "v-case", "v-default",
		"v-pre", "v-once", "v-cloak",
		"v-model", "v-on", "v-slot":
		return true
	}
	if strings.HasPrefix(key, "v-on:") || strings.HasPrefix(key, "@") {
		return true // client-side only
	}
	if strings.HasPrefix(key, "v-slot:") || strings.HasPrefix(key, "#") {
		return true
	}
	return false
}

// voidElements is the set of HTML5 void elements.
var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

func isVoidElement(tag string) bool { return voidElements[tag] }
