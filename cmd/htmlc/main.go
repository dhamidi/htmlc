package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

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

// parseProps parses the props flag value. stdin is used when propsFlag == "-".
// Returns raw errors without wrapping so callers can format them contextually.
func parseProps(propsFlag string, stdin io.Reader) (map[string]any, error) {
	if propsFlag == "" {
		return map[string]any{}, nil
	}
	src := []byte(propsFlag)
	if propsFlag == "-" {
		var err error
		if src, err = io.ReadAll(stdin); err != nil {
			return nil, err
		}
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
	hints := []string{fmt.Sprintf("  Components are loaded from: %s", dir)}
	if components := listComponents(dir); len(components) > 0 {
		const maxShow = 10
		shown, extra := components, 0
		if len(components) > maxShow {
			shown, extra = components[:maxShow], len(components)-maxShow
		}
		hints = append(hints, fmt.Sprintf("  Available components: %s", strings.Join(shown, ", ")))
		if extra > 0 {
			hints = append(hints, fmt.Sprintf("  ... and %d more", extra))
		}
		hints = append(hints, "  (listed by scanning *.vue files in the component directory)")
	}
	return cmdErrorMsg(cmd, fmt.Sprintf("component %q not found", name), hints...)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	// Pre-scan for -strict / --strict before dispatching to subcommands so
	// that the flag can appear in any position (e.g. before or after the
	// subcommand name).
	strictMode := false
	filteredArgs := args[:0:0]
	for i, a := range args {
		if a == "-strict" || a == "--strict" {
			strictMode = true
			continue
		}
		filteredArgs = append(filteredArgs, args[i])
	}
	args = filteredArgs

	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	subcmd, rest := args[0], args[1:]

	if subcmd == "--help" || subcmd == "-h" {
		printHelp(stdout)
		return 0
	}
	if subcmd == "help" {
		return runHelp(rest, stdout, stderr)
	}

	type cmdFn func([]string, io.Writer, io.Writer, bool) error
	cmds := map[string]cmdFn{
		"render":   runRender,
		"page":     runPage,
		"props":    runProps,
		"ast":      runAst,
		"build":    runBuild,
		"template": runTemplate,
	}
	fn, ok := cmds[subcmd]
	if !ok {
		fmt.Fprintf(stderr, "htmlc: unknown subcommand %q\n\nRun 'htmlc help' to see available subcommands.\n", subcmd)
		return 1
	}
	if err := fn(rest, stdout, stderr, strictMode); err != nil {
		if err != errSilent {
			fmt.Fprintln(stderr, err)
		}
		return 1
	}
	return 0
}
