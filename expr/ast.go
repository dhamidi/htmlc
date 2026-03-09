// Package expr implements the expression language used in htmlc templates.
package expr

// Node is the common interface implemented by every AST node.
type Node interface {
	nodeType() string
}

// Expr is a compiled expression ready for repeated evaluation.
type Expr interface {
	Eval(scope map[string]any) (any, error)
}

// --- Literal nodes ---

// NumberLit represents an integer or float literal, e.g. 42, 3.14.
type NumberLit struct {
	Value float64
}

func (n *NumberLit) nodeType() string { return "NumberLit" }

// StringLit represents a single- or double-quoted string literal.
type StringLit struct {
	Value string
}

func (n *StringLit) nodeType() string { return "StringLit" }

// BoolLit represents the boolean literals true and false.
type BoolLit struct {
	Value bool
}

func (n *BoolLit) nodeType() string { return "BoolLit" }

// NullLit represents the null literal.
type NullLit struct{}

func (n *NullLit) nodeType() string { return "NullLit" }

// UndefinedLit represents the undefined literal.
type UndefinedLit struct{}

func (n *UndefinedLit) nodeType() string { return "UndefinedLit" }

// --- Identifier ---

// Identifier represents a variable reference resolved in the evaluation scope.
type Identifier struct {
	Name string
}

func (n *Identifier) nodeType() string { return "Identifier" }

// --- Unary expression ---

// UnaryExpr represents a unary prefix expression.
// Op is one of: "!", "-", "+", "~", "typeof", "void".
type UnaryExpr struct {
	Op      string
	Operand Node
}

func (n *UnaryExpr) nodeType() string { return "UnaryExpr" }

// --- Binary expression ---

// BinaryExpr represents a binary infix expression.
// Op is one of the operators from §1.3: "**", "*", "/", "%", "+", "-",
// "<<", ">>", ">>>", "<", "<=", ">", ">=", "in", "instanceof",
// "==", "!=", "===", "!==", "&", "^", "|", "&&", "||", "??".
type BinaryExpr struct {
	Op    string
	Left  Node
	Right Node
}

func (n *BinaryExpr) nodeType() string { return "BinaryExpr" }

// --- Ternary / conditional expression ---

// TernaryExpr represents condition ? consequent : alternate.
type TernaryExpr struct {
	Condition  Node
	Consequent Node
	Alternate  Node
}

func (n *TernaryExpr) nodeType() string { return "TernaryExpr" }

// --- Member access ---

// MemberExpr represents dot or bracket member access.
// When Computed is false the expression is dot notation (Property is an Identifier).
// When Computed is true the expression is bracket notation (Property is any Node).
type MemberExpr struct {
	Object   Node
	Property Node
	Computed bool // true for obj[expr], false for obj.name
}

func (n *MemberExpr) nodeType() string { return "MemberExpr" }

// OptionalMemberExpr represents optional chaining member access: obj?.prop or
// obj?.[expr]. If the object is null or undefined the expression evaluates to
// undefined instead of returning an error.
type OptionalMemberExpr struct {
	Object   Node
	Property Node
	Computed bool // true for obj?.[expr], false for obj?.prop
}

func (n *OptionalMemberExpr) nodeType() string { return "OptionalMemberExpr" }

// --- Function call ---

// CallExpr represents a function call expression: callee(args...).
type CallExpr struct {
	Callee Node
	Args   []Node
}

func (n *CallExpr) nodeType() string { return "CallExpr" }

// --- Array literal ---

// ArrayLit represents an array literal, e.g. [1, 'two', item].
type ArrayLit struct {
	Elements []Node
}

func (n *ArrayLit) nodeType() string { return "ArrayLit" }

// --- Object literal ---

// Property is a single key-value pair in an object literal.
// Key is always a string (identifier or quoted string).
type Property struct {
	Key   string
	Value Node
}

// ObjectLit represents an object literal, e.g. { key: val, 'x': 1 }.
type ObjectLit struct {
	Properties []Property
}

func (n *ObjectLit) nodeType() string { return "ObjectLit" }
