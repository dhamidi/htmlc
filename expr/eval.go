// Package expr provides the evaluator that executes htmlc template expressions.
package expr

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// UndefinedValue is the type of the undefined sentinel.
type UndefinedValue struct{}

// Undefined is the Go representation of the JavaScript `undefined` value.
var Undefined = UndefinedValue{}

// builtins contains built-in functions always available in expression scope.
var builtins = map[string]any{}

// RegisterBuiltin registers a custom function under name so that it is
// callable in all subsequent Eval calls without being passed in the scope.
func RegisterBuiltin(name string, fn func(...any) (any, error)) {
	builtins[name] = fn
}

// Eval is a convenience wrapper: it compiles src and evaluates it against scope.
func Eval(src string, scope map[string]any) (any, error) {
	e, err := Compile(src)
	if err != nil {
		return nil, err
	}
	return e.Eval(scope)
}

// evalNode dispatches evaluation to the correct handler based on node type.
func evalNode(node Node, scope map[string]any) (any, error) {
	switch n := node.(type) {
	case *NumberLit:
		return n.Value, nil
	case *StringLit:
		return n.Value, nil
	case *BoolLit:
		return n.Value, nil
	case *NullLit:
		return nil, nil
	case *UndefinedLit:
		return Undefined, nil
	case *Identifier:
		if scope != nil {
			if v, ok := scope[n.Name]; ok {
				return v, nil
			}
		}
		if v, ok := builtins[n.Name]; ok {
			return v, nil
		}
		return Undefined, nil
	case *UnaryExpr:
		return evalUnary(n, scope)
	case *BinaryExpr:
		return evalBinary(n, scope)
	case *TernaryExpr:
		return evalTernary(n, scope)
	case *MemberExpr:
		return evalMember(n, scope)
	case *OptionalMemberExpr:
		return evalOptionalMember(n, scope)
	case *CallExpr:
		return evalCall(n, scope)
	case *ArrayLit:
		return evalArrayLit(n, scope)
	case *ObjectLit:
		return evalObjectLit(n, scope)
	default:
		return nil, fmt.Errorf("unknown node type %T", node)
	}
}

// --- Unary ---

// evalUnary evaluates a unary expression.
//
// Note on intentionally unsupported JavaScript-specific constructs:
//   - 'new'          — object instantiation; not supported (parser rejects it)
//   - 'delete'       — property deletion; not supported (parser rejects it)
//   - arrow functions (=>) — not supported (parser rejects them)
//   - 'Promise'      — async/await primitives; not supported
//
// 'typeof' and 'void' are present below in intentionally simplified forms:
//   - 'typeof' returns a JS-compatible type string but does not expose the full
//     JS type system (e.g. no Symbol, BigInt).
//   - 'void' evaluates its operand for side-effects then returns Undefined;
//     since template expressions are side-effect-free this is rarely useful.
//
// 'instanceof' is present as a binary operator but always returns false because
// Go objects do not share JavaScript's prototype chain semantics.
func evalUnary(n *UnaryExpr, scope map[string]any) (any, error) {
	// typeof and void are special: evaluate but don't propagate errors for typeof
	switch n.Op {
	case "typeof":
		val, err := evalNode(n.Operand, scope)
		if err != nil {
			return nil, err
		}
		return typeofValue(val), nil
	case "void":
		_, err := evalNode(n.Operand, scope)
		if err != nil {
			return nil, err
		}
		return Undefined, nil
	}

	val, err := evalNode(n.Operand, scope)
	if err != nil {
		return nil, err
	}
	switch n.Op {
	case "!":
		return !isTruthy(val), nil
	case "-":
		f, err := toNumber(val)
		if err != nil {
			return nil, fmt.Errorf("unary -: %w", err)
		}
		return -f, nil
	case "+":
		f, err := toNumber(val)
		if err != nil {
			return nil, fmt.Errorf("unary +: %w", err)
		}
		return f, nil
	case "~":
		i, err := toInt32(val)
		if err != nil {
			return nil, fmt.Errorf("unary ~: %w", err)
		}
		return float64(^i), nil
	}
	return nil, fmt.Errorf("unknown unary operator %q", n.Op)
}

// --- Binary ---

