package expr

import (
	"math"
	"testing"
)

// eval is a short helper that compiles and evaluates src.
func eval(t *testing.T, src string, scope map[string]any) any {
	t.Helper()
	v, err := Eval(src, scope)
	if err != nil {
		t.Fatalf("Eval(%q): unexpected error: %v", src, err)
	}
	return v
}

func evalErr(t *testing.T, src string, scope map[string]any) error {
	t.Helper()
	_, err := Eval(src, scope)
	if err == nil {
		t.Fatalf("Eval(%q): expected error, got nil", src)
	}
	return err
}

// TestLiteralEval verifies that literal nodes return their values.
func TestLiteralEval(t *testing.T) {
	if v := eval(t, "42", nil); v != float64(42) {
		t.Errorf("got %v (%T), want float64(42)", v, v)
	}
	if v := eval(t, "3.14", nil); v != 3.14 {
		t.Errorf("got %v, want 3.14", v)
	}
	if v := eval(t, "'hello'", nil); v != "hello" {
		t.Errorf("got %v, want hello", v)
	}
	if v := eval(t, "true", nil); v != true {
		t.Errorf("got %v, want true", v)
	}
	if v := eval(t, "false", nil); v != false {
		t.Errorf("got %v, want false", v)
	}
	if v := eval(t, "null", nil); v != nil {
		t.Errorf("got %v, want nil", v)
	}
	if v := eval(t, "undefined", nil); v != Undefined {
		t.Errorf("got %v, want Undefined", v)
	}
}

// TestArithmetic verifies numeric binary operators.
func TestArithmetic(t *testing.T) {
	cases := []struct {
		src  string
		want float64
	}{
		{"1 + 2", 3},
		{"10 - 3", 7},
		{"4 * 5", 20},
		{"10 / 4", 2.5},
		{"10 % 3", 1},
		{"2 ** 10", 1024},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			v := eval(t, tc.src, nil)
			if v != tc.want {
				t.Errorf("got %v, want %v", v, tc.want)
			}
		})
	}
}

// TestStringConcat verifies that + concatenates when either side is a string.
func TestStringConcat(t *testing.T) {
	if v := eval(t, "'foo' + 'bar'", nil); v != "foobar" {
		t.Errorf("got %v, want foobar", v)
	}
	if v := eval(t, "1 + '2'", nil); v != "12" {
		t.Errorf("got %v, want 12", v)
	}
	if v := eval(t, "'x' + 1", nil); v != "x1" {
		t.Errorf("got %v, want x1", v)
	}
}

// TestComparisons verifies relational and equality operators.
func TestComparisons(t *testing.T) {
	trueCases := []string{"1 < 2", "2 <= 2", "3 > 2", "3 >= 3", "1 == 1", "1 === 1"}
	for _, src := range trueCases {
		t.Run(src, func(t *testing.T) {
			if v := eval(t, src, nil); v != true {
				t.Errorf("got %v, want true", v)
			}
		})
	}
	falseCases := []string{"2 < 1", "3 <= 2", "1 > 2", "2 >= 3", "1 == 2", "1 === 2"}
	for _, src := range falseCases {
		t.Run(src, func(t *testing.T) {
			if v := eval(t, src, nil); v != false {
				t.Errorf("got %v, want false", v)
			}
		})
	}
}

// TestLooseEquality verifies JS-style loose equality (==).
func TestLooseEquality(t *testing.T) {
	trueExprs := []string{
		"0 == false",
		"1 == true",
		"null == undefined",
		"0 == ''",
		"'1' == 1",
	}
	for _, src := range trueExprs {
		t.Run(src, func(t *testing.T) {
			if v := eval(t, src, nil); v != true {
				t.Errorf("Eval(%q) = %v, want true", src, v)
			}
		})
	}
	falseExprs := []string{
		"null == 0",
		"null == false",
		"undefined == 0",
		"undefined == false",
	}
	for _, src := range falseExprs {
		t.Run(src, func(t *testing.T) {
			if v := eval(t, src, nil); v != false {
				t.Errorf("Eval(%q) = %v, want false", src, v)
			}
		})
	}
}

