package expr

import (
	"strings"
	"testing"
)

// rootNode extracts the root AST node from a compiled Expr.
func rootNode(t *testing.T, src string) Node {
	t.Helper()
	e, err := Compile(src)
	if err != nil {
		t.Fatalf("Compile(%q) error: %v", src, err)
	}
	return e.(*compiledExpr).node
}

// mustCompileError asserts that Compile returns an error containing substr.
func mustCompileError(t *testing.T, src, substr string) {
	t.Helper()
	_, err := Compile(src)
	if err == nil {
		t.Fatalf("Compile(%q): expected error, got nil", src)
	}
	if !strings.Contains(err.Error(), substr) {
		t.Fatalf("Compile(%q): error %q does not contain %q", src, err.Error(), substr)
	}
}

// TestPrecedenceAddMul verifies that * nests deeper than + in "a + b * c".
func TestPrecedenceAddMul(t *testing.T) {
	root := rootNode(t, "a + b * c")
	add, ok := root.(*BinaryExpr)
	if !ok || add.Op != "+" {
		t.Fatalf("expected root BinaryExpr(+), got %T %v", root, root)
	}
	mul, ok := add.Right.(*BinaryExpr)
	if !ok || mul.Op != "*" {
		t.Fatalf("expected right child BinaryExpr(*), got %T %v", add.Right, add.Right)
	}
	if id, ok := add.Left.(*Identifier); !ok || id.Name != "a" {
		t.Errorf("left of + should be Identifier(a), got %T", add.Left)
	}
	if id, ok := mul.Left.(*Identifier); !ok || id.Name != "b" {
		t.Errorf("left of * should be Identifier(b), got %T", mul.Left)
	}
	if id, ok := mul.Right.(*Identifier); !ok || id.Name != "c" {
		t.Errorf("right of * should be Identifier(c), got %T", mul.Right)
	}
}

// TestRightAssocExponent verifies that ** is right-associative in "a ** b ** c".
func TestRightAssocExponent(t *testing.T) {
	// Expected: a ** (b ** c)
	root := rootNode(t, "a ** b ** c")
	outer, ok := root.(*BinaryExpr)
	if !ok || outer.Op != "**" {
		t.Fatalf("expected root BinaryExpr(**), got %T", root)
	}
	if id, ok := outer.Left.(*Identifier); !ok || id.Name != "a" {
		t.Errorf("left of outer ** should be Identifier(a), got %T", outer.Left)
	}
	inner, ok := outer.Right.(*BinaryExpr)
	if !ok || inner.Op != "**" {
		t.Fatalf("expected right child BinaryExpr(**), got %T", outer.Right)
	}
	if id, ok := inner.Left.(*Identifier); !ok || id.Name != "b" {
		t.Errorf("left of inner ** should be Identifier(b), got %T", inner.Left)
	}
	if id, ok := inner.Right.(*Identifier); !ok || id.Name != "c" {
		t.Errorf("right of inner ** should be Identifier(c), got %T", inner.Right)
	}
}

// TestTernary verifies that "x ? y : z" produces a TernaryExpr.
func TestTernary(t *testing.T) {
	root := rootNode(t, "x ? y : z")
	tern, ok := root.(*TernaryExpr)
	if !ok {
		t.Fatalf("expected TernaryExpr, got %T", root)
	}
	if id, ok := tern.Condition.(*Identifier); !ok || id.Name != "x" {
		t.Errorf("Condition should be Identifier(x), got %T", tern.Condition)
	}
	if id, ok := tern.Consequent.(*Identifier); !ok || id.Name != "y" {
		t.Errorf("Consequent should be Identifier(y), got %T", tern.Consequent)
	}
	if id, ok := tern.Alternate.(*Identifier); !ok || id.Name != "z" {
		t.Errorf("Alternate should be Identifier(z), got %T", tern.Alternate)
	}
}

// TestTernaryRightAssoc verifies right-associativity of nested ternaries.
func TestTernaryRightAssoc(t *testing.T) {
	// a ? b : c ? d : e  →  a ? b : (c ? d : e)
	root := rootNode(t, "a ? b : c ? d : e")
	outer, ok := root.(*TernaryExpr)
	if !ok {
		t.Fatalf("expected outer TernaryExpr, got %T", root)
	}
	if id, ok := outer.Condition.(*Identifier); !ok || id.Name != "a" {
		t.Errorf("outer Condition should be a, got %T", outer.Condition)
	}
	inner, ok := outer.Alternate.(*TernaryExpr)
	if !ok {
		t.Fatalf("Alternate should be TernaryExpr, got %T", outer.Alternate)
	}
	if id, ok := inner.Condition.(*Identifier); !ok || id.Name != "c" {
		t.Errorf("inner Condition should be c, got %T", inner.Condition)
	}
}