func evalBinary(n *BinaryExpr, scope map[string]any) (any, error) {
	// Short-circuit operators must not evaluate the right side eagerly.
	switch n.Op {
	case "&&":
		left, err := evalNode(n.Left, scope)
		if err != nil {
			return nil, err
		}
		if !isTruthy(left) {
			return left, nil
		}
		return evalNode(n.Right, scope)
	case "||":
		left, err := evalNode(n.Left, scope)
		if err != nil {
			return nil, err
		}
		if isTruthy(left) {
			return left, nil
		}
		return evalNode(n.Right, scope)
	case "??":
		left, err := evalNode(n.Left, scope)
		if err != nil {
			return nil, err
		}
		if !isNullish(left) {
			return left, nil
		}
		return evalNode(n.Right, scope)
	}

	left, err := evalNode(n.Left, scope)
	if err != nil {
		return nil, err
	}
	right, err := evalNode(n.Right, scope)
	if err != nil {
		return nil, err
	}

	switch n.Op {
	case "+":
		return addValues(left, right)
	case "-", "*", "/", "%", "**":
		return numericBinOp(n.Op, left, right)
	case "<", "<=", ">", ">=":
		return compareOp(n.Op, left, right)
	case "==":
		return looseEqual(left, right), nil
	case "!=":
		return !looseEqual(left, right), nil
	case "===":
		return strictEqual(left, right), nil
	case "!==":
		return !strictEqual(left, right), nil
	case "&":
		a, err := toInt32(left)
		if err != nil {
			return nil, err
		}
		b, err := toInt32(right)
		if err != nil {
			return nil, err
		}
		return float64(a & b), nil
	case "^":
		a, err := toInt32(left)
		if err != nil {
			return nil, err
		}
		b, err := toInt32(right)
		if err != nil {
			return nil, err
		}
		return float64(a ^ b), nil
	case "|":
		a, err := toInt32(left)
		if err != nil {
			return nil, err
		}
		b, err := toInt32(right)
		if err != nil {
			return nil, err
		}
		return float64(a | b), nil
	case "<<":
		a, err := toInt32(left)
		if err != nil {
			return nil, err
		}
		b, err := toUint32(right)
		if err != nil {
			return nil, err
		}
		return float64(a << (b & 31)), nil
	case ">>":
		a, err := toInt32(left)
		if err != nil {
			return nil, err
		}
		b, err := toUint32(right)
		if err != nil {
			return nil, err
		}
		return float64(a >> (b & 31)), nil
	case ">>>":
		a, err := toUint32(left)
		if err != nil {
			return nil, err
		}
		b, err := toUint32(right)
		if err != nil {
			return nil, err
		}
		return float64(a >> (b & 31)), nil
	case "in":
		return inOperator(left, right)
	case "instanceof":
		return false, nil
	default:
		return nil, fmt.Errorf("unknown binary operator %q", n.Op)
	}
}

// --- Ternary ---

func evalTernary(n *TernaryExpr, scope map[string]any) (any, error) {
	cond, err := evalNode(n.Condition, scope)
	if err != nil {
		return nil, err
	}
	if isTruthy(cond) {
		return evalNode(n.Consequent, scope)
	}
	return evalNode(n.Alternate, scope)
}

// --- Member access ---

func evalMember(n *MemberExpr, scope map[string]any) (any, error) {
	obj, err := evalNode(n.Object, scope)
	if err != nil {
		return nil, err
	}
	var key any
	if n.Computed {
		key, err = evalNode(n.Property, scope)
		if err != nil {
			return nil, err
		}
	} else {
		key = n.Property.(*Identifier).Name
	}
	return accessMember(obj, key)
}

// evalOptionalMember handles optional chaining (obj?.prop, obj?.[expr]).
// If obj is null or undefined, the expression evaluates to Undefined rather
// than returning an error.
func evalOptionalMember(n *OptionalMemberExpr, scope map[string]any) (any, error) {
	obj, err := evalNode(n.Object, scope)
	if err != nil {
		return nil, err
	}
	if isNullish(obj) {
		return Undefined, nil
	}
	var key any
	if n.Computed {
		key, err = evalNode(n.Property, scope)
		if err != nil {
			return nil, err
		}
	} else {
		key = n.Property.(*Identifier).Name
	}
	return accessMember(obj, key)
}

