package htmlc

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/dhamidi/htmlc/expr"
)

// renderAttrFragment renders tmpl (the contents of a <template> block) as a
// single-component fragment via a real Engine/RenderFragmentString
// round-trip, using an in-memory fstest.MapFS component source.
func renderAttrFragment(t *testing.T, tmpl string, data map[string]any) string {
	t.Helper()
	memFS := fstest.MapFS{
		"Frag.vue": &fstest.MapFile{Data: []byte("<template>" + tmpl + "</template>")},
	}
	e, err := New(Options{FS: memFS, ComponentDir: "."})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := e.RenderFragmentString(context.Background(), "Frag", data)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	return out
}

// --- Direct :attr / v-bind:attr binding site ---

func TestAttrOmit_Direct_HxSwapOobFalse_Omitted(t *testing.T) {
	// RFC's own motivating example: a non-native attribute name outside the
	// fixed isBooleanAttr eight, bound to false, must be omitted entirely.
	out := renderAttrFragment(t, `<div :hx-swap-oob="false">x</div>`, nil)
	if strings.Contains(out, "hx-swap-oob") {
		t.Errorf("got %q, want hx-swap-oob omitted entirely", out)
	}
}

func TestAttrOmit_Direct_HxSwapOobTrue_RendersValued(t *testing.T) {
	// hx-swap-oob is not one of the fixed eight isBooleanAttr names, so a
	// non-omitted value must render key="value", not bare.
	out := renderAttrFragment(t, `<div :hx-swap-oob="true">x</div>`, nil)
	if !strings.Contains(out, `hx-swap-oob="true"`) {
		t.Errorf(`got %q, want it to contain hx-swap-oob="true" (valued, not bare)`, out)
	}
}

func TestAttrOmit_Direct_NilAndUndefined_Omitted(t *testing.T) {
	// null → Go nil, undefined → expr.Undefined; both must omit regardless
	// of attribute name.
	out := renderAttrFragment(t, `<div :some-attr="null" :other-attr="undefined">x</div>`, nil)
	if strings.Contains(out, "some-attr") {
		t.Errorf("got %q, want some-attr omitted (null)", out)
	}
	if strings.Contains(out, "other-attr") {
		t.Errorf("got %q, want other-attr omitted (undefined)", out)
	}
}

func TestAttrOmit_Direct_ZeroAndEmptyString_NotOmitted(t *testing.T) {
	// 0 and "" are JS-falsy (expr.IsTruthy would say false) but are NOT
	// omit-worthy under attrShouldOmit: "every other value, including the
	// empty string, renders normally" (RFC §4.4).
	out := renderAttrFragment(t, `<div :some-attr="0" :other-attr="''">x</div>`, nil)
	if !strings.Contains(out, `some-attr="0"`) {
		t.Errorf(`got %q, want it to contain some-attr="0"`, out)
	}
	if !strings.Contains(out, `other-attr=""`) {
		t.Errorf(`got %q, want it to contain other-attr=""`, out)
	}
}

func TestAttrOmit_Direct_DisabledFalse_StillOmits(t *testing.T) {
	// Unchanged behavior: :disabled="false" continues to omit the attribute.
	out := renderAttrFragment(t, `<button :disabled="false">x</button>`, nil)
	if strings.Contains(out, "disabled") {
		t.Errorf("got %q, want disabled omitted", out)
	}
}

func TestAttrOmit_Direct_DisabledZero_NoLongerOmits(t *testing.T) {
	// Intentional, RFC-driven narrowing (commit "feat: general
	// falsy-attribute omission for :attr/v-bind:attr bindings"): the
	// omission trigger for ALL attributes, including the original
	// isBooleanAttr eight, changed from "general JS-falsy" (expr.IsTruthy,
	// under which 0 is falsy) to "exactly false/nil/undefined"
	// (attrShouldOmit). 0 no longer omits disabled; since disabled is not
	// omitted and IS in the fixed isBooleanAttr list, it renders bare,
	// matching real Vue.js semantics (:disabled="0" does not omit in Vue
	// either). This behavior change is deliberate — do not "fix" this test
	// back to expecting omission.
	out := renderAttrFragment(t, `<button :disabled="0">x</button>`, nil)
	if !strings.Contains(out, "<button disabled>") {
		t.Errorf("got %q, want <button disabled> (bare, not omitted)", out)
	}
}

