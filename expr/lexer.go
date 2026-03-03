// Package expr provides the lexer that tokenizes htmlc template expressions.
package expr

import (
	"fmt"
	"strings"
)

// TokenType identifies the kind of a lexed token.
type TokenType int

const (
	// Special
	TokenEOF TokenType = iota
	TokenError

	// Literals
	TokenInt
	TokenFloat
	TokenString

	// Identifier
	TokenIdent

	// Keywords
	TokenTrue
	TokenFalse
	TokenNull
	TokenUndefined
	TokenTypeof
	TokenVoid
	TokenIn
	TokenInstanceof
	// TokenNew is recognised so the parser can return a clear error.
	// The 'new' keyword is intentionally unsupported — this is a declarative
	// template language, not a full JavaScript evaluator.
	TokenNew

	// Arithmetic operators
	TokenStar       // *
	TokenSlash      // /
	TokenPercent    // %
	TokenPlus       // +
	TokenMinus      // -
	TokenStarStar   // **

	// Bitwise / shift operators
	TokenAmp        // &
	TokenCaret      // ^
	TokenPipe       // |
	TokenTilde      // ~
	TokenLtLt       // <<
	TokenGtGt       // >>
	TokenGtGtGt     // >>>

	// Relational operators
	TokenLt         // <
	TokenLtEq       // <=
	TokenGt         // >
	TokenGtEq       // >=

	// Equality operators
	TokenEqEq       // ==
	TokenBangEq     // !=
	TokenEqEqEq     // ===
	TokenBangEqEq   // !==

	// Logical operators
	TokenBang       // !
	TokenAmpAmp     // &&
	TokenPipePipe   // ||
	TokenQQ         // ??

	// Punctuation
	TokenLParen     // (
	TokenRParen     // )
	TokenLBracket   // [
	TokenRBracket   // ]
	TokenLBrace     // {
	TokenRBrace     // }
	TokenDot        // .
	TokenComma      // ,
	TokenColon      // :
	TokenQuestion   // ?
)

var tokenTypeNames = map[TokenType]string{
	TokenEOF:        "EOF",
	TokenError:      "Error",
	TokenInt:        "Int",
	TokenFloat:      "Float",
	TokenString:     "String",
	TokenIdent:      "Ident",
	TokenTrue:       "true",
	TokenFalse:      "false",
	TokenNull:       "null",
	TokenUndefined:  "undefined",
	TokenTypeof:     "typeof",
	TokenVoid:       "void",
	TokenIn:         "in",
	TokenInstanceof: "instanceof",
	TokenNew:        "new",
	TokenStar:       "*",
	TokenSlash:      "/",
	TokenPercent:    "%",
	TokenPlus:       "+",
	TokenMinus:      "-",
	TokenStarStar:   "**",
	TokenAmp:        "&",
	TokenCaret:      "^",
	TokenPipe:       "|",
	TokenTilde:      "~",
	TokenLtLt:       "<<",
	TokenGtGt:       ">>",
	TokenGtGtGt:     ">>>",
	TokenLt:         "<",
	TokenLtEq:       "<=",
	TokenGt:         ">",
	TokenGtEq:       ">=",
	TokenEqEq:       "==",
	TokenBangEq:     "!=",
	TokenEqEqEq:     "===",
	TokenBangEqEq:   "!==",
	TokenBang:       "!",
	TokenAmpAmp:     "&&",
	TokenPipePipe:   "||",
	TokenQQ:         "??",
	TokenLParen:     "(",
	TokenRParen:     ")",
	TokenLBracket:   "[",
	TokenRBracket:   "]",
	TokenLBrace:     "{",
	TokenRBrace:     "}",
	TokenDot:        ".",
	TokenComma:      ",",
	TokenColon:      ":",
	TokenQuestion:   "?",
}

func (t TokenType) String() string {
	if s, ok := tokenTypeNames[t]; ok {
		return s
	}
	return fmt.Sprintf("Token(%d)", int(t))
}

// Pos records the byte offset, line, and column of a token within the source.
type Pos struct {
	Offset int // byte offset from the start of input
	Line   int // 1-based line number
	Col    int // 1-based column number
}

// Token is a single lexical token produced by the Lexer.
type Token struct {
	Type    TokenType
	Value   string // raw source text of the token
	Pos     Pos
}

func (t Token) String() string {
	return fmt.Sprintf("%s(%q)@%d:%d", t.Type, t.Value, t.Pos.Line, t.Pos.Col)
}

