package main

import (
	"os"
	"strings"
	"testing"

	"github.com/dhamidi/htmlc"
)

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func renderOutput(t *testing.T) (html string, css string) {
	t.Helper()

	layout, err := htmlc.ParseFile("components/Layout.vue", mustReadFile(t, "components/Layout.vue"))
	if err != nil {
		t.Fatalf("ParseFile Layout.vue: %v", err)
	}
	card, err := htmlc.ParseFile("components/Card.vue", mustReadFile(t, "components/Card.vue"))
	if err != nil {
		t.Fatalf("ParseFile Card.vue: %v", err)
	}
	homePage, err := htmlc.ParseFile("components/HomePage.vue", mustReadFile(t, "components/HomePage.vue"))
	if err != nil {
		t.Fatalf("ParseFile HomePage.vue: %v", err)
	}

	reg := htmlc.Registry{
		"Layout": layout,
		"Card":   card,
	}
	sc := &htmlc.StyleCollector{}

	htmlOut, err := htmlc.NewRenderer(homePage).WithStyles(sc).WithComponents(reg).RenderString(pageData())
	if err != nil {
		t.Fatalf("RenderString: %v", err)
	}

	var cssBuilder strings.Builder
	for _, c := range sc.All() {
		cssBuilder.WriteString(c.CSS)
		cssBuilder.WriteString("\n")
	}

	return htmlOut, cssBuilder.String()
}

// Test 1: .page-title and .page-nav elements carry the Page scope attr (named slot).
func TestScopeAttr_PageTitleAndNavHavePageScope(t *testing.T) {
	html, _ := renderOutput(t)
	pageScope := htmlc.ScopeID("components/HomePage.vue")

	for _, cls := range []string{"page-title", "page-nav"} {
		idx := strings.Index(html, `class="`+cls+`"`)
		if idx < 0 {
			t.Fatalf("expected element with class %q in output", cls)
		}
		end := idx + strings.Index(html[idx:], ">") + 1
		tag := html[idx:end]
		if !strings.Contains(tag, pageScope) {
			t.Errorf("element .%s should carry page scope %q:\ntag: %s\nhtml: %s", cls, pageScope, tag, html)
		}
	}
}

// Test 2: .post-title and .post-body carry the Page scope attr (default slot).
func TestScopeAttr_PostTitleAndBodyHavePageScope(t *testing.T) {
	html, _ := renderOutput(t)
	pageScope := htmlc.ScopeID("components/HomePage.vue")

	for _, cls := range []string{"post-title", "post-body"} {
		idx := strings.Index(html, `class="`+cls+`"`)
		if idx < 0 {
			t.Fatalf("expected element with class %q in output", cls)
		}
		end := idx + strings.Index(html[idx:], ">") + 1
		tag := html[idx:end]
		if !strings.Contains(tag, pageScope) {
			t.Errorf("element .%s should carry page scope %q:\ntag: %s", cls, pageScope, tag)
		}
	}
}

// Test 3: .card and .card-body carry the Card scope attr (child template).
func TestScopeAttr_CardElementsHaveCardScope(t *testing.T) {
	html, _ := renderOutput(t)
	cardScope := htmlc.ScopeID("components/Card.vue")

	for _, cls := range []string{"card", "card-body"} {
		idx := strings.Index(html, `class="`+cls+`"`)
		if idx < 0 {
			t.Fatalf("expected element with class %q in output", cls)
		}
		end := idx + strings.Index(html[idx:], ">") + 1
		tag := html[idx:end]
		if !strings.Contains(tag, cardScope) {
			t.Errorf("element .%s should carry card scope %q:\ntag: %s", cls, cardScope, tag)
		}
	}
}

// Test 4: .layout-header and .layout-main carry the Layout scope attr (child template).
func TestScopeAttr_LayoutElementsHaveLayoutScope(t *testing.T) {
	html, _ := renderOutput(t)
	layoutScope := htmlc.ScopeID("components/Layout.vue")

	for _, cls := range []string{"layout-header", "layout-main"} {
		idx := strings.Index(html, `class="`+cls+`"`)
		if idx < 0 {
			t.Fatalf("expected element with class %q in output", cls)
		}
		end := idx + strings.Index(html[idx:], ">") + 1
		tag := html[idx:end]
		if !strings.Contains(tag, layoutScope) {
			t.Errorf("element .%s should carry layout scope %q:\ntag: %s", cls, layoutScope, tag)
		}
	}
}

// Test 5: CSS block contains .page-title[data-v-<page>] selector.
func TestCSS_PageTitleHasPageScopeSelector(t *testing.T) {
	_, css := renderOutput(t)
	pageScope := htmlc.ScopeID("components/HomePage.vue")
	want := ".page-title[" + pageScope + "]"
	if !strings.Contains(css, want) {
		t.Errorf("CSS should contain %q:\n%s", want, css)
	}
}

// Test 6: CSS block contains .card[data-v-<card>] selector.
func TestCSS_CardHasCardScopeSelector(t *testing.T) {
	_, css := renderOutput(t)
	cardScope := htmlc.ScopeID("components/Card.vue")
	want := ".card[" + cardScope + "]"
	if !strings.Contains(css, want) {
		t.Errorf("CSS should contain %q:\n%s", want, css)
	}
}

// Test 7: No element carries both the Page scope attr AND the Layout/Card scope attr simultaneously.
func TestScopeAttr_NoElementHasBothPageAndChildScope(t *testing.T) {
	html, _ := renderOutput(t)
	pageScope := htmlc.ScopeID("components/HomePage.vue")
	layoutScope := htmlc.ScopeID("components/Layout.vue")
	cardScope := htmlc.ScopeID("components/Card.vue")

	// Walk through each opening tag and verify no tag has both page + child scope.
	remaining := html
	for {
		start := strings.Index(remaining, "<")
		if start < 0 {
			break
		}
		end := strings.Index(remaining[start:], ">")
		if end < 0 {
			break
		}
		tag := remaining[start : start+end+1]
		remaining = remaining[start+end+1:]

		hasPage := strings.Contains(tag, pageScope)
		hasLayout := strings.Contains(tag, layoutScope)
		hasCard := strings.Contains(tag, cardScope)

		if hasPage && hasLayout {
			t.Errorf("element has both page and layout scope attrs:\n%s", tag)
		}
		if hasPage && hasCard {
			t.Errorf("element has both page and card scope attrs:\n%s", tag)
		}
	}
}
