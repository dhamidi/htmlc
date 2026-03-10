package htmlc

import (
	"errors"
	"strings"
	"testing"
)

func TestParseError_WithLocation(t *testing.T) {
	loc := &SourceLocation{
		File:    "Card.vue",
		Line:    14,
		Column:  5,
		Snippet: "  13 | <div>\n> 14 | {{ bad }}\n  15 | </div>\n",
	}
	e := &ParseError{Path: "Card.vue", Msg: "unexpected token", Location: loc}
	got := e.Error()
	if !strings.Contains(got, "Card.vue:14:5") {
		t.Errorf("error %q should contain file:line:col", got)
	}
	if !strings.Contains(got, "unexpected token") {
		t.Errorf("error %q should contain the message", got)
	}
	if !strings.Contains(got, loc.Snippet) {
		t.Errorf("error %q should contain snippet", got)
	}
}

func TestParseError_WithoutLocation(t *testing.T) {
	e := &ParseError{Path: "Card.vue", Msg: "missing template"}
	got := e.Error()
	if !strings.HasPrefix(got, "htmlc: parse Card.vue:") {
		t.Errorf("error %q should use legacy format without location", got)
	}
	if strings.Contains(got, ":0:") {
		t.Errorf("error %q should not contain zero line number", got)
	}
}

func TestParseError_NilLocation_FallsBack(t *testing.T) {
	e := &ParseError{Path: "x.vue", Msg: "problem", Location: nil}
	got := e.Error()
	if !strings.Contains(got, "htmlc: parse x.vue: problem") {
		t.Errorf("error %q should fall back to legacy format", got)
	}
}

func TestParseError_ZeroLineFallsBack(t *testing.T) {
	// Location present but Line == 0: should fall back to legacy format.
	e := &ParseError{
		Path:     "x.vue",
		Msg:      "problem",
		Location: &SourceLocation{File: "x.vue", Line: 0},
	}
	got := e.Error()
	if !strings.Contains(got, "htmlc: parse x.vue: problem") {
		t.Errorf("error %q should fall back to legacy format when Line==0", got)
	}
}

func TestRenderError_WithLocation(t *testing.T) {
	loc := &SourceLocation{
		File:    "Card.vue",
		Line:    14,
		Column:  5,
		Snippet: "  13 | <div>\n> 14 | {{ post.Title }}\n  15 | </div>\n",
	}
	e := &RenderError{
		Component: "Card.vue",
		Expr:      "post.Title",
		Wrapped:   errors.New("identifier not found: post"),
		Location:  loc,
	}
	got := e.Error()
	if !strings.Contains(got, "Card.vue:14:") {
		t.Errorf("error %q should contain file:line", got)
	}
	if !strings.Contains(got, `"post.Title"`) {
		t.Errorf("error %q should contain the expression", got)
	}
	if !strings.Contains(got, "identifier not found: post") {
		t.Errorf("error %q should contain wrapped error", got)
	}
	if !strings.Contains(got, loc.Snippet) {
		t.Errorf("error %q should contain snippet", got)
	}
}

func TestRenderError_WithoutLocation(t *testing.T) {
	e := &RenderError{
		Component: "Card",
		Expr:      "title",
		Wrapped:   errors.New("identifier not found: title"),
	}
	got := e.Error()
	if !strings.Contains(got, "render Card") {
		t.Errorf("error %q should contain component name", got)
	}
	if !strings.Contains(got, `"title"`) {
		t.Errorf("error %q should contain the expression", got)
	}
}

func TestRenderError_NoExpr(t *testing.T) {
	e := &RenderError{
		Component: "Card",
		Wrapped:   errors.New("something broke"),
	}
	got := e.Error()
	if !strings.Contains(got, "render Card: something broke") {
		t.Errorf("error %q should use no-expr format", got)
	}
}

func TestRenderError_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	e := &RenderError{Component: "C", Wrapped: inner}
	if errors.Unwrap(e) != inner {
		t.Errorf("Unwrap should return the wrapped error")
	}
}
