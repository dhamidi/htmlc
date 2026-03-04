package htmlc_test

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dhamidi/htmlc"
)

// Example demonstrates end-to-end use of the htmlc engine: create an Engine
// from a directory of .vue files, then render a component as an HTML fragment.
func Example() {
	dir, err := os.MkdirTemp("", "htmlc-example-*")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)

	// Write a simple component with no scoped styles so the output is stable.
	vue := `<template><p>Hello, {{ name }}!</p></template>`
	if err := os.WriteFile(filepath.Join(dir, "Greeting.vue"), []byte(vue), 0644); err != nil {
		log.Fatal(err)
	}

	engine, err := htmlc.New(htmlc.Options{ComponentDir: dir})
	if err != nil {
		log.Fatal(err)
	}

	out, err := engine.RenderFragmentString("Greeting", map[string]any{"name": "World"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// <p>Hello, World!</p>
}
