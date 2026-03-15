package bridge

import (
	"fmt"
	"strings"
	"text/template/parse"
)

// TemplateToVueResult holds the converted .vue source text and any non-fatal
// warnings generated during conversion.
type TemplateToVueResult struct {
	Text     string
	Warnings []string
}

// TemplateToVue converts the text of an html/template file to .vue syntax.
// src is the raw template source; componentName is used only for error messages.
//
// The conversion is explicitly best-effort: only constructs with unambiguous
// .vue equivalents are translated.  The first unsupported construct halts
// conversion and returns a *ConversionError; no partial output is produced.
//
// # Supported constructs
//
//   - Text nodes: emitted verbatim
//   - {{.ident}} and {{.a.b.c}}: converted to {{ ident }} and {{ a.b.c }}
//   - {{if .cond}}…{{end}}: wraps body in <div v-if="cond">
//   - {{if .cond}}…{{else}}…{{end}}: emits v-if and v-else on synthetic <div>
//   - {{range .items}}…{{end}}: wraps body in <ul><li v-for="item in items">
//   - {{block "name" .}}…{{end}}: emits <slot name="name"> (or <slot> for "default")
//   - {{template "Name" .}}: emits <Name />
//
// # Unsupported constructs (return ConversionError)
//
//   - {{.items | len}} or any multi-command pipeline
//   - {{with .x}}
//   - Variable assignments ($x := …)
//   - Actions whose pipeline is not a single FieldNode
//   - {{template "Name" expr}} where expr is not .
//
// Note: {{block "name" .}} is desugared by the template parser into a
// {{define "name"}} block plus a {{template "name" .}} call.  TemplateToVue
// detects this by checking whether a parsed sub-tree with that name exists
// alongside the root tree in the parse result.
func TemplateToVue(src, componentName string) (*TemplateToVueResult, error) {
	trees, err := parse.Parse("tmpl", src, "", "")
	if err != nil {
		return nil, &ConversionError{Component: componentName, Message: fmt.Sprintf("parse: %s", err)}
	}
	tree, ok := trees["tmpl"]
	if !ok || tree == nil || tree.Root == nil {
		return nil, &ConversionError{Component: componentName, Message: "template is empty or has no root"}
	}

	ctx := &tmplConvCtx{
		sb:            new(strings.Builder),
		warnings:      new([]string),
		componentName: componentName,
		trees:         trees, // used to detect {{block}} vs {{template}}
	}
	if err := ctx.walkList(tree.Root); err != nil {
		return nil, err
	}
	return &TemplateToVueResult{
		Text:     fmt.Sprintf("<template>%s</template>", ctx.sb.String()),
		Warnings: *ctx.warnings,
	}, nil
}

// tmplConvCtx carries mutable state through the recursive walk.
type tmplConvCtx struct {
	sb            *strings.Builder
	warnings      *[]string
	componentName string
	// trees is the full map returned by parse.Parse.  When a TemplateNode's
	// name exists as a key in this map, it was originally a {{block}} action
	// and should be emitted as a <slot> rather than a component reference.
	trees map[string]*parse.Tree
}

func (ctx *tmplConvCtx) addWarning(msg string) {
	*ctx.warnings = append(*ctx.warnings, msg)
}

// isBlockTree reports whether name corresponds to a {{block}} definition,
// i.e. the name is a key in trees that is NOT the root "tmpl" tree.
func (ctx *tmplConvCtx) isBlockTree(name string) bool {
	if name == "tmpl" {
		return false
	}
	_, ok := ctx.trees[name]
	return ok
}

// walkList walks a ListNode and emits all its children.
func (ctx *tmplConvCtx) walkList(list *parse.ListNode) error {
	if list == nil {
		return nil
	}
	for _, node := range list.Nodes {
		if err := ctx.walkNode(node); err != nil {
			return err
		}
	}
	return nil
}

// walkNode dispatches a single parse.Node to the appropriate handler.
func (ctx *tmplConvCtx) walkNode(node parse.Node) error {
	switch n := node.(type) {
	case *parse.TextNode:
		ctx.sb.WriteString(string(n.Text))
		return nil
	case *parse.ActionNode:
		return ctx.walkAction(n)
	case *parse.IfNode:
		return ctx.walkIf(n)
	case *parse.RangeNode:
		return ctx.walkRange(n)
	case *parse.TemplateNode:
		return ctx.walkTemplate(n)
	case *parse.WithNode:
		return &ConversionError{
			Component: ctx.componentName,
			Message:   "{{with}} is not supported in tmpl→vue conversion",
		}
	case *parse.ListNode:
		return ctx.walkList(n)
	default:
		return &ConversionError{
			Component: ctx.componentName,
			Message:   fmt.Sprintf("unsupported template node type %T", node),
		}
	}
}

