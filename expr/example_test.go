package expr_test

import (
	"fmt"
	"sort"

	"github.com/dhamidi/htmlc/expr"
)

// To look up nested data, evaluate a dot-chained expression against a nested Go map.
func ExampleParse_memberAccess() {
	scope := map[string]any{
		"user": map[string]any{
			"address": map[string]any{
				"city": "Berlin",
			},
		},
	}
	result, _ := expr.Eval("user.address.city", scope)
	fmt.Println(result)
	// Output:
	// Berlin
}

// To apply conditional styling, use a ternary expression to select a CSS class.
func ExampleParse_ternary() {
	scope := map[string]any{"active": true}
	result, _ := expr.Eval(`active ? "btn-active" : "btn"`, scope)
	fmt.Println(result)
	// Output:
	// btn-active
}

// To safely read the first element of a list, guard the access with .length.
func ExampleParse_filterWithLen() {
	scope := map[string]any{"items": []any{"first", "second"}}
	result, _ := expr.Eval(`items.length > 0 ? items[0] : null`, scope)
	fmt.Println(result)
	// Output:
	// first
}

// To detect a missing map key, compare the result against expr.Undefined rather than nil.
func ExampleParse_undefinedKey() {
	scope := map[string]any{"user": map[string]any{"name": "Alice"}}
	result, _ := expr.Eval("user.age", scope)
	fmt.Println(result == expr.Undefined)
	fmt.Println(expr.IsTruthy(result))
	// Output:
	// true
	// false
}

// To provide a fallback when a variable is empty or absent, use || with a default string.
func ExampleParse_logicalShortCircuit() {
	scope := map[string]any{"title": ""}
	result, _ := expr.Eval(`title || "Untitled"`, scope)
	fmt.Println(result)
	// Output:
	// Untitled
}

// To discover which scope keys an expression reads, use CollectIdentifiers for prop discovery.
func ExampleCollectIdentifiers() {
	ids, _ := expr.CollectIdentifiers("user.name + suffix")
	sort.Strings(ids)
	fmt.Println(ids)
	// Output:
	// [suffix user]
}
