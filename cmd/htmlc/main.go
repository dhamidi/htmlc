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
	"golang.org/x/net/html"
)

const helpTop = `htmlc — server-side Vue.js component renderer

USAGE
  htmlc <subcommand> [flags] [args]

SUBCOMMANDS
  render   Render a component as an HTML fragment (stdout)
  page     Render a component as a full HTML page (stdout)
  props    List the props expected by a component
  ast      Print the template AST of a component (stdout)
  help     Show help for a subcommand

EXAMPLES
  # Render a fragment with inline props
  htmlc render -dir ./templates Card -props '{"title":"Hello"}'

  # Render a full page, reading props from stdin
  echo '{"slug":"intro"}' | htmlc page -dir ./templates PostPage -props -

  # List props for a component
  htmlc props -dir ./templates PostCard

  # Print the template AST of a component
  htmlc ast -dir ./templates PostPage

Run 'htmlc help <subcommand>' for detailed flags and examples.
`

const helpRender = `render — render a .vue component as an HTML fragment

SYNOPSIS
  htmlc render [-dir <path>] [-props <json|->] [-debug] <component>

DESCRIPTION
  Renders the named .vue component and writes the resulting HTML fragment to
  stdout. Scoped styles are prepended as a <style> block. The component name
  is matched case-insensitively against files in the component directory.

FLAGS
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

const helpBuild = `build — render a page tree to an output directory

SYNOPSIS
  htmlc build [-dir <path>] [-pages <path>] [-out <path>] [-layout <name>] [-debug]

DESCRIPTION
  Walks the pages directory recursively, renders every .vue file as a full
  HTML page, and writes the results to the output directory. The directory
  hierarchy is preserved: pages/posts/hello.vue becomes out/posts/hello.html.

  Props for each page are loaded by merging JSON data files in order:

    1. pages/_data.json          — root defaults (all pages)
    2. pages/subdir/_data.json   — subdirectory defaults (pages in that dir)
    3. pages/subdir/hello.json   — page-level props (highest priority)

  Each level is shallow-merged so page-level values always win. If no data
  files exist the page is rendered with no props.

FLAGS
  -dir string     Directory containing shared .vue components. (default ".")
  -pages string   Root of the page tree. (default "./pages")
  -out string     Output directory. Created if it does not exist. (default "./out")
  -layout string  Layout component (from -dir) to wrap every page. (default: none)
  -debug          Annotate output with diagnostic HTML comments.

EXAMPLES
  # Build all pages using defaults
  htmlc build

  # Build with an explicit component dir, pages dir, and output dir
  htmlc build -dir ./templates -pages ./pages -out ./dist

  # Build with a shared layout
  htmlc build -dir ./templates -pages ./pages -out ./dist -layout AppLayout
`

const helpPage = `page — render a .vue component as a full HTML page

SYNOPSIS
  htmlc page [-dir <path>] [-props <json|->] [-debug] [-layout <component>] <component>

DESCRIPTION
  Renders the named .vue component and writes a complete HTML document to
  stdout. The output includes a proper <!DOCTYPE html> wrapper around the
  rendered component. Scoped styles are injected into the document head.
  The component name is matched case-insensitively against files in the
  component directory.

FLAGS
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

