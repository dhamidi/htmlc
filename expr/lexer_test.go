package expr

import (
	"testing"
)

// tokenTypes extracts just the type sequence from a token slice.
func tokenTypes(toks []Token) []TokenType {
	types := make([]TokenType, len(toks))
	for i, t := range toks {
		types[i] = t.Type
	}
	return types
}

// tokenValues extracts the value sequence from a token slice.
func tokenValues(toks []Token) []string {
	vals := make([]string, len(toks))
	for i, t := range toks {
		vals[i] = t.Value
	}
	return vals
}

func mustTokenize(t *testing.T, src string) []Token {
	t.Helper()
	toks, err := Tokenize(src)
	if err != nil {
		t.Fatalf("Tokenize(%q) error: %v", src, err)
	}
	return toks
}

// TestAcceptanceCriteria tokenises the expression from the spec:
//
//	user.name === 'admin' ? items[0] : null
func TestAcceptanceCriteria(t *testing.T) {
	src := "user.name === 'admin' ? items[0] : null"
	toks := mustTokenize(t, src)

	wantTypes := []TokenType{
		TokenIdent,    // user
		TokenDot,      // .
		TokenIdent,    // name
		TokenEqEqEq,   // ===
		TokenString,   // 'admin'
		TokenQuestion, // ?
		TokenIdent,    // items
		TokenLBracket, // [
		TokenInt,      // 0
		TokenRBracket, // ]
		TokenColon,    // :
		TokenNull,     // null
		TokenEOF,
	}

	gotTypes := tokenTypes(toks)
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("token count: got %d, want %d\ntokens: %v", len(gotTypes), len(wantTypes), toks)
	}
	for i, want := range wantTypes {
		if gotTypes[i] != want {
			t.Errorf("token[%d]: got %s, want %s", i, gotTypes[i], want)
		}
	}

	// spot-check values
	if toks[0].Value != "user" {
		t.Errorf("token[0].Value: got %q, want %q", toks[0].Value, "user")
	}
	if toks[3].Value != "===" {
		t.Errorf("token[3].Value: got %q, want %q", toks[3].Value, "===")
	}
	if toks[4].Value != "admin" {
		t.Errorf("token[4].Value (string content): got %q, want %q", toks[4].Value, "admin")
	}
}

// TestAllBinaryOperators verifies every operator from §1.3 is recognised.
func TestAllBinaryOperators(t *testing.T) {
	cases := []struct {
		src  string
		want TokenType
	}{
		{"**", TokenStarStar},
		{"*", TokenStar},
		{"/", TokenSlash},
		{"%", TokenPercent},
		{"+", TokenPlus},
		{"-", TokenMinus},
		{"<<", TokenLtLt},
		{">>", TokenGtGt},
		{">>>", TokenGtGtGt},
		{"<", TokenLt},
		{"<=", TokenLtEq},
		{">", TokenGt},
		{">=", TokenGtEq},
		{"==", TokenEqEq},
		{"!=", TokenBangEq},
		{"===", TokenEqEqEq},
		{"!==", TokenBangEqEq},
		{"&", TokenAmp},
		{"^", TokenCaret},
		{"|", TokenPipe},
		{"&&", TokenAmpAmp},
		{"||", TokenPipePipe},
		{"??", TokenQQ},
	}

	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			toks := mustTokenize(t, tc.src)
			if len(toks) < 1 {
				t.Fatal("no tokens")
			}
			if toks[0].Type != tc.want {
				t.Errorf("got %s, want %s", toks[0].Type, tc.want)
			}
		})
	}
}

// TestUnaryOperators verifies unary operator tokens from §1.2.
func TestUnaryOperators(t *testing.T) {
	cases := []struct {
		src  string
		want TokenType
	}{
		{"!", TokenBang},
		{"-", TokenMinus},
		{"+", TokenPlus},
		{"~", TokenTilde},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			toks := mustTokenize(t, tc.src)
			if toks[0].Type != tc.want {
				t.Errorf("got %s, want %s", toks[0].Type, tc.want)
			}
		})
	}
}

// TestKeywords checks that keyword strings produce keyword token types.
func TestKeywords(t *testing.T) {
	cases := []struct {
		src  string
		want TokenType
	}{
		{"true", TokenTrue},
		{"false", TokenFalse},
		{"null", TokenNull},
		{"undefined", TokenUndefined},
		{"typeof", TokenTypeof},
		{"void", TokenVoid},
		{"in", TokenIn},
		{"instanceof", TokenInstanceof},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			toks := mustTokenize(t, tc.src)
			if toks[0].Type != tc.want {
				t.Errorf("got %s, want %s", toks[0].Type, tc.want)
			}
		})
	}
}

