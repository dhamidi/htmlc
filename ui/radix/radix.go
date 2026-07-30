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
