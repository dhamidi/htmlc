package htmlc

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// VHighlight is an example custom directive that sets the background colour of
// the host element. It is the htmlc equivalent of the v-highlight directive
// shown in the Vue.js custom directives guide
// (https://vuejs.org/guide/reusability/custom-directives.html).
//
// Register it on an engine and then use v-highlight in templates:
//
//	engine.RegisterDirective("highlight", &htmlc.VHighlight{})
//
// Template usage:
//
//	<p v-highlight="'yellow'">Highlight this text bright yellow</p>
//
// The expression must evaluate to a non-empty CSS colour string. If the host
// element already has a style attribute, the background property is appended;
// existing style declarations are preserved.
type VHighlight struct{}

// Created merges `background: <colour>` into the host element's style attribute.
func (v *VHighlight) Created(node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error {
	colour, _ := binding.Value.(string)
	if colour == "" {
		return nil // nothing to do
	}

	bgDecl := "background:" + colour

	// Locate an existing style attribute and merge.
	for i, attr := range node.Attr {
		if attr.Key == "style" {
			existing := strings.TrimRight(strings.TrimSpace(attr.Val), ";")
			if existing != "" {
				node.Attr[i].Val = existing + ";" + bgDecl
			} else {
				node.Attr[i].Val = bgDecl
			}
			return nil
		}
	}

	// No existing style attribute — append a new one.
	node.Attr = append(node.Attr, html.Attribute{Key: "style", Val: bgDecl})
	return nil
}

// Mounted is a no-op for VHighlight.
func (v *VHighlight) Mounted(_ io.Writer, _ *html.Node, _ DirectiveBinding, _ DirectiveContext) error {
	return nil
}
