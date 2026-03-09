// Package expr provides the parser that builds an AST from htmlc expression tokens.
package expr

import (
	"fmt"
	"strconv"
)

// Parse tokenizes src, parses it, and returns a compiled Expr ready for
// evaluation. It returns a descriptive error on any lexical or syntax error.
// Parse is an alias for Compile.
func Parse(src string) (Expr, error) {
	return Compile(src)
}

// Compile tokenizes src, parses it, and returns a compiled Expr ready for
// evaluation. It returns a descriptive error on any lexical or syntax error.
func Compile(src string) (Expr, error) {
	tokens, err := Tokenize(src)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	node, err := p.parseExpr()
	if err != nil {
		return nil, err
	}
	if !p.atEnd() {
		tok := p.peek()
		return nil, fmt.Errorf("%d:%d: unexpected token %s after expression", tok.Pos.Line, tok.Pos.Col, tok.Type)
	}
	return &compiledExpr{node: node}, nil
}

// compiledExpr wraps the root AST node and implements Expr.
type compiledExpr struct {
	node Node
}

// Eval satisfies the Expr interface by delegating to evalNode in eval.go.
func (c *compiledExpr) Eval(scope map[string]any) (any, error) {
	return evalNode(c.node, scope)
}

// --- parser ------------------------------------------------------------------

type parser struct {
	tokens []Token
	pos    int
}

func (p *parser) peek() Token {
	return p.tokens[p.pos]
}

func (p *parser) advance() Token {
	tok := p.tokens[p.pos]
	if tok.Type != TokenEOF {
		p.pos++
	}
	return tok
}

func (p *parser) atEnd() bool {
	return p.tokens[p.pos].Type == TokenEOF
}

func (p *parser) check(t TokenType) bool {
	return p.tokens[p.pos].Type == t
}

func (p *parser) expect(t TokenType) (Token, error) {
	if p.check(t) {
		return p.advance(), nil
	}
	tok := p.peek()
	return tok, fmt.Errorf("%d:%d: expected %s, got %s", tok.Pos.Line, tok.Pos.Col, t, tok.Type)
}

// --- precedence table --------------------------------------------------------

type opInfo struct {
	prec       int
	rightAssoc bool
}

// binaryPrec maps binary operator tokens to their precedence and associativity.
// Higher prec values bind more tightly.
var binaryPrec = map[TokenType]opInfo{
	TokenStarStar:   {12, true},  // ** (right-associative)
	TokenStar:       {11, false}, // *
	TokenSlash:      {11, false}, // /
	TokenPercent:    {11, false}, // %
	TokenPlus:       {10, false}, // +
	TokenMinus:      {10, false}, // -
	TokenLtLt:       {9, false},  // <<
	TokenGtGt:       {9, false},  // >>
	TokenGtGtGt:     {9, false},  // >>>
	TokenLt:         {8, false},  // <
	TokenLtEq:       {8, false},  // <=
	TokenGt:         {8, false},  // >
	TokenGtEq:       {8, false},  // >=
	TokenIn:         {8, false},  // in
	TokenInstanceof: {8, false},  // instanceof
	TokenEqEq:       {7, false},  // ==
	TokenBangEq:     {7, false},  // !=
	TokenEqEqEq:     {7, false},  // ===
	TokenBangEqEq:   {7, false},  // !==
	TokenAmp:        {6, false},  // &
	TokenCaret:      {5, false},  // ^
	TokenPipe:       {4, false},  // |
	TokenAmpAmp:     {3, false},  // &&
	TokenPipePipe:   {2, false},  // ||
	TokenQQ:         {1, false},  // ??
}

// --- parsing -----------------------------------------------------------------

// parseExpr parses a complete expression including ternary.
func (p *parser) parseExpr() (Node, error) {
	return p.parseTernary()
}

