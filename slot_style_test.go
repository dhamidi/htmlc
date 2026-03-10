package htmlc

import (
	"strings"
	"testing"
)

func TestScopedSlot_SlotContentGetsParentScopeAttr(t *testing.T) {
	childSrc := `<template><div class="card"><slot></slot></div></template>`
	parentSrc := `<template><Card><span class="inner">hello</span></Card></template>
<style scoped>.inner { color: red; }</style>`

	child, _ := ParseFile("Card.vue", childSrc)
	parent, _ := ParseFile("Page.vue", parentSrc)

	reg := Registry{"Card": child}
	sc := &StyleCollector{}
	out, err := NewRenderer(parent).WithStyles(sc).WithComponents(reg).RenderString(nil)
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}

	parentScope := ScopeID("Page.vue")
	childScope := ScopeID("Card.vue")

	// The <span> was authored in Page.vue → must carry the parent scope attr.
	if !strings.Contains(out, parentScope) {
		t.Errorf("slot content should carry parent scope attr %q:\n%s", parentScope, out)
	}
	// The <span> must NOT be stamped with the child scope attr.
	spanIdx := strings.Index(out, "<span")
	if spanIdx < 0 {
		t.Fatal("expected <span> in output")
	}
	spanTag := out[spanIdx : spanIdx+strings.Index(out[spanIdx:], ">")+1]
	if strings.Contains(spanTag, childScope) {
		t.Errorf("<span> slot content should not carry child scope attr %q:\n%s", childScope, spanTag)
	}
}

func TestScopedSlot_SlotContentNotStampedWithChildScope(t *testing.T) {
	childSrc := `<template><div><slot></slot></div></template><style scoped>.card{}</style>`
	parentSrc := `<template><Card><em>text</em></Card></template>`

	child, _ := ParseFile("Card.vue", childSrc)
	parent, _ := ParseFile("Page.vue", parentSrc)

	reg := Registry{"Card": child}
	out, _ := NewRenderer(parent).WithComponents(reg).RenderString(nil)

	childScope := ScopeID("Card.vue")
	// <em> is slot content from an unscoped parent – must not carry child scope.
	emIdx := strings.Index(out, "<em")
	if emIdx < 0 {
		t.Fatal("expected <em> in output")
	}
	emTag := out[emIdx : emIdx+strings.Index(out[emIdx:], ">")+1]
	if strings.Contains(emTag, childScope) {
		t.Errorf("<em> slot content should not carry child scope attr %q:\n%s", childScope, emTag)
	}
}

func TestScopedSlot_ChildTemplateElementsGetChildScope(t *testing.T) {
	childSrc := `<template><div class="wrapper"><slot></slot></div></template>
<style scoped>.wrapper{border:1px solid}</style>`
	parentSrc := `<template><Card><p>content</p></Card></template>`

	child, _ := ParseFile("Card.vue", childSrc)
	parent, _ := ParseFile("Page.vue", parentSrc)

	reg := Registry{"Card": child}
	sc := &StyleCollector{}
	out, _ := NewRenderer(parent).WithStyles(sc).WithComponents(reg).RenderString(nil)

	childScope := ScopeID("Card.vue")

	// <div class="wrapper"> is in Card's template → must carry child scope.
	if !strings.Contains(out, `class="wrapper" `+childScope) &&
		!strings.Contains(out, `class="wrapper"`+childScope) {
		t.Errorf("child template element should carry child scope attr %q:\n%s", childScope, out)
	}
}

func TestScopedSlot_BothScopedParentAndChild(t *testing.T) {
	childSrc := `<template><section><slot></slot></section></template>
<style scoped>section{padding:1rem}</style>`
	parentSrc := `<template><Card><article class="post">hi</article></Card></template>
<style scoped>.post{margin:0}</style>`

	child, _ := ParseFile("Card.vue", childSrc)
	parent, _ := ParseFile("Page.vue", parentSrc)

	reg := Registry{"Card": child}
	sc := &StyleCollector{}
	out, _ := NewRenderer(parent).WithStyles(sc).WithComponents(reg).RenderString(nil)

	parentScope := ScopeID("Page.vue")
	childScope := ScopeID("Card.vue")

	// <article> is slot content from Page → must carry Page's scope.
	articleIdx := strings.Index(out, "<article")
	if articleIdx < 0 {
		t.Fatal("expected <article> in output")
	}
	articleTag := out[articleIdx : articleIdx+strings.Index(out[articleIdx:], ">")+1]
	if !strings.Contains(articleTag, parentScope) {
		t.Errorf("<article> slot content should carry parent scope %q:\n%s", parentScope, articleTag)
	}
	if strings.Contains(articleTag, childScope) {
		t.Errorf("<article> slot content must not carry child scope %q:\n%s", childScope, articleTag)
	}

	// <section> is Card's own template element → must carry child scope.
	secIdx := strings.Index(out, "<section")
	if secIdx < 0 {
		t.Fatal("expected <section> in output")
	}
	secTag := out[secIdx : secIdx+strings.Index(out[secIdx:], ">")+1]
	if !strings.Contains(secTag, childScope) {
		t.Errorf("<section> child template element should carry child scope %q:\n%s", childScope, secTag)
	}
}