// TestStrictEquality verifies === does not coerce.
func TestStrictEquality(t *testing.T) {
	if v := eval(t, "0 === false", nil); v != false {
		t.Errorf("0 === false: got %v, want false", v)
	}
	if v := eval(t, "null === undefined", nil); v != false {
		t.Errorf("null === undefined: got %v, want false", v)
	}
	if v := eval(t, "1 === 1", nil); v != true {
		t.Errorf("1 === 1: got %v, want true", v)
	}
	if v := eval(t, "null === null", nil); v != true {
		t.Errorf("null === null: got %v, want true", v)
	}
	if v := eval(t, "undefined === undefined", nil); v != true {
		t.Errorf("undefined === undefined: got %v, want true", v)
	}
}

// TestLogicalShortCircuit verifies && / || return the operand value, not a bool.
func TestLogicalShortCircuit(t *testing.T) {
	scope := map[string]any{"a": float64(5), "b": "hello"}

	// a && b → b (a is truthy)
	if v := eval(t, "a && b", scope); v != "hello" {
		t.Errorf("a && b: got %v, want hello", v)
	}
	// false && b → false (short-circuit, b not evaluated)
	if v := eval(t, "false && b", scope); v != false {
		t.Errorf("false && b: got %v, want false", v)
	}
	// a || b → a (a is truthy)
	if v := eval(t, "a || b", scope); v != float64(5) {
		t.Errorf("a || b: got %v, want 5", v)
	}
	// 0 || b → b (0 is falsy)
	if v := eval(t, "0 || b", scope); v != "hello" {
		t.Errorf("0 || b: got %v, want hello", v)
	}
}

// TestNullCoalescing verifies ?? returns right side when left is null or undefined.
func TestNullCoalescing(t *testing.T) {
	scope := map[string]any{"a": nil, "b": float64(42)}

	if v := eval(t, "a ?? 'default'", scope); v != "default" {
		t.Errorf("null ?? 'default': got %v, want default", v)
	}
	// Undefined variable
	if v := eval(t, "missing ?? 'fallback'", scope); v != "fallback" {
		t.Errorf("undefined ?? 'fallback': got %v, want fallback", v)
	}
	// Non-null value: return left
	if v := eval(t, "b ?? 'default'", scope); v != float64(42) {
		t.Errorf("42 ?? 'default': got %v, want 42", v)
	}
	// 0 is not nullish
	scope2 := map[string]any{"x": float64(0)}
	if v := eval(t, "x ?? 'default'", scope2); v != float64(0) {
		t.Errorf("0 ?? 'default': got %v, want 0", v)
	}
}

// TestTernaryEval verifies the ternary operator.
func TestTernaryEval(t *testing.T) {
	if v := eval(t, "true ? 1 : 2", nil); v != float64(1) {
		t.Errorf("got %v, want 1", v)
	}
	if v := eval(t, "false ? 1 : 2", nil); v != float64(2) {
		t.Errorf("got %v, want 2", v)
	}
}

