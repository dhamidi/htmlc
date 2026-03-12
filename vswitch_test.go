package htmlc

import (
	"path/filepath"
	"strings"
	"testing"
)

// vswitchEngine creates a test engine with the given components available.
// VSwitch is enabled by default so no explicit directive registration is needed.
func vswitchEngine(t *testing.T, components map[string]string) *Engine {
	t.Helper()
	dir := t.TempDir()
	for name, tmpl := range components {
		writeVue(t, filepath.Join(dir, name+".vue"), "<template>"+tmpl+"</template>")
	}
	e, err := New(Options{
		ComponentDir: dir,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return e
}

// TestVSwitch_BasicDispatch verifies that v-switch="'Card'" replaces the host
// element with the Card component's output.
func TestVSwitch_BasicDispatch(t *testing.T) {
	e := vswitchEngine(t, map[string]string{
		"Host": `<div v-switch="'Card'"></div>`,
		"Card": `<article>card</article>`,
	})
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "card") {
		t.Errorf("got %q, want 'card' from Card component", out)
	}
	if strings.Contains(out, "<div") {
		t.Errorf("got %q, host <div> should be replaced by Card", out)
	}
}

// TestVSwitch_ExpressionEvaluation verifies that v-switch="item.type" evaluates
// the expression and dispatches to the matching component.
func TestVSwitch_ExpressionEvaluation(t *testing.T) {
	e := vswitchEngine(t, map[string]string{
		"Host":   `<div v-switch="item.type"></div>`,
		"Banner": `<section>banner</section>`,
	})
	out, err := e.RenderFragmentString("Host", map[string]any{
		"item": map[string]any{"type": "Banner"},
	})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "banner") {
		t.Errorf("got %q, want 'banner' from Banner component", out)
	}
}

// TestVSwitch_AttributeForwarding verifies that attributes on the host element
// (other than v-switch) are forwarded as props to the resolved component.
func TestVSwitch_AttributeForwarding(t *testing.T) {
	e := vswitchEngine(t, map[string]string{
		"Host": `<div v-switch="'Card'" :title="item.title"></div>`,
		"Card": `<div>{{ title }}</div>`,
	})
	out, err := e.RenderFragmentString("Host", map[string]any{
		"item": map[string]any{"title": "My Title"},
	})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "My Title") {
		t.Errorf("got %q, want 'My Title' forwarded as prop", out)
	}
}

// TestVSwitch_NonStringExpression verifies that a non-string expression value
// causes a descriptive error.
func TestVSwitch_NonStringExpression(t *testing.T) {
	e := vswitchEngine(t, map[string]string{
		"Host": `<div v-switch="42"></div>`,
	})
	_, err := e.RenderFragmentString("Host", nil)
	if err == nil {
		t.Fatal("expected error for non-string v-switch expression, got nil")
	}
	if !strings.Contains(err.Error(), "v-switch") {
		t.Errorf("error %q should mention v-switch", err.Error())
	}
}

// TestVSwitch_EmptyString verifies that an empty string expression causes a
// descriptive error.
func TestVSwitch_EmptyString(t *testing.T) {
	e := vswitchEngine(t, map[string]string{
		"Host": `<div v-switch="emptyVal"></div>`,
	})
	_, err := e.RenderFragmentString("Host", map[string]any{"emptyVal": ""})
	if err == nil {
		t.Fatal("expected error for empty string v-switch, got nil")
	}
	if !strings.Contains(err.Error(), "v-switch") {
		t.Errorf("error %q should mention v-switch", err.Error())
	}
}

// TestVSwitch_UnknownComponent verifies that dispatching to an unregistered
// component name returns a descriptive error.
func TestVSwitch_UnknownComponent(t *testing.T) {
	e := vswitchEngine(t, map[string]string{
		"Host": `<div v-switch="'NoSuchComponent'"></div>`,
	})
	_, err := e.RenderFragmentString("Host", nil)
	if err == nil {
		t.Fatal("expected error for unknown component, got nil")
	}
	if !strings.Contains(err.Error(), "NoSuchComponent") {
		t.Errorf("error %q should mention component name", err.Error())
	}
}

// TestVSwitch_DefaultEnabled verifies that v-switch works without any explicit
// directive registration — VSwitch is auto-registered by New.
func TestVSwitch_DefaultEnabled(t *testing.T) {
	dir := t.TempDir()
	writeVue(t, filepath.Join(dir, "Host.vue"),
		"<template><div v-switch=\"'Card'\"></div></template>")
	writeVue(t, filepath.Join(dir, "Card.vue"),
		"<template><article>card</article></template>")
	e, err := New(Options{ComponentDir: dir})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	out, err := e.RenderFragmentString("Host", nil)
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "card") {
		t.Errorf("got %q, want Card output when v-switch enabled by default", out)
	}
}

// TestVSwitch_CaseInsensitiveResolution verifies that providing "card"
// (lowercase) dispatches to a component registered as "Card".
func TestVSwitch_CaseInsensitiveResolution(t *testing.T) {
	e := vswitchEngine(t, map[string]string{
		"Host": `<div v-switch="lowerName"></div>`,
		"Card": `<article>card</article>`,
	})
	out, err := e.RenderFragmentString("Host", map[string]any{"lowerName": "card"})
	if err != nil {
		t.Fatalf("RenderFragmentString: %v", err)
	}
	if !strings.Contains(out, "card") {
		t.Errorf("got %q, want Card rendered for lowercase name", out)
	}
}
