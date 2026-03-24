package htmlc

import (
	"strings"
	"testing"

	"github.com/dhamidi/htmlc/internal/testhelpers"
)

// switchEngine creates an Engine with the given components for switch directive testing.
func switchEngine(t *testing.T, components map[string]string) *Engine {
	t.Helper()
	dir := t.TempDir()
	for name, tmpl := range components {
		testhelpers.WriteVue(t, dir, name+".vue", "<template>"+tmpl+"</template>")
	}
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

func TestVSwitch_BasicCaseMatch(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="'b'"><div v-case="'a'">alpha</div><div v-case="'b'">beta</div></template>`,
	})
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("got %q, want 'beta'", out)
	}
	if strings.Contains(out, "alpha") {
		t.Errorf("got %q, 'alpha' branch should not render", out)
	}
}

func TestVSwitch_NoMatchRendersDefault(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="'x'"><div v-case="'a'">alpha</div><div v-case="'b'">beta</div><div v-default>fallback</div></template>`,
	})
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "fallback") {
		t.Errorf("got %q, want 'fallback'", out)
	}
	if strings.Contains(out, "alpha") || strings.Contains(out, "beta") {
		t.Errorf("got %q, case branches should not render", out)
	}
}

func TestVSwitch_NoMatchNoDefault(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="'x'"><div v-case="'a'">alpha</div></template>`,
	})
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if strings.TrimSpace(out) != "" {
		t.Errorf("got %q, want empty output", out)
	}
}

func TestVSwitch_DynamicExpression(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="tab"><div v-case="'home'">home content</div><div v-case="'settings'">settings content</div></template>`,
	})
	out, err := e.RenderFragmentString("Host", map[string]any{"tab": "settings"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "settings content") {
		t.Errorf("got %q, want 'settings content'", out)
	}
	if strings.Contains(out, "home content") {
		t.Errorf("got %q, 'home content' branch should not render", out)
	}
}

func TestVSwitch_FirstMatchWins(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="'a'"><div v-case="'a'">first</div><div v-case="'a'">second</div></template>`,
	})
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "first") {
		t.Errorf("got %q, want 'first'", out)
	}
	if strings.Contains(out, "second") {
		t.Errorf("got %q, only first match should render", out)
	}
}

func TestVSwitch_DefaultSkippedWhenMatched(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="'a'"><div v-case="'a'">matched</div><div v-default>default content</div></template>`,
	})
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "matched") {
		t.Errorf("got %q, want 'matched'", out)
	}
	if strings.Contains(out, "default content") {
		t.Errorf("got %q, default branch should not render when case matches", out)
	}
}

func TestVSwitch_ComponentChildren(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host":   `<template v-switch="'card'"><Card v-case="'card'" /><Banner v-default /></template>`,
		"Card":   `<div class="card">card content</div>`,
		"Banner": `<div class="banner">banner content</div>`,
	})
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "card content") {
		t.Errorf("got %q, want 'card content'", out)
	}
	if strings.Contains(out, "banner content") {
		t.Errorf("got %q, Banner should not render when Card case matches", out)
	}
}

func TestVSwitch_SwitchExprError(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="badExpr["><div v-case="'a'">a</div></template>`,
	})
	_, err := e.RenderFragmentString("Host", nil)
	if err == nil {
		t.Fatal("expected error for invalid v-switch expression, got nil")
	}
	if !strings.Contains(err.Error(), "v-switch") {
		t.Errorf("error %q should mention 'v-switch'", err.Error())
	}
}

func TestVSwitch_CaseExprError(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="'a'"><div v-case="badExpr[">a</div></template>`,
	})
	_, err := e.RenderFragmentString("Host", nil)
	if err == nil {
		t.Fatal("expected error for invalid v-case expression, got nil")
	}
	if !strings.Contains(err.Error(), "v-case") {
		t.Errorf("error %q should mention 'v-case'", err.Error())
	}
}

func TestVSwitch_NonTemplateElement(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<div v-switch="'a'"><div v-case="'a'">yes</div></div>`,
	})
	_, err := e.RenderFragmentString("Host", nil)
	if err == nil {
		t.Fatal("expected error for v-switch on non-template element, got nil")
	}
	if !strings.Contains(err.Error(), "v-switch") {
		t.Errorf("error %q should mention 'v-switch'", err.Error())
	}
}

func TestVSwitch_ChildrenWithoutCaseIgnored(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="'b'"><div v-case="'a'">alpha</div><div>plain child</div><div v-case="'b'">beta</div></template>`,
	})
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "beta") {
		t.Errorf("got %q, want 'beta'", out)
	}
	if strings.Contains(out, "plain child") {
		t.Errorf("got %q, plain child should be silently ignored", out)
	}
	if strings.Contains(out, "alpha") {
		t.Errorf("got %q, 'alpha' branch should not render", out)
	}
}

func TestVSwitch_IntegerSwitchValue(t *testing.T) {
	e := switchEngine(t, map[string]string{
		"Host": `<template v-switch="count"><div v-case="1">one</div><div v-case="2">two</div><div v-default>other</div></template>`,
	})
	// The expr evaluator parses numeric literals as float64, so provide float64 in scope.
	out, err := e.RenderFragmentString("Host", map[string]any{"count": float64(2)})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "two") {
		t.Errorf("got %q, want 'two'", out)
	}
	if strings.Contains(out, "one") || strings.Contains(out, "other") {
		t.Errorf("got %q, only case 2 should render", out)
	}
}

func TestVSwitch_NilSwitchValue(t *testing.T) {
	e := switchEngine(t, map[string]string{
		// The expr evaluator uses "null" (not "nil") as the null literal.
		"Host": `<template v-switch="missing"><div v-case="null">nil matched</div><div v-default>no match</div></template>`,
	})
	// Provide missing=nil so the switch value is nil; v-case="null" also evaluates to nil.
	out, err := e.RenderFragmentString("Host", map[string]any{"missing": nil})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "nil matched") {
		t.Errorf("got %q, want 'nil matched' (nil == nil should match)", out)
	}
	if strings.Contains(out, "no match") {
		t.Errorf("got %q, default should not render when nil matches", out)
	}
}
