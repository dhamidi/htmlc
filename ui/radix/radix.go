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
package radix

import (
	"embed"
	"io/fs"
)

// componentsFS embeds every .vue Single File Component under components/.
//
// NOTE: components/Placeholder.vue exists only to satisfy this embed glob's
// requirement of matching at least one file; it is not a real component and
// should be deleted once the first real component (Accordion.vue) lands.
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
