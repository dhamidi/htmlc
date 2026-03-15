package main

import (
	"flag"
	"fmt"
	htmltemplate "html/template"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/template/parse"

	"github.com/dhamidi/htmlc"
)

const helpTemplate = `template — convert between .vue components and Go html/template files

SYNOPSIS
  htmlc template <subcommand> [flags] [args]

SUBCOMMANDS
  vue-to-tmpl   Convert a .vue component tree to a Go html/template file
  tmpl-to-vue   Convert a Go html/template file to a .vue component

Run 'htmlc help template vue-to-tmpl' or 'htmlc help template tmpl-to-vue' for
details on each subcommand.
`

const helpVueToTmpl = `template vue-to-tmpl — convert a .vue component to a Go html/template file

SYNOPSIS
  htmlc template vue-to-tmpl [-dir <path>] [-out <file>] <component-name>

FLAGS
  -dir string   Directory containing .vue component files. (default ".")
  -out string   Output file path. If omitted, writes to stdout.

DESCRIPTION
  Loads the named component from -dir, compiles it and all statically reachable
  sub-components to html/template syntax, and writes the result to -out (or
  stdout). The output contains one {{ define "Name" }}...{{ end }} block per
  component, forming a self-contained file suitable for html/template.ParseFiles.

  Supported htmlc constructs:
    {{ identifier }}           → {{ .identifier }}
    {{ identifier.path }}      → {{ .identifier.path }}
    :attr="expr"               → attr="{{ .expr }}"
    v-bind:attr="expr"         → attr="{{ .expr }}"
    v-if / v-else-if / v-else  → {{ if }} / {{ else if }} / {{ else }}
    v-for="item in collection" → {{ range .collection }}
    <slot> / <slot name="N">   → {{ block "N" . }}
    <ComponentName>            → {{ template "ComponentName" . }}

  Unsupported constructs (v-show, v-html, v-text, v-bind spread, v-switch,
  complex expressions, custom directives) cause the command to exit with an
  error. Fix or remove unsupported constructs before conversion.

EXAMPLES
  # Convert a component and print to stdout
  htmlc template vue-to-tmpl -dir ./components Card

  # Convert and write to a file
  htmlc template vue-to-tmpl -dir ./components -out ./out/Card.html Card
`

const helpTmplToVue = `template tmpl-to-vue — convert a Go html/template file to a .vue component

SYNOPSIS
  htmlc template tmpl-to-vue [-out <file>] <template-file>

FLAGS
  -out string   Output .vue file path. If omitted, writes to stdout.

DESCRIPTION
  Reads <template-file> as a Go html/template source. Translates html/template
  actions to htmlc equivalents and emits a .vue Single File Component.

  Supported html/template constructs:
    {{ .field }}               → {{ field }}
    {{ .field.sub }}           → {{ field.sub }}
    {{ if .cond }}             → v-if="cond"
    {{ else if .cond }}        → v-else-if="cond"
    {{ else }}                 → v-else
    {{ range .list }}          → v-for="item in list"
    {{ block "N" . }}          → <slot name="N">
    {{ template "N" . }}       → <ComponentName> (if N is PascalCase)

  Actions that cannot be translated are emitted as
  <!-- tmpl: original action --> HTML comments.

  The output begins with a review-required notice because the conversion is
  inherently best-effort and manual review is always recommended.

EXAMPLES
  # Convert a template and print to stdout
  htmlc template tmpl-to-vue ./legacy/nav.html

  # Convert and write to a .vue file
  htmlc template tmpl-to-vue -out ./components/Nav.vue ./legacy/nav.html
`

func runTemplateCommand(args []string, stdout, stderr io.Writer, strict bool) error {
	if len(args) == 0 {
		fmt.Fprint(stdout, helpTemplate)
		return nil
	}

	subcmd, rest := args[0], args[1:]

	switch subcmd {
	case "vue-to-tmpl":
		return runVueToTmpl(rest, stdout, stderr, strict)
	case "tmpl-to-vue":
		return runTmplToVue(rest, stdout, stderr, strict)
	case "-h", "--help", "help":
		fmt.Fprint(stdout, helpTemplate)
		return nil
	default:
		fmt.Fprintf(stderr, "htmlc template: unknown subcommand %q\n\nRun 'htmlc template' to see available subcommands.\n", subcmd)
		return errSilent
	}
}