// walkAction converts a template action ({{pipeline}}) to a Vue mustache.
// Only single-command, single-FieldNode pipelines are supported.
func (ctx *tmplConvCtx) walkAction(n *parse.ActionNode) error {
	// Variable assignments ($x := …) are not supported.
	if n.Pipe.IsAssign || len(n.Pipe.Decl) > 0 {
		return &ConversionError{
			Component: ctx.componentName,
			Message:   "variable assignment is not supported in tmpl→vue conversion",
		}
	}
	expr, err := pipeToVueExpr(n.Pipe)
	if err != nil {
		return &ConversionError{
			Component: ctx.componentName,
			Message:   fmt.Sprintf("action {{%s}}: %s", n.Pipe, err),
		}
	}
	fmt.Fprintf(ctx.sb, "{{ %s }}", expr)
	return nil
}

// walkIf converts an {{if}} node to one or more elements with v-if / v-else-if / v-else.
func (ctx *tmplConvCtx) walkIf(n *parse.IfNode) error {
	cond, err := pipeToVueExpr(n.Pipe)
	if err != nil {
		return &ConversionError{Component: ctx.componentName, Message: fmt.Sprintf("{{if}}: %s", err)}
	}
	fmt.Fprintf(ctx.sb, `<div v-if="%s">`, cond)
	if err := ctx.walkList(n.List); err != nil {
		return err
	}
	ctx.sb.WriteString("</div>")

	if n.ElseList != nil {
		return ctx.walkElseList(n.ElseList)
	}
	return nil
}

// walkElseList emits v-else-if or v-else content.
func (ctx *tmplConvCtx) walkElseList(list *parse.ListNode) error {
	if list == nil || len(list.Nodes) == 0 {
		return nil
	}
	// An {{else if}} is represented as a single IfNode inside the else list.
	if len(list.Nodes) == 1 {
		if elseIf, ok := list.Nodes[0].(*parse.IfNode); ok {
			cond, err := pipeToVueExpr(elseIf.Pipe)
			if err != nil {
				return &ConversionError{
					Component: ctx.componentName,
					Message:   fmt.Sprintf("{{else if}}: %s", err),
				}
			}
			fmt.Fprintf(ctx.sb, `<div v-else-if="%s">`, cond)
			if err := ctx.walkList(elseIf.List); err != nil {
				return err
			}
			ctx.sb.WriteString("</div>")
			if elseIf.ElseList != nil {
				return ctx.walkElseList(elseIf.ElseList)
			}
			return nil
		}
	}
	// Regular else.
	ctx.sb.WriteString(`<div v-else>`)
	if err := ctx.walkList(list); err != nil {
		return err
	}
	ctx.sb.WriteString("</div>")
	return nil
}

// walkRange converts a {{range}} node to a v-for element.
func (ctx *tmplConvCtx) walkRange(n *parse.RangeNode) error {
	coll, err := pipeToVueExpr(n.Pipe)
	if err != nil {
		return &ConversionError{Component: ctx.componentName, Message: fmt.Sprintf("{{range}}: %s", err)}
	}
	fmt.Fprintf(ctx.sb, `<ul><li v-for="item in %s">`, coll)
	if err := ctx.walkList(n.List); err != nil {
		return err
	}
	ctx.sb.WriteString("</li></ul>")
	return nil
}

// walkTemplate converts a {{template "Name" .}} or desugared {{block}} node.
//
// When the template name corresponds to a {{block}} definition (i.e. a parsed
// sub-tree with that name exists), the node is emitted as a <slot> element
// whose content comes from that sub-tree.  Otherwise it is emitted as a
// component element <Name />.
func (ctx *tmplConvCtx) walkTemplate(n *parse.TemplateNode) error {
	if ctx.isBlockTree(n.Name) {
		// This TemplateNode was originally {{block "name" .}}content{{end}}.
		// Emit it as a Vue <slot>.
		blockTree := ctx.trees[n.Name]
		if n.Name == "default" {
			ctx.sb.WriteString("<slot>")
		} else {
			fmt.Fprintf(ctx.sb, `<slot name="%s">`, n.Name)
		}
		if blockTree != nil && blockTree.Root != nil {
			if err := ctx.walkList(blockTree.Root); err != nil {
				return err
			}
		}
		ctx.sb.WriteString("</slot>")
		return nil
	}
	// Regular {{template "Name" .}} → component element.
	fmt.Fprintf(ctx.sb, "<%s />", n.Name)
	return nil
}

// pipeToVueExpr extracts a Vue expression string from a simple pipeline.
// Only single-command, single-FieldNode (or DotNode) pipelines are accepted;
// all other forms return an error.
func pipeToVueExpr(pipe *parse.PipeNode) (string, error) {
	if len(pipe.Cmds) != 1 {
		return "", fmt.Errorf("pipeline with %d commands is not supported; only a single field access (e.g. .name) is convertible", len(pipe.Cmds))
	}
	cmd := pipe.Cmds[0]
	if len(cmd.Args) != 1 {
		return "", fmt.Errorf("command with %d arguments is not supported", len(cmd.Args))
	}
	switch arg := cmd.Args[0].(type) {
	case *parse.FieldNode:
		if len(arg.Ident) == 0 {
			return ".", nil
		}
		return strings.Join(arg.Ident, "."), nil
	case *parse.DotNode:
		return ".", nil
	default:
		return "", fmt.Errorf("unsupported argument type %T; only field access (e.g. .name) is convertible", cmd.Args[0])
	}
}