// TestTypeofAndVoidAreKeywords explicitly verifies the spec requirement.
func TestTypeofAndVoidAreKeywords(t *testing.T) {
	for _, kw := range []string{"typeof", "void"} {
		toks := mustTokenize(t, kw)
		if toks[0].Type == TokenIdent {
			t.Errorf("%q tokenised as Ident, want keyword token", kw)
		}
	}
	if toks := mustTokenize(t, "typeof"); toks[0].Type != TokenTypeof {
		t.Errorf("typeof: got %s, want TokenTypeof", toks[0].Type)
	}
	if toks := mustTokenize(t, "void"); toks[0].Type != TokenVoid {
		t.Errorf("void: got %s, want TokenVoid", toks[0].Type)
	}
}

// TestStringLiterals verifies single- and double-quoted strings.
func TestStringLiterals(t *testing.T) {
	cases := []struct {
		src      string
		wantVal  string
	}{
		{`'hello'`, "hello"},
		{`"world"`, "world"},
		{`'it\'s'`, "it's"},
		{`"say \"hi\""`, `say "hi"`},
		{`'newline\n'`, "newline\n"},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			toks := mustTokenize(t, tc.src)
			if toks[0].Type != TokenString {
				t.Fatalf("got %s, want TokenString", toks[0].Type)
			}
			if toks[0].Value != tc.wantVal {
				t.Errorf("value: got %q, want %q", toks[0].Value, tc.wantVal)
			}
		})
	}
}

// TestNumberLiterals verifies integer, float, and leading-dot float literals.
func TestNumberLiterals(t *testing.T) {
	intCases := []string{"0", "42", "100"}
	for _, src := range intCases {
		t.Run("int_"+src, func(t *testing.T) {
			toks := mustTokenize(t, src)
			if toks[0].Type != TokenInt {
				t.Errorf("got %s, want TokenInt", toks[0].Type)
			}
			if toks[0].Value != src {
				t.Errorf("value: got %q, want %q", toks[0].Value, src)
			}
		})
	}

	floatCases := []struct {
		src string
	}{
		{"3.14"},
		{".5"},
		{"1e10"},
		{"2.5e-3"},
	}
	for _, tc := range floatCases {
		t.Run("float_"+tc.src, func(t *testing.T) {
			toks := mustTokenize(t, tc.src)
			if toks[0].Type != TokenFloat {
				t.Errorf("got %s, want TokenFloat", toks[0].Type)
			}
			if toks[0].Value != tc.src {
				t.Errorf("value: got %q, want %q", toks[0].Value, tc.src)
			}
		})
	}
}

// TestPunctuation checks all punctuation tokens.
func TestPunctuation(t *testing.T) {
	cases := []struct {
		ch   string
		want TokenType
	}{
		{"(", TokenLParen},
		{")", TokenRParen},
		{"[", TokenLBracket},
		{"]", TokenRBracket},
		{"{", TokenLBrace},
		{"}", TokenRBrace},
		{".", TokenDot},
		{",", TokenComma},
		{":", TokenColon},
		{"?", TokenQuestion},
	}
	for _, tc := range cases {
		t.Run(tc.ch, func(t *testing.T) {
			toks := mustTokenize(t, tc.ch)
			if toks[0].Type != tc.want {
				t.Errorf("got %s, want %s", toks[0].Type, tc.want)
			}
		})
	}
}

// TestPositionTracking verifies that token positions are reported correctly.
func TestPositionTracking(t *testing.T) {
	src := "a + b"
	toks := mustTokenize(t, src)

	if toks[0].Pos.Col != 1 {
		t.Errorf("token[0] col: got %d, want 1", toks[0].Pos.Col)
	}
	if toks[1].Pos.Col != 3 {
		t.Errorf("token[1] ('+') col: got %d, want 3", toks[1].Pos.Col)
	}
	if toks[2].Pos.Col != 5 {
		t.Errorf("token[2] col: got %d, want 5", toks[2].Pos.Col)
	}
}

// TestNullishCoalescing verifies ?? is a single token, not two ? tokens.
func TestNullishCoalescing(t *testing.T) {
	toks := mustTokenize(t, "??")
	if toks[0].Type != TokenQQ {
		t.Errorf("got %s, want TokenQQ (??)", toks[0].Type)
	}
}