// parseTernary parses the ternary conditional (condition ? consequent : alternate).
// Ternary is right-associative: a ? b : c ? d : e → a ? b : (c ? d : e).
func (p *parser) parseTernary() (Node, error) {
	cond, err := p.parseBinary(0)
	if err != nil {
		return nil, err
	}
	if !p.check(TokenQuestion) {
		return cond, nil
	}
	p.advance() // consume '?'
	cons, err := p.parseTernary()
	if err != nil {
		return nil, err
	}
	if _, err = p.expect(TokenColon); err != nil {
		return nil, err
	}
	alt, err := p.parseTernary()
	if err != nil {
		return nil, err
	}
	return &TernaryExpr{Condition: cond, Consequent: cons, Alternate: alt}, nil
}

// parseBinary uses precedence climbing to handle left- and right-associative
// binary operators. It collects operators whose precedence is strictly greater
// than minPrec.
func (p *parser) parseBinary(minPrec int) (Node, error) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for {
		info, ok := binaryPrec[p.peek().Type]
		if !ok || info.prec <= minPrec {
			break
		}
		op := p.advance()
		// For left-assoc, recurse with same prec so the next same-level op is
		// NOT consumed (left nesting). For right-assoc, recurse with prec-1 so
		// the next same-level op IS consumed (right nesting).
		nextPrec := info.prec
		if info.rightAssoc {
			nextPrec = info.prec - 1
		}
		right, err := p.parseBinary(nextPrec)
		if err != nil {
			return nil, err
		}
		left = &BinaryExpr{Op: op.Value, Left: left, Right: right}
	}
	return left, nil
}

// parseUnary parses prefix unary expressions: !, -, +, ~, typeof, void.
func (p *parser) parseUnary() (Node, error) {
	switch p.peek().Type {
	case TokenBang, TokenMinus, TokenPlus, TokenTilde:
		op := p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: op.Value, Operand: operand}, nil
	case TokenTypeof:
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "typeof", Operand: operand}, nil
	case TokenVoid:
		p.advance()
		operand, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		return &UnaryExpr{Op: "void", Operand: operand}, nil
	case TokenNew:
		// 'new' is a JavaScript-specific construct for object instantiation.
		// It is intentionally unsupported — template expressions are a
		// declarative, side-effect-free subset of Vue template syntax.
		tok := p.peek()
		return nil, fmt.Errorf("%d:%d: 'new' is not supported in template expressions", tok.Pos.Line, tok.Pos.Col)
	}
	return p.parsePostfix()
}

// parsePostfix parses left-hand-side postfix operations: dot member access,
// computed member access ([]), optional chaining (?.), and function calls.
func (p *parser) parsePostfix() (Node, error) {
	expr, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek().Type {
		case TokenDot:
			p.advance() // consume '.'
			tok, err := p.expect(TokenIdent)
			if err != nil {
				return nil, err
			}
			expr = &MemberExpr{
				Object:   expr,
				Property: &Identifier{Name: tok.Value},
				Computed: false,
			}
		case TokenLBracket:
			p.advance() // consume '['
			prop, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			if _, err = p.expect(TokenRBracket); err != nil {
				return nil, err
			}
			expr = &MemberExpr{Object: expr, Property: prop, Computed: true}
		case TokenOptionalChain:
			p.advance() // consume '?.'
			// After '?.' expect either an identifier or '[' for computed access.
			switch p.peek().Type {
			case TokenIdent:
				tok, err := p.expect(TokenIdent)
				if err != nil {
					return nil, err
				}
				expr = &OptionalMemberExpr{
					Object:   expr,
					Property: &Identifier{Name: tok.Value},
					Computed: false,
				}
			case TokenLBracket:
				p.advance() // consume '['
				prop, err := p.parseExpr()
				if err != nil {
					return nil, err
				}
				if _, err = p.expect(TokenRBracket); err != nil {
					return nil, err
				}
				expr = &OptionalMemberExpr{Object: expr, Property: prop, Computed: true}
			default:
				tok := p.peek()
				return nil, fmt.Errorf("%d:%d: expected identifier or '[' after '?.'", tok.Pos.Line, tok.Pos.Col)
			}
		case TokenLParen:
			p.advance() // consume '('
			args, err := p.parseArgList()
			if err != nil {
				return nil, err
			}
			expr = &CallExpr{Callee: expr, Args: args}
		default:
			return expr, nil
		}
	}
}

