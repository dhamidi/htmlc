package htmlc_test

import (
	"fmt"
	htmltemplate "html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhamidi/htmlc"
)

// ExampleEngine_CompileToTemplate demonstrates compiling a .vue component to
// a *html/template.Template and executing it with Go template data.
//
// The component name is lowercased to form the template name: "Hello" → "hello".
// Sub-components referenced by the root are included as named {{ define }} blocks
// in the same template set.
func ExampleEngine_CompileToTemplate() {
	dir, err := os.MkdirTemp("", "htmlc-ex-compile-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// A .vue component with a simple text interpolation.
	vue := `<template><p>{{ message }}</p></template>`
	if err := os.WriteFile(filepath.Join(dir, "Hello.vue"), []byte(vue), 0644); err != nil {
		log.Fatal(err)
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: dir})
	if err != nil {
		log.Fatal(err)
	}

	// CompileToTemplate converts the .vue component to a *html/template.Template.
	tmpl, err := engine.CompileToTemplate("Hello")
	if err != nil {
		log.Fatal(err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{"message": "Hello"}); err != nil {
		log.Fatal(err)
	}
	fmt.Println(buf.String())
	// Output:
	// <p>Hello</p>
}

// ExampleEngine_RegisterTemplate demonstrates registering an existing
// *html/template.Template as an htmlc component so it can be used inside
// .vue component trees.
//
// After registration, the component name resolves normally inside any .vue file:
// <site-header /> renders the registered template's output.
func ExampleEngine_RegisterTemplate() {
	engine, err := htmlc.New(htmlc.Options{})
	if err != nil {
		log.Fatal(err)
	}

	// An existing html/template from a legacy codebase.
	headerTmpl := htmltemplate.Must(
		htmltemplate.New("site-header").Parse(`<header>Site Title</header>`),
	)

	// Register it as an htmlc component. All named {{ define }} blocks within
	// headerTmpl are also registered under their own names.
	if err := engine.RegisterTemplate("site-header", headerTmpl); err != nil {
		log.Fatal(err)
	}

	fmt.Println(engine.Has("site-header"))
	// Output:
	// true
}