// TestLexer_EdgeCases covers boundary values and unusual-but-valid input that
// are not exercised by the main happy-path tests.
func TestLexer_EdgeCases(t *testing.T) {
	// Empty string must produce exactly one TokenEOF and no error.
	// This is the simplest possible input and must not cause any loop/index panic.
	t.Run("empty string produces single TokenEOF", func(t *testing.T) {
		toks, err := Tokenize("")
		if err != nil {
			t.Fatalf("Tokenize(%q): unexpected error: %v", "", err)
		}
		if len(toks) != 1 || toks[0].Type != TokenEOF {
			t.Errorf("Tokenize(%q): got %v, want [TokenEOF]", "", toks)
		}
	})

	// String literal with an escaped single quote: 'it\'s' → value "it's".
	// The escape sequence must be decoded, not passed through literally.
	t.Run("string with escaped quote is decoded correctly", func(t *testing.T) {
		toks, err := Tokenize(`'it\'s'`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if toks[0].Type != TokenString {
			t.Fatalf("got %s, want TokenString", toks[0].Type)
		}
		if toks[0].Value != "it's" {
			t.Errorf("value: got %q, want %q", toks[0].Value, "it's")
		}
	})

	// Unterminated string must produce TokenError and a non-nil error.
	// This guards against infinite loops or panics when the closing quote is absent.
	t.Run("unterminated string produces TokenError", func(t *testing.T) {
		toks, err := Tokenize(`"hello`)
		if err == nil {
			t.Error("Tokenize unterminated string: expected error, got nil")
		}
		if len(toks) == 0 || toks[len(toks)-1].Type != TokenError {
			t.Errorf("Tokenize unterminated string: last token should be TokenError, got %v", toks)
		}
	})

	// !!=: the lexer must produce TokenBang then TokenBangEq (greedy matching
	// picks "!=" as one token, not "!!==" as three).
	t.Run("!!= produces TokenBang then TokenBangEq", func(t *testing.T) {
		toks, err := Tokenize("!!=")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		wantTypes := []TokenType{TokenBang, TokenBangEq, TokenEOF}
		gotTypes := tokenTypes(toks)
		if len(gotTypes) != len(wantTypes) {
			t.Fatalf("token count: got %d, want %d; tokens: %v", len(gotTypes), len(wantTypes), toks)
		}
		for i, want := range wantTypes {
			if gotTypes[i] != want {
				t.Errorf("token[%d]: got %s, want %s", i, gotTypes[i], want)
			}
		}
	})

	// === must be tokenised as a single TokenEqEqEq, not three separate = tokens.
	t.Run("=== produces single TokenEqEqEq", func(t *testing.T) {
		toks, err := Tokenize("===")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if toks[0].Type != TokenEqEqEq {
			t.Errorf("got %s, want TokenEqEqEq", toks[0].Type)
		}
	})

	// !== must be tokenised as a single TokenBangEqEq.
	t.Run("!== produces single TokenBangEqEq", func(t *testing.T) {
		toks, err := Tokenize("!==")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if toks[0].Type != TokenBangEqEq {
			t.Errorf("got %s, want TokenBangEqEq", toks[0].Type)
		}
	})

	// Numeric literal edge cases: each must produce the expected token type.
	// 0.0 and 1E-2 are not covered by TestNumberLiterals above.
	numberCases := []struct {
		src      string
		wantType TokenType
	}{
		{"0", TokenInt},     // integer zero
		{"0.0", TokenFloat}, // floating-point zero (has decimal point)
		{".5", TokenFloat},  // leading-dot float
		{"1e3", TokenFloat}, // lower-case exponent
		{"1E-2", TokenFloat}, // upper-case exponent with negative sign
	}
	for _, tc := range numberCases {
		tc := tc
		t.Run("number_"+tc.src, func(t *testing.T) {
			toks, err := Tokenize(tc.src)
			if err != nil {
				t.Fatalf("Tokenize(%q): unexpected error: %v", tc.src, err)
			}
			if toks[0].Type != tc.wantType {
				t.Errorf("Tokenize(%q): got %s, want %s", tc.src, toks[0].Type, tc.wantType)
			}
		})
	}

	// Multi-line input: a newline inside an expression is treated as whitespace
	// and must not cause a panic or error.
	t.Run("multi-line input does not panic", func(t *testing.T) {
		toks, err := Tokenize("a\n+ b")
		if err != nil {
			t.Fatalf("Tokenize(multi-line): unexpected error: %v", err)
		}
		wantTypes := []TokenType{TokenIdent, TokenPlus, TokenIdent, TokenEOF}
		gotTypes := tokenTypes(toks)
		if len(gotTypes) != len(wantTypes) {
			t.Fatalf("token count: got %d, want %d; tokens: %v", len(gotTypes), len(wantTypes), toks)
		}
		for i, want := range wantTypes {
			if gotTypes[i] != want {
				t.Errorf("token[%d]: got %s, want %s", i, gotTypes[i], want)
			}
		}
	})
}

// TestComplexExpression tokenises a more complex expression end-to-end.
func TestComplexExpression(t *testing.T) {
	src := "typeof x === 'number' && x > 0"
	toks := mustTokenize(t, src)
	wantTypes := []TokenType{
		TokenTypeof,
		TokenIdent,  // x
		TokenEqEqEq, // ===
		TokenString, // 'number'
		TokenAmpAmp, // &&
		TokenIdent,  // x
		TokenGt,     // >
		TokenInt,    // 0
		TokenEOF,
	}
	gotTypes := tokenTypes(toks)
	if len(gotTypes) != len(wantTypes) {
		t.Fatalf("token count: got %d, want %d\ntokens: %v", len(gotTypes), len(wantTypes), toks)
	}
	for i, want := range wantTypes {
		if gotTypes[i] != want {
			t.Errorf("token[%d]: got %s, want %s", i, gotTypes[i], want)
		}
	}
}
