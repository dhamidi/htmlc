package htmlc

import (
	"fmt"
	"regexp"
	"strings"
)

// ExprKind classifies a template expression string.
type ExprKind int

const (
	ExprSimpleIdent ExprKind = iota // "name"
	ExprDotPath                     // "a.b.c"
	ExprComplex                     // anything else — errors on conversion
)

var (
	simpleIdentRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	dotPathRe     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)+$`)
)

// ClassifyExpr inspects expr and returns its kind.
// "." alone is treated as ExprSimpleIdent (maps to "." in html/template).
func ClassifyExpr(expr string) ExprKind {
	if expr == "." {
		return ExprSimpleIdent
	}
	if simpleIdentRe.MatchString(expr) {
		return ExprSimpleIdent
	}
	if dotPathRe.MatchString(expr) {
		return ExprDotPath
	}
	return ExprComplex
}

// DotPrefix converts an htmlc expression to a Go template dot-accessor.
//
//	"name"   → ".name"
//	"a.b.c"  → ".a.b.c"
//	"."      → "."
//
// Returns an error for ExprComplex inputs.
func DotPrefix(expr string) (string, error) {
	switch ClassifyExpr(expr) {
	case ExprSimpleIdent:
		if expr == "." {
			return ".", nil
		}
		return "." + expr, nil
	case ExprDotPath:
		return "." + expr, nil
	default:
		return "", &ConversionError{
			Message: fmt.Sprintf("complex expression %q cannot be converted to a Go template dot-accessor", expr),
		}
	}
}

// Snippet returns a ≈3-line context window around line (1-based) in src,
// with a ">" marker on the target line — matches existing htmlc error style.
func Snippet(src string, line int) string {
	lines := strings.Split(src, "\n")
	start := line - 2
	if start < 0 {
		start = 0
	}
	end := line + 1
	if end > len(lines) {
		end = len(lines)
	}
	var b strings.Builder
	for i := start; i < end; i++ {
		marker := "  "
		if i+1 == line {
			marker = "> "
		}
		fmt.Fprintf(&b, "%s%3d | %s\n", marker, i+1, lines[i])
	}
	return b.String()
}
