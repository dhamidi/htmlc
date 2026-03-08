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

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: htmlc <render|page|props> [flags] <component>\n")
		os.Exit(1)
	}

	subcmd := os.Args[1]
	args := os.Args[2:]

	switch subcmd {
	case "render":
		if err := runRender(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "page":
		if err := runPage(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "props":
		if err := runProps(args); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n", subcmd)
		os.Exit(1)
	}
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

func runRender(args []string) error {
	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	dir := fs.String("dir", ".", "directory containing .vue components")
	propsFlag := fs.String("props", "", "props as JSON object string, or - to read from stdin")
	if err := fs.Parse(args); err != nil {
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

	return engine.RenderFragment(os.Stdout, name, data)
}

func runPage(args []string) error {
	fs := flag.NewFlagSet("page", flag.ContinueOnError)
	dir := fs.String("dir", ".", "directory containing .vue components")
	propsFlag := fs.String("props", "", "props as JSON object string, or - to read from stdin")
	if err := fs.Parse(args); err != nil {
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

	return engine.RenderPage(os.Stdout, name, data)
}

func runProps(args []string) error {
	fs := flag.NewFlagSet("props", flag.ContinueOnError)
	dir := fs.String("dir", ".", "directory containing .vue components")
	if err := fs.Parse(args); err != nil {
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
		fmt.Println(n)
	}
	return nil
}
