package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/dhamidi/htmlc"
)

const helpTop = `htmlc — server-side Vue.js component renderer

USAGE
  htmlc <subcommand> [flags] [args]

SUBCOMMANDS
  render   Render a component as an HTML fragment (stdout)
  page     Render a component as a full HTML page (stdout)
  props    List the props expected by a component
  help     Show help for a subcommand

EXAMPLES
  # Render a fragment with inline props
  htmlc render -dir ./templates Card -props '{"title":"Hello"}'

  # Render a full page, reading props from stdin
  echo '{"slug":"intro"}' | htmlc page -dir ./templates PostPage -props -

  # List props for a component
  htmlc props -dir ./templates PostCard

Run 'htmlc help <subcommand>' for detailed flags and examples.
`

const helpRender = `render — render a .vue component as an HTML fragment

SYNOPSIS
  htmlc render [-dir <path>] [-props <json|->] <component>

DESCRIPTION
  Renders the named .vue component and writes the resulting HTML fragment to
  stdout. Scoped styles are prepended as a <style> block. The component name
  is matched case-insensitively against files in the component directory.

FLAGS
  -dir string   Directory containing .vue component files. (default ".")
  -props string Props as a JSON object, or "-" to read JSON from stdin.

EXAMPLES
  # Render Button with no props
  htmlc render Button

  # Render Card with inline props from a specific directory
  htmlc render -dir ./templates Card -props '{"title":"Hello","count":3}'

  # Render PostCard with props piped from another command
  echo '{"post":{"title":"Intro"}}' | htmlc render PostCard -props -
`

const helpPage = `page — render a .vue component as a full HTML page

SYNOPSIS
  htmlc page [-dir <path>] [-props <json|->] <component>

DESCRIPTION
  Renders the named .vue component and writes a complete HTML document to
  stdout. The output includes a proper <!DOCTYPE html> wrapper around the
  rendered component. Scoped styles are injected into the document head.
  The component name is matched case-insensitively against files in the
  component directory.

FLAGS
  -dir string   Directory containing .vue component files. (default ".")
  -props string Props as a JSON object, or "-" to read JSON from stdin.

EXAMPLES
  # Render HomePage as a full HTML page
  htmlc page HomePage

  # Render PostPage with props from a specific directory
  htmlc page -dir ./templates PostPage -props '{"slug":"intro","title":"Hello"}'

  # Render with props piped from stdin
  echo '{"slug":"intro"}' | htmlc page -dir ./templates PostPage -props -
`

const helpProps = `props — list the props expected by a component

SYNOPSIS
  htmlc props [-dir <path>] <component>

DESCRIPTION
  Parses the named .vue component and prints the name of each declared prop
  on its own line, sorted alphabetically. This is useful for discovering what
  data a component expects before rendering it.

FLAGS
  -dir string   Directory containing .vue component files. (default ".")

EXAMPLES
  # List props for a component in the current directory
  htmlc props PostCard

  # List props for a component in a specific directory
  htmlc props -dir ./templates Card
`

var subcommandHelp = map[string]string{
	"render": helpRender,
	"page":   helpPage,
	"props":  helpProps,
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, helpTop)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	subcmd := args[0]
	rest := args[1:]

	if subcmd == "--help" || subcmd == "-h" {
		printHelp(stdout)
		return 0
	}

	switch subcmd {
	case "render":
		if err := runRender(rest, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
	case "page":
		if err := runPage(rest, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
	case "props":
		if err := runProps(rest, stdout, stderr); err != nil {
			fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
	case "help":
		return runHelp(rest, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "htmlc: unknown subcommand %q\n\nRun 'htmlc help' to see available subcommands.\n", subcmd)
		return 1
	}
	return 0
}

func runHelp(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}
	name := args[0]
	if h, ok := subcommandHelp[name]; ok {
		fmt.Fprint(stdout, h)
		return 0
	}
	fmt.Fprintf(stderr, "htmlc: unknown subcommand %q\n\nRun 'htmlc help' to see available subcommands.\n", name)
	return 1
}

func parseProps(propsFlag string) (map[string]any, error) {
	if propsFlag == "" {
		return map[string]any{}, nil
	}

	var src []byte
	if propsFlag == "-" {
		var err error
		src, err = io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("reading props from stdin: %w", err)
		}
	} else {
		src = []byte(propsFlag)
	}

	var data map[string]any
	if err := json.Unmarshal(src, &data); err != nil {
		return nil, fmt.Errorf("parsing props JSON: %w", err)
	}
	return data, nil
}

func runRender(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .vue components")
	propsFlag := fs.String("props", "", "props as JSON object string, or - to read from stdin")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpRender)
			return nil
		}
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("render requires a component name")
	}
	name := fs.Arg(0)

	data, err := parseProps(*propsFlag)
	if err != nil {
		return err
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir})
	if err != nil {
		return fmt.Errorf("initializing engine: %w", err)
	}

	return engine.RenderFragment(stdout, name, data)
}

func runPage(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("page", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .vue components")
	propsFlag := fs.String("props", "", "props as JSON object string, or - to read from stdin")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpPage)
			return nil
		}
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("page requires a component name")
	}
	name := fs.Arg(0)

	data, err := parseProps(*propsFlag)
	if err != nil {
		return err
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir})
	if err != nil {
		return fmt.Errorf("initializing engine: %w", err)
	}

	return engine.RenderPage(stdout, name, data)
}

func runProps(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("props", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .vue components")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpProps)
			return nil
		}
		return err
	}
	if fs.NArg() < 1 {
		return fmt.Errorf("props requires a component name")
	}
	name := fs.Arg(0)

	path := filepath.Join(*dir, name+".vue")
	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading component file: %w", err)
	}

	component, err := htmlc.ParseFile(path, string(src))
	if err != nil {
		return fmt.Errorf("parsing component: %w", err)
	}

	props := component.Props()
	names := make([]string, 0, len(props))
	for _, p := range props {
		names = append(names, p.Name)
	}
	sort.Strings(names)

	for _, n := range names {
		fmt.Fprintln(stdout, n)
	}
	return nil
}