func accessMember(obj, key any) (any, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot access property %q of null", fmt.Sprint(key))
	}
	if _, ok := obj.(UndefinedValue); ok {
		return nil, fmt.Errorf("cannot access property %q of undefined", fmt.Sprint(key))
	}

	rv := reflect.ValueOf(obj)
	// Unwrap pointers
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil, fmt.Errorf("cannot access property %q of nil pointer", fmt.Sprint(key))
		}
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Map:
		if rv.IsNil() {
			return nil, fmt.Errorf("cannot access property %q of nil map", fmt.Sprint(key))
		}
		keyVal := reflect.ValueOf(key)
		mapKeyType := rv.Type().Key()
		if keyVal.Type() != mapKeyType {
			if keyVal.Type().ConvertibleTo(mapKeyType) {
				keyVal = keyVal.Convert(mapKeyType)
			} else {
				return Undefined, nil
			}
		}
		result := rv.MapIndex(keyVal)
		if !result.IsValid() {
			return Undefined, nil
		}
		return result.Interface(), nil

	case reflect.Struct:
		keyStr, ok := key.(string)
		if !ok {
			return nil, fmt.Errorf("struct field access requires string key, got %T", key)
		}
		return accessStructField(rv, keyStr)

	case reflect.Slice, reflect.Array:
		// Special property: "length" maps to Go's len().
		if keyStr, ok := key.(string); ok {
			if keyStr == "length" {
				return float64(rv.Len()), nil
			}
			return nil, fmt.Errorf("invalid index: non-integer string index %q", keyStr)
		}
		idx, err := toIndex(key)
		if err != nil {
			return nil, fmt.Errorf("invalid index: %w", err)
		}
		if idx < 0 || idx >= rv.Len() {
			return nil, fmt.Errorf("index %d out of bounds (length %d)", idx, rv.Len())
		}
		return rv.Index(idx).Interface(), nil

	default:
		return nil, fmt.Errorf("cannot access member of %T", obj)
	}
}

func accessStructField(rv reflect.Value, name string) (any, error) {
	rt := rv.Type()
	for i := 0; i < rt.NumField(); i++ {
		f := rt.Field(i)
		if !f.IsExported() {
			continue
		}
		if f.Name == name {
			return rv.Field(i).Interface(), nil
		}
		tag := f.Tag.Get("json")
		if tag != "" {
			tagName := strings.Split(tag, ",")[0]
			if tagName != "-" && tagName == name {
				return rv.Field(i).Interface(), nil
			}
		}
	}
	return Undefined, nil
}

// --- Call ---

func evalCall(n *CallExpr, scope map[string]any) (any, error) {
	callee, err := evalNode(n.Callee, scope)
	if err != nil {
		return nil, err
	}
	fn, ok := callee.(func(...any) (any, error))
	if !ok {
		return nil, fmt.Errorf("value of type %T is not callable", callee)
	}
	args := make([]any, len(n.Args))
	for i, arg := range n.Args {
		args[i], err = evalNode(arg, scope)
		if err != nil {
			return nil, err
		}
	}
	return fn(args...)
}

// --- Array / Object literals ---

func evalArrayLit(n *ArrayLit, scope map[string]any) (any, error) {
	result := make([]any, len(n.Elements))
	for i, elem := range n.Elements {
		val, err := evalNode(elem, scope)
		if err != nil {
			return nil, err
		}
		result[i] = val
	}
	return result, nil
}

func evalObjectLit(n *ObjectLit, scope map[string]any) (any, error) {
	result := make(map[string]any, len(n.Properties))
	for _, prop := range n.Properties {
		val, err := evalNode(prop.Value, scope)
		if err != nil {
			return nil, err
		}
		result[prop.Key] = val
	}
	return result, nil
}

// --- Helpers ---

// IsTruthy returns the JS-style truthiness of a value.
func IsTruthy(v any) bool { return isTruthy(v) }

// isTruthy returns the JS-style truthiness of a value.
func isTruthy(v any) bool {
	if v == nil {
		return false
	}
	if _, ok := v.(UndefinedValue); ok {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case float64:
		return val != 0 && !math.IsNaN(val)
	case string:
		return val != ""
	default:
		return true
	}
}

// isNullish returns true if v is null (nil) or undefined.
func isNullish(v any) bool {
	if v == nil {
		return true
	}
	_, ok := v.(UndefinedValue)
	return ok
}

// typeofValue returns the JS typeof string for a value.
func typeofValue(v any) string {
	if v == nil {
		return "object" // typeof null === "object" in JS
	}
	if _, ok := v.(UndefinedValue); ok {
		return "undefined"
	}
	switch v.(type) {
	case bool:
		return "boolean"
	case float64:
		return "number"
	case string:
		return "string"
	case func(...any) (any, error):
		return "function"
	default:
		return "object"
	}
}