// TestUnaryEval verifies prefix unary operators.
func TestUnaryEval(t *testing.T) {
	if v := eval(t, "!true", nil); v != false {
		t.Errorf("!true: got %v", v)
	}
	if v := eval(t, "!false", nil); v != true {
		t.Errorf("!false: got %v", v)
	}
	if v := eval(t, "!0", nil); v != true {
		t.Errorf("!0: got %v", v)
	}
	if v := eval(t, "-5", nil); v != float64(-5) {
		t.Errorf("-5: got %v", v)
	}
	if v := eval(t, "+3", nil); v != float64(3) {
		t.Errorf("+3: got %v", v)
	}
	if v := eval(t, "~0", nil); v != float64(-1) {
		t.Errorf("~0: got %v", v)
	}
	if v := eval(t, "void 0", nil); v != Undefined {
		t.Errorf("void 0: got %v", v)
	}

	// Arrays and slices are always truthy in JavaScript, so !array is always false.
	if v := eval(t, "![]", nil); v != false {
		t.Errorf("![]: got %v, want false", v)
	}
	if v := eval(t, "![1, 2, 3]", nil); v != false {
		t.Errorf("![1,2,3]: got %v, want false", v)
	}
	if v := eval(t, "!items", map[string]any{"items": []any{"a", "b"}}); v != false {
		t.Errorf("!items ([]any non-empty): got %v, want false", v)
	}
	if v := eval(t, "!items", map[string]any{"items": []any{}}); v != false {
		t.Errorf("!items ([]any empty): got %v, want false", v)
	}
	if v := eval(t, "!items", map[string]any{"items": []string{"a"}}); v != false {
		t.Errorf("!items ([]string non-empty): got %v, want false", v)
	}
	if v := eval(t, "!items", map[string]any{"items": []string{}}); v != false {
		t.Errorf("!items ([]string empty): got %v, want false", v)
	}
	var nilSlice []string
	if v := eval(t, "!items", map[string]any{"items": nilSlice}); v != false {
		t.Errorf("!items ([]string nil): got %v, want false", v)
	}
}

// TestIsTruthy_SliceAndArray directly tests IsTruthy for Go slice and array
// types to guard against regressions in isTruthy's explicit handling of these
// kinds.
func TestIsTruthy_SliceAndArray(t *testing.T) {
	// []any — produced by array literals in the expression evaluator.
	if got := IsTruthy([]any{}); !got {
		t.Error("IsTruthy([]any{}): got false, want true")
	}
	if got := IsTruthy([]any{1, 2, 3}); !got {
		t.Error("IsTruthy([]any{1,2,3}): got false, want true")
	}

	// Typed Go slices from caller scope.
	if got := IsTruthy([]string{"hello"}); !got {
		t.Error("IsTruthy([]string{\"hello\"}): got false, want true")
	}
	if got := IsTruthy([]string{}); !got {
		t.Error("IsTruthy([]string{}): got false, want true")
	}

	// Typed nil slice stored as interface: interface is non-nil, so truthy.
	var nilSlice []string
	if got := IsTruthy(nilSlice); !got {
		t.Error("IsTruthy(nil []string as any): got false, want true")
	}

	// Go arrays (fixed-size).
	if got := IsTruthy([3]int{1, 2, 3}); !got {
		t.Error("IsTruthy([3]int{1,2,3}): got false, want true")
	}
	if got := IsTruthy([0]int{}); !got {
		t.Error("IsTruthy([0]int{}): got false, want true")
	}
}

// TestTypeof verifies the typeof operator returns correct type strings.
func TestTypeof(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"typeof 42", "number"},
		{"typeof 'hello'", "string"},
		{"typeof true", "boolean"},
		{"typeof null", "object"},
		{"typeof undefined", "undefined"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			if v := eval(t, tc.src, nil); v != tc.want {
				t.Errorf("got %q, want %q", v, tc.want)
			}
		})
	}
	// typeof undeclared variable → "undefined" (not an error)
	if v := eval(t, "typeof undeclaredVar", nil); v != "undefined" {
		t.Errorf("typeof undeclaredVar: got %q, want undefined", v)
	}
}

// TestIdentifierLookup verifies identifier resolution from scope.
func TestIdentifierLookup(t *testing.T) {
	scope := map[string]any{"x": float64(99), "name": "Alice"}
	if v := eval(t, "x", scope); v != float64(99) {
		t.Errorf("got %v, want 99", v)
	}
	if v := eval(t, "name", scope); v != "Alice" {
		t.Errorf("got %v, want Alice", v)
	}
	if v := eval(t, "missing", scope); v != Undefined {
		t.Errorf("missing identifier: got %v, want Undefined", v)
	}
}