// TestChainedMemberAndCall verifies "obj.foo[0](a, b)" produces chained nodes.
func TestChainedMemberAndCall(t *testing.T) {
	root := rootNode(t, "obj.foo[0](a, b)")

	// outermost should be a call
	call, ok := root.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr at root, got %T", root)
	}
	if len(call.Args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(call.Args))
	}
	if id, ok := call.Args[0].(*Identifier); !ok || id.Name != "a" {
		t.Errorf("arg[0] should be Identifier(a), got %T", call.Args[0])
	}
	if id, ok := call.Args[1].(*Identifier); !ok || id.Name != "b" {
		t.Errorf("arg[1] should be Identifier(b), got %T", call.Args[1])
	}

	// callee should be obj.foo[0]
	idx, ok := call.Callee.(*MemberExpr)
	if !ok || !idx.Computed {
		t.Fatalf("callee should be computed MemberExpr, got %T", call.Callee)
	}
	if n, ok := idx.Property.(*NumberLit); !ok || n.Value != 0 {
		t.Errorf("computed property should be NumberLit(0), got %T", idx.Property)
	}

	// idx.Object should be obj.foo
	dot, ok := idx.Object.(*MemberExpr)
	if !ok || dot.Computed {
		t.Fatalf("idx.Object should be dot MemberExpr, got %T", idx.Object)
	}
	if id, ok := dot.Object.(*Identifier); !ok || id.Name != "obj" {
		t.Errorf("dot.Object should be Identifier(obj), got %T", dot.Object)
	}
	if id, ok := dot.Property.(*Identifier); !ok || id.Name != "foo" {
		t.Errorf("dot.Property should be Identifier(foo), got %T", dot.Property)
	}
}

// TestArrayLiteral verifies "[1, 2, 3]" produces an ArrayLit with three elements.
func TestArrayLiteral(t *testing.T) {
	root := rootNode(t, "[1, 2, 3]")
	arr, ok := root.(*ArrayLit)
	if !ok {
		t.Fatalf("expected ArrayLit, got %T", root)
	}
	if len(arr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr.Elements))
	}
	for i, want := range []float64{1, 2, 3} {
		n, ok := arr.Elements[i].(*NumberLit)
		if !ok {
			t.Errorf("element[%d] should be NumberLit, got %T", i, arr.Elements[i])
			continue
		}
		if n.Value != want {
			t.Errorf("element[%d] value: got %v, want %v", i, n.Value, want)
		}
	}
}

// TestObjectLiteral verifies "{ a: 1, 'b': 2 }" produces an ObjectLit.
func TestObjectLiteral(t *testing.T) {
	root := rootNode(t, "{ a: 1, 'b': 2 }")
	obj, ok := root.(*ObjectLit)
	if !ok {
		t.Fatalf("expected ObjectLit, got %T", root)
	}
	if len(obj.Properties) != 2 {
		t.Fatalf("expected 2 properties, got %d", len(obj.Properties))
	}
	if obj.Properties[0].Key != "a" {
		t.Errorf("prop[0].Key: got %q, want %q", obj.Properties[0].Key, "a")
	}
	if n, ok := obj.Properties[0].Value.(*NumberLit); !ok || n.Value != 1 {
		t.Errorf("prop[0].Value: expected NumberLit(1), got %T", obj.Properties[0].Value)
	}
	if obj.Properties[1].Key != "b" {
		t.Errorf("prop[1].Key: got %q, want %q", obj.Properties[1].Key, "b")
	}
	if n, ok := obj.Properties[1].Value.(*NumberLit); !ok || n.Value != 2 {
		t.Errorf("prop[1].Value: expected NumberLit(2), got %T", obj.Properties[1].Value)
	}
}

// TestMalformedExpressions verifies that invalid input returns descriptive errors.
func TestMalformedExpressions(t *testing.T) {
	cases := []struct {
		src    string
		errSub string
	}{
		{"", "unexpected end"},
		{"(", "unexpected end"},
		{"(a", "expected )"},
		{"[1, 2", "expected ]"},
		{"{ a", "expected :"},
		{"{ a: }", "unexpected token }"},
		{"a +", "unexpected end"},
		{"a ? b", "expected :"},
		{"obj.", "expected Ident"},
		{"1 2", "unexpected token"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			mustCompileError(t, tc.src, tc.errSub)
		})
	}
}

