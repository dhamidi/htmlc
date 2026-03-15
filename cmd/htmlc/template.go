package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/dhamidi/htmlc"
)

const helpTemplate = `template — convert between .vue components and html/template syntax

SYNOPSIS
  htmlc template <action> [flags] [args]

ACTIONS
  vue-to-tmpl   Convert a .vue component to html/template {{ define }} blocks
  tmpl-to-vue   Convert an html/template file (stdin) to a .vue component

Run 'htmlc template <action> -help' for action-specific flags.

EXAMPLES
  # Convert Card component to html/template syntax
  htmlc template vue-to-tmpl -dir ./templates Card

  # Convert an html/template file to .vue
  cat page.tmpl | htmlc template tmpl-to-vue -name MyPage
`

const helpTemplateVueToTmpl = `template vue-to-tmpl — convert a .vue component to html/template syntax

SYNOPSIS
  htmlc template vue-to-tmpl [-dir <path>] [-quiet] <ComponentName>

DESCRIPTION
  Loads the named .vue component (and all components it statically references)
  from the component directory and converts them to html/template {{ define }}
  blocks.  The output is written to stdout and is suitable for parsing with
  html/template.New("").Parse(output).

  Sub-components are emitted in dependency order (leaves first), followed by
  the root component.

  Scoped <style> blocks are stripped from the output.

FLAGS
  -dir string   Directory containing .vue component files. (default ".")
  -quiet        Suppress warnings; only emit errors.

EXIT STATUS
  0   Success
  1   Error (component not found, unsupported construct, etc.)

EXAMPLES
  # Convert Card component
  htmlc template vue-to-tmpl -dir ./templates Card

  # Suppress warnings
  htmlc template vue-to-tmpl -quiet -dir ./templates Card
`

const helpTemplateTmplToVue = `template tmpl-to-vue — convert an html/template file to a .vue component

SYNOPSIS
  htmlc template tmpl-to-vue [-name <ComponentName>] [-quiet]

DESCRIPTION
  Reads html/template source from stdin and converts it to a .vue component,
  writing the result to stdout.

  The conversion is best-effort.  Unsupported constructs (pipelines with
  multiple commands, {{with}} blocks, variable assignments) cause an error.

FLAGS
  -name string  Component name used for the generated root element. (default "Component")
  -quiet        Suppress warnings; only emit errors.

EXIT STATUS
  0   Success
  1   Error (unsupported construct, read error, etc.)

EXAMPLES
  # Convert from stdin
  cat page.tmpl | htmlc template tmpl-to-vue -name MyPage

  # Suppress warnings
  htmlc template tmpl-to-vue -quiet < page.tmpl
`

// runTemplate dispatches to the template subcommand actions.
func runTemplate(args []string, stdout, stderr io.Writer, _ bool) error {
	if len(args) == 0 || args[0] == "-help" || args[0] == "--help" || args[0] == "-h" {
		fmt.Fprint(stdout, helpTemplate)
		return nil
	}

	action, rest := args[0], args[1:]
	switch action {
	case "vue-to-tmpl":
		return runTemplateVueToTmpl(rest, stdout, stderr)
	case "tmpl-to-vue":
		return runTemplateTmplToVue(rest, os.Stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "htmlc template: unknown action %q\n\nRun 'htmlc template' to see available actions.\n", action)
		return errSilent
	}
}

// runTemplateVueToTmpl implements `htmlc template vue-to-tmpl`.
func runTemplateVueToTmpl(args []string, stdout, stderr io.Writer) error {
	args = normalizeArgs(args)
	fs := flag.NewFlagSet("template vue-to-tmpl", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .vue component files")
	quiet := fs.Bool("quiet", false, "suppress warnings; only emit errors")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpTemplateVueToTmpl)
			return nil
		}
		return err
	}

	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl", "missing component name",
			"",
			"USAGE",
			"  htmlc template vue-to-tmpl [-dir <path>] [-quiet] <ComponentName>",
			"",
			"EXAMPLE",
			"  htmlc template vue-to-tmpl -dir ./templates Card",
		))
		return errSilent
	}
	name := fs.Arg(0)

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir})
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl",
			fmt.Sprintf("failed to initialise engine: %v", err)))
		return errSilent
	}

	text, warnings, err := engine.TemplateText(name)
	if err != nil {
		if !engine.Has(name) {
			fmt.Fprintln(stderr, componentNotFoundError("template vue-to-tmpl", name, *dir))
		} else {
			fmt.Fprintln(stderr, cmdErrorMsg("template vue-to-tmpl", err.Error()))
		}
		return errSilent
	}

	if !*quiet {
		for _, w := range warnings {
			fmt.Fprintf(stderr, "warning: %s\n", w)
		}
	}

	fmt.Fprint(stdout, text)
	return nil
}

// runTemplateTmplToVue implements `htmlc template tmpl-to-vue`.
// stdin is used as the source of template text; callers pass os.Stdin for
// normal use and a bytes.Reader or strings.Reader for tests.
func runTemplateTmplToVue(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("template tmpl-to-vue", flag.ContinueOnError)
	fs.SetOutput(stderr)
	name := fs.String("name", "Component", "component name for the generated root element")
	quiet := fs.Bool("quiet", false, "suppress warnings; only emit errors")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpTemplateTmplToVue)
			return nil
		}
		return err
	}

	src, err := io.ReadAll(stdin)
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template tmpl-to-vue",
			fmt.Sprintf("failed to read stdin: %v", err)))
		return errSilent
	}

	result, err := htmlc.TemplateToVue(string(src), *name)
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("template tmpl-to-vue", err.Error()))
		return errSilent
	}

	if !*quiet {
		for _, w := range result.Warnings {
			fmt.Fprintf(stderr, "warning: %s\n", w)
		}
	}

	fmt.Fprint(stdout, result.Text)
	return nil
}