// toNumber converts a value to float64 following JS coercion rules.
func toNumber(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	case string:
		s := strings.TrimSpace(val)
		if s == "" {
			return 0, nil
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return math.NaN(), nil
		}
		return f, nil
	case nil: // null → 0
		return 0, nil
	default:
		if _, ok := v.(UndefinedValue); ok {
			return math.NaN(), nil
		}
		return 0, fmt.Errorf("cannot convert %T to number", v)
	}
}

// toInt32 converts a value to int32 for bitwise operations.
func toInt32(v any) (int32, error) {
	f, err := toNumber(v)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) || f == 0 {
		return 0, nil
	}
	return int32(uint32(int64(f))), nil
}

// toUint32 converts a value to uint32 for shift and >>> operations.
func toUint32(v any) (uint32, error) {
	f, err := toNumber(v)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(f) || math.IsInf(f, 0) || f == 0 {
		return 0, nil
	}
	return uint32(int64(f)), nil
}

// toIndex converts a value to a non-negative integer index.
func toIndex(v any) (int, error) {
	switch val := v.(type) {
	case float64:
		i := int(val)
		if float64(i) != val {
			return 0, fmt.Errorf("non-integer index %v", val)
		}
		return i, nil
	case int:
		return val, nil
	case int64:
		return int(val), nil
	case string:
		i, err := strconv.Atoi(val)
		if err != nil {
			return 0, fmt.Errorf("non-integer string index %q", val)
		}
		return i, nil
	default:
		return 0, fmt.Errorf("cannot use %T as index", v)
	}
}

// toString converts a value to its string representation.
func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if math.IsNaN(val) {
			return "NaN"
		}
		if math.IsInf(val, 1) {
			return "Infinity"
		}
		if math.IsInf(val, -1) {
			return "-Infinity"
		}
		if val == math.Trunc(val) && val >= -1e15 && val <= 1e15 {
			return strconv.FormatInt(int64(val), 10)
		}
		return strconv.FormatFloat(val, 'f', -1, 64)
	case bool:
		if val {
			return "true"
		}
		return "false"
	case nil:
		return "null"
	default:
		if _, ok := v.(UndefinedValue); ok {
			return "undefined"
		}
		return fmt.Sprintf("%v", v)
	}
}

// addValues implements the + operator with string coercion.
func addValues(left, right any) (any, error) {
	_, lIsStr := left.(string)
	_, rIsStr := right.(string)
	if lIsStr || rIsStr {
		return toString(left) + toString(right), nil
	}
	lf, err := toNumber(left)
	if err != nil {
		return nil, fmt.Errorf("+ left: %w", err)
	}
	rf, err := toNumber(right)
	if err != nil {
		return nil, fmt.Errorf("+ right: %w", err)
	}
	return lf + rf, nil
}

// numericBinOp performs a numeric binary operation.
func numericBinOp(op string, left, right any) (any, error) {
	lf, err := toNumber(left)
	if err != nil {
		return nil, fmt.Errorf("%s left: %w", op, err)
	}
	rf, err := toNumber(right)
	if err != nil {
		return nil, fmt.Errorf("%s right: %w", op, err)
	}
	switch op {
	case "-":
		return lf - rf, nil
	case "*":
		return lf * rf, nil
	case "/":
		return lf / rf, nil
	case "%":
		return math.Mod(lf, rf), nil
	case "**":
		return math.Pow(lf, rf), nil
	}
	return nil, fmt.Errorf("unknown numeric op %q", op)
}

// compareOp performs a relational comparison (<, <=, >, >=).
func compareOp(op string, left, right any) (any, error) {
	ls, lIsStr := left.(string)
	rs, rIsStr := right.(string)
	if lIsStr && rIsStr {
		switch op {
		case "<":
			return ls < rs, nil
		case "<=":
			return ls <= rs, nil
		case ">":
			return ls > rs, nil
		case ">=":
			return ls >= rs, nil
		}
	}
	lf, err := toNumber(left)
	if err != nil {
		return nil, fmt.Errorf("%s left: %w", op, err)
	}
	rf, err := toNumber(right)
	if err != nil {
		return nil, fmt.Errorf("%s right: %w", op, err)
	}
	switch op {
	case "<":
		return lf < rf, nil
	case "<=":
		return lf <= rf, nil
	case ">":
		return lf > rf, nil
	case ">=":
		return lf >= rf, nil
	}
	return nil, fmt.Errorf("unknown compare op %q", op)
}

