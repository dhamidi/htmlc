package htmlc

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// VSwitch is a built-in custom directive that replaces the host element with a
// registered component whose name is given by the directive's expression.
//
// Usage in a template:
//
//	<div v-switch="item.type" :title="item.title" />
//
// When rendered, the <div> tag is replaced by the component whose name matches
// the evaluated value of item.type (e.g. "CardWidget" or "card-widget"). Any
// other attributes on the host element (:title, class, etc.) are forwarded as
// props to the target component.
//
// Registration:
//
//	engine.RegisterDirective("switch", &htmlc.VSwitch{})
//
// VSwitch implements Directive via its Created hook; Mounted is a no-op.
type VSwitch struct{}

// Created changes the host element's tag to the component name supplied by the
// directive expression, and removes the v-switch attribute. After Created
// returns, the renderer sees a node whose Data is the component name and
// resolves it normally from the registry.
func (v *VSwitch) Created(node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error {
	compName, ok := binding.Value.(string)
	if !ok || compName == "" {
		return fmt.Errorf("v-switch: expression %q must evaluate to a non-empty string component name, got %T", binding.RawExpr, binding.Value)
	}

	// Verify the component exists in the registry (if one is attached).
	if ctx.Registry != nil {
		lower := strings.ToLower(compName)
		found := false
		for key := range ctx.Registry {
			if strings.ToLower(key) == lower {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("v-switch: component %q not found in registry", compName)
		}
	}

	// Redirect the host element to the target component.
	node.Data = compName

	// Remove the v-switch attribute from the working node so it is not
	// forwarded as an unknown attribute to the component.
	var filtered []html.Attribute
	for _, a := range node.Attr {
		if a.Key == "v-switch" {
			continue
		}
		filtered = append(filtered, a)
	}
	node.Attr = filtered

	return nil
}

// Mounted is a no-op for VSwitch.
func (v *VSwitch) Mounted(_ io.Writer, _ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	return nil
}
