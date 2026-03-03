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