// keywords maps keyword strings to their token types.
var keywords = map[string]TokenType{
	"true":       TokenTrue,
	"false":      TokenFalse,
	"null":       TokenNull,
	"undefined":  TokenUndefined,
	"typeof":     TokenTypeof,
	"void":       TokenVoid,
	"in":         TokenIn,
	"instanceof": TokenInstanceof,
	"new":        TokenNew,
}

// Lexer tokenizes an expression string.
type Lexer struct {
	src    string
	pos    int // current byte offset
	line   int // current 1-based line
	col    int // current 1-based column
	tokens []Token
	err    error
}

// NewLexer creates a Lexer for the given source string.
func NewLexer(src string) *Lexer {
	return &Lexer{src: src, line: 1, col: 1}
}

// Tokenize runs the lexer to completion and returns all tokens including EOF.
// On any error the returned error is non-nil and the token slice ends at the
// error token.
func Tokenize(src string) ([]Token, error) {
	l := NewLexer(src)
	for {
		tok := l.Next()
		if tok.Type == TokenError {
			return l.tokens, l.err
		}
		if tok.Type == TokenEOF {
			return l.tokens, nil
		}
	}
}

// Next scans and returns the next token. The token is also appended to the
// internal slice so callers can retrieve all tokens via Tokens().
func (l *Lexer) Next() Token {
	l.skipWhitespace()

	if l.pos >= len(l.src) {
		tok := l.makeToken(TokenEOF, "")
		l.tokens = append(l.tokens, tok)
		return tok
	}

	ch := l.src[l.pos]

	switch {
	case ch == '\'' || ch == '"':
		return l.lexString(ch)
	case isDigit(ch):
		return l.lexNumber()
	case ch == '.' && l.pos+1 < len(l.src) && isDigit(l.src[l.pos+1]):
		// leading-dot float like .5
		return l.lexNumber()
	case isIdentStart(ch):
		return l.lexIdent()
	default:
		return l.lexSymbol()
	}
}

// Tokens returns all tokens produced so far (including the current one).
func (l *Lexer) Tokens() []Token { return l.tokens }

// --- helpers -----------------------------------------------------------------

func (l *Lexer) currentPos() Pos {
	return Pos{Offset: l.pos, Line: l.line, Col: l.col}
}

func (l *Lexer) makeToken(typ TokenType, val string) Token {
	return Token{Type: typ, Value: val}
}

func (l *Lexer) makeTokenAt(typ TokenType, val string, p Pos) Token {
	return Token{Type: typ, Value: val, Pos: p}
}

func (l *Lexer) advance() byte {
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

func (l *Lexer) peekAt(offset int) byte {
	idx := l.pos + offset
	if idx >= len(l.src) {
		return 0
	}
	return l.src[idx]
}

func (l *Lexer) emit(typ TokenType, start Pos, raw string) Token {
	tok := Token{Type: typ, Value: raw, Pos: start}
	l.tokens = append(l.tokens, tok)
	return tok
}

func (l *Lexer) emitError(msg string, p Pos) Token {
	l.err = fmt.Errorf("%d:%d: %s", p.Line, p.Col, msg)
	tok := Token{Type: TokenError, Value: msg, Pos: p}
	l.tokens = append(l.tokens, tok)
	return tok
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.src) {
		ch := l.src[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\r' || ch == '\n' {
			l.advance()
		} else {
			break
		}
	}
}

// --- lexers for different token categories -----------------------------------

func (l *Lexer) lexString(quote byte) Token {
	start := l.currentPos()
	l.advance() // consume opening quote
	var sb strings.Builder
	for {
		if l.pos >= len(l.src) {
			return l.emitError("unterminated string literal", start)
		}
		ch := l.src[l.pos]
		if ch == quote {
			l.advance() // consume closing quote
			break
		}
		if ch == '\\' {
			l.advance() // consume backslash
			if l.pos >= len(l.src) {
				return l.emitError("unterminated escape sequence", start)
			}
			esc := l.advance()
			switch esc {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '\'':
				sb.WriteByte('\'')
			case '"':
				sb.WriteByte('"')
			default:
				sb.WriteByte('\\')
				sb.WriteByte(esc)
			}
		} else {
			sb.WriteByte(ch)
			l.advance()
		}
	}
	return l.emit(TokenString, start, sb.String())
}