// TestMapMemberAccess verifies dot and bracket access on nested maps.
func TestMapMemberAccess(t *testing.T) {
	scope := map[string]any{
		"user": map[string]any{
			"name": "Bob",
			"age":  float64(30),
		},
	}
	if v := eval(t, "user.name", scope); v != "Bob" {
		t.Errorf("user.name: got %v, want Bob", v)
	}
	if v := eval(t, "user['age']", scope); v != float64(30) {
		t.Errorf("user['age']: got %v, want 30", v)
	}
}

// TestSliceAccess verifies indexed access on slices.
func TestSliceAccess(t *testing.T) {
	scope := map[string]any{
		"items": []any{"a", "b", "c"},
	}
	if v := eval(t, "items[0]", scope); v != "a" {
		t.Errorf("items[0]: got %v, want a", v)
	}
	if v := eval(t, "items[2]", scope); v != "c" {
		t.Errorf("items[2]: got %v, want c", v)
	}
}

// TestOutOfBounds verifies that out-of-bounds index returns an error.
func TestOutOfBounds(t *testing.T) {
	scope := map[string]any{"items": []any{"x"}}
	evalErr(t, "items[5]", scope)
}

// TestNilMapAccess verifies that nil map member access returns an error.
func TestNilMapAccess(t *testing.T) {
	scope := map[string]any{"m": (map[string]any)(nil)}
	evalErr(t, "m.key", scope)
}

// TestStructFieldByName verifies accessing struct fields by exported name.
func TestStructFieldByName(t *testing.T) {
	type Point struct {
		X float64
		Y float64
	}
	scope := map[string]any{"p": Point{X: 3, Y: 4}}
	if v := eval(t, "p.X", scope); v != float64(3) {
		t.Errorf("p.X: got %v, want 3", v)
	}
	if v := eval(t, "p.Y", scope); v != float64(4) {
		t.Errorf("p.Y: got %v, want 4", v)
	}
}

// TestStructFieldByJSONTag verifies accessing struct fields by json tag.
func TestStructFieldByJSONTag(t *testing.T) {
	type User struct {
		Name  string  `json:"name"`
		Email string  `json:"email"`
		Score float64 `json:"score,omitempty"`
	}
	scope := map[string]any{
		"user": User{Name: "Carol", Email: "carol@example.com", Score: 9.5},
	}
	if v := eval(t, "user.name", scope); v != "Carol" {
		t.Errorf("user.name: got %v, want Carol", v)
	}
	if v := eval(t, "user.email", scope); v != "carol@example.com" {
		t.Errorf("user.email: got %v, want carol@example.com", v)
	}
	if v := eval(t, "user.score", scope); v != float64(9.5) {
		t.Errorf("user.score: got %v, want 9.5", v)
	}
}

// TestFunctionCall verifies calling a Go function stored in scope.
func TestFunctionCall(t *testing.T) {
	scope := map[string]any{
		"add": func(args ...any) (any, error) {
			a := args[0].(float64)
			b := args[1].(float64)
			return a + b, nil
		},
	}
	if v := eval(t, "add(3, 4)", scope); v != float64(7) {
		t.Errorf("add(3, 4): got %v, want 7", v)
	}
}

// TestBitwiseOps verifies bitwise operators.
func TestBitwiseOps(t *testing.T) {
	cases := []struct {
		src  string
		want float64
	}{
		{"5 & 3", 1},
		{"5 | 3", 7},
		{"5 ^ 3", 6},
		{"1 << 3", 8},
		{"8 >> 2", 2},
		{"8 >>> 2", 2},
		{"~0", -1},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			if v := eval(t, tc.src, nil); v != tc.want {
				t.Errorf("got %v, want %v", v, tc.want)
			}
		})
	}
}

// TestInOperator verifies the `in` operator for map key membership.
func TestInOperator(t *testing.T) {
	scope := map[string]any{
		"obj": map[string]any{"x": 1, "y": 2},
	}
	if v := eval(t, "'x' in obj", scope); v != true {
		t.Errorf("'x' in obj: got %v, want true", v)
	}
	if v := eval(t, "'z' in obj", scope); v != false {
		t.Errorf("'z' in obj: got %v, want false", v)
	}
}

