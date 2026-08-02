// Package radix is a small, in-tree, Radix-inspired component library for
// htmlc. It ships as its own independently-versioned Go module
// (github.com/dhamidi/htmlc/ui/radix) and has no dependency on the root
// github.com/dhamidi/htmlc module — it depends only on the standard
// library ("embed" and "io/fs"). A consuming project mounts it into an
// htmlc engine via Options.Mounts, e.g.:
//
//	import radixui "github.com/dhamidi/htmlc/ui/radix"
//
//	engine, err := htmlc.New(htmlc.Options{
//	    ComponentDir: "templates/",
//	    Mounts: []htmlc.Mount{
//	        {Prefix: "radix", FS: radixui.FS(), Dir: "components"},
//	    },
//	})
//
// This package deliberately receives no special treatment from htmlc: it is
// a directory of .vue files plus an exported FS() fs.FS over a //go:embed,
// the same convention any third-party component package can follow.
//
// # Mount prefix and custom-element tag names
//
// ui/radix's .vue components are authored standalone, with no way to know
// at authoring time what Prefix a given consumer will choose for its
// Mount. Their <script customelement> blocks therefore hardcode tag names
// (e.g. customElements.define('radix-accordion', ...)) that assume this
// package is mounted with Mount{Prefix: "radix", ...}, matching every
// example in RFC 014. Mounting under a different prefix still renders
// components correctly via template references (<Accordion>, aliases,
// <component is="...">, etc.) — only the <script customelement>
// enhancement's auto-activation depends on the tag name matching, and it
// won't auto-activate under a mismatched prefix. This is the same kind of
// manual-synchronization contract every custom-element component already
// has with its own file path today; there is no general mechanism in this
// codebase to template the tag name dynamically.
//
// The tag itself is derived deterministically by htmlc's own
// deriveCustomElementTag (component.go): drop the ".vue" extension, convert
// each path segment from PascalCase/camelCase to kebab-case at every
// lowercase-followed-by-uppercase boundary (e.g. "ContextMenu.vue" ->
// "context-menu"), and join segments with "-". Since every component here
// assumes Mount{Prefix: "radix", ...} (above), the tag actually registered
// is that derivation with a "radix-" prefix, e.g. "ContextMenu.vue" ->
// "radix-context-menu". Individual component files state only their own
// resulting tag name, not this algorithm.
//
// # Design tokens
//
// This package's components share one set of CSS custom-property design
// tokens — --radix-gray-*/--radix-blue-*/--radix-red-* (mirroring real
// Radix Colors' own gray/blue/red scales) and --radix-radius-*/
// --radix-space-* (mirroring Radix Themes' default radius/spacing scale)
// — plus one shared utility class, .radix-visually-hidden-input. The
// canonical, single-source-of-truth copy of this block lives in
// Button.vue's own <style> section; every other component includes the
// exact same block, byte-for-byte, in its own unscoped <style> section.
// htmlc's StyleCollector (style.go) dedupes style contributions by exact
// (scope, CSS text) match, so this block reaches the final rendered page
// only once no matter how many of this package's components a given page
// mounts — see Button.vue's header comment for the fuller explanation and
// the values themselves.
package radix

import (
	"embed"
	"io/fs"
)

// componentsFS embeds every .vue Single File Component under components/.
//
//go:embed components/*.vue
var componentsFS embed.FS

// FS returns the embedded filesystem containing this module's .vue
// components, rooted such that files appear under "components/...". Pass it
// as a Mount's FS field (with Dir: "components") to mount this package's
// components into an htmlc engine.
func FS() fs.FS {
	return componentsFS
}
