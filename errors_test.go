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

// TestParseError_Snippet verifies that snippet-related formatting is robust
// for short sources and boundary line positions, and that errors.As works.
func TestParseError_Snippet(t *testing.T) {
	// A single-line source must not cause an index out-of-bounds panic inside
	// snippet().  This is a useful edge case because the 3-line context window
	// starts at line-2 which would be negative without the clamp.
	t.Run("single-line source does not panic", func(t *testing.T) {
		src := "bad content"
		snip := snippet(src, 1)
		if !strings.Contains(snip, "bad content") {
			t.Errorf("snippet should include the source line, got %q", snip)
		}
	})

	// When the error is on line 1 the snippet has no preceding context lines.
	// Without the clamp `start = line-2` would be -1.
	t.Run("error on first line has no preceding context", func(t *testing.T) {
		src := "line1\nline2\nline3"
		snip := snippet(src, 1)
		// snippet window: start=max(0,-1)=0, end=min(3,2)=2 → lines 1-2 only.
		if strings.Contains(snip, "line3") {
			t.Errorf("snippet for line 1 should not include line3, got %q", snip)
		}
	})

	// When the error is on the last line the snippet has no following context.
	// Without the clamp `end = line+1` would exceed the slice length.
	t.Run("error on last line has no following context", func(t *testing.T) {
		src := "line1\nline2\nline3"
		snip := snippet(src, 3)
		// snippet window: start=1, end=min(3,3)=3 → lines 2-3 only.
		if strings.Contains(snip, "line1") {
			t.Errorf("snippet for line 3 should not include line1, got %q", snip)
		}
	})

	// ParseError.Error() must embed the file name and line number when a
	// Location is present so the user can navigate to the source position.
	t.Run("ParseError Error includes file and line number", func(t *testing.T) {
		e := &ParseError{
			Path: "foo.vue",
			Msg:  "syntax error",
			Location: &SourceLocation{
				File:   "foo.vue",
				Line:   7,
				Column: 3,
			},
		}
		got := e.Error()
		if !strings.Contains(got, "foo.vue") {
			t.Errorf("ParseError.Error() %q: should contain file name", got)
		}
		if !strings.Contains(got, "7") {
			t.Errorf("ParseError.Error() %q: should contain line number 7", got)
		}
	})

	// errors.As must succeed so callers can inspect the concrete *ParseError.
	t.Run("errors.As succeeds for *ParseError", func(t *testing.T) {
		inner := &ParseError{Path: "x.vue", Msg: "oops"}
		var target *ParseError
		if !errors.As(inner, &target) {
			t.Error("errors.As(*ParseError): expected true, got false")
		}
		if target.Path != "x.vue" {
			t.Errorf("target.Path = %q, want %q", target.Path, "x.vue")
		}
	})
}

// TestRenderError_Snippet verifies RenderError formatting for edge cases.
func TestRenderError_Snippet(t *testing.T) {
	// A RenderError whose Location points to line 1 of a single-line source
	// must not panic and must still include the component name in Error().
	t.Run("short snippet does not panic", func(t *testing.T) {
		e := &RenderError{
			Component: "Card",
			Expr:      "x",
			Wrapped:   errors.New("err"),
			Location: &SourceLocation{
				File:    "Card.vue",
				Line:    1,
				Column:  1,
				Snippet: "  1 | <p>{{ x }}</p>\n",
			},
		}
		got := e.Error()
		if !strings.Contains(got, "Card") {
			t.Errorf("RenderError.Error() %q: should contain component name", got)
		}
		if !strings.Contains(got, "1") {
			t.Errorf("RenderError.Error() %q: should contain line number 1", got)
		}
	})

	// errors.As must succeed so callers can inspect the concrete *RenderError.
	t.Run("errors.As succeeds for *RenderError", func(t *testing.T) {
		inner := &RenderError{Component: "C", Wrapped: errors.New("e")}
		var target *RenderError
		if !errors.As(inner, &target) {
			t.Error("errors.As(*RenderError): expected true, got false")
		}
		if target.Component != "C" {
			t.Errorf("target.Component = %q, want %q", target.Component, "C")
		}
	})
}
