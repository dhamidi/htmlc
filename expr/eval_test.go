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
