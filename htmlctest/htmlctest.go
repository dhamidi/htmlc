// Package htmlctest provides helpers for testing htmlc components.
//
// # Getting started
//
// Use [NewHarness] to create a test harness backed by an in-memory filesystem:
//
//	func TestGreeting(t *testing.T) {
//	    h := htmlctest.NewHarness(t, map[string]string{
//	        "Greeting.vue": `<template><p>Hello {{ name }}!</p></template>`,
//	    })
//	    h.Fragment("Greeting", map[string]any{"name": "World"}).
//	        AssertHTML("<p>Hello World!</p>")
//	}
//
// For single-component tests, use the [Build] shorthand:
//
//	htmlctest.Build(t, `<p>Hello {{ name }}!</p>`).
//	    Fragment("Root", map[string]any{"name": "World"}).
//	    Find(htmlctest.ByTag("p")).AssertText("Hello World!")
//
// # Querying elements
//
// [Result.Find] accepts a [Query] built from the [ByTag], [ByClass], and
// [ByAttr] constructors. Queries are immutable values; the [Query.WithClass],
// [Query.WithAttr], and [Query.Descendant] combinators return new Query values:
//
//	r.Find(htmlctest.ByTag("li").WithClass("active")).AssertCount(1)
//	r.Find(htmlctest.ByTag("p").Descendant(htmlctest.ByTag("div"))).AssertExists()
//
// The returned [Selection] supports fluent assertions that call [testing.TB.Fatalf]
// on failure and return the receiver to allow chaining.
package htmlctest
