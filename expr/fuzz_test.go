package expr

import (
	"encoding/json"
	"testing"
)

// FuzzTokenize verifies that Tokenize never panics on arbitrary input.
// Errors are acceptable; crashes are not.
func FuzzTokenize(f *testing.F) {
	// Seed corpus: representative valid expressions
	f.Add("42")
	f.Add("a.b.c")
	f.Add("items.map(x => x.name)")
	f.Add("a ? b : c")
	f.Add("x ?? y")
	f.Add("x?.y?.z")
	f.Add("!true && false || null")
	f.Add("[1, 2, 3]")
	f.Add("{key: value}")
	f.Add(`"hello \"world\""`)
	f.Add("1 + 2 * 3")
	f.Add("(a + b) * (c - d)")
	f.Add("typeof x")
	f.Add("void 0")
	f.Add("~bits")
	f.Add("a === b !== c")
	f.Add("x >= 0 && x <= 100")
	f.Add("fn(a, b, c)")
	f.Add("arr[0]")
	f.Add(`'single \'quoted\''`)
	f.Add("")
	f.Add("...")
	f.Add("a?.b?.c?.d")
	f.Add("x ** 2")

	f.Fuzz(func(t *testing.T, src string) {
		// Must not panic; errors are acceptable.
		tokens, _ := Tokenize(src)
		_ = tokens
	})
}

// FuzzParse verifies that Parse never panics and that a nil node is returned
// iff an error is returned. Errors are acceptable; crashes are not.
func FuzzParse(f *testing.F) {
	// Seed with valid expressions plus syntactically broken inputs
	f.Add("1 + 2 * 3")
	f.Add("(a + b) * (c - d)")
	f.Add("42")
	f.Add("a.b.c")
	f.Add("items.map(x => x.name)")
	f.Add("a ? b : c")
	f.Add("x ?? y")
	f.Add("x?.y?.z")
	f.Add("!true && false || null")
	f.Add("[1, 2, 3]")
	f.Add("{key: value}")
	f.Add(`"hello \"world\""`)
	f.Add("fn(a, b, c)")
	f.Add("arr[index]")
	f.Add("a || b && c")
	f.Add("x > 0 ? x : -x")
	f.Add("")          // empty: expect error
	f.Add("(")         // broken: expect error
	f.Add("1 +")       // broken: expect error
	f.Add("? :")       // broken: expect error
	f.Add("a..b")      // broken: expect error
	f.Add("((((a))))") // deeply nested

	f.Fuzz(func(t *testing.T, src string) {
		// Must not panic. Parse errors are acceptable.
		node, err := Parse(src)
		if err == nil {
			// If parse succeeded the AST must not be nil.
			if node == nil {
				t.Errorf("Parse(%q): err==nil but node==nil", src)
			}
		}
	})
}

// FuzzEval verifies that Eval never panics on arbitrary source and scope.
// Errors are acceptable; crashes are not.
func FuzzEval(f *testing.F) {
	f.Add("42", `{}`)
	f.Add("a + b", `{"a":1,"b":2}`)
	f.Add("items[0]", `{"items":[1,2,3]}`)
	f.Add("name.length", `{"name":"hello"}`)
	f.Add("x ? y : z", `{"x":true,"y":1,"z":2}`)
	f.Add("!flag", `{"flag":false}`)
	f.Add("a ?? b", `{"a":null,"b":"default"}`)
	f.Add("arr.map(x => x * 2)", `{"arr":[1,2,3]}`)
	f.Add("obj.key", `{"obj":{"key":"value"}}`)
	f.Add("a && b || c", `{"a":true,"b":false,"c":true}`)
	f.Add("x?.y?.z", `{"x":{"y":{"z":42}}}`)
	f.Add("typeof val", `{"val":42}`)
	f.Add("[1, 2, 3]", `{}`)
	f.Add("{k: v}", `{"v":"hello"}`)
	f.Add("", `{}`)

	f.Fuzz(func(t *testing.T, src string, scopeJSON string) {
		var scope map[string]any
		if err := json.Unmarshal([]byte(scopeJSON), &scope); err != nil {
			// Invalid JSON scope: skip (not the system under test).
			t.Skip()
		}
		// Must not panic. Eval errors are acceptable.
		v, err := Eval(src, scope)
		_, _ = v, err
	})
}