func (l *Lexer) lexNumber() Token {
	start := l.currentPos()
	begin := l.pos
	isFloat := false

	// optional leading dot (.5)
	if l.peek() == '.' {
		isFloat = true
		l.advance()
	}

	for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
		l.advance()
	}

	// check for decimal point (only if we didn't start with one)
	if !isFloat && l.peek() == '.' && isDigit(l.peekAt(1)) {
		isFloat = true
		l.advance() // consume '.'
		for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			l.advance()
		}
	}

	// optional exponent
	if l.peek() == 'e' || l.peek() == 'E' {
		isFloat = true
		l.advance()
		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}
		for l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			l.advance()
		}
	}

	raw := l.src[begin:l.pos]
	if isFloat {
		return l.emit(TokenFloat, start, raw)
	}
	return l.emit(TokenInt, start, raw)
}

func (l *Lexer) lexIdent() Token {
	start := l.currentPos()
	begin := l.pos
	for l.pos < len(l.src) && isIdentCont(l.src[l.pos]) {
		l.advance()
	}
	raw := l.src[begin:l.pos]
	typ := TokenIdent
	if kw, ok := keywords[raw]; ok {
		typ = kw
	}
	return l.emit(typ, start, raw)
}

func (l *Lexer) lexSymbol() Token {
	start := l.currentPos()
	ch := l.advance()

	switch ch {
	case '(':
		return l.emit(TokenLParen, start, "(")
	case ')':
		return l.emit(TokenRParen, start, ")")
	case '[':
		return l.emit(TokenLBracket, start, "[")
	case ']':
		return l.emit(TokenRBracket, start, "]")
	case '{':
		return l.emit(TokenLBrace, start, "{")
	case '}':
		return l.emit(TokenRBrace, start, "}")
	case ',':
		return l.emit(TokenComma, start, ",")
	case ':':
		return l.emit(TokenColon, start, ":")
	case ';':
		return l.emit(TokenColon, start, ";")
	case '~':
		return l.emit(TokenTilde, start, "~")
	case '^':
		return l.emit(TokenCaret, start, "^")
	case '%':
		return l.emit(TokenPercent, start, "%")
	case '+':
		return l.emit(TokenPlus, start, "+")
	case '-':
		return l.emit(TokenMinus, start, "-")
	case '/':
		return l.emit(TokenSlash, start, "/")
	case '.':
		return l.emit(TokenDot, start, ".")
	case '*':
		if l.peek() == '*' {
			l.advance()
			return l.emit(TokenStarStar, start, "**")
		}
		return l.emit(TokenStar, start, "*")
	case '!':
		if l.peek() == '=' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return l.emit(TokenBangEqEq, start, "!==")
			}
			return l.emit(TokenBangEq, start, "!=")
		}
		return l.emit(TokenBang, start, "!")
	case '=':
		if l.peek() == '=' {
			l.advance()
			if l.peek() == '=' {
				l.advance()
				return l.emit(TokenEqEqEq, start, "===")
			}
			return l.emit(TokenEqEq, start, "==")
		}
		return l.emitError(fmt.Sprintf("unexpected character %q (assignment is not supported)", ch), start)
	case '<':
		if l.peek() == '=' {
			l.advance()
			return l.emit(TokenLtEq, start, "<=")
		}
		if l.peek() == '<' {
			l.advance()
			return l.emit(TokenLtLt, start, "<<")
		}
		return l.emit(TokenLt, start, "<")
	case '>':
		if l.peek() == '>' {
			l.advance()
			if l.peek() == '>' {
				l.advance()
				return l.emit(TokenGtGtGt, start, ">>>")
			}
			return l.emit(TokenGtGt, start, ">>")
		}
		if l.peek() == '=' {
			l.advance()
			return l.emit(TokenGtEq, start, ">=")
		}
		return l.emit(TokenGt, start, ">")
	case '&':
		if l.peek() == '&' {
			l.advance()
			return l.emit(TokenAmpAmp, start, "&&")
		}
		return l.emit(TokenAmp, start, "&")
	case '|':
		if l.peek() == '|' {
			l.advance()
			return l.emit(TokenPipePipe, start, "||")
		}
		return l.emit(TokenPipe, start, "|")
	case '?':
		if l.peek() == '?' {
			l.advance()
			return l.emit(TokenQQ, start, "??")
		}
		return l.emit(TokenQuestion, start, "?")
	default:
		return l.emitError(fmt.Sprintf("unexpected character %q", ch), start)
	}
}

// --- character classification ------------------------------------------------

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentStart(ch byte) bool {
	return ch == '_' || ch == '$' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isIdentCont(ch byte) bool {
	return isIdentStart(ch) || isDigit(ch)
}