func TestScopedSlot_NamedSlotGetsParentScope(t *testing.T) {
	childSrc := `<template><header><slot name="header"></slot></header><main><slot></slot></main></template>`
	parentSrc := `<template>
<Layout>
  <template #header><h1 class="title">Title</h1></template>
  <p class="body">Body</p>
</Layout>
</template>
<style scoped>.title{font-size:2em} .body{color:black}</style>`

	child, _ := ParseFile("Layout.vue", childSrc)
	parent, _ := ParseFile("Page.vue", parentSrc)

	reg := Registry{"Layout": child}
	out, _ := NewRenderer(parent).WithComponents(reg).RenderString(nil)

	parentScope := ScopeID("Page.vue")
	childScope := ScopeID("Layout.vue")

	// Both the named and default slot content come from Page → parent scope.
	for _, tag := range []string{"<h1", "<p"} {
		idx := strings.Index(out, tag)
		if idx < 0 {
			t.Fatalf("expected %s in output:\n%s", tag, out)
		}
		end := idx + strings.Index(out[idx:], ">") + 1
		tagStr := out[idx:end]
		if !strings.Contains(tagStr, parentScope) {
			t.Errorf("%s slot content should carry parent scope %q:\n%s", tag, parentScope, tagStr)
		}
		if strings.Contains(tagStr, childScope) {
			t.Errorf("%s slot content must not carry child scope %q:\n%s", tag, childScope, tagStr)
		}
	}
}

func TestScopedSlot_FallbackContentGetsChildScope(t *testing.T) {
	childSrc := `<template><div><slot><span class="fallback">default</span></slot></div></template>
<style scoped>.fallback{opacity:.5}</style>`
	parentSrc := `<template><Card></Card></template>`

	child, _ := ParseFile("Card.vue", childSrc)
	parent, _ := ParseFile("Page.vue", parentSrc)

	reg := Registry{"Card": child}
	sc := &StyleCollector{}
	out, _ := NewRenderer(parent).WithStyles(sc).WithComponents(reg).RenderString(nil)

	childScope := ScopeID("Card.vue")
	// Fallback <span> is authored in Card → must carry child scope.
	spanIdx := strings.Index(out, "<span")
	if spanIdx < 0 {
		t.Fatal("expected fallback <span> in output")
	}
	spanTag := out[spanIdx : spanIdx+strings.Index(out[spanIdx:], ">")+1]
	if !strings.Contains(spanTag, childScope) {
		t.Errorf("fallback slot content should carry child scope %q:\n%s", childScope, spanTag)
	}
}

func TestScopedSlot_VForInSlotGetsParentScope(t *testing.T) {
	childSrc := `<template><ul><slot></slot></ul></template>`
	parentSrc := `<template>
<List>
  <li v-for="item in items">{{ item }}</li>
</List>
</template>
<style scoped>li{list-style:none}</style>`

	child, _ := ParseFile("List.vue", childSrc)
	parent, _ := ParseFile("Page.vue", parentSrc)

	reg := Registry{"List": child}
	out, _ := NewRenderer(parent).WithComponents(reg).RenderString(map[string]any{
		"items": []any{"a", "b"},
	})

	parentScope := ScopeID("Page.vue")
	// Both <li> iterations must carry the parent scope attr.
	count := strings.Count(out, parentScope)
	if count < 2 {
		t.Errorf("v-for slot content: want ≥2 elements with parent scope %q, got %d:\n%s",
			parentScope, count, out)
	}
}

func TestScopedSlot_StyleCollector_SlotCSS(t *testing.T) {
	childSrc := `<template><div><slot></slot></div></template>`
	parentSrc := `<template><Card><p class="msg">hello</p></Card></template>
<style scoped>.msg { color: hotpink; }</style>`

	child, _ := ParseFile("Card.vue", childSrc)
	parent, _ := ParseFile("Page.vue", parentSrc)

	reg := Registry{"Card": child}
	sc := &StyleCollector{}
	NewRenderer(parent).WithStyles(sc).WithComponents(reg).RenderString(nil) //nolint:errcheck

	contribs := sc.All()
	if len(contribs) == 0 {
		t.Fatal("expected at least one style contribution")
	}
	// Find the Page.vue contribution.
	parentScope := ScopeID("Page.vue")
	scopeAttr := "[" + parentScope + "]"
	found := false
	for _, c := range contribs {
		if strings.Contains(c.CSS, ".msg") && strings.Contains(c.CSS, scopeAttr) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected CSS contribution with .msg%s from Page.vue, got: %v", scopeAttr, contribs)
	}
}
