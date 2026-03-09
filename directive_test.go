package htmlc

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// addAttrDirective is a test directive that adds a data-directive attribute
// with the given value in the Created hook.
type addAttrDirective struct {
	attrName string
}

func (d *addAttrDirective) Created(node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error {
	node.Attr = append(node.Attr, html.Attribute{Key: d.attrName, Val: fmt.Sprintf("%v", binding.Value)})
	return nil
}

func (d *addAttrDirective) Mounted(_ io.Writer, _ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	return nil
}

// appendHTMLDirective is a test directive that appends HTML in the Mounted hook.
type appendHTMLDirective struct {
	suffix string
}

func (d *appendHTMLDirective) Created(_ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	return nil
}

func (d *appendHTMLDirective) Mounted(w io.Writer, _ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	_, err := io.WriteString(w, d.suffix)
	return err
}

// errorDirective is a test directive whose Created hook always returns an error.
type errorDirective struct {
	msg string
}

func (d *errorDirective) Created(_ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	return errors.New(d.msg)
}

func (d *errorDirective) Mounted(_ io.Writer, _ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	return nil
}

// renderWithDirectives is a helper that renders a template string with a
// custom directive registry attached.
func renderWithDirectives(t *testing.T, tmpl string, scope map[string]any, dr DirectiveRegistry) (string, error) {
	t.Helper()
	src := "<template>" + tmpl + "</template>"
	c, err := ParseFile("test.vue", src)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	var buf strings.Builder
	renderer := NewRenderer(c).WithDirectives(dr)
	err = renderer.Render(&buf, scope)
	return buf.String(), err
}

// TestDirective_CreatedMutatesAttr verifies that a Created hook can add
// attributes to the working node and that they appear in the rendered output.
func TestDirective_CreatedMutatesAttr(t *testing.T) {
	dr := DirectiveRegistry{
		"mark": &addAttrDirective{attrName: "data-mark"},
	}
	out, err := renderWithDirectives(t, `<div v-mark="'hello'"></div>`, map[string]any{}, dr)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(out, `data-mark="hello"`) {
		t.Errorf("got %q, want data-mark=\"hello\" attribute", out)
	}
}

// TestDirective_MountedAppendsHTML verifies that a Mounted hook can write
// HTML after the element's closing tag.
func TestDirective_MountedAppendsHTML(t *testing.T) {
	dr := DirectiveRegistry{
		"append": &appendHTMLDirective{suffix: "<span>after</span>"},
	}
	out, err := renderWithDirectives(t, `<p v-append="x">content</p>`, map[string]any{"x": 1}, dr)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if !strings.Contains(out, "</p><span>after</span>") {
		t.Errorf("got %q, want </p><span>after</span>", out)
	}
}

// TestDirective_ArgAndModifierParsing verifies that v-dir:arg.mod1.mod2
// populates DirectiveBinding.Arg and Modifiers correctly.
func TestDirective_ArgAndModifierParsing(t *testing.T) {
	var gotBinding DirectiveBinding
	capture := &captureBindingDirective{bindingOut: &gotBinding}
	dr := DirectiveRegistry{"dir": capture}

	_, err := renderWithDirectives(t, `<div v-dir:href.prevent.stop="x"></div>`, map[string]any{"x": 42}, dr)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if gotBinding.Arg != "href" {
		t.Errorf("Arg: got %q, want %q", gotBinding.Arg, "href")
	}
	if !gotBinding.Modifiers["prevent"] {
		t.Errorf("Modifiers: missing 'prevent', got %v", gotBinding.Modifiers)
	}
	if !gotBinding.Modifiers["stop"] {
		t.Errorf("Modifiers: missing 'stop', got %v", gotBinding.Modifiers)
	}
}

// captureBindingDirective captures the DirectiveBinding for inspection.
type captureBindingDirective struct {
	bindingOut *DirectiveBinding
}

func (d *captureBindingDirective) Created(node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error {
	*d.bindingOut = binding
	return nil
}

func (d *captureBindingDirective) Mounted(_ io.Writer, _ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	return nil
}

// TestDirective_UnknownPassthrough verifies that an unrecognised v-foo attribute
// is passed through as a plain attribute in the rendered output.
func TestDirective_UnknownPassthrough(t *testing.T) {
	// Empty registry — no directives registered.
	dr := DirectiveRegistry{}
	out, err := renderWithDirectives(t, `<div v-unknown="1"></div>`, map[string]any{}, dr)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	// The unknown directive should be emitted as a plain attribute.
	if !strings.Contains(out, "v-unknown") {
		t.Errorf("got %q, want v-unknown attribute to pass through", out)
	}
}

// TestDirective_ErrorPropagation verifies that a Created hook returning an
// error causes Render to return a wrapped error.
func TestDirective_ErrorPropagation(t *testing.T) {
	dr := DirectiveRegistry{
		"fail": &errorDirective{msg: "boom"},
	}
	_, err := renderWithDirectives(t, `<div v-fail="x"></div>`, map[string]any{"x": 1}, dr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error %q does not contain 'boom'", err.Error())
	}
	if !strings.Contains(err.Error(), "v-fail") {
		t.Errorf("error %q does not mention 'v-fail'", err.Error())
	}
}

// TestDirective_ScopeAccess verifies that binding.Value reflects the evaluated
// expression from the current scope.
func TestDirective_ScopeAccess(t *testing.T) {
	var gotVal any
	capture := &captureValueDirective{valOut: &gotVal}
	dr := DirectiveRegistry{"inspect": capture}

	scope := map[string]any{"item": map[string]any{"type": "Banner"}}
	_, err := renderWithDirectives(t, `<div v-inspect="item.type"></div>`, scope, dr)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if gotVal != "Banner" {
		t.Errorf("binding.Value: got %v (%T), want %q", gotVal, gotVal, "Banner")
	}
}

// captureValueDirective captures the evaluated Value for inspection.
type captureValueDirective struct {
	valOut *any
}

func (d *captureValueDirective) Created(_ *html.Node, binding DirectiveBinding, _ DirectiveContext) error {
	*d.valOut = binding.Value
	return nil
}

func (d *captureValueDirective) Mounted(_ io.Writer, _ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	return nil
}

// TestParseDirectiveKey_Basic verifies the basic parsing of v-name:arg.mod.
func TestParseDirectiveKey_Basic(t *testing.T) {
	tests := []struct {
		key      string
		wantName string
		wantArg  string
		wantMods map[string]bool
	}{
		{"v-switch", "switch", "", nil},
		{"v-bind:href", "bind", "href", nil},
		{"v-on:click.prevent", "on", "click", map[string]bool{"prevent": true}},
		{"v-dir:arg.mod1.mod2", "dir", "arg", map[string]bool{"mod1": true, "mod2": true}},
		{"class", "", "", nil},
		{":href", "", "", nil},
	}
	for _, tc := range tests {
		name, arg, mods := parseDirectiveKey(tc.key)
		if name != tc.wantName {
			t.Errorf("parseDirectiveKey(%q) name: got %q, want %q", tc.key, name, tc.wantName)
		}
		if arg != tc.wantArg {
			t.Errorf("parseDirectiveKey(%q) arg: got %q, want %q", tc.key, arg, tc.wantArg)
		}
		for wk := range tc.wantMods {
			if !mods[wk] {
				t.Errorf("parseDirectiveKey(%q) mods: missing %q, got %v", tc.key, wk, mods)
			}
		}
	}
}
