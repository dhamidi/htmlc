package main

import (
	_ "embed"
	"fmt"
	"io"
)

//go:embed README.md
var readmeContent string

var subcommandHelp = map[string]string{
	"render": helpRender,
	"page":   helpPage,
	"props":  helpProps,
	"ast":    helpAst,
	"build":  helpBuild,
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, readmeContent)
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