const helpAst = `ast — print the template AST of a .vue component

SYNOPSIS
  htmlc ast [-dir <path>] <component>

DESCRIPTION
  Parses the named .vue component and pretty-prints its template AST as
  indented pseudo-XML to stdout. This is useful for understanding how the
  parser sees the template without executing the render pipeline.

FLAGS
  -dir string   Directory containing .vue component files. (default ".")

EXAMPLES
  # Print the AST of PostPage
  htmlc ast -dir ./templates PostPage

  # Print the AST from the current directory
  htmlc ast Card
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
	"ast":    helpAst,
	"build":  helpBuild,
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
	case "ast":
		if err := runAst(rest, stdout, stderr); err != nil {
			if err != errSilent {
				fmt.Fprintln(stderr, err)
			}
			return 1
		}
	case "build":
		if err := runBuild(rest, stdout, stderr); err != nil {
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
			"  htmlc render [-dir <path>] [-props <json|->] [-debug] <component>",
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
			"  htmlc page [-dir <path>] [-props <json|->] [-debug] [-layout <component>] <component>",
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

// pageEntry describes a single page found during page discovery.
type pageEntry struct {
	// relPath is the path relative to the pages root, e.g. "posts/hello.vue"
	relPath string
	// absPath is the absolute path to the .vue file
	absPath string
	// dataPath is the path to the matching .json data file, or "" if none
	dataPath string
	// outPath is the resolved output path relative to -out, e.g. "posts/hello.html"
	outPath string
}

// discoverPages walks pagesDir recursively and returns a sorted slice of
// pageEntry for every .vue file found. Files whose base name starts with "_"
// are skipped (they are treated as shared partials, not pages).
func discoverPages(pagesDir string) ([]pageEntry, error) {
	var entries []pageEntry
	err := filepath.WalkDir(pagesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".vue" {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasPrefix(base, "_") {
			return nil
		}
		rel, err := filepath.Rel(pagesDir, path)
		if err != nil {
			return err
		}
		outPath := strings.TrimSuffix(rel, ".vue") + ".html"
		dataPath := strings.TrimSuffix(path, ".vue") + ".json"
		if _, statErr := os.Stat(dataPath); statErr != nil {
			dataPath = ""
		}
		entries = append(entries, pageEntry{
			relPath:  rel,
			absPath:  path,
			dataPath: dataPath,
			outPath:  outPath,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].relPath < entries[j].relPath
	})
	return entries, nil
}

// readJSONFile reads path and unmarshals its contents into a map.
// It returns a descriptive error if the file contains invalid JSON.
func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("%s: invalid JSON: %w", path, err)
	}
	return m, nil
}

// loadPageData loads and shallow-merges props for entry.
//
// It collects ancestor _data.json files from pagesRoot down to the page's
// parent directory (inclusive), then the page's own .json file
// (entry.dataPath), and shallow-merges them in that order so that
// page-level values take highest priority.
//
// Missing files are silently skipped. An error is returned only when a
// file that exists contains invalid JSON.
func loadPageData(entry pageEntry, pagesRoot string) (map[string]any, error) {
	pageDir := filepath.Dir(entry.absPath)
	rel, err := filepath.Rel(pagesRoot, pageDir)
	if err != nil {
		return nil, fmt.Errorf("resolving page directory: %w", err)
	}

	result := map[string]any{}

	// Collect _data.json paths from pagesRoot down to pageDir.
	// First check pagesRoot itself, then descend one component at a time.
	var dataDirs []string
	dataDirs = append(dataDirs, pagesRoot)
	if rel != "." {
		current := pagesRoot
		for _, part := range strings.Split(rel, string(filepath.Separator)) {
			current = filepath.Join(current, part)
			dataDirs = append(dataDirs, current)
		}
	}

	for _, dir := range dataDirs {
		candidate := filepath.Join(dir, "_data.json")
		if _, statErr := os.Stat(candidate); statErr != nil {
			continue
		}
		m, err := readJSONFile(candidate)
		if err != nil {
			return nil, err
		}
		for k, v := range m {
			result[k] = v
		}
	}

	// Apply page-level data last (highest priority).
	if entry.dataPath != "" {
		m, err := readJSONFile(entry.dataPath)
		if err != nil {
			return nil, err
		}
		for k, v := range m {
			result[k] = v
		}
	}

	return result, nil
}

func runBuild(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("dir", ".", "directory containing shared .vue components")
	pages := fs.String("pages", "./pages", "root of the page tree")
	out := fs.String("out", "./out", "output directory")
	_ = fs.String("layout", "", "layout component to wrap every page")
	_ = fs.Bool("debug", false, "enable debug render mode")
	if err := fs.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpBuild)
			return nil
		}
		return err
	}

	if _, statErr := os.Stat(*pages); statErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("cannot find pages directory %q", *pages),
			"  The pages directory does not exist. Create it and add .vue page files.",
			"",
			"  EXAMPLE",
			"    mkdir pages",
			fmt.Sprintf("    htmlc build -pages %s", *pages),
		))
		return errSilent
	}

	if _, statErr := os.Stat(*dir); statErr != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("cannot load components from %q", *dir),
			"  No such directory. Create the directory and add .vue component files.",
		))
		return errSilent
	}

	discovered, err := discoverPages(*pages)
	if err != nil {
		fmt.Fprintln(stderr, cmdErrorMsg("build", fmt.Sprintf("page discovery failed: %v", err)))
		return errSilent
	}

	_ = out
	failed := 0
	for _, e := range discovered {
		data, err := loadPageData(e, *pages)
		if err != nil {
			fmt.Fprintf(stderr, "htmlc build: %s: failed to load data: %v\n", e.relPath, err)
			failed++
			continue
		}
		_ = data
		fmt.Fprintf(stdout, "%s → %s\n", e.relPath, e.outPath)
	}
	if failed > 0 {
		fmt.Fprintf(stderr, "htmlc build: %d page(s) failed; see errors above.\n", failed)
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

// printASTNode recursively prints a node from a parsed template AST as
// indented pseudo-XML to w. The depth parameter controls indentation level.
func printASTNode(w io.Writer, n *html.Node, depth int) {
	indent := strings.Repeat("  ", depth)
	switch n.Type {
	case html.DocumentNode:
		fmt.Fprintln(w, "Document")
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			printASTNode(w, child, depth+1)
		}
	case html.ElementNode:
		var attrs []string
		var directives []string
		for _, attr := range n.Attr {
			if strings.HasPrefix(attr.Key, "v-") || strings.HasPrefix(attr.Key, ":") || strings.HasPrefix(attr.Key, "@") || strings.HasPrefix(attr.Key, "#") {
				if attr.Val != "" {
					directives = append(directives, fmt.Sprintf("%s=%q", attr.Key, attr.Val))
				} else {
					directives = append(directives, attr.Key)
				}
			} else {
				attrs = append(attrs, fmt.Sprintf("%s=%q", attr.Key, attr.Val))
			}
		}
		line := fmt.Sprintf("%sElement[%s]", indent, n.Data)
		if len(directives) > 0 {
			line += " " + strings.Join(directives, " ")
		}
		line += fmt.Sprintf(" attrs=%v", attrs)
		fmt.Fprintln(w, line)
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			printASTNode(w, child, depth+1)
		}
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text != "" {
			fmt.Fprintf(w, "%sText: %q\n", indent, text)
		}
	case html.CommentNode:
		fmt.Fprintf(w, "%sComment: %q\n", indent, strings.TrimSpace(n.Data))
	case html.DoctypeNode:
		fmt.Fprintf(w, "%sDoctype: %s\n", indent, n.Data)
	}
}

func runAst(args []string, stdout, stderr io.Writer) error {
	args = normalizeArgs(args)
	fset := flag.NewFlagSet("ast", flag.ContinueOnError)
	fset.SetOutput(stderr)
	dir := fset.String("dir", ".", "directory containing .vue components")
	if err := fset.Parse(args); err != nil {
		if err == flag.ErrHelp {
			fmt.Fprint(stdout, helpAst)
			return nil
		}
		return err
	}
	if fset.NArg() < 1 {
		fmt.Fprintln(stderr, cmdErrorMsg("ast", "missing component name",
			"",
			"USAGE",
			"  htmlc ast [-dir <path>] <component>",
			"",
			"EXAMPLE",
			"  htmlc ast -dir ./templates MyComponent",
		))
		return errSilent
	}
	name := fset.Arg(0)

	// Detect path-style argument (direct file path).
	var path string
	isPathStyle := strings.HasSuffix(name, ".vue") || strings.ContainsRune(name, os.PathSeparator) || strings.Contains(name, "/")
	if isPathStyle {
		path = name
	} else {
		path = filepath.Join(*dir, name+".vue")
	}

	src, err := os.ReadFile(path)
	if err != nil {
		if isPathStyle {
			fmt.Fprintln(stderr, cmdErrorMsg("ast", fmt.Sprintf("file %q not found", name)))
		} else {
			fmt.Fprintln(stderr, cmdErrorMsg("ast", fmt.Sprintf("component %q not found in %q", name, *dir),
				fmt.Sprintf("  Expected file: %s", path),
			))
		}
		return errSilent
	}

	component, err := htmlc.ParseFile(path, string(src))
	if err != nil {
		return fmt.Errorf("parsing component: %w", err)
	}

	if component.Template == nil {
		fmt.Fprintln(stderr, cmdErrorMsg("ast", "component has no template"))
		return errSilent
	}

	printASTNode(stdout, component.Template, 0)
	return nil
}