func runVueToTmpl(args []string, stdout, stderr io.Writer, _ bool) error {
	args = normalizeArgs(args)
	fs := flag.NewFlagSet("template vue-to-tmpl", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .vue components")
	out := fs.String("out", "", "output file path (stdout if omitted)")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpVueToTmpl)
			return nil
		}
		return err
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl",
			"component name required",
			"  Usage: htmlc template vue-to-tmpl [-dir <path>] [-out <file>] <component-name>",
		))
		return errSilent
	}

	name := fs.Arg(0)

	if _, statErr := os.Stat(*dir); statErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl",
			fmt.Sprintf("cannot load components from %q", *dir),
			"  Directory does not exist.",
		))
		return errSilent
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir})
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl",
			fmt.Sprintf("failed to initialise engine: %v", err),
		))
		return errSilent
	}

	src, err := engine.ExportTemplateSource(name)
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl", err.Error()))
		return errSilent
	}

	if *out == "" {
		_, writeErr := io.WriteString(stdout, src)
		return writeErr
	}

	if mkErr := os.MkdirAll(filepath.Dir(*out), 0755); mkErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl",
			fmt.Sprintf("cannot create output directory: %v", mkErr),
		))
		return errSilent
	}
	if writeErr := os.WriteFile(*out, []byte(src), 0644); writeErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl",
			fmt.Sprintf("cannot write output file: %v", writeErr),
		))
		return errSilent
	}
	return nil
}

func runTmplToVue(args []string, stdout, stderr io.Writer, _ bool) error {
	args = normalizeArgs(args)
	fs := flag.NewFlagSet("template tmpl-to-vue", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "output .vue file path (stdout if omitted)")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpTmplToVue)
			return nil
		}
		return err
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, cmdErrorMsg("template tmpl-to-vue",
			"template file required",
			"  Usage: htmlc template tmpl-to-vue [-out <file>] <template-file>",
		))
		return errSilent
	}

	tmplFile := fs.Arg(0)

	t, err := htmltemplate.ParseFiles(tmplFile)
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template tmpl-to-vue",
			fmt.Sprintf("cannot parse template file %q: %v", tmplFile, err),
		))
		return errSilent
	}

	var sb strings.Builder
	sb.WriteString("<!-- generated by htmlc template tmpl-to-vue; review required -->\n")
	sb.WriteString("<template>\n")

	// html/template.ParseFiles registers templates under their base filename
	// by default. Use the template whose name matches the file's base name.
	baseName := strings.TrimSuffix(filepath.Base(tmplFile), filepath.Ext(tmplFile))
	target := t.Lookup(filepath.Base(tmplFile))
	if target == nil {
		target = t.Lookup(baseName)
	}
	if target == nil {
		// Fall back to the first template with a non-empty tree.
		for _, tmpl := range t.Templates() {
			if tmpl.Tree != nil && tmpl.Tree.Root != nil {
				target = tmpl
				break
			}
		}
	}

	if target != nil && target.Tree != nil {
		convertTmplNodeToVue(&sb, target.Tree.Root, 0)
	}

	sb.WriteString("</template>\n")

	result := sb.String()

	if *out == "" {
		_, writeErr := io.WriteString(stdout, result)
		return writeErr
	}

	if mkErr := os.MkdirAll(filepath.Dir(*out), 0755); mkErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template tmpl-to-vue",
			fmt.Sprintf("cannot create output directory: %v", mkErr),
		))
		return errSilent
	}
	if writeErr := os.WriteFile(*out, []byte(result), 0644); writeErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template tmpl-to-vue",
			fmt.Sprintf("cannot write output file: %v", writeErr),
		))
		return errSilent
	}
	return nil
}