// parseArgList parses a comma-separated argument list up to ')'.
// The opening '(' must already have been consumed.
func (p *parser) parseArgList() ([]Node, error) {
	var args []Node
	if p.check(TokenRParen) {
		p.advance()
		return args, nil
	}
	for {
		arg, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		args = append(args, arg)
		if !p.check(TokenComma) {
			break
		}
		p.advance()
	}
	if _, err := p.expect(TokenRParen); err != nil {
		return nil, err
	}
	return args, nil
}

// parsePrimary parses atomic expressions: literals, identifiers, grouped
// expressions (parens), array literals, and object literals.
func (p *parser) parsePrimary() (Node, error) {
	tok := p.peek()
	switch tok.Type {
	case TokenInt:
		p.advance()
		v, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("%d:%d: invalid integer %q", tok.Pos.Line, tok.Pos.Col, tok.Value)
		}
		return &NumberLit{Value: v}, nil
	case TokenFloat:
		p.advance()
		v, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("%d:%d: invalid float %q", tok.Pos.Line, tok.Pos.Col, tok.Value)
		}
		return &NumberLit{Value: v}, nil
	case TokenString:
		p.advance()
		return &StringLit{Value: tok.Value}, nil
	case TokenTrue:
		p.advance()
		return &BoolLit{Value: true}, nil
	case TokenFalse:
		p.advance()
		return &BoolLit{Value: false}, nil
	case TokenNull:
		p.advance()
		return &NullLit{}, nil
	case TokenUndefined:
		p.advance()
		return &UndefinedLit{}, nil
	case TokenIdent:
		p.advance()
		return &Identifier{Name: tok.Value}, nil
	case TokenLParen:
		p.advance() // consume '('
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err = p.expect(TokenRParen); err != nil {
			return nil, err
		}
		return inner, nil
	case TokenLBracket:
		return p.parseArrayLit()
	case TokenLBrace:
		return p.parseObjectLit()
	case TokenEOF:
		return nil, fmt.Errorf("unexpected end of expression")
	default:
		return nil, fmt.Errorf("%d:%d: unexpected token %s", tok.Pos.Line, tok.Pos.Col, tok.Type)
	}
}

// parseArrayLit parses an array literal: [elem, elem, ...].
func (p *parser) parseArrayLit() (Node, error) {
	p.advance() // consume '['
	var elems []Node
	for !p.check(TokenRBracket) && !p.atEnd() {
		elem, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		elems = append(elems, elem)
		if p.check(TokenComma) {
			p.advance()
		} else {
			break
		}
	}
	if _, err := p.expect(TokenRBracket); err != nil {
		return nil, err
	}
	return &ArrayLit{Elements: elems}, nil
}

// parseObjectLit parses an object literal: { key: value, ... }.
// Keys must be identifiers or string literals.
func (p *parser) parseObjectLit() (Node, error) {
	p.advance() // consume '{'
	var props []Property
	for !p.check(TokenRBrace) && !p.atEnd() {
		tok := p.peek()
		var key string
		switch tok.Type {
		case TokenIdent:
			key = tok.Value
			p.advance()
		case TokenString:
			key = tok.Value
			p.advance()
		default:
			return nil, fmt.Errorf("%d:%d: expected property key (identifier or string), got %s",
				tok.Pos.Line, tok.Pos.Col, tok.Type)
		}
		if _, err := p.expect(TokenColon); err != nil {
			return nil, err
		}
		val, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		props = append(props, Property{Key: key, Value: val})
		if p.check(TokenComma) {
			p.advance()
		} else {
			break
		}
	}
	if _, err := p.expect(TokenRBrace); err != nil {
		return nil, err
	}
	return &ObjectLit{Properties: props}, nil
}
