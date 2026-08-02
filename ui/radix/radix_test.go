package radix

import (
	"io/fs"
	"strings"
	"testing"
)

func TestFS_ReturnsNonNil(t *testing.T) {
	got := FS()
	if got == nil {
		t.Fatal("FS() returned nil")
	}
}

func TestFS_AccordionComponentReadable(t *testing.T) {
	data, err := fs.ReadFile(FS(), "components/Accordion.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Accordion.vue) failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("components/Accordion.vue read back empty content")
	}
}

// readTokens returns Tokens.vue's source. Several other components' own
// tests (Checkbox/RadioGroup/Select/Switch — see each file's own _test.go)
// assert on the shared .radix-visually-hidden-input CSS rule; that rule
// lives in Tokens.vue, not in any consuming component's own <style>
// section (see Tokens.vue's and radix.go's header comments for why), so
// those assertions read it from here rather than from their own
// component's source.
func readTokens(t *testing.T) string {
	t.Helper()
	data, err := fs.ReadFile(FS(), "components/Tokens.vue")
	if err != nil {
		t.Fatalf("fs.ReadFile(components/Tokens.vue) failed: %v", err)
	}
	return string(data)
}

// tokensCSSRule returns the text of the first CSS rule in Tokens.vue's
// source starting with selectorOpen (e.g. ".radix-visually-hidden-input
// {"), up to and including its closing brace.
func tokensCSSRule(t *testing.T, selectorOpen string) string {
	t.Helper()
	src := readTokens(t)
	start := strings.Index(src, selectorOpen)
	if start == -1 {
		t.Fatalf("Tokens.vue missing expected CSS rule starting with %q; source was:\n%s", selectorOpen, src)
	}
	end := strings.Index(src[start:], "}")
	if end == -1 {
		t.Fatalf("Tokens.vue could not find end of CSS rule %q; source was:\n%s", selectorOpen, src)
	}
	return src[start : start+end+1]
}