// TestArrayLiteral verifies that array literals produce []any.
func TestArrayLiteralEval(t *testing.T) {
	v := eval(t, "[1, 2, 3]", nil)
	arr, ok := v.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", v)
	}
	if len(arr) != 3 || arr[0] != float64(1) || arr[1] != float64(2) || arr[2] != float64(3) {
		t.Errorf("got %v", arr)
	}
}

// TestObjectLiteralEval verifies that object literals produce map[string]any.
func TestObjectLiteralEval(t *testing.T) {
	v := eval(t, "{ a: 1, b: 'two' }", nil)
	obj, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", v)
	}
	if obj["a"] != float64(1) {
		t.Errorf("a: got %v, want 1", obj["a"])
	}
	if obj["b"] != "two" {
		t.Errorf("b: got %v, want two", obj["b"])
	}
}

// TestNewKeyword_ReturnsError verifies that the 'new' keyword is rejected by
// the parser with a descriptive error. 'new' is a JavaScript object-
// instantiation construct that is intentionally unsupported in the template
// expression language.
func TestNewKeyword_ReturnsError(t *testing.T) {
	cases := []string{
		"new Foo()",
		"new Date()",
		"new",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			_, err := Eval(src, nil)
			if err == nil {
				t.Fatalf("Eval(%q): expected error for 'new' keyword, got nil", src)
			}
		})
	}
}

// TestSliceLength verifies the .length property on slices and arrays.
func TestSliceLength(t *testing.T) {
	// Non-empty slice
	scope := map[string]any{"items": []any{"a", "b", "c"}}
	if v := eval(t, "items.length", scope); v != float64(3) {
		t.Errorf("items.length: got %v (%T), want float64(3)", v, v)
	}

	// Empty slice
	scope2 := map[string]any{"items": []any{}}
	if v := eval(t, "items.length", scope2); v != float64(0) {
		t.Errorf("items.length (empty): got %v (%T), want float64(0)", v, v)
	}

	// Typed Go slice
	scope3 := map[string]any{"nums": []int{10, 20}}
	if v := eval(t, "nums.length", scope3); v != float64(2) {
		t.Errorf("nums.length: got %v (%T), want float64(2)", v, v)
	}
}

// TestSliceLengthOnNonSlice verifies that .length on a non-slice returns an error.
func TestSliceLengthOnNonSlice(t *testing.T) {
	// A string is not a slice — member access on strings is not supported.
	scope := map[string]any{"s": "hello"}
	evalErr(t, "s.length", scope)

	// A number is also not a slice.
	scope2 := map[string]any{"n": float64(42)}
	evalErr(t, "n.length", scope2)
}

// TestBuiltinLen verifies that len() is no longer a built-in and that
// .length is the correct alternative.
func TestBuiltinLen(t *testing.T) {
	scope := map[string]any{"items": []any{"x", "y", "z"}}
	// len resolves to undefined, which is not callable — must return an error.
	evalErr(t, "len(items)", scope)
	// .length is the correct way to get slice length.
	if v := eval(t, "items.length", scope); v != float64(3) {
		t.Errorf("items.length: got %v (%T), want float64(3)", v, v)
	}
}

// TestSliceLengthStrictEqual verifies that .length === <number> works correctly
// when the slice is empty or non-empty.
func TestSliceLengthStrictEqual(t *testing.T) {
	// posts.length === 0 must be true for an empty slice.
	empty := map[string]any{"posts": []any{}}
	if v := eval(t, "posts.length === 0", empty); v != true {
		t.Errorf("posts.length === 0 (empty): got %v, want true", v)
	}

	// posts.length === 0 must be false for a non-empty slice.
	two := map[string]any{"posts": []any{"a", "b"}}
	if v := eval(t, "posts.length === 0", two); v != false {
		t.Errorf("posts.length === 0 (non-empty): got %v, want false", v)
	}

	// posts.length === 2 must be true when there are 2 elements.
	if v := eval(t, "posts.length === 2", two); v != true {
		t.Errorf("posts.length === 2: got %v, want true", v)
	}
}

