package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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
  htmlc props [-dir <path>] [-format <fmt>] <component>

DESCRIPTION
  Parses the named .vue component and prints the name of each declared prop
  on its own line, sorted alphabetically. This is useful for discovering what
  data a component expects before rendering it.

  The argument may be a component name (looked up in -dir) or a path ending
  in .vue or containing a path separator, which is opened directly.

FLAGS
  -dir string      Directory containing .vue component files. (default ".")
  -format string   Output format: text, json, env (default "text")

EXAMPLES
  # List props for a component in the current directory
  htmlc props PostCard

  # List props for a component in a specific directory
  htmlc props -dir ./templates Card

  # List props as JSON
  htmlc props -dir ./templates Card -format json

  # List props suitable for shell env export
  htmlc props -dir ./templates Card -format env

  # Use a direct file path
  htmlc props ./templates/PostCard.vue
`

var subcommandHelp = map[string]string{
	"render": helpRender,
	"page":   helpPage,
	"props":  helpProps,
}

// errSilent is returned when the error has already been written to stderr.
var errSilent = errors.New("")

// normalizeArgs moves flag tokens before positional tokens so that Go's
// flag.FlagSet can parse interspersed flags like `render Foo -props val`.
// Handles the special "-" value (stdin marker) for value-taking flags.
func normalizeArgs(args []string) []string {
	var flags, positional []string
	i := 0
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && len(arg) > 1 && arg != "-" {
			if strings.Contains(arg, "=") {
				// -flag=value form: keep as-is
				flags = append(flags, arg)
			} else if i+1 < len(args) && (!strings.HasPrefix(args[i+1], "-") || args[i+1] == "-") {
				// -flag value form: take next token as value
				flags = append(flags, arg, args[i+1])
				i += 2
				continue
			} else {
				flags = append(flags, arg)
			}
		} else {
			positional = append(positional, arg)
		}
		i++
	}
	return append(flags, positional...)
}

// cmdErrorMsg formats an actionable error message for a subcommand.
// First line: "htmlc <cmd>: <msg>"; hint lines follow verbatim.
func cmdErrorMsg(cmd, msg string, hints ...string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "htmlc %s: %s", cmd, msg)
	for _, h := range hints {
		sb.WriteString("\n")
		sb.WriteString(h)
	}
	return sb.String()
}

// listComponents returns sorted component names (without .vue) from dir.
// Returns nil if the directory cannot be read (best-effort).
func listComponents(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".vue") {
			names = append(names, strings.TrimSuffix(e.Name(), ".vue"))
		}
	}
	sort.Strings(names)
	return names
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
			if err != errSilent {
				fmt.Fprintln(stderr, err)
			}
			return 1
		}
	case "page":
		if err := runPage(rest, stdout, stderr); err != nil {
			if err != errSilent {
				fmt.Fprintln(stderr, err)
			}
			return 1
		}
	case "props":
		if err := runProps(rest, stdout, stderr); err != nil {
			if err != errSilent {
				fmt.Fprintln(stderr, err)
			}
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

// parseProps parses the props flag value. stdin is used when propsFlag == "-".
// Returns raw errors without wrapping so callers can format them contextually.
func parseProps(propsFlag string, stdin io.Reader) (map[string]any, error) {
	if propsFlag == "" {
		return map[string]any{}, nil
	}

	var src []byte
	if propsFlag == "-" {
		var err error
		src, err = io.ReadAll(stdin)
		if err != nil {
			return nil, err
		}
	} else {
		src = []byte(propsFlag)
	}

	var data map[string]any
	if err := json.Unmarshal(src, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// propsJSONError formats the props JSON error message for the given subcommand.
func propsJSONError(cmd string, fromStdin bool, err error) string {
	desc := "invalid JSON in -props flag"
	if fromStdin {
		desc = "invalid JSON read from stdin"
	}
	return cmdErrorMsg(cmd, desc,
		`  Expected a JSON object, e.g. '{"title":"Hello","count":3}'`,
		fmt.Sprintf("  Parse error: %v", err),
	)
}

// componentNotFoundError formats the "component not found" error message.
func componentNotFoundError(cmd, name, dir string) string {
	components := listComponents(dir)
	hints := []string{
		fmt.Sprintf("  Components are loaded from: %s", dir),
	}
	if len(components) > 0 {
		const maxShow = 10
		shown := components
		extra := 0
		if len(components) > maxShow {
			shown = components[:maxShow]
			extra = len(components) - maxShow
		}
		hints = append(hints, fmt.Sprintf("  Available components: %s", strings.Join(shown, ", ")))
		if extra > 0 {
			hints = append(hints, fmt.Sprintf("  ... and %d more", extra))
		}
		hints = append(hints, "  (listed by scanning *.vue files in the component directory)")
	}
	return cmdErrorMsg(cmd, fmt.Sprintf("component %q not found", name), hints...)
}

func runRender(args []string, stdout, stderr io.Writer) error {
	args = normalizeArgs(args)
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
		fmt.Fprintln(stderr, cmdErrorMsg("render", "missing component name",
			"",
			"USAGE",
			"  htmlc render [-dir <path>] [-props <json|->] <component>",
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

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir})
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("render", fmt.Sprintf("failed to initialise engine: %v", err),
			"  Run 'htmlc help render' for usage.",
		))
		return errSilent
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

func runPage(args []string, stdout, stderr io.Writer) error {
	args = normalizeArgs(args)
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
		fmt.Fprintln(stderr, cmdErrorMsg("page", "missing component name",
			"",
			"USAGE",
			"  htmlc page [-dir <path>] [-props <json|->] <component>",
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

	engine, err := htmlc.New(htmlc.Options{ComponentDir: *dir})
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("page", fmt.Sprintf("failed to initialise engine: %v", err),
			"  Run 'htmlc help page' for usage.",
		))
		return errSilent
	}

	if err := engine.RenderPage(stdout, name, data); err != nil {
		if strings.Contains(err.Error(), name) {
			fmt.Fprintln(stderr, componentNotFoundError("page", name, *dir))
		} else {
			fmt.Fprintln(stderr, cmdErrorMsg("page", err.Error()))
		}
		return errSilent
	}
	return nil
}

// camelToScreamingSnake converts camelCase to SCREAMING_SNAKE_CASE.
// E.g.: showDate → SHOW_DATE, postTitle → POST_TITLE.
var camelBoundary = regexp.MustCompile(`([a-z])([A-Z])`)

func camelToScreamingSnake(s string) string {
	s = camelBoundary.ReplaceAllString(s, "${1}_${2}")
	return strings.ToUpper(s)
}

// isTerminal reports whether w is a character device (TTY).
func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func runProps(args []string, stdout, stderr io.Writer) error {
	args = normalizeArgs(args)
	fs := flag.NewFlagSet("props", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing .vue components")
	format := fs.String("format", "text", "output format: text, json, env")
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

	switch *format {
	case "text", "json", "env":
	default:
		fmt.Fprintln(stderr, cmdErrorMsg("props", fmt.Sprintf("unknown format %q", *format),
			"  Supported formats: text, json, env",
		))
		return errSilent
	}

	name := fs.Arg(0)

	// Detect path-style argument (direct file path).
	var path, componentName string
	isPathStyle := strings.HasSuffix(name, ".vue") || strings.ContainsRune(name, os.PathSeparator) || strings.Contains(name, "/")
	if isPathStyle {
		path = name
		base := filepath.Base(name)
		componentName = strings.TrimSuffix(base, ".vue")
	} else {
		path = filepath.Join(*dir, name+".vue")
		componentName = name
	}

	src, err := os.ReadFile(path)
	if err != nil {
		if isPathStyle {
			fmt.Fprintln(stderr, cmdErrorMsg("props", fmt.Sprintf("file %q not found", name),
				"",
				"  Tip: provide the component name without the path or .vue extension.",
				fmt.Sprintf("  For example, use %q instead of %q.", componentName, name),
			))
		} else {
			hints := []string{
				fmt.Sprintf("  Expected file: %s", path),
				"  Run 'htmlc props -dir <path> <name>' with the correct directory.",
			}
			fmt.Fprintln(stderr, cmdErrorMsg("props", fmt.Sprintf("component %q not found in %q", name, *dir), hints...))
		}
		return errSilent
	}

	component, err := htmlc.ParseFile(path, string(src))
	if err != nil {
		return fmt.Errorf("parsing component: %w", err)
	}

	props := component.Props()
	sort.Slice(props, func(i, j int) bool { return props[i].Name < props[j].Name })

	switch *format {
	case "json":
		type propJSON struct {
			Name string `json:"name"`
			Expr string `json:"expr"`
		}
		type outputJSON struct {
			Component string     `json:"component"`
			Props     []propJSON `json:"props"`
		}
		out := outputJSON{Component: componentName, Props: []propJSON{}}
		for _, p := range props {
			expr := ""
			if len(p.Expressions) > 0 {
				expr = p.Expressions[0]
			}
			out.Props = append(out.Props, propJSON{Name: p.Name, Expr: expr})
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Fprintf(stdout, "%s\n", data)

	case "env":
		type envEntry struct {
			envName string
		}
		entries := make([]envEntry, len(props))
		for i, p := range props {
			entries[i] = envEntry{envName: camelToScreamingSnake(p.Name)}
		}
		sort.Slice(entries, func(i, j int) bool { return entries[i].envName < entries[j].envName })
		for _, e := range entries {
			fmt.Fprintf(stdout, "%s=\n", e.envName)
		}

	default: // text
		for _, p := range props {
			fmt.Fprintln(stdout, p.Name)
		}
		if isTerminal(stdout) {
			line := strings.Repeat("─", 5)
			fmt.Fprintln(stdout, line)
			fmt.Fprintf(stdout, "%d props\n", len(props))
		}
	}

	return nil
}
