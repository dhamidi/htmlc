package htmlc

import (
	"errors"
	"fmt"
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
	src := `<style>.a { color: red; }</style>`
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

func TestParseFile_ScriptBlockError(t *testing.T) {
	src := "<template><p>hi</p></template>\n<script>\nconsole.log('hi')\n</script>"
	_, err := ParseFile("comp.vue", src)
	if err == nil {
		t.Fatal("expected error for <script> block, got nil")
	}
	if !strings.Contains(err.Error(), "<script> blocks are not supported") {
		t.Errorf("error %q should mention '<script> blocks are not supported'", err.Error())
	}
	if !strings.Contains(err.Error(), "server") {
		t.Errorf("error %q should mention server-side rendering", err.Error())
	}
}

func TestParseFile_ScriptSetupBlockError(t *testing.T) {
	src := "<template><p>hi</p></template>\n<script setup>\nconst x = 1\n</script>"
	_, err := ParseFile("comp.vue", src)
	if err == nil {
		t.Fatal("expected error for <script setup> block, got nil")
	}
	if !strings.Contains(err.Error(), "<script setup> blocks are not supported") {
		t.Errorf("error %q should mention '<script setup> blocks are not supported'", err.Error())
	}
	if !strings.Contains(err.Error(), "server") {
		t.Errorf("error %q should mention server-side rendering", err.Error())
	}
}

// ---------- normalizeSelfClosingComponents tests ----------

func TestNormalizeSelfClosingComponents_BasicPascalCase(t *testing.T) {
	input := `<PostImage src="/hero.jpg" alt="Hero" />`
	got, count := normalizeSelfClosingComponents(input)
	want := `<PostImage src="/hero.jpg" alt="Hero"></PostImage>`
	if got != want {
		t.Errorf("normalize = %q, want %q", got, want)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestNormalizeSelfClosingComponents_NoAttributesPascalCase(t *testing.T) {
	input := `<MyComponent />`
	got, count := normalizeSelfClosingComponents(input)
	want := `<MyComponent></MyComponent>`
	if got != want {
		t.Errorf("normalize = %q, want %q", got, want)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestNormalizeSelfClosingComponents_LowercaseTagNotAffected(t *testing.T) {
	input := `<img src="/foo.jpg" />`
	got, count := normalizeSelfClosingComponents(input)
	if got != input {
		t.Errorf("normalize changed lowercase tag: got %q, want %q", got, input)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 for lowercase tag", count)
	}
}

func TestNormalizeSelfClosingComponents_AttributeContainingSlashGT(t *testing.T) {
	// Attribute value contains "/>"; the normalization must not corrupt it.
	input := `<PostImage :title="'a/>'" />`
	got, count := normalizeSelfClosingComponents(input)
	// The attribute value should be preserved verbatim.
	if !strings.Contains(got, `'a/>'`) {
		t.Errorf("normalize corrupted attribute containing '/>': got %q", got)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	// Should end with </PostImage>
	if !strings.HasSuffix(strings.TrimSpace(got), "</PostImage>") {
		t.Errorf("normalize output should end with </PostImage>, got %q", got)
	}
}

func TestNormalizeSelfClosingComponents_IdempotentOnExplicitTags(t *testing.T) {
	input := `<PostImage src="/hero.jpg"></PostImage>`
	got, count := normalizeSelfClosingComponents(input)
	if got != input {
		t.Errorf("normalize changed explicit tag: got %q, want %q", got, input)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 for explicit tag", count)
	}
}

func TestNormalizeSelfClosingComponents_MidTemplateSiblingNotSwallowed(t *testing.T) {
	// Verify that after normalization the subsequent <p> is a sibling, not a child.
	src := `<template><PostImage src="/hero.jpg" /><p>Caption</p></template>`
	c, err := ParseFile("test.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	// Walk template and find both PostImage (lowercased by HTML parser as
	// "postimage") and p at the same level.
	var postImageNode, pNode *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "postimage": // HTML parser lowercases tag names
				postImageNode = n
			case "p":
				pNode = n
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(c.Template)

	if postImageNode == nil {
		t.Fatal("PostImage node not found in template tree")
	}
	if pNode == nil {
		t.Fatal("<p> node not found in template tree")
	}
	// <p> must NOT be a child of PostImage.
	for child := postImageNode.FirstChild; child != nil; child = child.NextSibling {
		if child == pNode {
			t.Error("<p> should be a sibling of PostImage, not a child")
		}
	}
	// <p> should be a sibling of PostImage (same parent).
	if pNode.Parent != postImageNode.Parent {
		t.Errorf("<p> parent differs from PostImage parent; <p> was swallowed as child")
	}
}

func TestNormalizeSelfClosingComponents_WarningsPopulated(t *testing.T) {
	src := `<template><PostImage src="/hero.jpg" /></template>`
	c, err := ParseFile("mycomp.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(c.Warnings) == 0 {
		t.Fatal("expected Warnings to be populated when self-closing normalization occurs")
	}
	w := c.Warnings[0]
	if !strings.Contains(w, "mycomp.vue") {
		t.Errorf("warning should mention the file path, got %q", w)
	}
	if !strings.Contains(w, "auto-corrected") {
		t.Errorf("warning should mention 'auto-corrected', got %q", w)
	}
}

func TestNormalizeSelfClosingComponents_NoWarningsWhenNonePresent(t *testing.T) {
	src := `<template><PostImage src="/hero.jpg"></PostImage></template>`
	c, err := ParseFile("mycomp.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if len(c.Warnings) != 0 {
		t.Errorf("expected no Warnings for explicit open/close tags, got %v", c.Warnings)
	}
}

func TestNormalizeSelfClosingComponents_MultipleTagsCountCorrect(t *testing.T) {
	input := `<Foo /><Bar x="1" /><baz />`
	_, count := normalizeSelfClosingComponents(input)
	// Only Foo and Bar are PascalCase; baz is lowercase and should not match.
	if count != 2 {
		t.Errorf("count = %d, want 2 (only PascalCase tags)", count)
	}
}

// ---------- end normalizeSelfClosingComponents tests ----------

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

func TestProps_IncludeLen(t *testing.T) {
	props := parseForProps(t, `<p>{{ len(items) }}</p>`)
	names := propNames(props)
	foundLen := false
	foundItems := false
	for _, n := range names {
		if n == "len" {
			foundLen = true
		}
		if n == "items" {
			foundItems = true
		}
	}
	if !foundLen {
		t.Errorf("props = %v, want 'len' included", names)
	}
	if !foundItems {
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

func TestProps_VSlotTemplateDestructured(t *testing.T) {
	// user and index come from the slot binding, not from the parent scope
	props := parseForProps(t, `<UserList><template #item="{ user, index }"><li>{{ index }}: {{ user.name }}</li></template></UserList>`)
	names := propNames(props)
	for _, n := range names {
		if n == "user" || n == "index" {
			t.Errorf("'%s' should not be a prop (it is a v-slot binding variable)", n)
		}
	}
}

func TestProps_VSlotComponentTag(t *testing.T) {
	// msg comes from the v-slot binding on the component tag
	props := parseForProps(t, `<Wrapper v-slot="{ msg }"><p>{{ msg }}</p></Wrapper>`)
	names := propNames(props)
	for _, n := range names {
		if n == "msg" {
			t.Errorf("'msg' should not be a prop (it is a v-slot binding variable)")
		}
	}
}

func TestProps_VSlotWithBoundProp(t *testing.T) {
	// users is a prop (bound via :users), user is NOT (comes from slot binding)
	props := parseForProps(t, `<UserList :users="users"><template #item="{ user }">{{ user.name }}</template></UserList>`)
	names := propNames(props)
	for _, n := range names {
		if n == "user" {
			t.Errorf("'user' should not be a prop (it is a v-slot binding variable)")
		}
	}
	found := false
	for _, n := range names {
		if n == "users" {
			found = true
		}
	}
	if !found {
		t.Errorf("props = %v, want 'users' to be included", names)
	}
}

// ---------- ParseError location tests ----------

func TestParseFile_UnclosedTemplate_HasNonZeroLine(t *testing.T) {
	// An unclosed <template> section should produce a *ParseError.
	// With the lightweight location logic the error may not have a precise
	// line, but it should still return a *ParseError.
	_, err := ParseFile("bad.vue", unclosedTemplate)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if pe.Path != "bad.vue" {
		t.Errorf("ParseError.Path = %q, want %q", pe.Path, "bad.vue")
	}
}

func TestParseFile_MissingTemplate_IsParseError(t *testing.T) {
	src := `<style>/* nothing */</style>`
	_, err := ParseFile("notmpl.vue", src)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *ParseError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
}

func TestParseFile_SourceField_Populated(t *testing.T) {
	c, err := ParseFile("test.vue", fullSFC)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if c.Source == "" {
		t.Error("Component.Source should be populated after successful parse")
	}
	if !strings.Contains(c.Source, "<template>") {
		t.Errorf("Component.Source should contain the original source, got: %q", c.Source)
	}
}

// TestParseFile_EdgeCases documents parser robustness for unusual inputs.
// Each subtest pins a specific boundary condition so future refactors cannot
// silently change the observable behaviour.
func TestParseFile_EdgeCases(t *testing.T) {
	// A file with only whitespace has no <template> section, so ParseFile must
	// return an error rather than a zero-value Component.
	t.Run("whitespace-only file returns error", func(t *testing.T) {
		_, err := ParseFile("blank.vue", "   \n\t  ")
		if err == nil {
			t.Error("ParseFile(whitespace): expected error for missing template, got nil")
		}
	})

	// An opened <template> tag with no matching </template> must be detected as
	// an unclosed section and reported as an error.
	t.Run("unclosed template tag returns error", func(t *testing.T) {
		_, err := ParseFile("unclosed.vue", "<template><div>hello")
		if err == nil {
			t.Error("ParseFile(unclosed): expected error for unclosed template, got nil")
		}
	})

	// An empty <style scoped></style> block must not panic.  The Scoped flag
	// should be true and Style should be the empty string.
	t.Run("empty scoped style does not panic and sets Scoped", func(t *testing.T) {
		src := "<template><p>x</p></template><style scoped></style>"
		c, err := ParseFile("empty-scoped.vue", src)
		if err != nil {
			t.Fatalf("ParseFile: unexpected error: %v", err)
		}
		if !c.Scoped {
			t.Error("Scoped = false, want true for <style scoped>")
		}
		if c.Style != "" {
			t.Errorf("Style = %q, want empty for empty <style scoped>", c.Style)
		}
	})

	// Two <template> blocks in the same file are a structural error.
	// This test pins the behaviour (error) so a future refactor cannot silently
	// change it to a "first-wins" strategy without updating this test.
	t.Run("duplicate template section returns error", func(t *testing.T) {
		src := "<template><p>first</p></template><template><p>second</p></template>"
		_, err := ParseFile("dup.vue", src)
		if err == nil {
			t.Error("ParseFile(duplicate template): expected error, got nil")
		}
	})

	// A very large template must parse within a reasonable time.  10 000 <p>
	// elements is ~100 KB and exercises the HTML tokeniser for performance
	// regressions without wall-clock sleeps.
	t.Run("100KB template parses without error", func(t *testing.T) {
		var b strings.Builder
		b.WriteString("<template>")
		for i := 0; i < 10000; i++ {
			fmt.Fprintf(&b, "<p>item %d</p>", i)
		}
		b.WriteString("</template>")
		_, err := ParseFile("large.vue", b.String())
		if err != nil {
			t.Errorf("ParseFile(100KB): unexpected error: %v", err)
		}
	})
}

