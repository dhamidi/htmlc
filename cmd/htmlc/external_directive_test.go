package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dhamidi/htmlc"
	"golang.org/x/net/html"
)

// testdataPath returns the absolute path to a file in the testdata directory.
func testdataPath(t *testing.T, name string) string {
	t.Helper()
	abs, err := filepath.Abs(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("abs path: %v", err)
	}
	return abs
}

// newTestDirective creates a started externalDirective backed by the given script path.
func newTestDirective(t *testing.T, name, path string) *externalDirective {
	t.Helper()
	var stderr bytes.Buffer
	ed := &externalDirective{name: name, path: path, stderr: &stderr}
	if err := ed.start(); err != nil {
		t.Fatalf("start directive: %v", err)
	}
	t.Cleanup(func() { ed.stop() })
	return ed
}

// makeNode creates a simple HTML element node with the given tag and attributes.
func makeNode(tag string, attrs ...html.Attribute) *html.Node {
	return &html.Node{
		Type: html.ElementNode,
		Data: tag,
		Attr: attrs,
	}
}

// makeNodeWithText creates an element node with a single text child.
func makeNodeWithText(tag, text string) *html.Node {
	parent := makeNode(tag)
	child := &html.Node{Type: html.TextNode, Data: text}
	parent.FirstChild = child
	return parent
}

func TestExternalDirective_CreatedMutatesAttrs(t *testing.T) {
	ed := newTestDirective(t, "echo", testdataPath(t, "v-echo"))

	node := makeNode("p")
	binding := htmlc.DirectiveBinding{Value: "x", RawExpr: `"x"`}
	ctx := htmlc.DirectiveContext{}

	if err := ed.Created(node, binding, ctx); err != nil {
		t.Fatalf("Created: %v", err)
	}

	found := false
	for _, a := range node.Attr {
		if a.Key == "data-echo" && a.Val == "true" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected data-echo=true in attrs, got %v", node.Attr)
	}
}

func TestExternalDirective_CreatedWithInnerHTML(t *testing.T) {
	ed := newTestDirective(t, "echo", testdataPath(t, "v-echo"))

	node := makeNode("pre")
	binding := htmlc.DirectiveBinding{Value: "inner_html:<b>hello</b>", RawExpr: `"inner_html:<b>hello</b>"`}
	ctx := htmlc.DirectiveContext{}

	if err := ed.Created(node, binding, ctx); err != nil {
		t.Fatalf("Created: %v", err)
	}

	inner, ok := ed.InnerHTML()
	if !ok {
		t.Fatal("InnerHTML() should return ok=true after Created with inner_html")
	}
	if inner != "<b>hello</b>" {
		t.Errorf("InnerHTML() = %q, want %q", inner, "<b>hello</b>")
	}

	// Second call should return empty (field is cleared).
	inner2, ok2 := ed.InnerHTML()
	if ok2 || inner2 != "" {
		t.Errorf("InnerHTML() second call: got (%q, %v), want (\"\", false)", inner2, ok2)
	}
}

func TestExternalDirective_MountedInjectsHTML(t *testing.T) {
	ed := newTestDirective(t, "echo", testdataPath(t, "v-echo"))

	node := makeNode("div")
	binding := htmlc.DirectiveBinding{Value: "x", RawExpr: `"x"`}
	ctx := htmlc.DirectiveContext{}

	var out bytes.Buffer
	if err := ed.Mounted(&out, node, binding, ctx); err != nil {
		t.Fatalf("Mounted: %v", err)
	}

	if !strings.Contains(out.String(), "<!--mounted-->") {
		t.Errorf("Mounted output = %q, expected to contain <!--mounted-->", out.String())
	}
}

func TestExternalDirective_InvalidJSONResponse(t *testing.T) {
	// Script that outputs invalid JSON.
	dir := t.TempDir()
	script := filepath.Join(dir, "v-bad")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho 'not json'\n"), 0755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var stderrBuf bytes.Buffer
	ed := &externalDirective{name: "bad", path: script, stderr: &stderrBuf}
	if err := ed.start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	defer ed.stop()

	node := makeNode("p")
	binding := htmlc.DirectiveBinding{Value: "x"}
	ctx := htmlc.DirectiveContext{}

	// Should not return error (treated as no-op, warning logged to stderr).
	err := ed.Created(node, binding, ctx)
	if err != nil {
		t.Errorf("Created returned error %v, want nil (no-op on bad response)", err)
	}
	if stderrBuf.Len() == 0 {
		t.Errorf("expected warning on stderr for invalid JSON, got nothing")
	}
}

func TestExternalDirective_ExtractTextContent(t *testing.T) {
	parent := makeNodeWithText("pre", "hello world")
	text := extractTextContent(parent)
	if text != "hello world" {
		t.Errorf("extractTextContent = %q, want %q", text, "hello world")
	}
}

func TestExternalDirective_ExtractTextContentNested(t *testing.T) {
	parent := makeNode("div")
	child1 := &html.Node{Type: html.TextNode, Data: "foo"}
	inner := makeNode("span")
	child2 := &html.Node{Type: html.TextNode, Data: "bar"}
	inner.FirstChild = child2
	parent.FirstChild = child1
	child1.NextSibling = inner

	text := extractTextContent(parent)
	if text != "foobar" {
		t.Errorf("extractTextContent = %q, want %q", text, "foobar")
	}
}
