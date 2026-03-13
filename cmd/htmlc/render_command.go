package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dhamidi/htmlc"
)

const helpRender = `render — render a .vue component as an HTML fragment

SYNOPSIS
  htmlc render [-strict] [-dir <path>] [-props <json|->] [-debug] <component>

DESCRIPTION
  Renders the named .vue component and writes the resulting HTML fragment to
  stdout. Scoped styles are prepended as a <style> block. The component name
  is matched case-insensitively against files in the component directory.

FLAGS
  -strict       Enable strict mode: missing props abort with an error and all
                components are validated before rendering.
  -dir string   Directory containing .vue component files. (default ".")
  -props string Props as a JSON object, or "-" to read JSON from stdin.
  -debug        Enable debug render mode: annotate output with HTML comments
                describing component boundaries, expression values, slot
                contents, and skipped nodes. For development use only.

EXAMPLES
  # Render Button with no props
  htmlc render Button

  # Render Card with inline props from a specific directory
  htmlc render -dir ./templates Card -props '{"title":"Hello","count":3}'

  # Render PostCard with props piped from another command
  echo '{"post":{"title":"Intro"}}' | htmlc render PostCard -props -

  # Render with debug annotations
  htmlc render -debug -dir ./templates Card -props '{"title":"Hello"}'
`

func runRender(args []string, stdout, stderr io.Writer, strict bool) error {
	args = normalizeArgs(args)
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .vue components")
	propsFlag := fs.String("props", "", "props as JSON object string, or - to read from stdin")
	debugFlag := fs.Bool("debug", false, "enable debug render mode (annotates output with HTML comments)")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpRender)
			return nil
		}
		return err
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(stderr, cmdErrorMsg("render", "missing component name",
			"",
			"USAGE",
			"  htmlc render [-strict] [-dir <path>] [-props <json|->] [-debug] <component>",
			"",
			"EXAMPLE",
			"  htmlc render -dir ./templates MyComponent",
		))
		return errSilent
	}
	name := fs.Arg(0)

	data, err := parseProps(*propsFlag, os.Stdin)
	if err != nil {
		fmt.Fprintln(stderr, propsJSONError("render", *propsFlag == "-", err))
		return errSilent
	}

	if _, statErr := os.Stat(*dir); statErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("render", fmt.Sprintf("cannot load components from %q", *dir),
			"  No such directory. Create the directory and add .vue component files.",
			"",
			"  EXAMPLE",
			"    mkdir templates",
			"    cp MyComponent.vue templates/",
			fmt.Sprintf("    htmlc render -dir %s MyComponent", *dir),
		))
		return errSilent
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir, Debug: *debugFlag})
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("render", fmt.Sprintf("failed to initialise engine: %v", err),
			"  Run 'htmlc help render' for usage.",
		))
		return errSilent
	}

	if strict {
		engine.WithMissingPropHandler(htmlc.ErrorOnMissingProp)
		if errs := engine.ValidateAll(); len(errs) > 0 {
			for _, ve := range errs {
				fmt.Fprintf(stderr, "htmlc render: validation error in %s: %s\n", ve.Component, ve.Message)
			}
			return errSilent
		}
	}

	if err := engine.RenderFragment(stdout, name, data); err != nil {
		if strings.Contains(err.Error(), name) {
			fmt.Fprintln(stderr, componentNotFoundError("render", name, *dir))
		} else {
			fmt.Fprintln(stderr, cmdErrorMsg("render", err.Error()))
		}
		return errSilent
	}
	return nil
}
