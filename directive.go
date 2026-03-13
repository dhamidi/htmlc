package htmlc

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// DirectiveBinding holds the evaluated binding for a custom directive invocation.
type DirectiveBinding struct {
	// Value is the result of evaluating the directive expression against the
	// current scope. For example, v-switch="item.type" yields item.type's value.
	Value any

	// RawExpr is the un-evaluated expression string from the template attribute.
	RawExpr string

	// Arg is the directive argument after the colon, e.g. "href" in v-bind:href.
	// Empty string when no argument is present.
	Arg string

	// Modifiers is the set of dot-separated modifiers, e.g. {"prevent": true}
	// from v-on:click.prevent. Empty map when no modifiers are present.
	Modifiers map[string]bool
}

// DirectiveContext provides directive hooks read-only access to renderer state.
type DirectiveContext struct {
	// Registry is the component registry the renderer is using. Directives
	// can use this to verify or resolve component names.
	Registry Registry

	// RenderedChildHTML is the fully rendered inner HTML of the directive's
	// host element, with all template expressions evaluated and child
	// components expanded. It is empty for void elements.
	// Available in both Created and Mounted hooks.
	RenderedChildHTML string
}

// Directive is the interface implemented by custom directive types.
//
// Register a custom directive with Engine.RegisterDirective or
// Renderer.WithDirectives. In a template, reference it as v-<name>:
//
//	<div v-my-directive="someExpr">…</div>
//
// Only the Created and Mounted hooks are called because htmlc renders server-
// side. There are no DOM updates, component unmounting, or browser events.
type Directive interface {
	// Created is called before the element is rendered. The hook receives a
	// shallow-cloned working node whose Attr slice and Data field may be
	// freely mutated; mutations affect what the renderer emits for this
	// element but do not modify the shared parsed template.
	//
	// Common uses:
	//   - Add, remove, or rewrite attributes (node.Attr).
	//   - Change the element tag (node.Data) to redirect rendering to a
	//     different component.
	//   - Return a non-nil error to abort rendering of this element.
	Created(node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error

	// Mounted is called after the element's closing tag has been written to w.
	// The hook may write additional HTML after the element.
	//
	// w is the same writer the renderer uses; bytes written here appear
	// immediately after the element in the output stream.
	//
	// Return a non-nil error to abort rendering.
	Mounted(w io.Writer, node *html.Node, binding DirectiveBinding, ctx DirectiveContext) error
}

// DirectiveWithContent is an optional extension of the Directive interface.
// When a directive's Created hook wants to replace the element's children with
// custom HTML it should implement this interface.
//
// The renderer checks for this interface after calling Created.  If
// InnerHTML returns a non-empty string the element's template children are
// skipped and the string is written verbatim between the opening and closing
// tags (equivalent to v-html on the element itself).
type DirectiveWithContent interface {
	Directive
	// InnerHTML returns the raw HTML to use as the element's inner content.
	// Return ("", false) to fall back to normal child rendering.
	InnerHTML() (html string, ok bool)
}

// DirectiveRegistry maps directive names (without the "v-" prefix) to their
// implementations. Keys are lower-kebab-case; the renderer normalises names
// before lookup.
type DirectiveRegistry map[string]Directive

// parseDirectiveKey splits a raw attribute key like "v-switch:arg.mod1.mod2"
// into its name ("switch"), arg ("arg"), and modifiers ({"mod1":true,"mod2":true}).
// Returns name=="", arg=="", nil map when the key is not a v-directive or is
// one of the built-in directives handled elsewhere.
func parseDirectiveKey(key string) (name, arg string, modifiers map[string]bool) {
	if !strings.HasPrefix(key, "v-") {
		return
	}
	body := key[2:] // strip "v-"

	// Split arg.
	if i := strings.IndexByte(body, ':'); i >= 0 {
		arg = body[i+1:]
		body = body[:i]
	}

	// Split modifiers from arg.
	if i := strings.IndexByte(arg, '.'); i >= 0 {
		parts := strings.Split(arg[i+1:], ".")
		modifiers = make(map[string]bool, len(parts))
		for _, p := range parts {
			modifiers[p] = true
		}
		arg = arg[:i]
	} else if i := strings.IndexByte(body, '.'); i >= 0 {
		parts := strings.Split(body[i+1:], ".")
		modifiers = make(map[string]bool, len(parts))
		for _, p := range parts {
			modifiers[p] = true
		}
		body = body[:i]
	}

	name = body
	return
}