// convertTmplNodeToVue converts a text/template/parse node tree to htmlc
// .vue syntax. Unsupported actions are preserved as HTML comments.
func convertTmplNodeToVue(sb *strings.Builder, node parse.Node, depth int) {
	if node == nil {
		return
	}
	switch n := node.(type) {
	case *parse.ListNode:
		for _, child := range n.Nodes {
			convertTmplNodeToVue(sb, child, depth)
		}

	case *parse.TextNode:
		sb.WriteString(string(n.Text))

	case *parse.ActionNode:
		// {{ .field }} → {{ field }}
		if n.Pipe != nil && len(n.Pipe.Cmds) == 1 {
			cmd := n.Pipe.Cmds[0]
			if len(cmd.Args) == 1 {
				if field, ok := cmd.Args[0].(*parse.FieldNode); ok {
					sb.WriteString("{{ ")
					sb.WriteString(strings.Join(field.Ident, "."))
					sb.WriteString(" }}")
					return
				}
			}
		}
		// Unsupported action: emit as comment.
		fmt.Fprintf(sb, "<!-- tmpl: %s -->", n.String())

	case *parse.IfNode:
		// {{ if .cond }} → v-if="cond" on a <template> wrapper element.
		condStr := tmplPipeToExpr(n.Pipe)
		fmt.Fprintf(sb, "<template v-if=%q>", condStr)
		convertTmplNodeToVue(sb, n.List, depth+1)
		sb.WriteString("</template>")
		if n.ElseList != nil && len(n.ElseList.Nodes) > 0 {
			// Check if it's an else-if (single if node inside else) or plain else.
			if len(n.ElseList.Nodes) == 1 {
				if innerIf, ok := n.ElseList.Nodes[0].(*parse.IfNode); ok {
					innerCond := tmplPipeToExpr(innerIf.Pipe)
					fmt.Fprintf(sb, "<template v-else-if=%q>", innerCond)
					convertTmplNodeToVue(sb, innerIf.List, depth+1)
					sb.WriteString("</template>")
					if innerIf.ElseList != nil {
						sb.WriteString("<template v-else>")
						convertTmplNodeToVue(sb, innerIf.ElseList, depth+1)
						sb.WriteString("</template>")
					}
					return
				}
			}
			sb.WriteString("<template v-else>")
			convertTmplNodeToVue(sb, n.ElseList, depth+1)
			sb.WriteString("</template>")
		}

	case *parse.RangeNode:
		// {{ range .list }} → v-for="item in list"
		collStr := tmplPipeToExpr(n.Pipe)
		fmt.Fprintf(sb, "<template v-for=\"item in %s\">", collStr)
		convertTmplNodeToVue(sb, n.List, depth+1)
		sb.WriteString("</template>")

	case *parse.TemplateNode:
		// {{ template "Name" . }} → <Name></Name> if PascalCase (component call),
		// or <slot name="Name"> if the template name looks like a block/slot,
		// or <!-- tmpl: ... --> for everything else.
		if isPascalCase(n.Name) {
			fmt.Fprintf(sb, "<%s></%s>", n.Name, n.Name)
		} else if n.Name == "content" || n.Name == "default" {
			// {{ template "content" . }} / {{ block "content" . }} → default slot
			sb.WriteString("<slot></slot>")
		} else {
			// Named block/slot
			fmt.Fprintf(sb, "<slot name=%q></slot>", n.Name)
		}

	default:
		// Unknown node type: emit as comment.
		fmt.Fprintf(sb, "<!-- tmpl: %s -->", node.String())
	}
}

// tmplPipeToExpr extracts a simple field expression from a template pipeline.
// {{ .field }} → "field", {{ .field.sub }} → "field.sub".
// Falls back to the pipeline's String() representation for complex pipelines.
func tmplPipeToExpr(pipe *parse.PipeNode) string {
	if pipe == nil {
		return ""
	}
	if len(pipe.Cmds) == 1 && len(pipe.Cmds[0].Args) == 1 {
		if field, ok := pipe.Cmds[0].Args[0].(*parse.FieldNode); ok {
			return strings.Join(field.Ident, ".")
		}
	}
	return pipe.String()
}

// isPascalCase reports whether s starts with an uppercase ASCII letter,
// which is the htmlc convention for component names.
func isPascalCase(s string) bool {
	return len(s) > 0 && s[0] >= 'A' && s[0] <= 'Z'
}
