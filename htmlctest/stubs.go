package htmlctest

import (
	"testing"

	"github.com/dhamidi/htmlc"
)

// Deprecated: NewEngine is removed. Replace with [NewHarness].
//
//	h := htmlctest.NewHarness(t, files)
//	h.Fragment("ComponentName", data).AssertHTML(want)
func NewEngine(t testing.TB, files map[string]string, opts ...htmlc.Options) *htmlc.Engine {
	t.Helper()
	t.Fatalf("htmlctest.NewEngine is removed.\nReplace with:\n  h := htmlctest.NewHarness(t, files)\n  h.Fragment(\"ComponentName\", data).AssertHTML(want)")
	return nil
}

// Deprecated: AssertRendersHTML is removed. Replace with [NewHarness] + [Harness.Page].
//
//	h := htmlctest.NewHarness(t, files)
//	h.Page(name, data).AssertHTML(want)
func AssertRendersHTML(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string) {
	t.Helper()
	t.Fatalf("htmlctest.AssertRendersHTML is removed.\nReplace with:\n  h := htmlctest.NewHarness(t, files)\n  h.Page(%q, data).AssertHTML(want)", name)
}

// Deprecated: AssertFragment is removed. Replace with [NewHarness] + [Harness.Fragment].
//
//	h := htmlctest.NewHarness(t, files)
//	h.Fragment(name, data).AssertHTML(want)
func AssertFragment(t testing.TB, e *htmlc.Engine, name string, data map[string]any, want string) {
	t.Helper()
	t.Fatalf("htmlctest.AssertFragment is removed.\nReplace with:\n  h := htmlctest.NewHarness(t, files)\n  h.Fragment(%q, data).AssertHTML(want)", name)
}
