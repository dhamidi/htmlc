package main

import (
	"encoding/json"
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

// camelToScreamingSnake converts camelCase to SCREAMING_SNAKE_CASE.
// E.g.: showDate → SHOW_DATE, postTitle → POST_TITLE.
var camelBoundary = regexp.MustCompile(`([a-z])([A-Z])`)

func camelToScreamingSnake(s string) string {
	s = camelBoundary.ReplaceAllString(s, "${1}_${2}")
	return strings.ToUpper(s)
}

func runProps(args []string, stdout, stderr io.Writer, strict bool) error {
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
