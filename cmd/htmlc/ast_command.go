package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhamidi/htmlc"
	"golang.org/x/net/html"
)

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

func runAst(args []string, stdout, stderr io.Writer, strict bool) error {
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
