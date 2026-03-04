package htmlc

import (
	"sort"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

const fullSFC = `<template>
  <div class="hello">
    <h1>{{ msg }}</h1>
  </div>
</template>

<script>
export default {
  data() { return { msg: "hello" } }
}
</script>

<style scoped>
.hello { color: red; }
</style>
`

const templateOnly = `<template>
  <p>Simple</p>
</template>
`

const scopedFalse = `<template><span>x</span></template>
<style>.a { color: blue; }</style>
`

const unclosedTemplate = `<template>
  <div>no closing tag
`

func TestParseFile_AllSections(t *testing.T) {
	c, err := ParseFile("test.vue", fullSFC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Path != "test.vue" {
		t.Errorf("path = %q, want %q", c.Path, "test.vue")
	}

	// Script content should contain "export default"
	if !strings.Contains(c.Script, "export default") {
		t.Errorf("script = %q, want it to contain 'export default'", c.Script)
	}

	// Style content should contain ".hello"
	if !strings.Contains(c.Style, ".hello") {
		t.Errorf("style = %q, want it to contain '.hello'", c.Style)
	}

	// Scoped should be true
	if !c.Scoped {
		t.Errorf("scoped = false, want true")
	}

	// Template should be parsed into a node tree
	if c.Template == nil {
		t.Fatal("template is nil")
	}
}

func TestParseFile_TemplateOnly(t *testing.T) {
	c, err := ParseFile("simple.vue", templateOnly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if c.Script != "" {
		t.Errorf("script = %q, want empty", c.Script)
	}
	if c.Style != "" {
		t.Errorf("style = %q, want empty", c.Style)
	}
	if c.Scoped {
		t.Errorf("scoped = true, want false")
	}
	if c.Template == nil {
		t.Fatal("template is nil")
	}
}

func TestParseFile_ScopedAttributeDetected(t *testing.T) {
	c, err := ParseFile("scoped.vue", fullSFC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.Scoped {
		t.Error("scoped = false, want true for <style scoped>")
	}
}

func TestParseFile_NotScoped(t *testing.T) {
	c, err := ParseFile("noscope.vue", scopedFalse)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Scoped {
		t.Error("scoped = true, want false for plain <style>")
	}
}

func TestParseFile_TemplateNodeTreeWalkable(t *testing.T) {
	c, err := ParseFile("walk.vue", fullSFC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Walk tree looking for an h1 element.
	found := false
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "h1" {
			found = true
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(c.Template)

	if !found {
		t.Error("could not find <h1> node in walked template tree")
	}
}

func TestParseFile_UnclosedTemplateError(t *testing.T) {
	_, err := ParseFile("bad.vue", unclosedTemplate)
	if err == nil {
		t.Fatal("expected error for unclosed <template>, got nil")
	}
	if !strings.Contains(err.Error(), "unclosed") && !strings.Contains(err.Error(), "template") {
		t.Errorf("error %q should mention 'unclosed' or 'template'", err.Error())
	}
}

func TestParseFile_MissingTemplate(t *testing.T) {
	src := `<script>/* nothing */</script>`
	_, err := ParseFile("notmpl.vue", src)
	if err == nil {
		t.Fatal("expected error for missing <template>, got nil")
	}
	if !strings.Contains(err.Error(), "template") {
		t.Errorf("error %q should mention 'template'", err.Error())
	}
}

func TestParseFile_FullDocumentTemplate(t *testing.T) {
	// A template rooted at <html> must render with <html>, <head>, and <body>
	// preserved — html.ParseFragment silently drops these in a <div> context.
	src := `<template><html>
<head><title>My Page</title></head>
<body><main><p>Content</p></main></body>
</html></template>`
	c, err := ParseFile("layout.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	out, err := RenderString(c, nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}
	if !strings.Contains(out, "<html") {
		t.Errorf("output should contain <html, got: %q", out)
	}
	if !strings.Contains(out, "<head>") {
		t.Errorf("output should contain <head>, got: %q", out)
	}
	if !strings.Contains(out, "<body>") {
		t.Errorf("output should contain <body>, got: %q", out)
	}
}

func TestParseFile_TemplateContentExtracted(t *testing.T) {
	c, err := ParseFile("tmpl.vue", fullSFC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Walk tree and collect all element names.
	var names []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			names = append(names, n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(c.Template)

	// Expect at least "div" and "h1" from the template.
	wantTags := map[string]bool{"div": false, "h1": false}
	for _, name := range names {
		wantTags[name] = true
	}
	for tag, found := range wantTags {
		if !found {
			t.Errorf("expected <%s> in template tree, but not found (got %v)", tag, names)
		}
	}
}

// propNames returns a sorted slice of prop names from a []PropInfo.
func propNames(props []PropInfo) []string {
	names := make([]string, len(props))
	for i, p := range props {
		names[i] = p.Name
	}
	sort.Strings(names)
	return names
}

// propByName finds a PropInfo by name (returns zero value if not found).
func propByName(props []PropInfo, name string) PropInfo {
	for _, p := range props {
		if p.Name == name {
			return p
		}
	}
	return PropInfo{}
}

func parseForProps(t *testing.T, tmpl string) []PropInfo {
	t.Helper()
	src := "<template>" + tmpl + "</template>"
	c, err := ParseFile("test.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	return c.Props()
}

func TestProps_SimpleInterpolation(t *testing.T) {
	props := parseForProps(t, `<p>{{ title }}</p>`)
	names := propNames(props)
	if len(names) != 1 || names[0] != "title" {
		t.Errorf("props = %v, want [title]", names)
	}
}

func TestProps_BoundAttribute(t *testing.T) {
	props := parseForProps(t, `<div :class="cls"></div>`)
	names := propNames(props)
	if len(names) != 1 || names[0] != "cls" {
		t.Errorf("props = %v, want [cls]", names)
	}
}

func TestProps_VBindAttribute(t *testing.T) {
	props := parseForProps(t, `<a v-bind:href="url">link</a>`)
	names := propNames(props)
	if len(names) != 1 || names[0] != "url" {
		t.Errorf("props = %v, want [url]", names)
	}
}

func TestProps_DirectiveExpressions(t *testing.T) {
	props := parseForProps(t, `<div v-if="show" v-show="visible" v-text="msg" v-html="raw"></div>`)
	names := propNames(props)
	want := []string{"msg", "raw", "show", "visible"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Errorf("props = %v, want %v", names, want)
	}
}

func TestProps_VForScoping(t *testing.T) {
	// items should be a prop; item should NOT be
	props := parseForProps(t, `<ul><li v-for="item in items">{{ item }}</li></ul>`)
	names := propNames(props)
	for _, n := range names {
		if n == "item" {
			t.Errorf("'item' should not be a prop (it is a v-for loop variable)")
		}
	}
	found := false
	for _, n := range names {
		if n == "items" {
			found = true
		}
	}
	if !found {
		t.Errorf("props = %v, want 'items' to be included", names)
	}
}

func TestProps_VForWithIndex(t *testing.T) {
	props := parseForProps(t, `<ul><li v-for="(item, index) in list">{{ item }}-{{ index }}</li></ul>`)
	names := propNames(props)
	for _, n := range names {
		if n == "item" || n == "index" {
			t.Errorf("'%s' should not be a prop (it is a v-for loop variable)", n)
		}
	}
	found := false
	for _, n := range names {
		if n == "list" {
			found = true
		}
	}
	if !found {
		t.Errorf("props = %v, want 'list' to be included", names)
	}
}

func TestProps_NestedVFor(t *testing.T) {
	// outer loop var "section" used as inner collection: should be a prop of the outer loop? No.
	// outer: section in sections → sections is prop, section is local
	// inner: item in section.items → section is local (not a prop)
	props := parseForProps(t, `<div v-for="section in sections"><span v-for="item in section.items">{{ item.name }}</span></div>`)
	names := propNames(props)
	for _, n := range names {
		if n == "section" || n == "item" {
			t.Errorf("'%s' should not be a prop (v-for loop variable)", n)
		}
	}
	found := false
	for _, n := range names {
		if n == "sections" {
			found = true
		}
	}
	if !found {
		t.Errorf("props = %v, want 'sections' to be included", names)
	}
}

func TestProps_ExcludeDollarPrefixed(t *testing.T) {
	props := parseForProps(t, `<slot>{{ $slot }}</slot>`)
	for _, p := range props {
		if strings.HasPrefix(p.Name, "$") {
			t.Errorf("prop '%s' starts with '$' and should be excluded", p.Name)
		}
	}
}

func TestProps_ExcludeLen(t *testing.T) {
	props := parseForProps(t, `<p>{{ len(items) }}</p>`)
	names := propNames(props)
	for _, n := range names {
		if n == "len" {
			t.Errorf("'len' should be excluded from props")
		}
	}
	found := false
	for _, n := range names {
		if n == "items" {
			found = true
		}
	}
	if !found {
		t.Errorf("props = %v, want 'items' included", names)
	}
}

func TestProps_MultipleExpressionsAggregate(t *testing.T) {
	props := parseForProps(t, `<p>{{ title }}</p><h1>{{ title }}</h1>`)
	p := propByName(props, "title")
	if p.Name != "title" {
		t.Fatal("prop 'title' not found")
	}
	if len(p.Expressions) < 2 {
		t.Errorf("expected at least 2 expressions for 'title', got %v", p.Expressions)
	}
}