// TestRegisterBuiltin verifies that a custom function registered via
// RegisterBuiltin is callable from expression strings.
func TestRegisterBuiltin(t *testing.T) {
	RegisterBuiltin("double", func(args ...any) (any, error) {
		n := args[0].(float64)
		return n * 2, nil
	})
	if v := eval(t, "double(5)", nil); v != float64(10) {
		t.Errorf("double(5): got %v, want 10", v)
	}
}

// TestNumericTypeMismatchComparison verifies that int vs float64 comparisons
// work correctly in both === and == operators. Users may pass Go int values
// in scope; those must compare equal to float64 literals from the expression.
func TestNumericTypeMismatchComparison(t *testing.T) {
	// int(0) === float64(0) must be true.
	scope := map[string]any{"n": int(0)}
	if v := eval(t, "n === 0", scope); v != true {
		t.Errorf("int(0) === 0: got %v, want true", v)
	}
	// int(2) === float64(2) must be true.
	scope2 := map[string]any{"n": int(2)}
	if v := eval(t, "n === 2", scope2); v != true {
		t.Errorf("int(2) === 2: got %v, want true", v)
	}
	// int(1) == float64(1) must be true.
	if v := eval(t, "n == 2", scope2); v != true {
		t.Errorf("int(2) == 2: got %v, want true", v)
	}
}

// TestNaN verifies NaN semantics.
func TestNaN(t *testing.T) {
	v := eval(t, "0/0", nil)
	f, ok := v.(float64)
	if !ok || !math.IsNaN(f) {
		t.Errorf("0/0: expected NaN, got %v", v)
	}
	// NaN !== NaN
	if v := eval(t, "0/0 === 0/0", nil); v != false {
		t.Errorf("NaN === NaN: got %v, want false", v)
	}
}