// TestLiterals verifies parsing of all literal types.
func TestLiterals(t *testing.T) {
	t.Run("number_int", func(t *testing.T) {
		n := rootNode(t, "42").(*NumberLit)
		if n.Value != 42 {
			t.Errorf("got %v, want 42", n.Value)
		}
	})
	t.Run("number_float", func(t *testing.T) {
		n := rootNode(t, "3.14").(*NumberLit)
		if n.Value != 3.14 {
			t.Errorf("got %v, want 3.14", n.Value)
		}
	})
	t.Run("string", func(t *testing.T) {
		s := rootNode(t, "'hello'").(*StringLit)
		if s.Value != "hello" {
			t.Errorf("got %q, want %q", s.Value, "hello")
		}
	})
	t.Run("true", func(t *testing.T) {
		b := rootNode(t, "true").(*BoolLit)
		if !b.Value {
			t.Error("expected true")
		}
	})
	t.Run("false", func(t *testing.T) {
		b := rootNode(t, "false").(*BoolLit)
		if b.Value {
			t.Error("expected false")
		}
	})
	t.Run("null", func(t *testing.T) {
		if _, ok := rootNode(t, "null").(*NullLit); !ok {
			t.Error("expected NullLit")
		}
	})
	t.Run("undefined", func(t *testing.T) {
		if _, ok := rootNode(t, "undefined").(*UndefinedLit); !ok {
			t.Error("expected UndefinedLit")
		}
	})
}

// TestUnaryExpressions verifies prefix unary operators.
func TestUnaryExpressions(t *testing.T) {
	cases := []struct{ src, op string }{
		{"!x", "!"},
		{"-x", "-"},
		{"+x", "+"},
		{"~x", "~"},
		{"typeof x", "typeof"},
		{"void x", "void"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			u, ok := rootNode(t, tc.src).(*UnaryExpr)
			if !ok {
				t.Fatalf("expected UnaryExpr, got %T", rootNode(t, tc.src))
			}
			if u.Op != tc.op {
				t.Errorf("Op: got %q, want %q", u.Op, tc.op)
			}
		})
	}
}

// TestAllBinaryOps verifies that every binary operator is parsed.
func TestAllBinaryOps(t *testing.T) {
	cases := []string{
		"a ** b", "a * b", "a / b", "a % b",
		"a + b", "a - b",
		"a << b", "a >> b", "a >>> b",
		"a < b", "a <= b", "a > b", "a >= b", "a in b", "a instanceof b",
		"a == b", "a != b", "a === b", "a !== b",
		"a & b", "a ^ b", "a | b",
		"a && b", "a || b", "a ?? b",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			node := rootNode(t, src)
			if _, ok := node.(*BinaryExpr); !ok {
				t.Errorf("expected BinaryExpr, got %T", node)
			}
		})
	}
}

// TestGrouping verifies that parentheses override operator precedence.
func TestGrouping(t *testing.T) {
	// (a + b) * c — multiplication should be at the root
	root := rootNode(t, "(a + b) * c")
	mul, ok := root.(*BinaryExpr)
	if !ok || mul.Op != "*" {
		t.Fatalf("expected root BinaryExpr(*), got %T", root)
	}
	add, ok := mul.Left.(*BinaryExpr)
	if !ok || add.Op != "+" {
		t.Fatalf("expected left child BinaryExpr(+), got %T", mul.Left)
	}
	_ = add
}

// TestEmptyArray verifies that "[]" produces an empty ArrayLit.
func TestEmptyArray(t *testing.T) {
	root := rootNode(t, "[]")
	arr, ok := root.(*ArrayLit)
	if !ok {
		t.Fatalf("expected ArrayLit, got %T", root)
	}
	if len(arr.Elements) != 0 {
		t.Errorf("expected 0 elements, got %d", len(arr.Elements))
	}
}

// TestEmptyObject verifies that "{}" produces an empty ObjectLit.
func TestEmptyObject(t *testing.T) {
	root := rootNode(t, "{}")
	obj, ok := root.(*ObjectLit)
	if !ok {
		t.Fatalf("expected ObjectLit, got %T", root)
	}
	if len(obj.Properties) != 0 {
		t.Errorf("expected 0 properties, got %d", len(obj.Properties))
	}
}

// TestCallNoArgs verifies parsing of a zero-argument call.
func TestCallNoArgs(t *testing.T) {
	root := rootNode(t, "fn()")
	call, ok := root.(*CallExpr)
	if !ok {
		t.Fatalf("expected CallExpr, got %T", root)
	}
	if len(call.Args) != 0 {
		t.Errorf("expected 0 args, got %d", len(call.Args))
	}
}

// TestLeftAssocPlus verifies that "a + b + c" is left-associative.
func TestLeftAssocPlus(t *testing.T) {
	// a + b + c → (a + b) + c
	root := rootNode(t, "a + b + c")
	outer, ok := root.(*BinaryExpr)
	if !ok || outer.Op != "+" {
		t.Fatalf("expected root BinaryExpr(+), got %T", root)
	}
	inner, ok := outer.Left.(*BinaryExpr)
	if !ok || inner.Op != "+" {
		t.Fatalf("expected left child BinaryExpr(+), got %T", outer.Left)
	}
	if id, ok := inner.Left.(*Identifier); !ok || id.Name != "a" {
		t.Errorf("innermost left should be Identifier(a)")
	}
}
