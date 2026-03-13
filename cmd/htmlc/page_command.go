package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dhamidi/htmlc"
)

const helpPage = `page — render a .vue component as a full HTML page

SYNOPSIS
  htmlc page [-strict] [-dir <path>] [-props <json|->] [-debug] [-layout <component>] <component>

DESCRIPTION
  Renders the named .vue component and writes a complete HTML document to
  stdout. The output includes a proper <!DOCTYPE html> wrapper around the
  rendered component. Scoped styles are injected into the document head.
  The component name is matched case-insensitively against files in the
  component directory.

FLAGS
  -strict         Enable strict mode: missing props abort with an error and all
                  components are validated before rendering.
  -dir string     Directory containing .vue component files. (default ".")
  -props string   Props as a JSON object, or "-" to read JSON from stdin.
  -debug          Enable debug render mode: annotate output with HTML comments
                  describing component boundaries, expression values, slot
                  contents, and skipped nodes. For development use only.
  -layout string  Wrap the rendered page inside this layout component.
                  The layout receives the rendered HTML as a "content" prop.
                  (default: no layout)

EXAMPLES
  # Render HomePage as a full HTML page
  htmlc page HomePage

  # Render PostPage with props from a specific directory
  htmlc page -dir ./templates PostPage -props '{"slug":"intro","title":"Hello"}'

  # Render with props piped from stdin
  echo '{"slug":"intro"}' | htmlc page -dir ./templates PostPage -props -

  # Render with debug annotations
  htmlc page -debug -dir ./templates PostPage -props '{"slug":"intro"}'

  # Wrap a page component inside a layout
  htmlc page -dir ./templates -layout AppLayout PostPage \
    -props '{"title":"Hello","body":"World"}'
`

func runPage(args []string, stdout, stderr io.Writer, strict bool) error {
	args = normalizeArgs(args)
	fs := flag.NewFlagSet("page", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .vue components")
	propsFlag := fs.String("props", "", "props as JSON object string, or - to read from stdin")
	debugFlag := fs.Bool("debug", false, "enable debug render mode (annotates output with HTML comments)")
	layoutFlag := fs.String("layout", "", "wrap rendered page inside this layout component")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpPage)
			return nil
		}
		return err
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, cmdErrorMsg("page", "missing component name",
			"",
			"USAGE",
			"  htmlc page [-strict] [-dir <path>] [-props <json|->] [-debug] [-layout <component>] <component>",
			"",
			"EXAMPLE",
			"  htmlc page -dir ./templates MyPage",
		))
		return errSilent
	}
	name := fs.Arg(0)

	data, err := parseProps(*propsFlag, os.Stdin)
	if err != nil {
		fmt.Fprintln(stderr, propsJSONError("page", *propsFlag == "-", err))
		return errSilent
	}

	if _, statErr := os.Stat(*dir); statErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("page", fmt.Sprintf("cannot load components from %q", *dir),
			"  No such directory. Create the directory and add .vue component files.",
			"",
			"  EXAMPLE",
			"    mkdir templates",
			"    cp MyComponent.vue templates/",
			fmt.Sprintf("    htmlc page -dir %s MyPage", *dir),
		))
		return errSilent
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir, Debug: *debugFlag})
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("page", fmt.Sprintf("failed to initialise engine: %v", err),
			"  Run 'htmlc help page' for usage.",
		))
		return errSilent
	}

	if strict {
		engine.WithMissingPropHandler(htmlc.ErrorOnMissingProp)
		if errs := engine.ValidateAll(); len(errs) > 0 {
			for _, ve := range errs {
				fmt.Fprintf(stderr, "htmlc page: validation error in %s: %s\n", ve.Component, ve.Message)
			}
			return errSilent
		}
	}

	if *layoutFlag != "" {
		// Render the page component as a fragment first.
		content, err := engine.RenderFragmentString(name, data)
		if err != nil {
			if strings.Contains(err.Error(), name) {
				fmt.Fprintln(stderr, componentNotFoundError("page", name, *dir))
			} else {
				fmt.Fprintln(stderr, cmdErrorMsg("page", err.Error()))
			}
			return errSilent
		}
		// Build layout data: copy all top-level props and add "content".
		layoutData := make(map[string]any, len(data)+1)
		for k, v := range data {
			layoutData[k] = v
		}
		layoutData["content"] = content
		// Render the layout as the full page document.
		if err := engine.RenderPage(stdout, *layoutFlag, layoutData); err != nil {
			if strings.Contains(err.Error(), *layoutFlag) {
				fmt.Fprintln(stderr, componentNotFoundError("page", *layoutFlag, *dir))
			} else {
				fmt.Fprintln(stderr, cmdErrorMsg("page", err.Error()))
			}
			return errSilent
		}
	} else {
		if err := engine.RenderPage(stdout, name, data); err != nil {
			if strings.Contains(err.Error(), name) {
				fmt.Fprintln(stderr, componentNotFoundError("page", name, *dir))
			} else {
				fmt.Fprintln(stderr, cmdErrorMsg("page", err.Error()))
			}
			return errSilent
		}
	}
	return nil
}
