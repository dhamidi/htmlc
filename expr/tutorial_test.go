package expr_test

import (
	"sort"
	"testing"

	"github.com/dhamidi/htmlc/expr"
)

// TestTutorial walks a complete beginner through the expr package from first
// principles. Each sub-test introduces one new idea; together they build the
// mental model needed to use the package confidently.
func TestTutorial(t *testing.T) {

	// Step 1: Evaluate a number literal.
	//
	// The most basic thing the package can do is parse and evaluate a literal
	// value. Parse turns source text into an Expr — a compiled, reusable
	// representation of the expression. Eval executes that Expr against a
	// scope. Passing nil for the scope is fine when there are no identifiers to
	// resolve. All numbers in the evaluator are represented as float64,
	// mirroring JavaScript's single numeric type.
	t.Run("number literal", func(t *testing.T) {
		e, err := expr.Parse("42")
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		result, err := e.Eval(nil)
		if err != nil {
			t.Fatalf("Eval: %v", err)
		}
		if result != float64(42) {
			t.Fatalf("got %v (%T), want float64(42)", result, result)
		}
	})

	// Step 2: Evaluate an identifier from scope.
	//
	// Expressions become useful when they reference runtime data. The scope is
	// a plain map[string]any: the evaluator looks up each identifier by name
	// and returns the corresponding value. Any Go value can live in the scope.
	t.Run("identifier from scope", func(t *testing.T) {
		e, err := expr.Parse("name")
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		scope := map[string]any{"name": "Alice"}
		result, err := e.Eval(scope)
		if err != nil {
			t.Fatalf("Eval: %v", err)
		}
		if result != "Alice" {
			t.Fatalf("got %v, want %q", result, "Alice")
		}
	})

	// Step 3: Use arithmetic with scope variables.
	//
	// Binary operators work directly on scope variables. Here * multiplies two
	// values drawn from the scope. The evaluator coerces numeric scope values
	// to float64 automatically, so you can store int or float64 in the scope
	// and the result is always float64.
	t.Run("arithmetic", func(t *testing.T) {
		e, err := expr.Parse("price * quantity")
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		scope := map[string]any{
			"price":    float64(9.99),
			"quantity": float64(3),
		}
		result, err := e.Eval(scope)
		if err != nil {
			t.Fatalf("Eval: %v", err)
		}
		want := 9.99 * 3
		if result != want {
			t.Fatalf("got %v, want %v", result, want)
		}
	})

	// Step 4: Access a nested field with dot notation.
	//
	// Dot notation (user.name) traverses nested maps in the scope. Each
	// segment of the dot chain looks up the next key in the map returned by
	// the previous segment. This mirrors how you'd navigate a JSON object in
	// JavaScript or a nested Go map in ordinary code.
	t.Run("nested field access", func(t *testing.T) {
		e, err := expr.Parse("user.name")
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		scope := map[string]any{
			"user": map[string]any{
				"name": "Bob",
			},
		}
		result, err := e.Eval(scope)
		if err != nil {
			t.Fatalf("Eval: %v", err)
		}
		if result != "Bob" {
			t.Fatalf("got %v, want %q", result, "Bob")
		}
	})

	// Step 5: Use a conditional (ternary) expression.
	//
	// The ternary operator condition ? consequent : alternate lets you branch
	// inside an expression without writing a full if statement in Go. String
	// literals inside expressions use double quotes. This is the primary way
	// to produce different output based on a runtime condition — for example,
	// switching a label or a CSS class depending on state.
	t.Run("conditional expression", func(t *testing.T) {
		e, err := expr.Parse(`stock > 0 ? "In stock" : "Out of stock"`)
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}

		inStock := map[string]any{"stock": float64(5)}
		result, err := e.Eval(inStock)
		if err != nil {
			t.Fatalf("Eval (in stock): %v", err)
		}
		if result != "In stock" {
			t.Fatalf("got %v, want %q", result, "In stock")
		}

		outOfStock := map[string]any{"stock": float64(0)}
		result, err = e.Eval(outOfStock)
		if err != nil {
			t.Fatalf("Eval (out of stock): %v", err)
		}
		if result != "Out of stock" {
			t.Fatalf("got %v, want %q", result, "Out of stock")
		}
	})

	// Step 6: Handle a missing key gracefully.
	//
	// When an identifier is not present in the scope, the evaluator does not
	// return an error. Instead it returns expr.UndefinedValue{} — the same
	// sentinel exposed as expr.Undefined. This distinguishes "the key was
	// absent" from "the key was present but its value was nil (null)". Code
	// that consumes expression results should compare against expr.Undefined
	// rather than nil when checking for missing data.
	t.Run("missing key is undefined not error", func(t *testing.T) {
		e, err := expr.Parse("missing")
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		result, err := e.Eval(map[string]any{})
		if err != nil {
			t.Fatalf("Eval: unexpected error: %v", err)
		}
		if result != expr.Undefined {
			t.Fatalf("got %v (%T), want expr.UndefinedValue{}", result, result)
		}
	})

	// Step 7: Discover what identifiers an expression depends on.
	//
	// CollectIdentifiers parses an expression and returns the deduplicated set
	// of top-level identifier names it references. This is useful for
	// discovering which scope keys an expression reads — for example, to
	// pre-fetch only the props a component actually uses, or to validate that
	// all required variables are present before evaluating.
	t.Run("collect identifiers", func(t *testing.T) {
		ids, err := expr.CollectIdentifiers("price * quantity + tax")
		if err != nil {
			t.Fatalf("CollectIdentifiers: %v", err)
		}
		sort.Strings(ids)
		want := []string{"price", "quantity", "tax"}
		if len(ids) != len(want) {
			t.Fatalf("got %v, want %v", ids, want)
		}
		for i := range want {
			if ids[i] != want[i] {
				t.Fatalf("ids[%d] = %q, want %q", i, ids[i], want[i])
			}
		}
	})
}
