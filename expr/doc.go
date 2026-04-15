/*
Package expr is the expression evaluator for htmlc templates.

# Tutorial

The three most common entry points for Go callers are Eval, Compile, and
CollectIdentifiers.

## Eval — one-shot expression evaluation

Eval parses and evaluates an expression in a single call. Use it when the
expression string changes at runtime or is only evaluated once.

	result, err := expr.Eval("price * qty + 1", map[string]any{
	    "price": float64(10),
	    "qty":   float64(3),
	})
	// result == float64(31)

## Compile + Eval — pre-compile for repeated evaluation

Compile parses the expression once and returns an *Expr that can be re-evaluated
against many different scopes without re-parsing.

	e, err := expr.Compile("user.name + ' (' + user.role + ')'")
	if err != nil { // syntax error
	}

	// Re-use the compiled expression across many scopes:
	for _, user := range users {
	    val, err := e.Eval(map[string]any{"user": user})
	    _ = val
	}

## CollectIdentifiers — static analysis

CollectIdentifiers returns the sorted, deduplicated list of top-level
identifiers referenced by an expression. It does not evaluate the expression.

	names, err := expr.CollectIdentifiers("user.name + extra")
	// names == []string{"extra", "user"}

Used by Component.Props() (in the parent htmlc package) to discover which
scope keys a template depends on.

## RegisterBuiltin — custom functions

RegisterBuiltin adds a function to the global built-in table so it is
available by name in every expression. Call it once at program startup.

	expr.RegisterBuiltin("upper", func(args ...any) (any, error) {
	    if len(args) != 1 {
	        return nil, fmt.Errorf("upper: want 1 arg")
	    }
	    s, _ := args[0].(string)
	    return strings.ToUpper(s), nil
	})
	// Now usable in any expression: upper(name)

Note: RegisterBuiltin modifies global state; it is not safe to call
concurrently with expression evaluation.

## IsTruthy — truthiness outside templates

IsTruthy reports whether a Go value is truthy by the same rules used for
v-if, v-show, and boolean operators inside templates.

	expr.IsTruthy(0)              // false
	expr.IsTruthy("")             // false
	expr.IsTruthy(expr.Undefined) // false
	expr.IsTruthy(false)          // false
	expr.IsTruthy(1)              // true
	expr.IsTruthy("hello")        // true

# Expression Language Reference

This document describes the syntax and semantics of the expr expression
language. It is a declarative, side-effect-free subset of JavaScript
expression syntax evaluated against a Go scope map.

# Literals

	42          integer (stored as float64)
	3.14        float
	.5          leading-dot float
	1e6         scientific notation (float)
	"hello"     double-quoted string
	'hello'     single-quoted string
	true        boolean true
	false       boolean false
	null        null value (Go nil)
	undefined   undefined value (Go UndefinedValue{})

Supported string escape sequences: \n  \t  \r  \\  \'  \".
All numeric literals are represented internally as float64.

# Identifiers

An identifier is a sequence of letters, digits, underscores, or dollar signs
that does not begin with a digit. Names are resolved in the following order:

 1. The caller-supplied scope map (map[string]any).
 2. The built-in function table (registered via RegisterBuiltin).
 3. If absent from both, the name evaluates to UndefinedValue.

Scope map keys shadow built-in names.

# Operator Precedence

The table lists all operators from highest to lowest precedence. Operators on
the same row have equal precedence and are left-associative unless noted.

	Precedence  Operator(s)         Associativity  Category
	──────────  ──────────────────  ─────────────  ─────────────────────────────
	15          (unary) ! - + ~ typeof void        right (unary prefix)
	12          **                  right          exponentiation
	11          *  /  %             left           multiplicative
	10          +  -                left           additive
	 9          <<  >>  >>>         left           bitwise shift
	 8          <  <=  >  >=  in  instanceof       left           relational
	 7          ==  !=  ===  !==    left           equality
	 6          &                   left           bitwise AND
	 5          ^                   left           bitwise XOR
	 4          |                   left           bitwise OR
	 3          &&                  left           logical AND
	 2          ||                  left           logical OR
	 1          ??                  left           nullish coalescing
	 0          ?:                  right          ternary conditional

Grouping with parentheses overrides precedence.

## Unary Operators

	!x      logical NOT; returns bool
	-x      numeric negation; converts x to float64
	+x      numeric identity; converts x to float64
	~x      bitwise NOT; converts x to int32, returns float64
	typeof x  returns a JS-compatible type string (see Type Coercion)
	void x    evaluates x, returns UndefinedValue

## Binary Operators

	**        exponentiation (right-associative)
	*  /  %   multiplication, division, modulo (float64)
	+         addition or string concatenation (see Type Coercion)
	-         subtraction
	<<  >>    signed bitwise shift; operands converted to int32, result is float64
	>>>       unsigned right shift; operands converted to uint32, result is float64
	<  <=  >  >=   relational; string–string uses lexicographic order, otherwise float64
	in        key membership: "k" in obj (map or struct); returns bool
	instanceof  always returns false (Go objects have no prototype chain)
	==  !=    abstract (loose) equality/inequality (see Type Coercion)
	===  !==  strict equality/inequality (no type coercion)
	&  ^  |   bitwise AND, XOR, OR; operands converted to int32, result is float64
	&&        logical AND; short-circuits; returns the deciding operand value
	||        logical OR; short-circuits; returns the deciding operand value
	??        nullish coalescing; returns right operand when left is null or undefined

## Ternary Operator

	condition ? consequent : alternate

Evaluates condition; if truthy evaluates and returns consequent, otherwise
evaluates and returns alternate. Right-associative.
a ? b : c ? d : e  is parsed as  a ? b : (c ? d : e).

# Member Access

Dot notation accesses a named field or map key:

	obj.field
	obj.nested.field

Bracket notation accesses a computed key or numeric index:

	obj["key"]
	arr[0]
	arr[i]

For slices and arrays the special property "length" returns the element count
as float64. Struct fields are matched by exported name first, then by the
"json" struct tag (first tag segment). If the key is absent from a map or struct,
the result is UndefinedValue.

Accessing a member of null or UndefinedValue is a runtime error.

## Go Method Bindings

Exported Go methods on scope values are callable directly from expressions.

Rules:

  - Zero-argument methods are invoked implicitly via dot access (no parentheses
    needed). Parentheses are also accepted.
  - Methods with parameters must be called with explicit arguments.
  - Field access takes priority over a method with the same name (field-first).
  - Lowercase aliases work: post.summary resolves to func (Post) Summary(),
    mirroring the existing field-alias rule.
  - Pointer receivers are supported when the scope value is a pointer to the type.
  - Methods that return (value, error) are supported; a non-nil error surfaces as
    a template evaluation error.
  - Variadic methods are supported.
  - Optional chaining works with methods: post?.Summary on a nil pointer returns
    UndefinedValue.

	// Given these Go types in scope:
	//   type Post struct { Title string }
	//   func (p Post) Summary() string { ... }
	//   func (r *Router) LinkFor(route string) string { ... }

	// Zero-arg method — implicit call:
	//   post.Summary         → result of p.Summary()
	//   post.Summary()       → same, explicit form also works
	//   post.summary         → lowercase alias, same result

	// Method with arguments:
	//   router.LinkFor("home")   → result of r.LinkFor("home")

	// Field wins over method of the same name:
	//   post.Title           → struct field (not a method)

	// Error-returning method:
	//   formatter.FormatCurrency(9.99)  → string result, or error if method errors

	// Variadic method:
	//   v.Join(",", "a", "b", "c")  → "a,b,c"

	// Optional chaining:
	//   post?.Summary        → UndefinedValue if post is nil

# Function Calls

	callee(arg1, arg2)

The callee must evaluate to a Go value of type func(...any) (any, error), or be
a bound Go method on a scope value (see Go Method Bindings above).
Arguments are evaluated left-to-right before the function is called. Scope
values of the correct function type can be called directly.

# Built-in Functions

The engine ships with no pre-registered built-in functions. Callers add custom
functions via RegisterBuiltin:

	expr.RegisterBuiltin("upper", func(args ...any) (any, error) {
	    if len(args) != 1 {
	        return nil, fmt.Errorf("upper: want 1 arg")
	    }
	    s, _ := args[0].(string)
	    return strings.ToUpper(s), nil
	})

Functions registered this way are available in all expressions by name.
Scope map keys shadow built-in names.

For the common case of measuring collection sizes, use the built-in .length
member property instead of a function call. It is available on strings, slices,
arrays, and maps via member-access syntax and requires no registration:

	items.length     // number of elements in a slice or array
	name.length      // number of bytes in a string
	obj.length       // number of entries in a map

# Type Coercion

All numeric values are stored as float64. Go integer types (int, int8, int16,
int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64) are
normalised to float64 for all comparison and arithmetic operations.

The + operator performs string concatenation when either operand is a string;
otherwise both operands are converted to float64 and added numerically.

Relational operators (<, <=, >, >=) use lexicographic string comparison when
both operands are strings; otherwise both are converted to float64.

Abstract equality (==) follows JavaScript abstract equality rules:
  - null == undefined and undefined == null are true.
  - null or undefined compared to any other type is false.
  - bool operands are converted to float64 (true→1, false→0) before comparison.
  - A numeric operand compared to a string converts the string to float64.
  - All Go numeric types are treated as equivalent to float64 for ==.

Strict equality (===) does not coerce types; it returns true only when both
operands are the same type (or both numeric Go types) and equal in value.
NaN !== NaN.

Truthiness (used by !, &&, ||, ternary) follows JavaScript rules:
  - false, 0, NaN, "", null, and UndefinedValue are falsy.
  - All other values are truthy.

typeof returns:
  - "undefined"  for UndefinedValue
  - "boolean"    for bool
  - "number"     for float64
  - "string"     for string
  - "function"   for func(...any)(any,error) and for bound Go methods
  - "object"     for nil (mirrors typeof null in JavaScript)
  - "object"     for all other types

# UndefinedValue

UndefinedValue is a Go struct type (type UndefinedValue struct{}) used as a
sentinel for absent values. It is distinct from Go nil and from the null
literal:

	null       → Go nil         (NullLit in AST, nil in eval)
	undefined  → UndefinedValue (UndefinedLit in AST, Undefined in eval)

An identifier that is absent from both the scope map and the built-ins
evaluates to UndefinedValue, not to nil. This distinction is observable via ===
and typeof.

# Array and Object Literals

	[1, 'two', item]                array literal; evaluates to []any
	{key: val, "x": 1}              object literal; evaluates to map[string]any

Object keys must be identifiers or string literals.

# Unsupported Constructs

The following constructs look like JavaScript but are not supported. The parser
or evaluator will return an error rather than silently ignoring them.

	x = y               assignment (any form)
	x += y              compound assignment
	x++  x--            increment/decrement
	`template ${x}`     template literals (backtick strings)
	...spread           spread operator
	(x) => x            arrow functions
	function f() {}     function declarations
	new Foo()           object instantiation (new keyword)
	delete obj.key      property deletion
	class Foo {}        class declarations
*/
package expr