// looseEqual implements JS abstract equality (==).
func looseEqual(a, b any) bool {
	// Both null/undefined
	if isNullish(a) && isNullish(b) {
		return true
	}
	// One null/undefined and the other is not
	if isNullish(a) || isNullish(b) {
		return false
	}
	// Same type: use strict equality
	if reflect.TypeOf(a) == reflect.TypeOf(b) {
		return strictEqual(a, b)
	}
	// bool → convert to number first, then recurse
	if _, ok := a.(bool); ok {
		n, _ := toNumber(a)
		return looseEqual(n, b)
	}
	if _, ok := b.(bool); ok {
		n, _ := toNumber(b)
		return looseEqual(a, n)
	}
	// number vs string: convert string to number.
	// Recognise all Go numeric types as "number", not just float64.
	af, aIsNum := toNumericFloat64(a)
	bf, bIsNum := toNumericFloat64(b)
	_, aIsStr := a.(string)
	_, bIsStr := b.(string)
	if aIsNum && bIsStr {
		n, _ := toNumber(b)
		return looseEqual(af, n)
	}
	if bIsNum && aIsStr {
		n, _ := toNumber(a)
		return looseEqual(n, bf)
	}
	// Both numeric but different Go types: compare by value.
	if aIsNum && bIsNum {
		return af == bf
	}
	return false
}

// toNumericFloat64 converts any Go numeric type to float64.
// Returns (value, true) if v is a numeric type, (0, false) otherwise.
// This is used to normalise Go integer types so that, e.g., int(0) == float64(0)
// in the same way that JavaScript treats all numbers as float64.
func toNumericFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int8:
		return float64(val), true
	case int16:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case uint:
		return float64(val), true
	case uint8:
		return float64(val), true
	case uint16:
		return float64(val), true
	case uint32:
		return float64(val), true
	case uint64:
		return float64(val), true
	default:
		return 0, false
	}
}

// strictEqual implements JS strict equality (===).
func strictEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	_, aIsUndef := a.(UndefinedValue)
	_, bIsUndef := b.(UndefinedValue)
	if aIsUndef && bIsUndef {
		return true
	}
	if a == nil || b == nil || aIsUndef || bIsUndef {
		return false
	}
	// When both operands are numeric Go types, compare by value.
	// This mirrors JavaScript where all numbers are float64, so int(0) === float64(0).
	af, aIsNum := toNumericFloat64(a)
	bf, bIsNum := toNumericFloat64(b)
	if aIsNum && bIsNum {
		if math.IsNaN(af) || math.IsNaN(bf) {
			return false
		}
		return af == bf
	}
	return reflect.DeepEqual(a, b)
}

// inOperator implements key in obj (map key membership / struct field presence).
func inOperator(key, obj any) (any, error) {
	if obj == nil {
		return nil, fmt.Errorf("cannot use 'in' operator on null")
	}
	if _, ok := obj.(UndefinedValue); ok {
		return nil, fmt.Errorf("cannot use 'in' operator on undefined")
	}

	rv := reflect.ValueOf(obj)
	for rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}

	switch rv.Kind() {
	case reflect.Map:
		if rv.IsNil() {
			return nil, fmt.Errorf("cannot use 'in' operator on nil map")
		}
		keyVal := reflect.ValueOf(key)
		mapKeyType := rv.Type().Key()
		if keyVal.Type() != mapKeyType {
			if keyVal.Type().ConvertibleTo(mapKeyType) {
				keyVal = keyVal.Convert(mapKeyType)
			} else {
				return false, nil
			}
		}
		return rv.MapIndex(keyVal).IsValid(), nil

	case reflect.Struct:
		keyStr, ok := key.(string)
		if !ok {
			return false, nil
		}
		rt := rv.Type()
		for i := 0; i < rt.NumField(); i++ {
			f := rt.Field(i)
			if !f.IsExported() {
				continue
			}
			if f.Name == keyStr {
				return true, nil
			}
			tag := f.Tag.Get("json")
			if tag != "" {
				tagName := strings.Split(tag, ",")[0]
				if tagName != "-" && tagName == keyStr {
					return true, nil
				}
			}
		}
		return false, nil

	default:
		return nil, fmt.Errorf("cannot use 'in' operator on %T", obj)
	}
}