func TestAttrOmit_Direct_DisabledTrue_StillBare(t *testing.T) {
	// Unchanged behavior: :disabled="true" continues to render bare.
	out := renderAttrFragment(t, `<button :disabled="true">x</button>`, nil)
	if !strings.Contains(out, "<button disabled>") {
		t.Errorf("got %q, want <button disabled>", out)
	}
}

// --- v-bind="obj" spread binding site (applyAttrSpread) ---

func TestAttrOmit_Spread_HxSwapOobFalse_Omitted(t *testing.T) {
	out := renderAttrFragment(t, `<div v-bind='{"hx-swap-oob": false}'>x</div>`, nil)
	if strings.Contains(out, "hx-swap-oob") {
		t.Errorf("got %q, want hx-swap-oob omitted entirely", out)
	}
}

func TestAttrOmit_Spread_HxSwapOobTrue_RendersValued(t *testing.T) {
	out := renderAttrFragment(t, `<div v-bind='{"hx-swap-oob": true}'>x</div>`, nil)
	if !strings.Contains(out, `hx-swap-oob="true"`) {
		t.Errorf(`got %q, want it to contain hx-swap-oob="true" (valued, not bare)`, out)
	}
}

func TestAttrOmit_Spread_NilAndUndefined_Omitted(t *testing.T) {
	out := renderAttrFragment(t, `<div v-bind='{"some-attr": null, "other-attr": undefined}'>x</div>`, nil)
	if strings.Contains(out, "some-attr") {
		t.Errorf("got %q, want some-attr omitted (null)", out)
	}
	if strings.Contains(out, "other-attr") {
		t.Errorf("got %q, want other-attr omitted (undefined)", out)
	}
}

func TestAttrOmit_Spread_ZeroAndEmptyString_NotOmitted(t *testing.T) {
	out := renderAttrFragment(t, `<div v-bind='{"some-attr": 0, "other-attr": ""}'>x</div>`, nil)
	if !strings.Contains(out, `some-attr="0"`) {
		t.Errorf(`got %q, want it to contain some-attr="0"`, out)
	}
	if !strings.Contains(out, `other-attr=""`) {
		t.Errorf(`got %q, want it to contain other-attr=""`, out)
	}
}

func TestAttrOmit_Spread_DisabledFalse_StillOmits(t *testing.T) {
	out := renderAttrFragment(t, `<button v-bind='{"disabled": false}'>x</button>`, nil)
	if strings.Contains(out, "disabled") {
		t.Errorf("got %q, want disabled omitted", out)
	}
}

func TestAttrOmit_Spread_DisabledZero_NoLongerOmits(t *testing.T) {
	// Same intentional narrowing as the direct-binding case above, exercised
	// through the v-bind="obj" spread path (applyAttrSpread) to confirm both
	// call sites behave consistently for the same value.
	out := renderAttrFragment(t, `<button v-bind='{"disabled": 0}'>x</button>`, nil)
	if !strings.Contains(out, "<button disabled>") {
		t.Errorf("got %q, want <button disabled> (bare, not omitted)", out)
	}
}

func TestAttrOmit_Spread_DisabledTrue_StillBare(t *testing.T) {
	out := renderAttrFragment(t, `<button v-bind='{"disabled": true}'>x</button>`, nil)
	if !strings.Contains(out, "<button disabled>") {
		t.Errorf("got %q, want <button disabled>", out)
	}
}

// --- attrShouldOmit unit coverage ---

func TestAttrShouldOmit_Direct(t *testing.T) {
	cases := []struct {
		name string
		val  any
		want bool
	}{
		{"nil", nil, true},
		{"false", false, true},
		{"undefined", expr.Undefined, true},
		{"true", true, false},
		{"zero", float64(0), false},
		{"empty string", "", false},
		{"non-empty string", "x", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := attrShouldOmit(c.val); got != c.want {
				t.Errorf("attrShouldOmit(%#v) = %v, want %v", c.val, got, c.want)
			}
		})
	}
}