// TestEval_EdgeCases covers boundary values and unusual-but-valid expressions
// that are not exercised by the main happy-path tests.
func TestEval_EdgeCases(t *testing.T) {
	// 1/0 in Go float64 arithmetic is +Inf, not a panic or error.
	// This is consistent with IEEE-754 and mirrors JavaScript behaviour.
	t.Run("1/0 is +Inf", func(t *testing.T) {
		v, err := Eval("1 / 0", nil)
		if err != nil {
			t.Fatalf("1/0: unexpected error: %v", err)
		}
		f, ok := v.(float64)
		if !ok || !math.IsInf(f, 1) {
			t.Errorf("1/0: got %v (%T), want +Inf", v, v)
		}
	})

	// 0/0 → NaN (see also TestNaN; duplicated here for table-driven completeness).
	t.Run("0/0 is NaN", func(t *testing.T) {
		v, err := Eval("0 / 0", nil)
		if err != nil {
			t.Fatalf("0/0: unexpected error: %v", err)
		}
		f, ok := v.(float64)
		if !ok || !math.IsNaN(f) {
			t.Errorf("0/0: got %v (%T), want NaN", v, v)
		}
	})

	// null?.foo → Undefined: optional chaining short-circuits when the object
	// is null, returning Undefined instead of an error.
	t.Run("null?.foo is Undefined", func(t *testing.T) {
		v := eval(t, "null?.foo", nil)
		if v != Undefined {
			t.Errorf("null?.foo: got %v, want Undefined", v)
		}
	})

	// undefined?.foo → Undefined: same short-circuit for the undefined value.
	t.Run("undefined?.foo is Undefined", func(t *testing.T) {
		v := eval(t, "undefined?.foo", nil)
		if v != Undefined {
			t.Errorf("undefined?.foo: got %v, want Undefined", v)
		}
	})

	// a?.b?.c when a is nil in scope: a?.b short-circuits to Undefined, then
	// Undefined?.c short-circuits again — no panic, result is Undefined.
	t.Run("a?.b?.c with a=nil is Undefined", func(t *testing.T) {
		v := eval(t, "a?.b?.c", map[string]any{"a": nil})
		if v != Undefined {
			t.Errorf("a?.b?.c (a=nil): got %v, want Undefined", v)
		}
	})

	// a?.b?.c when a.b is nil: a?.b succeeds returning nil, then nil?.c
	// short-circuits to Undefined.
	t.Run("a?.b?.c with a.b=nil is Undefined", func(t *testing.T) {
		v := eval(t, "a?.b?.c", map[string]any{
			"a": map[string]any{"b": nil},
		})
		if v != Undefined {
			t.Errorf("a?.b?.c (a.b=nil): got %v, want Undefined", v)
		}
	})

	// [] + [] — neither side is a string, and toNumber([]any{}) fails.
	// The evaluator returns an error rather than silently coercing arrays.
	// This documents the current (non-JS) behaviour.
	t.Run("[] + [] returns error (arrays not coercible to number)", func(t *testing.T) {
		_, err := Eval("[] + []", nil)
		if err == nil {
			t.Error("[] + []: expected error for array + array, got nil")
		}
	})

	// {} ?? 'default' → {} (empty object is not null/undefined, so ?? returns
	// the left operand).
	t.Run("{} ?? 'default' returns {}", func(t *testing.T) {
		v := eval(t, "{} ?? 'default'", nil)
		obj, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("{} ?? 'default': got %T, want map[string]any", v)
		}
		if len(obj) != 0 {
			t.Errorf("{} ?? 'default': got non-empty map %v", obj)
		}
	})

	// '' ?? 'fallback' → '' (empty string is not null/undefined; ?? only
	// checks for null or undefined, not falsy values).
	t.Run("'' ?? 'fallback' returns ''", func(t *testing.T) {
		v := eval(t, "'' ?? 'fallback'", nil)
		if v != "" {
			t.Errorf("'' ?? 'fallback': got %v, want ''", v)
		}
	})

	// typeof [] → "object": arrays have no special type tag in this evaluator.
	t.Run("typeof [] is 'object'", func(t *testing.T) {
		v := eval(t, "typeof []", nil)
		if v != "object" {
			t.Errorf("typeof []: got %q, want %q", v, "object")
		}
	})

	// typeof {} → "object": plain objects are also "object".
	t.Run("typeof {} is 'object'", func(t *testing.T) {
		v := eval(t, "typeof {}", nil)
		if v != "object" {
			t.Errorf("typeof {}: got %q, want %q", v, "object")
		}
	})

	// typeof null → "object" (JS-compatible quirk).
	t.Run("typeof null is 'object'", func(t *testing.T) {
		v := eval(t, "typeof null", nil)
		if v != "object" {
			t.Errorf("typeof null: got %q, want %q", v, "object")
		}
	})

	// typeof undefined → "undefined".
	t.Run("typeof undefined is 'undefined'", func(t *testing.T) {
		v := eval(t, "typeof undefined", nil)
		if v != "undefined" {
			t.Errorf("typeof undefined: got %q, want %q", v, "undefined")
		}
	})

	// typeof 42 → "number".
	t.Run("typeof 42 is 'number'", func(t *testing.T) {
		v := eval(t, "typeof 42", nil)
		if v != "number" {
			t.Errorf("typeof 42: got %q, want %q", v, "number")
		}
	})

	// typeof 'x' → "string".
	t.Run("typeof 'x' is 'string'", func(t *testing.T) {
		v := eval(t, "typeof 'x'", nil)
		if v != "string" {
			t.Errorf("typeof 'x': got %q, want %q", v, "string")
		}
	})

	// typeof true → "boolean".
	t.Run("typeof true is 'boolean'", func(t *testing.T) {
		v := eval(t, "typeof true", nil)
		if v != "boolean" {
			t.Errorf("typeof true: got %q, want %q", v, "boolean")
		}
	})

	// [1,2,3][10] — out-of-bounds index returns an error (not Undefined).
	// This documents the current behaviour: the evaluator does not silently
	// return undefined for out-of-bounds array access.
	t.Run("[1,2,3][10] returns error (out-of-bounds)", func(t *testing.T) {
		_, err := Eval("[1,2,3][10]", nil)
		if err == nil {
			t.Error("[1,2,3][10]: expected error for out-of-bounds access, got nil")
		}
	})

	// 'abc'[10] — string values are not indexable; member access returns error.
	t.Run("'abc'[10] returns error (strings are not indexable)", func(t *testing.T) {
		_, err := Eval("'abc'[10]", nil)
		if err == nil {
			t.Error("'abc'[10]: expected error for string index access, got nil")
		}
	})

	// Very long identifier chain (50 levels of .a) must resolve to the leaf
	// value without stack overflow.  This guards against overly deep recursion
	// in evalNode / evalMember.
	t.Run("deep member chain does not stack overflow", func(t *testing.T) {
		const depth = 50
		// Build {"a": {"a": {"a": ... "leaf" ...}}} 50 levels deep.
		var inner any = "leaf"
		for i := 0; i < depth; i++ {
			inner = map[string]any{"a": inner}
		}
		scope := map[string]any{"a": inner}
		// Build expression: a.a.a... (depth accesses).
		chain := "a"
		for i := 0; i < depth; i++ {
			chain += ".a"
		}
		v := eval(t, chain, scope)
		if v != "leaf" {
			t.Errorf("deep member chain: got %v, want \"leaf\"", v)
		}
	})
}

// TestEval_TypeCoercion covers arithmetic between mixed types to document the
// JS-compatible coercion rules implemented by addValues and toNumber.
func TestEval_TypeCoercion(t *testing.T) {
	// "3" + 4 → string concatenation because the left operand is a string.
	// JS: "3" + 4 === "34" (number is coerced to string, not the reverse).
	t.Run("string + number concatenates as string", func(t *testing.T) {
		v := eval(t, "'3' + 4", nil)
		if v != "34" {
			t.Errorf("'3' + 4: got %v (%T), want \"34\"", v, v)
		}
	})

	// true + 1 → 2: booleans are coerced to numbers (true → 1) before addition.
	t.Run("true + 1 is 2", func(t *testing.T) {
		v := eval(t, "true + 1", nil)
		if v != float64(2) {
			t.Errorf("true + 1: got %v (%T), want float64(2)", v, v)
		}
	})

	// false + false → 0: both operands are coerced to 0.
	t.Run("false + false is 0", func(t *testing.T) {
		v := eval(t, "false + false", nil)
		if v != float64(0) {
			t.Errorf("false + false: got %v (%T), want float64(0)", v, v)
		}
	})
}

// TestEval_Builtins verifies that functions registered via RegisterBuiltin are
// correctly callable and that calling an unregistered name returns an error
// rather than panicking.
func TestEval_Builtins(t *testing.T) {
	// Register a variadic builtin that is safe to call with zero arguments.
	RegisterBuiltin("safeBuiltin", func(args ...any) (any, error) {
		if len(args) == 0 {
			return "no-args", nil
		}
		return args[0], nil
	})

	// Called with zero arguments must use the default branch, not panic.
	t.Run("safeBuiltin() with zero args", func(t *testing.T) {
		v := eval(t, "safeBuiltin()", nil)
		if v != "no-args" {
			t.Errorf("safeBuiltin(): got %v, want \"no-args\"", v)
		}
	})

	// Called with one argument must return that argument.
	t.Run("safeBuiltin(42) with one arg", func(t *testing.T) {
		v := eval(t, "safeBuiltin(42)", nil)
		if v != float64(42) {
			t.Errorf("safeBuiltin(42): got %v, want float64(42)", v)
		}
	})

	// Calling an unregistered identifier as a function must return an error
	// (Undefined is not callable) rather than panicking.
	t.Run("calling undefined identifier is an error", func(t *testing.T) {
		_, err := Eval("notABuiltin()", nil)
		if err == nil {
			t.Error("notABuiltin(): expected error for non-callable undefined, got nil")
		}
	})
}
